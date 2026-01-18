//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package codec

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONCodec(t *testing.T) {
	codec := NewJSONCodec(false)

	t.Run("ContentType", func(t *testing.T) {
		if got := codec.ContentType(); got != ContentJSON {
			t.Errorf("ContentType() = %v, want %v", got, ContentJSON)
		}
	})

	t.Run("Decode", func(t *testing.T) {
		input := `{"key": "value"}`
		result, err := codec.Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Decode() result type = %T, want map[string]any", result)
		}

		if m["key"] != "value" {
			t.Errorf("Decode() key = %v, want value", m["key"])
		}
	})

	t.Run("Encode", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		var buf bytes.Buffer
		if err := codec.Encode(&buf, data); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		if buf.Len() == 0 {
			t.Error("Encode() produced empty output")
		}
	})
}

func TestTextCodec(t *testing.T) {
	codec := NewTextCodec(ContentText, ".txt")

	t.Run("Decode", func(t *testing.T) {
		input := "Hello, World!"
		result, err := codec.Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		data, ok := result.([]byte)
		if !ok {
			t.Fatalf("Decode() result type = %T, want []byte", result)
		}

		if string(data) != input {
			t.Errorf("Decode() = %s, want %s", string(data), input)
		}
	})

	t.Run("Encode", func(t *testing.T) {
		var buf bytes.Buffer
		if err := codec.Encode(&buf, "Hello"); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		if buf.String() != "Hello" {
			t.Errorf("Encode() = %s, want Hello", buf.String())
		}
	})
}

func TestRegistry(t *testing.T) {
	t.Run("DetectContentType", func(t *testing.T) {
		r := NewRegistry()

		tests := []struct {
			path string
			want string
		}{
			{"file.json", ContentJSON},
			{"file.yaml", ContentYAML},
			{"file.txt", ContentText},
			{"file.unknown", ContentStream},
		}

		for _, tt := range tests {
			got := r.DetectContentType(tt.path)
			if got != tt.want {
				t.Errorf("DetectContentType(%s) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})

	t.Run("GetExtension", func(t *testing.T) {
		r := NewRegistry()

		tests := []struct {
			contentType string
			want        string
		}{
			{ContentJSON, ".json"},
			{ContentYAML, ".yaml"},
			{ContentText, ".txt"},
			{ContentPNG, ".png"},
			{ContentJPG, ".jpg"},
			{ContentJSONL, ".jsonl"},
			{ContentStream, ".bin"},
			{"unknown/type", ""},
		}

		for _, tt := range tests {
			got := r.GetExtension(tt.contentType)
			if got != tt.want {
				t.Errorf("GetExtension(%s) = %v, want %v", tt.contentType, got, tt.want)
			}
		}
	})

	t.Run("Encode and Decode", func(t *testing.T) {
		r := NewRegistry()

		data := map[string]any{"key": "value"}
		doc, err := r.Encode(data, ContentJSON)
		if err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		result, err := r.Decode(doc, ContentJSON)
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		m := result.(map[string]any)
		if m["key"] != "value" {
			t.Errorf("Round-trip key = %v, want value", m["key"])
		}
	})
}
