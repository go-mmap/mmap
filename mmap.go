// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mmap provides a way to memory-map a file.
package mmap

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var errBadFD = errors.New("bad file descriptor")

const (
	rFlag = 0x1
	wFlag = 0x2
)

// File reads/writes a memory-mapped file.
type File struct {
	data []byte
	c    int

	flag int
	fi   os.FileInfo
}

// Open memory-maps the named file for reading.
func Open(filename string) (*File, error) {
	return openFile(filename, flagFrom(os.O_RDONLY))
}

// OpenFile memory-maps the named file for reading/writing.
func OpenFile(filename string, flag int) (*File, error) {
	return openFile(filename, flagFrom(flag))
}

// Len returns the length of the underlying memory-mapped file.
func (f *File) Len() int {
	return len(f.data)
}

// At returns the byte at index i.
func (f *File) At(i int) byte {
	return f.data[i]
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *os.PathError.
func (f *File) Stat() (os.FileInfo, error) {
	if f == nil {
		return nil, os.ErrInvalid
	}

	return f.fi, nil
}

func (f *File) rflag() bool {
	return f.flag&rFlag != 0
}

func (f *File) wflag() bool {
	return f.flag&wFlag != 0
}

// Read implements the io.Reader interface.
func (f *File) Read(p []byte) (int, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	if !f.rflag() {
		return 0, errBadFD
	}
	if f.c >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.c:])
	f.c += n
	return n, nil
}

// ReadByte implements the io.ByteReader interface.
func (f *File) ReadByte() (byte, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	if !f.rflag() {
		return 0, errBadFD
	}
	if f.c >= len(f.data) {
		return 0, io.EOF
	}
	v := f.data[f.c]
	f.c++
	return v, nil
}

// ReadAt implements the io.ReaderAt interface.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	if !f.rflag() {
		return 0, errBadFD
	}
	if f.data == nil {
		return 0, errors.New("mmap: closed")
	}
	if off < 0 || int64(len(f.data)) < off {
		return 0, fmt.Errorf("mmap: invalid ReadAt offset %d", off)
	}
	n := copy(p, f.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Write implements the io.Writer interface.
func (f *File) Write(p []byte) (int, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	if !f.wflag() {
		return 0, errBadFD
	}
	if f.c >= len(f.data) {
		return 0, io.ErrShortWrite
	}
	n := copy(f.data[f.c:], p)
	f.c += n
	if len(p) > n {
		return n, io.ErrShortWrite
	}
	return n, nil
}

// WriteByte implements the io.ByteWriter interface.
func (f *File) WriteByte(c byte) error {
	if f == nil {
		return os.ErrInvalid
	}

	if !f.wflag() {
		return errBadFD
	}
	if f.c >= len(f.data) {
		return io.ErrShortWrite
	}
	f.data[f.c] = c
	f.c++
	return nil
}

// WriteAt implements the io.WriterAt interface.
func (f *File) WriteAt(p []byte, off int64) (int, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	if !f.wflag() {
		return 0, errBadFD
	}
	if f.data == nil {
		return 0, errors.New("mmap: closed")
	}
	if off < 0 || int64(len(f.data)) < off {
		return 0, fmt.Errorf("mmap: invalid WriteAt offset %d", off)
	}
	n := copy(f.data[off:], p)
	if n < len(p) {
		return n, io.ErrShortWrite
	}
	return n, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f == nil {
		return 0, os.ErrInvalid
	}

	switch whence {
	case io.SeekStart:
		f.c = int(offset)
	case io.SeekCurrent:
		f.c += int(offset)
	case io.SeekEnd:
		f.c = len(f.data) - int(offset)
	default:
		return 0, fmt.Errorf("mmap: invalid whence")
	}
	if f.c < 0 {
		return 0, fmt.Errorf("mmap: negative position")
	}
	return int64(f.c), nil
}

func flagFrom(fl int) int {
	var flag int

	fl &= 0xf

	switch {
	case fl == os.O_RDONLY:
		flag = rFlag
	case fl&os.O_RDWR != 0:
		flag = rFlag | wFlag
	case fl&os.O_WRONLY != 0:
		flag = wFlag
	}
	return flag
}

var (
	_ io.Reader     = (*File)(nil)
	_ io.ReaderAt   = (*File)(nil)
	_ io.ByteReader = (*File)(nil)
	_ io.Writer     = (*File)(nil)
	_ io.WriterAt   = (*File)(nil)
	_ io.ByteWriter = (*File)(nil)
	_ io.Closer     = (*File)(nil)
	_ io.Seeker     = (*File)(nil)
)
