package logger

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey struct{}

func NewCore(debug bool) zapcore.Core {
	var encoder zapcore.Encoder
	if debug {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	} else {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}

	return zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})),
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl < zapcore.ErrorLevel
		})),
	)
}

// GetLogEntry returns the in-context LogEntry for a request.
func Ctx(ctx context.Context) *zap.Logger {
	entry, ok := ctx.Value(contextKey{}).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}
	return entry
}

func RequestLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	core := logger.Core()
	rootHub := sentry.CurrentHub()
	if wrappedCore, ok := core.(SentryCoreWrapper); ok {
		core = wrappedCore.localCore
		rootHub = wrappedCore.sentryCore.hub
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := zap.New(zapcore.NewTee(core, NewSentryCore(rootHub)))

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				logger.Debug("",
					zap.Duration("duration", time.Since(t1)),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
				)
			}()

			next.ServeHTTP(ww, r.WithContext(context.WithValue(r.Context(), contextKey{}, logger)))
		})
	}
}
