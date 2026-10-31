// Package storage supplies file backends for native world files. Each backend
// and its handles belong to one caller; concurrent computation happens outside it.
package storage

import (
	"io"
	"io/fs"
	"os"
)

// File supports sector-based random access. Closing a file releases its handle;
// Commit on the owning backend determines when changes are durable.
type File interface {
	fs.File
	io.ReaderAt
	io.WriterAt
	io.Writer
	Truncate(int64) error
}

// Writer extends an fs.FS with the operations needed by world and region files.
// Read-only callers may supply an ordinary fs.FS instead.
type Writer interface {
	fs.FS
	OpenFile(name string, flag int) (File, error)
	MkdirAll(name string) error
	Remove(name string) error
	Rename(oldName, newName string) error
	Commit() error
}

// WriteFile stages one file. Durability is controlled by the backend's Commit.
func WriteFile(dst Writer, name string, data []byte) error {
	f, err := dst.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
