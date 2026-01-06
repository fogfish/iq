//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package progress

import (
	"context"
)

// Context keys - using plain strings to allow cross-package access
const (
	reporterKey    = "iq.progress.reporter"
	stepInfoKey    = "iq.progress.stepinfo"
	foreachModeKey = "iq.progress.foreachmode"
)

var silent = &Silent{}

// StepInfo carries information about the current step execution
type StepInfo struct {
	// Nesting support for router and foreach
	Parent *StepInfo // Link to parent step (for nested jobs)

	JobName string
	JobSize int

	StepID   int
	StepName string

	// Retry-specific fields
	Attempt int
	Delay   int

	// ForEach-specific fields
	ItemIndex int // Current item (1-based), 0 = not in foreach
	ItemTotal int // Total items, 0 = not in foreach
}

func ContextCallStack(ctx context.Context, jobName string, jobSize, stepID int) context.Context {
	info := GetStepInfo(ctx)

	// top-level invokation, no context info
	if info == nil {
		return WithStepInfo(ctx, StepInfo{
			JobName: jobName,
			StepID:  stepID,
			JobSize: jobSize,
		})
	}

	info.JobName = jobName
	info.JobSize = jobSize
	info.StepID = stepID

	return WithStepInfo(ctx, *info)
}

func ContextForEach(ctx context.Context, jobName string, index, total int) context.Context {
	info := GetStepInfo(ctx)
	if info == nil {
		return ctx
	}

	return WithStepInfo(ctx, StepInfo{
		JobName:   jobName,
		Parent:    info,
		ItemIndex: index,
		ItemTotal: total,
	})
}

func ContextRouter(ctx context.Context, jobName string) context.Context {
	info := GetStepInfo(ctx)
	if info == nil {
		return ctx
	}

	return WithStepInfo(ctx, StepInfo{
		JobName: jobName,
		Parent:  info,
	})
}

// WithReporter embeds a progress reporter in the context
func WithReporter(ctx context.Context, reporter *Reporter) context.Context {
	//lint:ignore SA1029 due to cross-package context key access
	return context.WithValue(ctx, reporterKey, reporter)
}

// FromContext extracts the progress reporter from context
func FromContext(ctx context.Context) ProgressBar {
	if reporter, ok := ctx.Value(reporterKey).(*Reporter); ok {
		return reporter
	}
	return silent
}

// WithStepInfo embeds step information in the context
func WithStepInfo(ctx context.Context, info StepInfo) context.Context {
	//lint:ignore SA1029 due to cross-package context key access
	return context.WithValue(ctx, stepInfoKey, info)
}

// GetStepInfo extracts step information from context
func GetStepInfo(ctx context.Context) *StepInfo {
	if info, ok := ctx.Value(stepInfoKey).(StepInfo); ok {
		return &info
	}
	return nil
}

// WithForeachMode marks the context as being inside a foreach loop
func WithForeachMode(ctx context.Context) context.Context {
	//lint:ignore SA1029 due to cross-package context key access
	return context.WithValue(ctx, foreachModeKey, true)
}

// IsInForeachMode checks if we're currently inside a foreach loop
func IsInForeachMode(ctx context.Context) bool {
	if inForeach, ok := ctx.Value(foreachModeKey).(bool); ok {
		return inForeach
	}
	return false
}
