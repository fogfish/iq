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

// Identity is a pass-through processor that returns documents unchanged.
// Useful for testing and as a base for more complex processors.
type Identity struct{}

// NewIdentity creates a processor that passes documents through unchanged.
func NewIdentity() iosystem.Processor {
	return &Identity{}
}

// Process returns the document unchanged in a slice.
func (p *Identity) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	return []*iosystem.Document{doc}, nil
}

// Close implements iosystem.Processor.
func (p *Identity) Close() error {
	return nil
}
