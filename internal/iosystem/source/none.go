//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source

import (
	"bytes"
	"context"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
)

type None struct {
	consumed bool
}

// NewNone creates a Source that yields a single document from the given reader.
func NewNone() *None {
	return &None{}
}

// Next returns a single empty document on the first call and io.EOF on subsequent calls.
func (s *None) Next(ctx context.Context) (*iosystem.Document, error) {
	if s.consumed {
		return nil, io.EOF
	}
	s.consumed = true
	return iosystem.NewDocument("", codec.ContentText, bytes.NewBuffer(nil)), nil
}

// Close does nothing since None doesn't own any resources.
func (s *None) Close() error {
	return nil
}
