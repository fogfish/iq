package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/progress"
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
	Node       *ast.RouterStepNode
	Prompter   Prompter
	Conditions []cel.Program
	Routes     map[string]Prompter
	Default    Prompter
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
		r.Default = jobs[r.Node.Default]
	}
	return nil
}

func (r *Router) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	reporter := progress.FromContext(ctx)
	reporter.RouterEvaluating()

	route, choice, err := r.evalChoice(ctx, in, opts...)
	switch {
	case err != nil:
		return in, err
	case route == "" && r.Default == nil:
		reporter.RouterNoMatch()
		return in, fmt.Errorf("no matching route for choice: %v", choice)
	case route == "":
		return r.evalDefault(ctx, in, opts...)
	default:
		return r.evalRoute(ctx, route, in, opts...)
	}
}

func (r *Router) evalChoice(ctx context.Context, in Event, opts ...chatter.Opt) (string, Gist, error) {
	choice := in.Current
	if r.Prompter != nil {
		c, err := r.Prompter.Prompt(ctx, in, opts...)
		if err != nil {
			return "", nil, fmt.Errorf("router agent failed: %w", err)
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
			return "", nil, fmt.Errorf("failed to evaluate route condition '%s': %w", route.When, err)
		}

		if matches, ok := result.Value().(bool); ok && matches {
			return route.Route, choice, nil
		}
	}

	return "", choice, nil
}

func (r *Router) evalDefault(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	reporter := progress.FromContext(ctx)
	reporter.RouterDefault(r.Node.Default)

	result, err := r.Default.Prompt(progress.ContextRouter(ctx, r.Node.Default), in, opts...)
	if err != nil {
		return in, err
	}

	return result, nil
}

func (r *Router) evalRoute(ctx context.Context, route string, in Event, opts ...chatter.Opt) (Event, error) {
	job := r.Routes[route]
	if job == nil {
		return in, fmt.Errorf("route '%s' not resolved", route)
	}

	reporter := progress.FromContext(ctx)
	reporter.RouterMatched(route, route)

	jobResult, err := job.Prompt(progress.ContextRouter(ctx, route), in, opts...)
	if err != nil {
		return in, err
	}

	return jobResult, nil
}
