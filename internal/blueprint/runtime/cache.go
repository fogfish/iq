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
	"log/slog"
	"path/filepath"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/kshard/chatter"
)

/*

TODO:

🔴 1. Cache Decorator (Empty Stub)
Current State:

What's Missing:

No cache key generation from emit context
No cache lookup before execution
No cache storage after execution
Needs to access CacheContext from context.Context
Old Implementation Reference:
The old code in AgentStep, RouterStep, ForeachStep, RunStep all had:


*/

type Cache struct {
	prefix string
	cache  storage.Storage

	Prompter
}

var _ Prompter = (*Cache)(nil)

func NewCache(cache storage.Storage, prefix string, p Prompter) *Cache {
	return &Cache{cache: cache, prefix: prefix, Prompter: p}
}

func (e *Cache) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	type cachable struct {
		Key  iosystem.Key `json:"key"`
		Gist any          `json:"gist"`
	}

	key := iosystem.Key(filepath.Join(e.prefix, string(in.Key)))
	has, err := e.cache.Has(ctx, key)
	if err != nil {
		return in, err
	}

	if has {
		r, err := e.cache.Get(ctx, key)
		if err != nil {
			return in, err
		}

		var val cachable
		if err := json.NewDecoder(r).Decode(&val); err != nil {
			return in, err
		}

		gist, err := ToGist(val.Gist)
		if err != nil {
			return in, err
		}

		in.Current = gist
		in.Key = val.Key
		return in, nil
	}

	val, err := e.Prompter.Prompt(ctx, in, opts...)
	if err != nil {
		return in, err
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(
		cachable{
			Key:  val.Key,
			Gist: val.Current,
		},
	)
	if err != nil {
		slog.Error("cache failed", "err", err)
		return val, nil
	}

	err = e.cache.Put(ctx, key, &buf)
	if err != nil {
		slog.Error("cache failed", "err", err)
		return val, nil
	}

	return val, nil
}
