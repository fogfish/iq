//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source

import (
	"context"
	"io"
	"os"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
)

// Reader wraps an io.Reader as a single-document Source.
// This is useful for integrating with spool.ForEach or other
// scenarios where you have an io.Reader and want to use it with Pipeline.
//
// Example with spool:
//
//	spool.ForEach(ctx, root, func(ctx, path, r, w) error {
//	    source := source.NewReaderSource(path, r)
//	    sink := sink.NewWriterSink(w)
//	    return pipeline.Run(ctx, source, sink)
//	})
type Reader struct {
	path     string
	reader   io.Reader
	content  string
	consumed bool
}

// NewReader creates a Source that yields a single document from the given reader.
func NewReader(path string, r io.Reader) *Reader {
	return &Reader{
		path:    path,
		reader:  r,
		content: codec.ContentStream,
	}
}

// NewStdin creates a source that reads from os.Stdin.
func NewStdin() *Reader {
	return NewReader("stdin", os.Stdin)
}

// NewReaderJSON creates a Source that yields a single document from the given reader
// with content type application/json. Use this when the reader contains structured JSON data.
func NewReaderJSON(path string, r io.Reader) iosystem.Source {
	return &Reader{
		path:    path,
		reader:  r,
		content: codec.ContentJSON,
	}
}

// Next returns the document on first call, then io.EOF.
func (s *Reader) Next(ctx context.Context) (*iosystem.Document, error) {
	if s.consumed {
		return nil, io.EOF
	}
	s.consumed = true
	doc := iosystem.NewDocument(iosystem.Key(s.path), codec.Default.DetectContentType(s.path), s.reader)
	doc.Type = s.content
	return doc, nil
}

// Close does nothing since ReaderSource doesn't own the reader.
// The reader lifecycle is managed by the caller (e.g., spool).
func (s *Reader) Close() error {
	return nil
}
