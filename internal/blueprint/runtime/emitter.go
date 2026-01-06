package runtime

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/codec"
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
	prefix string
	sink   iosystem.Sink

	Prompter
}

var _ Prompter = (*Emitter)(nil)

func NewEmitter(sink iosystem.Sink, prefix string, p Prompter) *Emitter {
	return &Emitter{
		sink:     sink,
		prefix:   prefix,
		Prompter: p,
	}
}

func (e *Emitter) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	val, err := e.Prompter.Prompt(ctx, in, opts...)
	if err != nil {
		return in, err
	}

	key := iosystem.Key(filepath.Join(e.prefix, string(val.Key)))
	dat, err := codec.Default.Encode(val.Current, val.Current.ContentType())
	if err != nil {
		return in, fmt.Errorf("emitter: failed to encode content: %w", err)
	}
	doc := iosystem.NewDocument(key, val.Current.ContentType(), dat)
	doc.EnsureExtension()

	err = e.sink.Write(ctx, doc)
	if err != nil {
		return in, fmt.Errorf("emitter: failed to put document at %s: %w", key, err)
	}

	return val, nil
}
