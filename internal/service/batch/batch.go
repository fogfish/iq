//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package batch

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fogfish/opts"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
)

type Builder struct {
	r    spool.FileSystem
	w    spool.FileSystem
	opts []opts.Option[spool.Spool]
	err  error
}

func New() *Builder {
	return &Builder{
		opts: []opts.Option[spool.Spool]{},
	}
}

func (b *Builder) Reader(path string) *Builder {
	if b.err != nil {
		return b
	}

	b.r, b.err = Mount(path)
	return b
}

func (b *Builder) Writer(path string) *Builder {
	if b.err != nil {
		return b
	}
	b.w, b.err = Mount(path)
	return b
}

func (b *Builder) Mutable(enable bool) *Builder {
	if b.err != nil {
		return b
	}

	if enable {
		b.opts = append(b.opts, spool.IsMutable)
	} else {
		b.opts = append(b.opts, spool.IsImmutable)
	}
	return b
}

func (b *Builder) Strict(enable bool) *Builder {
	if b.err != nil {
		return b
	}

	if enable {
		b.opts = append(b.opts, spool.WithStrict)
	} else {
		b.opts = append(b.opts, spool.WithSkipError)
	}

	return b
}

func (b *Builder) FileExt(ext string) *Builder {
	if b.err != nil {
		return b
	}

	if len(ext) > 0 {
		b.opts = append(b.opts, spool.WithFileExt(ext))
	}

	return b
}

func (b *Builder) Build() (*spool.Spool, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.r == nil {
		return nil, fmt.Errorf("input is not defined")
	}
	if b.w == nil {
		return nil, fmt.Errorf("output is not defined")
	}

	return spool.New(b.r, b.w, b.opts...), nil
}

func Mount(path string) (spool.FileSystem, error) {
	if strings.HasPrefix(path, "s3://") {
		fs, err := stream.NewFS(path[len("s3://"):])
		if err != nil {
			return nil, fmt.Errorf("failed to mount S3 bucket for path %s: %w", path, err)
		}
		return fs, nil
	}

	pabs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	// Use lfs for local paths to avoid versioning
	fs, err := lfs.New(pabs)
	if err != nil {
		return nil, fmt.Errorf("failed to mount path %s: %w", path, err)
	}
	return fs, nil
}
