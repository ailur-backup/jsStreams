package main

import (
	"fmt"
	"git.ailur.dev/ailur/jsStreams"
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

	js.Global().Set("TryWriterConversions", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			reader, writer := io.Pipe()
			go func() {
				writeStream := jsStreams.WriterToWritableStream(writer)
				buffer := js.Global().Get("Uint8Array").New(45)
				js.CopyBytesToJS(buffer, []byte("Hi, I've been piped through a WritableStream!"))
				writeStream.Call("getWriter").Call("write", buffer).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					go func() {
						err := writer.Close()
						if err != nil {
							fmt.Println(err)
						}
					}()
					return nil
				}))
			}()
			go func() {
				fmt.Println("Reading stream...")
				m, _ := io.ReadAll(reader)
				fmt.Println(string(m))
			}()
		}()

		return nil
	}))

	js.Global().Set("TryReaderConversions", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			reader, writer := io.Pipe()
			go func() {
				_, err := writer.Write([]byte("Hi, I've been piped through a ReadableStream!"))
				if err != nil {
					fmt.Println(err)
					return
				}
				err = writer.Close()
				if err != nil {
					fmt.Println(err)
					return
				}
			}()
			go func() {
				fmt.Println("Reading stream...")
				m, _ := io.ReadAll(jsStreams.NewReadableStream(jsStreams.ReaderToReadableStream(reader)))
				fmt.Println(string(m))
			}()
		}()

		return nil
	}))

	select {}
}
