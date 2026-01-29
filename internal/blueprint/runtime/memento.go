//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

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

	if m.variable != "" {
		switch v := evt.Current.(type) {
		case Text, Json, List:
			evt.Steps[m.variable] = v
		}
	}

	return evt, nil
}
