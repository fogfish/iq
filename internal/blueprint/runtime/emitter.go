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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
	"github.com/kshard/chatter"
)

/*

TODO:

🔴  2. Emitter Decorator (Empty Stub)
Current State:

What's Missing:

No emit context management
No emit prefix application to output keys
No foreach counter handling
Needs emit configuration passed from compiler


*/

type Emitter struct {
	prefix string
	sink   iosystem.Sink

	Prompter
}

var _ Prompter = (*Emitter)(nil)

func NewEmitter(sink iosystem.Sink, prefix string, p Prompter) *Emitter {
	return &Emitter{
		sink:     sink,
		prefix:   prefix,
		Prompter: p,
	}
}

func (e *Emitter) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	val, err := e.Prompter.Prompt(ctx, in, opts...)
	if err != nil {
		return in, err
	}

	// Handle Binary type specially
	if bin, ok := val.Current.(Binary); ok {
		prefixedKey := iosystem.Key(filepath.Join(e.prefix, string(val.Key)))
		dat, err := codec.Default.Encode(bin.Data, bin.Type)
		if err != nil {
			return in, fmt.Errorf("emitter: failed to encode binary: %w", err)
		}
		doc := iosystem.NewDocument(prefixedKey, codec.ContentJPG, dat)
		doc.EnsureExtension()

		path, err := e.sink.Write(ctx, doc)
		if err != nil {
			return in, fmt.Errorf("emitter: failed to put binary document at %s: %w", prefixedKey, err)
		}

		// Store emitted path in steps environment
		val.Steps[e.prefix] = Text(path)
		return val, nil
	}

	// Handle List of Binary (multiple images)
	if list, ok := val.Current.(List); ok {
		binaries := make([]Binary, 0)
		for _, item := range list {
			if bin, ok := item.(Binary); ok {
				binaries = append(binaries, bin)
			}
		}
		if len(binaries) > 0 {
			paths, err := e.emitMultipleBinariesWithPaths(ctx, val.Key, binaries)
			if err != nil {
				return in, err
			}
			// Store array of paths for multiple binaries
			pathsList := make([]any, len(paths))
			for i, p := range paths {
				pathsList[i] = p
			}
			val.Steps[e.prefix] = List(pathsList)
			return val, nil
		}
	}

	// Standard text/JSON encoding
	key := iosystem.Key(filepath.Join(e.prefix, string(val.Key)))
	dat, err := codec.Default.Encode(val.Current, val.Current.ContentType())
	if err != nil {
		return in, fmt.Errorf("emitter: failed to encode content: %w", err)
	}
	doc := iosystem.NewDocument(key, val.Current.ContentType(), dat)
	doc.EnsureExtension()

	path, err := e.sink.Write(ctx, doc)
	if err != nil {
		return in, fmt.Errorf("emitter: failed to put document at %s: %w", key, err)
	}

	// Store emitted path in steps environment
	val.Steps[e.prefix] = Text(path)

	return val, nil
}

func (e *Emitter) emitMultipleBinariesWithPaths(ctx context.Context, key iosystem.Key, binaries []Binary) ([]string, error) {
	paths := make([]string, 0, len(binaries))

	for i, bin := range binaries {
		// Generate numbered key: "image.0001.jpg", "image.0002.jpg"
		numberedKey := addNumberToKey(key, i+1)
		prefixedKey := iosystem.Key(filepath.Join(e.prefix, string(numberedKey)))

		// Use codec to re-encode (triggers JPEG conversion)
		dat, err := codec.Default.Encode(bin.Data, bin.Type)
		if err != nil {
			return nil, fmt.Errorf("emitter: failed to encode binary %d: %w", i+1, err)
		}

		doc := iosystem.NewDocument(prefixedKey, codec.ContentJPG, dat)
		doc.EnsureExtension()

		path, err := e.sink.Write(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("emitter: failed to put binary document at %s: %w", prefixedKey, err)
		}

		paths = append(paths, path)
	}

	return paths, nil
}

// addNumberToKey creates a numbered filename like "image.0001.jpg"
func addNumberToKey(key iosystem.Key, num int) iosystem.Key {
	keyStr := string(key)
	ext := filepath.Ext(keyStr)
	base := strings.TrimSuffix(keyStr, ext)
	return iosystem.Key(fmt.Sprintf("%s.%04d.jpg", base, num))
}
