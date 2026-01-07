//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package progress

import "time"

// Silent is a no-op implementation of ProgressBar that does nothing
type Silent struct{}

// NewSilent creates a new silent progress reporter
func NewSilent() *Silent {
	return &Silent{}
}

// SetTokenSource does nothing
func (s *Silent) SetTokenSource(source TokenSource) {}

// SetForeachMode does nothing
func (s *Silent) SetForeachMode(inForeach bool) {}

// WorkflowLoading does nothing
func (s *Silent) WorkflowLoading(path string) {}

// WorkflowCompiled does nothing
func (s *Silent) WorkflowCompiled(name string, jobCount, stepCount int) {}

// WorkflowError does nothing
func (s *Silent) WorkflowError(err error) {}

// DocumentStart does nothing
func (s *Silent) DocumentStart(path string, sizeKB float64) {}

// DocumentComplete does nothing
func (s *Silent) DocumentComplete(path string, duration time.Duration) {}

// DocumentError does nothing
func (s *Silent) DocumentError(path string, err error) {}

// DocumentSkipped does nothing
func (s *Silent) DocumentSkipped(path string, reason string) {}

// StepStart does nothing
func (s *Silent) StepStart(jobName, stepName string, stepNum, totalSteps int) {}

// StepProgress does nothing
func (s *Silent) StepProgress(message string) {}

// StepComplete does nothing
func (s *Silent) StepComplete(jobName, stepName string, stepNum, totalSteps int, duration time.Duration, tokens int) {
}

// StepSkipped does nothing
func (s *Silent) StepSkipped(jobName, stepName string, stepNum, totalSteps int, reason string) {}

// StepError does nothing
func (s *Silent) StepError(jobName, stepName string, stepNum, totalSteps int, err error) {}

// RouterEvaluating does nothing
func (s *Silent) RouterEvaluating() {}

// RouterMatched does nothing
func (s *Silent) RouterMatched(routeName string, targetJob string) {}

// RouterDefault does nothing
func (s *Silent) RouterDefault(targetJob string) {}

// RouterNoMatch does nothing
func (s *Silent) RouterNoMatch() {}

// ForeachStart does nothing
func (s *Silent) ForeachStart(itemCount int) {}

// ForeachItem does nothing
func (s *Silent) ForeachItem(index, total int, status string) {}

// ForeachComplete does nothing
func (s *Silent) ForeachComplete(successCount, totalCount int, duration time.Duration) {}

// RetryAttempt does nothing
func (s *Silent) RetryAttempt(attempt, maxAttempts int, delay time.Duration) {}

// RetrySuccess does nothing
func (s *Silent) RetrySuccess(attempt int) {}

// RetryExhausted does nothing
func (s *Silent) RetryExhausted(maxAttempts int) {}

// ThinkingContent does nothing
func (s *Silent) ThinkingContent(text string) {}

// BatchStart does nothing
func (s *Silent) BatchStart(inputPath, outputPath string, mutable bool) {}

// BatchProgress does nothing
func (s *Silent) BatchProgress(processed, total int) {}

// BatchComplete does nothing
func (s *Silent) BatchComplete() {}

// ChunkingStart does nothing
func (s *Silent) ChunkingStart(strategy string, chunkCount int) {}

// Summary does nothing
func (s *Silent) Summary() {}

// SummaryQuick does nothing
func (s *Silent) SummaryQuick(docCount int, duration time.Duration) {}

// UpdateTokens does nothing
func (s *Silent) UpdateTokens(inputTokens, outputTokens int) {}

// GetStats returns empty statistics
func (s *Silent) GetStats() Stats {
	return Stats{}
}
