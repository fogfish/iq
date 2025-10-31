package source

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
)

// mountFS creates a filesystem for the given path and returns the filename for fs.Open().
// IMPORTANT: All paths returned by this function start with "/" as required by the stream library.
// For S3 paths (s3://bucket/key), it returns the stream.FS for the bucket and the key.
// For local paths, it returns the lfs.FS for the directory and the filename.
// Note: stream library requires paths to start with /
func mountFS(path string) (fsys fs.FS, filename string, err error) {
	if strings.HasPrefix(path, "s3://") {
		// S3 path: s3://bucket/key
		parts := strings.SplitN(path[5:], "/", 2)
		if len(parts) < 2 {
			return nil, "", fmt.Errorf("invalid S3 path format: %s", path)
		}
		bucket := parts[0]
		filename = "/" + parts[1] // stream requires leading /

		fsys, err = stream.NewFS(bucket)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create S3 filesystem: %w", err)
		}
		return fsys, filename, nil
	}

	// Local path: extract directory and filename
	dir := "."
	filename = path

	if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
		dir = path[:lastSlash]
		if dir == "" {
			dir = "/"
		}
		filename = path[lastSlash:] // Keep the leading /
	} else {
		// No slash in path - use current dir and prepend /
		filename = "/" + path
	}

	fsys, err = lfs.New(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create local filesystem: %w", err)
	}
	return fsys, filename, nil
}

// FileSource reads a single file.
// It uses github.com/fogfish/stream for unified S3/local access.
type FileSource struct {
	path     string // Original path for document naming
	filename string // Extracted filename for fs.Open
	fsys     fs.FS
	read     bool
}

// NewFileSource creates a source that reads a single file.
// Supports both local paths and S3 paths (s3://bucket/key).
func NewFileSource(fs fs.FS, path string) (iosystem.Source, error) {
	if path == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	fsys, filename, err := mountFS(path)
	if err != nil {
		return nil, err
	}

	return &FileSource{
		path:     path,
		filename: filename,
		fsys:     fsys,
	}, nil
}

// Next returns the file document or io.EOF if already read.
func (f *FileSource) Next(ctx context.Context) (*iosystem.Document, error) {
	if f.read {
		return nil, io.EOF
	}

	f.read = true

	file, err := f.fsys.Open(f.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", f.path, err)
	}

	return iosystem.NewDocument(f.path, file), nil
}

// Close implements iosystem.Source.
func (f *FileSource) Close() error {
	return nil
}

// FileSeqSource reads multiple files in sequence.
// It uses github.com/fogfish/stream for unified S3/local access.
type FileSeqSource struct {
	paths     []string // Original paths for document naming
	filenames []string // Extracted filenames for fs.Open
	fsys      fs.FS
	index     int
}

// NewFileSeqSource creates a source that reads multiple files sequentially.
// All files must be from the same filesystem (all local or all S3 with same bucket).
func NewFileSeqSource(paths ...string) (iosystem.Source, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one file path is required")
	}

	// Mount filesystem using first path
	fsys, firstFilename, err := mountFS(paths[0])
	if err != nil {
		return nil, err
	}

	// Determine filesystem type from first path
	isS3 := strings.HasPrefix(paths[0], "s3://")
	var bucket string
	if isS3 {
		parts := strings.SplitN(paths[0][5:], "/", 2)
		bucket = parts[0]
	}

	// Extract all filenames and validate compatibility
	filenames := make([]string, len(paths))
	filenames[0] = firstFilename

	for i := 1; i < len(paths); i++ {
		path := paths[i]

		// Validate same filesystem type
		if strings.HasPrefix(path, "s3://") != isS3 {
			return nil, fmt.Errorf("mixed local and S3 paths not supported")
		}

		// For S3, validate same bucket
		if isS3 {
			parts := strings.SplitN(path[5:], "/", 2)
			if len(parts) < 2 {
				return nil, fmt.Errorf("invalid S3 path format: %s", path)
			}
			if parts[0] != bucket {
				return nil, fmt.Errorf("all S3 paths must use same bucket, got: %s and %s", bucket, parts[0])
			}
			filenames[i] = "/" + parts[1]
		} else {
			// Local path - extract filename part
			if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
				filenames[i] = path[lastSlash:] // Keep leading /
			} else {
				filenames[i] = "/" + path
			}
		}
	}

	return &FileSeqSource{
		paths:     paths,
		filenames: filenames,
		fsys:      fsys,
	}, nil
}

// Next returns the next file document or io.EOF when all files are read.
func (f *FileSeqSource) Next(ctx context.Context) (*iosystem.Document, error) {
	if f.index >= len(f.paths) {
		return nil, io.EOF
	}

	path := f.paths[f.index]
	filename := f.filenames[f.index]
	f.index++

	file, err := f.fsys.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	return iosystem.NewDocument(path, file), nil
}

// Close implements iosystem.Source.
func (f *FileSeqSource) Close() error {
	return nil
}
