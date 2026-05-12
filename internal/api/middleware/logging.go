package middleware

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	lobsterlog "github.com/bcp-technology-ug/lobster/internal/log"
)

// Logging returns a gRPC unary interceptor that injects a request-scoped zap
// logger into the context. Downstream handlers retrieve it with log.FromContext.
//
// Fields attached per-request:
//   - grpc.method: the full RPC method string (e.g. "/lobster.v1.run.RunService/RunSync")
//   - workspace_id: extracted from incoming "x-workspace-id" gRPC metadata when present
//
// The interceptor is intentionally non-intrusive: it never modifies the response
// or causes request failures.
func Logging(base *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx = injectLogger(ctx, base, info.FullMethod)
		return handler(ctx, req)
	}
}

// LoggingStream returns a gRPC stream interceptor equivalent.
func LoggingStream(base *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := injectLogger(ss.Context(), base, info.FullMethod)
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// injectLogger builds a child logger with method + optional workspace_id fields
// and stores it in ctx via log.WithLogger.
func injectLogger(ctx context.Context, base *zap.Logger, method string) context.Context {
	fields := []zap.Field{zap.String("grpc.method", method)}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-workspace-id"); len(vals) > 0 && vals[0] != "" {
			fields = append(fields, zap.String("workspace_id", vals[0]))
		}
	}
	return lobsterlog.WithLogger(ctx, base.With(fields...))
}

// wrappedStream replaces the context on an existing grpc.ServerStream.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }
