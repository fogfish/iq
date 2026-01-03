//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package ast

import "github.com/google/jsonschema-go/jsonschema"

// Well-known context keys for workflow execution
const (
	// ContextKeyDocument is the original workflow input (the source document/data being processed)
	ContextKeyDocument = "document"

	// ContextKeyInput is the agent's input (alias for current step value, used in agent templates)
	ContextKeyInput = "input"

	// ContextKeyCurrent is the most recent step output (changes with each step)
	ContextKeyCurrent = "current"

	// ContextKeySteps holds named outputs from previous steps
	ContextKeySteps = "steps"

	// ContextKeyState is a shared key-value store for workflow data
	ContextKeyState = "state"
)

// AST represents the complete workflow abstract syntax tree
type AST struct {
	Blueprint *BlueprintNode
	Agents    map[string]*AgentNode // Keyed by file path
}

// BlueprintNode represents the root workflow definition
type BlueprintNode struct {
	Name       string
	About      string // Optional: description of the blueprint
	Entrypoint string // Optional: default job to run. If empty, uses job named "main"
	RunsOn     string
	Schema     SchemaNode // Input/Output schemas
	Jobs       map[string]*JobNode
}

// JobNode represents a job within a blueprint
type JobNode struct {
	Name   string
	RunsOn string
	Steps  []StepNode
}

// StepNode is an interface for different step types
type StepNode interface {
	stepNode()
	GetName() string
	GetRunsOn() string
	GetUses() string
	GetRetry() *RetryNode
	GetOutput() string
}

// AgentStepNode represents a simple agent execution step
type AgentStepNode struct {
	Name   string
	RunsOn string
	Uses   string // Path to agent file
	Output string // Optional name to store output in context
	Retry  *RetryNode
}

func (n *AgentStepNode) stepNode()            {}
func (n *AgentStepNode) GetName() string      { return n.Name }
func (n *AgentStepNode) GetUses() string      { return n.Uses }
func (n *AgentStepNode) GetRetry() *RetryNode { return n.Retry }
func (n *AgentStepNode) GetOutput() string    { return n.Output }
func (n *AgentStepNode) GetRunsOn() string    { return n.RunsOn }

// RouterStepNode represents a conditional routing step
type RouterStepNode struct {
	Name    string
	RunsOn  string
	Uses    string      // Path to agent file
	Output  string      // Optional name to store output in context
	Routes  []RouteNode // Ordered routes (first match wins)
	Default string      // Default job name if no route matches
	Retry   *RetryNode
}

func (n *RouterStepNode) stepNode()            {}
func (n *RouterStepNode) GetName() string      { return n.Name }
func (n *RouterStepNode) GetUses() string      { return n.Uses }
func (n *RouterStepNode) GetRetry() *RetryNode { return n.Retry }
func (n *RouterStepNode) GetOutput() string    { return n.Output }
func (n *RouterStepNode) GetRunsOn() string    { return n.RunsOn }

// RouteNode represents a single conditional route
type RouteNode struct {
	When  string // CEL expression
	Route string // Target job name
}

// ForeachStepNode represents an array iteration step
type ForeachStepNode struct {
	Name     string
	RunsOn   string
	Uses     string      // Optional: Path to agent file to generate array
	Selector string      // Optional: CEL expression to extract array from input
	Job      string      // Job to execute for each array item
	Output   string      // Optional name to store results array in context
	Format   *FormatNode // Optional: output serialization format
	Retry    *RetryNode
}

func (n *ForeachStepNode) stepNode()            {}
func (n *ForeachStepNode) GetName() string      { return n.Name }
func (n *ForeachStepNode) GetUses() string      { return n.Uses }
func (n *ForeachStepNode) GetRetry() *RetryNode { return n.Retry }
func (n *ForeachStepNode) GetOutput() string    { return n.Output }
func (n *ForeachStepNode) GetRunsOn() string    { return n.RunsOn }

// RunStepNode represents a shell command execution step
type RunStepNode struct {
	Name   string
	RunsOn string // Shell to use (bash, zsh, sh); defaults to "sh"
	Run    string // Shell command with template variables
	Output string // Optional name to store output in context
	Retry  *RetryNode
}

func (n *RunStepNode) stepNode()            {}
func (n *RunStepNode) GetName() string      { return n.Name }
func (n *RunStepNode) GetUses() string      { return "" }
func (n *RunStepNode) GetRetry() *RetryNode { return n.Retry }
func (n *RunStepNode) GetOutput() string    { return n.Output }
func (n *RunStepNode) GetRunsOn() string    { return n.RunsOn }

type RetryNode struct {
	Attempts int    // Number of retry attempts
	Delay    int    // Delay between attempts in seconds
	Yield    string // Path to agent file if all retries fail
}

// FormatNode configures output serialization after foreach
type FormatNode struct {
	Type      string // "json", "jsonl", "text" (default: "json")
	Delimiter string // For text format (default: "\n")
}

// AgentNode represents an agent definition
type AgentNode struct {
	Name    string
	RunsOn  string
	Format  string       // "json" or empty
	Schema  SchemaNode   // Input/Output schemas
	Servers []ServerNode // MCP servers
	Prompt  string       // Template text
}

type SchemaNode struct {
	Input *jsonschema.Schema // JSON Schema for input validation
	Reply *jsonschema.Schema // JSON Schema for output validation
}

// ServerNode represents an MCP server configuration
type ServerNode struct {
	Type    string
	Name    string
	Command []string
	Url     string
}

// Validate performs semantic validation on the AST
func (ast *AST) Validate() error {
	// Will be implemented by compiler
	return nil
}

// JobNames returns all job names in the blueprint
func (b *BlueprintNode) JobNames() []string {
	names := make([]string, 0, len(b.Jobs))
	for name := range b.Jobs {
		names = append(names, name)
	}
	return names
}
