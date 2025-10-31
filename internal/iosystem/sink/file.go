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

// FileSink writes to a single file.
// All documents are concatenated into one output file.
type FileSink struct {
	path string
	fsys spool.FileSystem
	file stream.File
}

// NewFileSink creates a sink that writes to a single file.
// Supports both local paths and S3 paths (s3://bucket/key).
func NewFileSink(path string) (iosystem.Sink, error) {
	if path == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	var fsys spool.FileSystem
	var err error

	if strings.HasPrefix(path, "s3://") {
		// S3 path
		fsys, err = stream.NewFS(path[5:])
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 filesystem: %w", err)
		}
	} else {
		// Local path
		fsys, err = lfs.New(".")
		if err != nil {
			return nil, fmt.Errorf("failed to create local filesystem: %w", err)
		}
	}

	return &FileSink{
		path: path,
		fsys: fsys,
	}, nil
}

// Write appends the document content to the file.
func (f *FileSink) Write(ctx context.Context, doc *iosystem.Document) error {
	// Open file on first write
	if f.file == nil {
		filename := f.path
		if strings.HasPrefix(filename, "s3://") {
			parts := strings.SplitN(filename[5:], "/", 2)
			if len(parts) > 1 {
				filename = parts[1]
			} else {
				filename = ""
			}
		}

		file, err := f.fsys.Create(filename, nil)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", f.path, err)
		}
		f.file = file
	}

	// Write document content
	_, err := io.Copy(f.file, doc.Reader)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", f.path, err)
	}

	// Add newline separator between documents
	_, err = f.file.Write([]byte("\n"))
	return err
}

// Close closes the file and releases resources.
func (f *FileSink) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}
