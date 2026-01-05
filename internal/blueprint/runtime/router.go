package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

type Router struct {
	Nodes       []ast.RouteNode
	DefaultNode string
	Prompter    Prompter
	Conditions  []cel.Program
	Routes      map[string]Prompter
	Unknown     Prompter
}

var _ Prompter = (*Router)(nil)

func NewRouter(nodes []ast.RouteNode, def string, prompter Prompter, conditions []cel.Program) *Router {
	return &Router{
		Nodes:       nodes,
		DefaultNode: def,
		Prompter:    prompter,
		Conditions:  conditions,
		Routes:      make(map[string]Prompter),
	}
}

func (r *Router) Config(jobs map[string]*Job) error {
	for _, route := range r.Nodes {
		r.Routes[route.Route] = jobs[route.Route]
	}
	if r.DefaultNode != "" {
		r.Unknown = jobs[r.DefaultNode]
	}
	return nil
}

func (r *Router) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	choice := in.Current
	if r.Prompter != nil {
		c, err := r.Prompter.Prompt(ctx, in, opts...)
		if err != nil {
			return in, fmt.Errorf("router agent failed: %w", err)
		}
		choice = c.Current
	}

	variables := map[string]any{
		ast.ContextKeyDocument: in.Document,
		ast.ContextKeyCurrent:  in.Current,
		ast.ContextKeyInput:    in.Current,
		ast.ContextKeySteps:    in.Steps,
		ast.ContextKeyChoice:   choice,
	}

	for i, condition := range r.Conditions {
		result, _, err := condition.Eval(variables)
		if err != nil {
			return in, fmt.Errorf("failed to evaluate route condition '%s': %w",
				r.Nodes[i].When, err)
		}

		// Check if condition matches
		if matches, ok := result.Value().(bool); ok && matches {
			routeName := r.Nodes[i].Route
			job := r.Routes[routeName]
			if job == nil {
				return in, fmt.Errorf("route '%s' not resolved", routeName)
			}

			jobResult, err := job.Prompt(ctx, in, opts...)
			if err != nil {
				return in, err
			}

			return jobResult, nil
		}
	}

	if r.Unknown != nil {
		jobResult, err := r.Unknown.Prompt(ctx, in, opts...)
		if err != nil {
			return in, err
		}

		return jobResult, nil
	}

	return in, fmt.Errorf("no matching route for choice: %v", choice)
}
