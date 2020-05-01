package logger

import (
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func TestFilterFrames(t *testing.T) {
	rawStacktrace := &sentry.Stacktrace{Frames: []sentry.Frame{
		{
			Function: "Run.func2",
			Module:   "github.com/stretchr/testify/suite",
			AbsPath:  "/somewhere/github.com/stretchr/testify/suite/suite.go",
			InApp:    true,
		},
		{
			Function: "Value.Call",
			Module:   "reflect",
			AbsPath:  "/goroot/src/reflect/value.go",
			InApp:    false,
		},
		{
			Function: "Value.call",
			Module:   "reflect",
			AbsPath:  "/goroot/src/reflect/value.go",
			InApp:    false,
		},
		{
			Function: "go.pr0ger.dev/logger.(*SentryCoreSuite).TestWriteLevelFieldStoreExtraTags",
			Module:   "go.pr0ger.dev/logger",
			AbsPath:  "/somewhere/go.pr0ger.dev/logger/sentry_core_test.go",
			InApp:    true,
		},
		{
			Function: "go.uber.org/zap.(*Logger).Error",
			Module:   "go.uber.org/zap",
			AbsPath:  "/somewhere/go.uber.org/zap/logger.go",
			InApp:    true,
		},
		{
			Function: "go.uber.org/zap/zapcore.(*CheckedEntry).Write",
			Module:   "go.uber.org/zap",
			AbsPath:  "/somewhere/go.uber.org/zap/zapcore/entry.go",
			InApp:    true,
		},
		{
			Function: "go.pr0ger.dev/logger.(*SentryCore).Write",
			Module:   "go.uber.org/zap",
			AbsPath:  "/somewhere/go.pr0ger.dev/logger/sentry_core.go",
			InApp:    true,
		},
		{
			Function: "go.pr0ger.dev/logger.newStacktrace",
			Module:   "go.pr0ger.dev/logger",
			AbsPath:  "/somewhere/go.pr0ger.dev/logger/stacktrace.go",
			InApp:    true,
		},
	}}
	filteredStacktrace := &sentry.Stacktrace{Frames: []sentry.Frame{
		{
			Function: "Run.func2",
			Module:   "github.com/stretchr/testify/suite",
			AbsPath:  "/somewhere/github.com/stretchr/testify/suite/suite.go",
			InApp:    true,
		},
		{
			Function: "Value.Call",
			Module:   "reflect",
			AbsPath:  "/goroot/src/reflect/value.go",
			InApp:    false,
		},
		{
			Function: "Value.call",
			Module:   "reflect",
			AbsPath:  "/goroot/src/reflect/value.go",
			InApp:    false,
		},
		{
			Function: "go.pr0ger.dev/logger.(*SentryCoreSuite).TestWriteLevelFieldStoreExtraTags",
			Module:   "go.pr0ger.dev/logger",
			AbsPath:  "/somewhere/go.pr0ger.dev/logger/sentry_core_test.go",
			InApp:    true,
		},
	}}

	assert.Equal(t, filteredStacktrace, filterFrames(rawStacktrace))
}
