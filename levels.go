package logger

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

func SentryLevel(level zapcore.Level) sentry.Level {
	switch level {
	case zapcore.DebugLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel, zapcore.DPanicLevel:
		return sentry.LevelError
	case zapcore.PanicLevel, zapcore.FatalLevel:
		return sentry.LevelFatal
	default:
		return sentry.LevelDebug
	}
}

func SpanStatus(httpCode int) sentry.SpanStatus {
	if http.StatusOK <= httpCode && httpCode < 299 {
		return sentry.SpanStatusOK
	}
	switch httpCode {
	case http.StatusBadRequest:
		return sentry.SpanStatusInvalidArgument
	case http.StatusUnauthorized:
		return sentry.SpanStatusUnauthenticated
	case http.StatusForbidden:
		return sentry.SpanStatusPermissionDenied
	case http.StatusNotFound:
		return sentry.SpanStatusNotFound
	case http.StatusConflict:
		return sentry.SpanStatusAlreadyExists
	case 499: // nginx specific: client has closed connection
		return sentry.SpanStatusCanceled
	case http.StatusInternalServerError:
		return sentry.SpanStatusInternalError
	case http.StatusNotImplemented:
		return sentry.SpanStatusUnimplemented
	case http.StatusServiceUnavailable:
		return sentry.SpanStatusUnavailable
	case http.StatusGatewayTimeout:
		return sentry.SpanStatusDeadlineExceeded
	default:
		return sentry.SpanStatusUnknown
	}
}
