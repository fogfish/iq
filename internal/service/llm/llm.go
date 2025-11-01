//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/provider/autoconfig"
)

// Builder creates LLM instances with configured capabilities using builder pattern.
// Each method immediately creates/decorates the instance.
//
// Example:
//
//	llm, err := llm.New().
//	    Profile("bedrock/claude-3-sonnet").
//	    Debug(true).
//	    Think(true).
//	    MaxEpoch(10).
//	    Build()
type Builder struct {
	llm chatter.Chatter
	err error
}

// New creates a new LLM builder with mock LLM as default.
func New() *Builder {
	return &Builder{}
}

// Creates LLM from profile defined at ~/.netrc
//
// machine iq
//
//	provider provider:bedrock/foundation/converse
//	model us.anthropic.claude-3-7-sonnet-20250219-v1:0
//	region us-west-2
//
// Special value "mock" creates a mock LLM for testing of client applications.
func (b *Builder) Profile(profile, model string) *Builder {
	if b.err != nil || b.llm != nil || len(profile) == 0 {
		return b
	}

	// Mock mode
	if profile == "mock" || model == "mock" {
		b.llm = &Mock{}
		return b
	}

	// Create LLM from autoconfig
	switch {
	case len(model) > 0:
		b.llm, b.err = autoconfig.FromNetRC(profile, model)
	default:
		b.llm, b.err = autoconfig.FromNetRC(profile)
	}

	return b
}

// Think enables thinking mode (prints thinking content to stderr).
// Returns builder for chaining.
func (b *Builder) Think(enable bool) *Builder {
	if b.err != nil || b.llm == nil || !enable {
		return b
	}

	b.llm = &thinking{Chatter: b.llm}
	return b
}

// Debug enables debug logging to stderr.
// Returns builder for chaining.
func (b *Builder) Debug(enable bool) *Builder {
	if b.err != nil || b.llm == nil || !enable {
		return b
	}

	b.llm = aio.NewJsonLogger(os.Stderr, b.llm)
	return b
}

// Quota the maximum number of epochs and tokens.
// Returns builder for chaining.
func (b *Builder) Quota(epoch int, usage chatter.Usage) *Builder {
	if b.err != nil {
		return b
	}

	b.llm = aio.NewQuota(epoch, usage, b.llm)
	return b
}

// Build returns the configured LLM instance.
// Returns any error encountered during building.
func (b *Builder) Build() (chatter.Chatter, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.llm == nil {
		return nil, fmt.Errorf("llm: no LLM configured")
	}

	return b.llm, nil
}

//------------------------------------------------------------------------------
// Thinking Decorator
//------------------------------------------------------------------------------

// thinking wraps an LLM to print thinking content to stderr.
type thinking struct {
	chatter.Chatter
}

func (t *thinking) Prompt(ctx context.Context, prompt []chatter.Message, opts ...chatter.Opt) (*chatter.Reply, error) {
	reply, err := t.Chatter.Prompt(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}

	// Print thinking content
	for _, c := range reply.Content {
		switch v := c.(type) {
		case chatter.Text:
			fmt.Fprintf(os.Stderr, "\n  💭 %s\n\n", v)
		}
	}

	return reply, nil
}

//------------------------------------------------------------------------------
// Mock LLM
//------------------------------------------------------------------------------

// Mock is a simple mock LLM that echoes the input.
type Mock struct{}

func (m *Mock) Usage() chatter.Usage {
	return chatter.Usage{}
}

func (m *Mock) Prompt(ctx context.Context, prompt []chatter.Message, opt ...chatter.Opt) (*chatter.Reply, error) {
	// Echo all messages
	seq := make([]string, len(prompt))
	for i, msg := range prompt {
		seq[i] = msg.String()
	}
	reply := strings.Join(seq, " ")

	return &chatter.Reply{
		Stage: chatter.LLM_RETURN,
		Usage: chatter.Usage{
			InputTokens: len(reply),
			ReplyTokens: len(reply),
		},
		Content: []chatter.Content{
			chatter.Text(reply),
		},
	}, nil
}
