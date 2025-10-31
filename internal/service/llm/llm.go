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

// Config configures LLM creation.
type Config struct {
	Profile  string        // Provider profile (e.g., "bedrock", "openai")
	Model    string        // Model name
	MaxEpoch int           // Maximum number of epochs
	MaxUsage chatter.Usage // Maximum token usage
	Debug    bool          // Enable debug logging
	Think    bool          // Enable thinking mode
}

// Factory creates LLM instances with configured capabilities.
// Supports: debug logging, thinking mode, quota limits, mock mode.
type Factory struct {
	config Config
}

// New creates a new LLM factory from config.
func New(config Config) *Factory {
	return &Factory{
		config: config,
	}
}

// LLM creates a configured chatter.Chatter instance.
// Applies decorators in order: base → thinking → debug → quota.
func (f *Factory) LLM(model string) (chatter.Chatter, error) {
	// Create base LLM
	llm, err := f.create(model)
	if err != nil {
		return nil, err
	}

	// Apply thinking decorator
	if f.config.Think {
		llm = &Thinking{Chatter: llm}
	}

	// Apply debug decorator
	if f.config.Debug {
		llm = aio.NewJsonLogger(os.Stderr, llm)
	}

	// Apply quota decorator
	if f.config.MaxEpoch > 0 || f.config.MaxUsage.InputTokens > 0 || f.config.MaxUsage.ReplyTokens > 0 {
		llm = aio.NewQuota(f.config.MaxEpoch, f.config.MaxUsage, llm)
	}

	return llm, nil
}

// create creates the base LLM instance.
// Supports mock mode and autoconfig from ~/.netrc.
func (f *Factory) create(model string) (chatter.Chatter, error) {
	// Mock mode
	if f.config.Model == "mock" || model == "mock" {
		return &Mock{}, nil
	}

	// Use provided model parameter if set
	if model != "" {
		return autoconfig.FromNetRC(f.config.Profile, model)
	}

	// Use config model if set
	if f.config.Model != "" {
		return autoconfig.FromNetRC(f.config.Profile, f.config.Model)
	}

	// Use profile default
	return autoconfig.FromNetRC(f.config.Profile)
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
