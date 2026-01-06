//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nickwells/twrap.mod/twrap"
	"golang.org/x/term"
)

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens int
	ReplyTokens int
}

// ProfileTokenUsage represents token usage for a specific profile
type ProfileTokenUsage struct {
	Name  string
	Usage TokenUsage
}

// TokenSource interface for accessing token usage from LLM routers
type TokenSource interface {
	Usage() TokenUsage
	ProfileUsage() []ProfileTokenUsage
}

// Icons used for progress reporting - centralized for easy customization
const (
	IconWorkflowLoad     = "📋" // Loading workflow file
	IconWorkflowCompiled = "✅" // Workflow successfully compiled
	IconWorkflowError    = "❌" // Workflow error

	IconDocumentStart    = "📄"  // Document processing started
	IconDocumentComplete = "✅"  // Document processed successfully
	IconDocumentError    = "❌"  // Document processing error
	IconDocumentSkipped  = "⏭️" // Document skipped

	IconStepProcessing = "⚙️ " // Step is processing (mutable line)
	IconStepComplete   = "✅"   // Step completed successfully
	IconStepError      = "❌"   // Step failed with error

	IconRouterEvaluating = "🧭" // Router evaluating conditions
	IconRouterMatched    = "↳" // Router matched a route (text, not emoji)

	IconForeachStart = "🔁" // Foreach loop started
	IconForeachItem  = "✅" // Foreach item completed

	IconRetryAttempt   = "🔄" // Retry attempt
	IconRetrySuccess   = "✅" // Retry succeeded
	IconRetryExhausted = "❌" // All retries exhausted

	IconBatchProcessing = "📂"  // Batch processing
	IconChunking        = "✂️" // Document chunking

	IconThinking = "💭" // LLM thinking/reasoning content

	IconSummary = "📊"  // Pipeline summary
	IconWarning = "⚠️" // Warning message
)

// ProgressBar interface defines all progress reporting methods
type ProgressBar interface {
	// Token source management
	SetTokenSource(source TokenSource)
	SetForeachMode(inForeach bool)

	// Workflow lifecycle events
	WorkflowLoading(path string)
	WorkflowCompiled(name string, jobCount, stepCount int)
	WorkflowError(err error)

	// Document processing events
	DocumentStart(path string, sizeKB float64)
	DocumentComplete(path string, duration time.Duration)
	DocumentError(path string, err error)
	DocumentSkipped(path string, reason string)

	// Step execution events
	StepStart(jobName, stepName string, stepNum, totalSteps int)
	StepProgress(message string)
	StepComplete(jobName, stepName string, stepNum, totalSteps int, duration time.Duration, tokens int)
	StepSkipped(jobName, stepName string, stepNum, totalSteps int, reason string)
	StepError(jobName, stepName string, stepNum, totalSteps int, err error)

	// Router and control flow events
	RouterEvaluating()
	RouterMatched(routeName string, targetJob string)
	RouterDefault(targetJob string)
	RouterNoMatch()

	// Foreach iteration events
	ForeachStart(itemCount int)
	ForeachItem(index, total int, status string)
	ForeachComplete(successCount, totalCount int, duration time.Duration)

	// Retry and recovery events
	RetryAttempt(attempt, maxAttempts int, delay time.Duration)
	RetrySuccess(attempt int)
	RetryExhausted(maxAttempts int)

	// LLM thinking events
	ThinkingContent(text string)

	// Batch processing events
	BatchStart(inputPath, outputPath string, mutable bool)
	BatchProgress(processed, total int)
	BatchComplete()

	// Chunking events
	ChunkingStart(strategy string, chunkCount int)

	// Summary and statistics
	Summary()
	SummaryQuick(docCount int, duration time.Duration)
	UpdateTokens(inputTokens, outputTokens int)
	GetStats() Stats
}

// Reporter handles all progress output to stderr with educational messages
type Reporter struct {
	w               io.Writer
	mu              sync.Mutex
	quiet           bool
	lastLine        string
	hasThinking     bool // Set to true when thinking content is shown
	isInForeachMode bool // Set to true when inside a foreach loop
	startTime       time.Time
	stats           Stats
	tokenSource     TokenSource // Source for final token usage reporting
}

// Stats tracks processing metrics
type Stats struct {
	DocsProcessed int
	DocsSkipped   int
	DocsErrors    int
	TokensInput   int
	TokensOutput  int
	Errors        []error
}

// New creates a new progress reporter
func New(quiet bool) *Reporter {
	return &Reporter{
		w:         os.Stderr,
		quiet:     quiet,
		startTime: time.Now(),
	}
}

// NewWithWriter creates a reporter with a custom writer (for testing)
func NewWithWriter(w io.Writer, quiet bool) *Reporter {
	return &Reporter{
		w:         w,
		quiet:     quiet,
		startTime: time.Now(),
	}
}

// SetTokenSource sets the token source for final usage reporting
func (r *Reporter) SetTokenSource(source TokenSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokenSource = source
}

// SetForeachMode sets whether we're inside a foreach loop
func (r *Reporter) SetForeachMode(inForeach bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isInForeachMode = inForeach
}

// Workflow lifecycle events

// WorkflowLoading indicates workflow file is being loaded
func (r *Reporter) WorkflowLoading(path string) {
	r.println(fmt.Sprintf("%s Loading workflow from %s", IconWorkflowLoad, path))
}

// WorkflowCompiled indicates successful workflow compilation
func (r *Reporter) WorkflowCompiled(name string, jobCount, stepCount int) {
	if name == "" {
		name = "(unnamed)"
	}
	r.println(fmt.Sprintf("%s Workflow compiled: \"%s\" (%d jobs, %d steps)", IconWorkflowCompiled, name, jobCount, stepCount))
	r.println("")
}

// WorkflowError reports a workflow-level error
func (r *Reporter) WorkflowError(err error) {
	r.println(fmt.Sprintf("%s Workflow error: %v", IconWorkflowError, err))
}

// Document processing events

// DocumentStart indicates a document is starting to process
func (r *Reporter) DocumentStart(path string, sizeKB float64) {
	if sizeKB > 0 {
		r.println(fmt.Sprintf("%s Processing: %s (%.1f KB)", IconDocumentStart, path, sizeKB))
	} else {
		r.println(fmt.Sprintf("%s Processing: %s", IconDocumentStart, path))
	}
}

// DocumentComplete indicates successful document completion
func (r *Reporter) DocumentComplete(path string, duration time.Duration) {
	r.stats.DocsProcessed++
	r.println(fmt.Sprintf("%s Completed: %s (%.1fs total)", IconDocumentComplete, path, duration.Seconds()))
	r.println("")
}

// DocumentError reports a document processing error
func (r *Reporter) DocumentError(path string, err error) {
	r.stats.DocsErrors++
	r.stats.Errors = append(r.stats.Errors, err)
	r.println(fmt.Sprintf("%s Failed: %s - %v", IconDocumentError, path, err))
}

// DocumentSkipped indicates a document was skipped
func (r *Reporter) DocumentSkipped(path string, reason string) {
	r.stats.DocsSkipped++
	r.println(fmt.Sprintf("%s Skipped: %s (%s)", IconDocumentSkipped, path, reason))
}

// Step execution events

// StepStart indicates a workflow step is starting
func (r *Reporter) StepStart(jobName, stepName string, stepNum, totalSteps int) {
	r.mu.Lock()
	r.hasThinking = false // Reset thinking flag for new step
	r.mu.Unlock()

	if stepName != "" {
		r.updateLine(fmt.Sprintf("   %s Step %d/%d: %s.%s → Processing...",
			IconStepProcessing, stepNum, totalSteps, jobName, stepName))
	} else {
		r.updateLine(fmt.Sprintf("   %s Step %d/%d: %s → Processing...",
			IconStepProcessing, stepNum, totalSteps, jobName))
	}
}

// StepProgress updates the current step status (mutable)
func (r *Reporter) StepProgress(message string) {
	r.updateLine(fmt.Sprintf("   %s %s", IconStepProcessing, message))
}

// StepComplete indicates step completion
func (r *Reporter) StepComplete(jobName, stepName string, stepNum, totalSteps int, duration time.Duration, tokens int) {
	r.stats.TokensInput += tokens

	// Don't show step completion messages when inside a foreach loop
	// The foreach will handle its own reporting
	if r.isInForeachMode {
		return
	}

	if tokens > 0 {
		if stepName != "" {
			r.println(fmt.Sprintf("   %s Step %d/%d: %s.%s completed in %.1fs (%d tokens)",
				IconStepComplete, stepNum, totalSteps, jobName, stepName, duration.Seconds(), tokens))
		} else {
			r.println(fmt.Sprintf("   %s Step %d/%d: %s completed in %.1fs (%d tokens)",
				IconStepComplete, stepNum, totalSteps, jobName, duration.Seconds(), tokens))
		}
	} else {
		if stepName != "" {
			r.println(fmt.Sprintf("   %s Step %d/%d: %s.%s completed in %.1fs",
				IconStepComplete, stepNum, totalSteps, jobName, stepName, duration.Seconds()))
		} else {
			r.println(fmt.Sprintf("   %s Step %d/%d: %s completed in %.1fs",
				IconStepComplete, stepNum, totalSteps, jobName, duration.Seconds()))
		}
	}
}

// StepSkipped indicates step was skipped (e.g., cache hit)
func (r *Reporter) StepSkipped(jobName, stepName string, stepNum, totalSteps int, reason string) {
	// Don't show step skip messages when inside a foreach loop
	if r.isInForeachMode {
		return
	}

	if stepName != "" {
		r.println(fmt.Sprintf("   ⏭️  Step %d/%d: %s.%s skipped (%s)",
			stepNum, totalSteps, jobName, stepName, reason))
	} else {
		r.println(fmt.Sprintf("   ⏭️  Step %d/%d: %s skipped (%s)",
			stepNum, totalSteps, jobName, reason))
	}
}

// StepError reports a step error
func (r *Reporter) StepError(jobName, stepName string, stepNum, totalSteps int, err error) {
	if stepName != "" {
		r.println(fmt.Sprintf("   %s Step %d/%d: %s.%s failed - %v",
			IconStepError, stepNum, totalSteps, jobName, stepName, err))
	} else {
		r.println(fmt.Sprintf("   %s Step %d/%d: %s failed - %v",
			IconStepError, stepNum, totalSteps, jobName, err))
	}
}

// Router and control flow events

// RouterEvaluating indicates router is evaluating conditions
func (r *Reporter) RouterEvaluating() {
	r.updateLine(fmt.Sprintf("   %s Router evaluating conditions...", IconRouterEvaluating))
}

// RouterMatched indicates router matched a route
func (r *Reporter) RouterMatched(routeName string, targetJob string) {
	r.println(fmt.Sprintf("   %s Routing decision:", IconRouterEvaluating))
	r.println(fmt.Sprintf("      %s Matched route: %s → %s", IconRouterMatched, routeName, targetJob))
	r.println("")
}

// RouterDefault indicates router is using default route
func (r *Reporter) RouterDefault(targetJob string) {
	r.println(fmt.Sprintf("   %s Routing decision:", IconRouterEvaluating))
	r.println(fmt.Sprintf("      %s Using default route → %s", IconRouterMatched, targetJob))
	r.println("")
}

// RouterNoMatch indicates no route was matched
func (r *Reporter) RouterNoMatch() {
	r.println(fmt.Sprintf("   %s Router: No matching route found", IconWarning))
}

// Foreach iteration events

// ForeachStart indicates foreach loop is starting
func (r *Reporter) ForeachStart(itemCount int) {
	r.println(fmt.Sprintf("   %s Foreach: Processing %d items", IconForeachStart, itemCount))
}

// ForeachItem reports progress on a foreach item
func (r *Reporter) ForeachItem(index, total int, status string) {
	r.println(fmt.Sprintf("      [%d/%d] %s", index, total, status))
}

// ForeachComplete indicates foreach loop completed
func (r *Reporter) ForeachComplete(successCount, totalCount int, duration time.Duration) {
	r.println(fmt.Sprintf("   %s Foreach completed: %d/%d items succeeded (%.1fs)",
		IconForeachItem, successCount, totalCount, duration.Seconds()))
	r.println("")
}

// Retry and recovery events

// RetryAttempt indicates a retry is being attempted
func (r *Reporter) RetryAttempt(attempt, maxAttempts int, delay time.Duration) {
	if attempt == 1 {
		r.println(fmt.Sprintf("   %s Step failed (attempt %d/%d)", IconWarning, attempt, maxAttempts))
		if delay > 0 {
			r.println(fmt.Sprintf("      %s Retrying in %.0fs...", IconRouterMatched, delay.Seconds()))
		}
	} else {
		r.updateLine(fmt.Sprintf("   %s Retry attempt %d/%d...", IconRetryAttempt, attempt, maxAttempts))
	}
}

// RetrySuccess indicates retry succeeded
func (r *Reporter) RetrySuccess(attempt int) {
	r.println(fmt.Sprintf("   %s Retry succeeded on attempt %d", IconRetrySuccess, attempt))
}

// RetryExhausted indicates all retry attempts failed
func (r *Reporter) RetryExhausted(maxAttempts int) {
	r.println(fmt.Sprintf("   %s All %d retry attempts exhausted", IconRetryExhausted, maxAttempts))
}

// LLM thinking events

// ThinkingContent displays LLM thinking/reasoning content
// This is used when --llm-think is enabled to show intermediate reasoning
func (r *Reporter) ThinkingContent(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.quiet {
		return
	}

	// Mark that we've shown thinking content
	r.hasThinking = true

	// Store the current processing line to restore it after thinking
	processingLine := r.lastLine

	// If there's a mutable line (e.g., "   ⚙️  Step 1/2: main → Processing..."),
	// finalize it before showing thinking header
	if r.lastLine != "" {
		fmt.Fprintln(r.w) // Move to new line, keep the processing line visible
		r.lastLine = ""
	}

	// Print thinking header (e.g., "   💭 Step 1/2: main → Thinking")
	if processingLine != "" {
		// Extract step info from processing line and create thinking header
		thinkingHeader := strings.Replace(processingLine, "Processing", "Thinking", 1)
		thinkingHeader = strings.Replace(thinkingHeader, IconStepProcessing, IconThinking, 1)
		fmt.Fprintln(r.w, thinkingHeader)
	}

	// Get terminal width for wrapping (default to 80 if not a terminal)
	width := 80
	if fd, ok := r.w.(*os.File); ok {
		if w, _, err := term.GetSize(int(fd.Fd())); err == nil && w > 0 {
			width = w
		}
	}

	// Print thinking content with proper indentation and wrapping
	// Format: "   💭 <first line text>"
	//         "      <continuation lines>"

	// Use a buffer to capture wrapped output
	var buf strings.Builder
	tw, err := twrap.NewTWConf(
		twrap.SetWriter(&buf),
		twrap.SetTargetLineLen(width-6), // Reserve space for indent (6 chars: 3 spaces + "💭 ")
	)
	if err != nil {
		// Fallback: print without wrapping if configuration fails
		fmt.Fprintf(r.w, "   %s %s\n", IconThinking, text)
		return
	}

	// Wrap text without indent first
	tw.Wrap(text, 0)

	// Now add indentation to each line (thinking content is indented)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	for _, line := range lines {
		fmt.Fprintf(r.w, "      %s\n", line) // All lines indented (no emoji on content lines)
	}

	// Restore the processing line as mutable after thinking
	if processingLine != "" {
		r.lastLine = processingLine
		fmt.Fprint(r.w, processingLine) // Print without newline (mutable)
	} else {
		fmt.Fprintln(r.w) // Just add blank line if no processing line to restore
	}
}

// Batch processing events

// BatchStart indicates batch processing is starting
func (r *Reporter) BatchStart(inputPath, outputPath string, mutable bool) {
	if mutable {
		r.println(fmt.Sprintf("%s Batch processing: %s → %s (mutable mode)", IconBatchProcessing, inputPath, outputPath))
	} else {
		r.println(fmt.Sprintf("%s Batch processing: %s → %s", IconBatchProcessing, inputPath, outputPath))
	}
}

// BatchProgress reports batch processing progress
func (r *Reporter) BatchProgress(processed, total int) {
	if total > 0 {
		percent := int(float64(processed) / float64(total) * 100)
		bars := percent / 5 // 20 bars total
		progress := strings.Repeat("█", bars) + strings.Repeat("░", 20-bars)
		r.updateLine(fmt.Sprintf("Processing: %s %d/%d files (%d%%)",
			progress, processed, total, percent))
	} else {
		r.updateLine(fmt.Sprintf("Processing: %d files...", processed))
	}
}

// BatchComplete finalizes batch progress
func (r *Reporter) BatchComplete() {
	if r.lastLine != "" {
		r.println("") // Move to new line
	}
}

// Chunking events

// ChunkingStart indicates document is being split
func (r *Reporter) ChunkingStart(strategy string, chunkCount int) {
	r.println(fmt.Sprintf("   %s Splitting into %d chunks (strategy: %s)", IconChunking, chunkCount, strategy))
}

// Summary and statistics

// Summary prints final execution summary
func (r *Reporter) Summary() {
	duration := time.Since(r.startTime)

	r.println("")
	r.println(fmt.Sprintf("%s Pipeline Summary:", IconSummary))

	if r.stats.DocsProcessed > 0 {
		r.println(fmt.Sprintf("   %s Processed: %d documents", IconDocumentComplete, r.stats.DocsProcessed))
	}

	if r.stats.DocsSkipped > 0 {
		r.println(fmt.Sprintf("   %s Skipped: %d documents", IconDocumentSkipped, r.stats.DocsSkipped))
	}

	if r.stats.DocsErrors > 0 {
		r.println(fmt.Sprintf("   %s Errors: %d documents", IconWarning, r.stats.DocsErrors))
	}

	// Use token source for authoritative usage if available, otherwise fall back to step-level stats
	r.mu.Lock()
	tokenSource := r.tokenSource
	r.mu.Unlock()

	if tokenSource != nil {
		// Use Router's authoritative token data with per-profile breakdown
		usage := tokenSource.Usage()
		profiles := tokenSource.ProfileUsage()

		if usage.InputTokens > 0 || usage.ReplyTokens > 0 {
			total := usage.InputTokens + usage.ReplyTokens
			r.println(fmt.Sprintf("   📈 Tokens: %s total",
				formatNumber(total)))

			// Show per-profile breakdown
			for _, profile := range profiles {
				if profile.Usage.InputTokens > 0 || profile.Usage.ReplyTokens > 0 {
					profileTotal := profile.Usage.InputTokens + profile.Usage.ReplyTokens
					r.println(fmt.Sprintf("      └─ %s: %s (input: %s | output: %s)",
						profile.Name,
						formatNumber(profileTotal),
						formatNumber(profile.Usage.InputTokens),
						formatNumber(profile.Usage.ReplyTokens)))
				}
			}
		}
	} else if r.stats.TokensInput > 0 || r.stats.TokensOutput > 0 {
		// Fall back to step-level token tracking (legacy)
		total := r.stats.TokensInput + r.stats.TokensOutput
		r.println(fmt.Sprintf("   📈 Tokens: %s (input: %s | output: %s)",
			formatNumber(total),
			formatNumber(r.stats.TokensInput),
			formatNumber(r.stats.TokensOutput)))
	}

	r.println(fmt.Sprintf("   ⏱️  Duration: %s", formatDuration(duration)))

	// Add visual separator before output
	r.println("")
	r.println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	r.println("")
}

// SummaryQuick prints a compact one-line summary
func (r *Reporter) SummaryQuick(docCount int, duration time.Duration) {
	tokens := r.stats.TokensInput + r.stats.TokensOutput
	if tokens > 0 {
		r.println(fmt.Sprintf("\n📊 Summary: %d document(s), %s tokens, %s",
			docCount, formatNumber(tokens), formatDuration(duration)))
	} else {
		r.println(fmt.Sprintf("\n📊 Summary: %d document(s), %s",
			docCount, formatDuration(duration)))
	}
}

// Helper methods

// UpdateTokens updates token statistics
func (r *Reporter) UpdateTokens(inputTokens, outputTokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.TokensInput += inputTokens
	r.stats.TokensOutput += outputTokens
}

// GetStats returns a copy of current statistics
func (r *Reporter) GetStats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// updateLine updates the current line (mutable output)
func (r *Reporter) updateLine(text string) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear previous line
	if r.lastLine != "" {
		fmt.Fprintf(r.w, "\r\033[K")
	}

	fmt.Fprint(r.w, text)
	r.lastLine = text
}

// println writes a new line (finalize any mutable line first)
func (r *Reporter) println(text string) {
	if r.quiet {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.lastLine != "" {
		if r.hasThinking {
			// In thinking mode: keep the previous line, just move to next line
			fmt.Fprintln(r.w)
		} else {
			// Without thinking: clear the mutable line and replace it
			fmt.Fprint(r.w, "\r\033[K")
		}
		r.lastLine = ""
	}
	fmt.Fprintln(r.w, text)
}

// Formatting helpers

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", d.Seconds()*1000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
