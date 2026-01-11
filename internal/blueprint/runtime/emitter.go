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
		if err := e.emitSingleBinary(ctx, val.Key, bin); err != nil {
			return in, err
		}
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
			if err := e.emitMultipleBinaries(ctx, val.Key, binaries); err != nil {
				return in, err
			}
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

	err = e.sink.Write(ctx, doc)
	if err != nil {
		return in, fmt.Errorf("emitter: failed to put document at %s: %w", key, err)
	}

	return val, nil
}

func (e *Emitter) emitSingleBinary(ctx context.Context, key iosystem.Key, bin Binary) error {
	prefixedKey := iosystem.Key(filepath.Join(e.prefix, string(key)))

	// Use codec to re-encode (triggers JPEG conversion)
	dat, err := codec.Default.Encode(bin.Data, bin.Type)
	if err != nil {
		return fmt.Errorf("emitter: failed to encode binary: %w", err)
	}

	// Always output as JPEG after re-encoding
	doc := iosystem.NewDocument(prefixedKey, codec.ContentJPG, dat)
	doc.EnsureExtension()

	if err := e.sink.Write(ctx, doc); err != nil {
		return fmt.Errorf("emitter: failed to put binary document at %s: %w", prefixedKey, err)
	}

	return nil
}

func (e *Emitter) emitMultipleBinaries(ctx context.Context, key iosystem.Key, binaries []Binary) error {
	for i, bin := range binaries {
		// Generate numbered key: "image.0001.jpg", "image.0002.jpg"
		numberedKey := addNumberToKey(key, i+1)
		prefixedKey := iosystem.Key(filepath.Join(e.prefix, string(numberedKey)))

		// Use codec to re-encode (triggers JPEG conversion)
		dat, err := codec.Default.Encode(bin.Data, bin.Type)
		if err != nil {
			return fmt.Errorf("emitter: failed to encode binary %d: %w", i+1, err)
		}

		doc := iosystem.NewDocument(prefixedKey, codec.ContentJPG, dat)
		doc.EnsureExtension()

		if err := e.sink.Write(ctx, doc); err != nil {
			return fmt.Errorf("emitter: failed to put binary document at %s: %w", prefixedKey, err)
		}
	}

	return nil
}

// addNumberToKey creates a numbered filename like "image.0001.jpg"
func addNumberToKey(key iosystem.Key, num int) iosystem.Key {
	keyStr := string(key)
	ext := filepath.Ext(keyStr)
	base := strings.TrimSuffix(keyStr, ext)
	return iosystem.Key(fmt.Sprintf("%s.%04d.jpg", base, num))
}
