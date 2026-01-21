//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/goccy/go-yaml"
	"github.com/kshard/chatter"
)

type Cache struct {
	workflow string
	job      string
	step     string
	hash     string // Pre-computed SHA256 hash (first 6 hex chars)
	cache    storage.Storage

	Prompter
}

var _ Prompter = (*Cache)(nil)

// CacheMetadata represents the YAML front matter in cache files
type CacheMetadata struct {
	Key         string    `yaml:"key"`
	Workflow    string    `yaml:"workflow"`
	Job         string    `yaml:"job"`
	Step        string    `yaml:"step"`
	ContentType string    `yaml:"content_type"`
	Timestamp   time.Time `yaml:"timestamp"`
}

func NewCache(cache storage.Storage, workflow, job, step, hash string, p Prompter) *Cache {
	return &Cache{
		cache:    cache,
		workflow: workflow,
		job:      job,
		step:     step,
		hash:     hash,
		Prompter: p,
	}
}

// generateKey builds cache key using in.Key which already contains iteration info.
// Format: workflow/job/step-name-hash/doc-key.md
// The docKey already contains .SeqID info from foreach iterations.
func (e *Cache) generateKey(docKey iosystem.Key) iosystem.Key {
	// Simple format: workflow/job/step-hash
	stepKey := e.step
	if e.hash != "" {
		stepKey = e.step + "-" + e.hash
	}
	cachePath := filepath.Join(e.workflow, e.job, stepKey)

	// Append document key to make each document's cache unique
	// The docKey already contains .SeqID info from foreach
	// Add .md extension for markdown format
	return iosystem.Key(filepath.Join(cachePath, string(docKey)+".md"))
}

func (e *Cache) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	// Generate cache key using in.Key (already has iteration sequence)
	key := e.generateKey(in.Key)
	has, err := e.cache.Has(ctx, key)
	if err != nil {
		return in, err
	}

	if has {
		r, err := e.cache.Get(ctx, key)
		if err != nil {
			return in, err
		}

		// Read the entire content
		content, err := io.ReadAll(r)
		if err != nil {
			return in, err
		}

		// Parse markdown with YAML front matter
		cachedKey, gist, err := e.parseCacheFile(content)
		if err != nil {
			return in, err
		}

		in.Current = gist
		in.Key = cachedKey
		return in, nil
	}

	val, err := e.Prompter.Prompt(ctx, in, opts...)
	if err != nil {
		return in, err
	}

	// Write cache with YAML front matter
	content, err := e.writeCacheFile(val.Key, val.Current)
	if err != nil {
		slog.Error("cache failed", "err", err)
		return val, nil
	}

	err = e.cache.Put(ctx, key, bytes.NewReader(content))
	if err != nil {
		slog.Error("cache failed", "err", err)
		return val, nil
	}

	return val, nil
}

// writeCacheFile creates a markdown file with YAML front matter
func (e *Cache) writeCacheFile(docKey iosystem.Key, gist any) ([]byte, error) {
	// Determine content type and marshal content
	var content []byte
	var contentType string
	var err error

	// Get the actual Gist type to determine content type
	if g, ok := gist.(Gist); ok {
		contentType = g.ContentType()
		// For JSON, marshal with indentation
		// For text/markdown, use as-is (binary string)
		if contentType == "application/json" {
			content, err = json.MarshalIndent(gist, "", "  ")
			if err != nil {
				return nil, err
			}
		} else {
			content = fmt.Appendf(nil, "%v", gist)
		}
	} else {
		// Fallback: marshal as JSON
		contentType = "application/json"
		content, err = json.MarshalIndent(gist, "", "  ")
		if err != nil {
			return nil, err
		}
	}

	meta := CacheMetadata{
		Key:         string(docKey),
		Workflow:    e.workflow,
		Job:         e.job,
		Step:        e.step,
		ContentType: contentType,
		Timestamp:   time.Now(),
	}

	// Marshal metadata to YAML
	frontMatter, err := yaml.Marshal(meta)
	if err != nil {
		return nil, err
	}

	// Construct file with YAML front matter and content directly
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontMatter)
	buf.WriteString("---\n\n")
	buf.Write(content)

	return buf.Bytes(), nil
}

// parseCacheFile reads a markdown file with YAML front matter
func (e *Cache) parseCacheFile(content []byte) (iosystem.Key, Gist, error) {
	// Split front matter and content
	parts := bytes.SplitN(content, []byte("---"), 3)
	if len(parts) < 3 {
		return "", nil, fmt.Errorf("invalid cache file format: missing front matter")
	}

	// Parse YAML front matter
	var meta CacheMetadata
	if err := yaml.Unmarshal(parts[1], &meta); err != nil {
		return "", nil, fmt.Errorf("failed to parse cache metadata: %w", err)
	}

	// Extract content after front matter (skip leading whitespace)
	body := bytes.TrimSpace(parts[2])

	// Parse content based on content type
	var gist any
	var err error

	if meta.ContentType == "application/json" {
		// Parse JSON directly
		if err := json.Unmarshal(body, &gist); err != nil {
			return "", nil, fmt.Errorf("failed to parse cache gist: %w", err)
		}
	} else {
		// Text/markdown content - use as-is
		gist = string(body)
	}

	// Convert gist to proper type
	result, err := ToGist(gist)
	if err != nil {
		return "", nil, err
	}

	return iosystem.Key(meta.Key), result, nil
}
