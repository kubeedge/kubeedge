package mercury

import (
	"bufio"
	"io"
	"sync"
	"time"
)

// Writer extends a buffered writer that flushes itself asynchronously. It uses
// a timer to flush the buffered writer it it gets stale. Errors that occur
// during the flush are returned on the next call to Write, Flush or WriteAndFlush.
type Writer struct {
	w *bufio.Writer
	d time.Duration
	t *time.Timer
	e error
	m sync.Mutex
}

// NewWriter wraps the provided writer and enable buffering and asynchronous
// flushing using the specified maximum delay.
func NewWriter(w io.Writer, maxDelay time.Duration) *Writer {
	return &Writer{
		w: bufio.NewWriter(w),
		d: maxDelay,
	}
}

// NewWriterSize wraps the provided writer and enable buffering and asynchronous
// flushing using the specified maximum delay. This method allows configuration
// of the initial buffer size.
func NewWriterSize(w io.Writer, maxDelay time.Duration, size int) *Writer {
	return &Writer{
		w: bufio.NewWriterSize(w, size),
		d: maxDelay,
	}
}

// Write implements the io.Writer interface and writes data to the underlying
// buffered writer and flushes it asynchronously.
func (w *Writer) Write(p []byte) (int, error) {
	return w.write(p, false)
}

// Flush flushes the buffered writer immediately.
func (w *Writer) Flush() error {
	_, err := w.write(nil, true)
	return err
}

// WriteAndFlush writes data to the underlying buffered writer and flushes it
// immediately after writing.
func (w *Writer) WriteAndFlush(p []byte) (int, error) {
	return w.write(p, true)
}

func (w *Writer) write(p []byte, flush bool) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()

	// clear and return any error from flush
	if w.e != nil {
		err = w.e
		w.e = nil
		return 0, err
	}

	// write data if available
	if len(p) > 0 {
		n, err = w.w.Write(p)
		if err != nil {
			return n, err
		}
	}

	// flush immediately if requested
	if flush {
		err = w.w.Flush()
		if err != nil {
			return n, err
		}
	}

	// setup timer if data is buffered
	if w.w.Buffered() > 0 && w.t == nil {
		w.t = time.AfterFunc(w.d, w.flush)
	}

	// stop timer if no data is buffered
	if w.w.Buffered() == 0 && w.t != nil {
		w.t.Stop()
		w.t = nil
	}

	return n, nil
}

func (w *Writer) flush() {
	w.m.Lock()
	defer w.m.Unlock()

	// clear timer
	w.t = nil

	// flush buffer
	err := w.w.Flush()
	if err != nil && w.e == nil {
		w.e = err
	}
}
