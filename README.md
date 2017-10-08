## About

The registry-sample is intended to demonstrate Golang coding skills. It encapsulates some common Golang concepts: http handling, locking, channels, duck-typing, etc. The app runs server that can be used to view some data as HTML. Internally it represents a kind of MVC, though it wasn't a target of architecting. If you have any doubts, do not hesitate to discuss them on an interview.

## Tips

When running the app, use the following URLs to get some valuable output:
* http://127.0.0.1:5000/csv/spread-sheet-a
* http://127.0.0.1:5000/mon/spread-sheet-b

Some aspects of the app can be customized using arguments, see `main.go` for details