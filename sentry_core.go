package logger

import (
	"reflect"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

const (
	defaultBreadcrumbLevel = zapcore.DebugLevel
	defaultEventLevel      = zapcore.ErrorLevel
)

type SentryCore struct {
	zapcore.LevelEnabler

	hub   *sentry.Hub
	scope *sentry.Scope

	BreadcrumbLevel zapcore.Level
	EventLevel      zapcore.Level
}

type SentryCoreOption func(*SentryCore)

// BreadcrumbLevel will set a minimum level of messages should be stored as breadcrumbs.
func BreadcrumbLevel(level zapcore.Level) SentryCoreOption {
	return func(w *SentryCore) {
		w.BreadcrumbLevel = level
		w.LevelEnabler = level
		if level > w.EventLevel {
			w.EventLevel = level
		}
	}
}

// EventLevel will set a minimum level of messages should be sent as events.
func EventLevel(level zapcore.Level) SentryCoreOption {
	return func(w *SentryCore) {
		w.EventLevel = level
	}
}

func NewSentryCore(hub *sentry.Hub, options ...SentryCoreOption) zapcore.Core {
	if hub == nil {
		panic("hub should not be nil")
	}

	core := &SentryCore{
		LevelEnabler:    defaultBreadcrumbLevel,
		hub:             hub,
		scope:           hub.PushScope(),
		BreadcrumbLevel: defaultBreadcrumbLevel,
		EventLevel:      defaultEventLevel,
	}

	for _, option := range options {
		option(core)
	}

	return core
}

func (s *SentryCore) With(fields []zapcore.Field) zapcore.Core {
	clone := &SentryCore{
		LevelEnabler:    s.LevelEnabler,
		hub:             s.hub,
		scope:           s.hub.PushScope(),
		BreadcrumbLevel: s.BreadcrumbLevel,
		EventLevel:      s.EventLevel,
	}

	data := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(data)
	}
	clone.scope.SetExtras(data.Fields)

	return clone
}

func (s *SentryCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if ent.Level >= s.BreadcrumbLevel {
		ce = ce.AddCore(ent, s)
	}
	return ce
}

func (s *SentryCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	data := zapcore.NewMapObjectEncoder()
	var errField error
	for _, field := range fields {
		if field.Type == zapcore.ErrorType {
			errField = field.Interface.(error)
		} else {
			field.AddTo(data)
		}
	}

	if ent.Level >= s.EventLevel {
		event := sentry.NewEvent()
		event.Level = SentryLevel(ent.Level)
		event.Message = ent.Message
		event.Extra = data.Fields

		for i := 0; i < 10 && errField != nil; i++ {
			event.Exception = append(event.Exception, sentry.Exception{
				Value:      errField.Error(),
				Type:       reflect.TypeOf(errField).String(),
				Stacktrace: extractStacktrace(errField),
			})
			switch wrapped := errField.(type) { //nolint:errorlint
			case interface{ Unwrap() error }:
				errField = wrapped.Unwrap()
			case interface{ Cause() error }:
				errField = wrapped.Cause()
			default:
				errField = nil
			}
		}

		if len(event.Exception) != 0 {
			if event.Exception[0].Stacktrace == nil {
				event.Exception[0].Stacktrace = newStacktrace()
			}
			event.Exception[0].ThreadID = "current"
			event.Threads = []sentry.Thread{{
				ID:      "current",
				Current: true,
				Crashed: ent.Level >= zapcore.DPanicLevel,
			}}
		} else {
			event.Threads = []sentry.Thread{{
				ID:         "current",
				Stacktrace: newStacktrace(),
				Current:    true,
				Crashed:    ent.Level >= zapcore.DPanicLevel,
			}}
		}

		// event.Exception should be sorted such that the most recent error is last
		for i := len(event.Exception)/2 - 1; i >= 0; i-- {
			opp := len(event.Exception) - 1 - i
			event.Exception[i], event.Exception[opp] = event.Exception[opp], event.Exception[i]
		}

		s.hub.CaptureEvent(event)
	}

	breadcrumb := sentry.Breadcrumb{
		Data:      data.Fields,
		Level:     SentryLevel(ent.Level),
		Message:   ent.Message,
		Timestamp: time.Now().UTC(),
		Type:      BreadcrumbTypeDefault,
	}
	s.hub.AddBreadcrumb(&breadcrumb, nil)

	if ent.Level > zapcore.ErrorLevel {
		_ = s.Sync()
	}

	return nil
}

func (s *SentryCore) Sync() error {
	s.hub.Flush(30 * time.Second)
	return nil
}
