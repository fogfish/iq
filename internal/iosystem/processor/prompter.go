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
	"fmt"
	"io"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/kshard/chatter"
)

// Wrap LLM as processor
type Prompter struct {
	llm chatter.Chatter
}

// NewPrompter creates a processor that wraps a blueprint Prompter.
func NewPrompter(llm chatter.Chatter) *Prompter {
	return &Prompter{
		llm: llm,
	}
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
func (p *Prompter) Process(ctx context.Context, doc *iosystem.Document) ([]*iosystem.Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	content, err := p.decode(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	result, err := p.llm.Prompt(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("prompter processing failed for '%s': %w", doc.Path, err)
	}

	reply, err := p.encode(result)
	if err != nil {
		return nil, fmt.Errorf("failed to encode prompter response for '%s': %w", doc.Path, err)
	}

	for _, r := range reply {
		r.Path = doc.Path
		r.Metadata = copyMetadata(doc.Metadata)
	}

	return reply, nil
}

// prepare the document for processing by the blueprint.
func (p *Prompter) decode(doc *iosystem.Document) ([]chatter.Message, error) {
	// Read document content
	content, err := io.ReadAll(doc.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	return []chatter.Message{chatter.Text(content)}, nil
}

func (p *Prompter) encode(reply *chatter.Reply) ([]*iosystem.Document, error) {
	seq := []*iosystem.Document{}

	for _, c := range reply.Content {
		switch v := c.(type) {
		case chatter.Text:
			seq = append(seq, &iosystem.Document{
				Reader: strings.NewReader(string(v)),
				Type:   iosystem.ContentText,
			})
		case *chatter.Binary:
			seq = append(seq, &iosystem.Document{
				Reader: bytes.NewReader(v.Data),
				Type:   v.Type,
			})
		}
	}
	return seq, nil
}

// Close releases resources. For AgentProcessor, this is a no-op
// as the agent lifecycle is managed externally.
func (p *Prompter) Close() error {
	return nil
}
