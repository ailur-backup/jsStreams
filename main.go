package jsStreams

import (
	"errors"
	"fmt"
	"sync"

	"syscall/js"
)

// ReadableStream implements io.ReaderCloser for a JavaScript ReadableStream.
type ReadableStream struct {
	stream js.Value
	reader js.Value
	lock   sync.Mutex
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
// This implementation of Read does not use scratch space if n < len(p). If some data is available but not len(p) bytes,
// Read conventionally returns what is available instead of waiting for more. Note: Read will block until data is available,
// meaning in a WASM environment, you **must** use a goroutine to call Read.
func (r *ReadableStream) Read(inputBytes []byte) (n int, err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()

	r.lock.Lock()
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	if r.reader.IsUndefined() {
		r.reader = r.stream.Call("getReader", map[string]interface{}{"mode": "byob"})
	}

	resultBuffer := js.Global().Get("Uint8Array").New(len(inputBytes))
	readResult := r.reader.Call("read", resultBuffer)

	readResult.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer waitGroup.Done()
		data := args[0].Get("value")
		js.CopyBytesToGo(inputBytes, data)
		if args[0].Get("done").Bool() {
			err = errors.New("EOF")
		}
		n = data.Length()
		return nil
	}))

	readResult.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer waitGroup.Done()
		err = errors.New(args[0].Get("message").String())
		return nil
	}))

	waitGroup.Wait()

	return n, err
}

// Close closes the ReadableStream. If the stream is already closed, Close does nothing.
// If the stream is not yet closed, it is canceled. The reader is closed and the underlying source or pipeline is terminated.
// This method is idempotent, meaning that it can be called multiple times without causing an error.
func (r *ReadableStream) Close() (err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()

	if !r.reader.IsUndefined() {
		r.reader.Call("cancel")
	}

	return nil
}

func NewReadableStream(stream js.Value) *ReadableStream {
	return &ReadableStream{stream: stream}
}
