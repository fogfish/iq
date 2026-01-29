//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/blueprint/runtime"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

// Compiler compiles AST to executable workflow
type Compiler struct {
	llm    chatter.Chatter
	sink   iosystem.Sink
	cache  storage.Storage
	celEnv *cel.Env
}

// Workflow represents a compiled, executable workflow
type Workflow struct {
	Name       string
	About      string
	Entrypoint string // Optional: default job name, or "main" if empty
	Schema     ast.SchemaNode
	Jobs       map[string]*runtime.Job
}

// New creates a new compiler
func New(llm chatter.Chatter, sink iosystem.Sink, cache storage.Storage) (*Compiler, error) {
	// Create CEL environment for route conditions and selector expressions
	// Variables available in CEL expressions:
	// - choice: output from the router agent (router context)
	// - input: workflow input (selector context)
	// - current: current value being processed (selector context)
	// - state: workflow state (map[string]any)
	// - steps: named step outputs (map[string]any)
	// - document: original workflow input (router context)
	env, err := cel.NewEnv(
		cel.Variable(ast.ContextKeyChoice, cel.DynType),
		cel.Variable(ast.ContextKeyInput, cel.DynType),
		cel.Variable(ast.ContextKeyCurrent, cel.DynType),
		cel.Variable(ast.ContextKeySteps, cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable(ast.ContextKeyEnv, cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable(ast.ContextKeyDocument, cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Compiler{
		llm:    llm,
		sink:   sink,
		cache:  cache,
		celEnv: env,
	}, nil
}

// Compile validates AST and compiles to executable workflow
func (c *Compiler) Compile(ctx context.Context, tree *ast.AST) (*Workflow, error) {
	// Phase 1: Semantic validation
	if err := c.validate(tree); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Phase 2: Compile to executable
	workflow, err := c.compile(ctx, tree)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	return workflow, nil
}

// validate performs semantic validation
func (c *Compiler) validate(tree *ast.AST) error {
	bp := tree.Blueprint

	// Validate entrypoint if specified
	if bp.Entrypoint != "" {
		if _, exists := bp.Jobs[bp.Entrypoint]; !exists {
			return fmt.Errorf("entrypoint '%s' references undefined job", bp.Entrypoint)
		}
	}

	// Validate all route references point to existing jobs
	for jobName, job := range bp.Jobs {
		for i, step := range job.Steps {
			// Validate router steps
			if router, ok := step.(*ast.RouterStepNode); ok {
				// Validate routes
				for _, route := range router.Routes {
					if _, exists := bp.Jobs[route.Route]; !exists {
						return fmt.Errorf("job '%s' step %d: route references undefined job '%s'",
							jobName, i, route.Route)
					}

					// Validate CEL expression syntax
					if _, err := c.compileCEL(route.When); err != nil {
						return fmt.Errorf("job '%s' step %d: invalid CEL expression '%s': %w",
							jobName, i, route.When, err)
					}
				}

				// Validate default route if specified
				if router.Default != "" {
					if _, exists := bp.Jobs[router.Default]; !exists {
						return fmt.Errorf("job '%s' step %d: default route references undefined job '%s'",
							jobName, i, router.Default)
					}
				}
			}

			// Validate foreach steps
			if foreach, ok := step.(*ast.ForeachStepNode); ok {
				if _, exists := bp.Jobs[foreach.Job]; !exists {
					return fmt.Errorf("job '%s' step %d: foreach references undefined job '%s'",
						jobName, i, foreach.Job)
				}
			}
		}
	}

	// Validate all agent files were parsed
	for _, job := range bp.Jobs {
		for _, step := range job.Steps {
			// Skip validation for foreach steps without uses
			if uses := step.GetUses(); uses != "" {
				if _, exists := tree.Agents[uses]; !exists {
					return fmt.Errorf("agent file '%s' not found in AST", uses)
				}
			}

			retry := step.GetRetry()
			if retry != nil && retry.Yield != "" {
				if _, exists := tree.Agents[retry.Yield]; !exists {
					return fmt.Errorf("agent file '%s' not found in AST", retry.Yield)
				}
			}
		}
	}

	return nil
}

// compile converts validated AST to executable workflow
func (c *Compiler) compile(ctx context.Context, tree *ast.AST) (*Workflow, error) {
	bp := tree.Blueprint

	jobs := make(map[string]*runtime.Job)
	for jobName, jobNode := range bp.Jobs {
		job, err := c.compileJob(ctx, tree, bp.Name, jobName, jobNode)
		if err != nil {
			return nil, fmt.Errorf("failed to compile job '%s': %w", jobName, err)
		}
		jobs[jobName] = job
	}

	// Resolve job references (now that all jobs are compiled)
	for _, job := range jobs {
		if err := job.Config(jobs); err != nil {
			return nil, fmt.Errorf("job '%s' configuration failed: %w", job.Name, err)
		}
	}

	return &Workflow{
		Name:       bp.Name,
		About:      bp.About,
		Entrypoint: bp.Entrypoint,
		Schema:     bp.Schema,
		Jobs:       jobs,
	}, nil
}

// compileJob compiles a single job
func (c *Compiler) compileJob(ctx context.Context, tree *ast.AST, workflow, jobName string, node *ast.JobNode) (*runtime.Job, error) {
	steps := make([]runtime.Prompter, 0, len(node.Steps))
	for i, stepNode := range node.Steps {
		step, err := c.compileStep(ctx, tree, workflow, jobName, i, stepNode)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return &runtime.Job{
		Name:  jobName,
		Steps: steps,
	}, nil
}

// compileStep compiles a single step
func (c *Compiler) compileStep(ctx context.Context, tree *ast.AST, workflow, job string, stepIndex int, node ast.StepNode) (prompter runtime.Prompter, err error) {
	switch v := node.(type) {
	case *ast.AgentStepNode:
		prompter, err = c.compileAgentNode(ctx, tree, v)
		if err != nil {
			return nil, err
		}
	case *ast.RouterStepNode:
		prompter, err = c.compileRouterNode(ctx, tree, v)
		if err != nil {
			return nil, err
		}
	case *ast.ForeachStepNode:
		prompter, err = c.compileForEach(ctx, tree, v)
		if err != nil {
			return nil, err
		}
	case *ast.RunStepNode:
		prompter, err = c.compileShellNode(ctx, tree, v)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported step type: %+v", node)
	}

	prompter = c.compileCache(ctx, tree, workflow, job, stepIndex, node, prompter)
	prompter = c.compilePrinter(ctx, node, prompter)
	prompter = c.compileRepeater(ctx, node, prompter)
	prompter = c.compileEmitter(ctx, node, prompter)
	prompter = c.compileMemento(ctx, node, prompter)
	return prompter, nil
}

// compileCEL compiles a CEL expression
func (c *Compiler) compileCEL(expr string) (cel.Program, error) {
	parsed, issues := c.celEnv.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	checked, issues := c.celEnv.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	return c.celEnv.Program(checked)

}

func (c *Compiler) compileAgentNode(_ context.Context, tree *ast.AST, node *ast.AgentStepNode) (runtime.Prompter, error) {
	agent, ok := tree.Agents[node.Uses]
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found in AST", node.Uses)
	}

	agent.RunsOn = node.RunsOn

	var manifold runtime.Prompter
	manifold, err := runtime.NewManifold(agent, c.llm)
	if err != nil {
		return nil, fmt.Errorf("manifold compile error: %w", err)
	}

	return manifold, nil
}

func (c *Compiler) compileRouterNode(_ context.Context, tree *ast.AST, node *ast.RouterStepNode) (runtime.Prompter, error) {
	var manifold runtime.Prompter

	if node.Uses != "" {
		agent, ok := tree.Agents[node.Uses]
		if !ok {
			return nil, fmt.Errorf("agent '%s' not found in AST", node.Uses)
		}

		agent.RunsOn = node.RunsOn

		var err error
		manifold, err = runtime.NewManifold(agent, c.llm)
		if err != nil {
			return nil, fmt.Errorf("manifold compile error: %w", err)
		}
	}

	conditions := make([]cel.Program, 0, len(node.Routes))
	for _, route := range node.Routes {
		prog, err := c.compileCEL(route.When)
		if err != nil {
			return nil, fmt.Errorf("failed to compile CEL: %w", err)
		}
		conditions = append(conditions, prog)
	}

	return runtime.NewRouter(node, manifold, conditions), nil
}

func (c *Compiler) compileForEach(_ context.Context, tree *ast.AST, node *ast.ForeachStepNode) (runtime.Prompter, error) {
	// Compile CEL selector if provided
	var selector cel.Program
	if node.Selector != "" {
		prog, err := c.compileCEL(node.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to compile selector '%s': %w", node.Selector, err)
		}
		selector = prog
	}

	return runtime.NewForEach(node, selector)
}

func (c *Compiler) compileShellNode(_ context.Context, tree *ast.AST, node *ast.RunStepNode) (runtime.Prompter, error) {
	shell := node.RunsOn
	if shell == "" {
		shell = "sh"
	}

	return runtime.NewShell(shell, node.Run)
}

func (c *Compiler) compileMemento(_ context.Context, node ast.StepNode, prompter runtime.Prompter) runtime.Prompter {
	variable := node.GetOutput()
	if variable == "" {
		return prompter
	}
	return runtime.NewMemento(node.GetOutput(), prompter)

}

func (c *Compiler) compilePrinter(_ context.Context, node ast.StepNode, prompter runtime.Prompter) runtime.Prompter {
	return runtime.NewPrinter(prompter)
}

func (c *Compiler) compileRepeater(_ context.Context, node ast.StepNode, prompter runtime.Prompter) runtime.Prompter {
	retryNode := node.GetRetry()

	if retryNode == nil || retryNode.Attempts < 2 {
		return prompter
	}

	if retryNode.Attempts > 1 {
		return runtime.NewRepeater(retryNode, prompter)
	}

	return prompter
}

func (c *Compiler) compileEmitter(_ context.Context, node ast.StepNode, prompter runtime.Prompter) runtime.Prompter {
	emit := node.GetEmit()
	if emit == "" || c.sink == nil {
		return prompter
	}

	return runtime.NewEmitter(c.sink, emit, prompter)
}

func (c *Compiler) compileCache(ctx context.Context, tree *ast.AST, workflow, job string, stepIndex int, node ast.StepNode, prompter runtime.Prompter) runtime.Prompter {
	// Automatically enable caching for all steps when cache storage is available
	if c.cache == nil {
		return prompter
	}

	// Extract cacheable content based on node type
	content := c.extractStepContent(tree, node)
	if content == "" {
		return prompter // Not cacheable
	}

	// Calculate SHA256 hash (first 6 hex chars)
	hash := sha256.Sum256([]byte(content))
	contentHash := fmt.Sprintf("%x", hash[:3])

	// Get step name
	stepName := node.GetName()
	if stepName == "" {
		uses := node.GetUses()
		if uses != "" {
			stepName = strings.TrimSuffix(filepath.Base(uses), filepath.Ext(uses))
		}
	}
	if stepName == "" {
		stepName = fmt.Sprintf("step-%d", stepIndex)
	}

	return runtime.NewCache(c.cache, workflow, job, stepName, contentHash, prompter)
}

// extractStepContent extracts the content to be hashed for caching based on step type
func (c *Compiler) extractStepContent(tree *ast.AST, node ast.StepNode) string {
	switch v := node.(type) {
	case *ast.AgentStepNode:
		// Always has uses (required for agent steps)
		if agent, ok := tree.Agents[v.Uses]; ok {
			return agent.Prompt
		}
	case *ast.RouterStepNode:
		// Only cacheable if has uses (router agent)
		if v.Uses != "" {
			if agent, ok := tree.Agents[v.Uses]; ok {
				return agent.Prompt
			}
		}
	case *ast.ForeachStepNode:
		// Container step - not directly cacheable
		// If has uses (array generator), that agent's prompt is cached separately
		return ""
	case *ast.RunStepNode:
		// Not cacheable - commands are cheap and have side effects
		return ""
	}
	return ""
}
