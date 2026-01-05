package runtime

import (
	"context"

	"github.com/kshard/chatter"
)

type Emitter struct {
	Prompter
}

var _ Prompter = (*Emitter)(nil)

func NewEmitter(p Prompter) *Emitter {
	return &Emitter{Prompter: p}
}

func (e *Emitter) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	return e.Prompter.Prompt(ctx, evt, opts...)
}
