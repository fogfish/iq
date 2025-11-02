package compiler

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

// Compiler compiles AST to executable workflow
type Compiler struct {
	llm    chatter.Chatter
	celEnv *cel.Env
}

// Factory provides dependencies for compilation
type Factory interface {
	LLM(name string) (chatter.Chatter, error)
}

// New creates a new compiler
func New(llm chatter.Chatter) (*Compiler, error) {
	// Create CEL environment for route conditions
	// Variables available in CEL expressions:
	// - choice: output from the router agent
	// - state: workflow state (map[string]any)
	// - steps: named step outputs (map[string]any)
	// - document: original workflow input
	env, err := cel.NewEnv(
		cel.Variable("choice", cel.DynType),
		cel.Variable("state", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("steps", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("document", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Compiler{
		llm:    llm,
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

	// Compile all jobs
	jobs := make(map[string]*Job)
	for jobName, jobNode := range bp.Jobs {
		job, err := c.compileJob(ctx, jobName, jobNode, tree, c.llm)
		if err != nil {
			return nil, fmt.Errorf("failed to compile job '%s': %w", jobName, err)
		}
		jobs[jobName] = job
	}

	// Resolve job references (now that all jobs are compiled)
	for _, job := range jobs {
		for _, step := range job.Steps {
			// Resolve router step references
			if router, ok := step.(*RouterStep); ok {
				router.Routes = make(map[string]*Job)
				for _, route := range router.RouteNodes {
					router.Routes[route.Route] = jobs[route.Route]
				}
				if router.DefaultJob != "" {
					router.Default = jobs[router.DefaultJob]
				}
			}

			// Resolve foreach step references
			if foreach, ok := step.(*ForeachStep); ok {
				foreach.Job = jobs[foreach.JobName]
			}
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
func (c *Compiler) compileJob(ctx context.Context, name string, node *ast.JobNode, tree *ast.AST, sysLLM chatter.Chatter) (*Job, error) {
	// Determine LLM for this job
	llm := c.llm
	// if node.RunsOn != "" {
	// 	var err error
	// 	llm, err = c.factory.LLM(node.RunsOn)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// Compile all steps
	steps := make([]Step, 0, len(node.Steps))
	for i, stepNode := range node.Steps {
		step, err := c.compileStep(ctx, i, stepNode, tree, llm)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return &Job{
		Name:  name,
		Steps: steps,
	}, nil
}

// compileStep compiles a single step
func (c *Compiler) compileStep(ctx context.Context, index int, node ast.StepNode, tree *ast.AST, llm chatter.Chatter) (Step, error) {
	// Prepare retry config if present
	retryNode := &Retry{Attempts: 1}
	retryNAst := node.GetRetry()
	if retryNAst != nil {
		agtRetry := &Agent{Node: tree.Agents[retryNAst.Yield]}
		if err := agtRetry.compile(ctx, llm); err != nil {
			return nil, fmt.Errorf("failed to create retry agent: %w", err)
		}

		retryNode = &Retry{
			Attempts: retryNAst.Attempts,
			Delay:    retryNAst.Delay,
			Yield:    agtRetry,
		}
	}

	// Check if this is a foreach step
	if foreachNode, ok := node.(*ast.ForeachStepNode); ok {
		var usesAgent *Agent
		if foreachNode.Uses != "" {
			agentNode := tree.Agents[foreachNode.Uses]
			usesAgent = &Agent{Node: agentNode}
			if err := usesAgent.compile(ctx, llm); err != nil {
				return nil, fmt.Errorf("failed to initialize uses agent: %w", err)
			}
		}

		return &ForeachStep{
			UsesAgent:  usesAgent,
			JobName:    foreachNode.Job,
			OutputName: foreachNode.GetOutput(),
			Retry:      retryNode,
		}, nil
	}

	// Get agent definition (required for router and agent steps)
	agentNode := tree.Agents[node.GetUses()]

	// Create and initialize agent
	var agt *Agent
	if agentNode != nil {
		agt = &Agent{Node: agentNode}
		if err := agt.compile(ctx, llm); err != nil {
			return nil, fmt.Errorf("failed to initialize agent: %w", err)
		}
	}

	// Check if this is a router step
	if routerNode, ok := node.(*ast.RouterStepNode); ok {
		// Compile CEL expressions
		conditions := make([]cel.Program, 0, len(routerNode.Routes))
		for _, route := range routerNode.Routes {
			prog, err := c.compileCEL(route.When)
			if err != nil {
				return nil, fmt.Errorf("failed to compile CEL: %w", err)
			}
			conditions = append(conditions, prog)
		}

		return &RouterStep{
			Agent:      agt,
			OutputName: routerNode.GetOutput(),
			RouteNodes: routerNode.Routes,
			Conditions: conditions,
			DefaultJob: routerNode.Default,
			Retry:      retryNode,
		}, nil
	}

	// Simple agent step
	return &AgentStep{
		Agent:      agt,
		OutputName: node.GetOutput(),
		Retry:      retryNode,
	}, nil
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
