//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TylerBrock/colorjson"
	"github.com/fogfish/iq/internal/iosystem"
)

// Jsonify is a processor that formats JSON documents with color and indentation.
// It only processes documents with content-type "application/json" in metadata.
// Other documents are passed through unchanged.
//
// The processor:
//   - Checks document metadata for "content-type"
//   - Formats JSON with color syntax highlighting
//   - Pretty-prints with configurable indentation
//   - Passes through non-JSON documents unchanged
//
// Example:
//
//	proc := NewJSONFormatter(JSONFormatConfig{
//	    Indent: 2,
//	    Color:  true,
//	})
//	docs, err := proc.Process(ctx, inputDoc)
type Jsonify struct {
	config JsonifyConfig
}

// JsonifyConfig configures the JSON formatter processor.
type JsonifyConfig struct {
	Indent int  // Number of spaces for indentation (default: 2)
	Color  bool // Enable color syntax highlighting (default: true)
}

// NewJsonify creates a processor that formats JSON documents.
func NewJsonify(config JsonifyConfig) *Jsonify {
	// Set defaults
	if config.Indent == 0 {
		config.Indent = 2
	}

	return &Jsonify{
		config: config,
	}
}

// Process formats a JSON document with color and indentation.
// Only processes documents with content-type "application/json".
// Other documents are passed through unchanged.
func (p *Jsonify) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Check if document is JSON
	contentType := doc.Type
	if contentType != iosystem.ContentJSON {
		return []*iosystem.Document{doc}, nil
	}

	// Read document content
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		// On read error, pass through unchanged
		doc.Reader = bytes.NewReader(content)
		return []*iosystem.Document{doc}, nil
	}

	// Parse JSON to validate and prepare for formatting
	var obj any
	if err := json.Unmarshal(content, &obj); err != nil {
		// Invalid JSON, pass through unchanged
		doc.Reader = bytes.NewReader(content)
		return []*iosystem.Document{doc}, nil
	}

	// Format JSON with color
	var formatted []byte
	if p.config.Color {
		f := colorjson.NewFormatter()
		f.Indent = p.config.Indent
		formatted, err = f.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to format JSON for '%s': %w", doc.Path, err)
		}
	} else {
		// Format without color (standard pretty-print)
		formatted, err = json.MarshalIndent(obj, "", bytesIndent(p.config.Indent))
		if err != nil {
			return nil, fmt.Errorf("failed to format JSON for '%s': %w", doc.Path, err)
		}
	}

	// Create output document
	out := &iosystem.Document{
		Type:     doc.Type,
		Path:     doc.Path,
		Reader:   bytes.NewReader(formatted),
		Metadata: copyMetadata(doc.Metadata),
	}

	return []*iosystem.Document{out}, nil
}

// Close releases resources. For JSONFormatter, this is a no-op.
func (p *Jsonify) Close() error {
	return nil
}

// bytesIndent creates an indentation string with n spaces.
func bytesIndent(n int) string {
	indent := make([]byte, n)
	for i := range indent {
		indent[i] = ' '
	}
	return string(indent)
}
