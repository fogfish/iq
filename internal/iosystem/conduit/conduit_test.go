//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package conduit_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/it/v2"
)

func TestPipeline_Simple(t *testing.T) {
	src := newMockSource("test content")
	snk := newMockSink()

	pipeline := conduit.New(nil)

	ctx := context.Background()
	stats, err := pipeline.Run(ctx, src, snk)

	content, _ := io.ReadAll(snk.docs[0].Reader)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(snk.docs), 1),
		it.Equal(string(content), "test content"),
		it.Equal(stats.DocsProcessed, 1),
	)
}

func TestPipeline_WithIdentityProcessor(t *testing.T) {
	src := newMockSource("content 1", "content 2", "content 3")
	snk := newMockSink()
	proc := processor.NewIdentity()

	pipeline := conduit.New(nil).AddProcessor(proc)

	ctx := context.Background()
	stats, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(snk.docs), 3),
		it.Equal(stats.DocsProcessed, 3),
	)
}

func TestPipeline_WithChunking(t *testing.T) {
	src := newMockSource("First sentence. Second sentence. Third sentence.")
	snk := newMockSink()
	chunker := processor.NewChunker(
		processor.ChunkConfig{
			Strategy: processor.ChunkerSentence,
		},
	)

	pipeline := conduit.New(nil).AddProcessor(chunker)

	ctx := context.Background()
	_, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(snk.docs), 3),
	)

	// Verify each chunk has proper metadata
	for _, doc := range snk.docs {
		it.Then(t).Should(
			it.True(strings.Contains(string(doc.Key), "#chunk")),
			it.Equal(doc.Metadata.Custom["original_path"], "mock.txt"),
		)
	}
}

func TestPipeline_MultipleProcessors(t *testing.T) {
	src := newMockSource("Sentence one. Sentence two.")
	snk := newMockSink()

	chunker := processor.NewChunker(
		processor.ChunkConfig{
			Strategy: processor.ChunkerSentence,
		},
	)
	identity := processor.NewIdentity()

	pipeline := conduit.New(nil).
		AddProcessor(chunker).
		AddProcessor(identity)

	ctx := context.Background()
	_, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(snk.docs), 2),
	)
}

func TestPipeline_ErrorMode_FailFast(t *testing.T) {
	// This would need a failing source/processor to test properly
	// Skipping detailed implementation for Phase 1
}

func TestPipeline_ErrorMode_SkipError(t *testing.T) {
	// This would need a failing source/processor to test properly
	// Skipping detailed implementation for Phase 1
}

func TestPipeline_ProgressCallback(t *testing.T) {
	src := newMockSource("doc1", "doc2", "doc3")
	snk := newMockSink()

	progressCount := 0
	config := &conduit.Config{
		Progress: func(doc *iosystem.Document, err error) {
			progressCount++
			it.Then(t).Should(it.Nil(err))
		},
	}
	pipeline := conduit.New(config)

	ctx := context.Background()
	_, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(progressCount, 3),
	)
}

func TestPipeline_MetricsCallback(t *testing.T) {
	src := newMockSource("doc1", "doc2")
	snk := newMockSink()

	metricsCount := 0
	config := &conduit.Config{
		Metrics: func(stats conduit.Stats) {
			metricsCount++
			it.Then(t).Should(it.Equal(stats.DocsProcessed, 2))
		},
	}
	pipeline := conduit.New(config)

	ctx := context.Background()
	_, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(metricsCount, 1),
	)
}

func TestPipeline_Stats(t *testing.T) {
	src := newMockSource("doc1", "doc2", "doc3")
	snk := newMockSink()

	pipeline := conduit.New(nil)

	ctx := context.Background()
	stats, err := pipeline.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(stats.DocsProcessed, 3),
		it.Equal(stats.DocsSkipped, 0),
	)
}

//------------------------------------------------------------------------------

// mockSource is a simple source for testing
type mockSource struct {
	docs  []*iosystem.Document
	index int
}

func newMockSource(contents ...string) *mockSource {
	docs := make([]*iosystem.Document, len(contents))
	for i, content := range contents {
		docs[i] = iosystem.NewDocument("mock.txt", codec.ContentText, strings.NewReader(content))
	}
	return &mockSource{docs: docs}
}

func (m *mockSource) Next(ctx context.Context) (*iosystem.Document, error) {
	if m.index >= len(m.docs) {
		return nil, io.EOF
	}
	doc := m.docs[m.index]
	m.index++
	return doc, nil
}

func (m *mockSource) Close() error {
	return nil
}

// mockSink captures written documents for testing
type mockSink struct {
	docs []*iosystem.Document
}

func newMockSink() *mockSink {
	return &mockSink{docs: make([]*iosystem.Document, 0)}
}

func (m *mockSink) Write(ctx context.Context, doc *iosystem.Document) error {
	// Read content to buffer so we can inspect it later
	buf := &bytes.Buffer{}
	io.Copy(buf, doc.Reader)

	captured := iosystem.NewDocument(iosystem.Key(doc.Key), doc.Type, buf)
	for k, v := range doc.Metadata.Custom {
		captured.WithMetadata(k, v)
	}
	m.docs = append(m.docs, captured)
	return nil
}

func (m *mockSink) Close() error {
	return nil
}

func TestConduit_ArrayCollector(t *testing.T) {
	// Create source with multiple documents
	src := newMockSource("doc1", "doc2", "doc3")
	snk := newMockSink()

	// Create conduit with ArrayCollector + Identity processor
	// ArrayCollector should collect all, emit on EOF
	// Identity should receive array and pass it through
	cfg := &conduit.Config{Concurrency: 1, ErrorMode: conduit.FailFast}
	c := conduit.New(cfg)
	c.AddProcessor(processor.NewCollector(false))
	c.AddProcessor(processor.NewIdentity())

	ctx := context.Background()
	stats, err := c.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(stats.DocsProcessed, 3),
		it.Equal(len(snk.docs), 3),
	)
}
