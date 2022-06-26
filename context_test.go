package logger

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type TestContextHelpersSuite struct {
	suite.Suite
}

func (s *TestContextHelpersSuite) TestRequestIDDefault() {
	s.Equal("", RequestID(context.TODO()))
}

func (s *TestContextHelpersSuite) TestRequestIDStoring() {
	requestID := "request_id"

	ctx := WithRequestID(context.Background(), requestID)
	s.Equal(requestID, RequestID(ctx))
}

func (s *TestContextHelpersSuite) TestHubDefault() {
	s.Equal(sentry.CurrentHub(), Hub(context.TODO()))
}

func (s *TestContextHelpersSuite) TestHubStoring() {
	hub := sentry.CurrentHub().Clone()

	ctx := WithHub(context.Background(), hub)
	s.Equal(hub, Hub(ctx))
}

func (s *TestContextHelpersSuite) TestLoggerDefault() {
	s.Equal(zap.NewNop(), Ctx(context.TODO()))
}

func (s *TestContextHelpersSuite) TestLoggerStoring() {
	logger := zap.NewExample()

	ctx := WithLogger(context.Background(), logger)
	s.Equal(logger, Ctx(ctx))
}

func TestContextHelpers(t *testing.T) {
	suite.Run(t, new(TestContextHelpersSuite))
}
