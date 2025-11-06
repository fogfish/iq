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
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

// Union combines multiple sources into a single document.
// It reads all documents from all sources and concatenates their content.
// This is useful for --merge mode where multiple files should be processed as one.
type Union struct {
	sources []iosystem.Source
	merged  bool
	content *bytes.Buffer
	paths   []string
}

// NewUnion creates a source that merges multiple sources into one document.
func NewUnion(sources ...iosystem.Source) (iosystem.Source, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}

	return &Union{
		sources: sources,
		content: &bytes.Buffer{},
		paths:   make([]string, 0),
	}, nil
}

// Next returns a single merged document containing all content from all sources.
// Returns io.EOF on subsequent calls.
func (m *Union) Next(ctx context.Context) (*iosystem.Document, error) {
	// Only merge once
	if m.merged {
		return nil, io.EOF
	}
	m.merged = true

	// Read all documents from all sources
	for i, source := range m.sources {
		for {
			doc, err := source.Next(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read from source %d: %w", i, err)
			}
			m.paths = append(m.paths, doc.Path)

			n, err := io.Copy(m.content, doc.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to copy document %s: %w", doc.Path, err)
			}

			// Add separator between documents (newline)
			if n > 0 && !bytes.HasSuffix(m.content.Bytes(), []byte("\n")) {
				m.content.WriteString("\n")
			}
		}
	}

	if m.content.Len() == 0 {
		return nil, io.EOF
	}

	doc := iosystem.NewDocument(
		m.paths[0],
		io.NopCloser(bytes.NewReader(m.content.Bytes())),
	)

	return doc, nil
}

// Close closes all underlying sources.
func (m *Union) Close() error {
	var errs []error
	for i, source := range m.sources {
		if err := source.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close source %d: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing sources: %v", errs)
	}
	return nil
}
