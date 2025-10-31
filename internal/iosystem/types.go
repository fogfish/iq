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
// Processors are the middle stage: Source → Processor → Sink
//
// A processor can:
//   - Transform one document to one document (1:1)
//   - Split one document into many (1:N) - e.g., chunking
//   - Filter documents by returning empty slice (1:0)
//   - Combine is not directly supported (use MergedSource instead)
//
// Implementations should:
//   - Be stateless where possible
//   - Return errors for processing failures
//   - Release resources in Close()
type Processor interface {
	// Process takes an input document and produces zero or more output documents.
	// Return empty slice to filter out the document.
	// Return error for processing failures.
	Process(ctx context.Context, doc *Document) ([]*Document, error)

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
