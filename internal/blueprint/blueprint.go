package blueprint

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/blueprint/parser"
	"github.com/kshard/chatter"
)

// Blueprint represents a compiled workflow
type Blueprint struct {
	workflow *compiler.Workflow
}

// New loads and compiles a blueprint file
func New(file string, llm chatter.Chatter) (*Blueprint, error) {
	// Phase 1: Parse YAML to AST
	p := parser.New(".")
	tree, err := p.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Phase 2: Compile AST to executable workflow
	comp, err := compiler.New(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create compiler: %w", err)
	}

	workflow, err := comp.Compile(context.Background(), tree)
	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	return &Blueprint{
		workflow: workflow,
	}, nil
}

// Run executes the entrypoint job (or "main" if no entrypoint specified)
func (bp *Blueprint) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error) {
	// Determine which job to run
	jobName := bp.workflow.Entrypoint
	if jobName == "" {
		jobName = "main"
	}

	job, exists := bp.workflow.Jobs[jobName]
	if !exists {
		if bp.workflow.Entrypoint == "" {
			return nil, fmt.Errorf("no entrypoint specified and no 'main' job found")
		}
		return nil, fmt.Errorf("entrypoint job '%s' not found in workflow", jobName)
	}

	return job.Prompt(ctx, input, opt...)
}

// Jobs returns all available job names
func (bp *Blueprint) Jobs() []string {
	names := make([]string, 0, len(bp.workflow.Jobs))
	for name := range bp.workflow.Jobs {
		names = append(names, name)
	}
	return names
}

// GetJob returns a compiled job that implements the AI interface
func (bp *Blueprint) GetJob(name string) (*compiler.Job, error) {
	job, exists := bp.workflow.Jobs[name]
	if !exists {
		return nil, fmt.Errorf("job '%s' not found", name)
	}
	return job, nil
}
