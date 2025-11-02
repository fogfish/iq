//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package service

import (
	"context"

	"github.com/fogfish/iq/internal/core"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/prompt/jsonify"
)

type Worker struct {
	*agent.Manifold[*core.Prompt, []byte]
}

func NewWorker(llm chatter.Chatter, registry thinker.Registry) (w *Worker, err error) {
	w = &Worker{}
	w.Manifold = agent.NewManifold(llm,
		codec.FromEncoder(w.encode),
		codec.FromDecoder(w.decode),
		registry,
	)

	return
}

func (w *Worker) encode(in *core.Prompt) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(in.Task)

	if len(in.Blob) > 0 {
		prompt.WithBlob("Input document", in.Blob)
	}

	if in.Format == core.FORMAT_JSON {
		jsonify.Strings.Harden(&prompt, nil)
	}

	return &prompt, nil
}

func (w *Worker) decode(reply *chatter.Reply) (float64, []byte, error) {
	return 1.0, []byte(reply.String()), nil
}

func (w *Worker) PromptOnce(ctx context.Context, input *core.Prompt, opt ...chatter.Opt) ([]byte, error) {
	reply, err := w.Manifold.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (w *Worker) Prompt(ctx context.Context, input *core.Prompt, opt ...chatter.Opt) ([]byte, error) {
	reply, err := w.Manifold.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
