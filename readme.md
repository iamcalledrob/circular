# circular - Circular Buffer
circular provides a thread-safe, lock-free circular byte buffer for a single reader and single
writer. Thread safety is not provided for multiple concurrent readers or multiple concurrent
writers.

Inspired by the design from https://www.snellman.net/blog/archive/2016-12-13-ring-buffers/

## Why a circular buffer instead of bytes.Buffer?
`bytes.Buffer` wraps a slice. The memory allocated to that slice can be grown, but never shrunk.
This can lead to the buffer using much more memory than expected. Depending on the sizes of reads
and writes to the buffer it's possible to end up with an unexpectedly large slice containing
very little data.

By comparison, a circular buffer can sustain an unlimited amount of reads and writes without ever
needing to grow or move data around, by re-using the same memory again and again. The trade-off
being that the buffer is fixed-size. This can be very useful in scenarios where you want to
constrain memory usage and need to move a lot of data through a pipeline.

## Goals
- Be easy to follow, understand and test.
- Take no external dependencies -- similar packages exist, but are not self-contained.
- Act as a drop-in (fixed-size) replacement for most `bytes.Buffer` use-cases.
- Be lock-free, so no possibility of deadlocks for the caller.

## Usage
[Godoc](http://pkg.go.dev/github.com/iamcalledrob/circular)

```go
// Allocate a new buffer
buf := circular.NewBuffer(64*1024)

// Re-use an existing buffer
existing := make([]byte, 64*1024)
buf := &circular.Buffer{Buf: existing}

// Standard io.Reader/io.Writer behaviour.
n, err = buf.Read(p)
n, err = buf.Write(p)
```

## Notes
-   `Write(p []byte)` returns `ErrNoSpace` when the buffer does not have enough space to write
    all of p. Partial writes are supported. Data is not overwritten.
