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
	"os"

	"github.com/fogfish/iq/internal/prompt"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/spf13/viper"
)

type Worker struct {
	*agent.Worker[*viper.Viper]
}

func NewWorker(llm chatter.Chatter, c *viper.Viper, workdir string) (w *Worker, err error) {
	if len(workdir) == 0 {
		workdir, err = os.MkdirTemp(os.TempDir(), "iq-")
		if err != nil {
			return
		}
	}

	registry := command.NewRegistry()
	for _, cmd := range c.GetStringSlice(prompt.YAML_REGISTRY) {
		switch cmd {
		case command.BASH:
			registry.Register(command.Bash("", workdir))
		case command.PYTHON:
			registry.Register(command.Python(workdir))
		case command.GOLANG:
			registry.Register(command.Golang(workdir))
		}
	}

	w = &Worker{}
	w.Worker = agent.NewWorker(llm, 4, codec.FromEncoder(fromViper), registry)

	return
}

func (w *Worker) PromptOnce(ctx context.Context, input *viper.Viper, opt ...chatter.Opt) ([]byte, error) {
	reply, err := w.Worker.PromptOnce(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	return []byte(reply.Output), nil
}

func (w *Worker) Prompt(ctx context.Context, input *viper.Viper, opt ...chatter.Opt) ([]byte, error) {
	reply, err := w.Worker.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	return []byte(reply.Output), nil
}
