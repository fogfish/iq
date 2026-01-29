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
	"path/filepath"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
)

type FS interface {
	stream.CreateFS[struct{}]
	Stat(path string) (fs.FileInfo, error)
}

// FileSystem
type FileSystem struct {
	fs    FS
	mount string
	files []string
}

// NewFileSystem creates filesystem storage.
func NewFileSystem(mount string, files ...string) (*FileSystem, error) {
	if mount == "" {
		return nil, fmt.Errorf("mount cannot be empty")
	}

	const s3pfx = "s3://"
	if strings.HasPrefix(mount, s3pfx) {
		fsys, err := stream.NewFS(mount[len(s3pfx):])
		if err != nil {
			return nil, fmt.Errorf("failed to mount S3 storage at %s: %w", mount, err)
		}
		return &FileSystem{
			fs:    fsys,
			mount: mount,
			files: files,
		}, nil
	}

	absMount, err := filepath.Abs(mount)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for mount %s: %w", mount, err)
	}

	fsys, err := lfs.New(mount)
	if err != nil {
		return nil, fmt.Errorf("failed to mount local storage at %s: %w", mount, err)
	}

	return &FileSystem{
		fs:    fsys,
		mount: absMount,
		files: files,
	}, nil
}

// Put writes value to key.
func (s *FileSystem) Put(ctx context.Context, key iosystem.Key, value io.Reader) (string, error) {
	file, err := s.fs.Create(string(key), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create key %s: %w", key, err)
	}
	defer file.Close()

	_, err = io.Copy(file, value)
	if err != nil {
		file.Cancel()
		return "", fmt.Errorf("failed to write key %s: %w", key, err)
	}

	return s.mount + "/" + string(key), nil
}

// Get reads value from key.
func (s *FileSystem) Get(ctx context.Context, key iosystem.Key) (io.Reader, error) {
	reader, err := s.fs.Open(string(key))
	if err != nil {
		return nil, fmt.Errorf("failed to open key %s: %w", key, err)
	}
	return &safeIO{ReadCloser: reader}, nil
}

// Has checks if key exists.
func (s *FileSystem) Has(ctx context.Context, key iosystem.Key) (bool, error) {
	fi, err := s.fs.Stat(string(key))
	if err != nil {
		return false, nil
	}

	return fi != nil, nil
}

// Walk traverses keys matching prefix.
func (s *FileSystem) Walk(ctx context.Context, prefix iosystem.Key, visitor func(*iosystem.Document) error) error {
	// Walk only specified files
	if len(s.files) > 0 {
		for _, file := range s.files {
			file = strings.TrimPrefix(file, "./")
			key := iosystem.Key(file)
			reader, err := s.Get(ctx, key)
			if err != nil {
				return err
			}

			doc := iosystem.NewDocument(key, codec.Default.DetectContentType(string(key)), reader)
			if err := visitor(doc); err != nil {
				return err
			}
		}
		return nil
	}

	// Walk all files under prefix
	return s.walk(ctx, prefix, visitor)
}

func (s *FileSystem) walk(ctx context.Context, prefix iosystem.Key, visitor func(*iosystem.Document) error) error {
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

		key := iosystem.Key(strings.TrimPrefix(path, "/"))
		reader, err := s.Get(ctx, key)
		if err != nil {
			return err
		}

		doc := iosystem.NewDocument(key, codec.Default.DetectContentType(string(key)), reader)
		return visitor(doc)
	})

	if err != nil {
		return fmt.Errorf("failed to walk prefix %s: %w", prefix, err)
	}

	return nil
}

// safeIO wraps an io.ReadCloser and automatically closes it when:
// - Read returns io.EOF (file fully read)
// - Read returns any other error
// This prevents file descriptor leaks when documents are not explicitly closed.
type safeIO struct {
	io.ReadCloser
	closed bool
}

func (a *safeIO) Read(p []byte) (n int, err error) {
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

func (a *safeIO) Close() error {
	if a.closed {
		return nil
	}
	a.closed = true
	return a.ReadCloser.Close()
}
