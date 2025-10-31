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
		it.Equal(doc.Path, "test.txt"),
		it.True(doc.Reader != nil),
		it.True(doc.Metadata != nil),
	)
}

func TestDocumentWithMetadata(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", strings.NewReader("content")).
		WithMetadata("size", "1024").
		WithMetadata("type", "text/plain")

	it.Then(t).Should(
		it.Equal(doc.Metadata["size"], "1024"),
		it.Equal(doc.Metadata["type"], "text/plain"),
	)
}

func TestDocumentWithMetadata_Chaining(t *testing.T) {
	doc := iosystem.NewDocument("test.txt", strings.NewReader("content")).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	it.Then(t).Should(
		it.Equal(len(doc.Metadata), 2),
	)
}
