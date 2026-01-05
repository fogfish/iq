package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/fogfish/iq/internal/progress"
	"github.com/kshard/chatter"
)

type Repeater struct {
	attempts int
	delay    int

	Prompter
}

var _ Prompter = (*Repeater)(nil)

func NewRepeater(attempts int, delay int, p Prompter) *Repeater {
	return &Repeater{attempts: attempts, delay: delay, Prompter: p}
}

func (r *Repeater) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	stepInfo := progress.GetStepInfo(ctx)
	for i := range r.attempts {
		stepInfo.Attempt = i + 1
		stepInfo.Delay = r.delay
		ctx = progress.WithStepInfo(ctx, *stepInfo)

		result, err := r.Prompter.Prompt(ctx, evt, opts...)
		if err == nil {
			return result, nil
		}

		if i < r.attempts-1 {
			time.Sleep(time.Duration(r.delay) * time.Second)
		}
	}

	return evt, fmt.Errorf("all attempts failed")
}
