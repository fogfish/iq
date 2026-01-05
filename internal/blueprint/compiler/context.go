//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/blueprint/runtime"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/goccy/go-yaml"
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

	// LastEmitContext captures the final emit context from workflow execution
	LastEmitContext *EmitContext
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

func (c *WorkflowContext) SetStepFromEvent(name string, evt runtime.Event) {
	for k, v := range evt.Steps {
		c.Steps[k] = v
	}
	c.Current = evt.Current
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

func (c *WorkflowContext) ToEvent() runtime.Event {
	steps := make(map[string]runtime.Gist, len(c.Steps))
	for k, v := range c.Steps {
		steps[k] = anyToGist(v)
	}

	return runtime.Event{
		Document: anyToGist(c.Input),
		Current:  anyToGist(c.Current),
		Steps:    steps,
	}
}

func anyToGist(in any) runtime.Gist {
	switch v := in.(type) {
	case string:
		return runtime.Text(v)
	case []byte:
		return runtime.Text(v)
	case map[string]any:
		return runtime.Json(v)
	case runtime.Text:
		return v
	case runtime.Json:
		return v
	default:
		panic(fmt.Sprintf("unsupported runtime.Gist type: %T", v))
	}
}

type emitContextKey struct{}
type emitCaptureKey struct{}

// EmitContextCapture is a mutable struct that can capture emit context during execution
type EmitContextCapture struct {
	Captured *EmitContext
}

// WithEmitCapture adds an emit capture struct to context
func WithEmitCapture(ctx context.Context) (context.Context, *EmitContextCapture) {
	capture := &EmitContextCapture{}
	return context.WithValue(ctx, emitCaptureKey{}, capture), capture
}

// GetEmitCapture retrieves the emit capture from context
func GetEmitCapture(ctx context.Context) *EmitContextCapture {
	if capture, ok := ctx.Value(emitCaptureKey{}).(*EmitContextCapture); ok {
		return capture
	}
	return nil
}

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

type cacheContextKey struct{}

// CacheContext provides step-level output caching configuration.
type CacheContext struct {
	// Storage for reading/writing cached outputs
	Storage storage.Storage

	// Enabled indicates whether caching is active
	Enabled bool

	// DocumentKey is the current document's key (for computing cache keys)
	DocumentKey iosystem.Key
}

// WithCacheContext adds cache context to ctx.
func WithCacheContext(ctx context.Context, cache *CacheContext) context.Context {
	return context.WithValue(ctx, cacheContextKey{}, cache)
}

// GetCacheContext retrieves cache context from ctx.
func GetCacheContext(ctx context.Context) *CacheContext {
	if cc, ok := ctx.Value(cacheContextKey{}).(*CacheContext); ok {
		return cc
	}
	return nil
}

// TryLoadCached attempts to load cached output for a step.
// Returns (output, true) if cache hit, (nil, false) if cache miss.
func (cc *CacheContext) TryLoadCached(ctx context.Context, emit string) (any, bool) {
	if !cc.Enabled || emit == "" || cc.Storage == nil {
		return nil, false
	}

	// Compute cache key using emit and document key
	cacheKey := ApplyEmit(emit, cc.DocumentKey)

	// Check if cached output exists
	exists, err := cc.Storage.Has(ctx, cacheKey)
	if err != nil || !exists {
		return nil, false
	}

	// Load cached content
	reader, err := cc.Storage.Get(ctx, cacheKey)
	if err != nil {
		return nil, false
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, false
	}

	// Try to parse as JSON first, then YAML, otherwise return as string
	var result any
	if err := json.Unmarshal(content, &result); err == nil {
		return result, true
	}

	if err := yaml.Unmarshal(content, &result); err == nil {
		return result, true
	}

	// Return as string if not structured data
	return string(content), true
}

// SaveCached saves step output to cache.
func (cc *CacheContext) SaveCached(ctx context.Context, emit string, output any) error {
	if !cc.Enabled || emit == "" || cc.Storage == nil {
		return nil
	}

	// Compute cache key
	cacheKey := ApplyEmit(emit, cc.DocumentKey)

	// Serialize output
	var content []byte
	var err error

	switch v := output.(type) {
	case string:
		content = []byte(v)
	case []byte:
		content = v
	default:
		// Try JSON first, fall back to YAML
		content, err = json.MarshalIndent(output, "", "  ")
		if err != nil {
			content, err = yaml.Marshal(output)
			if err != nil {
				return fmt.Errorf("failed to serialize output: %w", err)
			}
		}
	}

	// Save to storage
	return cc.Storage.Put(ctx, cacheKey, bytes.NewReader(content))
}
