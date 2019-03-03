// Package pool provides a buffer pool.
package pool

import (
	"bytes"
	"sync"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

// AllocBuffer allocates a buffer, or re-uses an existing buffer.
func AllocBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// ReleaseBuffer returns a buffer to the pool to be re-used.
func ReleaseBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufferPool.Put(buf)
	}
}
