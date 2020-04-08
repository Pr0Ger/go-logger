package logger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:generate mockgen -package logger -destination mock_sentry_test.go github.com/getsentry/sentry-go Transport

type BreadcrumbTransportSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	ts   *httptest.Server

	hub           *sentry.Hub
	sendEventMock *gomock.Call
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

func (suite *BreadcrumbTransportSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *BreadcrumbTransportSuite) TestNew() {
	suite.T().Run("fallback to default transport", func(t *testing.T) {
		transport := NewBreadcrumbTransport(sentry.LevelDebug, nil)
		require.Equal(t, transport.(*breadcrumbTransport).Transport, http.DefaultTransport)
	})
}

func (suite *BreadcrumbTransportSuite) TestRoundTripSuccess() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
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
	}).MinTimes(1)

	client := http.Client{
		Transport: NewBreadcrumbTransport(sentry.LevelDebug, nil),
	}

	ctx := WithHub(context.Background(), suite.hub)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, suite.ts.URL, nil)
	suite.Require().NoError(err)

	resp, err := client.Do(req)
	suite.Require().NoError(err, "request should be success")
	defer resp.Body.Close()

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func (suite *BreadcrumbTransportSuite) TestRoundTripFailure() {
	suite.sendEventMock.Do(func(event *sentry.Event) {
		suite.Require().Len(event.Breadcrumbs, 1, "event should have one breadcrumb")

		breadcrumb := event.Breadcrumbs[0]
		suite.Assert().Equal(BreadcrumbTypeHTTP, breadcrumb.Type, "breadcrumb should have http type")
		suite.Assert().Equal("dial tcp 127.0.0.1:21: connect: connection refused", breadcrumb.Message)

		expectedData := map[string]interface{}{
			BreadcrumbDataMethod: "GET",
			BreadcrumbDataURL:    "http://127.0.0.1:21",
		}
		suite.Assert().Equal(expectedData, breadcrumb.Data, "breadcrumb should have data about http request")
	}).MinTimes(1)

	client := http.Client{
		Transport: NewBreadcrumbTransport(sentry.LevelDebug, nil),
	}

	ctx := WithHub(context.Background(), suite.hub)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:21", nil)
	suite.Require().NoError(err)

	_, err = client.Do(req) // nolint:bodyclose
	suite.Require().Error(err, "request should not be success")

	suite.hub.CaptureMessage("test event")
	suite.hub.Flush(1 * time.Second)
}

func TestBreadcrumbTransport(t *testing.T) {
	suite.Run(t, new(BreadcrumbTransportSuite))
}
