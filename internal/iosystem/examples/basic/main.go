// Example demonstrating basic pipeline usage with iosystem
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/iq/internal/iosystem/sink"
)

// Simple source for demonstration
type simpleSource struct {
	read bool
	text string
}

func (s *simpleSource) Next(ctx context.Context) (*iosystem.Document, error) {
	if s.read {
		return nil, fmt.Errorf("EOF")
	}
	s.read = true
	return iosystem.NewDocument("sample.txt", strings.NewReader(s.text)), nil
}

func (s *simpleSource) Close() error {
	return nil
}

// Multi-document source for demonstration
type multiSource struct {
	texts []string
	index int
}

func (s *multiSource) Next(ctx context.Context) (*iosystem.Document, error) {
	if s.index >= len(s.texts) {
		return nil, fmt.Errorf("EOF")
	}
	text := s.texts[s.index]
	s.index++
	doc := iosystem.NewDocument(fmt.Sprintf("doc%d.txt", s.index), strings.NewReader(text))
	return doc, nil
}

func (s *multiSource) Close() error {
	return nil
}

func main() {
	fmt.Println("IOSystem Phase 1 Examples")
	fmt.Println()
	example2_WithChunking()
	example3_WithProgressTracking()
}

// Example 2: Chunking text by sentences
func example2_WithChunking() {
	fmt.Println("=== Example: Text Chunking ===")

	text := "This is the first sentence. Here is the second one. And finally the third sentence."
	src := &simpleSource{text: text}

	snk := sink.NewStdoutSink()
	chunker := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.StrategySentence,
	})

	pipeline := iosystem.NewConduit(nil).
		AddProcessor(chunker)

	ctx := context.Background()
	stats, err := pipeline.Run(ctx, src, snk)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nProcessed %d documents in %s\n\n", stats.DocsProcessed, stats.Duration)
}

// Example 3: With progress tracking
func example3_WithProgressTracking() {
	fmt.Println("=== Example: Progress Tracking ===")

	texts := []string{
		"Document one content.",
		"Document two content.",
		"Document three content.",
	}

	src := &multiSource{texts: texts}
	snk := sink.NewStdoutSink()

	config := &iosystem.Config{
		Progress: func(doc *iosystem.Document, err error) {
			if err != nil {
				fmt.Printf("❌ Error processing %s: %v\n", doc.Path, err)
			} else {
				fmt.Printf("✅ Processed: %s\n", doc.Path)
			}
		},
	}
	pipeline := iosystem.NewConduit(config)

	ctx := context.Background()
	stats, err := pipeline.Run(ctx, src, snk)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFinal stats:\n")
	fmt.Printf("  Documents: %d\n", stats.DocsProcessed)
	fmt.Printf("  Duration: %s\n", stats.Duration)
}
