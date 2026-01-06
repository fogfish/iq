//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source_test

/* TODO: fix

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/it/v2"
	"github.com/fogfish/stream/lfs"
)

func TestWalkSource_SingleFile(t *testing.T) {
	// Create temporary directory with one file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	it.Then(t).Should(it.Nil(err))

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	// Walk the root directory - lfs uses "/" for root
	src, err := source.NewStorage(fsys, "/")
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

	// Second call should return EOF (no more files)
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_MultipleFiles(t *testing.T) {
	// Create temporary directory with multiple files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("content 1"), 0644)
	os.WriteFile(file2, []byte("content 2"), 0644)
	os.WriteFile(file3, []byte("content 3"), 0644)

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	src, err := source.NewStorage(fsys, "/")
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// Read all three files
	paths := make([]string, 0)
	for i := 0; i < 3; i++ {
		doc, err := src.Next(ctx)
		it.Then(t).Should(it.Nil(err))

		paths = append(paths, doc.Path)

		data, err := io.ReadAll(doc.Reader)
		it.Then(t).Should(
			it.Nil(err),
			it.True(len(data) > 0),
		)
	}

	// Verify we got all 3 files
	it.Then(t).Should(it.Equal(len(paths), 3))

	// Fourth call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_NestedDirectories(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	it.Then(t).Should(it.Nil(err))

	// Create files in root and subdirectory
	rootFile := filepath.Join(tmpDir, "root.txt")
	subFile := filepath.Join(subDir, "sub.txt")

	os.WriteFile(rootFile, []byte("root content"), 0644)
	os.WriteFile(subFile, []byte("sub content"), 0644)

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	// Use "/" for root directory with lfs
	src, err := source.NewStorage(fsys, "/")
	it.Then(t).Must(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// Read both files
	paths := make([]string, 0)
	for i := 0; i < 2; i++ {
		doc, err := src.Next(ctx)
		it.Then(t).Must(it.Nil(err))

		paths = append(paths, doc.Path)

		data, err := io.ReadAll(doc.Reader)
		it.Then(t).Should(
			it.Nil(err),
			it.True(len(data) > 0),
		)
	}

	// Verify we got both files
	it.Then(t).Should(it.Equal(len(paths), 2))

	// Third call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_EmptyDirectory(t *testing.T) {
	// Create empty directory
	tmpDir := t.TempDir()

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	src, err := source.NewStorage(fsys, "/")
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// First call should return EOF (no files)
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_NilFilesystem(t *testing.T) {
	_, err := source.NewStorage(nil, "/")
	it.Then(t).Should(it.True(err != nil))
}

func TestWalkSource_NonExistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	// Try to walk a non-existent directory
	wlk, err := source.NewStorage(fsys, "/nonexistent/")
	it.Then(t).Should(it.Nil(err))

	_, err = wlk.Next(context.Background())
	it.Then(t).ShouldNot(it.Nil(err))
}

func TestWalkSource_SubdirectoryWalk(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	it.Then(t).Should(it.Nil(err))

	// Create files in root and subdirectory
	rootFile := filepath.Join(tmpDir, "root.txt")
	subFile := filepath.Join(subDir, "sub.txt")

	os.WriteFile(rootFile, []byte("root content"), 0644)
	os.WriteFile(subFile, []byte("sub content"), 0644)

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	// Walk only the subdirectory
	src, err := source.NewStorage(fsys, "/subdir/")
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// Should only get the file from subdirectory
	doc, err := src.Next(ctx)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(doc.Path, "subdir/sub.txt"),
	)

	data, err := io.ReadAll(doc.Reader)
	it.Then(t).Should(
		it.Nil(err),
		it.Equal(string(data), "sub content"),
	)

	// Second call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_OnlyFilesNotDirectories(t *testing.T) {
	// Create directory structure with multiple nested directories
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	dir3 := filepath.Join(dir1, "dir3")
	err := os.MkdirAll(dir1, 0755)
	it.Then(t).Should(it.Nil(err))
	err = os.MkdirAll(dir2, 0755)
	it.Then(t).Should(it.Nil(err))
	err = os.MkdirAll(dir3, 0755)
	it.Then(t).Should(it.Nil(err))

	// Create one file
	file := filepath.Join(dir3, "file.txt")
	os.WriteFile(file, []byte("content"), 0644)

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	src, err := source.NewStorage(fsys, "/")
	it.Then(t).Should(it.Nil(err))

	if src != nil {
		defer src.Close()
	}

	ctx := context.Background()

	// Should only get one document (the file, not directories)
	doc, err := src.Next(ctx)
	it.Then(t).Must(it.Nil(err))
	it.Then(t).Should(it.Equal(doc.Path, "dir1/dir3/file.txt"))

	// Second call should return EOF
	_, err = src.Next(ctx)
	it.Then(t).Should(it.Equal(err, io.EOF))
}

func TestWalkSource_MultipleEOF(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	// Create filesystem using lfs
	fsys, err := lfs.New(tmpDir)
	it.Then(t).Should(it.Nil(err))

	src, err := source.NewStorage(fsys, "/")
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

*/
