package runtime

import (
	"context"
	"time"

	"github.com/fogfish/iq/internal/progress"
	"github.com/kshard/chatter"
)

type Printer struct {
	Prompter
}

var _ Prompter = (*Printer)(nil)

func NewPrinter(p Prompter) *Printer {
	return &Printer{Prompter: p}
}

func (p *Printer) Prompt(ctx context.Context, evt Event, opts ...chatter.Opt) (Event, error) {
	reporter := progress.FromContext(ctx)
	stepInfo := progress.GetStepInfo(ctx)

	if reporter == nil || stepInfo == nil {
		return p.Prompter.Prompt(ctx, evt, opts...)
	}

	startTime := time.Now()
	reporter.StepStart(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps)

	if stepInfo.Attempt > 1 {
		reporter.RetryAttempt(stepInfo.Attempt, stepInfo.TotalSteps, time.Duration(stepInfo.Delay)*time.Second)
	}

	result, err := p.Prompter.Prompt(ctx, evt, opts...)
	if err != nil {
		reporter.StepError(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, err)
		return result, err
	}

	if stepInfo.Attempt > 1 {
		reporter.RetrySuccess(stepInfo.Attempt)
	}

	duration := time.Since(startTime)
	// TODO: Track actual token usage from LLM response
	reporter.StepComplete(stepInfo.JobName, stepInfo.StepName, stepInfo.StepNum, stepInfo.TotalSteps, duration, 0)

	// TODO: reporter.RetryExhausted(step.Retry.Attempts)

	return result, nil
}
