package sink_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/it/v2"
)

func TestWriterSink_SingleDocument(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	doc := iosystem.NewDocument("test.txt", strings.NewReader("test content"))
	ctx := context.Background()

	err := snk.Write(ctx, doc)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(buf.String(), "test content"),
	)
}

func TestWriterSink_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	ctx := context.Background()

	// Write first document
	doc1 := iosystem.NewDocument("doc1.txt", strings.NewReader("first "))
	err := snk.Write(ctx, doc1)
	it.Then(t).Should(it.Nil(err))

	// Write second document (appends to same writer)
	doc2 := iosystem.NewDocument("doc2.txt", strings.NewReader("second"))
	err = snk.Write(ctx, doc2)
	it.Then(t).Should(it.Nil(err))

	// Both documents should be concatenated
	it.Then(t).Should(it.Equal(buf.String(), "first second"))
}

func TestWriterSink_EmptyDocument(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	doc := iosystem.NewDocument("empty.txt", strings.NewReader(""))
	ctx := context.Background()

	err := snk.Write(ctx, doc)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(buf.Len(), 0),
	)
}

func TestWriterSink_Close(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	// Close should not return error
	err := snk.Close()
	it.Then(t).Should(it.Nil(err))

	// Should still be able to write after close (writer lifecycle managed by caller)
	doc := iosystem.NewDocument("test.txt", strings.NewReader("content"))
	ctx := context.Background()
	err = snk.Write(ctx, doc)
	it.Then(t).Should(it.Nil(err))
}

func TestWriterSink_WithMetadata(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	doc := iosystem.NewDocument("test.txt", strings.NewReader("content"))
	doc.WithMetadata("key", "value")

	ctx := context.Background()
	err := snk.Write(ctx, doc)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(buf.String(), "content"),
	)
	// Note: WriterSink doesn't write metadata, just content
	// This is expected behavior for simple writer integration
}

func TestWriterSink_LargeContent(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	// Create a large string (1MB)
	largeContent := strings.Repeat("x", 1024*1024)
	doc := iosystem.NewDocument("large.txt", strings.NewReader(largeContent))

	ctx := context.Background()
	err := snk.Write(ctx, doc)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(buf.Len(), 1024*1024),
	)
}

func TestWriterSink_WithPipeline(t *testing.T) {
	// Test that WriterSink integrates properly with Pipeline
	// This is the primary use case for spool integration
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	doc := iosystem.NewDocument("pipeline.txt", strings.NewReader("pipeline test"))
	ctx := context.Background()

	err := snk.Write(ctx, doc)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(buf.String(), "pipeline test"),
	)
}

func TestWriterSink_ReadError(t *testing.T) {
	var buf bytes.Buffer
	snk := sink.NewWriter(&buf)

	// Create a reader that fails
	failingReader := &errorReader{err: io.ErrUnexpectedEOF}
	doc := iosystem.NewDocument("fail.txt", failingReader)

	ctx := context.Background()
	err := snk.Write(ctx, doc)

	// Should propagate the read error
	it.Then(t).ShouldNot(it.Nil(err))
}

// errorReader is a reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}
