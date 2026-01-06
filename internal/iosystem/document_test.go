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
	"github.com/fogfish/it/v2"
)

func TestNewDocument(t *testing.T) {
	reader := strings.NewReader("test content")
	doc := iosystem.NewDocument("test.txt", reader)

	it.Then(t).Should(
		it.Equal(doc.Key, "test.txt"),
		it.True(doc.Reader != nil),
		it.True(doc.Metadata.Custom != nil),
	)
}

func TestDocumentWithMetadata(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", strings.NewReader("content")).
		WithMetadata("size", "1024").
		WithMetadata("type", "text/plain")

	it.Then(t).Should(
		it.Equal(doc.Metadata.Custom["size"], "1024"),
		it.Equal(doc.Metadata.Custom["type"], "text/plain"),
	)
}

func TestDocumentWithMetadata_Chaining(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", strings.NewReader("content")).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	it.Then(t).Should(
		it.Equal(len(doc.Metadata.Custom), 2),
	)
}
