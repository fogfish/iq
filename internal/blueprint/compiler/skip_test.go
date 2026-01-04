package compiler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
)

func TestSkipChecker_ShouldSkip(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	// Create workflow
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.AgentStep{
						Emit: "summary",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	checker := compiler.NewSkipChecker(store, anchor, nil)

	ctx := context.Background()
	inputKey := iosystem.Key("a.txt")
	anchorKey := iosystem.Key("summary/a.txt")

	// Should not skip when anchor doesn't exist
	skip, err := checker.ShouldSkip(ctx, inputKey)
	if err != nil {
		t.Fatalf("ShouldSkip failed: %v", err)
	}
	if skip {
		t.Error("Should not skip when anchor doesn't exist")
	}

	// Create anchor file
	err = store.Put(ctx, anchorKey, strings.NewReader("output"))
	if err != nil {
		t.Fatalf("Failed to create anchor file: %v", err)
	}

	// Should skip when anchor exists
	skip, err = checker.ShouldSkip(ctx, inputKey)
	if err != nil {
		t.Fatalf("ShouldSkip failed: %v", err)
	}
	if !skip {
		t.Error("Should skip when anchor exists")
	}
}

func TestSkipChecker_NoEmit(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	// Workflow without emit - anchor same as input
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.AgentStep{
						Emit: "",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	checker := compiler.NewSkipChecker(store, anchor, nil)

	ctx := context.Background()
	inputKey := iosystem.Key("b.txt")

	// Should not skip when file doesn't exist
	skip, err := checker.ShouldSkip(ctx, inputKey)
	if err != nil {
		t.Fatalf("ShouldSkip failed: %v", err)
	}
	if skip {
		t.Error("Should not skip when file doesn't exist")
	}

	// Create file with same name as input
	err = store.Put(ctx, inputKey, strings.NewReader("output"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Should skip when file exists
	skip, err = checker.ShouldSkip(ctx, inputKey)
	if err != nil {
		t.Fatalf("ShouldSkip failed: %v", err)
	}
	if !skip {
		t.Error("Should skip when file exists")
	}
}

func TestSkipChecker_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFS(tmpDir)
	if err != nil {
		t.Fatalf("NewFS failed: %v", err)
	}

	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.AgentStep{
						Emit: "out",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	checker := compiler.NewSkipChecker(store, anchor, nil)

	ctx := context.Background()

	// Check multiple files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	
	// Initially none should be skipped
	for _, file := range files {
		skip, err := checker.ShouldSkip(ctx, iosystem.Key(file))
		if err != nil {
			t.Fatalf("ShouldSkip failed for %s: %v", file, err)
		}
		if skip {
			t.Errorf("Should not skip %s initially", file)
		}
	}

	// Create output for file2
	err = store.Put(ctx, iosystem.Key("out/file2.txt"), strings.NewReader("output"))
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	// Only file2 should be skipped
	expected := map[string]bool{
		"file1.txt": false,
		"file2.txt": true,
		"file3.txt": false,
	}

	for file, shouldSkip := range expected {
		skip, err := checker.ShouldSkip(ctx, iosystem.Key(file))
		if err != nil {
			t.Fatalf("ShouldSkip failed for %s: %v", file, err)
		}
		if skip != shouldSkip {
			t.Errorf("For %s: got skip=%v, want %v", file, skip, shouldSkip)
		}
	}
}
