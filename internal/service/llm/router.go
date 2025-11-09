//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package llm

import (
	"context"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/provider/autoconfig"
)

// Chatter interface option allowing to dynamically route prompts to choosen models
type Route string

func (Route) ChatterOpt() {}

// Dynamic routing strategy throught pool of LLMs.
// The LLMs pool consists of default "route" and multiple named models.
type Router struct {
	profiles map[string]chatter.Chatter
	usage    chatter.Usage
}

var _ chatter.Chatter = (*Router)(nil)

// Creates LLMs pools instance
func NewRouter(profile, model string) (*Router, error) {
	router := &Router{profiles: make(map[string]chatter.Chatter)}

	if profile == "mock" || model == "mock" {
		router.profiles["base"] = &Mock{}
		return router, nil
	}

	llm, err := autoconfig.FromFile(".iqrc", profile)
	if err != nil {
		return nil, err
	}
	router.profiles["base"] = llm

	return router, nil
}

func (router *Router) Usage() chatter.Usage { return router.usage }

func (router *Router) Prompt(ctx context.Context, prompt []chatter.Message, opts ...chatter.Opt) (reply *chatter.Reply, err error) {
	llm := router.profiles["base"]

	for _, opt := range opts {
		switch v := opt.(type) {
		case Route:
			if l, has := router.profiles[string(v)]; has {
				llm = l
			} else {
				llm, err = autoconfig.FromFile(".iqrc", string(v))
				if err != nil {
					return nil, err
				}
				router.profiles[string(v)] = llm
			}
		}
	}

	reply, err = llm.Prompt(ctx, prompt, opts...)
	if err != nil {
		return
	}

	router.usage.InputTokens += reply.Usage.InputTokens
	router.usage.ReplyTokens += reply.Usage.ReplyTokens

	return reply, nil
}
