//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor_test

/*

import (
	"bytes"
	"context"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
)

func TestArrayCollector_CollectAndEmit(t *testing.T) {
	collector := processor.NewArrayCollector()
	ctx := context.Background()

	// Create test documents
	doc1 := &iosystem.Document{
		Type:   iosystem.ContentText,
		Path:   "doc1.txt",
		Reader: bytes.NewReader([]byte("content 1")),
	}
	doc2 := &iosystem.Document{
		Type:   iosystem.ContentText,
		Path:   "doc2.txt",
		Reader: bytes.NewReader([]byte("content 2")),
	}
	doc3 := &iosystem.Document{
		Type:   iosystem.ContentText,
		Path:   "doc3.txt",
		Reader: bytes.NewReader([]byte("content 3")),
	}

	// Process documents - should return empty (collecting)
	result1, err := collector.Process(ctx, []*iosystem.Document{doc1})
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(result1) != 0 {
		t.Errorf("Expected empty result during collection, got %d docs", len(result1))
	}

	result2, err := collector.Process(ctx, []*iosystem.Document{doc2})
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(result2) != 0 {
		t.Errorf("Expected empty result during collection, got %d docs", len(result2))
	}

	result3, err := collector.Process(ctx, []*iosystem.Document{doc3})
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(result3) != 0 {
		t.Errorf("Expected empty result during collection, got %d docs", len(result3))
	}

	// Send EOF - should emit collected array
	eofDoc := &iosystem.Document{
		Type: iosystem.ContentEOF,
		Path: "",
	}
	resultEOF, err := collector.Process(ctx, []*iosystem.Document{eofDoc})
	if err != nil {
		t.Fatalf("Process EOF failed: %v", err)
	}

	// Verify array emission
	if len(resultEOF) != 3 {
		t.Fatalf("Expected 3 documents in array, got %d", len(resultEOF))
	}
	if resultEOF[0].Path != "doc1.txt" {
		t.Errorf("Expected doc1.txt, got %s", resultEOF[0].Path)
	}
	if resultEOF[1].Path != "doc2.txt" {
		t.Errorf("Expected doc2.txt, got %s", resultEOF[1].Path)
	}
	if resultEOF[2].Path != "doc3.txt" {
		t.Errorf("Expected doc3.txt, got %s", resultEOF[2].Path)
	}
}

func TestArrayCollector_EmptyCollection(t *testing.T) {
	collector := processor.NewArrayCollector()
	ctx := context.Background()

	// Send EOF without collecting anything
	eofDoc := &iosystem.Document{
		Type: iosystem.ContentEOF,
		Path: "",
	}
	result, err := collector.Process(ctx, []*iosystem.Document{eofDoc})
	if err != nil {
		t.Fatalf("Process EOF failed: %v", err)
	}

	// Should return empty array
	if len(result) != 0 {
		t.Errorf("Expected empty array, got %d docs", len(result))
	}
}

func TestArrayCollector_MultipleDocumentsAtOnce(t *testing.T) {
	collector := processor.NewArrayCollector()
	ctx := context.Background()

	// Send multiple documents in one call
	docs := []*iosystem.Document{
		{Type: iosystem.ContentText, Path: "a.txt", Reader: bytes.NewReader([]byte("a"))},
		{Type: iosystem.ContentText, Path: "b.txt", Reader: bytes.NewReader([]byte("b"))},
	}

	result, err := collector.Process(ctx, docs)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result during collection, got %d docs", len(result))
	}

	// Send EOF
	eofDoc := &iosystem.Document{Type: iosystem.ContentEOF, Path: ""}
	resultEOF, err := collector.Process(ctx, []*iosystem.Document{eofDoc})
	if err != nil {
		t.Fatalf("Process EOF failed: %v", err)
	}

	if len(resultEOF) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(resultEOF))
	}
}

func TestArrayCollector_Reset(t *testing.T) {
	collector := processor.NewArrayCollector()
	ctx := context.Background()

	// First collection cycle
	doc1 := &iosystem.Document{Type: iosystem.ContentText, Path: "doc1.txt", Reader: bytes.NewReader([]byte("1"))}
	_, _ = collector.Process(ctx, []*iosystem.Document{doc1})

	eofDoc := &iosystem.Document{Type: iosystem.ContentEOF, Path: ""}
	result1, _ := collector.Process(ctx, []*iosystem.Document{eofDoc})

	if len(result1) != 1 {
		t.Fatalf("First cycle: expected 1 doc, got %d", len(result1))
	}

	// Second collection cycle (reuse)
	doc2 := &iosystem.Document{Type: iosystem.ContentText, Path: "doc2.txt", Reader: bytes.NewReader([]byte("2"))}
	_, _ = collector.Process(ctx, []*iosystem.Document{doc2})

	result2, _ := collector.Process(ctx, []*iosystem.Document{eofDoc})

	if len(result2) != 1 {
		t.Fatalf("Second cycle: expected 1 doc, got %d", len(result2))
	}
	if result2[0].Path != "doc2.txt" {
		t.Errorf("Second cycle: expected doc2.txt, got %s", result2[0].Path)
	}
}

*/
