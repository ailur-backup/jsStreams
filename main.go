package jsStreams

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"syscall/js"
)

// ReadableStream implements io.ReadCloser for a JavaScript ReadableStream.
type ReadableStream struct {
	stream js.Value
	lock   sync.Mutex
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
// This implementation of Read does not use scratch space if n < len(p). If some data is available but not len(p) bytes,
// Read conventionally returns what is available instead of waiting for more. Note: Read will block until data is available,
// meaning in a WASM environment, you must use a goroutine to call Read.
func (r *ReadableStream) Read(p []byte) (n int, err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()

	r.lock.Lock()
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	reader := r.stream.Call("getReader", map[string]interface{}{"mode": "byob"})

	resultBuffer := js.Global().Get("Uint8Array").New(len(p))
	readResult := reader.Call("read", resultBuffer)

	readResult.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer waitGroup.Done()
		if args[0].Get("done").Bool() || args[0].Get("value").Length() == 0 {
			err = io.EOF
			return nil
		}
		data := args[0].Get("value")
		js.CopyBytesToGo(p, data)
		n = data.Length()
		return nil
	}))

	readResult.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer waitGroup.Done()
		err = errors.New(args[0].Get("message").String())
		return nil
	}))

	waitGroup.Wait()
	reader.Call("releaseLock")
	r.lock.Unlock()

	return n, err
}

// Close closes the ReadableStream. If the stream is already closed, Close does nothing.
func (r *ReadableStream) Close() (err error) {
	defer func() {
		// We don't want any errors to be thrown if the stream is already closed.
		recovery := recover()
		if !strings.Contains(recovery.(string), "Can not close stream after closing or error") {
			err = fmt.Errorf("panic: %v", recovery)
		}
	}()

	r.lock.Lock()
	r.stream.Call("cancel")
	r.lock.Unlock()
	return nil
}

// NewReadableStream creates a new ReadableStream from a JavaScript ReadableStream.
func NewReadableStream(stream js.Value) *ReadableStream {
	return &ReadableStream{stream: stream}
}

// WritableStream implements io.WriteCloser for a JavaScript WritableStream.
type WritableStream struct {
	stream js.Value
	lock   sync.Mutex
}

// Write writes len(p) bytes from p to the underlying data stream. It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early. Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
func (w *WritableStream) Write(p []byte) (n int, err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()

	w.lock.Lock()
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	writer := w.stream.Call("getWriter")

	writer.Get("ready").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer waitGroup.Done()

		buffer := js.Global().Get("Uint8Array").New(len(p))
		js.CopyBytesToJS(buffer, p)

		writeResult := writer.Call("write", buffer)

		writeResult.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer waitGroup.Done()
			n = len(p)
			return nil
		}))

		writeResult.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer waitGroup.Done()
			err = errors.New(args[0].Get("message").String())
			return nil
		}))

		return nil
	}))

	waitGroup.Wait()
	writer.Call("releaseLock")
	w.lock.Unlock()

	return n, err
}

// Close closes the WritableStream. If the stream is already closed, Close does nothing.
func (w *WritableStream) Close() (err error) {
	defer func() {
		// We don't want any errors to be thrown if the stream is already closed.
		recovery := recover()
		if !strings.Contains(recovery.(string), "Can not close stream after closing or error") {
			err = fmt.Errorf("panic: %v", recovery)
		}
	}()

	w.lock.Lock()
	w.stream.Call("close")
	w.lock.Unlock()

	return nil
}

// NewWritableStream creates a new WritableStream. If a JavaScript WritableStream is provided, it will be used.
// Otherwise, a new WritableStream will be created.
func NewWritableStream(stream ...js.Value) *WritableStream {
	if len(stream) > 0 {
		return &WritableStream{stream: stream[0]}
	} else {
		stream := js.Global().Get("WritableStream").New()
		return &WritableStream{stream: stream}
	}
}

// Now we do the vice versa: Reader to ReadableStream and Writer to WritableStream.

// ReaderToReadableStream converts an io.Reader to a JavaScript ReadableStream.
func ReaderToReadableStream(r io.Reader) js.Value {
	return js.Global().Get("ReadableStream").New(map[string]interface{}{
		"pull": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			readController := args[0]
			return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				var buffer []byte
				buffer, err := io.ReadAll(r)
				if err != nil {
					panic(err.Error())
				}
				if len(buffer) == 0 {
					readController.Call("close")
					return nil
				}
				jsBuffer := js.Global().Get("Uint8Array").New(len(buffer))
				js.CopyBytesToJS(jsBuffer, buffer)
				readController.Call("enqueue", jsBuffer)
				readController.Call("close")
				args[0].Invoke()
				return nil
			}))
		}),
		"type": "bytes",
	})
}

// WriterToWritableStream converts an io.Writer to a JavaScript WritableStream.
func WriterToWritableStream(w io.Writer) js.Value {
	return js.Global().Get("WritableStream").New(map[string]interface{}{
		"write": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			writeBuffer := args[0]
			return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				buffer := make([]byte, writeBuffer.Length())
				js.CopyBytesToGo(buffer, writeBuffer)
				_, err := w.Write(buffer)
				if err != nil {
					panic(err.Error())
				}
				args[0].Invoke()
				return nil
			}))
		}),
	})
}
