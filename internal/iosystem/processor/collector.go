//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"bytes"
	"context"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
)

// Collector collects all input documents and emits them as array on EOF.
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
type Collector struct {
	merge     bool
	collected []*iosystem.Document
}

// NewCollector creates processor that collects documents into array.
func NewCollector(merge bool) iosystem.Processor {
	return &Collector{
		merge:     merge,
		collected: make([]*iosystem.Document, 0, 16), // Pre-allocate reasonable size
	}
}

// Process collects documents or emits array on EOF signal.
//
// Normal documents: collected, return empty slice (stops propagation until EOF)
// EOF document: emit collected []*Document array (monadic - passes documents directly)
func (p *Collector) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Check for EOF signal
	if iosystem.IsEOF(docs) && !p.merge {
		result := p.collected
		p.collected = make([]*iosystem.Document, 0, 16) // Reset for potential reuse
		return result, nil
	}

	if iosystem.IsEOF(docs) && p.merge {
		var buf bytes.Buffer
		for _, doc := range p.collected {
			_, err := io.Copy(&buf, doc.Reader)
			if err != nil {
				return nil, err
			}
			buf.WriteByte('\n')
		}
		doc := iosystem.NewDocument(p.collected[0].Key, codec.ContentText, &buf)
		return []*iosystem.Document{doc}, nil
	}

	p.collected = append(p.collected, docs...)
	return []*iosystem.Document{}, nil
}

// Close finalizes collection
func (p *Collector) Close() error {
	p.collected = nil
	return nil
}
