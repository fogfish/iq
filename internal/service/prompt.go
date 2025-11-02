//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package service

import (
	"encoding/json"

	"github.com/fogfish/iq/internal/core"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/kshard/thinker/reasoner"
)

type Prompter struct {
	*agent.Automata[*core.Prompt, []byte]
	isJsonify bool
}

func NewPrompter(llm chatter.Chatter) (*Prompter, error) {
	w := &Prompter{}
	w.Automata = agent.NewAutomata(llm,
		memory.NewStream(memory.INFINITE, chatter.Stratum("")),
		codec.FromEncoder(w.encode),
		codec.FromDecoder(w.decode),
		reasoner.From(w.deduct),
	)

	return w, nil
}

func (w *Prompter) encode(in *core.Prompt) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(in.Task)

	if len(in.Blob) > 0 {
		prompt.WithBlob("Input document", in.Blob)
	}

	w.isJsonify = false
	if in.Format == core.FORMAT_JSON {
		jsonify.Strings.Harden(&prompt, nil)
		w.isJsonify = true
	}

	return &prompt, nil
}

func (w *Prompter) decode(reply *chatter.Reply) (float64, []byte, error) {
	if w.isJsonify {
		var seq []string
		if err := jsonify.Strings.Decode(reply, nil, &seq); err != nil {
			return 0.0, nil, err
		}

		b, err := json.Marshal(seq)
		if err != nil {
			return 0.0, nil, err
		}

		return 1.0, b, nil
	}

	return 1.0, []byte(reply.String()), nil
}

func (w *Prompter) deduct(state thinker.State[[]byte]) (thinker.Phase, chatter.Message, error) {
	// Provide feedback to LLM if there are no confidence about the results
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous request using the feedback below.")
		prompt.With(state.Feedback)
		return thinker.AGENT_REFINE, &prompt, nil
	}

	// We have sufficient confidence, return results
	return thinker.AGENT_RETURN, nil, nil
}
