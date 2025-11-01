//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package sink

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
)

// Builder creates iosystem.Sink instances from CLI flags.
// Supports stdout, single file, and directory outputs.
type Builder struct {
	snk iosystem.Sink
	err error
}

// New creates a sink builder.
func New() *Builder {
	return &Builder{}
}

// Stdout sets the output to write to stdout.
// If writer is provided, it will be used instead of os.Stdout.
func (b *Builder) Stdout(enabled bool) *Builder {
	if b.err != nil || b.snk != nil || !enabled {
		return b
	}

	b.snk = sink.NewStdout()
	return b
}

// File sets the output to write to a single file.
// All documents will be concatenated into this file.
func (b *Builder) File(path string) *Builder {
	if b.err != nil || b.snk != nil || len(path) == 0 {
		return b
	}

	if strings.HasPrefix(path, "s3://") {
		dir, base := filepath.Split(path[len("s3://"):])
		if dir == "" || base == "" {
			b.err = fmt.Errorf("invalid S3 file path: %s", path)
			return b
		}
		fs, err := stream.NewFS(dir)
		if err != nil {
			b.err = fmt.Errorf("failed to mount S3 bucket for path %s: %w", path, err)
			return b
		}
		b.snk, b.err = sink.NewFile(fs, "/"+base)
		return b
	}

	pabs, err := filepath.Abs(path)
	if err != nil {
		b.err = fmt.Errorf("failed to resolve path %s: %w", path, err)
		return b
	}

	dir, base := filepath.Split(pabs)
	fs, err := lfs.New(dir)
	if err != nil {
		b.err = fmt.Errorf("failed to mount directory %s: %w", dir, err)
		return b
	}

	b.snk, b.err = sink.NewFile(fs, "/"+base)
	return b
}

// Path sets the output directory.
// Documents will be written preserving their path structure.
func (b *Builder) Path(path string) *Builder {
	if b.err != nil || b.snk != nil || len(path) == 0 {
		return b
	}

	if strings.HasPrefix(path, "s3://") {
		fs, err := stream.NewFS(path[len("s3://"):])
		if err != nil {
			b.err = fmt.Errorf("failed to mount S3 bucket for path %s: %w", path, err)
			return b
		}
		b.snk, b.err = sink.NewFS(fs)
		return b
	}

	pabs, err := filepath.Abs(path)
	if err != nil {
		b.err = fmt.Errorf("failed to resolve path %s: %w", path, err)
		return b
	}

	// Use lfs for local paths to avoid versioning
	fs, err := lfs.New(pabs)
	if err != nil {
		b.err = fmt.Errorf("failed to mount path %s: %w", path, err)
		return b
	}

	b.snk, b.err = sink.NewFS(fs)
	return b
}

func (b *Builder) Build() (iosystem.Sink, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.snk == nil {
		return nil, fmt.Errorf("no output specified")
	}

	return b.snk, b.err
}
