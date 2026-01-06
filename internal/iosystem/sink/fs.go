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

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

// TODO: rename

// FileSystem writes documents to a filesystem directory.
// It preserves the document path structure.
// The filesystem is provided by the client (e.g., lfs, stream.NewFS).
type FileSystem struct {
	storage storage.Storage
}

// NewStorage creates a sink that writes to a filesystem directory.
// The fs parameter should be a CreateFS filesystem (e.g., from lfs.New or stream.NewFS).
// Document paths are passed as-is to fsys.Create() - the client is responsible for proper formatting.
func NewStorage(storage storage.Storage) (iosystem.Sink, error) {
	return &FileSystem{
		storage: storage,
	}, nil
}

// Write writes a document to the filesystem, preserving its path structure.
func (s *FileSystem) Write(ctx context.Context, doc *iosystem.Document) error {
	err := s.storage.Put(ctx, doc.Key, doc.Reader)
	if err != nil {
		return fmt.Errorf("failed to write key %s: %w", doc.Key, err)
	}

	return nil
	// panic("not implemented")
	/*
		if doc == nil {
			return fmt.Errorf("document is nil")
		}

		// Create the file using document's path
		file, err := s.fsys.Create(doc.FilePath(), nil)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", doc.Path, err)
		}
		defer file.Close()

		// Copy document content to file
		_, err = io.Copy(file, doc.Reader)
		if err != nil {
			// Try to cancel the file creation on error
			file.Cancel()
			return fmt.Errorf("failed to write file %s: %w", doc.Path, err)
		}

		return nil
	*/
}

// Close implements iosystem.Sink.
func (s *FileSystem) Close() error {
	// No cleanup needed - CreateFS doesn't require closing
	return nil
}
