package spreadsheet

import (
	"fmt"
	"html/template"
	"io"
)

// Reader stands as a data source for spreadsheet Producer.
type Reader interface {
	// Read reads spreadsheet with a given name. Producer will run Read
	// in a separate goroutine so all callbacks must be done by channels.
	// Read must send result of accessing a resource in confirm channel.
	// Afterwards, spreadsheet contents must be read row by row through the rows
	// channel. The stop channel provides a convenient way to stop read when
	// it is enough for Producer.
	Read(name string, confirm chan<- error, rows chan<- Row, stop <-chan struct{})
}

// Row represents a row in a spreadsheet. Readers must set
// error message if row read is failed.
type Row struct {
	Name         string
	Address      string
	Postcode     string
	Phone        string
	CreditLimit  string
	Birthday     string
	ErrorMessage *string
}

const (
	templateBody = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		<table style="font-family:Courier New, Courier, monospace; white-space:pre">
			<tr style="font-weight: Bold"><td>Name</td><td>Address</td><td>Postcode</td><td>Phone</td><td>Credit Limit</td><td>Birthday</td></tr>
			{{range .Rows}}<tr>{{if not .ErrorMessage}}<td>{{.Name}}</td><td>{{.Address}}</td><td>{{.Postcode}}</td><td>{{.Phone}}</td><td align="right">{{.CreditLimit}}</td><td align="right">{{.Birthday}}</td>{{else}}<td colspan="6">{{.ErrorMessage}}</td>{{end}}</tr>{{end}}
		</table>
	</body>
</html>`
)

// templateData provides data for spreadsheet HTML template.
type templateData struct {
	Title string
	Rows  <-chan Row
}

// Producer provides solutions for spreadsheet output.
type Producer struct {
	reader       Reader
	htmlTemplate *template.Template
}

// NewProducer creates and initializes a new instance of spreadsheet Producer.
func NewProducer(reader Reader) *Producer {
	return &Producer{
		reader:       reader,
		htmlTemplate: template.Must(template.New("spreadsheet").Parse(templateBody)),
	}
}

// HTML generates output to display spreadsheet as a web page.
func (p *Producer) HTML(w io.Writer, name string) error {
	done := make(chan error, 2)
	doneIfPanic := func(helper string) {
		if r := recover(); r != nil {
			done <- fmt.Errorf("%s on %s: %s", helper, name, r)
		}
	}

	stopRead := make(chan struct{})
	confirm := make(chan error)
	rows := make(chan Row)

	go func() {
		defer doneIfPanic(fmt.Sprintf("Reader %T paniced", p.reader))
		defer func() {
			close(rows)
			close(confirm)
		}()

		p.reader.Read(name, confirm, rows, stopRead)
		done <- nil
	}()

	go func() {
		defer doneIfPanic("Template paniced")

		if err, ok := <-confirm; !ok || err != nil {
			done <- err
			return
		}

		defer func() {
			close(stopRead)
			for _ = range rows {
				// allow reader to finish gracefully
			}
		}()
		data := templateData{
			Title: name,
			Rows:  rows,
		}
		done <- p.htmlTemplate.Execute(w, data)
	}()

	return waitForDone(done)
}

// waitForDone drains a given done channel according to its capacity.
// If there is more than one error (which is rare), compound error
// message will be returned.
func waitForDone(done <-chan error) error {
	var result error
	for i := 0; i < cap(done); i++ {
		if err := <-done; err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%s; [add] %s", result, err)
			}
		}
	}
	return result
}
