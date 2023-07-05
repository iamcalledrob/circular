// Package circular - thread-safe, lock-free circular byte buffer
package circular

import (
	"fmt"
	"io"
	"sync/atomic"
)

// Buffer provides a thread-safe, lock-free circular byte buffer for a single reader and
// single writer. Thread safety is not provided for multiple concurrent readers or multiple
// concurrent writers.
//
// Inspired by the design from https://www.snellman.net/blog/archive/2016-12-13-ring-buffers/
//
// Instantiate with a Buf of desired length, e.g: &Buffer{ Buf: make([]byte, 1024) }
type Buffer struct {
	Buf  []byte
	head uint64 // Number of bytes ever written
	tail uint64 // Number of bytes ever read
}

func NewBuffer(n int) *Buffer {
	return &Buffer{Buf: make([]byte, n)}
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read and any error
// encountered.
//
// Returns 0, io.EOF when the buffer is empty.
//
// Only one goroutine should Read at a time—Read is not thread safe (Read and Write can be called
// concurrently however).
func (b *Buffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	tail := atomic.LoadUint64(&b.tail)
	length := atomic.LoadUint64(&b.head) - tail

	if length == 0 {
		return 0, io.EOF
	}

	// Creates a reslice of p that's truncated to the buffer's unread length—so we can copy into
	// and fill dest without further size checks.
	var dest []byte
	if uint64(len(p)) > length {
		dest = p[:length]
	} else {
		dest = p[:]
	}

	bOffset := int(tail % uint64(len(b.Buf)))
	n = copy(dest, b.Buf[bOffset:])
	// Noop (n=0) if all the bytes were copied above
	n += copy(dest[n:], b.Buf[:len(dest)-n])

	atomic.AddUint64(&b.tail, uint64(n))
	return
}

// Write writes up to len(p) bytes from p to the underlying data stream.
//
// Returns n, ErrNoSpace when the buffer did not have enough space to write all of p.
//
// Only one goroutine should Write at a time—Write is not thread safe (Read and Write can be called
// concurrently however).
func (b *Buffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	head := atomic.LoadUint64(&b.head)
	length := head - atomic.LoadUint64(&b.tail)
	space := uint64(len(b.Buf)) - length

	// Creates a reslice of p that's truncated to the buffer's free space—so we can copy all of
	// src into the buffer without further size checks.
	var src []byte
	if uint64(len(p)) > space {
		err = ErrNoSpace
		src = p[:space]
	} else {
		src = p[:]
	}

	bOffset := int(head % uint64(len(b.Buf)))
	n = copy(b.Buf[bOffset:], src)
	// Noop (n=0) if all bytes were copied above
	n += copy(b.Buf[:len(src)-n], src[n:])

	atomic.AddUint64(&b.head, uint64(n))
	return
}

// Len returns the number of bytes of the unread portion of the buffer
//
// Calls to Len are thread-safe, however the value returned may immediately be stale if a Read or
// Write completes concurrently.
func (b *Buffer) Len() int {
	return int(atomic.LoadUint64(&b.head) - atomic.LoadUint64(&b.tail))
}

// Space returns the capacity the buffer has to hold more data.
//
// Calls to Space are thread-safe, however the value returned may immediately be stale if a Read or
// Write completes concurrently.
func (b *Buffer) Space() int {
	return len(b.Buf) - b.Len()
}

// Cap returns the capacity of the underlying buffer (Buf).
func (b *Buffer) Cap() int {
	return len(b.Buf)
}

// Reset clears the buffer by resetting head/tail offsets.
//
// Calls to Reset are not thread-safe, and should not be called concurrently with Read or Write.
func (b *Buffer) Reset() {
	atomic.StoreUint64(&b.head, 0)
	atomic.StoreUint64(&b.tail, 0)
}

// ErrNoSpace is the error returned by Write when bytes written is < len(p) due to limited space
// in the buffer, including when 0 bytes were written.
var ErrNoSpace = fmt.Errorf("no space in buffer")
