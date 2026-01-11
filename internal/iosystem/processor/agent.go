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
	"path/filepath"
	"strings"

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

	result, err := p.encode(docs[0].Key, reply)
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent response: %w", err)
	}

	return result, nil
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

func (p *Agent) encode(key iosystem.Key, reply runtime.Event) ([]*iosystem.Document, error) {
	// Check if Current is Binary type - handle specially for multiple images
	if bin, ok := reply.Current.(runtime.Binary); ok {
		return p.encodeSingleBinary(key, bin)
	}

	// Check if Current is List of Binary - handle multiple images
	if list, ok := reply.Current.(runtime.List); ok {
		binaries := make([]runtime.Binary, 0)
		for _, item := range list {
			if bin, ok := item.(runtime.Binary); ok {
				binaries = append(binaries, bin)
			}
		}
		if len(binaries) > 0 {
			return p.encodeMultipleBinaries(key, binaries)
		}
	}

	// Standard text/JSON encoding
	raw, err := p.codecs.Encode(reply.Current, reply.Current.ContentType())
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent reply: %w", err)
	}

	doc := iosystem.NewDocument(key, reply.Current.ContentType(), raw)
	doc.EnsureExtension()

	return []*iosystem.Document{doc}, nil
}

func (p *Agent) encodeSingleBinary(key iosystem.Key, bin runtime.Binary) ([]*iosystem.Document, error) {
	// Use codec to re-encode (this triggers JPEG conversion in ImageCodec)
	raw, err := p.codecs.Encode(bin.Data, bin.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to encode binary: %w", err)
	}

	// Always output as JPEG after re-encoding
	doc := iosystem.NewDocument(key, codec.ContentJPG, raw)
	doc.EnsureExtension()

	return []*iosystem.Document{doc}, nil
}

func (p *Agent) encodeMultipleBinaries(key iosystem.Key, binaries []runtime.Binary) ([]*iosystem.Document, error) {
	docs := make([]*iosystem.Document, 0, len(binaries))

	for i, bin := range binaries {
		// Re-encode via codec (triggers JPEG conversion)
		raw, err := p.codecs.Encode(bin.Data, bin.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to encode binary %d: %w", i+1, err)
		}

		// Generate numbered key: "image.0001.jpg", "image.0002.jpg"
		numberedKey := addNumberToKey(key, i+1)

		doc := iosystem.NewDocument(numberedKey, codec.ContentJPG, raw)
		doc.EnsureExtension()

		docs = append(docs, doc)
	}

	return docs, nil
}

// addNumberToKey creates a numbered filename like "image.0001.jpg"
func addNumberToKey(key iosystem.Key, num int) iosystem.Key {
	// Remove extension
	keyStr := string(key)
	ext := filepath.Ext(keyStr)
	base := strings.TrimSuffix(keyStr, ext)

	// Add number with 4-digit padding
	return iosystem.Key(fmt.Sprintf("%s.%04d.jpg", base, num))
}

// Close releases resources. For AgentProcessor, this is a no-op
// as the agent lifecycle is managed externally.
func (p *Agent) Close() error {
	return nil
}
