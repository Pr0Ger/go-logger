package logger

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type SentryCoreSuite struct {
	suite.Suite

	hub           *sentry.Hub
	sendEventMock *mock.Call
}

func (suite *SentryCoreSuite) SetupTest() {
	suite.hub = sentryHubMock(suite.T())
	transportMock := suite.hub.Client().Transport.(*sentryTransportMock)
	suite.sendEventMock = transportMock.On("SendEvent", mock.AnythingOfType("*sentry.Event"))
}

func (suite *SentryCoreSuite) TestWriteLevelStoreBreadcrumbMessage() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)

		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Assert().Equal(BreadcrumbTypeDefault, breadcrumb.Type)
		suite.Assert().Equal(sentry.LevelDebug, breadcrumb.Level)
		suite.Assert().Equal("test", breadcrumb.Message)
	})

	core := NewSentryCore(suite.hub)
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelStoreFieldForEvents() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)

		suite.Require().Len(event.Extra, 2)
	})

	core := NewSentryCore(suite.hub)
	logger := zap.New(core).With(zap.Int("global", 1))

	logger.Error("event", zap.Int("local", 1))
}

func (suite *SentryCoreSuite) TestWriteLevelFieldForwarding() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)

		suite.Require().Len(event.Breadcrumbs, 2, "event should have 2 breadcrumbs")
		suite.Equal("event without extra tags", event.Breadcrumbs[0].Message)
		suite.Len(event.Breadcrumbs[0].Data, 0)

		suite.Equal("event with extra tag", event.Breadcrumbs[1].Message)
		suite.Require().Len(event.Breadcrumbs[1].Data, 1)
		suite.EqualValues(2, event.Breadcrumbs[1].Data["tag"])
	})

	core := NewSentryCore(suite.hub)
	logger := zap.New(core).With(zap.Int("global", 1))

	logger.Debug("event without extra tags")
	logger.Debug("event with extra tag", zap.Int("tag", 2))

	logger.Error("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *SentryCoreSuite) TestWriteLevelSkip() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)

		suite.Require().Len(event.Breadcrumbs, 0, "event should not have breadcrumbs")
	})

	core := &SentryCore{hub: suite.hub, BreadcrumbLevel: zap.InfoLevel, EventLevel: zap.InfoLevel}
	logger := zap.New(core)

	logger.Debug("test")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func TestSentryCore(t *testing.T) {
	suite.Run(t, new(SentryCoreSuite))
}
