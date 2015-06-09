// Copyright (c) 2015 SermoDigital, LLC.

package bytepool

import (
	"errors"
	"io"
)

// These are loosely borrowed from bytes.Buffer. Refer to that pacakge
// for proper documentation.

var ErrTooLarge = errors.New("bffer: too large")

type Buffer struct {
	Buf       []byte   // Buffer contents.
	off       int      // read at &buf[off], write at &buf[len(off)]
	bootstrap [64]byte // memory to hold first slice; helps small Buffers (Printf) avoid allocation.
}

func NewBuffer(size int) *Buffer {
	return &Buffer{Buf: make([]byte, size)}
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	m := b.grow(len(p))
	return copy(b.Buf[m:], p), nil
}

func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	if b.off < len(b.Buf) {
		nBytes := b.Len()
		m, e := w.Write(b.Buf[b.off:])
		if m > nBytes {
			panic("Buffer.WriteTo: invalid Write count")
		}
		b.off += m
		n = int64(m)
		if e != nil {
			return n, e
		}
		// all bytes should have been written, by definition of
		// Write method in io.Writer
		if m != nBytes {
			return n, io.ErrShortWrite
		}
	}
	// Truncate removed.
	return
}

func (b *Buffer) Truncate(n int) {
	switch {
	case n < 0 || n > len(b.Buf):
		panic("Buffer: truncation out of range")
	case n == 0:
		b.off = 0
	}
	b.Buf = b.Buf[0 : b.off+n]
}

func (b *Buffer) Reset() {
	b.Truncate(0)
}

func (b *Buffer) Len() int {
	return len(b.Buf) - b.off
}

func (b *Buffer) grow(n int) int {
	m := b.Len()
	// If Buffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Truncate(0)
	}
	if len(b.Buf)+n > cap(b.Buf) {
		var buf []byte
		if b.Buf == nil && n <= len(b.bootstrap) {
			buf = b.bootstrap[0:]
		} else if m+n <= cap(b.Buf)/2 {
			// We can slide things down instead of allocating a new
			// slice. We only need m+n <= cap(b.Buf) to slide, but
			// we instead let capacity get twice as large so we
			// don't spend all our time copying.
			copy(b.Buf[:], b.Buf[b.off:])
			buf = b.Buf[:m]
		} else {
			// not enough space anywhere
			buf = makeSlice(2*cap(b.Buf) + n)
			copy(buf, b.Buf[b.off:])
		}
		b.Buf = buf
		b.off = 0
	}
	b.Buf = b.Buf[0 : b.off+m+n]
	return b.off + m
}

func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	return make([]byte, n)
}
