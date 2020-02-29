package logger

import (
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

type SentryCoreWrapper struct {
	zapcore.Core

	localCore  zapcore.Core
	sentryCore *SentryCore
}

func NewSentryCoreWrapper(localCore zapcore.Core, hub *sentry.Hub, options ...SentryCoreOption) zapcore.Core {
	sentryCore := NewSentryCore(hub, options...).(*SentryCore)
	return SentryCoreWrapper{
		Core:       zapcore.NewTee(localCore, sentryCore),
		localCore:  localCore,
		sentryCore: sentryCore,
	}
}

func (w SentryCoreWrapper) LocalCore() zapcore.Core {
	return w.localCore
}
