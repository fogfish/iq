package compiler

import (
	"context"

	"github.com/fogfish/iq/internal/blueprint/ast"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey int

const (
	// workflowContextKey is the key for workflow context in context.Context
	workflowContextKey contextKey = iota
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
