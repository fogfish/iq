//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package worker

import (
	"fmt"

	"github.com/fogfish/iq/internal/blueprint"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/kshard/chatter"
)

// Builder creates a configured conduit with processors using builder pattern.
// Each method immediately creates/configures the conduit.
//
// Example:
//
//	conduit, err := worker.New().
//	    Blueprint(bp).
//	    Job("process").
//	    Concurrency(4).
//	    Build()
type Builder struct {
	conduit conduit.Config
	runtime *conduit.Conduit
	err     error
}

// New creates a new conduit builder with default configuration.
func New() *Builder {
	return &Builder{
		conduit: conduit.Config{
			Concurrency: 1,
			ErrorMode:   conduit.FailFast,
		},
	}
}

// Concurrency sets the number of parallel processing workers.
// Default is 1 (sequential processing).
func (b *Builder) Concurrency(n int) *Builder {
	if b.err != nil {
		return b
	}

	if n > 0 {
		b.conduit.Concurrency = n
	}

	return b
}

// ErrorMode sets how errors are handled during processing.
// Options: conduit.FailFast (default) or conduit.SkipError.
func (b *Builder) ErrorMode(mode conduit.ErrorMode) *Builder {
	if b.err != nil {
		return b
	}

	b.conduit.ErrorMode = mode
	return b
}

// Progress sets a callback for progress updates after each document.
func (b *Builder) Progress(fn conduit.ProgressFunc) *Builder {
	if b.err != nil {
		return b
	}

	b.conduit.Progress = fn
	return b
}

// Metrics sets a callback for periodic metrics updates.
func (b *Builder) Metrics(fn conduit.MetricsFunc) *Builder {
	if b.err != nil {
		return b
	}

	b.conduit.Metrics = fn
	return b
}

func (b *Builder) Runtime() *Builder {
	if b.err != nil {
		return b
	}

	b.runtime = conduit.New(&b.conduit)
	return b
}

func (b *Builder) Splitter(conf processor.ChunkConfig) *Builder {
	if b.err != nil || b.runtime == nil || conf.Strategy == processor.StrategyNone {
		return b
	}

	b.runtime.AddProcessor(
		processor.NewChunker(conf),
	)
	return b
}

// Workflow sets the blueprint to use for creating processors.
// This is required.
func (b *Builder) Workflow(file string, llm chatter.Chatter) *Builder {
	if b.err != nil || b.runtime == nil {
		return b
	}

	wrk, err := blueprint.New(file, llm)
	if err != nil {
		b.err = fmt.Errorf("failed to create blueprint from %s: %w", file, err)
		return b
	}

	b.runtime.AddProcessor(
		processor.NewAgent(wrk, &processor.AgentConfig{}),
	)
	b.runtime.Name = wrk.Name()
	b.runtime.About = wrk.About()
	b.runtime.Input, b.runtime.Reply = wrk.Schema()

	return b
}

func (b *Builder) Jsonify(enable bool) *Builder {
	if b.err != nil || b.runtime == nil || !enable {
		return b
	}

	b.runtime.AddProcessor(
		processor.NewJsonify(processor.JsonifyConfig{
			Indent: 2,
			Color:  true,
		}),
	)
	return b
}

// Build creates the configured conduit with blueprint processor.
// Returns the conduit ready to run with source and sink.
func (b *Builder) Build() (*conduit.Conduit, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.runtime == nil {
		return nil, fmt.Errorf("undefined workflow")
	}

	return b.runtime, nil
}
