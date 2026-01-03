//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"context"

	"github.com/fogfish/iq/internal/iosystem"
)

// ArrayCollector collects all input documents and emits them as array on EOF.
// This processor enables batch processing mode via --array CLI flag.
//
// Behavior:
//   - Normal documents: collected in memory, returns empty slice
//   - EOF document: emits all collected documents as array, returns them
//   - After EOF: collection resets for potential reuse
//
// Memory consideration:
//   - Buffers ALL documents in memory until EOF
//   - Not suitable for very large document streams
//   - Use only with explicit --array flag
type ArrayCollector struct {
	collected []*iosystem.Document
}

// NewArrayCollector creates processor that collects documents into array.
func NewArrayCollector() iosystem.Processor {
	return &ArrayCollector{
		collected: make([]*iosystem.Document, 0, 16), // Pre-allocate reasonable size
	}
}

// Process collects documents or emits array on EOF signal.
//
// Normal documents: collected, return empty slice (stops propagation until EOF)
// EOF document: emit collected []*Document array (monadic - passes documents directly)
func (p *ArrayCollector) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Check for EOF signal
	if len(docs) > 0 && docs[0].Type == iosystem.ContentEOF {
		// Return collected documents as array (monadic - no encoding!)
		result := p.collected
		p.collected = make([]*iosystem.Document, 0, 16) // Reset for potential reuse
		return result, nil
	}

	// Collect all input documents
	p.collected = append(p.collected, docs...)

	// Return empty - continue collecting until EOF
	return []*iosystem.Document{}, nil
}

// Close finalizes collection
func (p *ArrayCollector) Close() error {
	p.collected = nil
	return nil
}
