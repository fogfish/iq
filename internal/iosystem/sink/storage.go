//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package sink

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

// StorageSink writes documents using Storage interface with emit support.
type StorageSink struct {
	storage storage.Storage
}

// NewStorage creates a sink that writes to storage.
func NewStorage(store storage.Storage) iosystem.Sink {
	return &StorageSink{
		storage: store,
	}
}

// Write stores a document applying emit context if present.
func (s *StorageSink) Write(ctx context.Context, doc *iosystem.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Get emit context - first from context, then from document metadata
	emitCtx := compiler.GetEmitContext(ctx)
	
	// If not in context, try to load from document metadata
	if emitCtx == nil || emitCtx.Prefix == "" {
		if prefix, ok := doc.Metadata.Custom["emit.prefix"]; ok && prefix != "" {
			emitCtx = &compiler.EmitContext{
				Prefix: prefix,
			}
			// Parse counters if present
			if countersJSON, ok := doc.Metadata.Custom["emit.counters"]; ok {
				var counters []int
				json.Unmarshal([]byte(countersJSON), &counters)
				emitCtx.Counters = counters
			}
		}
	}
	
	outputKey := doc.Key

	// Apply emit prefix and counters if present
	if emitCtx != nil {
		if len(emitCtx.Counters) > 0 {
			// Foreach mode: apply counters
			outputKey = compiler.ApplyEmitWithCounters(emitCtx.Prefix, doc.Key, emitCtx.Counters)
		} else if emitCtx.Prefix != "" {
			// Regular emit prefix
			outputKey = compiler.ApplyEmit(emitCtx.Prefix, doc.Key)
		}
	}

	// Write to storage
	err := s.storage.Put(ctx, outputKey, doc.Reader)
	if err != nil {
		return fmt.Errorf("failed to write key %s: %w", outputKey, err)
	}

	return nil
}

// Close finalizes the sink (no-op for storage).
func (s *StorageSink) Close() error {
	return nil
}
