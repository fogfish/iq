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
	"github.com/goccy/go-yaml"
	"github.com/kshard/chatter"
)

// Worker abstraction
type Worker interface {
	Prompt(ctx context.Context, key iosystem.Key, input any, opt ...chatter.Opt) (any, error)
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
func (p *Agent) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
	// Passthrough EOF or empty
	if len(docs) == 0 || (len(docs) == 1 && docs[0].Type == iosystem.ContentEOF) {
		return docs, nil
	}

	// Inject document key into cache context for step-level caching
	// if len(docs) > 0 {
	// 	cacheCtx := compiler.GetCacheContext(ctx)
	// 	if cacheCtx != nil {
	// 		cacheCtx.DocumentKey = iosystem.Key(docs[0].Path)
	// 		ctx = compiler.WithCacheContext(ctx, cacheCtx)
	// 	}
	// }

	var input any
	key := docs[0].Key

	// Create emit capture to retrieve emit context after workflow execution
	// ctx, emitCapture := compiler.WithEmitCapture(ctx)

	items := make([]any, 0, len(docs))
	for _, doc := range docs {
		content, err := p.decode(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to read document: %w", err)
		}
		items = append(items, content)
	}
	input = items

	if len(items) == 1 {
		input = items[0]
	}

	result, err := p.w.Prompt(ctx, key, input, p.config.Options...)
	if err != nil {
		docPath := docs[0].Path
		if len(docs) > 1 {
			docPath = fmt.Sprintf("%s (array of %d)", docPath, len(docs))
		}
		return nil, fmt.Errorf("agent processing failed for '%s': %w", docPath, err)
	}

	reply, err := p.encode(result)
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent response: %w", err)
	}

	reply.Key = docs[0].Key
	reply.Path = docs[0].Path + p.config.Suffix
	reply.Metadata = copyMetadata(docs[0].Metadata)

	// Store captured emit context in document metadata
	// if emitCapture != nil && emitCapture.Captured != nil {
	// 	if reply.Metadata.Custom == nil {
	// 		reply.Metadata.Custom = make(map[string]string)
	// 	}
	// 	reply.Metadata.Custom["emit.prefix"] = emitCapture.Captured.Prefix
	// 	// Store counters as JSON if present
	// 	if len(emitCapture.Captured.Counters) > 0 {
	// 		countersJSON, _ := json.Marshal(emitCapture.Captured.Counters)
	// 		reply.Metadata.Custom["emit.counters"] = string(countersJSON)
	// 	}
	// }

	return []*iosystem.Document{reply}, nil
}

// prepare the document for processing by the blueprint.
func (p *Agent) decode(doc *iosystem.Document) (any, error) {
	// Read document content
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	switch doc.Type {
	case iosystem.ContentJSON:
		var input map[string]any
		if err := json.Unmarshal(content, &input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON document: %w", err)
		}
		return input, nil
	case iosystem.ContentYAML:
		var input map[string]any
		if err := yaml.Unmarshal(content, &input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML document: %w", err)
		}
		return input, nil
	default:
		return content, nil
	}
}

func (p *Agent) encode(reply any) (*iosystem.Document, error) {
	switch v := reply.(type) {
	case string:
		return &iosystem.Document{
			Reader: bytes.NewReader([]byte(v)),
			Type:   iosystem.ContentText,
		}, nil
	case []byte:
		return &iosystem.Document{
			Reader: bytes.NewReader(v),
			Type:   iosystem.ContentStream,
		}, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal agent response: %w", err)
		}
		return &iosystem.Document{
			Reader: bytes.NewReader(data),
			Type:   iosystem.ContentJSON,
		}, nil
	}
}

// Close releases resources. For AgentProcessor, this is a no-op
// as the agent lifecycle is managed externally.
func (p *Agent) Close() error {
	return nil
}

// copyMetadata creates a shallow copy of metadata.
func copyMetadata(m iosystem.Metadata) iosystem.Metadata {
	copy := iosystem.Metadata{
		ContentType: m.ContentType,
		Extension:   m.Extension,
		Size:        m.Size,
	}
	if m.Custom != nil {
		copy.Custom = make(map[string]string, len(m.Custom))
		for k, v := range m.Custom {
			copy.Custom[k] = v
		}
	}
	return copy
}
