// Example demonstrating spool integration with the refactored Pipeline
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"io"

	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/iq/internal/iosystem/source"
)

func main() {
	fmt.Println("Spool Integration Example")
	fmt.Println("==========================")
	fmt.Println()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// Create temporary directories for input/output
	inputDir, err := os.MkdirTemp("", "spool-input-*")
	if err != nil {
		return fmt.Errorf("failed to create input dir: %w", err)
	}
	defer os.RemoveAll(inputDir)

	outputDir, err := os.MkdirTemp("", "spool-output-*")
	if err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// Create some test files
	files := map[string]string{
		"doc1.txt": "Hello, World! This is document one.",
		"doc2.txt": "Welcome to document two. It has multiple sentences. This is the third one.",
		"doc3.txt": "Short document.",
	}

	for filename, content := range files {
		path := fmt.Sprintf("%s/%s", inputDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	fmt.Printf("Created %d test files in %s\n", len(files), inputDir)
	fmt.Printf("Output directory: %s\n\n", outputDir)

	// Build reusable pipeline with processors
	pipeline := iosystem.NewConduit(&iosystem.Config{
		ErrorMode: iosystem.SkipError, // Continue on errors
		Progress: func(doc *iosystem.Document, err error) {
			if err != nil {
				fmt.Printf("  ❌ %s: %v\n", doc.Path, err)
			} else {
				fmt.Printf("  ✅ %s\n", doc.Path)
			}
		},
	})

	// Add processors - let's chunk by sentences
	chunker := processor.NewChunker(processor.ChunkConfig{
		Strategy: processor.StrategySentence,
	})
	pipeline.AddProcessor(chunker)

	// Mount filesystems using lfs (local filesystem)
	inFS, err := lfs.New(inputDir)
	if err != nil {
		return fmt.Errorf("failed to mount input FS: %w", err)
	}

	outFS, err := lfs.New(outputDir)
	if err != nil {
		return fmt.Errorf("failed to mount output FS: %w", err)
	}

	// Create spool with transactional semantics
	// IsMutable: remove input files after successful processing
	sp := spool.New(inFS, outFS,
		spool.IsMutable,     // Queue mode - remove files after processing
		spool.WithSkipError, // Continue on errors
	)

	fmt.Println("Processing files through pipeline...")
	totalFiles := 0
	totalChunks := 0

	// Process all files transactionally using spool
	err = sp.ForEach(ctx, "/",
		func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
			totalFiles++

			// Create ephemeral source/sink from spool's reader/writer
			src := source.NewReader(path, r)
			snk := sink.NewWriter(w)

			// Run the reusable pipeline with this file
			stats, err := pipeline.Run(ctx, src, snk)
			if err != nil {
				return err
			}

			totalChunks += stats.DocsProcessed
			return nil
		})

	if err != nil {
		return fmt.Errorf("spool processing failed: %w", err)
	}

	fmt.Printf("\n✅ Successfully processed %d files into %d chunks\n", totalFiles, totalChunks)
	fmt.Printf("\nOutput files in %s:\n", outputDir)

	// List output files
	entries, _ := os.ReadDir(outputDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			content, _ := os.ReadFile(fmt.Sprintf("%s/%s", outputDir, entry.Name()))
			preview := string(content)
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			fmt.Printf("  - %s: %q\n", entry.Name(), preview)
		}
	}

	// Verify input files were removed (mutable mode)
	fmt.Printf("\nInput directory after processing:\n")
	inputEntries, _ := os.ReadDir(inputDir)
	if len(inputEntries) == 0 {
		fmt.Println("  (empty - files were removed after successful processing)")
	} else {
		fmt.Printf("  %d files remaining\n", len(inputEntries))
	}

	return nil
}

// Example showing how to migrate from SpoolPipeline to new pattern
func migrationExample() {
	// OLD WAY (using SpoolPipeline):
	/*
		pipeline := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			Mutable: true,
			Strict:  false,
			AllowEmptyOutput: true,
		})
		pipeline.AddProcessor(processor)
		pipeline.Run(ctx, "/")
	*/

	// NEW WAY (using Pipeline + spool):
	ctx := context.Background()
	inputDir := "/path/to/input"
	outputDir := "/path/to/output"

	// 1. Build reusable pipeline
	pipeline := iosystem.NewConduit(&iosystem.Config{
		ErrorMode: iosystem.SkipError,
	})
	// pipeline.AddProcessor(...)

	// 2. Mount filesystems
	inFS, _ := lfs.New(inputDir)
	outFS, _ := lfs.New(outputDir)

	// 3. Create spool with desired options
	sp := spool.New(inFS, outFS,
		spool.IsMutable,     // Corresponds to Mutable: true
		spool.WithSkipError, // Corresponds to Strict: false
	)

	// 4. Process with transactional semantics
	_ = sp.ForEach(ctx, "/",
		func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
			// Wrap spool's reader/writer
			src := source.NewReader(path, r)
			snk := sink.NewWriter(w)

			// Run pipeline
			stats, err := pipeline.Run(ctx, src, snk)

			// Handle AllowEmptyOutput logic
			if stats.BytesWritten == 0 {
				// Decide whether to return error or not
				// return fmt.Errorf("no output for %s", path)
			}

			return err
		})
}

type uppercaser struct{}

func (u *uppercaser) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, err
	}

	upper := strings.ToUpper(string(content))
	result := iosystem.NewDocument(doc.Path, strings.NewReader(upper))

	// Copy metadata
	for k, v := range doc.Metadata {
		result.WithMetadata(k, v)
	}

	return []*iosystem.Document{result}, nil
}

func (u *uppercaser) Close() error {
	return nil
}
