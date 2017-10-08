package producers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testProducer struct {
	htmlWriter io.Writer
	htmlName   string
	err        error
	panic      interface{}
}

func (p *testProducer) HTML(w io.Writer, name string) error {
	p.htmlName = name
	p.htmlWriter = w
	if p.panic != nil {
		panic(p.panic)
	}
	return p.err
}

func TestAddProducer_NilProducer_ErrorReturned(t *testing.T) {
	mux := NewServeMux("")
	err := mux.AddProducer("key1", nil)
	assert.EqualError(t, err, "Trying to add nil Producer with key key1")
}

func TestAddProducer_BlankKey_ErrorReturned(t *testing.T) {
	blankKeys := []string{"", " ", "  "}
	for _, key := range blankKeys {
		mux := NewServeMux("")
		err := mux.AddProducer(key, &testProducer{})
		assert.EqualError(t, err, "Producer's key cannot be blank")
	}
}

func TestAddProducer_ValidArgs_NilReturned(t *testing.T) {
	mux := NewServeMux("")
	err := mux.AddProducer("key1", &testProducer{})
	assert.NoError(t, err)
}

func TestAddProducer_AddKeyTwice_ErrorReturned(t *testing.T) {
	mux := NewServeMux("")
	_ = mux.AddProducer("key1", &testProducer{})
	err := mux.AddProducer("key1", &testProducer{})
	assert.EqualError(t, err, "Another Producer with key key1 is already registered")
}

func TestServeHTTP_WrongURL_StatusNotFoundWritten(t *testing.T) {
	wrongURLs := []string{"/key", "/", "/key/page1/page2"}
	for _, url := range wrongURLs {
		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		mux := NewServeMux("/")
		mux.AddProducer("key", &testProducer{})
		mux.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code, "url: %s", url)
		assert.Equal(t, "404 page not found\n", w.Body.String())
	}
}

func TestServeHTTP_UnmappedKeyInURL_StatusNotImplementedWritten(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key2/name", nil)
	w := httptest.NewRecorder()

	mux := NewServeMux("/")
	mux.AddProducer("key1", &testProducer{})
	mux.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "key2 is not supported\n", w.Body.String())
}

func TestServeHTTP_WrongMethod_StatusMethodNotAllowedWritten(t *testing.T) {
	wrongMethods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range wrongMethods {
		r := httptest.NewRequest(method, "/key/name", nil)
		w := httptest.NewRecorder()

		mux := NewServeMux("/")
		mux.AddProducer("key", &testProducer{})
		mux.ServeHTTP(w, r)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "method: %s", method)
		assert.Equal(t, fmt.Sprintf("%s is not allowed\n", method), w.Body.String())
	}
}

func TestServeHTTP_ValidRequest_StatusOKWritten(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()

	mux := NewServeMux("/")
	mux.AddProducer("key", &testProducer{})
	mux.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeHTTP_ValidRequest_ProducerInvoked(t *testing.T) {
	w := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/key1/name2", nil)
	r2 := httptest.NewRequest(http.MethodGet, "/key2/name3", nil)
	p1 := testProducer{}
	p2 := testProducer{}

	mux := NewServeMux("/")
	mux.AddProducer("key1", &p1)
	mux.AddProducer("key2", &p2)

	mux.ServeHTTP(w, r1)
	assert.Equal(t, "name2", p1.htmlName)
	assert.Equal(t, w, p1.htmlWriter)

	mux.ServeHTTP(w, r2)
	assert.Equal(t, "name3", p2.htmlName)
	assert.Equal(t, w, p2.htmlWriter)
}

func TestServeHTTP_ProducerErrorErrNotExist_StatusNotFoundWritten(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()
	p := testProducer{err: os.ErrNotExist}

	mux := NewServeMux("/")
	mux.AddProducer("key", &p)
	mux.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "404 page not found\n", w.Body.String())
}

func TestServeHTTP_ProducerError_StatusInternalServerErrorReturned(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()
	p := testProducer{err: errors.New("sad-but-true")}

	mux := NewServeMux("/")
	mux.AddProducer("key", &p)
	mux.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Can't produce output\n", w.Body.String())
}

func TestServeHTTP_ProducerError_ErrorWrittenToLog(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()
	p := testProducer{err: errors.New("sad-but-true")}
	logBuf := bytes.Buffer{}
	log.SetOutput(&logBuf)

	mux := NewServeMux("/")
	mux.AddProducer("key", &p)
	mux.ServeHTTP(w, r)

	assert.Contains(t, logBuf.String(), "[ERROR] sad-but-true")
}

func TestServeHTTP_ProducerPaniced_StatusInternalServerErrorReturned(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()
	p := testProducer{panic: "it-happens"}

	mux := NewServeMux("/")
	mux.AddProducer("key", &p)
	mux.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Unexpected error occured\n", w.Body.String())
}

func TestServeHTTP_ProducerPaniced_PanicWrittenToLog(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/key/name", nil)
	w := httptest.NewRecorder()
	p := testProducer{panic: "it-happens"}
	logBuf := bytes.Buffer{}
	log.SetOutput(&logBuf)

	mux := NewServeMux("/")
	mux.AddProducer("key", &p)
	mux.ServeHTTP(w, r)

	assert.Contains(t, logBuf.String(), "[PANIC] it-happens")
}
