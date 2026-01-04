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
	"path/filepath"
	"strings"
	"sync"

	"github.com/fogfish/iq/internal/iosystem"
)

// FS traverses a filesystem directory and yields documents for each file.
// It uses fs.WalkDir in a background goroutine for progressive file discovery.
// The goroutine is lazily initialized on the first call to Next().
// Directories are skipped - only regular files are yielded as documents.
type FS struct {
	fsys fs.FS
	dir  string

	// Lazy initialization
	once     sync.Once
	pathChan chan string
	errChan  chan error
	doneChan chan struct{}
}

// NewFS creates a source that walks through a filesystem directory progressively.
// The fs.FS parameter should support fs.ReadDirFS (e.g., from lfs.New or stream.NewFS).
// The dir parameter specifies the starting directory path.
// For directories, the path should end with `/` as per stream library conventions.
//
// The walk operation is lazily initialized on the first call to Next(), running in
// a background goroutine that feeds file paths through a buffered channel. This means:
//   - No startup delay: NewFS returns immediately
//   - Directory validation is deferred until Next() is called
//   - Memory efficient: Only buffer size (100) paths held in memory at a time
//   - Works with both local filesystem (lfs) and S3 (stream)
func NewFS(fsys fs.FS, dir string) (*FS, error) {
	if fsys == nil {
		return nil, fmt.Errorf("filesystem is nil")
	}

	return &FS{
		fsys: fsys,
		dir:  dir,
	}, nil
}

// init lazily initializes the channels and starts the background walker goroutine.
// It's safe to call multiple times due to sync.Once.
func (w *FS) init() {
	w.once.Do(func() {
		w.pathChan = make(chan string)
		w.errChan = make(chan error, 1)
		w.doneChan = make(chan struct{})

		// Start background goroutine to walk directory
		go w.walkInBackground()
	})
}

// walkInBackground runs fs.WalkDir in a goroutine, sending discovered file paths
// to pathChan. It respects cancellation via doneChan.
func (w *FS) walkInBackground() {
	defer close(w.pathChan)
	defer close(w.errChan)

	err := fs.WalkDir(w.fsys, w.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-w.doneChan:
			return fs.SkipAll
		default:
		}

		if d.IsDir() {
			return nil
		}

		w.pathChan <- path

		return nil
	})

	// Send error before closing channels
	if err != nil {
		w.errChan <- fmt.Errorf("failed to walk directory %s: %w", w.dir, err)
	}
}

// Next returns the next file document or io.EOF when all files are read.
// Files are wrapped in an auto-closer to prevent descriptor leaks.
// The first call to Next() lazily initializes the background walker goroutine.
func (w *FS) Next(ctx context.Context) (*iosystem.Document, error) {
	// Lazy initialization on first call
	w.init()

	// Check context cancellation first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read next path
	select {
	case path, ok := <-w.pathChan:
		if !ok {
			return nil, io.EOF
		}

		// Construct relative key from filesystem path
		key := w.fsPathToKey(path)

		file, err := w.fsys.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", path, err)
		}

		doc := iosystem.NewDocument(key, &autoCloser{ReadCloser: file})
		switch filepath.Ext(path) {
		case ".json":
			doc.Type = iosystem.ContentJSON
		case ".yaml", ".yml":
			doc.Type = iosystem.ContentYAML
		default:
			doc.Type = iosystem.ContentText
		}

		return doc, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	case err, ok := <-w.errChan:
		if ok {
			return nil, err
		}
		return nil, io.EOF
	}
}

// fsPathToKey converts filesystem path to relative key.
// Examples:
//
//	"/base/sub/a.txt" → "sub/a.txt"
//	"/base/a.txt" → "a.txt"
//	"sub/a.txt" → "sub/a.txt" (already relative)
func (w *FS) fsPathToKey(fsPath string) iosystem.Key {
	// Remove leading slash if present (from fs.WalkDir)
	clean := strings.TrimPrefix(fsPath, "/")

	// Remove base directory prefix
	relPath := strings.TrimPrefix(clean, w.dir)
	relPath = strings.TrimPrefix(relPath, "/")

	// Ensure forward slashes (portable)
	relPath = filepath.ToSlash(relPath)

	return iosystem.Key(relPath)
}

// Close signals the background walker to stop and drains the path channel
// to prevent goroutine leaks. It's safe to call Close() even if Next() was never called.
func (w *FS) Close() error {
	// Only close if initialized
	w.once.Do(func() {
		// Not initialized, nothing to close
	})

	// If channels exist, signal and drain
	if w.doneChan != nil {
		close(w.doneChan)

		// Drain path channel to unblock walker goroutine
		if w.pathChan != nil {
			for range w.pathChan {
			}
		}
	}

	return nil
}
