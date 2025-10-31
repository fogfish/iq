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
	StrategyNone      = "none"
	StrategySentence  = "sentence"
	StrategyParagraph = "paragraph"
	StrategyChunk     = "chunk"
)

// Chunker splits documents into chunks based on a strategy.
// Integrates with the existing github.com/fogfish/scanner library.
type Chunker struct {
	strategy       string
	chunkSize      int
	delimiterChars string
}

// ChunkConfig configures the chunker processor.
type ChunkConfig struct {
	Strategy       string // "none", "sentence", "paragraph", "chunk"
	ChunkSize      int    // Size for chunk strategy (default: 1024)
	DelimiterChars string // Delimiter characters (defaults vary by strategy)
}

// NewChunker creates a processor that splits documents into chunks.
// Returns multiple documents from a single input, one per chunk.
func NewChunker(config ChunkConfig) iosystem.Processor {
	if config.Strategy == "" {
		config.Strategy = StrategyNone
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024
	}

	return &Chunker{
		strategy:       config.Strategy,
		chunkSize:      config.ChunkSize,
		delimiterChars: config.DelimiterChars,
	}
}

// Process splits the input document into chunks and returns multiple documents.
func (p *Chunker) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
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
		return []*iosystem.Document{doc}, nil
	}

	return chunks, nil
}

// createScanner creates the appropriate scanner based on strategy.
func (p *Chunker) createScanner(r io.Reader) scanner.Scanner {
	switch p.strategy {
	case StrategySentence:
		chars := p.delimiterChars
		if chars == "" {
			chars = scanner.EndOfSentence
		}
		return scanner.NewSentencer(chars, r)

	case StrategyParagraph:
		chars := p.delimiterChars
		if chars == "" {
			chars = "\n\n"
		}
		return scanner.NewSlicer(chars, r)

	case StrategyChunk:
		chars := p.delimiterChars
		if chars == "" {
			chars = scanner.EndOfSentence
		}
		size := p.chunkSize
		if size == 0 {
			size = 1024
		}
		return scanner.NewChunker(size, scanner.NewSentencer(chars, r))

	default: // StrategyNone
		return scanner.NewIdentity(r)
	}
}

// Close implements iosystem.Processor.
func (p *Chunker) Close() error {
	return nil
}
