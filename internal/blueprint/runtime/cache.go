package runtime

import (
	"context"

	"github.com/kshard/chatter"
)

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
