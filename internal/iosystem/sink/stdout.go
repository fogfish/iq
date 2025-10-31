package sink

import (
	"context"
	"io"
	"os"

	"github.com/fogfish/iq/internal/iosystem"
)

// StdoutSink writes documents to standard output.
// Each document is written in sequence with a newline separator.
type StdoutSink struct {
	writer io.Writer
}

// NewStdoutSink creates a sink that writes to os.Stdout.
func NewStdoutSink() iosystem.Sink {
	return &StdoutSink{writer: os.Stdout}
}

// Write writes the document content to stdout.
func (s *StdoutSink) Write(ctx context.Context, doc *iosystem.Document) error {
	_, err := io.Copy(s.writer, doc.Reader)
	if err != nil {
		return err
	}

	// Add newline after each document for readability
	_, err = s.writer.Write([]byte("\n"))
	return err
}

// Close implements iosystem.Sink.
// Stdout is not closed to allow continued use.
func (s *StdoutSink) Close() error {
	return nil
}
