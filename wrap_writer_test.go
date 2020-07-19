package logger

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type WrapWriterSuite struct {
	suite.Suite
}

func (s WrapWriterSuite) TestImplements() {
	s.Implements((*http.Flusher)(nil), new(flushWriter))

	s.Implements((*http.Flusher)(nil), new(httpFancyWriter))
	s.Implements((*http.Hijacker)(nil), new(httpFancyWriter))
	s.Implements((*io.ReaderFrom)(nil), new(httpFancyWriter))

	s.Implements((*http.Flusher)(nil), new(http2FancyWriter))
	s.Implements((*http.Pusher)(nil), new(http2FancyWriter))
}

func (s WrapWriterSuite) TestBasicWrapperReturnsCodeAndBodyLength() {
	f := &basicWriter{ResponseWriter: httptest.NewRecorder()}
	f.WriteHeader(http.StatusCreated)

	_, err := f.Write([]byte{0x00, 0x01})
	s.NoError(err)

	s.Equal(http.StatusCreated, f.Status())
	s.EqualValues(2, f.BytesWritten())
}

func (s WrapWriterSuite) TestBasicWrapperHaveDefaultStatusCode() {
	f := &basicWriter{ResponseWriter: httptest.NewRecorder()}

	_, err := f.Write([]byte{0x00, 0x01})
	s.NoError(err)

	s.Equal(http.StatusOK, f.Status())
}

func (s WrapWriterSuite) TestFlushWriterRemembersWroteHeaderWhenFlushed() {
	f := &flushWriter{basicWriter{ResponseWriter: httptest.NewRecorder()}}
	f.Flush()

	s.True(f.wroteHeader, "want Flush to have set wroteHeader=true")
}

func (s WrapWriterSuite) TestHttpFancyWriterRemembersWroteHeaderWhenFlushed() {
	f := &httpFancyWriter{basicWriter{ResponseWriter: httptest.NewRecorder()}}
	f.Flush()

	s.True(f.wroteHeader, "want Flush to have set wroteHeader=true")
}

func (s WrapWriterSuite) TestHttp2FancyWriterRemembersWroteHeaderWhenFlushed() {
	f := &http2FancyWriter{basicWriter{ResponseWriter: httptest.NewRecorder()}}
	f.Flush()

	s.True(f.wroteHeader, "want Flush to have set wroteHeader=true")
}

func TestWrapWriter(t *testing.T) {
	suite.Run(t, new(WrapWriterSuite))
}
