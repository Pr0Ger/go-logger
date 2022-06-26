package logger

import (
	"math"
	"net/http"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestSentryLevel(t *testing.T) {
	tests := []struct {
		arg  zapcore.Level
		want sentry.Level
	}{
		{zapcore.DebugLevel, sentry.LevelDebug},
		{zapcore.InfoLevel, sentry.LevelInfo},
		{zapcore.WarnLevel, sentry.LevelWarning},
		{zapcore.ErrorLevel, sentry.LevelError},
		{zapcore.DPanicLevel, sentry.LevelError},
		{zapcore.PanicLevel, sentry.LevelFatal},
		{zapcore.FatalLevel, sentry.LevelFatal},
		{zapcore.Level(math.MaxInt8), sentry.LevelDebug},
	}

	for _, tt := range tests {
		//nolint:scopelint
		t.Run(tt.arg.String(), func(t *testing.T) {
			res := SentryLevel(tt.arg)
			assert.Equal(t, tt.want, res, "SentryLevel() = %v, want %v", res, tt.want)
		})
	}
}

func TestSpanStatus(t *testing.T) {
	tests := []struct {
		arg  int
		want sentry.SpanStatus
	}{
		{http.StatusOK, sentry.SpanStatusOK},
		{http.StatusBadRequest, sentry.SpanStatusInvalidArgument},
		{http.StatusUnauthorized, sentry.SpanStatusUnauthenticated},
		{http.StatusForbidden, sentry.SpanStatusPermissionDenied},
		{http.StatusNotFound, sentry.SpanStatusNotFound},
		{http.StatusConflict, sentry.SpanStatusAlreadyExists},
		{499, sentry.SpanStatusCanceled},
		{http.StatusInternalServerError, sentry.SpanStatusInternalError},
		{http.StatusNotImplemented, sentry.SpanStatusUnimplemented},
		{http.StatusServiceUnavailable, sentry.SpanStatusUnavailable},
		{http.StatusGatewayTimeout, sentry.SpanStatusDeadlineExceeded},
	}

	for _, tt := range tests {
		//nolint:scopelint
		t.Run(http.StatusText(tt.arg), func(t *testing.T) {
			res := SpanStatus(tt.arg)
			assert.Equal(t, tt.want, res, "SentryLevel() = %v, want %v", res, tt.want)
		})
	}
}
