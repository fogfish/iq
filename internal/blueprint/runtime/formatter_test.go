//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/ast"
)

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()

	results := List{
		map[string]any{"name": "Alice", "score": 95},
		map[string]any{"name": "Bob", "score": 87},
	}

	contentType, output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Should return application/json content type
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}

	// Should return results as-is (List)
	outputList, ok := output.(List)
	if !ok {
		t.Fatalf("Expected List, got %T", output)
	}

	if len(outputList) != 2 {
		t.Errorf("Expected 2 items, got %d", len(outputList))
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
	formatter := NewJSONFormatter()

	contentType, output, err := formatter.Format(List{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}

	outputList := output.(List)
	if len(outputList) != 0 {
		t.Errorf("Expected empty list, got %d items", len(outputList))
	}
}

func TestTextFormatter_Newline(t *testing.T) {
	formatter := NewTextFormatter("\n")

	results := List{
		"First summary",
		"Second summary",
		"Third summary",
	}

	contentType, output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}

	outputText, ok := output.(Text)
	if !ok {
		t.Fatalf("Expected Text, got %T", output)
	}

	expected := "First summary\nSecond summary\nThird summary"
	if string(outputText) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, outputText)
	}
}

func TestTextFormatter_DoubleNewline(t *testing.T) {
	formatter := NewTextFormatter("\n\n")

	results := List{
		"Paragraph 1",
		"Paragraph 2",
	}

	contentType, output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}

	outputText := output.(Text)
	expected := "Paragraph 1\n\nParagraph 2"
	if string(outputText) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, outputText)
	}
}

func TestTextFormatter_CustomDelimiter(t *testing.T) {
	formatter := NewTextFormatter(" | ")

	results := List{
		"Item A",
		"Item B",
		"Item C",
	}

	contentType, output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}

	outputText := output.(Text)
	expected := "Item A | Item B | Item C"
	if string(outputText) != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, outputText)
	}
}

func TestTextFormatter_MixedTypes(t *testing.T) {
	formatter := NewTextFormatter("\n")

	results := List{
		"String value",
		map[string]any{"key": "object value"},
		42,
	}

	contentType, output, err := formatter.Format(results)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}

	outputText := string(output.(Text))

	// String should appear as-is
	if !strings.Contains(outputText, "String value") {
		t.Errorf("Missing string value in output")
	}

	// Object should be JSON-encoded
	if !strings.Contains(outputText, `{"key":"object value"}`) {
		t.Errorf("Object not JSON-encoded properly")
	}

	// Number should be JSON-encoded
	if !strings.Contains(outputText, "42") {
		t.Errorf("Number not encoded properly")
	}
}

func TestTextFormatter_Empty(t *testing.T) {
	formatter := NewTextFormatter("\n")

	contentType, output, err := formatter.Format(List{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}

	outputText := output.(Text)
	if string(outputText) != "" {
		t.Errorf("Expected empty string, got %q", outputText)
	}
}

func TestNewFormatter_DefaultJSON(t *testing.T) {
	formatter, err := NewFormatter(nil)
	if err != nil {
		t.Fatalf("NewFormatter failed: %v", err)
	}

	_, ok := formatter.(*JSONFormatter)
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
			format:       &ast.FormatNode{Type: "json", Divider: "\n"},
			expectedType: &JSONFormatter{},
		},
		{
			name:         "Text",
			format:       &ast.FormatNode{Type: "text", Divider: "\n\n"},
			expectedType: &TextFormatter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := NewFormatter(tt.format)
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
	format := &ast.FormatNode{Type: "xml", Divider: "\n"}

	_, err := NewFormatter(format)
	if err == nil {
		t.Fatal("Expected error for invalid format type")
	}

	if !strings.Contains(err.Error(), "unknown format type") {
		t.Errorf("Expected 'unknown format type' error, got: %v", err)
	}
}
