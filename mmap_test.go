// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mmap

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
				return OpenFile(fname, Read)
			},
		},
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

func TestWrite(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mmap-")
	if err != nil {
		t.Fatalf("could not create temp dir: %+v", err)
	}
	defer os.RemoveAll(tmp)

	display := func(fname string) []byte {
		t.Helper()
		raw, err := ioutil.ReadFile(fname)
		if err != nil {
			t.Fatalf("could not read file %q: %+v", fname, err)
		}
		return raw
	}

	for _, tc := range []struct {
		name  string
		flags Flag
	}{
		// {
		// 	name:  "write-only",
		// 	flags: Write,
		// },
		{
			name:  "read-write",
			flags: Read | Write,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fname := filepath.Join(tmp, tc.name+".txt")
			err := ioutil.WriteFile(fname, []byte("hello world!\nbye.\n"), 0644)
			if err != nil {
				t.Fatalf("could not seed file: %+v", err)
			}

			f, err := OpenFile(fname, tc.flags)
			if err != nil {
				t.Fatalf("could not mmap file: %+v", err)
			}
			defer f.Close()

			_, err = f.WriteAt([]byte("bye!\n"), 3)
			if err != nil {
				t.Fatalf("could not write-at: %+v", err)
			}

			if got, want := display(fname), []byte("helbye!\nrld!\nbye.\n"); !bytes.Equal(got, want) {
				t.Fatalf("invalid content:\ngot= %q\nwant=%q\n", got, want)
			}

			_, err = f.Seek(0, io.SeekStart)
			if err != nil {
				t.Fatalf("could not seek to start: %+v", err)
			}

			_, err = f.Write([]byte("hello world!\nbye\n"))
			if err != nil {
				t.Fatalf("could not write: %+v", err)
			}

			if got, want := display(fname), []byte("hello world!\nbye\n\n"); !bytes.Equal(got, want) {
				t.Fatalf("invalid content:\ngot= %q\nwant=%q\n", got, want)
			}

			_, err = f.Seek(5, io.SeekEnd)
			if err != nil {
				t.Fatalf("could not seek from end: %+v", err)
			}

			err = f.WriteByte('t')
			if err != nil {
				t.Fatalf("could not write-byte: %+v", err)
			}

			if got, want := display(fname), []byte("hello world!\ntye\n\n"); !bytes.Equal(got, want) {
				t.Fatalf("invalid content:\ngot= %q\nwant=%q\n", got, want)
			}

		})
	}
}
