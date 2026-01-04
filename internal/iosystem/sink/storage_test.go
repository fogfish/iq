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
	"testing"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

func TestStorageSink_WithEmit(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create storage
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create sink
	snk := sink.NewStorage(store)
	defer snk.Close()
	
	// Create document
	doc := &iosystem.Document{
		Key:    "test.txt",
		Path:   "input/test.txt",
		Reader: bytes.NewReader([]byte("test content")),
	}
	
	// Create context with emit
	ctx := context.Background()
	emitCtx := &compiler.EmitContext{
		Prefix: "output",
	}
	ctx = compiler.WithEmitContext(ctx, emitCtx)
	
	// Write document
	err = snk.Write(ctx, doc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Verify output at correct location
	expectedKey := iosystem.Key("output/test.txt")
	exists, err := store.Has(ctx, expectedKey)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected output at %s", expectedKey)
	}
}

func TestStorageSink_WithEmitCounters(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create storage
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create sink
	snk := sink.NewStorage(store)
	defer snk.Close()
	
	// Create document
	doc := &iosystem.Document{
		Key:    "test.txt",
		Path:   "input/test.txt",
		Reader: bytes.NewReader([]byte("test content")),
	}
	
	// Create context with emit and counters
	ctx := context.Background()
	emitCtx := &compiler.EmitContext{
		Prefix:   "output",
		Counters: []int{1, 2},
	}
	ctx = compiler.WithEmitContext(ctx, emitCtx)
	
	// Write document
	err = snk.Write(ctx, doc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Verify output with counters
	expectedKey := iosystem.Key("output/test.000001.000002.txt")
	exists, err := store.Has(ctx, expectedKey)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected output at %s", expectedKey)
	}
}

func TestStorageSink_WithoutEmit(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create storage
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create sink
	snk := sink.NewStorage(store)
	defer snk.Close()
	
	// Create document
	doc := &iosystem.Document{
		Key:    "test.txt",
		Path:   "input/test.txt",
		Reader: bytes.NewReader([]byte("test content")),
	}
	
	// Create context without emit
	ctx := context.Background()
	
	// Write document
	err = snk.Write(ctx, doc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Verify output at original key
	expectedKey := iosystem.Key("test.txt")
	exists, err := store.Has(ctx, expectedKey)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected output at %s", expectedKey)
	}
}
