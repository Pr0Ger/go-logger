package logger

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

type sentryHubCtxKey struct{}
type zapLoggerCtxKey struct{}

func Hub(ctx context.Context) *sentry.Hub {
	if hub, ok := ctx.Value(sentryHubCtxKey{}).(*sentry.Hub); ok {
		return hub
	}
	return sentry.CurrentHub()
}

func WithHub(ctx context.Context, hub *sentry.Hub) context.Context {
	return context.WithValue(ctx, sentryHubCtxKey{}, hub)
}

// GetLogEntry returns the in-context LogEntry for a request.
func Ctx(ctx context.Context) *zap.Logger {
	if entry, ok := ctx.Value(zapLoggerCtxKey{}).(*zap.Logger); ok {
		return entry
	}
	return zap.NewNop()
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, zapLoggerCtxKey{}, logger)
}
