//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package core

type Format int

const (
	FORMAT_TEXT Format = iota
	FORMAT_JSON
)

// The core object used by iq for LLM I/O
type Prompt struct {
	// The textual description of the task to complete
	Task string

	// The input blob for processing by the task
	Blob string

	// The output format
	Format Format
}
