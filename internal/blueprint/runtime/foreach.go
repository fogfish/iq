package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/cel-go/cel"
	"github.com/kshard/chatter"
)

type ForEach struct {
	Node     string
	selector cel.Program
	prompter Prompter
}

var _ Prompter = (*ForEach)(nil)

func NewForEach(node string, selector cel.Program) *ForEach {
	return &ForEach{Node: node, selector: selector}
}

func (f *ForEach) Config(jobs map[string]*Job) error {
	if f.Node != "" {
		f.prompter = jobs[f.Node]
	}

	return nil
}

func (f *ForEach) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	var list List

	if f.selector != nil {
		val, _, err := f.selector.Eval(map[string]any{
			ast.ContextKeyDocument: in.Document,
			ast.ContextKeyInput:    in.Current,
			ast.ContextKeyCurrent:  in.Current,
			ast.ContextKeySteps:    in.Steps,
		})
		if err != nil {
			return in, fmt.Errorf("selector evaluation failed: %w", err)
		}
		switch v := val.Value().(type) {
		case List:
			list = v
		default:
			return in, fmt.Errorf("selector must return a list, got %T", val)
		}
	}

	if len(list) == 0 {
		switch v := in.Current.(type) {
		case List:
			list = v
		default:
			return in, fmt.Errorf("foreach input must be a list, got %T", in.Current)
		}
	}

	output := make(List, len(list))
	for i, item := range list {
		val, err := ToGist(item)
		if err != nil {
			return in, fmt.Errorf("foreach item conversion failed: %w", err)
		}

		evt := Event{
			Document: in.Document,
			Current:  val,
			Steps:    in.Steps,
		}
		reply, err := f.prompter.Prompt(ctx, evt, opts...)
		if err != nil {
			return in, fmt.Errorf("foreach item processing failed: %w", err)
		}
		// TODO: merge steps
		// in.Steps = reply.Steps
		output[i] = reply.Current
	}

	return in.copy(output), nil
}
