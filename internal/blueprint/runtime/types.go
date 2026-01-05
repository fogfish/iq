package runtime

import (
	"context"

	"github.com/kshard/chatter"
)

// Input/Output content
type Gist interface{ HKT1(Gist) }

// Input/Output context is plain text
type Text string

func (Text) HKT1(Gist) {}

// Input/Output context is JSON
type Json map[string]any

func (Json) HKT1(Gist) {}

// Event represents an event processed by the workflow.
type Event struct {
	// Original workflow input
	Document Gist

	// Current input being processed
	Current Gist

	// Named step outputs
	Steps map[string]Gist
}

func (evt Event) copy(current Gist) Event {
	return Event{
		Document: evt.Document,
		Current:  current,
		Steps:    evt.Steps,
	}
}

type Prompter interface {
	Prompt(context.Context, Event, ...chatter.Opt) (Event, error)
}
