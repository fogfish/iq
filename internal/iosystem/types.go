//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package iosystem

import "context"

// Source produces documents from an input source.
// Sources are the entry point of a processing pipeline.
//
// Implementations should:
//   - Return documents one at a time via Next()
//   - Return io.EOF when no more documents are available
//   - Be safe for single-goroutine use (not necessarily concurrent)
//   - Release resources in Close()
type Source interface {
	// Next returns the next document or io.EOF when complete.
	// Other errors indicate retrieval failures.
	Next(ctx context.Context) (*Document, error)

	// Close releases any resources held by the source.
	Close() error
}

// Processor transforms documents in a pipeline.
// Processors are monadic: accept array of documents, return array of documents.
//
// A processor can:
//   - Transform documents (map)
//   - Split documents (flatMap - one doc becomes many)
//   - Filter documents (return empty slice)
//   - Collect/aggregate documents (ArrayCollector pattern)
//
// Implementations should:
//   - Be stateless where possible
//   - Return errors for processing failures
//   - Release resources in Close()
type Processor interface {
	// Process takes input documents and produces zero or more output documents.
	//   - Single doc processing: len(docs)==1 (normal case)
	//   - Array processing: len(docs)>1 (array mode after collection)
	//   - EOF signal: docs[0].Type == ContentEOF (end of stream)
	// Return empty slice to filter out documents.
	// Return error for processing failures.
	Process(ctx context.Context, docs []*Document) ([]*Document, error)

	// Close releases any resources held by the processor.
	Close() error
}

// Sink consumes processed documents.
// Sinks are the exit point of a processing pipeline.
//
// Implementations should:
//   - Write documents to their destination
//   - Handle errors gracefully
//   - Buffer writes if beneficial for performance
//   - Flush buffers and release resources in Close()
type Sink interface {
	// Write stores a document to the sink's destination.
	// Returns error if the write fails.
	Write(ctx context.Context, doc *Document) error

	// Close finalizes output and releases resources.
	// Should flush any buffered writes.
	Close() error
}

// Key is a simple filesystem-style path used as document identity.
// Keys are relative paths using forward slashes as separators.
//
// Examples:
//
//	"a.txt"           - file in root
//	"sub/a.txt"       - file in subdirectory
//	"summary/a.txt"   - file with emit prefix
type Key string
