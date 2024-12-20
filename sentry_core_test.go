package logger

import (
	stderrors "errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate mockgen -package logger -destination mock_sentry_test.go github.com/getsentry/sentry-go Transport

type SentryCoreSuite struct {
	suite.Suite

	ctrl *gomock.Controller

	hub           *sentry.Hub
	sendEventMock func() *gomock.Call
	flushMock     *gomock.Call
}

func (suite *SentryCoreSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	transportMock := NewMockTransport(suite.ctrl)
	transportMock.EXPECT().
		Configure(gomock.AssignableToTypeOf(sentry.ClientOptions{})).
		Return()
	suite.sendEventMock = func() *gomock.Call {
		return transportMock.EXPECT().
			SendEvent(gomock.AssignableToTypeOf(&sentry.Event{})).
			Return()
	}
	suite.flushMock = transportMock.EXPECT().
		Flush(gomock.Any()).
		Return(true).
		MinTimes(0)

	client, err := sentry.NewClient(sentry.ClientOptions{Transport: transportMock})
	suite.Require().NoError(err)
	suite.hub = sentry.NewHub(client, sentry.NewScope())
}

func (suite *SentryCoreSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *SentryCoreSuite) TestNew() {
	suite.Run("defaults", func() {
		hub := NewSentryCore(suite.hub).(*SentryCore)

		suite.Equal(zapcore.ErrorLevel, hub.EventLevel)
		suite.Equal(zapcore.DebugLevel, hub.BreadcrumbLevel)
	})
	suite.Run("hub should not be nil", func() {
		suite.Panics(func() {
			NewSentryCore(nil)
		})
	})
	suite.Run("options", func() {
		suite.Run("breadcrumb level", func() {
			hub := NewSentryCore(suite.hub, BreadcrumbLevel(zapcore.InfoLevel)).(*SentryCore)

			suite.Equal(zapcore.InfoLevel, hub.BreadcrumbLevel)
		})

		suite.Run("event level", func() {
			hub := NewSentryCore(suite.hub, EventLevel(zapcore.WarnLevel)).(*SentryCore)

			suite.Equal(zapcore.WarnLevel, hub.EventLevel)
		})

		suite.Run("breadcrumb level will also update event level", func() {
			hub := NewSentryCore(suite.hub, BreadcrumbLevel(zapcore.PanicLevel)).(*SentryCore)

			suite.Equal(zapcore.PanicLevel, hub.BreadcrumbLevel)
			suite.Equal(zapcore.PanicLevel, hub.EventLevel)
		})

		suite.Run("sentry  user tag map", func() {
			tm := SentryUserTagMap{ID: "username"}
			hub := NewSentryCore(suite.hub, UserTags(tm)).(*SentryCore)

			suite.Equal(tm, hub.UserTags)
		})

		suite.Run("sentry generic tags", func() {
			hub := NewSentryCore(suite.hub, GenericTags("t1", "t2")).(*SentryCore)

			suite.Equal([]string{"t1", "t2"}, hub.GenericTags)
		})
	})
}

func (suite *SentryCoreSuite) TestWriteLevelStoreBreadcrumbMessage() {
	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Equal(BreadcrumbTypeDefault, breadcrumb.Type)
		suite.Equal(sentry.LevelDebug, breadcrumb.Level)
		suite.Equal("test", breadcrumb.Message)
	})

	core := NewSentryCore(suite.hub)
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelSkipTooVerboseMessages() {
	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Require().Empty(event.Breadcrumbs, "event should not have breadcrumbs")
	})

	core := NewSentryCore(suite.hub, BreadcrumbLevel(zap.InfoLevel))
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelFieldStoreExtraTags() {
	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal("test event", event.Message)
		suite.Equal(map[string]interface{}{
			"global":    int64(1),
			"event tag": int64(3),
		}, event.Extra)

		suite.Require().Len(event.Breadcrumbs, 2, "event should have 2 breadcrumbs")

		suite.Equal("event without extra tags", event.Breadcrumbs[0].Message)
		suite.Empty(event.Breadcrumbs[0].Data)

		suite.Equal("event with extra tag", event.Breadcrumbs[1].Message)
		suite.Require().Len(event.Breadcrumbs[1].Data, 1)
		suite.EqualValues(2, event.Breadcrumbs[1].Data["tag"])
	})

	core := NewSentryCore(suite.hub)
	logger := zap.New(core).With(zap.Int("global", 1))

	logger.Debug("event without extra tags")
	logger.Debug("event with extra tag", zap.Int("tag", 2))

	logger.Error("test event", zap.Int("event tag", 3))
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteOnFatalLevelsTriggerSync() {
	suite.sendEventMock()
	suite.flushMock.MinTimes(1)

	logger := zap.New(NewSentryCore(suite.hub))

	suite.Panics(func() {
		// panic is used because we can't override os.exit(1)
		logger.Panic("panic msg")
	})
}

func (suite *SentryCoreSuite) TestWriteWillAttachStacktrace() {
	core := NewSentryCore(suite.hub)
	logger := zap.New(core)

	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal("test message with default stacktrace", event.Message)

		suite.Empty(event.Exception)

		suite.Require().Len(event.Threads, 1)
		thread := event.Threads[0]
		suite.False(thread.Crashed)
		suite.True(thread.Current)
		suite.Equal("0", thread.ID)
		suite.NotNil(thread.Stacktrace)
	})
	logger.Error("test message with default stacktrace")

	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal("message from panic", event.Message)

		suite.Empty(event.Exception)

		suite.Require().Len(event.Threads, 1)
		thread := event.Threads[0]
		suite.True(thread.Crashed)
		suite.True(thread.Current)
		suite.Equal("0", thread.ID)
		suite.NotNil(thread.Stacktrace)
	})
	suite.Panics(func() {
		logger.Panic("message from panic")
	})

	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal("error with exception", event.Message)

		suite.Require().Len(event.Exception, 1)
		suite.Require().Len(event.Threads, 1)

		exception := event.Exception[0]
		suite.Equal("*errors.fundamental", exception.Type)
		suite.Equal("error from pkg/errors", exception.Value)
		suite.NotNil(exception.Stacktrace)

		thread := event.Threads[0]
		suite.False(thread.Crashed)
		suite.True(thread.Current)
		suite.Nil(thread.Stacktrace)
	})
	logger.Error("error with exception", zap.Error(errors.New("error from pkg/errors")))
}

func (suite *SentryCoreSuite) TestWriteChainedErrors() {
	core := NewSentryCore(suite.hub)
	logger := zap.New(core)

	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal("message with chained error", event.Message)

		suite.Len(event.Exception, 3)
		suite.Equal("*errors.errorString", event.Exception[0].Type)
		suite.Equal("simple error", event.Exception[0].Value)
		suite.Nil(event.Exception[0].Stacktrace)

		suite.Equal("*errors.withStack", event.Exception[1].Type)
		suite.Equal("simple error", event.Exception[1].Value)
		suite.NotNil(event.Exception[1].Stacktrace)

		suite.Equal("wrapped<*errors.withStack>", event.Exception[2].Type)
		suite.Equal("wrap with fmt.Errorf: simple error", event.Exception[2].Value)
		suite.NotNil(event.Exception[2].Stacktrace)

		suite.Require().Len(event.Threads, 1)
		thread := event.Threads[0]
		suite.False(thread.Crashed)
		suite.True(thread.Current)
		suite.Equal("0", thread.ID)
		suite.Nil(thread.Stacktrace)
	})

	err := stderrors.New("simple error") //nolint:err113
	err = errors.WithStack(err)
	err = fmt.Errorf("wrap with fmt.Errorf: %w", err)

	logger.Error("message with chained error", zap.Error(err))
}

func (suite *SentryCoreSuite) TestStrippingWrappedErrors() {
	core := NewSentryCore(suite.hub).(*SentryCore)

	err := stderrors.New("simple error") //nolint:err113
	err = fmt.Errorf("first wrap: %w", err)
	err = fmt.Errorf("second wrap: %w", err)

	exceptions := core.convertErrorToException(err)
	suite.Len(exceptions, 3)

	suite.Equal("wrapped<*errors.errorString>", exceptions[0].Type)
	suite.Equal("second wrap: first wrap: simple error", exceptions[0].Value)

	suite.Equal("wrapped<*errors.errorString>", exceptions[1].Type)
	suite.Equal("first wrap: simple error", exceptions[1].Value)

	suite.Equal("*errors.errorString", exceptions[2].Type)
	suite.Equal("simple error", exceptions[2].Value)
}

func (suite *SentryCoreSuite) TestParsingSentryTags() {
	userTags := SentryUserTagMap{ID: "username", Username: "tag_wil_not_be_present"}
	core := NewSentryCore(suite.hub, UserTags(userTags), GenericTags("t1", "t2")).(*SentryCore)
	logger := zap.New(core)
	testUser, testTagValue, testExtraValue := "test_user", 42, 69
	suite.sendEventMock().Do(func(event *sentry.Event) {
		suite.Equal(testUser, event.User.ID)
		suite.Equal(strconv.Itoa(testTagValue), event.Tags["t2"])
		suite.Len(event.Tags, 1)  // ["t1"]
		suite.Len(event.Extra, 1) // ["t3"]
		suite.Equal(testExtraValue, int(event.Extra["t3"].(int64)))
	})
	logger.Error(
		"message with fields",
		zap.String("username", testUser),
		zap.Int("t2", testTagValue),
		zap.Int("t3", testExtraValue),
	)
}

func TestSentryCore(t *testing.T) {
	suite.Run(t, new(SentryCoreSuite))
}
