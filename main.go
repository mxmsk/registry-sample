package main

import (
	"flag"
	"net/http"
	"registry-sample/producers"
	"registry-sample/producers/spreadsheet"
	"registry-sample/readers/csv"
	"registry-sample/readers/loader"
	"registry-sample/readers/mon"
)

func main() {
	dataDir := flag.String("datadir", "./data", "Directory where data files are stored")
	port := flag.String("port", "5000", "Port to listen requests on")
	flag.Parse()

	mux := producers.NewServeMux("/")
	ld := loader.NewFS(*dataDir)
	mux.AddProducer("csv", spreadsheet.NewProducer(csv.NewReader(ld)))
	mux.AddProducer("mon", spreadsheet.NewProducer(mon.NewReader(ld)))

	http.ListenAndServe(":"+*port, mux)
}
