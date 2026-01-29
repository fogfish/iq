//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/kshard/chatter"
)

// Input/Output content
type Gist interface {
	HKT1(Gist)
	ContentType() string
}

// Input/Output content is plain text
type Text string

func (Text) HKT1(Gist)           {}
func (Text) ContentType() string { return codec.ContentMarkdown }

// Input/Output content is JSON
type Json map[string]any

func (Json) HKT1(Gist)           {}
func (Json) ContentType() string { return codec.ContentJSON }

// Input/Output content is a JSON array
type List []any

func (List) HKT1(Gist)           {}
func (List) ContentType() string { return codec.ContentJSON }

// Input/Output content is binary data (images, files, etc)
type Binary struct {
	Type string // MIME type (e.g., "image/png", "image/jpeg")
	Data []byte
}

func (Binary) HKT1(Gist)             {}
func (b Binary) ContentType() string { return b.Type }

func ToGist(x any) (Gist, error) {
	switch v := x.(type) {
	case string:
		return Text(v), nil
	case []byte:
		return Text(v), nil
	case map[string]any:
		// Check if this is a Binary type reconstructed from JSON cache
		if typ, hasType := v["Type"].(string); hasType {
			if data, hasData := v["Data"]; hasData {
				// Try to decode the base64-encoded data
				var dataBytes []byte
				switch d := data.(type) {
				case string:
					// JSON unmarshaler decodes base64 strings to []byte, but sometimes it's still a string
					decoded, err := base64DecodeIfNeeded(d)
					if err != nil {
						return nil, fmt.Errorf("failed to decode binary data: %w", err)
					}
					dataBytes = decoded
				case []byte:
					dataBytes = d
				default:
					// Not a valid Binary representation, treat as regular JSON
					return Json(v), nil
				}

				return Binary{Type: typ, Data: dataBytes}, nil
			}
		}
		// Regular JSON object
		return Json(v), nil
	case []any:
		return List(v), nil
	case Gist:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported Gist type: %T", v)
	}
}

// base64DecodeIfNeeded attempts to decode a base64 string
// If it's already raw bytes or fails to decode, returns as-is
func base64DecodeIfNeeded(s string) ([]byte, error) {
	// First try standard base64
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}
	// Try URL encoding
	if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}
	// If it doesn't decode, assume it's raw string data
	return []byte(s), nil
}

// Event represents an event processed by the workflow.
type Event struct {
	// Unique identity of the document
	Key iosystem.Key

	// Original workflow input
	Document Gist

	// Current input being processed
	Current Gist

	// Named step outputs
	Steps map[string]Gist
}

func NewEvent(key iosystem.Key, doc Gist) Event {
	return Event{
		Key:      key,
		Document: doc,
		Current:  doc,
		Steps:    make(map[string]Gist),
	}
}

func (evt Event) copy(key iosystem.Key, current Gist) Event {
	return Event{
		Key:      key,
		Document: evt.Document,
		Current:  current,
		Steps:    evt.Steps,
	}
}

func (evt Event) ToState() map[string]any {
	return map[string]any{
		ast.ContextKeyDocument: evt.Document,
		ast.ContextKeyInput:    evt.Current,
		ast.ContextKeyCurrent:  evt.Current,
		ast.ContextKeySteps:    evt.Steps,
		ast.ContextKeyEnv: map[string]any{
			"time": time.Now().Format(time.RFC3339),
		},
	}
}

type Prompter interface {
	Config(map[string]*Job) error
	Prompt(context.Context, Event, ...chatter.Opt) (Event, error)
}
