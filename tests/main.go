package main

import (
	"git.ailur.dev/ailur/jsStreams"

	"fmt"
	"io"

	"syscall/js"
)

// NOTE: Please do not use this code as an example. It never closes the stream and will leak memory.
// It is intended for use in the developer console, where you can close the stream via JavaScript.

func main() {
	js.Global().Set("TryReadStream", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			readStream := jsStreams.NewReadableStream(args[0])
			var buffer []byte
			buffer, err := io.ReadAll(readStream)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(buffer))
		}()

		return nil
	}))

	js.Global().Set("TryWriteStream", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			writeStream := jsStreams.NewWritableStream(args[0])
			_, err := writeStream.Write([]byte(args[1].String()))
			if err != nil {
				fmt.Println(err)
				return
			}
		}()

		return nil
	}))

	select {}
}
