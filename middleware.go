package logger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Middleware struct {
	localCore      zapcore.Core
	sentryCoreOpts []SentryCoreOption

	client *sentry.Client
	hub    *sentry.Hub
}

func (h *Middleware) Handle(handler http.Handler) http.Handler {
	return h.handle(handler)
}

func (h *Middleware) HandleFunc(handler http.HandlerFunc) http.HandlerFunc {
	return h.handle(handler)
}

func (h *Middleware) handle(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ww := NewWrapResponseWriter(w, r.ProtoMajor)

		var hub *sentry.Hub
		var span *sentry.Span
		var loggerOptions []zap.Option
		core := h.localCore
		if h.client != nil {
			hub = sentry.NewHub(h.client, sentry.NewScope())
			hub.Scope().SetRequest(r)

			ctx = WithHub(ctx, hub)

			span = sentry.StartSpan(ctx, "http.handler",
				sentry.TransactionName(fmt.Sprintf("%s %s", r.Method, r.URL.Path)),
				sentry.ContinueFromRequest(r),
			)
			ctx = span.Context()

			core = NewSentryCoreWrapper(h.localCore, hub, h.sentryCoreOpts...)

			loggerOptions = append(loggerOptions, zap.Hooks(func(entry zapcore.Entry) error {
				if entry.Level >= core.(sentryCoreWrapper).SentryCore().EventLevel && hub.LastEventID() != "" {
					ww.Header().Add(sentryEventIDHeader, string(hub.LastEventID()))
				}
				return nil
			}))
		}

		logger := zap.New(core, loggerOptions...)
		ctx = WithLogger(ctx, logger)

		t1 := time.Now()
		defer func() {
			if span != nil {
				span.Status = SpanStatus(ww.Status())
				span.Finish()
			}
			logger.Debug("-",
				zap.Duration("duration", time.Since(t1)),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.String("ip", r.RemoteAddr),
			)
		}()

		handler.ServeHTTP(w, r)
	}
}
