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

	"github.com/fogfish/iq/internal/prompt"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/kshard/thinker/reasoner"
	"github.com/spf13/viper"
)

type Prompter struct {
	*agent.Automata[*viper.Viper, []byte]
	config *viper.Viper
}

func NewPrompter(llm chatter.Chatter, c *viper.Viper, maxEpoch int) (*Prompter, error) {
	w := &Prompter{config: c}
	w.Automata = agent.NewAutomata(llm,
		memory.NewStream(memory.INFINITE, chatter.Stratum(c.GetString("role"))),
		codec.FromEncoder(fromViper),
		codec.FromDecoder(w.decode),
		reasoner.NewEpoch(maxEpoch, reasoner.From(w.deduct)),
	)

	return w, nil
}

func (w *Prompter) decode(reply chatter.Reply) (float64, []byte, error) {
	if w.config.GetString(prompt.YAML_FORMAT) == "json" {
		var seq []string
		if err := jsonify.Strings.Decode(reply, &seq); err != nil {
			return 0.0, nil, err
		}

		b, err := json.Marshal(seq)
		if err != nil {
			return 0.0, nil, err
		}

		return 1.0, b, nil
	}

	return 1.0, []byte(reply.Text), nil
}

func (w *Prompter) deduct(state thinker.State[[]byte]) (thinker.Phase, chatter.Prompt, error) {
	// Provide feedback to LLM if there are no confidence about the results
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous request using the feedback below.")
		prompt.With(state.Feedback)
		return thinker.AGENT_REFINE, prompt, nil
	}

	// We have sufficient confidence, return results
	return thinker.AGENT_RETURN, chatter.Prompt{}, nil
}
