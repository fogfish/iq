package runtime

import (
	"context"

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
	Prompter
}

var _ Prompter = (*Emitter)(nil)

func NewEmitter(p Prompter) *Emitter {
	return &Emitter{Prompter: p}
}

func (e *Emitter) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	return e.Prompter.Prompt(ctx, evt, opts...)
}
