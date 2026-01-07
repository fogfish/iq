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
	"sync"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

// Storage traverses a filesystem directory and yields documents for each file.
// It uses fs.WalkDir in a background goroutine for progressive file discovery.
// The goroutine is lazily initialized on the first call to Next().
// Directories are skipped - only regular files are yielded as documents.
type Storage struct {
	storage storage.Storage
	prefix  iosystem.Key

	// Lazy initialization
	once     sync.Once
	docChan  chan *iosystem.Document
	errChan  chan error
	doneChan chan struct{}
}

// NewStorage creates a source that walks through a filesystem directory progressively.
// The fs.FS parameter should support fs.ReadDirFS (e.g., from lfs.New or stream.NewStorage).
// The dir parameter specifies the starting directory path.
// For directories, the path should end with `/` as per stream library conventions.
//
// The walk operation is lazily initialized on the first call to Next(), running in
// a background goroutine that feeds file paths through a buffered channel. This means:
//   - No startup delay: NewStorage returns immediately
//   - Directory validation is deferred until Next() is called
//   - Memory efficient: Only buffer size (100) paths held in memory at a time
//   - Works with both local filesystem (lfs) and S3 (stream)
func NewStorage(storage storage.Storage, prefix iosystem.Key) (*Storage, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage is not defined")
	}

	return &Storage{
		storage: storage,
		prefix:  prefix,
	}, nil
}

// init lazily initializes the channels and starts the background walker goroutine.
// It's safe to call multiple times due to sync.Once.
func (w *Storage) init() {
	w.once.Do(func() {
		w.docChan = make(chan *iosystem.Document)
		w.errChan = make(chan error, 1)
		w.doneChan = make(chan struct{})

		go w.walkInBackground()
	})
}

// walkInBackground runs fs.WalkDir in a goroutine, sending discovered file paths
// to pathChan. It respects cancellation via doneChan.
func (w *Storage) walkInBackground() {
	defer close(w.docChan)
	defer close(w.errChan)

	err := w.storage.Walk(context.Background(), w.prefix,
		func(doc *iosystem.Document) error {
			select {
			case <-w.doneChan:
				return fs.SkipAll
			default:
			}
			w.docChan <- doc

			return nil
		},
	)

	// Send error before closing channels
	if err != nil {
		w.errChan <- fmt.Errorf("failed to walk directory %s: %w", w.prefix, err)
	}
}

// Next returns the next file document or io.EOF when all files are read.
// Files are wrapped in an auto-closer to prevent descriptor leaks.
// The first call to Next() lazily initializes the background walker goroutine.
func (w *Storage) Next(ctx context.Context) (*iosystem.Document, error) {
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
	case doc, ok := <-w.docChan:
		if !ok {
			return nil, io.EOF
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

// Close signals the background walker to stop and drains the path channel
// to prevent goroutine leaks. It's safe to call Close() even if Next() was never called.
func (w *Storage) Close() error {
	// Only close if initialized
	w.once.Do(func() {
		// Not initialized, nothing to close
	})

	// If channels exist, signal and drain
	if w.doneChan != nil {
		close(w.doneChan)

		// Drain path channel to unblock walker goroutine
		if w.docChan != nil {
			for range w.docChan {
			}
		}
	}

	return nil
}
