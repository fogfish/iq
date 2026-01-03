//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/blueprint/compiler"
)

func TestJSONFormatter(t *testing.T) {
	formatter := compiler.NewJSONFormatter()

	results := []any{
		map[string]any{"name": "Alice", "score": 95},
		map[string]any{"name": "Bob", "score": 87},
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Should return results as-is (Go slice)
	outputSlice, ok := output.([]any)
	if !ok {
		t.Fatalf("Expected []any, got %T", output)
	}

	if len(outputSlice) != 2 {
		t.Errorf("Expected 2 items, got %d", len(outputSlice))
	}

	// Verify JSON encoding works
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(jsonBytes)
	if !strings.Contains(jsonStr, "Alice") || !strings.Contains(jsonStr, "Bob") {
		t.Errorf("JSON output missing expected data: %s", jsonStr)
	}
}

func TestJSONFormatter_Empty(t *testing.T) {
	formatter := compiler.NewJSONFormatter()

	output, err := formatter.Format([]any{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputSlice := output.([]any)
	if len(outputSlice) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(outputSlice))
	}
}

func TestJSONLFormatter(t *testing.T) {
	formatter := compiler.NewJSONLFormatter()

	results := []any{
		map[string]any{"name": "Alice", "score": 95},
		map[string]any{"name": "Bob", "score": 87},
		map[string]any{"name": "Carol", "score": 92},
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr, ok := output.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", output)
	}

	// Should be 3 lines of JSON
	lines := strings.Split(outputStr, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}

	// Verify content
	if !strings.Contains(outputStr, "Alice") {
		t.Errorf("Missing Alice in output")
	}
	if !strings.Contains(outputStr, "Bob") {
		t.Errorf("Missing Bob in output")
	}
}

func TestJSONLFormatter_Empty(t *testing.T) {
	formatter := compiler.NewJSONLFormatter()

	output, err := formatter.Format([]any{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)
	if outputStr != "" {
		t.Errorf("Expected empty string, got %q", outputStr)
	}
}

func TestTextFormatter_Newline(t *testing.T) {
	formatter := compiler.NewTextFormatter("\n")

	results := []any{
		"First summary",
		"Second summary",
		"Third summary",
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)
	expected := "First summary\nSecond summary\nThird summary"
	if outputStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, outputStr)
	}
}

func TestTextFormatter_DoubleNewline(t *testing.T) {
	formatter := compiler.NewTextFormatter("\n\n")

	results := []any{
		"Paragraph 1",
		"Paragraph 2",
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)
	expected := "Paragraph 1\n\nParagraph 2"
	if outputStr != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, outputStr)
	}
}

func TestTextFormatter_CustomDelimiter(t *testing.T) {
	formatter := compiler.NewTextFormatter(" | ")

	results := []any{
		"Item A",
		"Item B",
		"Item C",
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)
	expected := "Item A | Item B | Item C"
	if outputStr != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, outputStr)
	}
}

func TestTextFormatter_MixedTypes(t *testing.T) {
	formatter := compiler.NewTextFormatter("\n")

	results := []any{
		"String value",
		map[string]any{"key": "object value"},
		42,
	}

	output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)

	// String should appear as-is
	if !strings.Contains(outputStr, "String value") {
		t.Errorf("Missing string value in output")
	}

	// Object should be JSON-encoded
	if !strings.Contains(outputStr, `{"key":"object value"}`) {
		t.Errorf("Object not JSON-encoded properly")
	}

	// Number should be JSON-encoded
	if !strings.Contains(outputStr, "42") {
		t.Errorf("Number not encoded properly")
	}
}

func TestTextFormatter_Empty(t *testing.T) {
	formatter := compiler.NewTextFormatter("\n")

	output, err := formatter.Format([]any{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	outputStr := output.(string)
	if outputStr != "" {
		t.Errorf("Expected empty string, got %q", outputStr)
	}
}

func TestNewFormatter_DefaultJSON(t *testing.T) {
	formatter, err := compiler.NewFormatter(nil)
	if err != nil {
		t.Fatalf("NewFormatter failed: %v", err)
	}

	_, ok := formatter.(*compiler.JSONFormatter)
	if !ok {
		t.Errorf("Expected JSONFormatter by default, got %T", formatter)
	}
}

func TestNewFormatter_ExplicitTypes(t *testing.T) {
	tests := []struct {
		name         string
		format       *ast.FormatNode
		expectedType interface{}
	}{
		{
			name:         "JSON",
			format:       &ast.FormatNode{Type: "json", Delimiter: "\n"},
			expectedType: &compiler.JSONFormatter{},
		},
		{
			name:         "JSONL",
			format:       &ast.FormatNode{Type: "jsonl", Delimiter: "\n"},
			expectedType: &compiler.JSONLFormatter{},
		},
		{
			name:         "Text",
			format:       &ast.FormatNode{Type: "text", Delimiter: "\n\n"},
			expectedType: &compiler.TextFormatter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := compiler.NewFormatter(tt.format)
			if err != nil {
				t.Fatalf("NewFormatter failed: %v", err)
			}

			expectedTypeName := fmt.Sprintf("%T", tt.expectedType)
			actualTypeName := fmt.Sprintf("%T", formatter)
			if expectedTypeName != actualTypeName {
				t.Errorf("Expected %s, got %s", expectedTypeName, actualTypeName)
			}
		})
	}
}

func TestNewFormatter_InvalidType(t *testing.T) {
	format := &ast.FormatNode{Type: "xml", Delimiter: "\n"}

	_, err := compiler.NewFormatter(format)
	if err == nil {
		t.Fatal("Expected error for invalid format type")
	}

	if !strings.Contains(err.Error(), "unknown format type") {
		t.Errorf("Expected 'unknown format type' error, got: %v", err)
	}
}
