//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/kshard/chatter"
)

// Worker abstraction
type Worker interface {
	Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error)
}

// Agent wraps a blueprint to use as a pipeline processor.
// It processes documents through the agent's Prompt() method and returns
// the agent's response as a new document.
//
// The processor:
//   - Reads document content as input
//   - Passes it to agent.Prompt()
//   - Returns agent response as new document
//   - Preserves document path with optional suffix
//   - Supports JSON format output from agents
//
// Example:
//
//	agent := getAgentFromBlueprint()
//	proc := NewAgentProcessor(agent, AgentConfig{
//	    Suffix: ".processed",
//	})
//	docs, err := proc.Process(ctx, inputDoc)
type Agent struct {
	w      Worker
	config AgentConfig
}

// AgentConfig configures the agent processor.
type AgentConfig struct {
	Suffix  string        // Suffix to add to output document path (default: empty)
	Options []chatter.Opt // Chatter options to pass to agent.Prompt() (temperature, max_tokens, etc.)
}

// NewAgent creates a processor that wraps a blueprint Agent.
func NewAgent(w Worker, config *AgentConfig) *Agent {
	p := &Agent{
		w:      w,
		config: AgentConfig{Options: []chatter.Opt{}},
	}

	if config != nil {
		p.config = *config
	}

	return p
}

// Process transforms a document by passing its content through the agent.
//
// Input document content is read and passed to agent.Prompt() as:
//   - string if content is valid UTF-8
//   - []byte otherwise
//   - map[string]any if document has metadata that should be templated
//
// The agent's response becomes the content of the output document.
// Output format depends on agent's configuration (text or JSON).
func (p *Agent) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Read document content
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	// Process through agent
	result, err := p.w.Prompt(ctx, content, p.config.Options...)
	if err != nil {
		return nil, fmt.Errorf("agent processing failed for '%s': %w", doc.Path, err)
	}

	// Convert result to bytes
	var reply []byte
	switch v := result.(type) {
	case string:
		reply = []byte(v)
	case []byte:
		reply = v
	default:
		reply, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal agent response for '%s': %w", doc.Path, err)
		}
	}

	// Create output document
	path := doc.Path
	if p.config.Suffix != "" {
		path = doc.Path + p.config.Suffix
	}

	out := &iosystem.Document{
		Path:     path,
		Reader:   bytes.NewReader(reply),
		Metadata: copyMetadata(doc.Metadata),
	}

	return []*iosystem.Document{out}, nil
}

// Close releases resources. For AgentProcessor, this is a no-op
// as the agent lifecycle is managed externally.
func (p *Agent) Close() error {
	return nil
}

// copyMetadata creates a shallow copy of metadata map.
func copyMetadata(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	copy := make(map[string]string, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
