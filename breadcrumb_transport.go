package logger

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

type breadcrumbTransport struct {
	Level sentry.Level

	Transport http.RoundTripper
}

func NewBreadcrumbTransport(level sentry.Level, transport http.RoundTripper) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &breadcrumbTransport{
		Level:     level,
		Transport: transport,
	}
}

func (b breadcrumbTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	breadcrumb := sentry.Breadcrumb{
		Data: map[string]interface{}{
			BreadcrumbDataURL:    req.URL.String(),
			BreadcrumbDataMethod: req.Method,
		},
		Level:     b.Level,
		Timestamp: time.Now().Unix(),
		Type:      BreadcrumbTypeHTTP,
	}

	resp, err := b.Transport.RoundTrip(req)

	if err == nil {
		breadcrumb.Data[BreadcrumbDataStatusCode] = resp.StatusCode
		breadcrumb.Data[BreadcrumbDataReason] = resp.Status
	} else {
		breadcrumb.Message = err.Error()
	}

	Hub(req.Context()).AddBreadcrumb(&breadcrumb, nil)

	return resp, err
}
