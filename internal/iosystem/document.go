//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package iosystem

import (
	"io"
	"path/filepath"
	"strings"
)

// Content type constants
const (
	ContentStream = "application/octet-stream"
	ContentJSON   = "application/json"
	ContentYAML   = "application/x-yaml"
	ContentPNG    = "image/png"
	ContentJPG    = "image/jpeg"
	ContentText   = "text/plain"
	ContentEOF    = "application/x-eof" // Signals end of stream
)

// Metadata holds document attributes (NOT part of key identity).
type Metadata struct {
	ContentType string            // MIME type (e.g., "text/plain", "application/json")
	Extension   string            // File extension for output (e.g., ".txt", ".md")
	Size        int64             // Document size in bytes
	Custom      map[string]string // User-defined attributes
}

// Document represents a single input document with metadata.
// Documents flow through the pipeline from Source → Processor → Sink.
type Document struct {
	// Path is the logical identifier for this document (e.g., "stdin", "file.txt", "dir/file.txt")
	// DEPRECATED: Use Key instead. Kept for backward compatibility.
	Path string

	// Key is the simple path-based identity (e.g., "sub/a.txt")
	Key Key

	// Type specifies the content type of the document
	// (e.g., "application/octet-stream", "application/json")
	Type string

	// Reader provides streaming access to document content
	Reader io.Reader

	// Metadata contains additional information about the document
	Metadata Metadata
}

// NewDocument creates a new document with the given key and reader.
// The content type defaults to application/octet-stream.
func NewDocument(key Key, reader io.Reader) *Document {
	return &Document{
		Key:      key,
		Path:     string(key), // Backward compatibility
		Type:     ContentStream,
		Reader:   reader,
		Metadata: Metadata{Custom: make(map[string]string)},
	}
}

// WithMetadata adds metadata to the document and returns it for chaining.
func (d *Document) WithMetadata(key, value string) *Document {
	if d.Metadata.Custom == nil {
		d.Metadata.Custom = make(map[string]string)
	}
	d.Metadata.Custom[key] = value
	return d
}

func (d *Document) FilePath() string {
	keyStr := string(d.Key)
	if keyStr == "" {
		keyStr = d.Path // Fallback for backward compatibility
	}

	// Use explicit extension if provided
	if d.Metadata.Extension != "" {
		// Remove existing extension and add metadata extension
		ext := filepath.Ext(keyStr)
		base := strings.TrimSuffix(keyStr, ext)
		return base + d.Metadata.Extension
	}

	// Derive extension from content type
	ext := filepath.Ext(keyStr)
	base := strings.TrimSuffix(keyStr, ext)
	switch d.Type {
	case ContentJSON:
		return base + ".json"
	case ContentYAML:
		return base + ".yaml"
	case ContentPNG:
		return base + ".png"
	case ContentJPG:
		return base + ".jpg"
	default:
		return keyStr
	}
}
