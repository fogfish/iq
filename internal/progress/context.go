//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package progress

import "context"

// Context keys - using plain strings to allow cross-package access
const (
	reporterKey = "iq.progress.reporter"
	stepInfoKey = "iq.progress.stepinfo"
)

// StepInfo carries information about the current step execution
type StepInfo struct {
	JobName    string
	StepName   string
	StepNum    int
	TotalSteps int
}

// WithReporter embeds a progress reporter in the context
func WithReporter(ctx context.Context, reporter *Reporter) context.Context {
	return context.WithValue(ctx, reporterKey, reporter)
}

// FromContext extracts the progress reporter from context
func FromContext(ctx context.Context) *Reporter {
	if reporter, ok := ctx.Value(reporterKey).(*Reporter); ok {
		return reporter
	}
	return nil
}

// WithStepInfo embeds step information in the context
func WithStepInfo(ctx context.Context, info StepInfo) context.Context {
	return context.WithValue(ctx, stepInfoKey, info)
}

// GetStepInfo extracts step information from context
func GetStepInfo(ctx context.Context) *StepInfo {
	if info, ok := ctx.Value(stepInfoKey).(StepInfo); ok {
		return &info
	}
	return nil
}
