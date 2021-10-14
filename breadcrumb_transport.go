package logger

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

type breadcrumbTransport struct {
	Transport http.RoundTripper

	Level sentry.Level
}

func NewBreadcrumbTransport(level sentry.Level, transport http.RoundTripper) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &breadcrumbTransport{
		Transport: transport,
		Level:     level,
	}
}

func (b breadcrumbTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	span := sentry.StartSpan(req.Context(), req.URL.String(), sentry.ContinueFromRequest(req))
	defer span.Finish()

	req.Header.Add("sentry-trace", span.ToSentryTrace())

	breadcrumb := sentry.Breadcrumb{
		Data: map[string]interface{}{
			BreadcrumbDataURL:    req.URL.String(),
			BreadcrumbDataMethod: req.Method,
		},
		Level:     b.Level,
		Timestamp: time.Now().UTC(),
		Type:      BreadcrumbTypeHTTP,
	}

	resp, err := b.Transport.RoundTrip(req.WithContext(span.Context()))

	if err == nil {
		breadcrumb.Data[BreadcrumbDataStatusCode] = resp.StatusCode
		breadcrumb.Data[BreadcrumbDataReason] = resp.Status
	} else {
		breadcrumb.Message = err.Error()
	}

	Hub(span.Context()).AddBreadcrumb(&breadcrumb, nil)

	return resp, err //nolint:wrapcheck
}
