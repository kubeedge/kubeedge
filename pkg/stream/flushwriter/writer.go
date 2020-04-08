package flushwriter

import (
	"io"
	"net/http"
)

type FlushWriter struct {
	flusher http.Flusher
	writer  io.Writer
}

func (f FlushWriter) Write(p []byte) (n int, err error) {
	n, err = f.writer.Write(p)
	if err != nil {
		return
	}
	if f.flusher != nil {
		f.flusher.Flush()
	}
	return
}

func Wrap(w io.Writer) io.Writer {
	writer := &FlushWriter{
		writer: w,
	}
	if flusher, ok := w.(http.Flusher); ok {
		writer.flusher = flusher
	}
	return writer
}
