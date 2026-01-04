//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

func TestFSStorage_PutGet(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	ctx := context.Background()
	key := iosystem.Key("test/file.txt")
	content := []byte("hello world")

	// Put
	err = store.Put(ctx, key, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	reader, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("Content mismatch: got %q, want %q", got, content)
	}
}

func TestFSStorage_Has(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	ctx := context.Background()
	key := iosystem.Key("test.txt")

	// Should not exist
	exists, err := store.Has(ctx, key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if exists {
		t.Error("Key should not exist")
	}

	// Create file
	store.Put(ctx, key, bytes.NewReader([]byte("data")))

	// Should exist now
	exists, err = store.Has(ctx, key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist")
	}
}

func TestFSStorage_Walk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "b.txt"), []byte("b"), 0644)

	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	ctx := context.Background()
	var keys []iosystem.Key

	err = store.Walk(ctx, "", func(doc *iosystem.Document) error {
		keys = append(keys, doc.Key)
		// Close the reader if it's a ReadCloser
		if closer, ok := doc.Reader.(io.Closer); ok {
			closer.Close()
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d: %v", len(keys), keys)
	}
}

func TestFSStorage_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	ctx := context.Background()
	key := iosystem.Key("nonexistent.txt")

	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting nonexistent key")
	}
}

func TestFSStorage_EmptyPath(t *testing.T) {
	_, err := storage.NewFS("")
	if err == nil {
		t.Error("Expected error when creating storage with empty path")
	}
}

func TestFSStorage_PutOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	ctx := context.Background()
	key := iosystem.Key("test.txt")

	// Put first version
	content1 := []byte("version 1")
	err = store.Put(ctx, key, bytes.NewReader(content1))
	if err != nil {
		t.Fatalf("First Put failed: %v", err)
	}

	// Overwrite with second version
	content2 := []byte("version 2")
	err = store.Put(ctx, key, bytes.NewReader(content2))
	if err != nil {
		t.Fatalf("Second Put failed: %v", err)
	}

	// Get should return second version
	reader, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(got, content2) {
		t.Errorf("Content mismatch: got %q, want %q", got, content2)
	}
}
