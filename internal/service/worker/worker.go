//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package worker

import (
	"errors"

	"github.com/fogfish/iq/internal/blueprint"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/kshard/chatter"
)

var (
	ErrBlueprintRequired = errors.New("blueprint is required")
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
	blueprint   *blueprint.Blueprint
	concurrency int
	errorMode   conduit.ErrorMode
	progress    conduit.ProgressFunc
	metrics     conduit.MetricsFunc
	err         error
}

// New creates a new conduit builder with default configuration.
func New() *Builder {
	return &Builder{
		concurrency: 1,
		errorMode:   conduit.FailFast,
	}
}

// Blueprint sets the blueprint to use for creating processors.
// This is required.
func (b *Builder) Blueprint(file string, llm chatter.Chatter) *Builder {
	if b.err != nil {
		return b
	}

	b.blueprint, b.err = blueprint.New(file, llm)
	return b
}

// Concurrency sets the number of parallel processing workers.
// Default is 1 (sequential processing).
func (b *Builder) Concurrency(n int) *Builder {
	if b.err != nil {
		return b
	}

	if n > 0 {
		b.concurrency = n
	}

	return b
}

// ErrorMode sets how errors are handled during processing.
// Options: conduit.FailFast (default) or conduit.SkipError.
func (b *Builder) ErrorMode(mode conduit.ErrorMode) *Builder {
	if b.err != nil {
		return b
	}

	b.errorMode = mode
	return b
}

// Progress sets a callback for progress updates after each document.
func (b *Builder) Progress(fn conduit.ProgressFunc) *Builder {
	if b.err != nil {
		return b
	}

	b.progress = fn
	return b
}

// Metrics sets a callback for periodic metrics updates.
func (b *Builder) Metrics(fn conduit.MetricsFunc) *Builder {
	if b.err != nil {
		return b
	}

	b.metrics = fn
	return b
}

// Build creates the configured conduit with blueprint processor.
// Returns the conduit ready to run with source and sink.
func (b *Builder) Build() (*conduit.Conduit, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.blueprint == nil {
		return nil, ErrBlueprintRequired
	}

	// Create conduit with configuration
	pipe := conduit.New(&conduit.Config{
		Concurrency: b.concurrency,
		ErrorMode:   b.errorMode,
		Progress:    b.progress,
		Metrics:     b.metrics,
	})

	pipe.AddProcessor(
		processor.NewAgent(b.blueprint, &processor.AgentConfig{}),
	)

	return pipe, nil
}
