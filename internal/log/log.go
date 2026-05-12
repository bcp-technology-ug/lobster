// Package log provides context-aware structured logging for Lobster using zap.
//
// Usage:
//
//	// Inject at entry point:
//	ctx = log.WithLogger(ctx, logger)
//
//	// Retrieve anywhere:
//	log.FromContext(ctx).Info("message", zap.String("run_id", id))
//
// When no logger has been injected, FromContext returns a no-op logger so
// callers never need to nil-check.
package log

import (
	"context"

	"go.uber.org/zap"
)

type contextKey struct{}

// WithLogger returns a child context carrying logger.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger stored in ctx. Returns a no-op logger when
// none has been injected, ensuring callers never receive a nil pointer.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(contextKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.NewNop()
}

// With returns a child logger with additional fields attached and a context
// carrying the child logger. This is the canonical way to add request-scoped
// fields (run_id, workspace_id, etc.) to all subsequent log calls.
func With(ctx context.Context, fields ...zap.Field) (context.Context, *zap.Logger) {
	child := FromContext(ctx).With(fields...)
	return WithLogger(ctx, child), child
}
