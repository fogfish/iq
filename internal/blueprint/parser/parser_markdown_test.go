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

	"github.com/fogfish/iq/internal/blueprint/parser"
)

func TestParser_ParseMarkdown_WithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "prompt.md")

	content := `---
format: json
---
Extract specifications from: {{.input}}`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(mdFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify AST structure
	if tree.Blueprint == nil {
		t.Fatal("expected blueprint, got nil")
	}

	if tree.Blueprint.Entrypoint != "main" {
		t.Errorf("expected entrypoint 'main', got '%s'", tree.Blueprint.Entrypoint)
	}

	mainJob, exists := tree.Blueprint.Jobs["main"]
	if !exists {
		t.Fatal("expected 'main' job, not found")
	}

	if len(mainJob.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(mainJob.Steps))
	}

	step := mainJob.Steps[0]
	if step.GetUses() != mdFile {
		t.Errorf("expected step to use '%s', got '%s'", mdFile, step.GetUses())
	}

	// Verify agent
	agent, exists := tree.Agents[mdFile]
	if !exists {
		t.Fatalf("expected agent for '%s', not found", mdFile)
	}

	if agent.Format != "json" {
		t.Errorf("expected format 'json', got '%s'", agent.Format)
	}

	expectedPrompt := "Extract specifications from: {{.input}}"
	if agent.Prompt != expectedPrompt {
		t.Errorf("expected prompt '%s', got '%s'", expectedPrompt, agent.Prompt)
	}
}

func TestParser_ParseMarkdown_WithoutFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "simple.md")

	content := `Analyze the following text and provide a summary.

Input: {{.input}}`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(mdFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify blueprint structure
	if tree.Blueprint.Entrypoint != "main" {
		t.Errorf("expected entrypoint 'main', got '%s'", tree.Blueprint.Entrypoint)
	}

	mainJob := tree.Blueprint.Jobs["main"]
	if len(mainJob.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(mainJob.Steps))
	}

	// Verify agent (should have default text format)
	agent := tree.Agents[mdFile]
	if agent.Format != "" {
		t.Errorf("expected empty format (text), got '%s'", agent.Format)
	}

	if agent.Prompt != content {
		t.Errorf("expected full content as prompt, got different content")
	}
}

func TestParser_ParseMarkdown_WithServers(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "with-tools.md")

	content := `---
format: json
servers:
  - type: stdio
    name: filesystem
    command: npx -y @modelcontextprotocol/server-filesystem /tmp
---
List files in the directory: {{.path}}`

	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(mdFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	agent := tree.Agents[mdFile]
	if len(agent.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(agent.Servers))
	}

	server := agent.Servers[0]
	if server.Type != "stdio" {
		t.Errorf("expected server type 'stdio', got '%s'", server.Type)
	}

	if server.Name != "filesystem" {
		t.Errorf("expected server name 'filesystem', got '%s'", server.Name)
	}
}

func TestParser_ParseMarkdown_BlueprintStructure(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")

	content := "Simple prompt"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(mdFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify blueprint has expected structure
	bp := tree.Blueprint

	if bp.Name != "test.md" {
		t.Errorf("expected blueprint name 'test.md', got '%s'", bp.Name)
	}

	if bp.Entrypoint != "main" {
		t.Errorf("expected entrypoint 'main', got '%s'", bp.Entrypoint)
	}

	if bp.RunsOn != "" {
		t.Errorf("expected empty RunsOn (use default), got '%s'", bp.RunsOn)
	}

	if len(bp.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(bp.Jobs))
	}

	mainJob, exists := bp.Jobs["main"]
	if !exists {
		t.Fatal("expected 'main' job")
	}

	if mainJob.Name != "main" {
		t.Errorf("expected job name 'main', got '%s'", mainJob.Name)
	}

	if len(mainJob.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(mainJob.Steps))
	}

	step := mainJob.Steps[0]
	if step.GetName() != "prompt" {
		t.Errorf("expected step name 'prompt', got '%s'", step.GetName())
	}
}

func TestParser_Parse_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(txtFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := parser.New(tmpDir)
	_, err := p.Parse(txtFile)

	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}

	expectedMsg := "unsupported file format: .txt"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestParser_ParseYAML_StillWorking(t *testing.T) {
	tmpDir := t.TempDir()
	ymlFile := filepath.Join(tmpDir, "workflow.yml")

	content := `name: test
jobs:
  main:
    steps:
      - name: step1
        uses: prompt.md
`

	if err := os.WriteFile(ymlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create the referenced prompt file
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("Test prompt"), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	p := parser.New(tmpDir)
	tree, err := p.Parse(ymlFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify YAML parsing still works
	if tree.Blueprint.Name != "test" {
		t.Errorf("expected blueprint name 'test', got '%s'", tree.Blueprint.Name)
	}

	if len(tree.Blueprint.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(tree.Blueprint.Jobs))
	}

	// Verify agent was parsed
	if len(tree.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(tree.Agents))
	}
}
