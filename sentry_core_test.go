package logger

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate mockgen -package logger -destination mock_sentry_test.go github.com/getsentry/sentry-go Transport

type SentryCoreSuite struct {
	suite.Suite

	ctrl *gomock.Controller

	hub           *sentry.Hub
	sendEventMock *gomock.Call
}

func (suite *SentryCoreSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	transportMock := NewMockTransport(suite.ctrl)
	transportMock.EXPECT().
		Configure(gomock.AssignableToTypeOf(sentry.ClientOptions{})).
		Return()
	suite.sendEventMock = transportMock.EXPECT().
		SendEvent(gomock.AssignableToTypeOf(&sentry.Event{})).
		Return().
		MinTimes(0)
	transportMock.EXPECT().
		Flush(gomock.Any()).
		Return(true).
		MinTimes(0)

	client, err := sentry.NewClient(sentry.ClientOptions{Transport: transportMock})
	suite.NoError(err)
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
	})
}

func (suite *SentryCoreSuite) TestWriteLevelStoreBreadcrumbMessage() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Assert().Equal(BreadcrumbTypeDefault, breadcrumb.Type)
		suite.Assert().Equal(sentry.LevelDebug, breadcrumb.Level)
		suite.Assert().Equal("test", breadcrumb.Message)
	}).MinTimes(1)

	core := NewSentryCore(suite.hub)
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelStoreFieldForEvents() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
		suite.Require().Len(event.Extra, 2)
	}).MinTimes(1)

	core := NewSentryCore(suite.hub)
	logger := zap.New(core).With(zap.Int("global", 1))

	logger.Error("event", zap.Int("local", 1))
}

func (suite *SentryCoreSuite) TestWriteLevelFieldForwarding() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
		suite.Require().Len(event.Breadcrumbs, 2, "event should have 2 breadcrumbs")
		suite.Equal("event without extra tags", event.Breadcrumbs[0].Message)
		suite.Len(event.Breadcrumbs[0].Data, 0)

		suite.Equal("event with extra tag", event.Breadcrumbs[1].Message)
		suite.Require().Len(event.Breadcrumbs[1].Data, 1)
		suite.EqualValues(2, event.Breadcrumbs[1].Data["tag"])
	}).MinTimes(1)

	core := NewSentryCore(suite.hub)
	logger := zap.New(core).With(zap.Int("global", 1))

	logger.Debug("event without extra tags")
	logger.Debug("event with extra tag", zap.Int("tag", 2))

	logger.Error("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelSkip() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
		suite.Require().Len(event.Breadcrumbs, 0, "event should not have breadcrumbs")
	})

	core := &SentryCore{hub: suite.hub, BreadcrumbLevel: zap.InfoLevel, EventLevel: zap.InfoLevel}
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteOnFatalLevelsTriggerSync() {
	logger := zap.New(NewSentryCore(suite.hub))

	suite.Panics(func() {
		// panic is used because we can't override os.exit(1)
		logger.Panic("panic msg")
	})
}

func TestSentryCore(t *testing.T) {
	suite.Run(t, new(SentryCoreSuite))
}
