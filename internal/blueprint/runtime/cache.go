package runtime

import (
	"context"

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
	Prompter
}

var _ Prompter = (*Cache)(nil)

func NewCache(p Prompter) *Cache {
	return &Cache{Prompter: p}
}

func (e *Cache) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	return e.Prompter.Prompt(ctx, evt, opts...)
}
