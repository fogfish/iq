//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/blueprint/parser"
)

func TestParser_SplitStep_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_basic.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - split:
          strategy: paragraph
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	if mainJob == nil {
		t.Fatal("expected 'main' job, not found")
	}

	if len(mainJob.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(mainJob.Steps))
	}

	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Strategy != "paragraph" {
		t.Errorf("expected strategy 'paragraph', got '%s'", splitStep.Strategy)
	}

	if splitStep.Output != "chunks" {
		t.Errorf("expected output 'chunks', got '%s'", splitStep.Output)
	}

	// Check defaults
	if splitStep.Size != 1024 {
		t.Errorf("expected default size 1024, got %d", splitStep.Size)
	}

	if splitStep.Overlap != 0 {
		t.Errorf("expected default overlap 0, got %d", splitStep.Overlap)
	}
}

func TestParser_SplitStep_DefaultStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_default.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - split: {}
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Strategy != "none" {
		t.Errorf("expected default strategy 'none', got '%s'", splitStep.Strategy)
	}
}

func TestParser_SplitStep_ChunkWithSize(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_chunk.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - split:
          strategy: chunk
          size: 2048
          overlap: 200
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Strategy != "chunk" {
		t.Errorf("expected strategy 'chunk', got '%s'", splitStep.Strategy)
	}

	if splitStep.Size != 2048 {
		t.Errorf("expected size 2048, got %d", splitStep.Size)
	}

	if splitStep.Overlap != 200 {
		t.Errorf("expected overlap 200, got %d", splitStep.Overlap)
	}
}

func TestParser_SplitStep_AllStrategies(t *testing.T) {
	strategies := []string{"none", "sentence", "paragraph", "chunk", "tag"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			tmpDir := t.TempDir()
			yamlFile := filepath.Join(tmpDir, "split.yml")

			content := `name: test-split
jobs:
  main:
    steps:
      - split:
          strategy: ` + strategy + `
        output: chunks`

			if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			p := parser.New(tmpDir)
			tree, err := p.Parse(yamlFile)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			mainJob := tree.Blueprint.Jobs["main"]
			splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
			if !ok {
				t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
			}

			if splitStep.Strategy != strategy {
				t.Errorf("expected strategy '%s', got '%s'", strategy, splitStep.Strategy)
			}
		})
	}
}

func TestParser_SplitStep_WithChars(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_chars.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - split:
          strategy: sentence
          chars: ".!?"
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Chars != ".!?" {
		t.Errorf("expected chars '.!?', got '%s'", splitStep.Chars)
	}
}

func TestParser_SplitStep_WithRetry(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_retry.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - split:
          strategy: paragraph
        output: chunks
        retry:
          attempts: 3
          delay: 5`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Retry == nil {
		t.Fatal("expected retry configuration, got nil")
	}

	if splitStep.Retry.Attempts != 3 {
		t.Errorf("expected retry attempts 3, got %d", splitStep.Retry.Attempts)
	}

	if splitStep.Retry.Delay != 5 {
		t.Errorf("expected retry delay 5, got %d", splitStep.Retry.Delay)
	}
}

func TestParser_SplitStep_WithName(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_name.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - name: split-document
        split:
          strategy: paragraph
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Name != "split-document" {
		t.Errorf("expected name 'split-document', got '%s'", splitStep.Name)
	}
}

func TestParser_SplitStep_WithRunsOn(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_runson.yml")

	content := `name: test-split
jobs:
  main:
    steps:
      - runs-on: custom-runner
        split:
          strategy: chunk
          size: 4096
        output: chunks`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.RunsOn != "custom-runner" {
		t.Errorf("expected runs-on 'custom-runner', got '%s'", splitStep.RunsOn)
	}
}

func TestParser_SplitStep_IntegrationWithForeach(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "split_foreach.yml")

	content := `name: test-split-foreach
jobs:
  main:
    steps:
      - split:
          strategy: paragraph
        output: chunks
      - foreach:
          selector: chunks
          job: process

  process:
    steps:
      - uses: process.md`

	// Create the process.md file
	processFile := filepath.Join(tmpDir, "process.md")
	if err := os.WriteFile(processFile, []byte("Process: {{.input}}"), 0644); err != nil {
		t.Fatalf("failed to write process file: %v", err)
	}

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(yamlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	if len(mainJob.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(mainJob.Steps))
	}

	splitStep, ok := mainJob.Steps[0].(*ast.SplitStepNode)
	if !ok {
		t.Fatalf("expected first step to be *ast.SplitStepNode, got %T", mainJob.Steps[0])
	}

	if splitStep.Output != "chunks" {
		t.Errorf("expected split output 'chunks', got '%s'", splitStep.Output)
	}

	foreachStep, ok := mainJob.Steps[1].(*ast.ForeachStepNode)
	if !ok {
		t.Fatalf("expected second step to be *ast.ForeachStepNode, got %T", mainJob.Steps[1])
	}

	if foreachStep.Selector != "chunks" {
		t.Errorf("expected foreach selector 'chunks', got '%s'", foreachStep.Selector)
	}
}
