package iosystem_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/it/v2"
)

func TestSpoolPipeline(t *testing.T) {
	t.Run("Basic/ImmutableMode", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test files
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file1.txt"), []byte("content1"), 0644)),
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file2.txt"), []byte("content2"), 0644)),
		)

		// Create pipeline with identity processor (pass-through)
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			Mutable:          false, // Immutable - keep input files
			AllowEmptyOutput: true,
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(processor.NewIdentity())

		// Run pipeline
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		it.Then(t).Should(it.Nil(err))

		// Verify output files were created
		out1, err := os.ReadFile(filepath.Join(outputDir, "file1.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(out1), "content1"),
		)

		out2, err := os.ReadFile(filepath.Join(outputDir, "file2.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(out2), "content2"),
		)

		// Verify input files still exist (immutable mode)
		_, err = os.Stat(filepath.Join(inputDir, "file1.txt"))
		it.Then(t).Should(it.Nil(err))
		_, err = os.Stat(filepath.Join(inputDir, "file2.txt"))
		it.Then(t).Should(it.Nil(err))

		// Check stats
		stats := pipeline.Stats()
		it.Then(t).Should(
			it.Equal(stats.DocsProcessed, 2),
			it.True(stats.BytesRead > 0),
			it.True(stats.BytesWritten > 0),
		)
	})

	t.Run("Basic/MutableMode", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test files
		file1 := filepath.Join(inputDir, "file1.txt")
		file2 := filepath.Join(inputDir, "file2.txt")
		it.Then(t).Should(
			it.Nil(os.WriteFile(file1, []byte("content1"), 0644)),
			it.Nil(os.WriteFile(file2, []byte("content2"), 0644)),
		)

		// Create pipeline with mutable mode
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			Mutable:          true, // Mutable - remove input files after success
			AllowEmptyOutput: true,
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(processor.NewIdentity())

		// Run pipeline
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		it.Then(t).Should(it.Nil(err))

		// Verify output files were created
		_, err = os.Stat(filepath.Join(outputDir, "file1.txt"))
		it.Then(t).Should(it.Nil(err))
		_, err = os.Stat(filepath.Join(outputDir, "file2.txt"))
		it.Then(t).Should(it.Nil(err))

		// Verify input files were removed (mutable mode)
		_, err = os.Stat(file1)
		it.Then(t).Should(it.True(os.IsNotExist(err)))
		_, err = os.Stat(file2)
		it.Then(t).Should(it.True(os.IsNotExist(err)))
	})

	t.Run("AllowEmptyOutput/True", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test file
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file1.txt"), []byte("content"), 0644)),
		)

		// Create processor that returns no documents
		emptyProcessor := &emptyOutputProcessor{}

		// Create pipeline with AllowEmptyOutput=true
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			AllowEmptyOutput: true, // Allow empty output
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(emptyProcessor)

		// Run pipeline - should succeed even though no output
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		it.Then(t).Should(it.Nil(err))

		// Stats should show processing happened
		stats := pipeline.Stats()
		it.Then(t).Should(
			it.Equal(stats.DocsProcessed, 1),
			it.Equal(stats.BytesWritten, int64(0)), // No output written
		)
	})

	t.Run("AllowEmptyOutput/False", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test file
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file1.txt"), []byte("content"), 0644)),
		)

		// Create processor that returns no documents
		emptyProcessor := &emptyOutputProcessor{}

		// Create pipeline with AllowEmptyOutput=false
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			AllowEmptyOutput: false, // Require output
			Strict:           false, // But skip errors
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(emptyProcessor)

		// Run pipeline - in non-strict mode, errors are logged but not returned
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		// Non-strict mode: pipeline doesn't fail but errors are recorded
		it.Then(t).Should(it.Nil(err))

		// Stats should show error
		stats := pipeline.Stats()
		it.Then(t).Should(
			it.True(len(stats.Errors) > 0),
		)
	})

	t.Run("MultipleProcessors", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test file
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file.txt"), []byte("hello"), 0644)),
		)

		// Create pipeline with multiple processors
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			AllowEmptyOutput: true,
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		// Add processors that transform content
		pipeline.
			AddProcessor(&uppercaseProcessor{}).               // HELLO
			AddProcessor(&prefixProcessor{prefix: "OUTPUT: "}) // OUTPUT: HELLO

		// Run pipeline
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		it.Then(t).Should(it.Nil(err))

		// Verify output
		out, err := os.ReadFile(filepath.Join(outputDir, "file.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(out), "OUTPUT: HELLO"),
		)
	})

	t.Run("Error/InvalidInputPath", func(t *testing.T) {
		_, err := iosystem.NewSpoolPipeline("", "/tmp/output", iosystem.SpoolConfig{})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("Error/InvalidOutputPath", func(t *testing.T) {
		_, err := iosystem.NewSpoolPipeline("/tmp/input", "", iosystem.SpoolConfig{})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("Error/ProcessorFailure", func(t *testing.T) {
		// Create temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create test file
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(inputDir, "file.txt"), []byte("content"), 0644)),
		)

		// Create pipeline with failing processor
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{
			Strict: false, // Skip errors
		})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(&failingProcessor{})

		// Run pipeline
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		// Should not fail in non-strict mode but should have errors
		it.Then(t).Should(it.Nil(err))

		// Stats should show errors
		stats := pipeline.Stats()
		it.Then(t).Should(
			it.True(len(stats.Errors) > 0),
		)
	})

	t.Run("EmptyInputDirectory", func(t *testing.T) {
		// Create empty temp directories
		inputDir := t.TempDir()
		outputDir := t.TempDir()

		// Create pipeline
		pipeline, err := iosystem.NewSpoolPipeline(inputDir, outputDir, iosystem.SpoolConfig{})
		it.Then(t).Should(it.Nil(err))
		defer pipeline.Close()

		pipeline.AddProcessor(processor.NewIdentity())

		// Run pipeline on empty directory
		ctx := context.Background()
		err = pipeline.Run(ctx, "/")
		it.Then(t).Should(it.Nil(err))

		// Stats should show no processing
		stats := pipeline.Stats()
		it.Then(t).Should(
			it.Equal(stats.DocsProcessed, 0),
		)
	})
}

// Test helper processors

type emptyOutputProcessor struct{}

func (p *emptyOutputProcessor) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	// Return empty slice - no output
	return []*iosystem.Document{}, nil
}

func (p *emptyOutputProcessor) Close() error {
	return nil
}

type uppercaseProcessor struct{}

func (p *uppercaseProcessor) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, err
	}
	upper := strings.ToUpper(string(content))
	return []*iosystem.Document{
		iosystem.NewDocument(doc.Path, io.NopCloser(strings.NewReader(upper))),
	}, nil
}

func (p *uppercaseProcessor) Close() error {
	return nil
}

type prefixProcessor struct {
	prefix string
}

func (p *prefixProcessor) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, err
	}
	prefixed := p.prefix + string(content)
	return []*iosystem.Document{
		iosystem.NewDocument(doc.Path, io.NopCloser(strings.NewReader(prefixed))),
	}, nil
}

func (p *prefixProcessor) Close() error {
	return nil
}

type failingProcessor struct{}

func (p *failingProcessor) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	return nil, io.ErrUnexpectedEOF
}

func (p *failingProcessor) Close() error {
	return nil
}
