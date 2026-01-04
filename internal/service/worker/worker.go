//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package worker

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint"
	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/fogfish/iq/internal/progress"
	"github.com/kshard/chatter"
)

// Builder creates a configured conduit with processors using builder pattern.
//
// Example:
//
//	conduit, err := worker.New().
//	    Blueprint(bp).
//	    Job("process").
//	    Concurrency(4).
//	    Build()
type Builder struct {
	conduit    conduit.Config
	runtime    *conduit.Conduit
	reporter   *progress.Reporter
	workflow   *blueprint.Blueprint
	cacheStore storage.Storage // Storage for skip-if-exists caching
	err        error
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

// Reporter sets the progress reporter and wires up callbacks
func (b *Builder) Reporter(r *progress.Reporter) *Builder {
	if b.err != nil || r == nil {
		return b
	}

	b.reporter = r

	// Wire up progress callback
	b.conduit.Progress = func(doc *iosystem.Document, err error) {
		if err != nil {
			r.DocumentError(doc.Path, err)
		} else {
			// We'll report completion at the end of processing
		}
	}

	// Wire up metrics callback for final summary
	b.conduit.Metrics = func(stats conduit.Stats) {
		if stats.DocsProcessed > 0 || stats.DocsSkipped > 0 || len(stats.Errors) > 0 {
			r.Summary()
		}
	}

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
	if b.err != nil || b.runtime == nil || conf.Strategy == processor.ChunkerNone {
		return b
	}

	b.runtime.AddProcessor(
		processor.NewChunker(conf),
	)
	return b
}

// ArrayMode enables batch processing by collecting all documents into array.
// When enabled, adds ArrayCollector processor as first stage after Runtime.
// ArrayCollector buffers all documents until EOF, then emits them as array.
//
// Use cases:
//   - Process JSON array files
//   - Batch process split documents
//   - Enable foreach with selector: document
//
// Memory warning: All documents buffered in memory until EOF.
// Not suitable for very large document streams.
func (b *Builder) ArrayMode(enable bool) *Builder {
	if b.err != nil || b.runtime == nil || !enable {
		return b
	}

	// Add ArrayCollector as first processor
	// It will collect documents and emit array on EOF
	b.runtime.AddProcessor(processor.NewArrayCollector())

	return b
}

// Workflow sets the blueprint to use for creating processors.
// This is required.
func (b *Builder) Workflow(file string, llm chatter.Chatter) *Builder {
	if b.err != nil || b.runtime == nil {
		return b
	}

	if len(file) == 0 {
		b.runtime.AddProcessor(
			processor.NewPrompter(llm),
		)
		return b
	}

	wrk, err := blueprint.New(file, llm)
	if err != nil {
		b.err = fmt.Errorf("failed to create blueprint from %s: %w", file, err)
		return b
	}

	// Store workflow for potential use in SkipIfExists
	b.workflow = wrk

	// Report workflow compiled with actual counts
	if b.reporter != nil {
		b.reporter.WorkflowCompiled(wrk.Name(), wrk.JobCount(), wrk.StepCount())
	}

	b.runtime.AddProcessor(
		processor.NewAgent(wrk, &processor.AgentConfig{}),
	)
	b.runtime.Name = wrk.Name()
	b.runtime.About = wrk.About()
	b.runtime.Input, b.runtime.Reply = wrk.Schema()

	return b
}

// SkipIfExists enables step-level output caching for incremental processing.
// When enabled, each step with an 'emit' attribute will check if its output
// already exists in storage and skip LLM operations if found, using cached output instead.
// Must be called after Workflow() and before Build().
func (b *Builder) SkipIfExists(outputPath string) *Builder {
	if b.err != nil || b.runtime == nil || b.workflow == nil || outputPath == "" {
		return b
	}

	// Create storage for output checking/caching
	store, err := storage.NewFS(outputPath)
	if err != nil {
		b.err = fmt.Errorf("failed to create storage for skip-if-exists: %w", err)
		return b
	}

	// Store reference for use in Build()
	b.cacheStore = store

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
// Returns a wrapped conduit that includes progress reporter in context.
func (b *Builder) Build() (*ConduitWithReporter, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.runtime == nil {
		return nil, fmt.Errorf("undefined workflow")
	}

	return &ConduitWithReporter{
		Conduit:    b.runtime,
		reporter:   b.reporter,
		cacheStore: b.cacheStore,
	}, nil
}

// ConduitWithReporter wraps a conduit and injects progress reporter into context
type ConduitWithReporter struct {
	*conduit.Conduit
	reporter   *progress.Reporter
	cacheStore storage.Storage // Storage for skip-if-exists caching
}

// Run executes the pipeline with progress reporter in context
func (c *ConduitWithReporter) Run(ctx context.Context, source iosystem.Source, sink iosystem.Sink) (*conduit.Stats, error) {
	// Add reporter to context if available
	if c.reporter != nil {
		ctx = progress.WithReporter(ctx, c.reporter)

		// Wrap sink to buffer output until after summary
		sink = newBufferingSink(sink)
	}

	// Add cache context if storage configured
	if c.cacheStore != nil {
		cacheCtx := &compiler.CacheContext{
			Storage: c.cacheStore,
			Enabled: true,
		}
		ctx = compiler.WithCacheContext(ctx, cacheCtx)
	}

	stats, err := c.Conduit.Run(ctx, source, sink)

	// If we have a buffering sink, flush it after summary
	if bufSink, ok := sink.(*bufferingSink); ok {
		bufSink.Flush()
	}

	return stats, err
}

// bufferingSink buffers all writes and flushes them on demand
type bufferingSink struct {
	target iosystem.Sink
	buffer []*iosystem.Document
}

func newBufferingSink(target iosystem.Sink) *bufferingSink {
	return &bufferingSink{
		target: target,
		buffer: make([]*iosystem.Document, 0),
	}
}

func (s *bufferingSink) Write(ctx context.Context, doc *iosystem.Document) error {
	// Buffer the document instead of writing immediately
	s.buffer = append(s.buffer, doc)
	return nil
}

func (s *bufferingSink) Flush() error {
	// Write all buffered documents to the target sink
	for _, doc := range s.buffer {
		if err := s.target.Write(context.Background(), doc); err != nil {
			return err
		}
	}
	s.buffer = nil
	return nil
}

func (s *bufferingSink) Close() error {
	return s.target.Close()
}
