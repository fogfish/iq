package compiler_test

import (
	"testing"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
)

func TestAnchorKeyComputer_RegularStep(t *testing.T) {
	// Create workflow with regular step with emit
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

	inputKey := iosystem.Key("a.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	want := iosystem.Key("summary/a.txt")
	if anchorKey != want {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, want)
	}
}

func TestAnchorKeyComputer_ForeachStep(t *testing.T) {
	// Create workflow with foreach step
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.ForeachStep{
						Emit: "processed",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	inputKey := iosystem.Key("a.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	// For foreach, anchor is array file with emit prefix
	want := iosystem.Key("processed/a.txt")
	if anchorKey != want {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, want)
	}
}

func TestAnchorKeyComputer_NoEmit(t *testing.T) {
	// Create workflow without emit
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

	inputKey := iosystem.Key("a.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	// No emit, anchor same as input
	if anchorKey != inputKey {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, inputKey)
	}
}

func TestAnchorKeyComputer_RouterStep(t *testing.T) {
	// Create workflow with router step
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.RouterStep{
						Emit: "routed",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	inputKey := iosystem.Key("b.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	want := iosystem.Key("routed/b.txt")
	if anchorKey != want {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, want)
	}
}

func TestAnchorKeyComputer_RunStep(t *testing.T) {
	// Create workflow with run step
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.RunStep{
						Emit: "output",
					},
				},
			},
		},
	}

	anchor, err := compiler.NewAnchorKeyComputer(workflow)
	if err != nil {
		t.Fatalf("NewAnchorKeyComputer failed: %v", err)
	}

	inputKey := iosystem.Key("c.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	want := iosystem.Key("output/c.txt")
	if anchorKey != want {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, want)
	}
}

func TestAnchorKeyComputer_SubdirectoryKey(t *testing.T) {
	// Test with subdirectory input key
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

	inputKey := iosystem.Key("subdir/file.txt")
	anchorKey := anchor.ComputeAnchorKey(inputKey)

	want := iosystem.Key("out/subdir/file.txt")
	if anchorKey != want {
		t.Errorf("ComputeAnchorKey() = %q, want %q", anchorKey, want)
	}
}

func TestAnchorKeyComputer_NoJobs(t *testing.T) {
	// Create workflow with no jobs
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs:       map[string]*compiler.Job{},
	}

	_, err := compiler.NewAnchorKeyComputer(workflow)
	if err == nil {
		t.Error("Expected error for workflow with no jobs")
	}
}

func TestAnchorKeyComputer_NoSteps(t *testing.T) {
	// Create workflow with job but no steps
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "main",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name:  "main",
				Steps: []compiler.Step{},
			},
		},
	}

	_, err := compiler.NewAnchorKeyComputer(workflow)
	if err == nil {
		t.Error("Expected error for job with no steps")
	}
}

func TestAnchorKeyComputer_MissingEntrypoint(t *testing.T) {
	// Create workflow with missing entrypoint
	workflow := &compiler.Workflow{
		Name:       "test",
		Entrypoint: "missing",
		Jobs: map[string]*compiler.Job{
			"main": {
				Name: "main",
				Steps: []compiler.Step{
					&compiler.AgentStep{Emit: "test"},
				},
			},
		},
	}

	_, err := compiler.NewAnchorKeyComputer(workflow)
	if err == nil {
		t.Error("Expected error for missing entrypoint job")
	}
}
