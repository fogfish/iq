package sink

import (
	"context"
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/stream"
)

// FS writes documents to a filesystem directory.
// It preserves the document path structure.
// The filesystem is provided by the client (e.g., lfs, stream.NewFS).
type FS struct {
	fsys stream.CreateFS[struct{}]
}

// NewFS creates a sink that writes to a filesystem directory.
// The fs parameter should be a CreateFS filesystem (e.g., from lfs.New or stream.NewFS).
// Document paths are passed as-is to fsys.Create() - the client is responsible for proper formatting.
func NewFS(fsys stream.CreateFS[struct{}]) (iosystem.Sink, error) {
	return &FS{
		fsys: fsys,
	}, nil
}

// Write writes a document to the filesystem, preserving its path structure.
func (s *FS) Write(ctx context.Context, doc *iosystem.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Create the file using document's path
	file, err := s.fsys.Create(doc.Path, nil)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", doc.Path, err)
	}
	defer file.Close()

	// Copy document content to file
	_, err = io.Copy(file, doc.Reader)
	if err != nil {
		// Try to cancel the file creation on error
		file.Cancel()
		return fmt.Errorf("failed to write file %s: %w", doc.Path, err)
	}

	return nil
}

// Close implements iosystem.Sink.
func (s *FS) Close() error {
	// No cleanup needed - CreateFS doesn't require closing
	return nil
}
