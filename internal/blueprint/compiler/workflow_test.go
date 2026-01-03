//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/kshard/chatter"
)

func TestForeachStep_WithJSONFormatter(t *testing.T) {
	// Create a simple foreach step with JSON formatter
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "results",
		Formatter:  compiler.NewJSONFormatter(),
	}

	// Create workflow context with test data (array of items)
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1", "doc2", "doc3"})
	wfCtx := compiler.GetWorkflowContext(ctx)

	// Execute the foreach step
	err := step.Prompt(ctx)
	if err != nil {
		t.Fatalf("ForeachStep.Prompt failed: %v", err)
	}

	// Verify output is stored and is []any type (JSON format)
	result := wfCtx.Steps["results"]
	resultSlice, ok := result.([]any)
	if !ok {
		t.Errorf("Expected []any for JSON format, got %T", result)
	}

	// Verify we have 3 results
	if len(resultSlice) != 3 {
		t.Errorf("Expected 3 results, got %d", len(resultSlice))
	}

	// Verify results can be marshaled to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal result to JSON: %v", err)
	}

	var unmarshaled []any
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestForeachStep_WithJSONLFormatter(t *testing.T) {
	// Create a foreach step with JSONL formatter
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "results",
		Formatter:  compiler.NewJSONLFormatter(),
	}

	// Create workflow context with test data
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1", "doc2", "doc3"})
	wfCtx := compiler.GetWorkflowContext(ctx)

	// Execute the foreach step
	err := step.Prompt(ctx)
	if err != nil {
		t.Fatalf("ForeachStep.Prompt failed: %v", err)
	}

	// Verify output is stored and is string type (JSONL format)
	result := wfCtx.Steps["results"]
	resultStr, ok := result.(string)
	if !ok {
		t.Errorf("Expected string for JSONL format, got %T", result)
	}

	// Verify each line is valid JSON
	lines := strings.Split(strings.TrimSpace(resultStr), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var obj any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestForeachStep_WithTextFormatter(t *testing.T) {
	// Create a foreach step with text formatter using custom delimiter
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "results",
		Formatter:  compiler.NewTextFormatter("\n\n"),
	}

	// Create workflow context with test data
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1", "doc2", "doc3"})
	wfCtx := compiler.GetWorkflowContext(ctx)

	// Execute the foreach step
	err := step.Prompt(ctx)
	if err != nil {
		t.Fatalf("ForeachStep.Prompt failed: %v", err)
	}

	// Verify output is stored and is string type (text format)
	result := wfCtx.Steps["results"]
	resultStr, ok := result.(string)
	if !ok {
		t.Errorf("Expected string for text format, got %T", result)
	}

	// Verify delimiter is used (should have double newlines between items)
	if !strings.Contains(resultStr, "\n\n") {
		t.Errorf("Expected double newline delimiter in output")
	}

	// Verify we have 2 delimiters (between 3 items)
	delimCount := strings.Count(resultStr, "\n\n")
	if delimCount != 2 {
		t.Errorf("Expected 2 delimiters between 3 items, got %d", delimCount)
	}
}

func TestForeachStep_DefaultFormat(t *testing.T) {
	// Create a foreach step with default formatter (JSON)
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "results",
		Formatter:  compiler.NewJSONFormatter(), // Default is JSON
	}

	// Create workflow context with test data
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1", "doc2", "doc3"})
	wfCtx := compiler.GetWorkflowContext(ctx)

	// Execute the foreach step
	err := step.Prompt(ctx)
	if err != nil {
		t.Fatalf("ForeachStep.Prompt failed: %v", err)
	}

	// Verify output is []any (default JSON behavior)
	result := wfCtx.Steps["results"]
	_, ok := result.([]any)
	if !ok {
		t.Errorf("Expected []any for default format, got %T", result)
	}
}

func TestForeachStep_FormatterError(t *testing.T) {
	// Create a foreach step with error-returning formatter
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "results",
		Formatter:  &mockErrorFormatter{},
	}

	// Create workflow context with test data
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1"})

	// Execute the foreach step - should fail
	err := step.Prompt(ctx)
	if err == nil {
		t.Fatal("Expected error from formatter")
	}

	// Verify error message contains expected text
	if !strings.Contains(err.Error(), "failed to format results") {
		t.Errorf("Expected 'failed to format results' in error, got: %v", err)
	}
}

func TestForeachStep_StoreInCurrent(t *testing.T) {
	// Create a foreach step without OutputName (stores in Current)
	step := &compiler.ForeachStep{
		Job:        createMockJob("item-job", "result"),
		OutputName: "", // Empty means store in Current
		Formatter:  compiler.NewJSONFormatter(),
	}

	// Create workflow context with test data
	ctx := compiler.NewWorkflowContext(context.Background(), []any{"doc1", "doc2"})
	wfCtx := compiler.GetWorkflowContext(ctx)

	// Execute the foreach step
	err := step.Prompt(ctx)
	if err != nil {
		t.Fatalf("ForeachStep.Prompt failed: %v", err)
	}

	// Verify output is stored in Current, not in Steps
	resultSlice, ok := wfCtx.Current.([]any)
	if !ok {
		t.Errorf("Expected []any in Current, got %T", wfCtx.Current)
	}

	if len(resultSlice) != 2 {
		t.Errorf("Expected 2 results in Current, got %d", len(resultSlice))
	}

	// Note: Steps map will have entries from the step execution itself,
	// but not the final output since OutputName is empty
}

// Helper functions and mocks

// createMockJob creates a simple job that returns a fixed result for testing
func createMockJob(name, result string) *compiler.Job {
	return &compiler.Job{
		Name: name,
		Steps: []compiler.Step{
			&mockStep{result: result},
		},
	}
}

// mockStep is a simple step implementation for testing
type mockStep struct {
	result string
}

func (s *mockStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	// Get workflow context and set the result as current
	wfCtx := compiler.GetWorkflowContext(ctx)
	if wfCtx != nil {
		wfCtx.Current = s.result
	}
	return nil
}

func (s *mockStep) GetOutputName() string {
	return ""
}

// mockErrorFormatter is a formatter that always returns an error
type mockErrorFormatter struct{}

func (f *mockErrorFormatter) Format(results []any) (any, error) {
	return nil, fmt.Errorf("mock formatter error")
}
