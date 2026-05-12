package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// dialDaemon establishes a gRPC client connection to the lobsterd daemon.
// TLS is enabled when caFile or certFile/keyFile are provided; otherwise
// an insecure connection is used.  authToken, if non-empty, is attached as
// a Bearer token to every RPC via interceptors.
func dialDaemon(_ context.Context, addr, authToken, caFile, certFile, keyFile string) (*grpc.ClientConn, error) {
	// grpc-go v1.68+ uses the dns resolver by default for bare host:port
	// addresses.  DNS resolution of localhost can produce zero addresses on
	// some systems (macOS, Docker Desktop).  Use the passthrough resolver so
	// the address is used verbatim.
	if !strings.Contains(addr, "://") {
		addr = "passthrough:///" + addr
	}

	var opts []grpc.DialOption

	switch {
	case certFile != "" && keyFile != "":
		// mTLS: client certificate + optional CA bundle.
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		if caFile != "" {
			pool, err := loadCertPool(caFile)
			if err != nil {
				return nil, err
			}
			tlsCfg.RootCAs = pool
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))

	case caFile != "":
		// Server TLS with custom CA only.
		pool, err := loadCertPool(caFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: pool})))

	default:
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if authToken != "" {
		opts = append(opts,
			grpc.WithUnaryInterceptor(bearerUnaryInterceptor(authToken)),
			grpc.WithStreamInterceptor(bearerStreamInterceptor(authToken)),
		)
	}

	return grpc.NewClient(addr, opts...)
}

func loadCertPool(caFile string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate %q: %w", caFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("parse CA certificate %q: no valid PEM block found", caFile)
	}
	return pool, nil
}

func bearerUnaryInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func bearerStreamInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return streamer(ctx, desc, cc, method, opts...)
	}
}
