//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm

import (
	"github.com/fogfish/golem/pipe/v2"
	"github.com/fogfish/golem/pure/monoid"
)

// App is the root of a control-flow AST.
type App struct {
	Root Node
}

// Node represents a control-flow operation in the pipeline.
type Node interface{ HKT1(Node) }

// Seq executes a list of nodes sequentially as a channel chain.
// Data flows: ch0 → Step[0] → ch1 → Step[1] → ch2 → ...
type Seq struct {
	Steps []Node
}

func (Seq) HKT1(Node) {}

// MapNode applies a function F[*Cell, *Cell] to each element (1:1 transform).
type MapNode struct {
	F pipe.F[*Cell, *Cell]
}

func (MapNode) HKT1(Node) {}

// FMapNode applies a function FF[*Cell, *Cell] to each element (1:N fan-out).
type FMapNode struct {
	FF pipe.FF[*Cell, *Cell]
}

func (FMapNode) HKT1(Node) {}

// PartitionNode splits a stream by predicate into two branches.
type PartitionNode struct {
	F       func(*Cell) bool
	Match   Node
	Default Node
}

func (PartitionNode) HKT1(Node) {}

// FoldNode terminates a stream by consuming all elements.
type FoldNode struct {
	M monoid.Monoid[*Cell]
}

func (FoldNode) HKT1(Node) {}

// UnfoldNode produces a stream from a source.
type UnfoldNode struct {
	F pipe.F[*Cell, *Cell]
}

func (UnfoldNode) HKT1(Node) {}

// JoinNode merges multiple sub-pipeline outputs into a single stream.
// Inputs are the nodes whose output channels will be merged.
type JoinNode struct {
	Inputs []Node
}

func (JoinNode) HKT1(Node) {}
