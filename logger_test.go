package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type TestLoggerSuite struct {
	suite.Suite

	ctrl          *gomock.Controller
	sendEventMock *gomock.Call

	logger *zap.Logger
}

func (s *TestLoggerSuite) SetupTest() {
	s.logger = zap.New(zapcore.NewNopCore())

	s.ctrl = gomock.NewController(s.T())

	transportMock := NewMockTransport(s.ctrl)
	transportMock.EXPECT().
		Configure(gomock.AssignableToTypeOf(sentry.ClientOptions{})).
		Return()
	s.sendEventMock = transportMock.EXPECT().
		SendEvent(gomock.AssignableToTypeOf(&sentry.Event{})).
		Return().
		MinTimes(0)
	transportMock.EXPECT().
		Flush(gomock.Any()).
		Return(true).
		MinTimes(0)

	_ = sentry.Init(sentry.ClientOptions{
		Transport: transportMock,
	})
}

func (s *TestLoggerSuite) TearDownTest() {
	s.ctrl.Finish()

	// reset sentry client to default
	_ = sentry.Init(sentry.ClientOptions{})
}

func (s *TestLoggerSuite) wrapHandler(handler http.HandlerFunc) http.Handler {
	return RequestLogger(s.logger)(handler)
}

func (s *TestLoggerSuite) TestLoggerShouldSendEventToSentryAndReturnEventID() {
	s.logger = zap.New(NewSentryCoreWrapper(zapcore.NewNopCore(), sentry.CurrentHub()))

	eventID := sentry.EventID("<not valid>")
	s.sendEventMock.Do(func(event *sentry.Event) {
		eventID = event.EventID
	})

	wrappedHandler := s.wrapHandler(func(w http.ResponseWriter, r *http.Request) {
		Ctx(r.Context()).Error("test error")
	})

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	s.EqualValues(w.Header().Get("X-Sentry-Id"), eventID)
}

func (s *TestLoggerSuite) TestLoggerWithInjectedExtraFields() {
	s.logger = zap.New(NewSentryCoreWrapper(zapcore.NewNopCore(), sentry.CurrentHub()))

	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Ctx(r.Context()).Error("test error")
	})

	wrappedHandler := WithExtraFields(func(r *http.Request) []zap.Field {
		return []zap.Field{
			zap.String("key", r.URL.String()),
		}
	})(handlerFunc)
	wrappedHandler = RequestLogger(s.logger)(wrappedHandler)

	called := false
	s.sendEventMock.Do(func(event *sentry.Event) {
		called = true

		s.EqualValues("test error", event.Message)
		s.EqualValues("http://example.com/foo", event.Extra["key"])
	})

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)
	s.True(called)
}

func (s *TestLoggerSuite) TestForkedLoggerShouldOnlyLogRelatedEvents() {
	s.logger = zap.New(NewSentryCoreWrapper(zapcore.NewNopCore(), sentry.CurrentHub()))

	called := false
	s.sendEventMock.Do(func(event *sentry.Event) {
		called = true

		s.Require().Len(event.Breadcrumbs, 1)
		s.EqualValues("debug breadcrumb", event.Breadcrumbs[0].Message)

		s.EqualValues("error message to create event in sentry", event.Message)
	})

	s.logger.Debug("should not be sent")

	forkedLogger := ForkedLogger(s.logger)
	forkedLogger.Debug("debug breadcrumb")
	forkedLogger.Error("error message to create event in sentry")

	s.True(called)
}

func TestRequestLogger(t *testing.T) {
	suite.Run(t, new(TestLoggerSuite))
}
