package template

import (
	"bytes"
	"sync"
)

// BufferPool defines a Pool of Buffers
type BufferPool struct {
	sync.Pool
}

// NewBufferPool creates a new BufferPool with a custom buffer size
func NewBufferPool(s int) *BufferPool {
	return &BufferPool{
		Pool: sync.Pool{
			New: func() interface{} {
				b := bytes.NewBuffer(make([]byte, 0, s))
				return b
			},
		},
	}
}

// Get returns a Buffer from the pool
func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.Pool.Get().(*bytes.Buffer)
}

// Put resets ans returns a Buffer to the pool
func (bp *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()
	bp.Pool.Put(b)
}
