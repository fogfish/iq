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
	"io"
	"os"

	"github.com/fogfish/iq/internal/iosystem"
)

// Writer wraps an io.Writer as a Sink.
// This is useful for integrating with spool.ForEach or other
// scenarios where you have an io.Writer and want to use it with Pipeline.
//
// Example with spool:
//
//	spool.ForEach(ctx, root, func(ctx, path, r, w) error {
//	    source := source.NewReaderSource(path, r)
//	    sink := sink.NewWriterSink(w)
//	    return pipeline.Run(ctx, source, sink)
//	})
type Writer struct {
	writer io.Writer
}

// NewWriter creates a Sink that writes all documents to the given writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func NewStdout() *Writer {
	return &Writer{writer: os.Stdout}
}

// Write copies the document content to the underlying writer.
func (s *Writer) Write(ctx context.Context, doc *iosystem.Document) error {
	_, err := io.Copy(s.writer, doc.Reader)
	return err
}

// Close does nothing since WriterSink doesn't own the writer.
// The writer lifecycle is managed by the caller (e.g., spool).
func (s *Writer) Close() error {
	return nil
}
