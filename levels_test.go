package logger

import (
	"math"
	"strings"
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
		t.Run(strings.Title(tt.arg.String()), func(t *testing.T) {
			res := SentryLevel(tt.arg)
			assert.Equal(t, tt.want, res, "SentryLevel() = %v, want %v", res, tt.want)
		})
	}
}
