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
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

// Storage provides pure key/value operations for document persistence.
// Implementations use github.com/fogfish/stream for filesystem and S3.
type Storage interface {
	// Put writes value to key (overwrites if exists).
	// Key should be a relative path like "sub/file.txt".
	Put(ctx context.Context, key iosystem.Key, value io.Reader) error

	// Get reads value from key (returns auto-closing reader).
	// Returns error if key does not exist.
	Get(ctx context.Context, key iosystem.Key) (io.ReadCloser, error)

	// Has checks if key exists (cheap operation for skip-if-exists logic).
	Has(ctx context.Context, key iosystem.Key) (bool, error)

	// Walk traverses all keys matching prefix pattern.
	// Visitor function is called for each matching document.
	Walk(ctx context.Context, prefix iosystem.Key, visitor func(*iosystem.Document) error) error
}
