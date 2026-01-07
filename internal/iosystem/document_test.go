//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package iosystem_test

import (
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/fogfish/it/v2"
)

func TestNewDocument(t *testing.T) {
	reader := strings.NewReader("test content")
	doc := iosystem.NewDocument("test.txt", codec.ContentText, reader)

	it.Then(t).Should(
		it.Equal(doc.Key, "test.txt"),
		it.True(doc.Reader != nil),
		it.True(doc.Metadata.Custom != nil),
	)
}

func TestDocumentWithMetadata(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", codec.ContentText, strings.NewReader("content")).
		WithMetadata("size", "1024").
		WithMetadata("type", "text/plain")

	it.Then(t).Should(
		it.Equal(doc.Metadata.Custom["size"], "1024"),
		it.Equal(doc.Metadata.Custom["type"], "text/plain"),
	)
}

func TestDocumentWithMetadata_Chaining(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", codec.ContentText, strings.NewReader("content")).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	it.Then(t).Should(
		it.Equal(len(doc.Metadata.Custom), 2),
	)
}

func TestDocumentEnsureExtension(t *testing.T) {
	tests := []struct {
		name        string
		key         iosystem.Key
		contentType string
		want        iosystem.Key
	}{
		{
			name:        "JSON document with wrong extension",
			key:         "output.txt",
			contentType: "application/json",
			want:        "output.json",
		},
		{
			name:        "JSON document with correct extension",
			key:         "output.json",
			contentType: "application/json",
			want:        "output.json",
		},
		{
			name:        "Text document without extension",
			key:         "output",
			contentType: "text/plain",
			want:        "output.txt",
		},
		{
			name:        "YAML document with yml extension",
			key:         "config.yml",
			contentType: "application/x-yaml",
			want:        "config.yaml",
		},
		{
			name:        "PNG image with correct extension",
			key:         "image.png",
			contentType: "image/png",
			want:        "image.png",
		},
		{
			name:        "Empty key",
			key:         "",
			contentType: "application/json",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &iosystem.Document{
				Key:    tt.key,
				Type:   tt.contentType,
				Reader: strings.NewReader("content"),
			}
			doc.EnsureExtension()
			it.Then(t).Should(
				it.Equal(doc.Key, tt.want),
			)
		})
	}
}
