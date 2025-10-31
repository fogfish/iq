# Pipeline Config Pattern Refactoring

**Date:** 31 October 2025  
**Status:** ✅ Complete

## Overview

Refactored the Pipeline API from option pattern to config structure pattern for better clarity and consistency with the codebase conventions.

## Changes Made

### 1. Core API Changes

**Before (Option Pattern):**
```go
pipeline := iosystem.NewPipeline(src, snk,
    iosystem.WithProgress(progressFunc),
    iosystem.WithMetrics(metricsFunc),
    iosystem.WithConcurrency(4),
    iosystem.WithErrorMode(iosystem.FailFast),
)
```

**After (Config Structure):**
```go
config := &iosystem.PipelineConfig{
    Progress:    progressFunc,
    Metrics:     metricsFunc,
    Concurrency: 4,
    ErrorMode:   iosystem.FailFast,
}
pipeline := iosystem.NewPipeline(src, snk, config)

// Or use defaults with nil
pipeline := iosystem.NewPipeline(src, snk, nil)
```

### 2. Files Modified

- **`internal/iosystem/pipeline.go`**
  - Renamed `PipelineOptions` → `PipelineConfig`
  - Renamed field `options` → `config` in Pipeline struct
  - Changed `NewPipeline` signature: `func NewPipeline(source Source, sink Sink, config *PipelineConfig)`
  - Removed option functions: `WithProgress`, `WithMetrics`, `WithConcurrency`, `WithErrorMode`
  - Updated all internal references from `p.options.*` to `p.config.*`

- **`internal/iosystem/pipeline_test.go`**
  - Updated all test cases to use config struct instead of option functions
  - Changed `NewPipeline(src, snk)` → `NewPipeline(src, snk, nil)` for default config
  - Replaced option pattern with inline config structs

- **`internal/iosystem/examples/basic/main.go`**
  - Updated both examples to use config struct
  - Demonstrated config creation and usage

- **`internal/iosystem/README.md`**
  - Updated all code examples to use new config-based API
  - Maintained consistency across all examples

### 3. Benefits

✅ **Clearer Intent:** Config struct shows all options at a glance  
✅ **Easier to Test:** Can create and reuse config objects  
✅ **Better Consistency:** Matches other config patterns in the codebase (e.g., `ChunkConfig`, `WalkConfig`)  
✅ **Simpler API:** Fewer exported functions, cleaner interface  
✅ **Nil-safe:** Passing `nil` config uses sensible defaults  

### 4. Default Configuration

When `nil` is passed as config, defaults are applied:
- `Concurrency: 1` (sequential processing)
- `ErrorMode: FailFast` (stop on first error)
- `Progress: nil` (no progress callback)
- `Metrics: nil` (no metrics callback)

### 5. Testing

All tests pass:
```bash
$ go test ./internal/iosystem/...
ok      github.com/fogfish/iq/internal/iosystem         0.310s
ok      github.com/fogfish/iq/internal/iosystem/processor 0.167s
ok      github.com/fogfish/iq/internal/iosystem/sink    0.188s
ok      github.com/fogfish/iq/internal/iosystem/source  0.368s
```

No lint issues:
```bash
$ go vet ./internal/iosystem/...
(clean output)
```

## Migration Guide

For any existing code using the old API:

**Old Code:**
```go
pipeline := iosystem.NewPipeline(src, snk,
    iosystem.WithProgress(progressFunc),
)
```

**New Code:**
```go
config := &iosystem.PipelineConfig{
    Progress: progressFunc,
}
pipeline := iosystem.NewPipeline(src, snk, config)
```

For pipelines with no options:
```go
// Old: pipeline := iosystem.NewPipeline(src, snk)
// New:
pipeline := iosystem.NewPipeline(src, snk, nil)
```

## Backwards Compatibility

⚠️ **Breaking Change:** This is a breaking API change. The old option pattern functions have been removed.

Since this is Phase 1 and the iosystem package is new/internal, there are no external consumers to migrate.
