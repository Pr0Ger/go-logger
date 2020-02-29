package logger

import (
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
