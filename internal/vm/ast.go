//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm

// Program is the root of a control-flow AST.
type Program struct {
	Root Node
}

// Node represents a control-flow operation in the pipeline.
// All node types implement this sealed interface.
type Node interface {
	node() // sealed marker
}

// Seq executes a list of nodes sequentially as a channel chain.
// Data flows: ch0 → Step[0] → ch1 → Step[1] → ch2 → ...
type Seq struct {
	Steps []Node
}

func (Seq) node() {}

// MapNode applies a function F[*Cell, *Cell] to each element (1:1 transform).
// Name is the vtable key used to bind a concrete function in Phase 2.
type MapNode struct {
	Name string
}

func (MapNode) node() {}

// FMapNode applies a function FF[*Cell, *Cell] to each element (1:N fan-out).
// Used for document splitting, foreach expansion, etc.
// Name is the vtable key used to bind a concrete function in Phase 2.
type FMapNode struct {
	Name string
}

func (FMapNode) node() {}

// PartitionNode splits a stream by predicate into two branches.
// Elements matching the predicate flow to Match; others flow to Default.
// Both branches are sub-pipelines that run concurrently.
// Name is the vtable key for the predicate function.
type PartitionNode struct {
	Name    string // vtable key for predicate
	Match   Node   // sub-pipeline for matched elements
	Default Node   // sub-pipeline for unmatched elements
}

func (PartitionNode) node() {}

// FoldNode terminates a stream by consuming all elements.
// Name is the vtable key for the fold function.
type FoldNode struct {
	Name string
}

func (FoldNode) node() {}

// UnfoldNode produces a stream from a source.
// Name is the vtable key for the source function.
type UnfoldNode struct {
	Name string
}

func (UnfoldNode) node() {}

// JoinNode merges multiple sub-pipeline outputs into a single stream.
// Inputs are the nodes whose output channels will be merged.
type JoinNode struct {
	Inputs []Node
}

func (JoinNode) node() {}
