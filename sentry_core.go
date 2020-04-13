package logger

import (
	"reflect"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type SentryCore struct {
	zapcore.LevelEnabler

	hub   *sentry.Hub
	scope *sentry.Scope

	BreadcrumbLevel zapcore.Level
	EventLevel      zapcore.Level
}

type SentryCoreOption func(*SentryCore)

// BreadcrumbLevel will set a minimum level of messages should be stored as breadcrumbs
func BreadcrumbLevel(level zapcore.Level) SentryCoreOption {
	return func(w *SentryCore) {
		w.BreadcrumbLevel = level
		if level > w.EventLevel {
			w.EventLevel = level
		}
	}
}

// EventLevel will set a minimum level of messages should be sent as events
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
		hub:             hub,
		scope:           hub.PushScope(),
		BreadcrumbLevel: zapcore.DebugLevel,
		EventLevel:      zapcore.ErrorLevel,
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

		var stacktrace *sentry.Stacktrace
		if errField != nil {
			stacktrace = sentry.ExtractStacktrace(errField)
		}
		if stacktrace == nil {
			stacktrace = sentry.NewStacktrace()
		}
		filteredFrames := make([]sentry.Frame, 0, len(stacktrace.Frames))
		for _, frame := range stacktrace.Frames {
			if strings.HasPrefix(frame.Module, "go.uber.org/zap") ||
				strings.HasPrefix(frame.Function, "go.uber.org/zap") {
				break
			}

			filteredFrames = append(filteredFrames, frame)
		}
		stacktrace.Frames = filteredFrames

		if errField != nil {
			cause := errors.Cause(errField)

			event.Exception = []sentry.Exception{{
				Value:      cause.Error(),
				Type:       reflect.TypeOf(cause).String(),
				ThreadID:   "current",
				Stacktrace: stacktrace,
			}}
			event.Threads = []sentry.Thread{{
				ID:      "current",
				Current: true,
				Crashed: ent.Level >= zapcore.DPanicLevel,
			}}
		} else {
			event.Threads = []sentry.Thread{{
				ID:         "current",
				Stacktrace: stacktrace,
				Current:    true,
				Crashed:    ent.Level >= zapcore.DPanicLevel,
			}}
		}

		s.hub.CaptureEvent(event)
	}

	breadcrumb := sentry.Breadcrumb{
		Data:      data.Fields,
		Level:     SentryLevel(ent.Level),
		Message:   ent.Message,
		Timestamp: time.Now().Unix(),
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
