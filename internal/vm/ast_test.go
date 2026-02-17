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
)

func TestASTConstruction(t *testing.T) {
	// Build a representative pipeline AST:
	// Unfold → Map("extract") → Partition("classify") → Join → Map("format") → Fold
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

	// Verify structure is constructable
	seq, ok := prog.Root.(vm.Seq)
	if !ok {
		t.Fatal("root should be Seq")
	}
	if len(seq.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(seq.Steps))
	}

	// Verify node types
	if _, ok := seq.Steps[0].(vm.UnfoldNode); !ok {
		t.Error("step 0 should be UnfoldNode")
	}
	if _, ok := seq.Steps[1].(vm.MapNode); !ok {
		t.Error("step 1 should be MapNode")
	}
	if p, ok := seq.Steps[2].(vm.PartitionNode); !ok {
		t.Error("step 2 should be PartitionNode")
	} else {
		if p.Name != "classify" {
			t.Errorf("partition name should be classify, got %s", p.Name)
		}
	}
}

func TestFMapNode(t *testing.T) {
	// FMapNode for foreach/chunker patterns
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
	if _, ok := seq.Steps[1].(vm.FMapNode); !ok {
		t.Error("step 1 should be FMapNode")
	}
}
