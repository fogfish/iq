//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
)

// Parser handles YAML to AST conversion
type Parser struct {
	baseDir string // Base directory for resolving relative paths
}

// New creates a new parser
func New(baseDir string) *Parser {
	return &Parser{baseDir: baseDir}
}

// Parse parses a file (YAML or Markdown) and returns unified AST
func (p *Parser) Parse(file string) (*ast.AST, error) {
	ext := filepath.Ext(file)

	switch ext {
	case ".yml", ".yaml":
		return p.parseYAML(file)
	case ".md", ".markdown":
		return p.parseMarkdown(file)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (expected .yml, .yaml, .md, .markdown)", ext)
	}
}

// parseYAML parses a blueprint YAML file and all referenced agents
func (p *Parser) parseYAML(file string) (*ast.AST, error) {
	yaml, blueprint, err := p.ParseBlueprint(file)
	if err != nil {
		return nil, err
	}

	// cascade-down runs-on settings
	for _, job := range blueprint.Jobs {
		if blueprint.RunsOn == "" {
			blueprint.RunsOn = "base"
		}

		if job.RunsOn == "" {
			job.RunsOn = blueprint.RunsOn
		}

		for _, step := range job.Steps {
			switch s := step.(type) {
			case *ast.AgentStepNode:
				if s.RunsOn == "" {
					s.RunsOn = job.RunsOn
				}
			case *ast.RouterStepNode:
				if s.RunsOn == "" {
					s.RunsOn = job.RunsOn
				}
			case *ast.ForeachStepNode:
				if s.RunsOn == "" {
					s.RunsOn = job.RunsOn
				}
			}
		}
	}

	// Collect all agent file references
	agentFiles := make(map[string]bool)
	for _, job := range blueprint.Jobs {
		for _, step := range job.Steps {
			if step.GetUses() != "" {
				agentFiles[step.GetUses()] = true
			}

			retry := step.GetRetry()
			if retry != nil && retry.Yield != "" {
				agentFiles[retry.Yield] = true
			}
		}
	}

	// Parse all agents
	agents := make(map[string]*ast.AgentNode)
	for agentFile := range agentFiles {
		agentPath := p.resolvePath(agentFile)
		agent, err := p.ParseAgent(agentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent %s: %w", agentFile, err)
		}
		agents[agentFile] = agent
	}

	// Parse all inline agents
	for jobID, job := range yaml.Jobs {
		for stepID, step := range job.Steps {
			if step.Prompt != "" {
				syntheticPath := fmt.Sprintf("__inline_%s_%d__", jobID, stepID)
				syntheticNode := &ast.AgentNode{
					Name:   syntheticPath,
					RunsOn: step.RunsOn,
					Schema: ast.SchemaNode{},
					Prompt: step.Prompt,
				}
				agents[syntheticPath] = syntheticNode

				step := blueprint.Jobs[jobID].Steps[stepID]
				switch s := step.(type) {
				case *ast.AgentStepNode:
					s.Uses = syntheticPath
				case *ast.RouterStepNode:
					s.Uses = syntheticPath
				case *ast.ForeachStepNode:
					s.Uses = syntheticPath
				}
			}
		}
	}

	return &ast.AST{
		Blueprint: blueprint,
		Agents:    agents,
	}, nil
}

// parseMarkdown parses a standalone Markdown file as a single-step workflow
func (p *Parser) parseMarkdown(file string) (*ast.AST, error) {
	agent, err := p.ParseAgent(file)
	if err != nil {
		return nil, err
	}

	synthetic := &ast.BlueprintNode{
		Name:       filepath.Base(file),
		Entrypoint: "main",
		Jobs: map[string]*ast.JobNode{
			"main": {
				Name: "main",
				Steps: []ast.StepNode{
					&ast.AgentStepNode{
						Name: "prompt",
						Uses: file,
					},
				},
			},
		},
	}

	// Return AST with inline agent
	return &ast.AST{
		Blueprint: synthetic,
		Agents: map[string]*ast.AgentNode{
			file: agent,
		},
	}, nil
}

// ParseBlueprint parses a blueprint YAML file
func (p *Parser) ParseBlueprint(file string) (*blueprintYAML, *ast.BlueprintNode, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open blueprint: %w", err)
	}
	defer fd.Close()

	var raw blueprintYAML
	if err := yaml.NewDecoder(fd).Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse blueprint YAML: %w", err)
	}

	// Update baseDir based on blueprint file location
	p.baseDir = filepath.Dir(file)

	return &raw, p.convertBlueprint(&raw), nil
}

// ParseAgent parses an agent definition file
func (p *Parser) ParseAgent(file string) (*ast.AgentNode, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open agent file: %w", err)
	}
	defer fd.Close()

	content, err := io.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file: %w", err)
	}

	return p.parseAgentContent(filepath.Base(file), content)
}

// parseAgentContent parses agent content (YAML frontmatter + prompt OR pure markdown)
func (p *Parser) parseAgentContent(name string, content []byte) (*ast.AgentNode, error) {
	// Check if content starts with YAML frontmatter
	if bytes.HasPrefix(content, []byte("---\n")) {
		// Has frontmatter delimiter at start
		parts := bytes.Split(content[4:], []byte("\n---\n")) // Skip first "---\n"

		switch len(parts) {
		case 1:
			// Only starting delimiter, no ending delimiter - treat as pure prompt
			return &ast.AgentNode{
				Name:   name,
				Prompt: string(content),
				Format: "", // text format
			}, nil

		case 2:
			// Has frontmatter and prompt
			var raw agentYAML
			if err := yaml.Unmarshal(parts[0], &raw); err != nil {
				return nil, fmt.Errorf("failed to parse agent frontmatter: %w", err)
			}

			agent := p.convertAgent(&raw)
			agent.Name = name
			agent.Prompt = string(parts[1])
			return agent, nil

		default:
			return nil, fmt.Errorf("invalid agent format: expected 1 '\\n---\\n' separator after frontmatter, got %d", len(parts)-1)
		}
	}

	// Pure markdown - no frontmatter
	return &ast.AgentNode{
		Name:   name,
		Prompt: string(content),
		Format: "", // text format
	}, nil
}

// resolvePath resolves relative paths based on baseDir
func (p *Parser) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(p.baseDir, path)
}

// YAML structure definitions (internal)

type blueprintYAML struct {
	Name       string             `yaml:"name,omitempty"`
	About      string             `yaml:"about,omitempty"`
	Entrypoint string             `yaml:"entrypoint,omitempty"`
	RunsOn     string             `yaml:"runs-on,omitempty"`
	Schema     *schemaYAML        `yaml:"schema,omitempty"`
	Jobs       map[string]jobYAML `yaml:"jobs,omitempty"`
}

type jobYAML struct {
	RunsOn string     `yaml:"runs-on,omitempty"`
	Steps  []stepYAML `yaml:"steps,omitempty"`
}

type stepYAML struct {
	Name    string       `yaml:"name,omitempty"`
	RunsOn  string       `yaml:"runs-on,omitempty"`
	Uses    string       `yaml:"uses,omitempty"`
	Output  string       `yaml:"output,omitempty"`
	Switch  []routeYAML  `yaml:"switch,omitempty"`
	Default string       `yaml:"default,omitempty"`
	Foreach *foreachYAML `yaml:"foreach,omitempty"`
	Retry   *retryYAML   `yaml:"retry,omitempty"`
	Prompt  string       `yaml:"prompt,omitempty"`
}

type foreachYAML struct {
	Job string `yaml:"job,omitempty"`
}

type routeYAML struct {
	When  string `yaml:"when,omitempty"`
	Route string `yaml:"route,omitempty"`
}

type retryYAML struct {
	Attempts int    `yaml:"attempts,omitempty"`
	Delay    int    `yaml:"delay,omitempty"` // in seconds
	Yield    string `yaml:"yield,omitempty"`
}

type agentYAML struct {
	Name    string       `yaml:"name,omitempty"`
	Format  string       `yaml:"format,omitempty"`
	Schema  *schemaYAML  `yaml:"schema,omitempty"`
	Servers []serverYAML `yaml:"servers,omitempty"`
}

type schemaYAML struct {
	Input map[string]any `yaml:"input,omitempty"`
	Reply map[string]any `yaml:"reply,omitempty"`
}

type serverYAML struct {
	Type    string   `yaml:"type,omitempty"`
	Name    string   `yaml:"name,omitempty"`
	Command []string `yaml:"command,omitempty"`
	Url     string   `yaml:"url,omitempty"`
}

// Conversion functions

func (p *Parser) convertBlueprint(raw *blueprintYAML) *ast.BlueprintNode {
	jobs := make(map[string]*ast.JobNode)
	for name, jobYAML := range raw.Jobs {
		jobs[name] = p.convertJob(name, &jobYAML)
	}

	var inputSchema, replySchema *jsonschema.Schema
	if raw.Schema != nil {
		inputSchema = p.convertSchema(raw.Schema.Input)
		replySchema = p.convertSchema(raw.Schema.Reply)
	}

	return &ast.BlueprintNode{
		Name:       raw.Name,
		About:      raw.About,
		Entrypoint: raw.Entrypoint,
		RunsOn:     raw.RunsOn,
		Schema: ast.SchemaNode{
			Input: inputSchema,
			Reply: replySchema,
		},
		Jobs: jobs,
	}
}

func (p *Parser) convertJob(name string, raw *jobYAML) *ast.JobNode {
	steps := make([]ast.StepNode, 0, len(raw.Steps))
	for _, stepYAML := range raw.Steps {
		steps = append(steps, p.convertStep(&stepYAML))
	}

	return &ast.JobNode{
		Name:   name,
		RunsOn: raw.RunsOn,
		Steps:  steps,
	}
}

func (p *Parser) convertStep(raw *stepYAML) ast.StepNode {
	var retry *ast.RetryNode
	if raw.Retry != nil {
		retry = &ast.RetryNode{
			Attempts: raw.Retry.Attempts,
			Delay:    raw.Retry.Delay,
			Yield:    raw.Retry.Yield,
		}
	}

	// Check if this is a foreach step
	if raw.Foreach != nil {
		return &ast.ForeachStepNode{
			Name:   raw.Name,
			RunsOn: raw.RunsOn,
			Uses:   raw.Uses,
			Job:    raw.Foreach.Job,
			Output: raw.Output,
			Retry:  retry,
		}
	}

	// Check if this is a router step
	if len(raw.Switch) > 0 {
		routes := make([]ast.RouteNode, 0, len(raw.Switch))
		for _, routeYAML := range raw.Switch {
			routes = append(routes, ast.RouteNode{
				When:  routeYAML.When,
				Route: routeYAML.Route,
			})
		}

		return &ast.RouterStepNode{
			Name:    raw.Name,
			RunsOn:  raw.RunsOn,
			Uses:    raw.Uses,
			Output:  raw.Output,
			Routes:  routes,
			Default: raw.Default,
			Retry:   retry,
		}
	}

	// Simple agent step
	return &ast.AgentStepNode{
		Name:   raw.Name,
		RunsOn: raw.RunsOn,
		Uses:   raw.Uses,
		Output: raw.Output,
		Retry:  retry,
	}
}

func (p *Parser) convertAgent(raw *agentYAML) *ast.AgentNode {
	servers := make([]ast.ServerNode, 0, len(raw.Servers))
	for _, srv := range raw.Servers {
		servers = append(servers, ast.ServerNode{
			Type:    srv.Type,
			Name:    srv.Name,
			Command: srv.Command,
			Url:     srv.Url,
		})
	}

	// Handle optional schema
	var inputSchema, replySchema *jsonschema.Schema
	if raw.Schema != nil {
		inputSchema = p.convertSchema(raw.Schema.Input)
		replySchema = p.convertSchema(raw.Schema.Reply)
	}

	return &ast.AgentNode{
		Name:   raw.Name,
		Format: raw.Format,
		Schema: ast.SchemaNode{
			Input: inputSchema,
			Reply: replySchema,
		},
		Servers: servers,
	}
}

func (p *Parser) convertSchema(schema map[string]any) *jsonschema.Schema {
	if schema == nil {
		return nil
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil
	}

	var js jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &js); err != nil {
		return nil
	}

	return &js
}
