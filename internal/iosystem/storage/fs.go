//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
)

// FSStorage wraps github.com/fogfish/stream for filesystem and S3 operations.
type FSStorage struct {
	fs   spool.FileSystem
	base string // Base directory path
}

// NewFS creates filesystem storage.
// Supports local paths and s3:// URLs via stream library.
func NewFS(path string) (*FSStorage, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	var fsys spool.FileSystem
	var err error
	var base string

	// Mount appropriate filesystem
	const s3pfx = "s3://"
	if strings.HasPrefix(path, s3pfx) {
		base = path[len(s3pfx):]
		fsys, err = stream.NewFS(base)
	} else {
		base = path
		fsys, err = lfs.New(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to mount storage at %s: %w", path, err)
	}

	return &FSStorage{
		fs:   fsys,
		base: base,
	}, nil
}

// Put writes value to key.
func (s *FSStorage) Put(ctx context.Context, key iosystem.Key, value io.Reader) error {
	file, err := s.fs.Create(string(key), nil)
	if err != nil {
		return fmt.Errorf("failed to create key %s: %w", key, err)
	}
	defer file.Close()

	_, err = io.Copy(file, value)
	if err != nil {
		file.Cancel()
		return fmt.Errorf("failed to write key %s: %w", key, err)
	}

	return nil
}

// Get reads value from key.
func (s *FSStorage) Get(ctx context.Context, key iosystem.Key) (io.ReadCloser, error) {
	reader, err := s.fs.Open(string(key))
	if err != nil {
		return nil, fmt.Errorf("failed to open key %s: %w", key, err)
	}
	return reader, nil
}

// Has checks if key exists.
func (s *FSStorage) Has(ctx context.Context, key iosystem.Key) (bool, error) {
	reader, err := s.fs.Open(string(key))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check key %s: %w", key, err)
	}
	reader.Close()
	return true, nil
}

// Walk traverses keys matching prefix.
func (s *FSStorage) Walk(ctx context.Context, prefix iosystem.Key, visitor func(*iosystem.Document) error) error {
	searchPath := string(prefix)
	if searchPath == "" {
		searchPath = "."
	}

	err := fs.WalkDir(s.fs, searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Create document from file
		key := iosystem.Key(path)
		reader, err := s.Get(ctx, key)
		if err != nil {
			return err
		}

		doc := iosystem.NewDocument(key, reader)
		return visitor(doc)
	})

	if err != nil {
		return fmt.Errorf("failed to walk prefix %s: %w", prefix, err)
	}

	return nil
}
