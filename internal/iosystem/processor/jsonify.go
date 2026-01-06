//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"context"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
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
	codec  codec.Codec
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
		codec:  codec.NewPrettyJSONCodec(config.Color, config.Indent),
	}
}

// Process formats a JSON document with color and indentation.
// Only processes documents with content-type "application/json".
// Other documents are passed through unchanged.
func (p *Jsonify) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Passthrough EOF or empty
	if iosystem.IsEOF(docs) {
		return docs, nil
	}

	results := make([]*iosystem.Document, 0, len(docs))

	/* TODO: fix
	for _, doc := range docs {
		if doc == nil {
			return nil, fmt.Errorf("document is nil")
		}

		// Check if document is JSON
		if doc.Type != iosystem.ContentJSON {
			results = append(results, doc)
			continue
		}

		// Decode document
		data, err := codec.Default.Decode(doc)
		if err != nil {
			// On decode error, pass through unchanged
			results = append(results, doc)
			continue
		}

		// Encode back with formatting using the configured codec
		var buf bytes.Buffer
		if err := p.codec.Encode(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to format JSON for '%s': %w", doc.Path, err)
		}

		formatted := &iosystem.Document{
			Key:      doc.Key,
			Path:     doc.Path,
			Type:     iosystem.ContentJSON,
			Reader:   bytes.NewReader(buf.Bytes()),
			Metadata: copyMetadata(doc.Metadata),
		}

		results = append(results, formatted)
	}
	*/

	return results, nil
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
