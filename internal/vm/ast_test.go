//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm_test

import (
	"context"
	"testing"

	"github.com/fogfish/golem/pipe/v2"
	"github.com/fogfish/golem/pure/monoid"
	"github.com/fogfish/iq/internal/vm"
	"github.com/fogfish/it/v2"
)

func identity(c *vm.Cell) (*vm.Cell, error) { return c, nil }

func TestASTConstruction(t *testing.T) {
	extract := pipe.Lift(identity)
	technical := pipe.Lift(identity)
	creative := pipe.Lift(identity)
	format := pipe.Lift(identity)

	prog := &vm.App{
		Root: vm.Seq{
			Steps: []vm.Node{
				vm.UnfoldNode{F: pipe.Lift(identity)},
				vm.MapNode{F: extract},
				vm.PartitionNode{
					F: func(c *vm.Cell) bool { return c.Key == "tech" },
					Match: vm.Seq{
						Steps: []vm.Node{
							vm.MapNode{F: technical},
						},
					},
					Default: vm.Seq{
						Steps: []vm.Node{
							vm.MapNode{F: creative},
						},
					},
				},
				vm.MapNode{F: format},
				vm.FoldNode{M: monoid.FromOp[*vm.Cell](nil, func(a, b *vm.Cell) *vm.Cell { return b })},
			},
		},
	}

	seq, ok := prog.Root.(vm.Seq)
	it.Then(t).Should(
		it.True(ok),
		it.Equal(len(seq.Steps), 5),
	)

	_, ok0 := seq.Steps[0].(vm.UnfoldNode)
	_, ok1 := seq.Steps[1].(vm.MapNode)
	p, ok2 := seq.Steps[2].(vm.PartitionNode)

	it.Then(t).Should(
		it.True(ok0),
		it.True(ok1),
		it.True(ok2),
		it.True(p.F != nil),
	)
}

func TestFMapNode(t *testing.T) {
	prog := &vm.App{
		Root: vm.Seq{
			Steps: []vm.Node{
				vm.UnfoldNode{F: pipe.Lift(identity)},
				vm.FMapNode{FF: pipe.LiftF(func(_ context.Context, c *vm.Cell, out chan<- *vm.Cell) error {
					out <- c
					return nil
				})},
				vm.FoldNode{M: monoid.FromOp[*vm.Cell](nil, func(a, b *vm.Cell) *vm.Cell { return b })},
			},
		},
	}

	seq := prog.Root.(vm.Seq)
	_, ok := seq.Steps[1].(vm.FMapNode)

	it.Then(t).Should(
		it.True(ok),
	)
}
