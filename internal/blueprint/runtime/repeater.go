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
	"time"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/kshard/chatter"
)

type Repeater struct {
	Node     *ast.RetryNode
	fallback Prompter

	Prompter
}

var _ Prompter = (*Repeater)(nil)

func NewRepeater(node *ast.RetryNode, p Prompter) *Repeater {
	return &Repeater{Node: node, Prompter: p}
}

func (r *Repeater) Config(jobs map[string]*Job) error {
	// if r.Node.Yield != "" {
	// 	r.Prompter = jobs[r.Node.Yield]
	// }
	return nil
}

func (r *Repeater) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	// stepInfo := progress.GetStepInfo(ctx)
	for i := range r.Node.Attempts {
		// stepInfo.Attempt = i + 1
		// stepInfo.Delay = r.delay
		// ctx = progress.WithStepInfo(ctx, *stepInfo)

		result, err := r.Prompter.Prompt(ctx, evt, opts...)
		if err == nil {
			return result, nil
		}

		if i < r.Node.Attempts-1 {
			time.Sleep(time.Duration(r.Node.Delay) * time.Second)
		}
	}

	if r.fallback != nil {
		result, err := r.fallback.Prompt(ctx, evt, opts...)
		if err != nil {
			return evt, err
		}

		return result, nil
	}

	return evt, fmt.Errorf("all attempts failed")
}
