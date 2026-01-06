package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
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
	prefix   string
	snapshot storage.Storage

	Prompter
}

var _ Prompter = (*Emitter)(nil)

func NewEmitter(snapshot storage.Storage, prefix string, p Prompter) *Emitter {
	return &Emitter{snapshot: snapshot, prefix: prefix, Prompter: p}
}

func (e *Emitter) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	val, err := e.Prompter.Prompt(ctx, in, opts...)
	if err != nil {
		return in, err
	}

	key := iosystem.Key(filepath.Join(e.prefix, string(in.Key)))
	var buf bytes.Buffer
	switch v := val.Current.(type) {
	case Text:
		_, err = buf.Write([]byte(v))
		if err != nil {
			return in, fmt.Errorf("emitter: failed to write text content: %w", err)
		}
	case Json, List:
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		err = enc.Encode(v)
		if err != nil {
			return in, fmt.Errorf("emitter: failed to encode JSON content: %w", err)
		}
	default:
		return in, fmt.Errorf("emitter: unsupported content type: %T", v)
	}

	err = e.snapshot.Put(ctx, key, &buf)
	if err != nil {
		return in, fmt.Errorf("emitter: failed to put document at %s: %w", key, err)
	}

	fmt.Printf("[Emitter] Emitting with prefix '%s' : %v \n", e.prefix, val.Current)

	return val, nil
}
