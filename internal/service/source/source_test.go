//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/it/v2"
)

func TestBuilder_NoInput(t *testing.T) {
	// No stdin, no files → returns nil source
	_, err := source.New().Build()

	it.Then(t).ShouldNot(
		it.Nil(err),
	)
}

func TestBuilder_SingleFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("file content"), 0644)
	it.Then(t).Should(it.Nil(err))

	src, err := source.New().
		Files(tmpDir, "test.txt").
		Build()

	it.Then(t).Should(it.Nil(err))
	defer src.Close()

	// Read document
	doc, err := src.Next(context.Background())
	it.Then(t).Should(it.Nil(err))

	content, err := io.ReadAll(doc.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(content), "file content"),
	)

	// Should return EOF on second call
	_, err = src.Next(context.Background())
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestBuilder_MultipleFiles(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	it.Then(t).Should(
		it.Nil(os.WriteFile(file1, []byte("content1"), 0644)),
		it.Nil(os.WriteFile(file2, []byte("content2"), 0644)),
	)

	src, err := source.New().
		Files(tmpDir, "file1.txt", "file2.txt").
		Build()

	it.Then(t).Should(it.Nil(err))
	defer src.Close()

	ctx := context.Background()

	// Read first document
	doc1, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))

	content1, err := io.ReadAll(doc1.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(content1), "content1"),
	)

	// Read second document
	doc2, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))

	content2, err := io.ReadAll(doc2.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(content2), "content2"),
	)

	// Should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

/*
func TestBuilder_MergeFiles(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	it.Then(t).Should(
		it.Nil(os.WriteFile(file1, []byte("content1"), 0644)),
		it.Nil(os.WriteFile(file2, []byte("content2"), 0644)),
	)

	src, err := source.New().
		Files(tmpDir, "file1.txt", "file2.txt").
		Merge(true).
		Build()

	it.Then(t).Should(it.Nil(err))
	defer src.Close()

	// Should get single merged document
	doc, err := src.Next(context.Background())
	it.Then(t).Should(it.Nil(err))

	content, err := io.ReadAll(doc.Reader)
	// Union adds newline separator between documents and at the end
	expected := "content1\ncontent2\n"
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(content), expected),
	)

	// Should return EOF on second call (only one merged document)
	_, err = src.Next(context.Background())
	it.Then(t).Should(it.Equal(err, io.EOF))
}
*/

func TestBuilder_FilesCleanup(t *testing.T) {
	// Test that source properly closes file handles
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	it.Then(t).Should(
		it.Nil(os.WriteFile(file1, []byte("content1"), 0644)),
		it.Nil(os.WriteFile(file2, []byte("content2"), 0644)),
	)

	src, err := source.New().
		Files(tmpDir, "file1.txt", "file2.txt").
		Build()

	it.Then(t).Should(it.Nil(err))

	ctx := context.Background()

	// Read both documents
	doc1, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))
	io.ReadAll(doc1.Reader)

	doc2, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))
	io.ReadAll(doc2.Reader)

	// Close source
	err = src.Close()
	it.Then(t).Should(it.Nil(err))

	// Files should be accessible (not locked)
	_, err = os.Stat(file1)
	it.Then(t).Should(it.Nil(err))
}
