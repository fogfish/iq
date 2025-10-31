# Context Management - The Idiomatic Way

## Design Decision

Initially implemented as `*Context` parameter (wtfCtx 😄), but refactored to use Go's idiomatic `context.Context` with `context.WithValue`.

## Implementation

### Context Key
```go
// Private type to avoid collisions
type contextKey int

const workflowContextKey contextKey = iota
```

### WorkflowContext Structure
```go
type WorkflowContext struct {
    Input   any            // Original workflow input
    State   map[string]any // Shared state
    Steps   map[string]any // Named step outputs
    Current any            // Latest output (for chaining)
}
```

### Usage

**Creating context** (in Job.Prompt):
```go
ctx = NewWorkflowContext(ctx, input)
```

**Extracting context** (in steps):
```go
wfCtx := GetWorkflowContext(ctx)
```

**Sub-job execution** (in routers):
```go
// Job.Prompt checks if workflow context already exists
// If yes, reuses it; if no, creates new one
jobResult, err := job.Prompt(ctx, savedCurrent, opt...)
```

## Benefits

1. **Idiomatic Go**: Uses standard `context.Context` patterns
2. **Automatic propagation**: Context flows through all function calls
3. **Cancellation support**: Inherits Go's cancellation semantics
4. **Type safety**: Private contextKey prevents collisions
5. **Clean signatures**: No extra `wfCtx *Context` parameter
6. **Testability**: Easy to inject test contexts

## API

### Step Interface
```go
type Step interface {
    Prompt(ctx context.Context, opt ...chatter.Opt) error
    GetOutputName() string
}
```

### Job Execution
```go
type Job struct {
    Name  string
    Steps []Step
}

func (j *Job) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error)
```

## Example Flow

```
User calls: job.Prompt(ctx, input)
  ↓
Job creates: ctx = NewWorkflowContext(ctx, input)
  ↓
Step 1: wfCtx := GetWorkflowContext(ctx)
        → Execute agent with wfCtx.ToMap()
        → Store result: wfCtx.SetStepOutput("step1", result)
  ↓
Step 2: wfCtx := GetWorkflowContext(ctx)  // Same context!
        → Access: {{.steps.step1}}, {{.input}}, {{.current}}
        → Execute and update context
  ↓
Job returns: wfCtx.Current
```

## Router Sub-Job Handling

Routers save/restore `Current` when calling sub-jobs:

```go
savedCurrent := wfCtx.Current
jobResult, err := job.Prompt(ctx, savedCurrent, opt...)
wfCtx.Current = jobResult // or store with name
```

This allows routed jobs to execute with their own `Current` value while maintaining the workflow context.

## No More wtfCtx! 🎉

Clean, idiomatic, and leverages Go's standard patterns.
