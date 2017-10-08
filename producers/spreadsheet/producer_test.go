package spreadsheet

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testReader struct {
	readName string
	rows     []Row
	err      error
	panic    interface{}
}

func (r *testReader) Read(name string, confirm chan<- error, rows chan<- Row, stop <-chan struct{}) {
	r.readName = name
	if r.panic != nil {
		panic(r.panic)
	}
	confirm <- r.err
	if len(r.rows) != 0 {
		for _, row := range r.rows {
			rows <- row
		}
	}
}

func TestHtml_EmptyRead_NoError(t *testing.T) {
	p := NewProducer(&testReader{})
	err := p.HTML(&bytes.Buffer{}, "name")
	assert.NoError(t, err)
}

func TestHtml_ReadError_ErrorReturned(t *testing.T) {
	r := testReader{err: errors.New("must read, but won't")}
	p := NewProducer(&r)
	err := p.HTML(&bytes.Buffer{}, "name")
	assert.EqualError(t, err, "must read, but won't")
}

func TestHtml_ReadError_NothingWritten(t *testing.T) {
	r := testReader{err: errors.New("must read, but won't")}
	p := NewProducer(&r)
	b := bytes.Buffer{}
	p.HTML(&b, "name")
	assert.Len(t, b.Bytes(), 0)
}

func TestHtml_ReadPanic_ErrorReturned(t *testing.T) {
	r := testReader{panic: "something went wrong"}
	p := NewProducer(&r)
	err := p.HTML(&bytes.Buffer{}, "name1")
	assert.EqualError(t, err, "Reader *spreadsheet.testReader paniced on name1: something went wrong")
}

func TestHtml_ReadPanic_NothingWritten(t *testing.T) {
	r := testReader{panic: "something went wrong"}
	p := NewProducer(&r)
	b := bytes.Buffer{}
	p.HTML(&b, "name1")
	assert.Len(t, b.Bytes(), 0)
}

func TestHtml_SuccessfulRead_CorrectHtml(t *testing.T) {
	r := testReader{
		rows: []Row{
			{
				Name:        "name1",
				Address:     "addr1",
				Postcode:    "postcode1",
				Phone:       "phone1",
				CreditLimit: "1.45",
				Birthday:    "1991-01-02",
			}, {
				Name:        "name2",
				Address:     "addr2",
				Postcode:    "postcode2",
				Phone:       "phone2",
				CreditLimit: "2.31",
				Birthday:    "1992-06-05",
			},
		},
	}
	var buf bytes.Buffer

	p := NewProducer(&r)
	err := p.HTML(&buf, "success")
	assert.NoError(t, err)

	s := buf.String()
	assert.Contains(t, s, `<title>success</title>`)
	assert.Contains(t, s, `<td>name1</td><td>addr1</td><td>postcode1</td><td>phone1</td><td align="right">1.45</td><td align="right">1991-01-02</td>`)
	assert.Contains(t, s, `<td>name2</td><td>addr2</td><td>postcode2</td><td>phone2</td><td align="right">2.31</td><td align="right">1992-06-05</td>`)
}

func TestHtml_ErrorInSomeRows_CorrectHtml(t *testing.T) {
	errMsg := "oops sorry"
	r := testReader{
		rows: []Row{
			{
				Name:         "name1",
				Address:      "addr1",
				Postcode:     "postcode1",
				Phone:        "phone1",
				CreditLimit:  "1.45",
				Birthday:     "1991-01-02",
				ErrorMessage: &errMsg,
			}, {
				Name:        "name2",
				Address:     "addr2",
				Postcode:    "postcode2",
				Phone:       "phone2",
				CreditLimit: "2.31",
				Birthday:    "1992-06-05",
			},
		},
	}
	var buf bytes.Buffer

	p := NewProducer(&r)
	err := p.HTML(&buf, "success")
	assert.NoError(t, err)

	s := buf.String()
	assert.Contains(t, s, `<title>success</title>`)
	assert.Contains(t, s, `<td colspan="6">oops sorry</td>`)
	assert.Contains(t, s, `<td>name2</td><td>addr2</td><td>postcode2</td><td>phone2</td><td align="right">2.31</td><td align="right">1992-06-05</td>`)
}

func TestWaitForDone_DoneWithoutErrors_NoError(t *testing.T) {
	done := make(chan error, 2)
	done <- nil
	done <- nil
	err := waitForDone(done)
	assert.NoError(t, err)
}

func TestWaitForDone_DoneWithError_ErrorReturned(t *testing.T) {
	testCases := []struct {
		errs []error
		want string
	}{
		{
			errs: []error{errors.New("err1"), nil},
			want: "err1",
		}, {
			errs: []error{nil, errors.New("err1")},
			want: "err1",
		}, {
			errs: []error{errors.New("err1"), errors.New("err2"), errors.New("err3")},
			want: "err1; [add] err2; [add] err3",
		},
	}

	for _, testCase := range testCases {
		done := make(chan error, len(testCase.errs))
		for _, err := range testCase.errs {
			done <- err
		}
		err := waitForDone(done)
		assert.EqualError(t, err, testCase.want)
	}
}
