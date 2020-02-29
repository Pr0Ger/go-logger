package logger

// https://docs.sentry.io/development/sdk-dev/event-payloads/breadcrumbs/#breadcrumb-types
const (
	// BreadcrumbTypeDefault describes a generic breadcrumb
	BreadcrumbTypeDefault = "default"

	// BreadcrumbTypeHTTP describes an HTTP request breadcrumb
	BreadcrumbTypeHTTP = "http"
)

// Describes data for an HTTP request breadcrumb
// https://docs.sentry.io/development/sdk-dev/event-payloads/breadcrumbs/#http
const (
	BreadcrumbDataURL        = "url"
	BreadcrumbDataMethod     = "method"
	BreadcrumbDataStatusCode = "status_code"
	BreadcrumbDataReason     = "reason"
)
