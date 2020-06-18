// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux darwin

// Package mmap provides a way to memory-map a file.
package mmap

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
)

// Reader reads a memory-mapped file.
//
// Like any io.ReaderAt, clients can execute parallel ReadAt calls, but it is
// not safe to call Close and reading methods concurrently.
type Reader struct {
	data []byte
	c    int
}

// Close closes the reader.
func (r *Reader) Close() error {
	if r.data == nil {
		return nil
	}
	data := r.data
	r.data = nil
	runtime.SetFinalizer(r, nil)
	return syscall.Munmap(data)
}

// Len returns the length of the underlying memory-mapped file.
func (r *Reader) Len() int {
	return len(r.data)
}

// At returns the byte at index i.
func (r *Reader) At(i int) byte {
	return r.data[i]
}

func (r *Reader) Read(p []byte) (int, error) {
	if r.c >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.c:])
	r.c += n
	return n, nil
}

func (r *Reader) ReadByte() (byte, error) {
	if r.c >= len(r.data) {
		return 0, io.EOF
	}
	v := r.data[r.c]
	r.c++
	return v, nil
}

// ReadAt implements the io.ReaderAt interface.
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	if r.data == nil {
		return 0, errors.New("mmap: closed")
	}
	if off < 0 || int64(len(r.data)) < off {
		return 0, fmt.Errorf("mmap: invalid ReadAt offset %d", off)
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.c = int(offset)
	case io.SeekCurrent:
		r.c += int(offset)
	case io.SeekEnd:
		r.c = len(r.data) - int(offset)
	default:
		return 0, fmt.Errorf("mmap: invalid whence")
	}
	if r.c < 0 {
		return 0, fmt.Errorf("mmap: negative position")
	}
	return int64(r.c), nil
}

// Open memory-maps the named file for reading.
func Open(filename string) (*Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		return &Reader{}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	r := &Reader{data: data}
	runtime.SetFinalizer(r, (*Reader).Close)
	return r, nil
}

var (
	_ io.Reader     = (*Reader)(nil)
	_ io.ReaderAt   = (*Reader)(nil)
	_ io.Seeker     = (*Reader)(nil)
	_ io.Closer     = (*Reader)(nil)
	_ io.ByteReader = (*Reader)(nil)
)

// File reads/writes a memory-mapped file.
type File struct {
	data []byte
	c    int
}

// Close closes the memory-mapped file.
func (f *File) Close() error {
	if f.data == nil {
		return nil
	}
	data := f.data
	f.data = nil
	runtime.SetFinalizer(f, nil)
	return syscall.Munmap(data)
}

// Len returns the length of the underlying memory-mapped file.
func (f *File) Len() int {
	return len(f.data)
}

// At returns the byte at index i.
func (f *File) At(i int) byte {
	return f.data[i]
}

// Read implements the io.Reader interface.
func (f *File) Read(p []byte) (int, error) {
	if f.c >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.c:])
	f.c += n
	return n, nil
}

// ReadByte implements the io.ByteReader interface.
func (f *File) ReadByte() (byte, error) {
	if f.c >= len(f.data) {
		return 0, io.EOF
	}
	v := f.data[f.c]
	f.c++
	return v, nil
}

// ReadAt implements the io.ReaderAt interface.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
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
	if f.c >= len(f.data) {
		return io.ErrShortWrite
	}
	f.data[f.c] = c
	f.c++
	return nil
}

// WriteAt implements the io.WriterAt interface.
func (f *File) WriteAt(p []byte, off int64) (int, error) {
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

// OpenFile memory-maps the named file for reading/writing.
func OpenFile(filename string, prot int) (*File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		return &File{}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), prot, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	mem := &File{data: data}
	runtime.SetFinalizer(mem, (*File).Close)
	return mem, nil
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
