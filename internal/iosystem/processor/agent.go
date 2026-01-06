//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/blueprint/runtime"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/kshard/chatter"
)

// Worker abstraction
type Worker interface {
	Prompt(ctx context.Context, in runtime.Event, opt ...chatter.Opt) (runtime.Event, error)
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
	codecs *codec.Registry
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
		codecs: codec.Default,
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
func (p *Agent) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	if iosystem.IsEOF(docs) {
		return docs, nil
	}

	input, err := p.decode(docs)
	if err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	reply, err := p.w.Prompt(ctx, input, p.config.Options...)
	if err != nil {
		return nil, fmt.Errorf("agent processing failed for '%s': %w", docs[0].Key, err)
	}

	doc, err := p.encode(reply)
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent response: %w", err)
	}

	return []*iosystem.Document{doc}, nil
}

// decode prepares the document for processing by the blueprint.
func (p *Agent) decode(docs []*iosystem.Document) (runtime.Event, error) {
	data := make([]any, 0, len(docs))
	for _, doc := range docs {
		content, err := p.codecs.Decode(doc.Reader, doc.Type)
		if err != nil {
			return runtime.Event{}, fmt.Errorf("failed to read document: %w", err)
		}
		data = append(data, content)
	}

	if len(data) == 1 {
		doc, err := runtime.ToGist(data[0])
		if err != nil {
			return runtime.Event{}, fmt.Errorf("input conversion failed: %w", err)
		}
		return runtime.NewEvent(docs[0].Key, doc), nil
	}

	doc, err := runtime.ToGist(data)
	if err != nil {
		return runtime.Event{}, fmt.Errorf("input conversion failed: %w", err)
	}
	return runtime.NewEvent(docs[0].Key, doc), nil
}

func (p *Agent) encode(reply runtime.Event) (*iosystem.Document, error) {
	raw, err := p.codecs.Encode(reply.Current, reply.Current.ContentType())
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent reply: %w", err)
	}

	return iosystem.NewDocument(reply.Key, raw), nil
}

// Close releases resources. For AgentProcessor, this is a no-op
// as the agent lifecycle is managed externally.
func (p *Agent) Close() error {
	return nil
}
