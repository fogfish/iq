//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package sink

import (
	"context"
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/fogfish/stream"
)

// File writes documents to a single file.
// All documents are concatenated into one output file with newline separators.
// The filesystem is provided by the client (e.g., lfs, stream.NewFS).
type File struct {
	fsys stream.CreateFS[struct{}]
	path string
	file stream.File
}

// NewFile creates a sink that writes to a single file.
// The fs parameter should be a CreateFS filesystem (e.g., from lfs.New or stream.NewFS).
// Path is passed as-is to fsys.Create() - the client is responsible for proper formatting.
func NewFile(fsys stream.CreateFS[struct{}], path string) (iosystem.Sink, error) {
	if path == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	return &File{
		fsys: fsys,
		path: path,
	}, nil
}

// Write appends the document content to the file.
// The file is created on the first write and kept open until Close().
func (f *File) Write(ctx context.Context, doc *iosystem.Document) error {
	switch doc.Type {
	case codec.ContentPNG, codec.ContentJPG:
		return f.writeImage(ctx, doc)
	default:
		return f.writeText(ctx, doc)
	}
}

func (f *File) writeText(ctx context.Context, doc *iosystem.Document) error {
	// Open file on first write
	if f.file == nil {
		file, err := f.fsys.Create(f.path, nil)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", f.path, err)
		}
		f.file = file
	}

	// Write document content
	_, err := io.Copy(f.file, doc.Reader)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", f.path, err)
	}

	// Add newline separator between documents
	_, err = f.file.Write([]byte("\n"))
	return err
}

func (f *File) writeImage(ctx context.Context, doc *iosystem.Document) error {
	doc.Key = iosystem.Key(f.path)
	fd, err := f.fsys.Create(string(doc.Key), nil)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", doc.Key, err)
	}
	defer fd.Close()

	// Write document content
	_, err = io.Copy(fd, doc.Reader)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", doc.Key, err)
	}

	return nil
}

// Close closes the file and releases resources.
func (f *File) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}
