//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source_test

/*

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/it/v2"
)

func TestFileSource_SingleFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	it.Then(t).Should(it.Nil(err))

	// Create filesystem
	fsys := os.DirFS(tmpDir)

	src, err := source.NewFile(fsys, "test.txt")
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
		it.Equal(doc.Path, "test.txt"),
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
	tmpDir := t.TempDir()
	fsys := os.DirFS(tmpDir)

	_, err := source.NewFile(fsys)
	it.Then(t).Should(it.True(err != nil))
}

func TestFileSource_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	fsys := os.DirFS(tmpDir)

	src, err := source.NewFile(fsys, "nonexistent.txt")
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()
	_, err = src.Next(ctx)
	it.Then(t).Should(it.True(err != nil))
}

func TestFileSource_MultipleFiles(t *testing.T) {
	// Create temporary files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("content 1"), 0644)
	os.WriteFile(file2, []byte("content 2"), 0644)
	os.WriteFile(file3, []byte("content 3"), 0644)

	// Create filesystem
	fsys := os.DirFS(tmpDir)

	src, err := source.NewFile(fsys, "file1.txt", "file2.txt", "file3.txt")
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

func TestFileSource_MultipleEOF(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	// Create filesystem
	fsys := os.DirFS(tmpDir)

	src, err := source.NewFile(fsys, "test.txt")
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

func TestFileSource_AutoClose(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("auto close test"), 0644)

	// Create filesystem
	fsys := os.DirFS(tmpDir)

	src, err := source.NewFile(fsys, "test.txt")
	it.Then(t).Should(it.Nil(err))
	defer src.Close()

	ctx := context.Background()
	doc, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))

	// Read the entire file - this should trigger auto-close
	data, err := io.ReadAll(doc.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(data), "auto close test"),
	)

	// Attempt to read again from the same reader should return EOF
	// because the auto-closer closed the file
	buf := make([]byte, 10)
	n, err := doc.Reader.Read(buf)
	it.Then(t).Should(
		it.Equal(n, 0),
		it.Equal(err, io.EOF),
	)
}

*/
