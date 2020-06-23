// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mmap

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	syscall "golang.org/x/sys/windows"
)

func openFile(filename string, fl Flag) (*File, error) {
	f, err := os.OpenFile(filename, fl.flag(), 0666)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		return &File{fd: f, flag: fl, fi: fi}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	prot := uint32(syscall.PAGE_READONLY)
	view := uint32(syscall.FILE_MAP_READ)
	if fl&Write != 0 {
		prot = syscall.PAGE_READWRITE
		view = syscall.FILE_MAP_WRITE
	}

	low, high := uint32(size), uint32(size>>32)
	fmap, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, prot, high, low, nil)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(fmap)
	ptr, err := syscall.MapViewOfFile(fmap, view, 0, 0, uintptr(size))
	if err != nil {
		return nil, err
	}
	data := (*[maxBytes]byte)(unsafe.Pointer(ptr))[:size]

	fd := &File{
		data: data,
		fd:   f,
		fi:   fi,
		flag: fl,
	}
	runtime.SetFinalizer(fd, (*File).Close)
	return fd, nil

}

// Sync commits the current contents of the file to stable storage.
func (f *File) Sync() error {
	if !f.wflag() {
		return errBadFD
	}

	err := syscall.FlushViewOfFile(f.addr(), uintptr(len(f.data)))
	if err != nil {
		return fmt.Errorf("mmap: could not sync view: %w", err)
	}

	err = syscall.FlushFileBuffers(syscall.Handle(f.fd.Fd()))
	if err != nil {
		return fmt.Errorf("mmap: could not sync file buffers: %w", err)
	}

	return nil
}

// Close closes the reader.
func (f *File) Close() error {
	if f.data == nil {
		return nil
	}
	defer f.fd.Close()

	addr := f.addr()
	f.data = nil
	runtime.SetFinalizer(f, nil)
	return syscall.UnmapViewOfFile(addr)
}

func (f *File) addr() uintptr {
	data := f.data
	return uintptr(unsafe.Pointer(&data[0]))
}
