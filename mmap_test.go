// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mmap

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestOpen(t *testing.T) {
	const filename = "mmap_test.go"
	for _, tc := range []struct {
		name string
		open func(fname string) (*File, error)
	}{
		{
			name: "open",
			open: Open,
		},
		{
			name: "open-read-only",
			open: func(fname string) (*File, error) {
				return OpenFile(fname, os.O_RDONLY)
			},
		},
		//		{
		//			name: "open-read-write",
		//			open: func(fname string) (*File, error) {
		//				return OpenFile(fname, os.O_RDWR)
		//			},
		//		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Open(filename)
			if err != nil {
				t.Fatalf("Open: %+v", err)
			}
			defer r.Close()

			_, err = r.Stat()
			if err != nil {
				t.Fatalf("could not stat file: %+v", err)
			}

			if !r.rflag() {
				t.Fatal("not open for reading")
			}

			got := make([]byte, r.Len())
			if _, err := r.ReadAt(got, 0); err != nil && err != io.EOF {
				t.Fatalf("ReadAt: %v", err)
			}
			want, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatalf("ioutil.ReadFile: %v", err)
			}
			if len(got) != len(want) {
				t.Fatalf("got %d bytes, want %d", len(got), len(want))
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("\ngot  %q\nwant %q", string(got), string(want))
			}

			t.Run("Read", func(t *testing.T) {
				got := make([]byte, 32)
				_, err := io.ReadFull(r, got)
				if err != nil {
					t.Fatalf("%+v", err)
				}

				if got, want := got, want[:len(got)]; !bytes.Equal(got, want) {
					t.Fatalf("invalid Read: got=%q, want=%q", got, want)
				}

				pos, err := r.Seek(0, io.SeekCurrent)
				if err != nil {
					t.Fatalf("could not seek: %+v", err)
				}
				if got, want := pos, int64(32); got != want {
					t.Fatalf("invalid position: got=%d, want=%d", got, want)
				}
			})

			t.Run("At", func(t *testing.T) {
				got := r.At(32)
				if got, want := got, want[32]; got != want {
					t.Fatalf("invalid At: got=%q, want=%q", got, want)
				}
			})

			t.Run("ReadByte", func(t *testing.T) {
				_, err := r.Seek(32, io.SeekStart)
				if err != nil {
					t.Fatalf("could not seek: %+v", err)
				}

				got, err := r.ReadByte()
				if err != nil {
					t.Fatalf("could not read byte: %+v", err)
				}

				if got, want := got, want[32]; got != want {
					t.Fatalf("invalid byte: got=%q, want=%q", got, want)
				}
			})

			t.Run("Seek", func(t *testing.T) {
				_, err := r.Seek(32, io.SeekStart)
				if err != nil {
					t.Fatalf("could not seek: %+v", err)
				}

				got, err := r.ReadByte()
				if err != nil {
					t.Fatalf("could not read byte: %+v", err)
				}

				if got, want := got, want[32]; got != want {
					t.Fatalf("invalid byte: got=%q, want=%q", got, want)
				}

				_, err = r.Seek(32, io.SeekCurrent)
				if err != nil {
					t.Fatalf("could not seek: %+v", err)
				}

				got, err = r.ReadByte()
				if err != nil {
					t.Fatalf("could not read byte: %+v", err)
				}

				if got, want := got, want[64+1]; got != want {
					t.Fatalf("invalid byte: got=%q, want=%q", got, want)
				}

				_, err = r.Seek(32, io.SeekEnd)
				if err != nil {
					t.Fatalf("could not seek: %+v", err)
				}

				got, err = r.ReadByte()
				if err != nil {
					t.Fatalf("could not read byte: %+v", err)
				}

				if got, want := got, want[len(want)-32]; got != want {
					t.Fatalf("invalid byte: got=%q, want=%q", got, want)
				}

			})

			t.Run("write", func(t *testing.T) {
				_, err = r.Write([]byte("hello"))
				if err == nil {
					t.Fatal("expected an error")
				}
				if got, want := err, errBadFD; got.Error() != want.Error() {
					t.Fatalf("invalid error:\ngot= %+v\nwant=%+v", got, want)
				}

			})

			t.Run("write-at", func(t *testing.T) {
				_, err = r.WriteAt([]byte("hello"), 0)
				if err == nil {
					t.Fatal("expected an error")
				}
				if got, want := err, errBadFD; got.Error() != want.Error() {
					t.Fatalf("invalid error:\ngot= %+v\nwant=%+v", got, want)
				}

			})

			t.Run("write-byte", func(t *testing.T) {
				err = r.WriteByte('h')
				if err == nil {
					t.Fatal("expected an error")
				}
				if got, want := err, errBadFD; got.Error() != want.Error() {
					t.Fatalf("invalid error:\ngot= %+v\nwant=%+v", got, want)
				}
			})

			err = r.Close()
			if err != nil {
				t.Fatalf("could not close mmap reader: %+v", err)
			}
		})
	}
}
