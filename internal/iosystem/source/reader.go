package source

import (
	"context"
	"io"
	"os"

	"github.com/fogfish/iq/internal/iosystem"
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
	consumed bool
}

// NewReader creates a Source that yields a single document from the given reader.
func NewReader(path string, r io.Reader) *Reader {
	return &Reader{
		path:   path,
		reader: r,
	}
}

// NewStdin creates a source that reads from os.Stdin.
func NewStdin() iosystem.Source {
	return NewReader("stdin", os.Stdin)
}

// Next returns the document on first call, then io.EOF.
func (s *Reader) Next(ctx context.Context) (*iosystem.Document, error) {
	if s.consumed {
		return nil, io.EOF
	}
	s.consumed = true
	return iosystem.NewDocument(s.path, s.reader), nil
}

// Close does nothing since ReaderSource doesn't own the reader.
// The reader lifecycle is managed by the caller (e.g., spool).
func (s *Reader) Close() error {
	return nil
}
