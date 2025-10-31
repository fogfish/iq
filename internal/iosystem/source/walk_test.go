package source_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/it/v2"
)

func TestWalkSource(t *testing.T) {
	t.Run("Walk/LocalDir", func(t *testing.T) {
		// Create temp directory with test files
		tmpDir := t.TempDir()
		it.Then(t).Should(
			it.Nil(os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)),
			it.Nil(os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)),
			it.Nil(os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("content3"), 0644)),
		)

		// Create WalkSource
		src, err := source.NewWalkSource(tmpDir, source.WalkConfig{})
		it.Then(t).Should(it.Nil(err))
		defer src.Close()

		// Read all documents
		ctx := context.Background()
		docs := make([]string, 0)
		for {
			doc, err := src.Next(ctx)
			if err == io.EOF {
				break
			}
			it.Then(t).Should(it.Nil(err))

			content, err := io.ReadAll(doc.Reader)
			it.Then(t).Should(it.Nil(err))

			docs = append(docs, string(content))
		}

		// Verify we got all files
		it.Then(t).Should(
			it.Equal(len(docs), 3),
		)
	})

	t.Run("Walk/Empty", func(t *testing.T) {
		// Create empty temp directory
		tmpDir := t.TempDir()

		// Create WalkSource
		src, err := source.NewWalkSource(tmpDir, source.WalkConfig{})
		it.Then(t).Should(it.Nil(err))
		defer src.Close()

		// Should immediately return EOF for empty directory
		ctx := context.Background()
		_, err = src.Next(ctx)
		it.Then(t).Should(
			it.Equal(err, io.EOF),
		)
	})

	t.Run("Walk/ErrorInvalidPath", func(t *testing.T) {
		// Try to create WalkSource with empty path
		src, err := source.NewWalkSource("", source.WalkConfig{})
		it.Then(t).ShouldNot(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.Nil(src),
		)
	})
}
