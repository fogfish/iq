//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm_test

import (
	"testing"

	"github.com/fogfish/iq/internal/vm"
	"github.com/fogfish/it/v2"
)

func TestASTConstruction(t *testing.T) {
	prog := &vm.Program{
		Root: vm.Seq{
			Steps: []vm.Node{
				vm.UnfoldNode{Name: "source"},
				vm.MapNode{Name: "extract"},
				vm.PartitionNode{
					Name: "classify",
					Match: vm.Seq{
						Steps: []vm.Node{
							vm.MapNode{Name: "technical"},
						},
					},
					Default: vm.Seq{
						Steps: []vm.Node{
							vm.MapNode{Name: "creative"},
						},
					},
				},
				vm.MapNode{Name: "format"},
				vm.FoldNode{Name: "sink"},
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
		it.Equal(p.Name, "classify"),
	)
}

func TestFMapNode(t *testing.T) {
	prog := &vm.Program{
		Root: vm.Seq{
			Steps: []vm.Node{
				vm.UnfoldNode{Name: "source"},
				vm.FMapNode{Name: "foreach"},
				vm.FoldNode{Name: "sink"},
			},
		},
	}

	seq := prog.Root.(vm.Seq)
	_, ok := seq.Steps[1].(vm.FMapNode)

	it.Then(t).Should(
		it.True(ok),
	)
}
