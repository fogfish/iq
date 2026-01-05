package runtime

import (
	"context"

	"github.com/kshard/chatter"
)

// Memento stores the output of the Prompter into a variable.
type Memento struct {
	variable string

	Prompter
}

var _ Prompter = (*Memento)(nil)

func NewMemento(variable string, p Prompter) *Memento {
	return &Memento{variable: variable, Prompter: p}
}

func (m *Memento) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	evt, err := m.Prompter.Prompt(ctx, evt, opts...)
	if err != nil {
		return evt, err
	}
	evt.Steps[m.variable] = evt.Current

	return evt, nil
}
