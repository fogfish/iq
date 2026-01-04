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

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/fogfish/iq/internal/progress"
)

// SkipIfExists is a processor that skips documents whose output already exists.
// It checks if the anchor key (expected output) exists in storage and filters
// out documents that have already been processed.
type SkipIfExists struct {
	checker *compiler.SkipChecker
}

// NewSkipIfExists creates a processor that filters documents based on existing output.
func NewSkipIfExists(store storage.Storage, anchor *compiler.AnchorKeyComputer, reporter *progress.Reporter) *SkipIfExists {
	return &SkipIfExists{
		checker: compiler.NewSkipChecker(store, anchor, reporter),
	}
}

// Process filters out documents that already have output.
func (p *SkipIfExists) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Passthrough EOF or empty
	if len(docs) == 0 || (len(docs) == 1 && docs[0].Type == iosystem.ContentEOF) {
		return docs, nil
	}

	// Filter documents
	filtered := make([]*iosystem.Document, 0, len(docs))
	for _, doc := range docs {
		// Skip if anchor exists
		skip, err := p.checker.ShouldSkip(ctx, iosystem.Key(doc.Path))
		if err != nil {
			return nil, err
		}

		if !skip {
			filtered = append(filtered, doc)
		}
	}

	return filtered, nil
}

// Close implements the Processor interface (no-op for skip processor).
func (p *SkipIfExists) Close() error {
	return nil
}
