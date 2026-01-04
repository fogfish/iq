//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

import (
	"fmt"

	"github.com/fogfish/iq/internal/iosystem"
)

// AnchorKeyComputer computes expected output keys for skip-if-exists logic.
type AnchorKeyComputer struct {
	lastStep Step
}

// NewAnchorKeyComputer creates a computer from workflow's last step.
func NewAnchorKeyComputer(workflow *Workflow) (*AnchorKeyComputer, error) {
	if len(workflow.Jobs) == 0 {
		return nil, fmt.Errorf("workflow has no jobs")
	}

	// Get entrypoint job (default to "main")
	entrypoint := workflow.Entrypoint
	if entrypoint == "" {
		entrypoint = "main"
	}

	job, ok := workflow.Jobs[entrypoint]
	if !ok {
		return nil, fmt.Errorf("entrypoint job %q not found", entrypoint)
	}

	if len(job.Steps) == 0 {
		return nil, fmt.Errorf("job %q has no steps", entrypoint)
	}

	// Get last step (anchor)
	lastStep := job.Steps[len(job.Steps)-1]

	return &AnchorKeyComputer{
		lastStep: lastStep,
	}, nil
}

// ComputeAnchorKey calculates the expected output key for a document.
// This is the key that would be written by the last step in the workflow.
//
// Logic:
//   - Regular step: apply emit prefix to input key
//   - Foreach step: check for array file (JSONL or JSON)
//   - No emit: anchor is same as input key
func (c *AnchorKeyComputer) ComputeAnchorKey(inputKey iosystem.Key) iosystem.Key {
	// Check step type and extract emit
	var emit string
	var isForeach bool

	switch step := c.lastStep.(type) {
	case *AgentStep:
		emit = step.Emit
	case *RouterStep:
		emit = step.Emit
	case *ForeachStep:
		emit = step.Emit
		isForeach = true
	case *RunStep:
		emit = step.Emit
	default:
		// Unknown step type, use input key
		return inputKey
	}

	// For foreach, output is array file (not individual elements)
	// Check for emit prefix application
	if isForeach {
		if emit != "" {
			return ApplyEmit(emit, inputKey)
		}
		return inputKey
	}

	// Regular step: apply emit if present
	if emit != "" {
		return ApplyEmit(emit, inputKey)
	}

	return inputKey
}
