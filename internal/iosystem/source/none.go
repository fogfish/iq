package source

import (
	"bytes"
	"context"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
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
	return iosystem.NewDocument("", bytes.NewBuffer(nil)), nil
}

// Close does nothing since None doesn't own any resources.
func (s *None) Close() error {
	return nil
}
