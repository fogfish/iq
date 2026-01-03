//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/scanner"
)

// Chunking strategies
const (
	ChunkerNone      = "none"
	ChunkerSentence  = "sentence"
	ChunkerParagraph = "paragraph"
	ChunkerChunk     = "chunk"
	ChunkerTag       = "tag"
)

// Chunker splits documents into chunks based on a strategy.
// Integrates with the existing github.com/fogfish/scanner library.
type Chunker struct {
	strategy       string
	chunkSize      int
	delimiterChars string
	lobuffer       int
	hibuffer       int
}

// ChunkConfig configures the chunker processor.
type ChunkConfig struct {
	Strategy       string // "none", "sentence", "paragraph", "chunk"
	ChunkSize      int    // Size for chunk strategy (default: 1024)
	DelimiterChars string // Delimiter characters (defaults vary by strategy)
	Buffer         int
}

// NewChunker creates a processor that splits documents into chunks.
// Returns multiple documents from a single input, one per chunk.
func NewChunker(config ChunkConfig) iosystem.Processor {
	if config.Strategy == "" {
		config.Strategy = ChunkerNone
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024
	}
	if config.Buffer == 0 {
		config.Buffer = 1024 * 1024
	}

	return &Chunker{
		strategy:       config.Strategy,
		chunkSize:      config.ChunkSize,
		delimiterChars: config.DelimiterChars,
		lobuffer:       64 * 1024,
		hibuffer:       config.Buffer,
	}
}

// Process splits the input document into chunks and returns multiple documents.
func (p *Chunker) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Passthrough EOF or empty
	if len(docs) == 0 || (len(docs) == 1 && docs[0].Type == iosystem.ContentEOF) {
		return docs, nil
	}

	results := make([]*iosystem.Document, 0)

	for _, doc := range docs {
		// Create scanner based on strategy
		s := p.createScanner(doc.Reader)

		var chunks []*iosystem.Document
		chunkNum := 0

		for s.Scan() {
			txt := strings.TrimSpace(s.Text())
			if len(txt) == 0 {
				continue
			}

			chunkNum++
			chunkPath := fmt.Sprintf("%s#chunk%d", doc.Path, chunkNum)

			chunkDoc := iosystem.NewDocument(chunkPath, strings.NewReader(txt))

			// Copy metadata from original
			for k, v := range doc.Metadata {
				chunkDoc.WithMetadata(k, v)
			}
			chunkDoc.WithMetadata("chunk_num", fmt.Sprintf("%d", chunkNum))
			chunkDoc.WithMetadata("original_path", doc.Path)

			chunks = append(chunks, chunkDoc)
		}

		if err := s.Err(); err != nil {
			return nil, fmt.Errorf("chunking error: %w", err)
		}

		// If no chunks were produced, return original document
		if len(chunks) == 0 {
			results = append(results, doc)
		} else {
			results = append(results, chunks...)
		}
	}

	return results, nil
}

// createScanner creates the appropriate scanner based on strategy.
func (p *Chunker) createScanner(r io.Reader) scanner.Scanner {
	switch p.strategy {
	case ChunkerSentence:
		chars := p.delimiterChars
		if chars == "" {
			chars = scanner.EndOfSentence
		}
		s := scanner.NewSentencer(chars, r)
		s.Buffer(make([]byte, 0, p.lobuffer), p.hibuffer)
		return s

	case ChunkerParagraph:
		chars := p.delimiterChars
		if chars == "" {
			chars = "\n\n"
		}
		s := scanner.NewSlicer(chars, r)
		s.Buffer(make([]byte, 0, p.lobuffer), p.hibuffer)
		return s

	case ChunkerChunk:
		chars := p.delimiterChars
		if chars == "" {
			chars = scanner.EndOfSentence
		}
		size := p.chunkSize
		if size == 0 {
			size = 1024
		}
		s := scanner.NewSentencer(chars, r)
		s.Buffer(make([]byte, 0, p.lobuffer), p.hibuffer)
		return scanner.NewChunker(size, s)

	case ChunkerTag:
		chars := p.delimiterChars
		if chars == "" {
			chars = "details"
		}
		s := scanner.NewTagger(fmt.Sprintf("<%s>", chars), fmt.Sprintf("</%s>", chars), r)
		s.Buffer(make([]byte, 0, p.lobuffer), p.hibuffer)
		return s

	default: // StrategyNone
		return scanner.NewIdentity(r)
	}
}

// Close implements iosystem.Processor.
func (p *Chunker) Close() error {
	return nil
}
