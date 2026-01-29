//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package conduit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/google/jsonschema-go/jsonschema"
)

// Conduit coordinates the flow: Source → Processor(s) → Sink.
// The pipeline is reusable across multiple source/sink pairs.
type Conduit struct {
	Name       string
	About      string
	Input      *jsonschema.Schema
	Reply      *jsonschema.Schema
	processors []iosystem.Processor
	config     Config
}

// Config configures pipeline behavior.
type Config struct {
	// Concurrency sets the number of parallel processing workers.
	// 0 or 1 means sequential processing.
	Concurrency int

	// ErrorMode determines how errors are handled.
	ErrorMode ErrorMode

	// Progress callback is invoked after each document is processed.
	Progress ProgressFunc

	// Metrics callback is invoked periodically with pipeline statistics.
	Metrics MetricsFunc
}

// ErrorMode defines error handling strategies.
type ErrorMode int

const (
	// FailFast stops the pipeline on first error.
	FailFast ErrorMode = iota

	// SkipError continues processing remaining documents after errors.
	SkipError
)

// ProgressFunc is called after processing each document.
type ProgressFunc func(doc *iosystem.Document, err error)

// MetricsFunc is called periodically with pipeline statistics.
type MetricsFunc func(stats Stats)

// Stats tracks processing metrics.
type Stats struct {
	DocsProcessed int       // Successfully processed documents
	DocsSkipped   int       // Documents skipped due to errors
	BytesRead     int64     // Total bytes read from source
	BytesWritten  int64     // Total bytes written to sink
	Errors        []error   // Collected errors
	Created       time.Time // When processing started
	Stopped       time.Time // When processing ended
}

// New creates a new reusable processing pipeline.
// The pipeline can be run multiple times with different sources and sinks.
// Pass nil for config to use defaults (sequential, fail-fast).
func New(config *Config) *Conduit {
	p := &Conduit{
		processors: make([]iosystem.Processor, 0),
		config: Config{
			Concurrency: 1,
			ErrorMode:   FailFast,
		},
	}

	if config != nil {
		p.config = *config
	}

	// Ensure valid concurrency value
	if p.config.Concurrency < 1 {
		p.config.Concurrency = 1
	}

	return p
}

// AddProcessor adds a processing stage to the pipeline.
// Processors are applied in the order they are added.
func (p *Conduit) AddProcessor(proc iosystem.Processor) *Conduit {
	p.processors = append(p.processors, proc)
	return p
}

// Run executes the pipeline with the given source and sink.
// The pipeline can be run multiple times with different sources and sinks.
// Returns statistics and an error if processing failed.
func (p *Conduit) Run(ctx context.Context, source iosystem.Source, sink iosystem.Sink) (*Stats, error) {
	if source == nil {
		return nil, fmt.Errorf("source cannot be nil")
	}
	if sink == nil {
		return nil, fmt.Errorf("sink cannot be nil")
	}

	stats := &Stats{Created: time.Now()}

	var err error
	// TODO: Concurrency
	if p.config.Concurrency <= 1 {
		err = p.runSequential(ctx, source, sink, stats)
	} else {
		err = p.runConcurrent(ctx, source, sink, stats)
	}

	// Report metrics after processing but before returning
	// This ensures summary appears before any final output
	stats.Stopped = time.Now()
	if p.config.Metrics != nil {
		p.config.Metrics(*stats)
	}

	return stats, err
}

// Cmd executes the pipeline in the context of MCP.
func (p *Conduit) Cmd(ctx context.Context, json json.RawMessage) (*bytes.Buffer, error) {
	r := source.NewReaderJSON("", bytes.NewBuffer(json))

	var out bytes.Buffer
	w := sink.NewWriter(&out)

	_, err := p.Run(ctx, r, w)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// runSequential processes documents one at a time.
func (p *Conduit) runSequential(ctx context.Context, source iosystem.Source, sink iosystem.Sink, stats *Stats) error {
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get next document
		doc, err := source.Next(ctx)
		if err == io.EOF {
			// Inject EOF document into pipeline
			_ = p.processDocument(ctx, []*iosystem.Document{iosystem.EOF()}, sink, stats)
			return nil // Normal completion
		}
		if err != nil {
			stats.Errors = append(stats.Errors, err)
			if p.config.ErrorMode == FailFast {
				return err
			}
			continue
		}

		if err := p.processDocument(ctx, []*iosystem.Document{doc}, sink, stats); err != nil {
			stats.Errors = append(stats.Errors, err)
			if p.config.Progress != nil {
				p.config.Progress(doc, err)
			}
			if p.config.ErrorMode == FailFast {
				return err
			}
			stats.DocsSkipped++
			continue
		}

		stats.DocsProcessed++
		if p.config.Progress != nil {
			p.config.Progress(doc, nil)
		}
	}
}

// runConcurrent processes documents concurrently (stub for Phase 4).
func (p *Conduit) runConcurrent(ctx context.Context, source iosystem.Source, sink iosystem.Sink, stats *Stats) error {
	// TODO: Implement in Phase 4
	return fmt.Errorf("concurrent processing not yet implemented")
}

// processDocument runs documents through all processors and to the sink.
func (p *Conduit) processDocument(ctx context.Context, docs []*iosystem.Document, sink iosystem.Sink, stats *Stats) error {
	// Pass documents through processor chain
	currentDocs := docs

	for _, processor := range p.processors {
		// Monadic call: []*Document -> []*Document
		nextDocs, err := processor.Process(ctx, currentDocs)
		if err != nil {
			return fmt.Errorf("processor error: %w", err)
		}
		currentDocs = nextDocs

		// If processor filtered out all documents, stop
		if len(currentDocs) == 0 {
			return nil
		}
	}

	// Write final documents to sink
	for _, d := range currentDocs {
		if d.Type == codec.ContentEOF {
			continue // Don't write EOF markers to output
		}
		if _, err := sink.Write(ctx, d); err != nil {
			return fmt.Errorf("sink error: %w", err)
		}
	}

	return nil
}

// Close closes all processors.
// Note: Source and Sink are managed by the caller.
func (p *Conduit) Close() error {
	var errs []error
	for i, proc := range p.processors {
		if err := proc.Close(); err != nil {
			errs = append(errs, fmt.Errorf("processor %d: %w", i, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing processors: %v", errs)
	}
	return nil
}
