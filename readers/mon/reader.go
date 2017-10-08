package mon

import (
	"bufio"
	"io"
	"log"
	"registry-sample/producers/spreadsheet"
	"registry-sample/readers/loader"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	columnParseError = "Unable to parse columns"
	rowReadError     = "Invalid row"
)

type column struct {
	name     string
	occupies int
}

// layout defines spreadsheet layout of .mon file as a map
// of columns where keys are where a column is started.
type layout map[int]column

// Reader allows to read formatted monospace delimited .mon files.
type Reader struct {
	ld loader.Interface
}

// NewReader creates and initializes a new .mon spreadsheet reader.
func NewReader(ld loader.Interface) *Reader {
	return &Reader{ld: ld}
}

func (rd Reader) Read(name string, confirm chan<- error, rows chan<- spreadsheet.Row, stop <-chan struct{}) {
	f, err := rd.ld.Load(name + ".mon")
	if err != nil {
		confirm <- err
		return
	}
	defer f.Close()
	confirm <- nil

	r := bufio.NewReader(f)
	layout, err := readLayout(r)
	if err != nil {
		if err != io.EOF {
			// if we can't read layout, we can't read the entire file.
			log.Println("[MON]", err)
			rows <- spreadsheet.Row{ErrorMessage: &columnParseError}
		}
		return
	}

	for {
		select {
		case <-stop:
			return
		default:
			row, err := readRow(r, layout)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Println("[MON]", err)
				rows <- spreadsheet.Row{ErrorMessage: &rowReadError}
				return
			}
			rows <- row
		}
	}
}

func readLayout(r *bufio.Reader) (layout, error) {
	record, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	lt := layout{}

	// Column search is case-sensitive for now.
	// Consider make it insensitive in a future.
	findCol := func(name string) {
		if idx := strings.Index(record, name); idx >= 0 {
			// we must walk runes not bytes because space-separated
			// content is tightly coupled to visual representation.
			start := utf8.RuneCountInString(record[:idx])
			count := utf8.RuneCountInString(name)
			// count space runes to the next word or EOL
			for _, r := range record[idx+len(name):] {
				if unicode.IsSpace(r) && r != '\n' {
					count++
				} else {
					break
				}
			}
			lt[start] = column{
				name:     strings.ToLower(name),
				occupies: count,
			}
		}
	}

	findCol("Name")
	findCol("Address")
	findCol("Postcode")
	findCol("Phone")
	findCol("Credit Limit")
	findCol("Birthday")
	return lt, nil
}

func readRow(r *bufio.Reader, lt layout) (spreadsheet.Row, error) {
	row := spreadsheet.Row{}

	record, err := r.ReadString('\n')
	if err != nil {
		return row, err
	}

	runeNum := 0
	waitRuneNum := -1
	colIdx := 0
	colName := ""

	for i := range record {
		if runeNum > waitRuneNum {
			// look for a column started at the current rune
			if col, ok := lt[runeNum]; ok {
				colIdx = i
				colName = col.name
				waitRuneNum = runeNum + col.occupies - 1
			}
		} else if runeNum == waitRuneNum {
			// we've reached the rune where the current col ends
			v := strings.TrimSpace(record[colIdx : i+1])
			switch colName {
			case "name":
				row.Name = v
			case "address":
				row.Address = v
			case "postcode":
				row.Postcode = v
			case "phone":
				row.Phone = v
			case "credit limit":
				row.CreditLimit = v
			case "birthday":
				if t, err := time.Parse("20060102", v); err == nil {
					row.Birthday = t.Format("2006-01-02")
				} else {
					row.Birthday = v
				}
			}
		}
		runeNum++
	}
	return row, nil
}
