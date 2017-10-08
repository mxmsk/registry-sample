package loader

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// Interface of loader abstracts persistent storage for readers.
type Interface interface {
	// Load returns the object that can read from storage.
	// If storage is inaccessible, the error is returned.
	Load(name string) (io.ReadCloser, error)
}

// fsLoader implements loader abstraction over file system.
type fsLoader struct {
	dataDir string
}

// NewFS creates loader that uses file system as a storage.
func NewFS(dataDir string) Interface {
	return &fsLoader{dataDir: dataDir}
}

func (ld fsLoader) Load(name string) (io.ReadCloser, error) {
	dataDir := filepath.Clean(ld.dataDir)
	fileName := filepath.Join(dataDir, name)

	// Make sure that the file is within data directory.
	if filepath.Dir(fileName) != dataDir {
		return nil, os.ErrNotExist
	}
	return os.Open(fileName)
}

// Test provides a way to test usage of loader.
type Test struct {
	buf   *bytes.Buffer
	rdErr error
	ldErr error

	LoadName     string
	ReaderClosed bool
}

// NewTest creates stub for testing with loader.
func NewTest(content string) *Test {
	return &Test{buf: bytes.NewBufferString(content)}
}

// NewTestLoadError creates stub for testing with loader that returns error on load.
func NewTestLoadError(err error) *Test {
	return &Test{ldErr: err}
}

// NewTestReadError creates stub for testing with loader which reader returns error.
func NewTestReadError(err error) *Test {
	return &Test{rdErr: err}
}

func (ld *Test) Load(name string) (io.ReadCloser, error) {
	ld.LoadName = name
	if ld.ldErr != nil {
		return nil, ld.ldErr
	}
	return testReader{ld: ld}, nil
}

type testReader struct {
	ld *Test
}

func (r testReader) Read(p []byte) (n int, err error) {
	if r.ld.rdErr != nil {
		return 0, r.ld.rdErr
	}
	return r.ld.buf.Read(p)
}

func (r testReader) Close() error {
	r.ld.ReaderClosed = true
	return nil
}
