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

func openFile(filename string, fl int) (*File, error) {
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
		return &File{flag: fl, fi: fi}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	prot := uint32(syscall.PAGE_READONLY)
	if fl&wFlag != 0 {
		prot = syscall.PAGE_READWRITE
	}

	low, high := uint32(size), uint32(size>>32)
	fmap, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, prot, high, low, nil)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(fmap)
	ptr, err := syscall.MapViewOfFile(fmap, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		return nil, err
	}
	data := (*[maxBytes]byte)(unsafe.Pointer(ptr))[:size]

	fd := &File{
		data: data,
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
	panic("not implemented")
}

// Close closes the reader.
func (f *File) Close() error {
	if f.data == nil {
		return nil
	}
	data := f.data
	f.data = nil
	runtime.SetFinalizer(f, nil)
	return syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&data[0])))
}
