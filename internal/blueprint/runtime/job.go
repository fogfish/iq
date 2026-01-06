package runtime

import (
	"context"
	"fmt"

	"github.com/fogfish/iq/internal/progress"
	"github.com/kshard/chatter"
)

type Job struct {
	Name  string
	Steps []Prompter
}

var _ Prompter = (*Job)(nil)

func NewJob(name string, steps []Prompter) *Job {
	return &Job{Name: name, Steps: steps}
}

func (j *Job) Config(jobs map[string]*Job) error {
	for _, step := range j.Steps {
		if err := step.Config(jobs); err != nil {
			return fmt.Errorf("job '%s' configuration failed: %w", j.Name, err)
		}
	}
	return nil
}

func (j *Job) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	evt := in
	var err error

	jobSize := len(j.Steps)
	for i, step := range j.Steps {
		evt, err = step.Prompt(
			progress.ContextCallStack(ctx, j.Name, jobSize, i+1),
			evt,
			opts...,
		)
		if err != nil {
			return evt, fmt.Errorf("step %d failed: %w", i, err)
		}
	}

	return evt, nil
}
