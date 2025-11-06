# ADR 0003: Progress Reporting System

**Status:** Accepted  
**Date:** 2025-11-05  
**Deciders:** Architecture Team  
**Context:** Users need visibility into workflow execution progress

## Context and Problem Statement

The `iq` tool executes complex multi-step AI workflows that can take significant time. Users had no visibility into:
- What the system is currently doing
- How long steps take
- Whether the system is making progress or stuck
- Which workflow paths are being executed (routing, iterations)

This lack of feedback creates poor user experience, especially for long-running workflows or batch processing.

## Decision Drivers

- **Educational over decorative**: Users should understand what's happening, not just see activity
- **Non-intrusive**: Progress must never interfere with actual output (stdout)
- **Visually appealing**: Modern, emoji-based UI preferred over ASCII colors/spinners
- **Optional**: Must support quiet mode for scripting
- **Maintainable**: Clean architecture that doesn't couple layers

## Considered Options

1. **Callback-based only** - Use existing ProgressFunc/MetricsFunc throughout
2. **Context-based only** - Pass reporter through context, remove callbacks
3. **Hybrid approach** - Context for rich reporting, callbacks for lifecycle hooks

## Decision Outcome

**Chosen option: Hybrid approach (Option 3)**

### Architecture

```
| Layer          | Responsibility              | Mechanism                |
| -------------- | --------------------------- | ------------------------ |
| cmd/           | Create & configure reporter | Direct instantiation     |
| service/worker | Wire to conduit lifecycle   | Callbacks → Reporter     |
| conduit/       | Document lifecycle events   | ProgressFunc/MetricsFunc |
| workflow/      | Step-level progress         | Context-based Reporter   |
```

### Key Components

#### 1. Progress Reporter (`internal/progress`)
- Central progress reporting with emoji-based UI
- Thread-safe mutable output (updates lines in place)
- Comprehensive event methods for all workflow phases
- Passed through context for availability everywhere

#### 2. Context Integration
- `progress.WithReporter(ctx, reporter)` embeds reporter in context
- `progress.FromContext(ctx)` extracts reporter from context
- Flows transparently through entire execution chain
- No explicit parameter passing needed at deep levels

#### 3. Lifecycle Callbacks (conduit layer)
- `ProgressFunc(doc, err)` - Document-level events
- `MetricsFunc(stats)` - Final pipeline statistics
- Provide **extensibility** for monitoring/metrics/logging
- Keep conduit **decoupled** from specific progress implementations

### Rationale

**Why hybrid approach?**

1. **Separation of concerns**: Conduit doesn't depend on emoji progress reporter
2. **Testability**: Can test conduit with mock callbacks, test reporter independently
3. **Extensibility**: Callbacks allow future metrics exporters (Prometheus, webhooks, etc.)
4. **Context benefits**: Reporter available deep in workflow without parameter drilling
5. **Clean boundaries**: Infrastructure (conduit) vs. domain (workflow) vs. presentation (reporter)

**Why context over parameters?**

- Workflow execution is deeply nested (jobs → steps → retries)
- Passing reporter explicitly through 5+ function layers is unwieldy
- Context is idiomatic Go for cross-cutting concerns
- Easy to make optional (nil check on `FromContext`)

**Why keep callbacks?**

- They're **lifecycle hooks**, not UI
- Enable future observability integrations
- Maintain architectural cleanliness
- Minimal overhead (2 callback fields)

## User Experience Design

## Progress Points

### Workflow Loading & Compilation
```
📋 Loading workflow from research.yml
✅ Workflow compiled: "research" (2 jobs, 4 steps)
```

### Document Processing
```
📄 Processing: document.txt (2.3 KB)
```

### Step Execution
```
   🤖 Step 1/2: main.extract → Processing...
   ✅ Step 1/2: main.extract completed in 2.1s
```

### Router Decisions
```
   🧭 Router evaluating conditions...
   🧭 Routing decision:
      ↳ Matched route: high_priority → urgent_handler
```

### Foreach Iterations
```
   🔁 Foreach: Processing 5 items
      [1/5] ✅ item-1 → completed
      [2/5] ✅ item-2 → completed
      [3/5] ✅ item-3 → completed
```

### Retry Attempts
```
   ⚠️  Step failed (attempt 1/3)
      ↳ Retrying in 2s...
   🔄 Retry attempt 2/3...
   ✅ Retry succeeded on attempt 2
```

### Final Summary
```
📊 Pipeline Summary:
   ✅ Processed: 95 documents
   ⚠️  Errors: 2 documents
   📈 Tokens: 1.2M (input: 890.1K | output: 344.4K)
   ⏱️  Duration: 5m 32s
```

## Emoji Legend

| Emoji | Meaning               |
| ----- | --------------------- |
| 📋     | Loading/configuration |
| 🔍     | Validation/analysis   |
| ✅     | Success/completion    |
| ⚠️     | Warning/error         |
| 🔄     | Retry/recovery        |
| 📄     | Document/file         |
| 📂     | Batch/directory       |
| 🤖     | LLM/agent execution   |
| 🧭     | Router/decision       |
| 🔁     | Foreach/iteration     |
| 🔪     | Chunking/splitting    |
| 📊     | Statistics/summary    |
| ⏱️     | Time/duration         |
| 📈     | Metrics/tokens        |
| ⏭️     | Skipped               |
| ❌     | Failed                |


### Visual Language

Educational progress using emoji icons for universal recognition:

| Icon | Meaning               | Usage                   |
| ---- | --------------------- | ----------------------- |
| 📋    | Loading/configuration | Workflow file loading   |
| ✅    | Success/completion    | Successful completion   |
| 🤖    | LLM/agent execution   | Step processing         |
| 🧭    | Router/decision       | Conditional routing     |
| 🔁    | Foreach/iteration     | Loop processing         |
| 🔄    | Retry/recovery        | Error recovery attempts |
| ⚠️    | Warning/error         | Failures, issues        |
| 📊    | Statistics/summary    | Final metrics           |
| ⏱️    | Time/duration         | Timing information      |
| 📈    | Metrics/tokens        | Token usage             |

### Output Principles

1. **Mutable when possible**: Update current line for compact display
2. **Finalize important events**: New line for completed steps
3. **Hierarchical indentation**: Visual structure (main → step → iteration)
4. **Informative messages**: "Step 1/3: main.extract → Processing..." not "Processing..."
5. **Clean separation**: All progress to stderr, results to stdout

### Example Output

```
📋 Loading workflow from research.yml
✅ Workflow compiled: "research" (2 jobs, 4 steps)

   🤖 Step 1/2: main.extract → Processing...
   ✅ Step 1/2: main.extract completed in 2.1s
   
   🧭 Router evaluating conditions...
   🧭 Routing decision:
      ↳ Matched route: high_priority → urgent_handler
   
   🔁 Foreach: Processing 5 items
      [1/5] ✅ item-1 → completed
      [2/5] ✅ item-2 → completed
      ...

📊 Pipeline Summary:
   ⏱️ Duration: 4.8s
```

## Implementation Details

### Reporter Creation & Propagation

```go
// cmd/agent.go
reporter := progress.New(quiet)
srv, err := worker.Build(llm, reporter)

// service/worker/worker.go
ctx = progress.WithReporter(ctx, reporter)
conduit.Run(ctx, source, sink)

// compiler/workflow.go
reporter := progress.FromContext(ctx)
if reporter != nil {
    reporter.StepStart(...)
}
```

### Lifecycle Hook Usage

```go
// service/worker/worker.go
conduit.Progress = func(doc *iosystem.Document, err error) {
    // Bridge to reporter
    if err != nil {
        reporter.DocumentError(doc.Path, err)
    }
}

conduit.Metrics = func(stats conduit.Stats) {
    // Final summary
    reporter.Summary()
}
```

## Consequences

### Positive

- ✅ Users have clear visibility into workflow execution
- ✅ Educational messages help understand system behavior
- ✅ Quiet mode supports scripting/automation
- ✅ Clean architecture maintains separation of concerns
- ✅ Extensible for future monitoring/observability
- ✅ All existing tests pass without modification

### Negative

- ⚠️ Two mechanisms (callbacks + context) may confuse new contributors
- ⚠️ Progress reporter in context is implicit (not in function signatures)
- ⚠️ Emoji output may not render correctly in all terminals

### Neutral

- 📝 Requires documentation for maintainers (this ADR)
- 📝 Future: Consider i18n for educational messages
- 📝 Future: Progress bar support for batch processing

## Compliance

- Follows Go idioms (context for cross-cutting concerns)
- Maintains backward compatibility (all tests pass)
- stderr for progress, stdout for results (UNIX convention)
- Quiet flag for scripting (common CLI pattern)


## References

- Implementation: `internal/progress/progress.go`
- Tests: `internal/progress/progress_test.go`
- Documentation: `doc/progress-reporting.md`
- Context pattern: https://go.dev/blog/context
