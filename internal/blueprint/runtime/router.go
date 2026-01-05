package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

/*

TODO:

5. Router Context Passing
Potential Issue:

Question: Does the routed job receive the correct event context?

The in Event should have the parent's Steps map
But after routing, should Current be the choice or the original input?
Old code preserved choice separately and passed it in context variables.

*/

type Router struct {
	Node *ast.RouterStepNode
	// Nodes       []ast.RouteNode
	// DefaultNode string
	Prompter   Prompter
	Conditions []cel.Program
	Routes     map[string]Prompter
	Unknown    Prompter
}

var _ Prompter = (*Router)(nil)

func NewRouter(node *ast.RouterStepNode, prompter Prompter, conditions []cel.Program) *Router {
	return &Router{
		Node:       node,
		Prompter:   prompter,
		Conditions: conditions,
		Routes:     make(map[string]Prompter),
	}
}

func (r *Router) Config(jobs map[string]*Job) error {
	for _, route := range r.Node.Routes {
		r.Routes[route.Route] = jobs[route.Route]
	}
	if r.Node.Default != "" {
		r.Unknown = jobs[r.Node.Default]
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
		route := r.Node.Routes[i]
		result, _, err := condition.Eval(variables)
		if err != nil {
			return in, fmt.Errorf("failed to evaluate route condition '%s': %w",
				route.When, err)
		}

		// Check if condition matches
		if matches, ok := result.Value().(bool); ok && matches {
			job := r.Routes[route.Route]
			if job == nil {
				return in, fmt.Errorf("route '%s' not resolved", route.Route)
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
