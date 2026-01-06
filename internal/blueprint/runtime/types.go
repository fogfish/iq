package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/kshard/chatter"
)

// Input/Output content
type Gist interface{ HKT1(Gist) }

// Input/Output content is plain text
type Text string

func (Text) HKT1(Gist) {}

// Input/Output content is JSON
type Json map[string]any

func (Json) HKT1(Gist) {}

// Input/Output content is a JSON array
type List []any

func (List) HKT1(Gist) {}

func ToGist(x any) (Gist, error) {
	switch v := x.(type) {
	case string:
		return Text(v), nil
	case []byte:
		return Text(v), nil
	case map[string]any:
		return Json(v), nil
	case []any:
		return List(v), nil
	case Gist:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported Gist type: %T", v)
	}
}

// Event represents an event processed by the workflow.
type Event struct {
	// Unique identity of the document
	Key iosystem.Key

	// Original workflow input
	Document Gist

	// Current input being processed
	Current Gist

	// Named step outputs
	Steps map[string]Gist
}

func (evt Event) copy(current Gist) Event {
	return Event{
		Key:      evt.Key,
		Document: evt.Document,
		Current:  current,
		Steps:    evt.Steps,
	}
}

type Prompter interface {
	Config(map[string]*Job) error
	Prompt(context.Context, Event, ...chatter.Opt) (Event, error)
}
