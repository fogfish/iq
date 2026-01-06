//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/progress"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

type ForEach struct {
	Node     *ast.ForeachStepNode
	selector cel.Program
	prompter Prompter
}

var _ Prompter = (*ForEach)(nil)

func NewForEach(node *ast.ForeachStepNode, selector cel.Program) *ForEach {
	return &ForEach{Node: node, selector: selector}
}

func (f *ForEach) Config(jobs map[string]*Job) error {
	if f.Node.Job != "" {
		f.prompter = jobs[f.Node.Job]
	}

	return nil
}

func (f *ForEach) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	list, err := f.evalSelect(in)
	if err != nil {
		return in, fmt.Errorf("foreach selection failed: %w", err)
	}

	reporter := progress.FromContext(ctx)
	reporter.ForeachStart(len(list))
	reporter.SetForeachMode(true)
	defer reporter.SetForeachMode(false)
	defer reporter.ForeachComplete(len(list), len(list), 0)

	output := make(List, len(list))
	for i, item := range list {
		val, err := ToGist(item)
		if err != nil {
			return in, fmt.Errorf("foreach item conversion failed: %w", err)
		}

		reply, err := f.prompter.Prompt(
			progress.ContextForEach(ctx, f.Node.Job, i+1, len(list)),
			in.copy(in.Key.SeqID(i+1), val),
			opts...,
		)
		if err != nil {
			reporter.ForeachItem(i+1, len(list), fmt.Sprintf("❌ %s → failed: %v", f.Node.Job, err))
			return in, fmt.Errorf("foreach item %d processing failed: %w", i+1, err)
		}
		reporter.ForeachItem(i+1, len(list), fmt.Sprintf("✅ %s → completed", f.Node.Job))

		// TODO: merge steps
		// in.Steps = reply.Steps
		output[i] = reply.Current
	}

	return in.copy(in.Key, output), nil
}

func (f *ForEach) evalSelect(in Event) (List, error) {
	var list List

	if f.selector != nil {
		val, _, err := f.selector.Eval(map[string]any{
			ast.ContextKeyDocument: in.Document,
			ast.ContextKeyInput:    in.Current,
			ast.ContextKeyCurrent:  in.Current,
			ast.ContextKeySteps:    in.Steps,
		})
		if err != nil {
			return nil, fmt.Errorf("selector evaluation failed: %w", err)
		}
		switch v := val.Value().(type) {
		case List:
			list = v
		case []any:
			list = List(v)
		default:
			return nil, fmt.Errorf("selector must return a list, got %T", val)
		}
	}

	if len(list) == 0 {
		switch v := in.Current.(type) {
		case List:
			list = v
		default:
			return nil, fmt.Errorf("foreach input must be a list, got %T", in.Current)
		}
	}

	return list, nil
}
