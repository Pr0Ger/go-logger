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

type TestRequestLoggerSuite struct {
	suite.Suite

	ctrl          *gomock.Controller
	sendEventMock *gomock.Call

	logger *zap.Logger
}

func (s *TestRequestLoggerSuite) SetupTest() {
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

func (s *TestRequestLoggerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *TestRequestLoggerSuite) wrapHandler(handler http.HandlerFunc) http.Handler {
	return RequestLogger(s.logger)(handler)
}

func (s *TestRequestLoggerSuite) TestLoggerShouldSendEventToSentryAndReturnEventID() {
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

func TestRequestLogger(t *testing.T) {
	suite.Run(t, new(TestRequestLoggerSuite))
}
