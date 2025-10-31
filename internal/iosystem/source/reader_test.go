package source_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/it/v2"
)

func TestReaderSource_SingleDocument(t *testing.T) {
	content := "test content"
	reader := strings.NewReader(content)

	src := source.NewReader("test.txt", reader)
	ctx := context.Background()

	// First call should return document
	doc, err := src.Next(ctx)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(doc.Path, "test.txt"),
	)
	it.Then(t).ShouldNot(
		it.Nil(doc),
	)

	// Read content
	gotContent, err := io.ReadAll(doc.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(gotContent), content),
	)

	// Second call should return EOF
	doc2, err := src.Next(ctx)
	it.Then(t).Should(
		it.Equal(err, io.EOF),
	)

	// doc2 should be nil
	if doc2 != nil {
		t.Errorf("expected nil document, got %v", doc2)
	}
}

func TestReaderSource_Close(t *testing.T) {
	reader := strings.NewReader("content")
	src := source.NewReader("test.txt", reader)

	// Close should not return error
	err := src.Close()
	it.Then(t).Should(it.Nil(err))

	// Should still be able to read after close (reader lifecycle managed by caller)
	ctx := context.Background()
	doc, err := src.Next(ctx)
	it.Then(t).Should(
		it.Nil(err),
	)
	it.Then(t).ShouldNot(
		it.Nil(doc),
	)
}

func TestReaderSource_EmptyContent(t *testing.T) {
	reader := strings.NewReader("")
	src := source.NewReader("empty.txt", reader)

	ctx := context.Background()
	doc, err := src.Next(ctx)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(doc.Path, "empty.txt"),
	)
	it.Then(t).ShouldNot(
		it.Nil(doc),
	)

	// Content should be empty
	content, _ := io.ReadAll(doc.Reader)
	it.Then(t).Should(it.Equal(len(content), 0))
}

func TestReaderSource_WithPipeline(t *testing.T) {
	// Test that ReaderSource integrates properly with Pipeline
	// This is the primary use case for spool integration
	reader := strings.NewReader("pipeline test")
	src := source.NewReader("pipeline.txt", reader)

	ctx := context.Background()
	doc, err := src.Next(ctx)

	it.Then(t).Should(
		it.Nil(err),
	)
	it.Then(t).ShouldNot(
		it.Nil(doc),
	)

	// Verify document can be read multiple times if needed (io.TeeReader pattern)
	content, _ := io.ReadAll(doc.Reader)
	it.Then(t).Should(it.Equal(string(content), "pipeline test"))
}
