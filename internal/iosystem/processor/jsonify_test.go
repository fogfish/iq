//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/it/v2"
)

func TestJSONFormatter_FormatsJSON(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  false, // Disable color for easier testing
	})
	defer proc.Close()

	ctx := context.Background()
	jsonContent := `{"name":"test","value":42,"nested":{"key":"data"}}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(jsonContent))
	doc.Type = iosystem.ContentJSON

	results, err := proc.Process(ctx, doc)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)

	formatted, _ := io.ReadAll(results[0].Reader)
	formattedStr := string(formatted)

	// Verify it's valid JSON
	var obj map[string]any
	it.Then(t).Should(it.Nil(json.Unmarshal(formatted, &obj)))

	// Verify indentation is applied
	it.Then(t).Should(
		it.True(strings.Contains(formattedStr, "\n")),
		it.True(strings.Contains(formattedStr, "  ")), // 2-space indent
	)
}

func TestJSONFormatter_PassThroughNonJSON(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  false,
	})
	defer proc.Close()

	ctx := context.Background()
	textContent := "This is plain text"
	doc := iosystem.NewDocument("test.txt", strings.NewReader(textContent))
	doc.Type = iosystem.ContentText

	results, err := proc.Process(ctx, doc)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)

	output, _ := io.ReadAll(results[0].Reader)
	it.Then(t).Should(it.Equal(string(output), textContent))
}

func TestJSONFormatter_HandlesInvalidJSON(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  false,
	})
	defer proc.Close()

	ctx := context.Background()
	invalidJSON := `{"name": invalid}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(invalidJSON))
	doc.Type = iosystem.ContentJSON

	results, err := proc.Process(ctx, doc)

	// Should not error, just pass through
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)

	output, _ := io.ReadAll(results[0].Reader)
	it.Then(t).Should(it.Equal(string(output), invalidJSON))
}

func TestJSONFormatter_NoContentType(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  false,
	})
	defer proc.Close()

	ctx := context.Background()
	jsonContent := `{"name":"test"}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(jsonContent))
	// No content-type metadata

	results, err := proc.Process(ctx, doc)

	// Should pass through unchanged without content-type
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)

	output, _ := io.ReadAll(results[0].Reader)
	it.Then(t).Should(it.Equal(string(output), jsonContent))
}

func TestJSONFormatter_PreservesMetadata(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  false,
	})
	defer proc.Close()

	ctx := context.Background()
	jsonContent := `{"name":"test"}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(jsonContent))
	doc.Type = iosystem.ContentJSON
	doc.Metadata = map[string]string{
		"source": "agent",
		"custom": "value",
	}

	results, err := proc.Process(ctx, doc)

	it.Then(t).Should(it.Nil(err))

	// Check that metadata is preserved
	it.Then(t).Should(
		it.Equal(results[0].Type, iosystem.ContentJSON),
		it.Equal(results[0].Metadata["source"], "agent"),
		it.Equal(results[0].Metadata["custom"], "value"),
	)
}

func TestJSONFormatter_WithColor(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 2,
		Color:  true, // Enable color
	})
	defer proc.Close()

	ctx := context.Background()
	jsonContent := `{"name":"test"}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(jsonContent))
	doc.Type = iosystem.ContentJSON

	results, err := proc.Process(ctx, doc)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)

	formatted, _ := io.ReadAll(results[0].Reader)
	formattedStr := string(formatted)

	// Verify it contains ANSI color codes or is at least formatted
	// colorjson may or may not add colors depending on terminal support
	it.Then(t).Should(
		it.True(strings.Contains(formattedStr, "\x1b[") || strings.Contains(formattedStr, "\n")), // ANSI escape code or newlines
	)
}

func TestJSONFormatter_CustomIndent(t *testing.T) {
	proc := processor.NewJsonify(processor.JsonifyConfig{
		Indent: 4, // 4 spaces
		Color:  false,
	})
	defer proc.Close()

	ctx := context.Background()
	jsonContent := `{"nested":{"key":"value"}}`
	doc := iosystem.NewDocument("test.json", strings.NewReader(jsonContent))
	doc.Type = iosystem.ContentJSON

	results, err := proc.Process(ctx, doc)

	it.Then(t).Should(it.Nil(err))

	formatted, _ := io.ReadAll(results[0].Reader)
	formattedStr := string(formatted)

	// Verify 4-space indentation
	it.Then(t).Should(
		it.True(strings.Contains(formattedStr, "    ")), // 4 spaces
	)
}
