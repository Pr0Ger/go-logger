package logger

import (
	"github.com/getsentry/sentry-go"
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

// sentryCoreWrapper is something like multiCore but only for two cores when second core is a Sentry core
type sentryCoreWrapper [2]zapcore.Core

// NewSentryCoreWrapper creates a Core that duplicates log entries into
// provided local Core and implicitly created Sentry core
func NewSentryCoreWrapper(localCore zapcore.Core, hub *sentry.Hub, options ...SentryCoreOption) zapcore.Core {
	return sentryCoreWrapper{
		localCore,
		NewSentryCore(hub, options...),
	}
}

func (w sentryCoreWrapper) LocalCore() zapcore.Core {
	return w[0]
}

func (w sentryCoreWrapper) SentryCore() *SentryCore {
	return w[1].(*SentryCore)
}

func (w sentryCoreWrapper) Enabled(lvl zapcore.Level) bool {
	return w[0].Enabled(lvl) || w[1].Enabled(lvl)
}

func (w sentryCoreWrapper) With(fields []zapcore.Field) zapcore.Core {
	return sentryCoreWrapper{
		w.LocalCore().With(fields),
		w.SentryCore().With(fields),
	}
}

func (w sentryCoreWrapper) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return w[0].Check(ent, w[1].Check(ent, ce))
}

func (w sentryCoreWrapper) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return multierr.Append(
		w[0].Write(ent, fields),
		w[1].Write(ent, fields),
	)
}

func (w sentryCoreWrapper) Sync() error {
	return multierr.Append(
		w[0].Sync(),
		w[1].Sync(),
	)
}
