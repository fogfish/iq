package sink

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
)

// FSSink writes documents to a filesystem directory (local or S3).
// It preserves the document path structure under the root directory.
// Uses github.com/fogfish/stream/spool for unified filesystem access.
type FSSink struct {
	fsys spool.FileSystem // Unified interface for local/S3
	root string           // Root directory path
}

// NewFSSink creates a sink that writes to a filesystem directory.
// Path can be local (e.g., "/path/to/dir") or S3 (e.g., "s3://bucket/prefix").
// Uses spool.FileSystem interface which is implemented by both stream and lfs.
func NewFSSink(path string) (iosystem.Sink, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Mount filesystem
	fsys, root, err := mountFSSink(path)
	if err != nil {
		return nil, fmt.Errorf("failed to mount filesystem: %w", err)
	}

	return &FSSink{
		fsys: fsys,
		root: root,
	}, nil
}

// Write writes a document to the filesystem, preserving its path structure.
func (s *FSSink) Write(ctx context.Context, doc *iosystem.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Construct full path: root + document path
	// Ensure path starts with / for stream library
	fullPath := doc.Path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	// If root is not just "/", prepend it
	if s.root != "/" {
		fullPath = s.root + fullPath
	}

	// Create the file
	file, err := s.fsys.Create(fullPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	// Copy document content to file
	_, err = io.Copy(file, doc.Reader)
	if err != nil {
		// Try to cancel the file creation on error
		file.Cancel()
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// Close implements iosystem.Sink.
func (s *FSSink) Close() error {
	// No cleanup needed - spool.FileSystem doesn't require closing
	return nil
}

// mountFSSink mounts a filesystem from a path (local or S3) for use with FSSink
func mountFSSink(path string) (spool.FileSystem, string, error) {
	if strings.HasPrefix(path, "s3://") {
		// S3 path: s3://bucket/prefix
		parts := strings.SplitN(path[5:], "/", 2)
		if len(parts) < 1 {
			return nil, "", fmt.Errorf("invalid S3 path format: %s", path)
		}
		bucket := parts[0]
		root := "/"
		if len(parts) > 1 {
			root = "/" + parts[1]
		}

		fsys, err := stream.NewFS(bucket)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create S3 filesystem: %w", err)
		}
		return fsys, root, nil
	}

	// Local path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fsys, err := lfs.New(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create local filesystem: %w", err)
	}
	return fsys, "/", nil
}
