package logger

import (
	"strings"

	"github.com/getsentry/sentry-go"
)

func extractStacktrace(err error) *sentry.Stacktrace {
	stacktrace := sentry.ExtractStacktrace(err)
	if stacktrace == nil {
		return nil
	}
	return filterFrames(stacktrace)
}

func newStacktrace() *sentry.Stacktrace {
	return filterFrames(sentry.NewStacktrace())
}

func filterFrames(stacktrace *sentry.Stacktrace) *sentry.Stacktrace {
	filteredFrames := make([]sentry.Frame, 0, len(stacktrace.Frames))
	for _, frame := range stacktrace.Frames {
		if strings.HasPrefix(frame.Module, "go.uber.org/zap") ||
			strings.HasPrefix(frame.Function, "go.uber.org/zap") {
			break
		}

		filteredFrames = append(filteredFrames, frame)
	}
	stacktrace.Frames = filteredFrames

	return stacktrace
}
