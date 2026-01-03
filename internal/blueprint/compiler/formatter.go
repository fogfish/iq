//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
)

// Formatter serializes foreach results into output format.
// This is a CODEC for result serialization, NOT a reduce function.
//
// Use cases:
//   - JSON: Structured arrays for downstream processing
//   - JSONL: Streaming-friendly newline-delimited JSON
//   - Text: Human-readable concatenation with custom delimiters
type Formatter interface {
	// Format converts array of results into serialized output.
	// Input: results from foreach iterations ([]any)
	// Output: serialized data (any) - typically string or []any
	Format(results []any) (any, error)
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
func (f *JSONFormatter) Format(results []any) (any, error) {
	return results, nil // Already array, JSON-encoded downstream
}

//------------------------------------------------------------------------------

// JSONLFormatter returns results as newline-delimited JSON.
// Each result on separate line, suitable for streaming.
type JSONLFormatter struct{}

// NewJSONLFormatter creates a JSONL formatter
func NewJSONLFormatter() *JSONLFormatter {
	return &JSONLFormatter{}
}

// Format converts results to JSONL string.
// Each result encoded as JSON on separate line.
func (f *JSONLFormatter) Format(results []any) (any, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	for i, result := range results {
		if err := encoder.Encode(result); err != nil {
			return nil, fmt.Errorf("failed to encode result %d: %w", i, err)
		}
	}

	// Remove trailing newline (Encode adds one)
	output := buf.String()
	if len(output) > 0 && output[len(output)-1] == '\n' {
		output = output[:len(output)-1]
	}

	return output, nil
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
func (f *TextFormatter) Format(results []any) (any, error) {
	var buf bytes.Buffer

	for i, result := range results {
		if i > 0 {
			buf.WriteString(f.Delimiter)
		}

		// Convert result to string
		switch v := result.(type) {
		case string:
			buf.WriteString(v)
		case []byte:
			buf.Write(v)
		default:
			// Marshal to JSON if not string
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result %d: %w", i, err)
			}
			buf.Write(data)
		}
	}

	return buf.String(), nil
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
	case "jsonl":
		return NewJSONLFormatter(), nil
	case "text":
		return NewTextFormatter(format.Delimiter), nil
	default:
		return nil, fmt.Errorf("unknown format type: %s (must be json, jsonl, or text)", format.Type)
	}
}
