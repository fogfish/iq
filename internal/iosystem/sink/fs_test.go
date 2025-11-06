//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package sink_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/it/v2"
	"github.com/fogfish/stream/lfs"
)

func TestFSSink(t *testing.T) {
	t.Run("Write/SingleFile", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Write document (lfs paths don't need leading slash)
		ctx := context.Background()
		doc := iosystem.NewDocument("/test.txt", io.NopCloser(strings.NewReader("test content")))
		err = snk.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(content), "test content"),
		)
	})

	t.Run("Write/PreservePathStructure", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Write document with nested path
		ctx := context.Background()
		doc := iosystem.NewDocument("/subdir/nested/file.txt",
			io.NopCloser(strings.NewReader("nested content")))
		err = snk.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))

		// Verify file was created with correct path
		content, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "nested", "file.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(content), "nested content"),
		)
	})

	t.Run("Write/MultipleFiles", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Write multiple documents
		ctx := context.Background()
		docs := []struct {
			path    string
			content string
		}{
			{"/file1.txt", "content1"},
			{"/file2.txt", "content2"},
			{"/dir/file3.txt", "content3"},
		}

		for _, d := range docs {
			doc := iosystem.NewDocument(d.path, io.NopCloser(strings.NewReader(d.content)))
			err := snk.Write(ctx, doc)
			it.Then(t).Should(it.Nil(err))
		}

		// Verify all files were created
		for _, d := range docs {
			relPath := strings.TrimPrefix(d.path, "/")
			content, err := os.ReadFile(filepath.Join(tmpDir, relPath))
			it.Then(t).Should(
				it.Nil(err),
				it.Equal(string(content), d.content),
			)
		}
	})

	// t.Run("Write/PathWithoutLeadingSlash", func(t *testing.T) {
	// 	// Create temp directory
	// 	tmpDir := t.TempDir()

	// 	// Create filesystem
	// 	fsys, err := lfs.New(tmpDir)
	// 	it.Then(t).Should(it.Nil(err))

	// 	// Create FSSink
	// 	snk, err := sink.NewFS(fsys)
	// 	it.Then(t).Should(it.Nil(err))
	// 	defer snk.Close()

	// 	// Write document without leading slash (lfs requires leading slash)
	// 	// This should fail
	// 	ctx := context.Background()
	// 	doc := iosystem.NewDocument("test.txt", io.NopCloser(strings.NewReader("test content")))
	// 	err = snk.Write(ctx, doc)
	// 	it.Then(t).ShouldNot(it.Nil(err)) // Expect error - lfs requires leading /
	// })

	t.Run("Write/LargeFile", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Create large content (1MB)
		largeContent := strings.Repeat("x", 1024*1024)

		// Write document
		ctx := context.Background()
		doc := iosystem.NewDocument("/large.txt", io.NopCloser(strings.NewReader(largeContent)))
		err = snk.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))

		// Verify file size
		info, err := os.Stat(filepath.Join(tmpDir, "large.txt"))
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(info.Size(), int64(1024*1024)),
		)
	})

	t.Run("Write/ErrorNilDocument", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Try to write nil document
		ctx := context.Background()
		err = snk.Write(ctx, nil)
		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("ErrorNonexistentPath", func(t *testing.T) {
		// Try to create filesystem with nonexistent path
		_, err := lfs.New("/nonexistent/path/that/does/not/exist")
		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("Write/OverwriteExistingFile", func(t *testing.T) {
		// Create temp directory with existing file
		tmpDir := t.TempDir()
		existingFile := filepath.Join(tmpDir, "test.txt")
		it.Then(t).Should(
			it.Nil(os.WriteFile(existingFile, []byte("old content"), 0644)),
		)

		// Create filesystem
		fsys, err := lfs.New(tmpDir)
		it.Then(t).Should(it.Nil(err))

		// Create FSSink
		snk, err := sink.NewFS(fsys)
		it.Then(t).Should(it.Nil(err))
		defer snk.Close()

		// Write document with same path
		ctx := context.Background()
		doc := iosystem.NewDocument("/test.txt", io.NopCloser(strings.NewReader("new content")))
		err = snk.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))

		// Verify file was overwritten
		content, err := os.ReadFile(existingFile)
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(string(content), "new content"),
		)
	})
}
