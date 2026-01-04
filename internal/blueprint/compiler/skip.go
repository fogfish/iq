//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/fogfish/iq/internal/progress"
)

// SkipChecker determines if documents should be skipped based on existing output.
type SkipChecker struct {
	storage  storage.Storage
	anchor   *AnchorKeyComputer
	reporter *progress.Reporter
}

// NewSkipChecker creates a skip checker.
func NewSkipChecker(store storage.Storage, anchor *AnchorKeyComputer, reporter *progress.Reporter) *SkipChecker {
	return &SkipChecker{
		storage:  store,
		anchor:   anchor,
		reporter: reporter,
	}
}

// ShouldSkip checks if document should be skipped.
// Returns true if anchor output exists, false otherwise.
func (s *SkipChecker) ShouldSkip(ctx context.Context, inputKey iosystem.Key) (bool, error) {
	// Compute expected output key
	anchorKey := s.anchor.ComputeAnchorKey(inputKey)

	// Check if output exists
	exists, err := s.storage.Has(ctx, anchorKey)
	if err != nil {
		return false, fmt.Errorf("failed to check anchor key %s: %w", anchorKey, err)
	}

	if exists && s.reporter != nil {
		s.reporter.DocumentSkipped(string(inputKey), "output exists at "+string(anchorKey))
	}

	return exists, nil
}
