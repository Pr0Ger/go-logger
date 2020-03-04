package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BreadcrumbTransportSuite struct {
	suite.Suite

	ts            *httptest.Server
	sendEventMock *mock.Call
}

func (suite *BreadcrumbTransportSuite) SetupSuite() {
	suite.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func (suite *BreadcrumbTransportSuite) TearDownSuite() {
	suite.ts.Close()
}

func (suite *BreadcrumbTransportSuite) SetupTest() {
	transportMock := sentryTransport()
	client, err := sentry.NewClient(sentry.ClientOptions{Transport: transportMock})
	suite.NoError(err)

	sentry.CurrentHub().BindClient(client)
	suite.sendEventMock = transportMock.On("SendEvent", mock.AnythingOfType("*sentry.Event"))
}

func (suite *BreadcrumbTransportSuite) TearDownTest() {
	sentry.CurrentHub().Scope().ClearBreadcrumbs()
}

func (suite *BreadcrumbTransportSuite) TestNew() {
	suite.T().Run("fallback to default transport", func(t *testing.T) {
		transport := NewBreadcrumbTransport(sentry.LevelDebug, nil)
		require.Equal(t, transport.(*breadcrumbTransport).Transport, http.DefaultTransport)
	})
}

func (suite *BreadcrumbTransportSuite) TestRoundTripSuccess() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)
		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Assert().Equal(BreadcrumbTypeHTTP, breadcrumb.Type, "breadcrumb should have http type")

		expectedData := map[string]interface{}{
			BreadcrumbDataMethod:     "GET",
			BreadcrumbDataReason:     "204 No Content",
			BreadcrumbDataStatusCode: 204,
			BreadcrumbDataURL:        suite.ts.URL,
		}
		suite.Assert().Equal(expectedData, breadcrumb.Data, "breadcrumb should have data about http request")
	})

	client := http.Client{
		Transport: NewBreadcrumbTransport(sentry.LevelDebug, nil),
	}

	resp, err := client.Get(suite.ts.URL)
	suite.Require().NoError(err, "request should be success")
	defer resp.Body.Close()

	sentry.CaptureMessage("test event")
	sentry.Flush(1 * time.Second)
}

func (suite *BreadcrumbTransportSuite) TestRoundTripFailure() {
	suite.sendEventMock.Run(func(args mock.Arguments) {
		event := args.Get(0).(*sentry.Event)
		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Assert().Equal(BreadcrumbTypeHTTP, breadcrumb.Type, "breadcrumb should have http type")
		suite.Assert().Equal("dial tcp 127.0.0.1:21: connect: connection refused", breadcrumb.Message)

		expectedData := map[string]interface{}{
			BreadcrumbDataMethod: "GET",
			BreadcrumbDataURL:    "http://127.0.0.1:21",
		}
		suite.Assert().Equal(expectedData, breadcrumb.Data, "breadcrumb should have data about http request")
	})

	client := http.Client{
		Transport: NewBreadcrumbTransport(sentry.LevelDebug, nil),
	}

	_, err := client.Get("http://127.0.0.1:21") // nolint:bodyclose
	suite.Require().Error(err, "request should not be success")

	sentry.CaptureMessage("test event")
	sentry.Flush(1 * time.Second)
}

func TestBreadcrumbTransport(t *testing.T) {
	suite.Run(t, new(BreadcrumbTransportSuite))
}
