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
	return &Builder{
		llm: &Mock{},
	}
}

// Profile sets the LLM provider profile and model.
// Format: "provider/model" or just "provider" to use default model.
// Special value "mock" creates a mock LLM for testing.
func (b *Builder) Profile(profile string) *Builder {
	if b.err != nil {
		return b
	}

	parts := strings.Split(profile, "/")
	if len(parts) == 0 {
		b.err = fmt.Errorf("invalid profile format: %s", profile)
		return b
	}

	provider := parts[0]
	model := ""
	if len(parts) > 1 {
		model = parts[1]
	}

	// Mock mode
	if provider == "mock" || model == "mock" {
		b.llm = &Mock{}
		return b
	}

	// Create LLM from autoconfig
	var llm chatter.Chatter
	var err error
	if model != "" {
		llm, err = autoconfig.FromNetRC(provider, model)
	} else {
		llm, err = autoconfig.FromNetRC(provider)
	}

	if err != nil {
		b.err = err
		return b
	}

	b.llm = llm
	return b
}

// Think enables thinking mode (prints thinking content to stderr).
// Returns builder for chaining.
func (b *Builder) Think(enable bool) *Builder {
	if b.err != nil || !enable {
		return b
	}

	b.llm = &Thinking{Chatter: b.llm}
	return b
}

// Debug enables debug logging to stderr.
// Returns builder for chaining.
func (b *Builder) Debug(enable bool) *Builder {
	if b.err != nil || !enable {
		return b
	}

	b.llm = aio.NewJsonLogger(os.Stderr, b.llm)
	return b
}

// MaxEpoch sets the maximum number of epochs (quota decorator).
// Returns builder for chaining.
func (b *Builder) MaxEpoch(max int) *Builder {
	if b.err != nil || max <= 0 {
		return b
	}

	b.llm = aio.NewQuota(max, chatter.Usage{}, b.llm)
	return b
}

// MaxTokens sets the maximum number of reply tokens (quota decorator).
// Returns builder for chaining.
func (b *Builder) MaxTokens(max int) *Builder {
	if b.err != nil || max <= 0 {
		return b
	}

	b.llm = aio.NewQuota(0, chatter.Usage{ReplyTokens: max}, b.llm)
	return b
}

// Build returns the configured LLM instance.
// Returns any error encountered during building.
func (b *Builder) Build() (chatter.Chatter, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.llm, nil
}

//------------------------------------------------------------------------------
// Thinking Decorator
//------------------------------------------------------------------------------

// Thinking wraps an LLM to print thinking content to stderr.
type Thinking struct {
	chatter.Chatter
}

func (t *Thinking) Prompt(ctx context.Context, prompt []chatter.Message, opts ...chatter.Opt) (*chatter.Reply, error) {
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
