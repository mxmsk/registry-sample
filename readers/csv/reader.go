package csv

import (
	"io"
	"log"
	"registry-sample/producers/spreadsheet"
	"registry-sample/readers/loader"
	"strings"
	"time"

	csv_enc "encoding/csv"
)

var (
	columnParseError = "Unable to parse columns"
	rowReadError     = "Invalid row"
)

// layout defines column indices for a CSV file.
type layout struct {
	name        int
	address     int
	postcode    int
	phone       int
	creditLimit int
	birthday    int
}

// Reader allows to read comma-separated .csv files.
type Reader struct {
	ld loader.Interface
}

// NewReader creates and initializes a new .csv spreadsheet reader.
func NewReader(ld loader.Interface) *Reader {
	return &Reader{ld: ld}
}

func (rd Reader) Read(name string, confirm chan<- error, rows chan<- spreadsheet.Row, stop <-chan struct{}) {
	f, err := rd.ld.Load(name + ".csv")
	if err != nil {
		confirm <- err
		return
	}
	defer f.Close()
	confirm <- nil

	r := csv_enc.NewReader(f)
	lt, err := readLayout(r)
	if err != nil {
		if err != io.EOF {
			// if we can't read layout, we can't read the entire file.
			log.Println("[CSV]", err)
			rows <- spreadsheet.Row{ErrorMessage: &columnParseError}
		}
		return
	}

	for {
		select {
		case <-stop:
			return
		default:
			row, err := readRow(r, lt)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Println("[CSV]", err)
				rows <- spreadsheet.Row{ErrorMessage: &rowReadError}
				return
			}
			rows <- row
		}
	}
}

func readLayout(r *csv_enc.Reader) (layout, error) {
	lt := layout{
		name:        -1,
		address:     -1,
		postcode:    -1,
		phone:       -1,
		creditLimit: -1,
		birthday:    -1,
	}

	record, err := r.Read()
	if err != nil {
		return lt, err
	}

	for i, column := range record {
		key := strings.ToLower(column)
		switch key {
		case "name":
			lt.name = i
		case "address":
			lt.address = i
		case "postcode":
			lt.postcode = i
		case "phone":
			lt.phone = i
		case "credit limit":
			lt.creditLimit = i
		case "birthday":
			lt.birthday = i
		}
	}
	return lt, nil
}

func readRow(r *csv_enc.Reader, lt layout) (spreadsheet.Row, error) {
	row := spreadsheet.Row{}

	record, err := r.Read()
	if err != nil && !isCsvParseError(err) {
		return row, err
	}

	if lt.name >= 0 && lt.name < len(record) {
		row.Name = record[lt.name]
	}
	if lt.address >= 0 && lt.address < len(record) {
		row.Address = record[lt.address]
	}
	if lt.postcode >= 0 && lt.postcode < len(record) {
		row.Postcode = record[lt.postcode]
	}
	if lt.phone >= 0 && lt.phone < len(record) {
		row.Phone = record[lt.phone]
	}
	if lt.creditLimit >= 0 && lt.creditLimit < len(record) {
		row.CreditLimit = record[lt.creditLimit]
	}
	if lt.birthday >= 0 && lt.birthday < len(record) {
		if t, err := time.Parse("02/01/2006", record[lt.birthday]); err == nil {
			row.Birthday = t.Format("2006-01-02")
		} else {
			row.Birthday = record[lt.birthday]
		}
	}
	return row, nil
}

func isCsvParseError(err error) bool {
	if parseErr, ok := err.(*csv_enc.ParseError); ok {
		return parseErr.Err == csv_enc.ErrFieldCount
	}
	return false
}
