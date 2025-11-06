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

	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/opts"
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

	b.r, b.err = source.Mount(path)
	return b
}

func (b *Builder) Writer(path string) *Builder {
	if b.err != nil {
		return b
	}
	b.w, b.err = sink.Mount(path)
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
