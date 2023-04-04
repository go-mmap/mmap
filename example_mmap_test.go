// Copyright 2020 The go-mmap Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mmap_test

import (
	"fmt"
	"log"
	"os"

	"github.com/go-mmap/mmap"
)

func ExampleOpen() {
	f, err := mmap.Open("example_mmap_test.go")
	if err != nil {
		log.Fatalf("could not mmap file: %+v", err)
	}
	defer f.Close()

	buf := make([]byte, 32)
	_, err = f.Read(buf)
	if err != nil {
		log.Fatalf("could not read into buffer: %+v", err)
	}

	fmt.Printf("%s\n", buf[:12])

	// Output:
	// // Copyright
}

func ExampleOpenFile_read() {
	f, err := mmap.OpenFile("example_mmap_test.go", mmap.Read)
	if err != nil {
		log.Fatalf("could not mmap file: %+v", err)
	}
	defer f.Close()

	buf := make([]byte, 32)
	_, err = f.ReadAt(buf, 0)
	if err != nil {
		log.Fatalf("could not read into buffer: %+v", err)
	}

	fmt.Printf("%s\n", buf[:12])

	// Output:
	// // Copyright
}

func ExampleOpenFile_readwrite() {
	f, err := os.CreateTemp("", "mmap-")
	if err != nil {
		log.Fatalf("could not create tmp file: %+v", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write([]byte("hello world!"))
	if err != nil {
		log.Fatalf("could not write data: %+v", err)
	}

	err = f.Close()
	if err != nil {
		log.Fatalf("could not close file: %+v", err)
	}

	raw, err := os.ReadFile(f.Name())
	if err != nil {
		log.Fatalf("could not read back data: %+v", err)
	}

	fmt.Printf("%s\n", raw)

	rw, err := mmap.OpenFile(f.Name(), mmap.Read|mmap.Write)
	if err != nil {
		log.Fatalf("could not open mmap file: %+v", err)
	}
	defer rw.Close()

	_, err = rw.Write([]byte("bye!"))
	if err != nil {
		log.Fatalf("could not write to mmap file: %+v", err)
	}

	raw, err = os.ReadFile(f.Name())
	if err != nil {
		log.Fatalf("could not read back data: %+v", err)
	}

	fmt.Printf("%s\n", raw)

	// Output:
	// hello world!
	// bye!o world!
}
