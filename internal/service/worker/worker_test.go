//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package worker_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/iq/internal/service/llm"
	"github.com/fogfish/iq/internal/service/worker"
)

func TestBuilder_NoBlueprint(t *testing.T) {
	// Try to build without blueprint
	_, err := worker.New().Build()

	if err == nil {
		t.Fatal("expected error when building without blueprint")
	}

	if err != worker.ErrBlueprintRequired {
		t.Errorf("expected ErrBlueprintRequired, got %v", err)
	}
}

func TestBuilder_WithBlueprint(t *testing.T) {
	// Create a simple blueprint YAML
	tmpDir := t.TempDir()
	bpFile := filepath.Join(tmpDir, "test.yml")

	bpContent := `
runs-on: mock
entrypoint: main
jobs:
  main:
    prompt: "Echo this input"
`
	if err := os.WriteFile(bpFile, []byte(bpContent), 0644); err != nil {
		t.Fatalf("failed to write blueprint file: %v", err)
	}

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock").Build()
	if err != nil {
		t.Fatalf("failed to create mock LLM: %v", err)
	}

	// Build conduit with blueprint
	pipe, err := worker.New().
		Blueprint(bpFile, mockLLM).
		Build()

	if err != nil {
		t.Fatalf("failed to build conduit: %v", err)
	}

	if pipe == nil {
		t.Fatal("expected conduit, got nil")
	}
}

func TestBuilder_WithOptions(t *testing.T) {
	// Create a simple blueprint
	tmpDir := t.TempDir()
	bpFile := filepath.Join(tmpDir, "test.yml")

	bpContent := `
runs-on: mock
entrypoint: main
jobs:
  main:
    prompt: "Process this"
`
	if err := os.WriteFile(bpFile, []byte(bpContent), 0644); err != nil {
		t.Fatalf("failed to write blueprint file: %v", err)
	}

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock").Build()
	if err != nil {
		t.Fatalf("failed to create mock LLM: %v", err)
	}

	// Build with options
	pipe, err := worker.New().
		Blueprint(bpFile, mockLLM).
		Concurrency(4).
		ErrorMode(conduit.SkipError).
		Progress(func(doc *iosystem.Document, err error) {
			// Progress callback
		}).
		Build()

	if err != nil {
		t.Fatalf("failed to build conduit: %v", err)
	}

	if pipe == nil {
		t.Fatal("expected conduit, got nil")
	}
}

func TestBuilder_BuildAndRun(t *testing.T) {
	// Create a simple blueprint
	tmpDir := t.TempDir()
	bpFile := filepath.Join(tmpDir, "test.yml")

	bpContent := `
runs-on: mock
entrypoint: main
jobs:
  main:
    prompt: "Process this input"
`
	if err := os.WriteFile(bpFile, []byte(bpContent), 0644); err != nil {
		t.Fatalf("failed to write blueprint file: %v", err)
	}

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock").Build()
	if err != nil {
		t.Fatalf("failed to create mock LLM: %v", err)
	}

	// Build conduit
	pipe, err := worker.New().
		Blueprint(bpFile, mockLLM).
		Build()

	if err != nil {
		t.Fatalf("failed to build conduit: %v", err)
	}

	// Create source and sink
	input := []byte("test input data")
	src := source.NewReader("test.txt", bytes.NewReader(input))

	var output bytes.Buffer
	snk := sink.NewWriter(&output)

	// Run conduit
	stats, err := pipe.Run(nil, src, snk)

	if err != nil {
		t.Fatalf("conduit run failed: %v", err)
	}

	if stats == nil {
		t.Fatal("expected stats, got nil")
	}

	if stats.DocsProcessed == 0 {
		t.Error("expected at least one document processed")
	}

	if output.Len() == 0 {
		t.Error("expected output, got empty")
	}
}
