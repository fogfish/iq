//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source

import (
	"fmt"
	"os"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

// Builder creates iosystem.Source instances from CLI flags.
// Supports stdin, files, and merge modes.
type Builder struct {
	src iosystem.Source
	err error
}

// New creates a source builder.
func New() *Builder {
	return &Builder{}
}

// Stdin sets the input to read from stdin.
// If reader is nil, os.Stdin is used.
func (b *Builder) Stdin() *Builder {
	if b.err != nil || b.src != nil {
		return b
	}

	if HasStdinBytes() {
		b.src = source.NewStdin()
	}

	return b
}

// Files sets the input to read from file paths.
// Multiple files will be read sequentially unless Merge is enabled.
func (b *Builder) Files(dir string, file ...string) *Builder {
	if b.err != nil || b.src != nil || (dir == "" && len(file) == 0) {
		return b
	}

	if dir == "" {
		dir = "."
	}

	// if
	src, err := storage.NewFileSystem(dir, file...)
	if err != nil {
		b.err = fmt.Errorf("failed to mount input dir %s: %w", dir, err)
		return b
	}

	b.src, b.err = source.NewStorage(src, "")
	return b
}

func (b *Builder) None() *Builder {
	if b.err != nil || b.src != nil {
		return b
	}

	b.src = source.NewNone()
	return b
}

// Build creates the source based on configuration.
func (b *Builder) Build() (iosystem.Source, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.src == nil {
		return nil, fmt.Errorf("no input specified")
	}

	return b.src, b.err
}

// Checks if stdin has data available (not a TTY).
func HasStdinBytes() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) == 0
}
