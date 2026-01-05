//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package blueprint

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/blueprint/parser"
	"github.com/fogfish/iq/internal/blueprint/runtime"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kshard/chatter"
)

// Blueprint represents a compiled workflow
type Blueprint struct {
	workflow        *compiler.Workflow
	lastEmitContext *compiler.EmitContext // Captured from last execution
}

// LastEmitContext returns the emit context from the last execution
func (bp *Blueprint) LastEmitContext() *compiler.EmitContext {
	return bp.lastEmitContext
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

func (bp *Blueprint) Name() string  { return bp.workflow.Name }
func (bp *Blueprint) About() string { return bp.workflow.About }
func (bp *Blueprint) Schema() (*jsonschema.Schema, *jsonschema.Schema) {
	return bp.workflow.Schema.Input, bp.workflow.Schema.Reply
}

// Workflow returns the compiled workflow (for internal use, e.g., skip-if-exists)
func (bp *Blueprint) Workflow() *compiler.Workflow {
	return bp.workflow
}

// JobCount returns the number of jobs in the workflow
func (bp *Blueprint) JobCount() int {
	return len(bp.workflow.Jobs)
}

// StepCount returns the total number of steps across all jobs
func (bp *Blueprint) StepCount() int {
	count := 0
	for _, job := range bp.workflow.Jobs {
		count += len(job.Steps)
	}
	return count
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

	val, err := runtime.ToGist(input)
	if err != nil {
		return nil, fmt.Errorf("input conversion failed: %w", err)
	}

	evt := runtime.Event{
		Document: val,
		Current:  val,
		Steps:    make(map[string]runtime.Gist),
	}

	result, err := job.Prompt(ctx, evt, opt...)

	// Capture LastEmitContext from the workflow context if present
	// Note: We can't get the modified context back from job.Prompt because
	// Go contexts are immutable. But we can store it in a side channel.
	// For now, this won't work until we refactor job.Prompt to return context.

	return result.Current, err
}
