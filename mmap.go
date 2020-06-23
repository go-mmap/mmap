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

// Flag specifies how a mmap file should be opened.
type Flag int

const (
	Read  Flag = 0x1 // Read enables read-access to a mmap file.
	Write Flag = 0x2 // Write enables write-access to a mmap file.
)

func (fl Flag) flag() int {
	var flag int

	switch fl {
	case Read:
		flag = os.O_RDONLY
	case Write:
		flag = os.O_WRONLY
	case Read | Write:
		flag = os.O_RDWR
	}

	return flag
}

// File reads/writes a memory-mapped file.
type File struct {
	data []byte
	c    int

	flag Flag
	fi   os.FileInfo
}

// Open memory-maps the named file for reading.
func Open(filename string) (*File, error) {
	return openFile(filename, Read)
}

// OpenFile memory-maps the named file for reading/writing, depending on
// the flag value.
func OpenFile(filename string, flag Flag) (*File, error) {
	return openFile(filename, flag)
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
	return f.flag&Read != 0
}

func (f *File) wflag() bool {
	return f.flag&Write != 0
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
