//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm

// Gist is the polymorphic content interface.
// Implementations exist in runtime package (Text, Json, List, Binary).
// The vm package defines the interface; concrete types are in the domain layer.
type Gist interface {
	ContentType() string
}

// Cell is the unit of data flowing through the pipeline.
// It is a reference to a mutable memory location within the workflow's
// address space. Channels pass *Cell pointers — lightweight, zero-copy.
type Cell struct {
	// Key is the document identity (filesystem-style relative path).
	Key string

	// Value is the polymorphic content (Text, Json, List, Binary).
	Value Gist

	// Steps holds named outputs from prior steps (shared across cells).
	// Downstream stages reference these via Go templates (e.g., {{.steps.extract}}).
	Steps map[string]Gist

	// Env holds environment/metadata key-value pairs.
	Env map[string]any
}

// NewCell creates a Cell with initialized maps.
func NewCell(key string, value Gist) *Cell {
	return &Cell{
		Key:   key,
		Value: value,
		Steps: make(map[string]Gist),
		Env:   make(map[string]any),
	}
}

// Copy creates a shallow copy of the Cell with a new Value.
// Steps and Env maps are shared (not deep-copied) — this is intentional
// for shared memory semantics within a pipeline.
func (c *Cell) Copy(value Gist) *Cell {
	return &Cell{
		Key:   c.Key,
		Value: value,
		Steps: c.Steps,
		Env:   c.Env,
	}
}
