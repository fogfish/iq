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

func TestParseForeachWithFormat_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
          format:
            type: json
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format == nil {
		t.Fatal("Format is nil")
	}
	if step.Format.Type != "json" {
		t.Errorf("Expected type 'json', got '%s'", step.Format.Type)
	}
}

func TestParseForeachWithFormat_JSONL(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
          format:
            type: jsonl
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format.Type != "jsonl" {
		t.Errorf("Expected type 'jsonl', got '%s'", step.Format.Type)
	}
}

func TestParseForeachWithFormat_Text(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
          format:
            type: text
            divider: "\n\n"
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", step.Format.Type)
	}
	if step.Format.Divider != "\n\n" {
		t.Errorf("Expected delimiter '\\n\\n', got '%s'", step.Format.Divider)
	}
}

func TestParseForeachWithFormat_DefaultJSON(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format == nil {
		t.Fatal("Format should have default value")
	}
	if step.Format.Type != "json" {
		t.Errorf("Expected default type 'json', got '%s'", step.Format.Type)
	}
}

func TestParseForeachWithFormat_InvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
          format:
            type: xml
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed unexpectedly: %v", err)
	}

	// Invalid type should default to json
	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format.Type != "json" {
		t.Errorf("Expected invalid type to default to 'json', got '%s'", step.Format.Type)
	}
}

func TestParseForeachWithFormat_DefaultDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: test-workflow
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: process
          format:
            type: text
        output: results
  process:
    steps:
      - uses: prompts/process.md
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	step := tree.Jobs["main"].Steps[0].(*ast.ForeachStepNode)
	if step.Format.Divider != "\n" {
		t.Errorf("Expected default delimiter '\\n', got '%s'", step.Format.Divider)
	}
}

func TestParser_EmitAttribute(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yml")

	content := `name: emit-test
jobs:
  main:
    steps:
      - uses: prompts/summarize.md
        emit: summary

      - foreach:
          selector: document
          job: process
        emit: processed

      - run: echo "test"
        emit: output

  router:
    steps:
      - uses: prompts/decide.md
        switch:
          - when: "true"
            route: next
        emit: decision
`

	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, tree, err := p.ParseBlueprint(yamlFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check agent step emit
	agentStep := tree.Jobs["main"].Steps[0].(*ast.AgentStepNode)
	if agentStep.Emit != "summary" {
		t.Errorf("AgentStep emit: got %q, want %q", agentStep.Emit, "summary")
	}

	// Check foreach step emit
	foreachStep := tree.Jobs["main"].Steps[1].(*ast.ForeachStepNode)
	if foreachStep.Emit != "processed" {
		t.Errorf("ForeachStep emit: got %q, want %q", foreachStep.Emit, "processed")
	}

	// Check run step emit
	runStep := tree.Jobs["main"].Steps[2].(*ast.RunStepNode)
	if runStep.Emit != "output" {
		t.Errorf("RunStep emit: got %q, want %q", runStep.Emit, "output")
	}

	// Check router step emit
	routerStep := tree.Jobs["router"].Steps[0].(*ast.RouterStepNode)
	if routerStep.Emit != "decision" {
		t.Errorf("RouterStep emit: got %q, want %q", routerStep.Emit, "decision")
	}
}
