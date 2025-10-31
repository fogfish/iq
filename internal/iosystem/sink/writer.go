package sink

import (
	"context"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

// WriterSink wraps an io.Writer as a Sink.
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
type WriterSink struct {
	writer io.Writer
}

// NewWriterSink creates a Sink that writes all documents to the given writer.
func NewWriterSink(w io.Writer) *WriterSink {
	return &WriterSink{writer: w}
}

// Write copies the document content to the underlying writer.
func (s *WriterSink) Write(ctx context.Context, doc *iosystem.Document) error {
	_, err := io.Copy(s.writer, doc.Reader)
	return err
}

// Close does nothing since WriterSink doesn't own the writer.
// The writer lifecycle is managed by the caller (e.g., spool).
func (s *WriterSink) Close() error {
	return nil
}
