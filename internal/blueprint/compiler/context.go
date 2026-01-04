//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/iosystem"
)

const (
	// workflowContextKey is the key for workflow context in context.Context
	workflowContextKey = "iq.workflow"
)

// WorkflowContext holds the execution state for a workflow
type WorkflowContext struct {
	// Input is the original workflow input
	Input any

	// State is a shared key-value store for workflow data
	State map[string]any

	// Steps holds named outputs from previous steps
	Steps map[string]any

	// Current is the most recent step output (for chaining)
	Current any
}

// NewWorkflowContext creates a new workflow context embedded in Go context
func NewWorkflowContext(ctx context.Context, input any) context.Context {
	switch v := input.(type) {
	case []byte:
		input = string(v)
	}

	wfCtx := &WorkflowContext{
		Input:   input,
		State:   make(map[string]any),
		Steps:   make(map[string]any),
		Current: input,
	}

	//lint:ignore SA1029 due to cross-package context key access
	return context.WithValue(ctx, workflowContextKey, wfCtx)
}

// GetWorkflowContext extracts the workflow context from Go context
func GetWorkflowContext(ctx context.Context) *WorkflowContext {
	if wfCtx, ok := ctx.Value(workflowContextKey).(*WorkflowContext); ok {
		return wfCtx
	}
	return nil
}

// Set stores a value in the workflow state
func (c *WorkflowContext) Set(key string, value any) {
	c.State[key] = value
}

// Get retrieves a value from the workflow state
func (c *WorkflowContext) Get(key string) (any, bool) {
	val, ok := c.State[key]
	return val, ok
}

// SetStepOutput stores the output of a named step
func (c *WorkflowContext) SetStepOutput(name string, output any) {
	c.Steps[name] = output
	c.Current = output
}

// GetStepOutput retrieves the output of a named step
func (c *WorkflowContext) GetStepOutput(name string) (any, bool) {
	val, ok := c.Steps[name]
	return val, ok
}

// ToMap returns the full context as a map for template rendering
func (c *WorkflowContext) ToMap() map[string]any {
	return map[string]any{
		ast.ContextKeyDocument: c.Input,   // Original workflow input
		ast.ContextKeyCurrent:  c.Current, // Current step value
		ast.ContextKeySteps:    c.Steps,   // Named step outputs
		ast.ContextKeyState:    c.State,   // Shared workflow state
		// Note: .input is added as alias in agent.encodeStruct()
	}
}

type emitContextKey struct{}

// EmitContext tracks output key prefixes for workflow steps.
type EmitContext struct {
	// Emit prefix for current step
	Prefix string

	// Foreach iteration counters (stacked for nested loops)
	// Example: [1, 5] means first loop iteration 1, nested loop iteration 5
	Counters []int
}

// WithEmitContext adds emit context to ctx.
func WithEmitContext(ctx context.Context, emit *EmitContext) context.Context {
	return context.WithValue(ctx, emitContextKey{}, emit)
}

// GetEmitContext retrieves emit context from ctx.
func GetEmitContext(ctx context.Context) *EmitContext {
	if ec, ok := ctx.Value(emitContextKey{}).(*EmitContext); ok {
		return ec
	}
	return &EmitContext{}
}

// ApplyEmit transforms input key by adding emit prefix.
// Examples:
//
//	emit="summary", key="a.txt" → "summary/a.txt"
//	emit="", key="a.txt" → "a.txt" (no change)
func ApplyEmit(emit string, key iosystem.Key) iosystem.Key {
	if emit == "" {
		return key
	}
	return iosystem.Key(emit + "/" + string(key))
}

// ApplyEmitWithCounters adds emit prefix and foreach counters to key.
// Examples:
//
//	emit="research", key="a.txt", counters=[1] → "research/a.000001.txt"
//	emit="research", key="a.txt", counters=[1,5] → "research/a.000001.000005.txt"
func ApplyEmitWithCounters(emit string, key iosystem.Key, counters []int) iosystem.Key {
	keyStr := string(key)

	// Build counter suffix
	suffix := ""
	for _, counter := range counters {
		suffix += fmt.Sprintf(".%06d", counter)
	}

	// Insert suffix before file extension (if present)
	if suffix != "" {
		// Find the last dot for extension
		lastDot := -1
		for i := len(keyStr) - 1; i >= 0; i-- {
			if keyStr[i] == '.' {
				lastDot = i
				break
			}
			// Stop searching if we hit a directory separator
			if keyStr[i] == '/' {
				break
			}
		}

		if lastDot > 0 {
			// Insert suffix before extension
			keyStr = keyStr[:lastDot] + suffix + keyStr[lastDot:]
		} else {
			// No extension, append suffix
			keyStr = keyStr + suffix
		}
	}

	// Apply emit prefix
	if emit == "" {
		return iosystem.Key(keyStr)
	}

	return iosystem.Key(emit + "/" + keyStr)
}

// PushCounter adds a foreach iteration counter to the stack.
func (ec *EmitContext) PushCounter(iteration int) {
	ec.Counters = append(ec.Counters, iteration)
}

// PopCounter removes the last foreach counter from the stack.
func (ec *EmitContext) PopCounter() {
	if len(ec.Counters) > 0 {
		ec.Counters = ec.Counters[:len(ec.Counters)-1]
	}
}
