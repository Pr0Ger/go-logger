package logger

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type sentryTransportMock struct {
	mock.Mock
}

func (s *sentryTransportMock) Flush(timeout time.Duration) bool {
	args := s.Called(timeout)
	return args.Bool(0)
}

func (s *sentryTransportMock) Configure(options sentry.ClientOptions) {
	s.Called(options)
}

func (s *sentryTransportMock) SendEvent(event *sentry.Event) {
	s.Called(event)
}

func sentryTransport() *sentryTransportMock {
	transport := &sentryTransportMock{}
	transport.On("Configure", mock.Anything).Return()
	transport.On("Flush", mock.Anything).Return(true)
	return transport
}

func sentryHubMock(t *testing.T) *sentry.Hub {
	client, err := sentry.NewClient(sentry.ClientOptions{
		Transport: sentryTransport(),
	})
	require.NoError(t, err)

	return sentry.NewHub(client, sentry.NewScope())
}
