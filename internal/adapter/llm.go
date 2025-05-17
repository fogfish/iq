//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package adapter

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/llm/autoconfig"
)

// LLM adapter for
type LLM struct {
	Profile  string
	Model    string
	MaxEpoch int
	MaxUsage chatter.Usage
}

// Create instance of LLM for usage with iq
func (adapter LLM) Create(model string, debug bool, think bool) (chatter.Chatter, error) {
	llm, err := adapter.create(model)
	if err != nil {
		return nil, err
	}

	if think {
		llm = &Thinking{Chatter: llm}
	}

	if debug {
		llm = aio.NewJsonLogger(os.Stderr, llm)
	}

	if adapter.MaxEpoch > 0 || adapter.MaxUsage.InputTokens > 0 || adapter.MaxUsage.ReplyTokens > 0 {
		llm = aio.NewQuota(adapter.MaxEpoch, adapter.MaxUsage, llm)
	}

	return llm, nil
}

func (adapter LLM) create(model string) (chatter.Chatter, error) {
	if adapter.Model == "mock" || model == "mock" {
		return llmock(0), nil
	}

	if len(model) != 0 {
		return autoconfig.New(adapter.Profile, model)
	}

	if len(adapter.Model) != 0 {
		return autoconfig.New(adapter.Profile, adapter.Model)
	}

	return autoconfig.New(adapter.Profile)
}

//------------------------------------------------------------------------------

type Thinking struct {
	chatter.Chatter
}

func (lib *Thinking) Prompt(ctx context.Context, prompt []chatter.Message, opts ...chatter.Opt) (*chatter.Reply, error) {
	reply, err := lib.Chatter.Prompt(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}

	for _, c := range reply.Content {
		switch v := (c).(type) {
		case chatter.Text:
			fmt.Fprintf(os.Stderr, "\n  💭 %s\n\n", v)
		}
	}

	return reply, nil
}

//------------------------------------------------------------------------------

type llmock int

func (llmock) Usage() chatter.Usage { return chatter.Usage{} }

func (llmock) Prompt(ctx context.Context, prompt []chatter.Message, opt ...chatter.Opt) (*chatter.Reply, error) {
	seq := make([]string, len(prompt))
	for i, s := range prompt {
		seq[i] = s.String()
	}
	reply := strings.Join(seq, " ")

	return &chatter.Reply{
		Stage: chatter.LLM_RETURN,
		Usage: chatter.Usage{InputTokens: len(reply), ReplyTokens: len(reply)},
		Content: []chatter.Content{
			chatter.Text(reply),
		},
	}, nil
}
