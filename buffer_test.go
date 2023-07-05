package circular

import (
	"bytes"
	cryptorand "crypto/rand"
	"io"
	"testing"
)

// Tests for "buffer full or empty" problem with circular buffers
func TestCircularBuffer_LenAndSpace(t *testing.T) {
	b := &Buffer{Buf: make([]byte, 10)}
	l := b.Len()
	s := b.Space()
	if l != 0 {
		t.Errorf("invalid length of default buffer (expected 0, got %d)", b.Len())
	}
	if s != 10 {
		t.Errorf("invalid space of default buffer (expected 0, got %d)", s)
	}

	_, _ = b.Write(make([]byte, 10))
	l = b.Len()
	s = b.Space()
	if l != 10 {
		t.Errorf("invalid length of full buffer (expected 10, got %d)", l)
	}
	if s != 0 {
		t.Errorf("invalid space of full buffer (expected 0, got %d)", s)
	}
}

func TestCircularBuffer_Write(t *testing.T) {
	b := &Buffer{Buf: make([]byte, 10)}

	n, err := b.Write([]byte("abcdefghij"))
	if err != nil {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 10 {
		t.Errorf("incorrect n (got: %d, expected: 10)", n)
	}

	n, err = b.Write([]byte("k"))
	if n != 0 {
		t.Errorf("wrote bytes to full buffer (wrote %d)", n)
	}
	if err != ErrNoSpace {
		t.Errorf("did not return ErrNoSpace when writing to full buffer")
	}

	_, _ = b.Read(make([]byte, 4))
	n, err = b.Write([]byte("KLM"))
	if err != nil {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 3 {
		t.Errorf("incorrect n (got: %d, expected: 3)", n)
	}

	// TODO: Test writing > buffer size bytes
}

func TestCircularBuffer_Read(t *testing.T) {
	b := &Buffer{Buf: make([]byte, 10)}
	p := []byte("abcdefghij")
	_, _ = b.Write(p)

	out, err := io.ReadAll(b)
	if err != nil {
		t.Errorf("failed to ReadAll (err: %v)", err)
	}
	if !bytes.Equal(p, out) {
		t.Errorf("read incorrect bytes (read: %s, expected: %s)", out, p)
	}

	n, err := b.Read(make([]byte, 1))
	if n != 0 {
		t.Errorf("read bytes from empty buffer (read %d)", n)
	}
	if err != io.EOF {
		t.Errorf("did not return io.EOF when reading from empty buffer")
	}

	_, _ = b.Write([]byte("a"))

	n, err = b.Read(make([]byte, 1))
	if n != 1 || err != nil {
		t.Errorf("failed to read from previously empty buffer (wrote %d, err: %v)", n, err)
	}

}

// Fill buffer then read, so head == 0 and tail > head
func TestCircularBuffer_WraparoundToZero(t *testing.T) {
	// abcdefghij
	// ^h+t
	// _______hij
	// ^h     ^t
	// KLMNOPQhij
	//        ^ht
	// __________
	//        ^ht

	// abcdefghij
	b := &Buffer{Buf: make([]byte, 10)}
	p := []byte("abcdefghij")
	n, err := b.Write(p)
	s := b.Space()
	l := b.Len()
	if err != nil {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 10 {
		t.Errorf("incorrect n (got: %d, expected: 10)", n)
	}
	if s != 0 {
		t.Errorf("incorrect space (got: %d, expected: 0)", s)
	}
	if l != 10 {
		t.Errorf("incorrect len (got: %d, expected: 10) ", l)
	}

	// _______hij
	o := make([]byte, 7)
	n, err = b.Read(o)
	s = b.Space()
	l = b.Len()
	if err != nil {
		t.Errorf("failed to read from buffer (err: %v)", err)
	}
	if n != 7 {
		t.Errorf("incorrect n (got: %d, expected: 7)", n)
	}
	if !bytes.Equal(o[:n], []byte("abcdefg")) {
		t.Errorf("read incorrect bytes (read: %s, expected: abcdefg)", o)
	}
	if s != 7 {
		t.Errorf("incorrect space (got: %d, expected: 7)", s)
	}
	if l != 3 {
		t.Errorf("incorrect len (got: %d, expected: 3) ", l)
	}

	// KLMNOPQhij
	p = []byte("KLMNOPQ")
	n, err = b.Write(p)
	s = b.Space()
	l = b.Len()

	if err != nil {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 7 {
		t.Errorf("incorrect n (got: %d, expected: 7)", n)
	}
	if s != 0 {
		t.Errorf("incorrect space (got: %d, expected: 0)", s)
	}
	if l != 10 {
		t.Errorf("incorrect len (got: %d, expected: 10) ", l)
	}

	// __________
	o = make([]byte, 20)
	n, err = b.Read(o)
	s = b.Space()
	l = b.Len()
	if err != nil {
		t.Errorf("failed to read from buffer (err: %v)", err)
	}
	if n != 10 {
		t.Errorf("incorrect n (got: %d, expected: 10)", n)
	}
	if !bytes.Equal(o[:n], []byte("hijKLMNOPQ")) {
		t.Errorf("read incorrect bytes (read %s, expected hijKLMNOPQ)", o)
	}
	if s != 10 {
		t.Errorf("incorrect space (got: %d, expected: 10)", s)
	}
	if l != 0 {
		t.Errorf("incorrect len (got: %d, expected: 0) ", l)
	}
}

// Partially fill buffer, read, then fill the buffer, so head is > 0 and tail is < head
func TestCircularBuffer_WraparoundPastZero(t *testing.T) {
	// abcdefghij
	// ^h+t
	// ___defghij
	// ^h ^t
	// KLMdefghij
	//    ^h+t
	// _LM_______
	//  ^t^h
	// uLMnopqrst
	//  ^h+t

	// abcdefghij
	b := &Buffer{Buf: make([]byte, 10)}
	p := []byte("abcdefghijklm")
	n, err := b.Write(p)
	s := b.Space()
	l := b.Len()
	if err != ErrNoSpace {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 10 {
		t.Errorf("incorrect n (got: %d, expected: 10)", n)
	}
	if s != 0 {
		t.Errorf("incorrect space (got: %d, expected: 0)", s)
	}
	if l != 10 {
		t.Errorf("incorrect len (got: %d, expected: 10) ", l)
	}

	// ___defghij
	o := make([]byte, 3)
	n, err = b.Read(o)
	s = b.Space()
	l = b.Len()
	if err != nil {
		t.Errorf("failed to read from buffer (err: %v)", err)
	}
	if !bytes.Equal(o[:n], []byte("abc")) {
		t.Errorf("read incorrect bytes (read: %s, expected: abc)", o)
	}
	if n != 3 {
		t.Errorf("incorrect n (got: %d, expected: 3)", n)
	}
	if s != 3 {
		t.Errorf("incorrect space (got: %d, expected: 3)", s)
	}
	if l != 7 {
		t.Errorf("incorrect len (got: %d, expected: 7) ", l)
	}

	// KLMdefghij
	n, err = b.Write([]byte("KLMNOP"))
	s = b.Space()
	l = b.Len()
	if err != ErrNoSpace {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 3 {
		t.Errorf("incorrect n (got: %d, expected: 3)", n)
	}
	if s != 0 {
		t.Errorf("incorrect space (got: %d, expected: 0)", s)
	}
	if l != 10 {
		t.Errorf("incorrect len (got: %d, expected: 10) ", l)
	}

	// _LM_______
	o = make([]byte, 8)
	n, err = b.Read(o)
	s = b.Space()
	l = b.Len()
	if err != nil {
		t.Errorf("failed to read from buffer (err: %v)", err)
	}
	if !bytes.Equal(o[:n], []byte("defghijK")) {
		t.Errorf("read incorrect bytes (read: %s, expected: defghijK)", o)
	}
	if n != 8 {
		t.Errorf("incorrect n (got: %d, expected: 8)", n)
	}
	if s != 8 {
		t.Errorf("incorrect space (got: %d, expected: 8)", s)
	}
	if l != 2 {
		t.Errorf("incorrect len (got: %d, expected: 2) ", l)
	}

	// uLMnopqrst
	n, err = b.Write([]byte("nopqrstuvwxyz"))
	s = b.Space()
	l = b.Len()
	if err != ErrNoSpace {
		t.Errorf("failed to write to buffer (err: %v)", err)
	}
	if n != 8 {
		t.Errorf("incorrect n (got: %d, expected: 8)", n)
	}
	if s != 0 {
		t.Errorf("incorrect space (got: %d, expected: 0)", s)
	}
	if l != 10 {
		t.Errorf("incorrect len (got: %d, expected: 10) ", l)
	}

	// __________
	o = make([]byte, 20)
	n, err = b.Read(o)
	s = b.Space()
	l = b.Len()
	if err != nil {
		t.Errorf("failed to read from buffer (err: %v)", err)
	}
	if !bytes.Equal(o[:n], []byte("LMnopqrstu")) {
		t.Errorf("read incorrect bytes (read: %s, expected: LMnopqrstu)", o)
	}
	if n != 10 {
		t.Errorf("incorrect n (got: %d, expected: 10)", n)
	}
	if s != 10 {
		t.Errorf("incorrect space (got: %d, expected: 10)", s)
	}
	if l != 0 {
		t.Errorf("incorrect len (got: %d, expected: 0) ", l)
	}
}

// Ensures that random concurrent reads and writes preserve the integrity of the data read from
// the buffer. Can be run with -race detector.
func TestCircularBuffer_Concurrency(t *testing.T) {
	b := &Buffer{Buf: make([]byte, 15)}

	rBuf := make([]byte, 100_000)
	_, _ = cryptorand.Read(rBuf)

	go func() {
		for i := 0; i < len(rBuf); i++ {
			for {
				n, err := b.Write(rBuf[i : i+1])
				if err != nil || n != 1 {
					if n == 0 && err == ErrNoSpace {
						continue
					} else {
						t.Errorf("failed to write to buffer (n: %d, err: %v)", n, err)
					}
				}
				break
			}
		}
	}()

	p := make([]byte, len(rBuf))
	for i := 0; i < len(rBuf); i++ {
		for {
			n, err := b.Read(p[i : i+1])
			if err != nil || n != 1 {
				if n == 0 && err == io.EOF {
					continue
				} else {
					t.Errorf("failed to read from buffer (n: %d, err: %v)", n, err)
				}
			}
			break
		}
	}

	if !bytes.Equal(p, rBuf) {
		t.Errorf("read incorrect bytes")
	}
}
