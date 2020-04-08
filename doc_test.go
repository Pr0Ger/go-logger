package logger

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func ExampleSimple() {
	// This will not work without SENTRY_DSN environment variable
	_ = sentry.Init(sentry.ClientOptions{
		Transport: sentry.NewHTTPSyncTransport(),
	})

	// Create logger for logging directly to sentry (without local output)
	log := zap.New(NewSentryCore(sentry.CurrentHub()))

	log.Debug("this message will be logged as breadcrumb", zap.Int("key", 1337))
	log.Error("and this will create event in sentry")

	log.Error("and this message will attach stacktrace", zap.Error(errors.New("error from pkg/errors")))
}

func ExampleBreadcrumbTransport() {
	// This will not work without SENTRY_DSN environment variable
	_ = sentry.Init(sentry.ClientOptions{
		Transport: sentry.NewHTTPSyncTransport(),
	})

	// Create non-default http-client with
	client := http.Client{
		Transport: NewBreadcrumbTransport(sentry.LevelDebug, nil),
	}

	// Create context with sentry.Hub
	// This is not required: if hub is not available from context sentry.CurrentHub() will be used instead
	ctx := WithHub(context.Background(), sentry.CurrentHub())

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://go.pr0ger.dev/", nil)
	resp, err := client.Do(req)
	if err != nil {
		// Either send error to sentry
		sentry.CaptureException(err)
		return
	}
	defer resp.Body.Close()

	// Or just log response
	sentry.CaptureMessage(fmt.Sprintf("Response status: %s", resp.Status))

	// Either way it will contain full info about request in breadcrumb
}

func ExampleWebServer() {
	// This will not work without SENTRY_DSN environment variable
	_ = sentry.Init(sentry.ClientOptions{
		Transport: sentry.NewHTTPSyncTransport(),
	})

	// Create core for logging to stdout/stderr
	localCore := NewCore(true)

	// Create core splitter to logging both to local and sentry
	// zapcore.NewTee also can be used, but is not recommended if you want to use RequestLogger middleware
	core := NewSentryCoreWrapper(localCore, sentry.CurrentHub())

	// And create logger
	logger := zap.New(core)

	logger.Debug("this is event will be logged to stdout but will not appear in request breadcrumbs")

	// Create handler for network requests
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := Ctx(r.Context())

		log.Debug("some debug logs from request")

		_, _ = w.Write([]byte("ok"))

		log.Error("let's assume we have an error here")
	})

	// And use it with our middleware
	server := &http.Server{
		Addr:    ":8080",
		Handler: RequestLogger(logger)(handler),
	}

	_ = server.ListenAndServe()
}
