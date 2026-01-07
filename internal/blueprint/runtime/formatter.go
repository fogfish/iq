//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/iosystem/codec"
)

// Formatter serializes results into output format.
//
// Use cases:
//   - JSON: Structured arrays for downstream processing
//   - JSONL: Streaming-friendly newline-delimited JSON
//   - Text: Human-readable concatenation with custom delimiters
type Formatter interface {
	// Format converts Gist into serialized output.
	// Input: results from agent as Gist
	// Output: serialized data represented as Gist and its content type
	Format(Gist) (string, Gist, error)
}

//------------------------------------------------------------------------------

// JSONFormatter returns results as JSON array (default).
// Output: Go slice ([]any) that will be JSON-encoded downstream.
type JSONFormatter struct{}

// NewJSONFormatter creates a JSON array formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format returns results as-is (already a Go array).
// Downstream JSON encoding will serialize it as [...].
func (f *JSONFormatter) Format(results Gist) (string, Gist, error) {
	return results.ContentType(), results, nil
}

//------------------------------------------------------------------------------

// TextFormatter concatenates results with configurable delimiter.
// Useful for human-readable summaries or simple text aggregation.
type TextFormatter struct {
	Delimiter string
}

// NewTextFormatter creates a text formatter with custom delimiter
func NewTextFormatter(delimiter string) *TextFormatter {
	return &TextFormatter{
		Delimiter: delimiter,
	}
}

// Format converts results to delimited text string.
// Non-string results are JSON-encoded first.
func (f *TextFormatter) Format(results Gist) (string, Gist, error) {
	switch v := results.(type) {
	case Text:
		return results.ContentType(), v, nil
	case Json:
		return f.fromJson(v)
	case List:
		return f.fromList(v)
	default:
		return "", nil, fmt.Errorf("text formatter unsupported Gist type: %T", v)
	}
}

func (f *TextFormatter) fromList(results List) (string, Gist, error) {
	var buf bytes.Buffer

	for i, result := range results {
		if i > 0 {
			buf.WriteString(f.Delimiter)
		}

		switch v := result.(type) {
		case Text:
			buf.WriteString(string(v))
		case string:
			buf.WriteString(v)
		case []byte:
			buf.Write(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return "", nil, fmt.Errorf("failed to marshal result %d: %w", i, err)
			}
			buf.Write(data)
		}
	}

	return codec.ContentText, Text(buf.String()), nil
}

func (f *TextFormatter) fromJson(results Json) (string, Gist, error) {
	data, err := json.Marshal(results)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return codec.ContentText, Text(data), nil
}

//------------------------------------------------------------------------------

// NewFormatter creates formatter based on AST format node.
// Returns appropriate formatter or error for unknown types.
func NewFormatter(format *ast.FormatNode) (Formatter, error) {
	if format == nil {
		return NewJSONFormatter(), nil
	}

	switch format.Type {
	case "json":
		return NewJSONFormatter(), nil
	case "text":
		return NewTextFormatter(format.Divider), nil
	default:
		return nil, fmt.Errorf("unknown format type: %s (must be json or text)", format.Type)
	}
}
