// Package middleware provides gRPC server interceptors for Lobster.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthConfig controls how the JWKS auth interceptor validates tokens.
type AuthConfig struct {
	// JWKSUrl is the URL of the JWKS endpoint. Required in production.
	JWKSUrl string
	// AllowInsecureLocal skips token validation entirely. Must only be used
	// when ExplicitLocalMode is true; any other usage is rejected at startup.
	AllowInsecureLocal bool
	// ExplicitLocalMode must be true to activate AllowInsecureLocal.
	ExplicitLocalMode bool

	// StaticToken is used for a simple single-token check when JWKS is not
	// configured. Exactly one of JWKSUrl or StaticToken should be set.
	StaticToken string

	// CacheRefreshInterval controls how often the JWK set is refreshed.
	// Defaults to 5 minutes.
	CacheRefreshInterval time.Duration
}

// JWKSAuth is a gRPC unary + stream interceptor pair that validates Bearer
// tokens against a JWKS endpoint.
type JWKSAuth struct {
	cfg   AuthConfig
	cache *jwk.Cache
}

// NewJWKSAuth creates a JWKSAuth from cfg. Returns an error if cfg is invalid.
func NewJWKSAuth(cfg AuthConfig) (*JWKSAuth, error) {
	if cfg.AllowInsecureLocal && !cfg.ExplicitLocalMode {
		return nil, errors.New("auth: AllowInsecureLocal requires ExplicitLocalMode=true")
	}
	interval := cfg.CacheRefreshInterval
	if interval == 0 {
		interval = 5 * time.Minute
	}
	a := &JWKSAuth{cfg: cfg}
	if cfg.JWKSUrl != "" {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		c := jwk.NewCache(context.Background())
		if err := c.Register(cfg.JWKSUrl,
			jwk.WithRefreshInterval(interval),
			jwk.WithHTTPClient(httpClient),
		); err != nil {
			return nil, err
		}
		// Pre-fetch; ignore error so startup doesn't hard-fail on transient
		// network hiccups. The interceptor will reject requests until keys load.
		_, _ = c.Refresh(context.Background(), cfg.JWKSUrl)
		a.cache = c
	}
	return a, nil
}

// UnaryInterceptor returns a grpc.UnaryServerInterceptor.
func (a *JWKSAuth) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := a.authorize(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a grpc.StreamServerInterceptor.
func (a *JWKSAuth) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := a.authorize(ss.Context()); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func (a *JWKSAuth) authorize(ctx context.Context) error {
	if a.cfg.AllowInsecureLocal && a.cfg.ExplicitLocalMode {
		return nil
	}

	token, err := bearerToken(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	// Static token check.
	if a.cfg.StaticToken != "" {
		if token != a.cfg.StaticToken {
			return status.Error(codes.Unauthenticated, "invalid token")
		}
		return nil
	}

	// JWKS validation.
	if a.cache == nil {
		return status.Error(codes.Unauthenticated, "no auth backend configured")
	}
	keySet, err := a.cache.Get(ctx, a.cfg.JWKSUrl)
	if err != nil {
		return status.Error(codes.Unauthenticated, "unable to fetch signing keys")
	}
	if _, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	); err != nil {
		return status.Error(codes.Unauthenticated, "invalid token")
	}
	return nil
}

func bearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("missing Authorization header")
	}
	raw := values[0]
	if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return "", errors.New("authorization must use Bearer scheme")
	}
	tok := strings.TrimSpace(raw[7:])
	if tok == "" {
		return "", errors.New("empty bearer token")
	}
	return tok, nil
}
