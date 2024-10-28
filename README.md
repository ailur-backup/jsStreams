# jsStreams

Go library to communicate with the JS Stream API by bridging the JS ReadableStream and WritableStream objects to a Go io.ReaderCloser and io.WriterCloser.
It also works vice versa, and with pipe readers/writers.

[![Go Report Card](https://goreportcard.com/badge/git.ailur.dev/ailur/jsStreams)](https://goreportcard.com/report/git.ailur.dev/ailur/jsStreams) [![Go Reference](https://pkg.go.dev/badge/git.ailur.dev/ailur/jsStreams.svg)](https://pkg.go.dev/git.ailur.dev/ailur/jsStreams)

The API is pretty self-explanatory, see the Go Reference badge above for the full documentation.