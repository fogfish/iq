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

func TestFileSource(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	it.Then(t).Should(it.Nil(err))

	src, err := source.NewFileSource(testFile)
	it.Then(t).Should(it.Nil(err))

	if src == nil {
		t.Fatal("src is nil")
	}
	defer src.Close()

	ctx := context.Background()

	// First read should succeed
	doc, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))

	if doc == nil {
		t.Fatal("doc is nil")
	}

	it.Then(t).Should(
		it.Equal(doc.Path, testFile),
		it.True(doc.Reader != nil),
	)

	// Read content
	data, err := io.ReadAll(doc.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(data), "test content"),
	)

	// Second call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestFileSource_EmptyPath(t *testing.T) {
	_, err := source.NewFileSource("")
	it.Then(t).Should(it.True(err != nil))
}

func TestFileSource_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")

	src, err := source.NewFileSource(nonExistent)
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()
	_, err = src.Next(ctx)
	it.Then(t).Should(it.True(err != nil))
}

func TestFileSeqSource(t *testing.T) {
	// Create temporary files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("content 1"), 0644)
	os.WriteFile(file2, []byte("content 2"), 0644)
	os.WriteFile(file3, []byte("content 3"), 0644)

	src, err := source.NewFileSeqSource(file1, file2, file3)
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// Read all three files
	for i := 1; i <= 3; i++ {
		doc, err := src.Next(ctx)
		it.Then(t).Should(it.Nil(err))

		data, err := io.ReadAll(doc.Reader)
		it.Then(t).Should(
			it.Nil(err),
			it.True(len(data) > 0),
		)
	}

	// Fourth call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestFileSeqSource_EmptyList(t *testing.T) {
	_, err := source.NewFileSeqSource()
	it.Then(t).Should(it.True(err != nil))
}

func TestFileSeqSource_MultipleEOF(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	src, err := source.NewFileSeqSource(file)
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()
	src.Next(ctx) // Consume the file

	// Multiple EOF calls should all return EOF
	for i := 0; i < 3; i++ {
		_, err := src.Next(ctx)
		it.Then(t).Should(it.Equal(err, io.EOF))
	}
}
