package logger

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

type sentryHubCtxKey struct{}
type zapLoggerCtxKey struct{}

// Hub returns the sentry.Hub associated with the context. If no hub is associated returns CurrentHub()
func Hub(ctx context.Context) *sentry.Hub {
	if hub, ok := ctx.Value(sentryHubCtxKey{}).(*sentry.Hub); ok {
		return hub
	}
	return sentry.CurrentHub()
}

// WithHub returns a copy of provided context with added hub field
func WithHub(ctx context.Context, hub *sentry.Hub) context.Context {
	return context.WithValue(ctx, sentryHubCtxKey{}, hub)
}

// Ctx returns the in-context Logger for a request. If no logger is associated returns returns no-op logger
func Ctx(ctx context.Context) *zap.Logger {
	if entry, ok := ctx.Value(zapLoggerCtxKey{}).(*zap.Logger); ok {
		return entry
	}
	return zap.NewNop()
}

// WithLogger returns a copy of provided context with added logger field
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, zapLoggerCtxKey{}, logger)
}