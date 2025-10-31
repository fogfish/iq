package compiler

import (
	"context"
	"fmt"
	"time"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

// Workflow represents a compiled, executable workflow
type Workflow struct {
	Name       string
	Entrypoint string // Optional: default job name, or "main" if empty
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

func (j *Job) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error) {
	// Check if we already have a workflow context (sub-job call from router)
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		// Create new workflow context and embed in Go context
		ctx = NewWorkflowContext(ctx, input)
	}

	// Execute all steps
	for i, step := range j.Steps {
		if err := step.Prompt(ctx, opt...); err != nil {
			return nil, fmt.Errorf("step %d failed: %w", i, err)
		}
	}

	// Extract final output from context
	wfCtx = GetWorkflowContext(ctx)
	return wfCtx.Current, nil
}

// AgentStep is a simple agent execution step
type AgentStep struct {
	Agent      *Agent
	OutputName string
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (s *AgentStep) GetOutputName() string {
	return s.OutputName
}

// Prompt executes the agent
func (s *AgentStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	// Extract workflow context from Go context
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	var result any
	var err error

	// Retry logic
	for i := range s.Retry.Attempts {
		// Pass the full context map to agent templates
		result, err = s.Agent.Prompt(ctx, wfCtx.ToMap(), opt...)
		if err == nil {
			break
		}
		if i < s.Retry.Attempts-1 {
			time.Sleep(time.Duration(s.Retry.Delay) * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("all attempts failed: %w", err)
	}

	// Store output in context
	if s.OutputName != "" {
		wfCtx.SetStepOutput(s.OutputName, result)
	} else {
		wfCtx.Current = result
	}

	return nil
}

// RouterStep is a conditional routing step
type RouterStep struct {
	Agent      *Agent
	OutputName string
	RouteNodes []ast.RouteNode // For reference
	Conditions []cel.Program   // Compiled CEL expressions
	Routes     map[string]*Job // Resolved job references
	DefaultJob string          // Default job name
	Default    *Job            // Resolved default job
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (r *RouterStep) GetOutputName() string {
	return r.OutputName
}

func (r *RouterStep) prompt(ctx context.Context, opt ...chatter.Opt) (any, error) {
	// Extract workflow context
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return nil, fmt.Errorf("workflow context not found in context")
	}

	for i := range r.Retry.Attempts {
		// Pass the full context map to agent templates
		reply, err := r.Agent.Prompt(ctx, wfCtx.ToMap(), opt...)
		if err == nil {
			return reply, nil
		}
		if i < r.Retry.Attempts-1 {
			time.Sleep(time.Duration(r.Retry.Delay) * time.Second)
		}
	}
	return nil, fmt.Errorf("all attempts failed")
}

// Prompt executes the router: runs agent, evaluates conditions, routes to appropriate job
func (r *RouterStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	// Extract workflow context
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Execute agent to get choice
	choice, err := r.prompt(ctx, opt...)
	if err != nil {
		return fmt.Errorf("router agent failed: %w", err)
	}

	// Evaluate conditions in order
	for i, condition := range r.Conditions {
		result, _, err := condition.Eval(map[string]any{
			"choice": choice,
		})
		if err != nil {
			return fmt.Errorf("failed to evaluate route condition '%s': %w",
				r.RouteNodes[i].When, err)
		}

		// Check if condition matches
		if matches, ok := result.Value().(bool); ok && matches {
			routeName := r.RouteNodes[i].Route
			job := r.Routes[routeName]
			if job == nil {
				return fmt.Errorf("route '%s' not resolved", routeName)
			}

			// Save current value before job execution
			savedCurrent := wfCtx.Current

			// Execute the routed job (it will modify wfCtx.Current)
			jobResult, err := job.Prompt(ctx, savedCurrent, opt...)
			if err != nil {
				return err
			}

			// Store output in context
			if r.OutputName != "" {
				wfCtx.SetStepOutput(r.OutputName, jobResult)
			} else {
				wfCtx.Current = jobResult
			}

			return nil
		}
	}

	// No route matched, use default
	if r.Default != nil {
		// Save current value before job execution
		savedCurrent := wfCtx.Current

		jobResult, err := r.Default.Prompt(ctx, savedCurrent, opt...)
		if err != nil {
			return err
		}

		// Store output in context
		if r.OutputName != "" {
			wfCtx.SetStepOutput(r.OutputName, jobResult)
		} else {
			wfCtx.Current = jobResult
		}

		return nil
	}

	return fmt.Errorf("no matching route for choice: %v", choice)
}

// ForeachStep executes a job for each item in an array
type ForeachStep struct {
	UsesAgent  *Agent // Optional: agent to generate array
	Job        *Job   // Job to execute for each item (resolved)
	JobName    string // Job name for resolution
	OutputName string
	Retry      *Retry
}

// GetOutputName returns the name to store output under
func (s *ForeachStep) GetOutputName() string {
	return s.OutputName
}

// Prompt executes the foreach step
func (s *ForeachStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
	// Extract workflow context
	wfCtx := GetWorkflowContext(ctx)
	if wfCtx == nil {
		return fmt.Errorf("workflow context not found in context")
	}

	// Get or generate the array
	var items []any
	if s.UsesAgent != nil {
		// Generate array using the agent
		var result any
		var err error

		for i := range s.Retry.Attempts {
			result, err = s.UsesAgent.Prompt(ctx, wfCtx.ToMap(), opt...)
			if err == nil {
				break
			}
			if i < s.Retry.Attempts-1 {
				time.Sleep(time.Duration(s.Retry.Delay) * time.Second)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to generate array: %w", err)
		}

		// Convert result to array
		if arr, ok := result.([]any); ok {
			items = arr
		} else {
			return fmt.Errorf("agent result is not an array: %T", result)
		}
	} else {
		// Use current value from context
		if arr, ok := wfCtx.Current.([]any); ok {
			items = arr
		} else {
			return fmt.Errorf("current value is not an array: %T", wfCtx.Current)
		}
	}

	// Execute job for each item
	results := make([]any, 0, len(items))
	for i, item := range items {
		result, err := s.Job.Prompt(ctx, item, opt...)
		if err != nil {
			return fmt.Errorf("foreach iteration %d failed: %w", i, err)
		}
		results = append(results, result)
	}

	// Store results in context
	if s.OutputName != "" {
		wfCtx.SetStepOutput(s.OutputName, results)
	} else {
		wfCtx.Current = results
	}

	return nil
}

type Retry struct {
	Attempts int
	Delay    int
	Yield    *Agent
}
