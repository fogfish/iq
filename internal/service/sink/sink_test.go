//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package sink_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/it/v2"
)

func TestBuilder_NoOutput(t *testing.T) {
	// No output specified → should error
	snk, _, err := sink.New().Build()

	it.Then(t).Should(
		it.True(err != nil),
		it.Nil(snk),
	)
}

func TestBuilder_File(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.txt")

	snk, _, err := sink.New().
		File(outFile).
		Build()

	it.Then(t).Should(it.Nil(err))
	defer snk.Close()

	// Write multiple documents
	ctx := context.Background()

	doc1 := iosystem.NewDocument("doc1.txt", io.NopCloser(bytes.NewBufferString("first")))
	err = snk.Write(ctx, doc1)
	it.Then(t).Should(it.Nil(err))

	doc2 := iosystem.NewDocument("doc2.txt", io.NopCloser(bytes.NewBufferString("second")))
	err = snk.Write(ctx, doc2)
	it.Then(t).Should(it.Nil(err))

	// Close to flush
	snk.Close()

	// Read file
	content, err := os.ReadFile(outFile)
	it.Then(t).Should(it.Nil(err))

	// File sink concatenates with newlines
	expected := "first\nsecond\n"
	it.Then(t).Should(
		it.Equal(string(content), expected),
	)
}

func TestBuilder_Dir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	snk, _, err := sink.New().
		Path(tmpDir).
		Build()

	it.Then(t).Should(it.Nil(err))

	// Write document with path
	ctx := context.Background()
	doc := iosystem.NewDocument("/subdir/output.txt", io.NopCloser(bytes.NewBufferString("directory content")))
	err = snk.Write(ctx, doc)
	it.Then(t).Should(it.Nil(err))

	// Close to flush writes
	snk.Close()

	// Verify file was created in correct location
	// stream.NewFS creates files directly in the specified path
	filePath := filepath.Join(tmpDir, "subdir", "output.txt")
	content, err := os.ReadFile(filePath)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(string(content), "directory content"),
	)
}

func TestBuilder_FileBeforeDir(t *testing.T) {
	// File takes priority over directory
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.txt")

	snk, _, err := sink.New().
		File(outFile).
		Path(tmpDir).
		Build()

	it.Then(t).Should(it.Nil(err))
	defer snk.Close()

	// Write document
	ctx := context.Background()
	doc := iosystem.NewDocument("whatever.txt", io.NopCloser(bytes.NewBufferString("file mode")))
	err = snk.Write(ctx, doc)
	it.Then(t).Should(it.Nil(err))
	snk.Close()

	// Should create output.txt (file mode), not whatever.txt (dir mode)
	content, err := os.ReadFile(outFile)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(string(content), "file mode\n"),
	)

	// whatever.txt should NOT exist
	_, err = os.Stat(filepath.Join(tmpDir, "whatever.txt"))
	it.Then(t).Should(
		it.True(err != nil),
	)
}

func TestBuilder_DirMultipleFiles(t *testing.T) {
	// Test directory sink with multiple documents
	tmpDir := t.TempDir()

	snk, _, err := sink.New().
		Path(tmpDir).
		Build()

	it.Then(t).Should(it.Nil(err))
	defer snk.Close()

	// Write multiple documents
	ctx := context.Background()

	docs := []struct {
		path    string
		content string
	}{
		{"/file1.txt", "content 1"},
		{"/subdir/file2.txt", "content 2"},
		{"/subdir/nested/file3.txt", "content 3"},
	}

	for _, d := range docs {
		doc := iosystem.NewDocument(iosystem.Key(d.path), io.NopCloser(bytes.NewBufferString(d.content)))
		err = snk.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))
	}

	// Close to flush writes
	snk.Close()

	// Verify all files exist with correct content
	for _, d := range docs {
		filePath := filepath.Join(tmpDir, d.path)
		content, err := os.ReadFile(filePath)
		it.Then(t).Should(it.Nil(err))

		it.Then(t).Should(
			it.Equal(string(content), d.content),
		)
	}
}
