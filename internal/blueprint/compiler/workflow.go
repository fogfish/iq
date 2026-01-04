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
	"fmt"
	"os/exec"
	"text/template"
	"time"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/progress"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

// Workflow represents a compiled, executable workflow
type Workflow struct {
	Name       string
	About      string
	Entrypoint string // Optional: default job name, or "main" if empty
	Schema     ast.SchemaNode
	Jobs       map[string]*Job
}

// Job represents a compiled job with executable steps
type Job struct {
	Name  string
	Steps []Step
}

// Step is an interface for executable steps
type Step interface {
	Prompt(ctx context.Context, opt ...chatter.Opt) error
	GetOutputName() string
}

func (job *Job) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error) {
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		ctx = NewWorkflowContext(ctx, input)
	}

	// Initialize emit context if not present
	emitCtx := GetEmitContext(ctx)
	if emitCtx == nil {
		ctx = WithEmitContext(ctx, &EmitContext{})
	}

	totalSteps := len(job.Steps)
	for i, step := range job.Steps {
		stepInfo := progress.StepInfo{
			JobName:    job.Name,
			StepName:   step.GetOutputName(),
			StepNum:    i + 1,
			TotalSteps: totalSteps,
		}
		stepCtx := progress.WithStepInfo(ctx, stepInfo)

		if err := step.Prompt(stepCtx, opt...); err != nil {
			return nil, fmt.Errorf("step %d failed: %w", i, err)
		}
	}

	wfCtx = GetWorkflowContext(ctx)
	return wfCtx.Current, nil
}

//------------------------------------------------------------------------------

// AgentStep is a simple agent execution step
type AgentStep struct {
	Agent      *Agent
	OutputName string
	Emit       string
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (step *AgentStep) GetOutputName() string {
	return step.OutputName
}

// Prompt executes the agent
func (step *AgentStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Set emit context for this step
	if step.Emit != "" {
		emitCtx := GetEmitContext(ctx)
		if emitCtx == nil {
			emitCtx = &EmitContext{}
		}
		// Create a new context with updated emit prefix
		newEmitCtx := &EmitContext{
			Prefix:   step.Emit,
			Counters: emitCtx.Counters, // Preserve parent counters (for nested foreach)
		}
		ctx = WithEmitContext(ctx, newEmitCtx)
		
		// Store in workflow context so it can be retrieved later
		wfCtx.LastEmitContext = newEmitCtx
		
		// Also capture it in the mutable capture struct if present
		if capture := GetEmitCapture(ctx); capture != nil {
			capture.Captured = newEmitCtx
		}
	}

	// Get progress reporter and step info
	reporter := progress.FromContext(ctx)
	stepInfo := progress.GetStepInfo(ctx)

	// Check cache if emit is set and caching is enabled
	cacheCtx := GetCacheContext(ctx)
	if cacheCtx != nil && step.Emit != "" {
		if cachedOutput, found := cacheCtx.TryLoadCached(ctx, step.Emit); found {
			// Cache hit - use cached output and skip LLM
			if reporter != nil && stepInfo != nil {
				reporter.StepSkipped(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, "using cached output")
			}

			// Store output in context
			if step.OutputName != "" {
				wfCtx.SetStepOutput(step.OutputName, cachedOutput)
			} else {
				wfCtx.Current = cachedOutput
			}

			return nil
		}
	}

	startTime := time.Now()

	if reporter != nil && stepInfo != nil {
		reporter.StepStart(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps)
	}

	var result any
	var err error

	// Retry logic
	for i := range step.Retry.Attempts {
		if i > 0 && reporter != nil && step.Retry.Attempts > 1 {
			reporter.RetryAttempt(i+1, step.Retry.Attempts, time.Duration(step.Retry.Delay)*time.Second)
		}

		result, err = step.Agent.Prompt(ctx, wfCtx.ToMap(), opt...)
		if err == nil {
			if i > 0 && reporter != nil {
				reporter.RetrySuccess(i + 1)
			}
			break
		}
		if i < step.Retry.Attempts-1 {
			time.Sleep(time.Duration(step.Retry.Delay) * time.Second)
		}
	}

	if err != nil {
		if reporter != nil && stepInfo != nil {
			reporter.StepError(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, err)
		}
		if step.Retry.Attempts > 1 && reporter != nil {
			reporter.RetryExhausted(step.Retry.Attempts)
		}
		return fmt.Errorf("all attempts failed: %w", err)
	}

	// Report step complete
	if reporter != nil && stepInfo != nil {
		duration := time.Since(startTime)
		// TODO: Track actual token usage from LLM response
		reporter.StepComplete(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, duration, 0)
	}

	// Save to cache if emit is set and caching is enabled
	if cacheCtx != nil && step.Emit != "" {
		if err := cacheCtx.SaveCached(ctx, step.Emit, result); err != nil {
			// Log error but don't fail the step
			if reporter != nil {
				// TODO: Add a method to report cache save errors
			}
		}
	}

	// Store output in context
	if step.OutputName != "" {
		wfCtx.SetStepOutput(step.OutputName, result)
	} else {
		wfCtx.Current = result
	}

	return nil
}

//------------------------------------------------------------------------------

// RouterStep is a conditional routing step
type RouterStep struct {
	Agent      *Agent
	OutputName string
	Emit       string
	RouteNodes []ast.RouteNode // For reference
	Conditions []cel.Program   // Compiled CEL expressions
	Routes     map[string]*Job // Resolved job references
	DefaultJob string          // Default job name
	Default    *Job            // Resolved default job
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (step *RouterStep) GetOutputName() string {
	return step.OutputName
}

func (step *RouterStep) prompt(ctx context.Context, opt ...chatter.Opt) (any, error) {
	if step.Agent == nil {
		return nil, nil
	}

	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return nil, fmt.Errorf("workflow context not found in context")
	}

	for i := range step.Retry.Attempts {
		reply, err := step.Agent.Prompt(ctx, wfCtx.ToMap(), opt...)
		if err == nil {
			return reply, nil
		}
		if i < step.Retry.Attempts-1 {
			time.Sleep(time.Duration(step.Retry.Delay) * time.Second)
		}
	}
	return nil, fmt.Errorf("all attempts failed")
}

// Prompt executes the router: runs agent, evaluates conditions, routes to appropriate job
func (step *RouterStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Set emit context for this step
	if step.Emit != "" {
		emitCtx := GetEmitContext(ctx)
		if emitCtx == nil {
			emitCtx = &EmitContext{}
		}
		// Create a new context with updated emit prefix
		newEmitCtx := &EmitContext{
			Prefix:   step.Emit,
			Counters: emitCtx.Counters, // Preserve parent counters (for nested foreach)
		}
		ctx = WithEmitContext(ctx, newEmitCtx)
		// Capture emit context for retrieval
		if capture := GetEmitCapture(ctx); capture != nil {
			capture.Captured = newEmitCtx
		}
	}

	reporter := progress.FromContext(ctx)

	// Check cache if emit is set and caching is enabled
	cacheCtx := GetCacheContext(ctx)
	var choice any
	var err error
	var usedCache bool

	if cacheCtx != nil && step.Emit != "" {
		if cachedOutput, found := cacheCtx.TryLoadCached(ctx, step.Emit); found {
			// Cache hit - use cached choice
			choice = cachedOutput
			usedCache = true

			if reporter != nil {
				stepInfo := progress.GetStepInfo(ctx)
				if stepInfo != nil {
					reporter.StepSkipped(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, "using cached output")
				}
			}
		}
	}

	if !usedCache {
		if reporter != nil {
			reporter.RouterEvaluating()
		}

		choice, err = step.prompt(ctx, opt...)
		if err != nil {
			return fmt.Errorf("router agent failed: %w", err)
		}

		// Save to cache if emit is set and caching is enabled
		if cacheCtx != nil && step.Emit != "" {
			if err := cacheCtx.SaveCached(ctx, step.Emit, choice); err != nil {
				// Log error but don't fail the step
			}
		}
	}

	// Evaluate conditions in order
	// Pass full workflow context to CEL expressions
	for i, condition := range step.Conditions {
		result, _, err := condition.Eval(map[string]any{
			"choice":   choice,
			"state":    wfCtx.State,
			"steps":    wfCtx.Steps,
			"document": wfCtx.Input,
		})
		if err != nil {
			return fmt.Errorf("failed to evaluate route condition '%s': %w",
				step.RouteNodes[i].When, err)
		}

		// Check if condition matches
		if matches, ok := result.Value().(bool); ok && matches {
			routeName := step.RouteNodes[i].Route
			job := step.Routes[routeName]
			if job == nil {
				return fmt.Errorf("route '%s' not resolved", routeName)
			}

			if reporter != nil {
				reporter.RouterMatched(routeName, job.Name)
			}

			savedCurrent := wfCtx.Current

			jobResult, err := job.Prompt(ctx, savedCurrent, opt...)
			if err != nil {
				return err
			}

			if step.OutputName != "" {
				wfCtx.SetStepOutput(step.OutputName, jobResult)
			} else {
				wfCtx.Current = jobResult
			}

			return nil
		}
	}

	// No route matched, use default
	if step.Default != nil {
		if reporter != nil {
			reporter.RouterDefault(step.Default.Name)
		}

		savedCurrent := wfCtx.Current

		jobResult, err := step.Default.Prompt(ctx, savedCurrent, opt...)
		if err != nil {
			return err
		}

		if step.OutputName != "" {
			wfCtx.SetStepOutput(step.OutputName, jobResult)
		} else {
			wfCtx.Current = jobResult
		}

		return nil
	}

	// No route matched at all
	if reporter != nil {
		reporter.RouterNoMatch()
	}

	return fmt.Errorf("no matching route for choice: %v", choice)
}

//------------------------------------------------------------------------------

// ForeachStep executes a job for each item in an array
type ForeachStep struct {
	UsesAgent  *Agent      // Optional: agent to generate array
	Selector   cel.Program // Optional: CEL program to extract array
	Job        *Job        // Job to execute for each item (resolved)
	JobName    string      // Job name for resolution
	OutputName string
	Emit       string
	Retry      *Retry
	Formatter  Formatter // Output serialization format
}

// GetOutputName returns the name to store output under
func (step *ForeachStep) GetOutputName() string {
	return step.OutputName
}

// Prompt executes the foreach step
func (step *ForeachStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Set emit context for foreach
	parentEmitCtx := GetEmitContext(ctx)
	foreachEmitCtx := &EmitContext{
		Prefix:   step.Emit,
		Counters: make([]int, 0),
	}
	// Inherit parent counters if in nested foreach
	if parentEmitCtx != nil {
		foreachEmitCtx.Counters = append([]int{}, parentEmitCtx.Counters...)
	}
	// Capture emit context for retrieval
	if capture := GetEmitCapture(ctx); capture != nil {
		capture.Captured = foreachEmitCtx
	}

	reporter := progress.FromContext(ctx)
	startTime := time.Now()

	// Get or generate the array
	var items []any
	if step.UsesAgent != nil {
		// Check cache for array generation if emit is set
		cacheCtx := GetCacheContext(ctx)
		var result any
		var err error
		var usedCache bool

		if cacheCtx != nil && step.Emit != "" {
			if cachedOutput, found := cacheCtx.TryLoadCached(ctx, step.Emit); found {
				result = cachedOutput
				usedCache = true

				if reporter != nil {
					stepInfo := progress.GetStepInfo(ctx)
					if stepInfo != nil {
						reporter.StepSkipped(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, "using cached array")
					}
				}
			}
		}

		if !usedCache {
			for i := range step.Retry.Attempts {
				result, err = step.UsesAgent.Prompt(ctx, wfCtx.ToMap(), opt...)
				if err == nil {
					break
				}
				if i < step.Retry.Attempts-1 {
					time.Sleep(time.Duration(step.Retry.Delay) * time.Second)
				}
			}

			if err != nil {
				return fmt.Errorf("failed to generate array: %w", err)
			}

			// Save to cache if emit is set
			if cacheCtx != nil && step.Emit != "" {
				if err := cacheCtx.SaveCached(ctx, step.Emit, result); err != nil {
					// Log error but don't fail
				}
			}
		}

		if arr, ok := result.([]any); ok {
			items = arr
		} else {
			return fmt.Errorf("agent result is not an array: %T", result)
		}
	} else {
		var sourceData any = wfCtx.Current

		// Apply selector if provided
		if step.Selector != nil {
			celVars := map[string]any{
				"input":    wfCtx.Input,
				"current":  wfCtx.Current,
				"state":    wfCtx.State,
				"steps":    wfCtx.Steps,
				"document": wfCtx.Input,
			}

			result, _, err := step.Selector.Eval(celVars)
			if err != nil {
				return fmt.Errorf("selector evaluation failed: %w", err)
			}

			sourceData = result.Value()
		}

		// Ensure result is an array
		if arr, ok := sourceData.([]any); ok {
			items = arr
		} else {
			return fmt.Errorf("selector result is not an array: %T", sourceData)
		}
	}

	if reporter != nil {
		reporter.ForeachStart(len(items))
		// Enable foreach mode to suppress individual step reporting
		reporter.SetForeachMode(true)
	}

	// Execute job for each item
	results := make([]any, 0, len(items))
	successCount := 0
	for i, item := range items {
		// Convert []byte to string to avoid JSON marshaling issues
		// When splitters create arrays, they often produce []byte which
		// gets marshaled as [byte, byte, ...] instead of a string
		if bytes, ok := item.([]byte); ok {
			item = string(bytes)
		}

		// Push iteration counter to emit context (1-based indexing)
		iterEmitCtx := &EmitContext{
			Prefix:   foreachEmitCtx.Prefix,
			Counters: append([]int{}, foreachEmitCtx.Counters...),
		}
		iterEmitCtx.PushCounter(i + 1)

		// Create a new workflow context that inherits parent context but uses item as current
		// This preserves state, steps, and original input while making item the current value
		//lint:ignore SA1029 due to cross-package context key access
		itemCtx := context.WithValue(ctx, workflowContextKey, &WorkflowContext{
			Input:   wfCtx.Input, // Preserve original workflow input
			State:   wfCtx.State, // Preserve shared workflow state
			Steps:   wfCtx.Steps, // Preserve named step outputs
			Current: item,        // Set current item for this iteration
		})
		// Set emit context with counter for this iteration
		itemCtx = WithEmitContext(itemCtx, iterEmitCtx)

		result, err := step.Job.Prompt(itemCtx, item, opt...)
		if err != nil {
			if reporter != nil {
				reporter.ForeachItem(i+1, len(items), fmt.Sprintf("❌ item-%d → failed: %v", i+1, err))
			}
			return fmt.Errorf("foreach iteration %d failed: %w", i, err)
		}

		if reporter != nil {
			reporter.ForeachItem(i+1, len(items), fmt.Sprintf("✅ item-%d → completed", i+1))
		}
		successCount++
		results = append(results, result)
	}

	// Disable foreach mode after processing all items
	if reporter != nil {
		reporter.SetForeachMode(false)
	}

	// Report foreach complete
	if reporter != nil {
		reporter.ForeachComplete(successCount, len(items), time.Since(startTime))
	}

	// Format results using configured formatter
	formattedResults, err := step.Formatter.Format(results)
	if err != nil {
		return fmt.Errorf("failed to format results: %w", err)
	}

	// Store results in context
	if step.OutputName != "" {
		wfCtx.SetStepOutput(step.OutputName, formattedResults)
	} else {
		wfCtx.Current = formattedResults
	}

	return nil
}

type Retry struct {
	Attempts int
	Delay    int
	Yield    *Agent
}

//------------------------------------------------------------------------------

// RunStep executes a shell command
type RunStep struct {
	Command    string // Compiled template for the command
	Shell      string // Shell to use (sh, bash, zsh, etc.)
	OutputName string
	Emit       string
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (step *RunStep) GetOutputName() string {
	return step.OutputName
}

// Prompt executes the shell command
func (step *RunStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Set emit context for this step
	if step.Emit != "" {
		emitCtx := GetEmitContext(ctx)
		if emitCtx == nil {
			emitCtx = &EmitContext{}
		}
		// Create a new context with updated emit prefix
		newEmitCtx := &EmitContext{
			Prefix:   step.Emit,
			Counters: emitCtx.Counters, // Preserve parent counters (for nested foreach)
		}
		ctx = WithEmitContext(ctx, newEmitCtx)
		// Capture emit context for retrieval
		if capture := GetEmitCapture(ctx); capture != nil {
			capture.Captured = newEmitCtx
		}
	}

	reporter := progress.FromContext(ctx)
	stepInfo := progress.GetStepInfo(ctx)

	// Check cache if emit is set and caching is enabled
	cacheCtx := GetCacheContext(ctx)
	if cacheCtx != nil && step.Emit != "" {
		if cachedOutput, found := cacheCtx.TryLoadCached(ctx, step.Emit); found {
			// Cache hit - use cached output
			if reporter != nil && stepInfo != nil {
				reporter.StepSkipped(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, "using cached output")
			}

			// Store output in context
			if step.OutputName != "" {
				wfCtx.SetStepOutput(step.OutputName, cachedOutput)
			} else {
				wfCtx.Current = cachedOutput
			}

			return nil
		}
	}

	startTime := time.Now()

	if reporter != nil && stepInfo != nil {
		reporter.StepStart(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps)
	}

	var result string
	var err error

	// Retry logic
	for i := range step.Retry.Attempts {
		if i > 0 && reporter != nil && step.Retry.Attempts > 1 {
			reporter.RetryAttempt(i+1, step.Retry.Attempts, time.Duration(step.Retry.Delay)*time.Second)
		}

		result, err = step.executeCommand(ctx, wfCtx)
		if err == nil {
			if i > 0 && reporter != nil {
				reporter.RetrySuccess(i + 1)
			}
			break
		}
		if i < step.Retry.Attempts-1 {
			time.Sleep(time.Duration(step.Retry.Delay) * time.Second)
		}
	}

	if err != nil {
		if reporter != nil && stepInfo != nil {
			reporter.StepError(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, err)
		}
		if step.Retry.Attempts > 1 && reporter != nil {
			reporter.RetryExhausted(step.Retry.Attempts)
		}
		return fmt.Errorf("all attempts failed: %w", err)
	}

	// Report step complete
	if reporter != nil && stepInfo != nil {
		duration := time.Since(startTime)
		reporter.StepComplete(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, duration, 0)
	}

	// Save to cache if emit is set and caching is enabled
	if cacheCtx != nil && step.Emit != "" {
		if err := cacheCtx.SaveCached(ctx, step.Emit, result); err != nil {
			// Log error but don't fail the step
		}
	}

	// Store output in context
	if step.OutputName != "" {
		wfCtx.SetStepOutput(step.OutputName, result)
	} else {
		wfCtx.Current = result
	}

	return nil
}

func (step *RunStep) executeCommand(ctx context.Context, wfCtx *WorkflowContext) (string, error) {
	// Render command template with workflow context
	tmpl, err := template.New("run").Parse(step.Command)
	if err != nil {
		return "", fmt.Errorf("failed to parse command template: %w", err)
	}

	var cmdBuf bytes.Buffer
	if err := tmpl.Execute(&cmdBuf, wfCtx.ToMap()); err != nil {
		return "", fmt.Errorf("failed to render command template: %w", err)
	}
	rendered := cmdBuf.String()

	// Execute command using exec package
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, step.Shell, "-c", rendered)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include stderr in error message
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("command failed: %w\nCommand: %s\nStderr: %s", err, rendered, stderrStr)
		}
		return "", fmt.Errorf("command failed: %w\nCommand: %s", err, rendered)
	}

	return stdout.String(), nil
}
