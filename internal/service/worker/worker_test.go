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
	"github.com/fogfish/it/v2"
)

func TestBuilder_NoBlueprint(t *testing.T) {
	// Try to build without blueprint
	_, err := worker.New().Build()

	it.Then(t).Should(
		it.True(err != nil),
	)
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
	err := os.WriteFile(bpFile, []byte(bpContent), 0644)
	it.Then(t).Should(it.Nil(err))

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock", "").Build()
	it.Then(t).Should(it.Nil(err))

	// Build conduit with blueprint
	pipe, err := worker.New().
		Runtime().
		Workflow(bpFile, mockLLM).
		Build()

	it.Then(t).Should(
		it.Nil(err),
		it.True(pipe != nil),
	)
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
	err := os.WriteFile(bpFile, []byte(bpContent), 0644)
	it.Then(t).Should(it.Nil(err))

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock", "").Build()
	it.Then(t).Should(it.Nil(err))

	// Build with options
	pipe, err := worker.New().
		Concurrency(4).
		ErrorMode(conduit.SkipError).
		Progress(func(doc *iosystem.Document, err error) {
			// Progress callback
		}).
		Runtime().
		Workflow(bpFile, mockLLM).
		Build()

	it.Then(t).Should(
		it.Nil(err),
		it.True(pipe != nil),
	)
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
	err := os.WriteFile(bpFile, []byte(bpContent), 0644)
	it.Then(t).Should(it.Nil(err))

	// Create mock LLM
	mockLLM, err := llm.New().Profile("mock", "").Build()
	it.Then(t).Should(it.Nil(err))

	// Build conduit
	pipe, err := worker.New().
		Runtime().
		Workflow(bpFile, mockLLM).
		Build()
	it.Then(t).Must(it.Nil(err))

	// Create source and sink
	input := []byte("test input data")
	src := source.NewReader("test.txt", bytes.NewReader(input))

	var output bytes.Buffer
	snk := sink.NewWriter(&output)

	// Run conduit
	ctx := t.Context()
	stats, err := pipe.Run(ctx, src, snk)

	it.Then(t).Should(
		it.Nil(err),
		it.True(stats != nil),
		it.True(stats.DocsProcessed > 0),
		it.True(output.Len() > 0),
	)
}
