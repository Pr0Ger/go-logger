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

//go:generate mockgen -package logger -destination mock_zapcore_test.go go.uber.org/zap/zapcore Core

type SentryCoreWrapperSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *SentryCoreWrapperSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *SentryCoreWrapperSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *SentryCoreWrapperSuite) TestNew() {
	localCore := NewMockCore(s.ctrl)

	wrappedCore := NewSentryCoreWrapper(localCore, sentry.CurrentHub()).(sentryCoreWrapper)
	s.Equal(localCore, wrappedCore.LocalCore())
	s.Equal(sentry.CurrentHub(), wrappedCore.SentryCore().hub)
}

func (s *SentryCoreWrapperSuite) TestCoreFunctions() {
	localCore := NewMockCore(s.ctrl)

	wrappedCore := NewSentryCoreWrapper(localCore, sentry.CurrentHub()).(sentryCoreWrapper)

	testEntry := zapcore.Entry{
		LoggerName: "test",
		Time:       time.Now(),
		Level:      zapcore.ErrorLevel,
		Message:    "testMessage",
	}
	fields := []zapcore.Field{
		zap.Int("key", 1337),
	}

	s.Run("Enabled", func() {
		localCore.EXPECT().Enabled(gomock.Any()).Return(true)

		s.Equal(true, wrappedCore.Enabled(zapcore.InfoLevel))
	})

	s.Run("With", func() {
		localCore.EXPECT().With(gomock.Eq(fields)).Return(localCore).Times(1)

		newCore := wrappedCore.With(fields)
		s.NotEqual(wrappedCore, newCore)
	})

	s.Run("Check", func() {
		localCore.EXPECT().
			Check(gomock.AssignableToTypeOf(testEntry), gomock.Any()).
			DoAndReturn(func(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
				return ce.AddCore(ent, localCore)
			})

		checkedEntry := wrappedCore.Check(testEntry, nil)
		s.Require().NotNil(testEntry)
		s.Equal(testEntry.Message, checkedEntry.Message)
	})

	s.Run("Write", func() {
		localCore.EXPECT().Write(gomock.Eq(testEntry), gomock.Eq(fields)).Return(nil)

		s.NoError(wrappedCore.Write(testEntry, fields))
	})

	s.Run("Sync", func() {
		localCore.EXPECT().Sync().Return(nil)
		s.NoError(wrappedCore.Sync())
	})
}

func TestSentryCoreWrapper(t *testing.T) {
	suite.Run(t, new(SentryCoreWrapperSuite))
}
