package iosystem

import (
	"io"
)

// Content type constants
const (
	ContentStream = "application/octet-stream"
	ContentJSON   = "application/json"
	ContentText   = "text/plain"
)

// Document represents a single input document with metadata.
// Documents flow through the pipeline from Source → Processor → Sink.
type Document struct {
	// Path is the logical identifier for this document (e.g., "stdin", "file.txt", "dir/file.txt")
	Path string

	// Type specifies the content type of the document
	// (e.g., "application/octet-stream", "application/json")
	Type string

	// Reader provides streaming access to document content
	Reader io.Reader

	// Metadata contains additional information about the document
	// (e.g., content-type, size, timestamp, custom attributes)
	Metadata map[string]string
}

// NewDocument creates a new document with the given path and reader.
// The content type defaults to application/octet-stream.
func NewDocument(path string, reader io.Reader) *Document {
	return &Document{
		Path:     path,
		Type:     ContentStream,
		Reader:   reader,
		Metadata: make(map[string]string),
	}
}

// WithMetadata adds metadata to the document and returns it for chaining.
func (d *Document) WithMetadata(key, value string) *Document {
	if d.Metadata == nil {
		d.Metadata = make(map[string]string)
	}
	d.Metadata[key] = value
	return d
}
