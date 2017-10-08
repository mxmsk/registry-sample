package csv

import (
	"errors"
	"registry-sample/producers/spreadsheet"
	"registry-sample/readers/loader"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReaderRead_LoadError_ExpectErrorOnConfirmed(t *testing.T) {
	ld := loader.NewTestLoadError(errors.New("file is somewhere, but not here"))
	confirm := make(chan error, 2)

	r := NewReader(ld)
	r.Read("name1", confirm, nil, nil)

	err := <-confirm

	assert.EqualError(t, err, "file is somewhere, but not here")
}

func TestReaderRead_LoadOk_ExpectNilOnConfirmed(t *testing.T) {
	ld := loader.NewTest("col")
	confirm := make(chan error, 2)

	r := NewReader(ld)
	r.Read("name1", confirm, nil, nil)

	err := <-confirm

	assert.NoError(t, err)
}

func TestReaderRead_LoaderLoad_ExpectCorrectArgs(t *testing.T) {
	ld := loader.NewTestLoadError(errors.New("doesn't matter"))
	confirm := make(chan error, 2)

	r := NewReader(ld)
	r.Read("name1", confirm, nil, nil)

	<-confirm

	assert.Equal(t, "name1.csv", ld.LoadName)
}

func TestReaderRead_LoadOk_ExpectContentOnRows(t *testing.T) {
	ld := loader.NewTest(
		"Name,Address,Postcode,Phone,Credit Limit,Birthday\n" +
			"\"Stewart, Jamie\",Voorstraat 47,3123gg,020 7899381,50000,01/02/1982\n" +
			"\"Leon, Mike\",Dorpsplein 5A,4532 AA,030 2288986,201092,03/11/1967\n")
	confirm := make(chan error, 2)
	rows := make(chan spreadsheet.Row)
	expected := []spreadsheet.Row{
		{
			Name:        "Stewart, Jamie",
			Address:     "Voorstraat 47",
			Postcode:    "3123gg",
			Phone:       "020 7899381",
			CreditLimit: "50000",
			Birthday:    "1982-02-01",
		}, {
			Name:        "Leon, Mike",
			Address:     "Dorpsplein 5A",
			Postcode:    "4532 AA",
			Phone:       "030 2288986",
			CreditLimit: "201092",
			Birthday:    "1967-11-03",
		},
	}

	r := NewReader(ld)
	go func() {
		defer close(rows)
		r.Read("name1", confirm, rows, nil)
	}()

	var received []spreadsheet.Row
	for row := range rows {
		received = append(received, row)
	}

	assert.Len(t, received, 2)
	assert.Contains(t, received, expected[0])
	assert.Contains(t, received, expected[1])
}

func TestReaderRead_LoadOk_ExpectOnlyKnownColumns(t *testing.T) {
	ld := loader.NewTest(
		"Name,Address111,Postcode,Phone222,Credit Limit,Birthday\n" +
			"\"Stewart, Jamie\",Voorstraat 47,3123gg,020 7899381,50000,01/02/1982\n")
	confirm := make(chan error, 2)
	rows := make(chan spreadsheet.Row)
	expected := spreadsheet.Row{
		Name:        "Stewart, Jamie",
		Postcode:    "3123gg",
		CreditLimit: "50000",
		Birthday:    "1982-02-01",
	}

	r := NewReader(ld)
	go func() {
		defer close(rows)
		r.Read("name1", confirm, rows, nil)
	}()

	var received []spreadsheet.Row
	for row := range rows {
		received = append(received, row)
	}

	assert.Len(t, received, 1)
	assert.Contains(t, received, expected)
}

func TestReaderRead_ReadError_ExpectErrorOnRows(t *testing.T) {
	ld := loader.NewTestReadError(errors.New("wrong content"))
	confirm := make(chan error, 2)
	rows := make(chan spreadsheet.Row)

	r := NewReader(ld)
	go func() {
		defer close(rows)
		r.Read("name1", confirm, rows, nil)
	}()

	var received []spreadsheet.Row
	for row := range rows {
		received = append(received, row)
	}

	assert.Len(t, received, 1)
	assert.NotNil(t, received[0].ErrorMessage)
}

func TestReaderRead_WrongFieldCount_ExpectAvailableColsOnRows(t *testing.T) {
	ld := loader.NewTest(
		"Name,Address,Postcode,Phone,Credit Limit,Birthday\n" +
			"\"Stewart, Jamie\",Voorstraat 47,3123gg,020 7899381,50000,01/02/1982,Dorpsplein 5B\n" +
			"\"Leon, Mike\",Dorpsplein 5A,4532 AA,030 2288986,03/11/1967\n" +
			"\n")
	confirm := make(chan error, 2)
	rows := make(chan spreadsheet.Row)
	expected := []spreadsheet.Row{
		{
			Name:        "Stewart, Jamie",
			Address:     "Voorstraat 47",
			Postcode:    "3123gg",
			Phone:       "020 7899381",
			CreditLimit: "50000",
			Birthday:    "1982-02-01",
		}, {
			Name:        "Leon, Mike",
			Address:     "Dorpsplein 5A",
			Postcode:    "4532 AA",
			Phone:       "030 2288986",
			CreditLimit: "03/11/1967",
		},
	}

	r := NewReader(ld)
	go func() {
		defer close(rows)
		r.Read("name1", confirm, rows, nil)
	}()

	var received []spreadsheet.Row
	for row := range rows {
		received = append(received, row)
	}

	assert.Len(t, received, 2)
	assert.Contains(t, received, expected[0])
	assert.Contains(t, received, expected[1])
}

func TestReaderRead_LoadOk_ExpectReaderClosed(t *testing.T) {
	ld := loader.NewTest(
		"Name,Address,Postcode,Phone,Credit Limit,Birthday\n" +
			"\"Stewart, Jamie\",Voorstraat 47,3123gg,020 7899381,50000,01/02/1982\n" +
			"\"Leon, Mike\",Dorpsplein 5A,4532 AA,030 2288986,201092,03/11/1967\n")
	confirm := make(chan error, 2)
	rows := make(chan spreadsheet.Row)

	r := NewReader(ld)
	go func() {
		defer close(rows)
		r.Read("name1", confirm, rows, nil)
	}()

	for _ = range rows {
	}

	assert.True(t, ld.ReaderClosed)
}
