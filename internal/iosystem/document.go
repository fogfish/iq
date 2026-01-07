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

	"github.com/fogfish/iq/internal/iosystem/codec"
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

func EOF() *Document {
	return &Document{
		Type: codec.ContentEOF,
	}
}

func IsEOF(docs []*Document) bool {
	return len(docs) == 0 || (len(docs) == 1 && docs[0].Type == codec.ContentEOF)
}

// NewDocument creates a new document with the given key and reader.
// The content type defaults to application/octet-stream.
func NewDocument(key Key, contentType string, reader io.Reader) *Document {
	return &Document{
		Key:      key,
		Type:     contentType, // codec.Default.DetectContentType(string(key)),
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

// EnsureExtension updates the document key to have the correct file extension
// based on the document's content type. If the extension already matches,
// the key is unchanged.
func (d *Document) EnsureExtension() {
	if d.Key == "" {
		return
	}

	// Get the expected extension for this content type
	expectedExt := codec.Default.GetExtension(d.Type)
	if expectedExt == "" {
		return // No mapping for this content type
	}

	// Get current extension
	currentExt := filepath.Ext(string(d.Key))

	// If extension already matches, nothing to do
	if currentExt == expectedExt {
		return
	}

	// Replace or add the correct extension
	keyStr := string(d.Key)
	if currentExt != "" {
		keyStr = strings.TrimSuffix(keyStr, currentExt)
	}
	d.Key = Key(keyStr + expectedExt)
}

/*
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
*/
