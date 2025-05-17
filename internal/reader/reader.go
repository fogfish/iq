//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package reader

import (
	"context"
	"io"
	"strings"

	"github.com/fogfish/iq/internal/core"
	"github.com/fogfish/scanner"
	"github.com/kshard/chatter"
)

const (
	STRATEGY_NONE      = "none"
	STRATEGY_SENTENCE  = "sentence"
	STRATEGY_PARAGRAPH = "paragraph"
	STRATEGY_CHUNK     = "chunk"
)

func New(kind string, chars string, size int, r io.Reader) scanner.Scanner {
	switch kind {
	case STRATEGY_SENTENCE:
		if len(chars) == 0 {
			chars = scanner.EndOfSentence
		}
		return scanner.NewSentencer(chars, r)

	case STRATEGY_PARAGRAPH:
		if len(chars) == 0 {
			chars = "\n\n"
		}
		return scanner.NewSlicer(chars, r)

	case STRATEGY_CHUNK:
		if len(chars) == 0 {
			chars = scanner.EndOfSentence
		}
		if size == 0 {
			size = 1024
		}
		return scanner.NewChunker(size, scanner.NewSentencer(chars, r))

	default:
		return scanner.NewIdentity(r)
	}
}

type Agent interface {
	PromptOnce(context.Context, *core.Prompt, ...chatter.Opt) ([]byte, error)
}

func Process(ctx context.Context, agt Agent, req *core.Prompt, r scanner.Scanner, w io.Writer) error {
	for r.Scan() {
		txt := strings.TrimSpace(r.Text())
		if len(txt) == 0 {
			continue
		}

		req.Blob = txt
		reply, err := agt.PromptOnce(context.Background(), req)
		if err != nil {
			return err
		}
		if _, err := w.Write(reply); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := r.Err(); err != nil {
		return err
	}

	return nil
}
