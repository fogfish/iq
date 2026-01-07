//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor_test

/*

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/it/v2"
)

func TestIdentityProcessor(t *testing.T) {
	proc := processor.NewIdentity()
	defer proc.Close()

	ctx := context.Background()
	doc := iosystem.NewDocument("test.txt", strings.NewReader("test content"))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
		it.Equal(results[0], doc),
	)
}

func TestChunkerProcessor_None(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.ChunkerNone,
	})
	defer proc.Close()

	ctx := context.Background()
	content := "Line 1\nLine 2\nLine 3"
	doc := iosystem.NewDocument("test.txt", strings.NewReader(content))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})
	data, _ := io.ReadAll(results[0].Reader)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
		it.Equal(string(data), content),
	)
}

func TestChunkerProcessor_Sentence(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.ChunkerSentence,
	})
	defer proc.Close()

	ctx := context.Background()
	content := "First sentence. Second sentence. Third sentence."
	doc := iosystem.NewDocument("test.txt", strings.NewReader(content))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 3),
	)

	// Check chunk paths and metadata
	for _, result := range results {
		it.Then(t).Should(
			it.True(strings.Contains(result.Path, "test.txt#chunk")),
			it.Equal(result.Metadata.Custom["original_path"], "test.txt"),
		)
	}
}

func TestChunkerProcessor_Paragraph(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.ChunkerParagraph,
	})
	defer proc.Close()

	ctx := context.Background()
	content := "Paragraph 1\n\nParagraph 2\n\nParagraph 3"
	doc := iosystem.NewDocument("test.txt", strings.NewReader(content))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 3),
	)
}

func TestChunkerProcessor_Chunk(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy:  processor.ChunkerChunk,
		ChunkSize: 50,
	})
	defer proc.Close()

	ctx := context.Background()
	content := strings.Repeat("This is a sentence. ", 10)
	doc := iosystem.NewDocument("test.txt", strings.NewReader(content))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(
		it.Nil(err),
		it.True(len(results) >= 2),
	)
}

func TestChunkerProcessor_EmptyContent(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.ChunkerSentence,
	})
	defer proc.Close()

	ctx := context.Background()
	doc := iosystem.NewDocument("test.txt", strings.NewReader(""))

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(results), 1),
	)
}

func TestChunkerProcessor_PreservesMetadata(t *testing.T) {
	proc := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.ChunkerSentence,
	})
	defer proc.Close()

	ctx := context.Background()
	doc := iosystem.NewDocument("test.txt", strings.NewReader("First. Second.")).
		WithMetadata("source", "test").
		WithMetadata("size", "14")

	results, err := proc.Process(ctx, []*iosystem.Document{doc})

	it.Then(t).Should(it.Nil(err))

	// Check that metadata is preserved in chunks
	for _, result := range results {
		it.Then(t).Should(
			it.Equal(result.Metadata.Custom["source"], "test"),
			it.Equal(result.Metadata.Custom["size"], "14"),
		)
	}
}

*/
