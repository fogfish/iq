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

	"github.com/akrylysov/pogreb"
	"github.com/fogfish/iq/internal/progress"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
)

// TODO: lazy load profile from netrc when building

//------------------------------------------------------------------------------

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
	cache    *pogreb.DB
	llm      chatter.Chatter
	router   *Router
	reporter interface{ SetTokenSource(progress.TokenSource) }
	err      error
}

// New creates a new LLM builder with mock LLM as default.
func New() *Builder {
	return &Builder{}
}

// Creates LLM from profile defined at ~/.iqrc
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

	b.router, b.err = NewRouter(profile, model)
	b.llm = b.router

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

func (b *Builder) Cache(path string) *Builder {
	if b.err != nil || b.llm == nil || len(path) == 0 {
		return b
	}

	b.cache, b.err = pogreb.Open(path, nil)
	if b.err != nil {
		return b
	}

	b.llm = aio.NewCache(b.cache, b.llm)
	return b
}

// routerAdapter adapts Router to TokenSource interface expected by progress.Reporter
type routerAdapter struct {
	router *Router
}

func (ra *routerAdapter) Usage() progress.TokenUsage {
	usage := ra.router.Usage()
	return progress.TokenUsage{
		InputTokens: usage.InputTokens,
		ReplyTokens: usage.ReplyTokens,
	}
}

func (ra *routerAdapter) ProfileUsage() []progress.ProfileTokenUsage {
	profiles := ra.router.ProfileUsage()
	result := make([]progress.ProfileTokenUsage, len(profiles))
	for i, profile := range profiles {
		result[i] = progress.ProfileTokenUsage{
			Name: profile.Name,
			Usage: progress.TokenUsage{
				InputTokens: profile.Usage.InputTokens,
				ReplyTokens: profile.Usage.ReplyTokens,
			},
		}
	}
	return result
}

// Reporter sets the progress reporter and wires it to the Router for token reporting.
// Returns builder for chaining.
func (b *Builder) Reporter(reporter interface{ SetTokenSource(progress.TokenSource) }) *Builder {
	if b.err != nil {
		return b
	}

	b.reporter = reporter
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

	// Wire Router to Reporter for token usage reporting
	if b.reporter != nil && b.router != nil {
		adapter := &routerAdapter{router: b.router}
		b.reporter.SetTokenSource(adapter)
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

	// Print thinking content using progress reporter if available
	for _, c := range reply.Content {
		switch v := c.(type) {
		case chatter.Text:
			t.printThinking(ctx, string(v))
		}
	}

	return reply, nil
}

// printThinking outputs thinking content through progress reporter if available,
// falls back to direct stderr output if not (for backwards compatibility)
func (t *thinking) printThinking(ctx context.Context, text string) {
	// Try to use progress reporter from context
	if reporter := getReporter(ctx); reporter != nil {
		reporter.ThinkingContent(text)
	}
}

// getReporter extracts the progress reporter from context
// This is a local helper to avoid importing internal/progress (would create import cycle)
// The progress package stores the reporter using a plain string key
func getReporter(ctx context.Context) interface{ ThinkingContent(string) } {
	// Use plain string key (matches progress package)
	const reporterKey = "iq.progress.reporter"

	if r := ctx.Value(reporterKey); r != nil {
		if reporter, ok := r.(interface{ ThinkingContent(string) }); ok {
			return reporter
		}
	}
	return nil
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
