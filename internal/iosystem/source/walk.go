package source

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/opts"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
)

// WalkSource walks a filesystem tree (local or S3) and produces documents for each file.
// It uses github.com/fogfish/stream/spool for battle-tested file walking with mutable mode support.
// The source wraps spool.Spool's ForEach callback pattern into an iterator pattern using channels.
type WalkSource struct {
	spool  *spool.Spool       // Battle-tested spool for walking files
	root   string             // Root path
	ch     chan *docOrError   // Channel for documents
	ctx    context.Context    // Context for walking
	cancel context.CancelFunc // Cancel function
	done   chan struct{}      // Done signal
}

// WalkConfig configures WalkSource behavior.
type WalkConfig struct {
	// Mutable mode - removes files after reading (queue/spool mode)
	// Enables resume capability for batch processing
	Mutable bool

	// Strict mode - fail on first error
	Strict bool
}

// docOrError wraps a document or error for channel communication
type docOrError struct {
	doc *iosystem.Document
	err error
}

// NewWalkSource creates a source that walks a filesystem tree.
// Path can be local (e.g., "/path/to/dir") or S3 (e.g., "s3://bucket/prefix").
// Uses spool.Spool which is battle-tested for mutable/immutable file processing.
func NewWalkSource(path string, config WalkConfig) (iosystem.Source, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Mount input filesystem
	dir, err := mountSpool(path)
	if err != nil {
		return nil, fmt.Errorf("failed to mount input path: %w", err)
	}

	// Create a dummy output filesystem (spool requires both, but we only read)
	// Use a no-op writer that discards everything
	out := &noopFS{}

	// Configure spool options
	opt := []opts.Option[spool.Spool]{}
	if config.Mutable {
		opt = append(opt, spool.IsMutable)
	} else {
		opt = append(opt, spool.IsImmutable)
	}

	if config.Strict {
		opt = append(opt, spool.WithStrict)
	} else {
		opt = append(opt, spool.WithSkipError)
	}

	// Create spool
	sp := spool.New(dir, out, opt...)

	// Determine root path
	root := "/"
	if strings.HasPrefix(path, "s3://") {
		parts := strings.SplitN(path[5:], "/", 2)
		if len(parts) > 1 {
			root = "/" + parts[1]
		}
	}

	// Create source with channel
	ctx, cancel := context.WithCancel(context.Background())
	w := &WalkSource{
		spool:  sp,
		root:   root,
		ch:     make(chan *docOrError, 10), // Buffer for smoother operation
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Start walking in background goroutine
	go w.walk()

	return w, nil
}

// Next returns the next file document or io.EOF when all files are processed.
func (w *WalkSource) Next(ctx context.Context) (*iosystem.Document, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item, ok := <-w.ch:
		if !ok {
			// Channel closed, all files processed
			return nil, io.EOF
		}
		if item.err != nil {
			return nil, item.err
		}
		return item.doc, nil
	}
}

// Close implements iosystem.Source.
func (w *WalkSource) Close() error {
	w.cancel()
	<-w.done // Wait for walk goroutine to finish
	return nil
}

// walk runs in a goroutine and feeds documents into the channel
func (w *WalkSource) walk() {
	defer close(w.ch)
	defer close(w.done)

	// Use spool.ForEach which handles mutable/immutable mode automatically
	err := w.spool.ForEach(w.ctx, w.root,
		func(ctx context.Context, path string, r io.Reader, _ io.Writer) error {
			// Read all content into memory (necessary for Document pattern)
			content, err := io.ReadAll(r)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			// Create document with content reader
			doc := iosystem.NewDocument(path, io.NopCloser(strings.NewReader(string(content))))

			// Send to channel
			select {
			case <-ctx.Done():
				return ctx.Err()
			case w.ch <- &docOrError{doc: doc}:
				return nil
			}
		})

	if err != nil && err != context.Canceled {
		// Send error to channel
		select {
		case w.ch <- &docOrError{err: err}:
		case <-w.ctx.Done():
		}
	}
}

// mountSpool mounts a filesystem from a path (local or S3) for use with spool
func mountSpool(path string) (spool.FileSystem, error) {
	if strings.HasPrefix(path, "s3://") {
		bucket := strings.TrimPrefix(path, "s3://")
		parts := strings.SplitN(bucket, "/", 2)
		return stream.NewFS(parts[0])
	}
	return lfs.New(path)
}

// noopFS is a dummy filesystem for spool's output (we only read, never write)
type noopFS struct{}

func (n *noopFS) Open(name string) (fs.File, error) {
	return nil, fmt.Errorf("noop filesystem does not support Open")
}

func (n *noopFS) Create(path string, attr *struct{}) (spool.File, error) {
	return &noopFile{}, nil
}

func (n *noopFS) Remove(path string) error {
	return nil
}

type noopFile struct{}

func (n *noopFile) Write(p []byte) (int, error) { return len(p), nil }
func (n *noopFile) Close() error                { return nil }
func (n *noopFile) Stat() (fs.FileInfo, error)  { return nil, fmt.Errorf("not implemented") }
func (n *noopFile) Cancel() error               { return nil }
