//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/fogfish/iq/internal/iosystem"
)

// File reads one or more files sequentially from a filesystem.
// A single file is a special case where paths contains one element.
type File struct {
	fsys  fs.FS
	paths []string
	index int
}

// NewFile creates a source that reads files from the given filesystem.
// The fs.FS parameter is provided by the client (e.g., lfs.FS, stream.FS).
// Paths are passed as-is to fs.Open() - the client is responsible for proper formatting.
func NewFile(fsys fs.FS, paths ...string) (iosystem.Source, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one file path is required")
	}

	return &File{
		fsys:  fsys,
		paths: paths,
	}, nil
}

// Next returns the next file document or io.EOF when all files are read.
// Files are wrapped in an auto-closer to prevent descriptor leaks.
func (f *File) Next(ctx context.Context) (*iosystem.Document, error) {
	if f.index >= len(f.paths) {
		return nil, io.EOF
	}

	path := f.paths[f.index]
	f.index++

	file, err := f.fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	return iosystem.NewDocument(path, &autoCloser{ReadCloser: file}), nil
}

// Close implements iosystem.Source.
func (f *File) Close() error {
	return nil
}

//------------------------------------------------------------------------------

// autoCloser wraps an io.ReadCloser and automatically closes it when:
// - Read returns io.EOF (file fully read)
// - Read returns any other error
// This prevents file descriptor leaks when documents are not explicitly closed.
type autoCloser struct {
	io.ReadCloser
	closed bool
}

func (a *autoCloser) Read(p []byte) (n int, err error) {
	if a.closed {
		return 0, io.EOF
	}

	n, err = a.ReadCloser.Read(p)
	if err != nil {
		a.Close()
		a.closed = true
	}
	return n, err
}

func (a *autoCloser) Close() error {
	if a.closed {
		return nil
	}
	a.closed = true
	return a.ReadCloser.Close()
}
