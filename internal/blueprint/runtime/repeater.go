package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/kshard/chatter"
)

type Repeater struct {
	Node     string
	attempts int
	delay    int
	fallback Prompter

	Prompter
}

var _ Prompter = (*Repeater)(nil)

func NewRepeater(attempts int, delay int, fallback string, p Prompter) *Repeater {
	return &Repeater{attempts: attempts, delay: delay, Node: fallback, Prompter: p}
}

func (r *Repeater) Config(jobs map[string]*Job) error {
	if r.Node != "" {
		r.Prompter = jobs[r.Node]
	}
	return nil
}

func (r *Repeater) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	// stepInfo := progress.GetStepInfo(ctx)
	for i := range r.attempts {
		// stepInfo.Attempt = i + 1
		// stepInfo.Delay = r.delay
		// ctx = progress.WithStepInfo(ctx, *stepInfo)

		result, err := r.Prompter.Prompt(ctx, evt, opts...)
		if err == nil {
			return result, nil
		}

		if i < r.attempts-1 {
			time.Sleep(time.Duration(r.delay) * time.Second)
		}
	}

	if r.fallback != nil {
		result, err := r.fallback.Prompt(ctx, evt, opts...)
		if err != nil {
			return evt, err
		}

		return result, nil
	}

	return evt, fmt.Errorf("all attempts failed")
}
