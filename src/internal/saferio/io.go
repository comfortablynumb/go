// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package saferio provides I/O functions that avoid allocating large
// amounts of memory unnecessarily. This is intended for packages that
// read data from an [io.Reader] where the size is part of the input
// data but the input may be corrupt, or may be provided by an
// untrustworthy attacker.
package saferio

import (
	"io"
	"reflect"
)

// chunk is an arbitrary limit on how much memory we are willing
// to allocate without concern.
const chunk = 10 << 20 // 10M

// ReadData reads n bytes from the input stream, but avoids allocating
// all n bytes if n is large. This avoids crashing the program by
// allocating all n bytes in cases where n is incorrect.
func ReadData(r io.Reader, n uint64) ([]byte, error) {
	if int64(n) < 0 || n != uint64(int(n)) {
		// n is too large to fit in int, so we can't allocate
		// a buffer large enough. Treat this as a read failure.
		return nil, io.ErrUnexpectedEOF
	}

	if n < chunk {
		buf := make([]byte, n)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	var buf []byte
	buf1 := make([]byte, chunk)
	for n > 0 {
		next := n
		if next > chunk {
			next = chunk
		}
		_, err := io.ReadFull(r, buf1[:next])
		if err != nil {
			return nil, err
		}
		buf = append(buf, buf1[:next]...)
		n -= next
	}
	return buf, nil
}

// ReadDataAt reads n bytes from the input stream at off, but avoids
// allocating all n bytes if n is large. This avoids crashing the program
// by allocating all n bytes in cases where n is incorrect.
func ReadDataAt(r io.ReaderAt, n uint64, off int64) ([]byte, error) {
	if int64(n) < 0 || n != uint64(int(n)) {
		// n is too large to fit in int, so we can't allocate
		// a buffer large enough. Treat this as a read failure.
		return nil, io.ErrUnexpectedEOF
	}

	if n < chunk {
		buf := make([]byte, n)
		_, err := r.ReadAt(buf, off)
		if err != nil {
			// io.SectionReader can return EOF for n == 0,
			// but for our purposes that is a success.
			if err != io.EOF || n > 0 {
				return nil, err
			}
		}
		return buf, nil
	}

	var buf []byte
	buf1 := make([]byte, chunk)
	for n > 0 {
		next := n
		if next > chunk {
			next = chunk
		}
		_, err := r.ReadAt(buf1[:next], off)
		if err != nil {
			return nil, err
		}
		buf = append(buf, buf1[:next]...)
		n -= next
		off += int64(next)
	}
	return buf, nil
}

// SliceCap returns the capacity to use when allocating a slice.
// After the slice is allocated with the capacity, it should be
// built using append. This will avoid allocating too much memory
// if the capacity is large and incorrect.
//
// A negative result means that the value is always too big.
//
// The element type is described by passing a value of that type.
// This would ideally use generics, but this code is built with
// the bootstrap compiler which need not support generics.
func SliceCap(v any, c uint64) int {
	if int64(c) < 0 || c != uint64(int(c)) {
		return -1
	}
	size := reflect.TypeOf(v).Size()
	if uintptr(c)*size > chunk {
		c = uint64(chunk / size)
		if c == 0 {
			c = 1
		}
	}
	return int(c)
}
