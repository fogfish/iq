# I/O System Phases 1-4: Comprehensive Review

**Date:** October 31, 2025  
**Status:** Pre-Phase 5 Review  
**Test Results:** 49 tests, 47 passing, 2 deferred to Phase 5

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Architecture Review](#architecture-review)
3. [Implementation Analysis](#implementation-analysis)
4. [Test Coverage Assessment](#test-coverage-assessment)
5. [Critical Design Decisions](#critical-design-decisions)
6. [Issues and Improvements](#issues-and-improvements)
7. [Phase 5 Readiness](#phase-5-readiness)

---

## Executive Summary

### What We Built

**Phase 1: Core Abstractions** ✅
- `Document`, `Source`, `Processor`, `Sink` interfaces
- Basic implementations: stdin/stdout, file sources/sinks
- Identity and Chunker processors
- Generic `Pipeline` orchestrator

**Phase 2: Filesystem Sources/Sinks** ✅
- `FSSink` for writing to local/S3 filesystems
- `MergedSource` for combining multiple sources
- **`SpoolPipeline`** - Special pipeline for transactional directory processing

**Phase 3: Blueprint Integration** ✅
- `AgentProcessor` - Wraps `blueprint/compiler.Agent`
- `BlueprintProcessor` - Executes complete blueprint workflows

**Phase 4: Pipeline Orchestrator** ✅
- Generic `Pipeline` with error handling, callbacks, stats
- Already existed, confirmed complete

### Key Achievement: Solved Transactional Semantics Problem

**Problem Discovered:** Original `WalkSource` approach broke `spool.ForEach` transactions by faking output filesystem, causing input files to be removed before processing completed.

**Solution:** Created `SpoolPipeline` - a specialized pipeline that wraps the entire processor chain inside `spool.ForEach`, ensuring transactional guarantees.

### Architecture Pattern

```
Regular Workflows:
  Source → Processor → Sink (via generic Pipeline)
  Examples: stdin/stdout, single files, streaming

Directory Processing (Transactional):
  SpoolPipeline wraps: Source → Processor → Sink
  Guarantees: Input removed only after successful output write
  Examples: ask command, run command with mutable directories
```

---

## Architecture Review

### Current Structure

```
internal/iosystem/
├── document.go              # Document type with metadata
├── source.go                # Source interface
├── sink.go                  # Sink interface
├── processor.go             # Processor interface
├── pipeline.go              # Generic pipeline orchestrator
├── spool_pipeline.go        # Transactional directory pipeline
│
├── source/
│   ├── stdin.go             # StdinSource
│   ├── file.go              # FileSource, FileSeqSource
│   ├── merged.go            # MergedSource
│   └── walk.go              # WalkSource (DEPRECATED)
│
├── processor/
│   ├── processor.go         # IdentityProcessor, ChunkerProcessor
│   ├── agent.go             # AgentProcessor (blueprint integration)
│   └── blueprint.go         # BlueprintProcessor (workflow execution)
│
└── sink/
    ├── stdout.go            # StdoutSink
    └── fs.go                # FSSink (local/S3 via stream)
```

### Design Patterns Used

1. **Interface Segregation**: Small, focused interfaces (Source, Processor, Sink)
2. **Composition**: Pipeline composes sources/processors/sinks
3. **Strategy Pattern**: Different processors for different transformations
4. **Iterator Pattern**: Source.Next() for streaming
5. **Pipeline Pattern**: Chained processing stages
6. **Options Pattern**: Functional options for configuration

---

## Implementation Analysis

### Phase 1: Core Abstractions

#### Document Type
```go
type Document struct {
    Path     string
    Content  io.Reader
    Metadata map[string]any
}
```

**Strengths:**
- ✅ Simple, clear structure
- ✅ Metadata flexibility with `map[string]any`
- ✅ Streaming with `io.Reader`

**Concerns:**
- ⚠️ **No Close() method** - Documents can hold open file handles
- ⚠️ **No size tracking** - Can't estimate memory/bandwidth
- ⚠️ **Metadata type safety** - `any` requires type assertions everywhere
- ⚠️ **No error context** - Can't track where document came from for debugging

**Potential Improvements:**
```go
type Document struct {
    Path     string
    Content  io.ReadCloser  // Can close resources
    Metadata Metadata       // Type-safe metadata
    Size     int64          // Known size (-1 if unknown)
    Source   string         // Origin for debugging
}

type Metadata map[string]any

// Helper methods
func (m Metadata) GetString(key string) (string, bool)
func (m Metadata) GetInt(key string) (int, bool)
func (m Metadata) Set(key string, value any)
```

#### Source Interface
```go
type Source interface {
    Next(ctx context.Context) (*Document, error)
    Close() error
}
```

**Strengths:**
- ✅ Clean iterator pattern
- ✅ Context support for cancellation
- ✅ Close for cleanup

**Concerns:**
- ⚠️ **No peek/rewind** - Can't look ahead or restart
- ⚠️ **No count estimation** - Pipeline can't show progress percentage
- ⚠️ **No metadata** - Source can't describe itself (type, config)

**Potential Improvements:**
```go
type Source interface {
    Next(ctx context.Context) (*Document, error)
    Close() error
    
    // Optional: Estimate total documents (return -1 if unknown)
    Count() int64
    
    // Optional: Get source metadata
    Info() SourceInfo
}

type SourceInfo struct {
    Type   string         // "stdin", "file", "fs", "http"
    Path   string         // Logical path
    Config map[string]any // Source configuration
}
```

#### Processor Interface
```go
type Processor interface {
    Process(ctx context.Context, doc *Document) ([]*Document, error)
    Close() error
}
```

**Strengths:**
- ✅ 1→N transformation (chunking, expanding)
- ✅ Context support
- ✅ Close for cleanup

**Concerns:**
- ⚠️ **No filtering** - Can't signal "skip this document" cleanly (return empty slice?)
- ⚠️ **No state** - Can't aggregate across documents (e.g., "collect all, then merge")
- ⚠️ **No metadata** - Processor can't describe capabilities

**Potential Improvements:**
```go
type Processor interface {
    Process(ctx context.Context, doc *Document) ([]*Document, error)
    Close() error
    
    // Optional: Get processor metadata
    Info() ProcessorInfo
}

type ProcessorInfo struct {
    Name        string         // "chunker", "agent", "blueprint"
    Description string
    Config      map[string]any
}

// Special case: Return nil slice to skip document
// Return error to signal processing failure
```

#### Sink Interface
```go
type Sink interface {
    Write(ctx context.Context, doc *Document) error
    Close() error
}
```

**Strengths:**
- ✅ Simple write interface
- ✅ Context support
- ✅ Close for finalization

**Concerns:**
- ⚠️ **No batch writes** - Each document written individually
- ⚠️ **No transaction support** - Can't rollback on error
- ⚠️ **No write confirmation** - Can't get bytes written or location

**Potential Improvements:**
```go
type Sink interface {
    Write(ctx context.Context, doc *Document) (*WriteResult, error)
    Close() error
}

type WriteResult struct {
    Path         string // Where document was written
    BytesWritten int64
    Metadata     map[string]any
}

// Optional: Batch write interface
type BatchSink interface {
    Sink
    WriteBatch(ctx context.Context, docs []*Document) ([]*WriteResult, error)
}
```

#### Generic Pipeline
```go
type Pipeline struct {
    source     Source
    processors []Processor
    sink       Sink
    config     PipelineConfig
}
```

**Strengths:**
- ✅ Clean composition model
- ✅ Error handling (FailFast/SkipError)
- ✅ Progress callbacks
- ✅ Stats tracking
- ✅ Processor chaining

**Concerns:**
- ⚠️ **No concurrency** - Processes documents sequentially (intentionally skipped)
- ⚠️ **No backpressure** - Fast sources can overwhelm slow sinks
- ⚠️ **No retry logic** - Single failure = skip or abort
- ⚠️ **Stats limited** - No timing per stage, no throughput metrics

**Potential Improvements:**
```go
type PipelineStats struct {
    DocsProcessed   int64
    DocsSkipped     int64
    DocsFailed      int64
    BytesRead       int64
    BytesWritten    int64
    Duration        time.Duration
    
    // Per-stage timing
    SourceTime      time.Duration
    ProcessorTimes  []time.Duration // One per processor
    SinkTime        time.Duration
    
    // Throughput
    DocsPerSecond   float64
    BytesPerSecond  float64
}
```

### Phase 2: Filesystem Sources/Sinks

#### FSSink
```go
type FSSink struct {
    fsys spool.FileSystem
    root string
}
```

**Strengths:**
- ✅ Uses `stream` library for S3/local abstraction
- ✅ Preserves document path structure
- ✅ Clean interface

**Concerns:**
- ⚠️ **No directory creation check** - Assumes all parent dirs exist
- ⚠️ **No overwrite control** - Always overwrites existing files
- ⚠️ **No atomic writes** - Partial writes on error leave corrupt files
- ⚠️ **No compression** - Could support .gz suffix auto-compression

**Potential Improvements:**
```go
type FSSinkConfig struct {
    Root          string
    Overwrite     bool           // Allow overwriting existing files
    CreateDirs    bool           // Create parent directories
    TempSuffix    string         // Write to .tmp, rename on success
    Compression   string         // "gzip", "none"
    FileMode      os.FileMode    // Unix permissions
}

// Atomic write pattern:
// 1. Write to path + ".tmp"
// 2. On success, rename to final path
// 3. On error, remove .tmp file
```

#### MergedSource
```go
type MergedSource struct {
    sources   []Source
    separator []byte
}
```

**Strengths:**
- ✅ Combines multiple sources into one document
- ✅ Configurable separator
- ✅ Metadata tracking

**Concerns:**
- ⚠️ **Memory buffering** - Loads all content into memory
- ⚠️ **No streaming** - Could use io.MultiReader for zero-copy
- ⚠️ **Fixed separator** - No template/formatting options

**Potential Improvements:**
```go
type MergedSource struct {
    sources   []Source
    separator []byte
    buffer    bool  // If false, use streaming io.MultiReader
    format    string // Template: "{{.Path}}:\n{{.Content}}\n\n"
}

// Streaming implementation:
func (s *MergedSource) Next(ctx context.Context) (*Document, error) {
    if s.consumed {
        return nil, io.EOF
    }
    
    var readers []io.Reader
    for _, source := range s.sources {
        doc, err := source.Next(ctx)
        if err != nil {
            return nil, err
        }
        readers = append(readers, doc.Content, bytes.NewReader(s.separator))
    }
    
    return &Document{
        Path:    "merged",
        Content: io.MultiReader(readers...),
    }, nil
}
```

#### SpoolPipeline (Key Innovation)
```go
type SpoolPipeline struct {
    spool      *spool.Spool
    processors []Processor
    config     SpoolConfig
}

type SpoolConfig struct {
    Mutable          bool
    Strict           bool
    AllowEmptyOutput bool
}
```

**Strengths:**
- ✅ Solves transactional semantics problem
- ✅ Integrates with `spool.ForEach`
- ✅ Supports mutable mode (queue pattern)
- ✅ AllowEmptyOutput flag (user request)
- ✅ Comprehensive tests (9/9 passing)

**Concerns:**
- ⚠️ **Not a true Source/Sink pipeline** - Special implementation
- ⚠️ **Limited to directory processing** - Can't mix with other sources
- ⚠️ **No progress per file** - Callback only at end
- ⚠️ **No partial recovery** - If one file fails, whole batch may be lost

**Potential Improvements:**
```go
type SpoolConfig struct {
    Mutable          bool
    Strict           bool
    AllowEmptyOutput bool
    
    // New options
    Progress         func(path string, stage string) // Per-file progress
    RetryAttempts    int                             // Retry failed files
    RetryDelay       time.Duration
    Checkpoint       string                          // Save progress to file
}

// Checkpoint format (resume capability):
// processed_files.txt:
//   file1.txt: success
//   file2.txt: failed
//   file3.txt: success
// On restart, skip successful files, retry failed
```

### Phase 3: Blueprint Integration

#### AgentProcessor
```go
type AgentProcessor struct {
    agent  *compiler.Agent
    suffix string
    opts   []chatter.Opt
}
```

**Strengths:**
- ✅ Clean wrapper around `compiler.Agent`
- ✅ Metadata templating support
- ✅ Multiple input formats (string, []byte, map)
- ✅ JSON format output support

**Concerns:**
- ⚠️ **No streaming** - Loads entire document into memory
- ⚠️ **No retry logic** - LLM failures are immediate errors
- ⚠️ **No token counting** - Can't estimate costs or limits
- ⚠️ **No timeout control** - Uses default context timeout
- ⚠️ **Metadata preparation** - Complex logic in Process(), hard to test

**Potential Improvements:**
```go
type AgentConfig struct {
    Suffix       string
    Options      []chatter.Opt
    MaxTokens    int           // Reject if input too large
    Timeout      time.Duration // Per-agent timeout
    RetryCount   int           // LLM retry attempts
    RetryDelay   time.Duration
    StreamOutput bool          // Stream LLM response
}

// Add token counting
func (p *AgentProcessor) EstimateTokens(doc *Document) (int, error)

// Better metadata preparation - extract to helper
func prepareAgentInput(doc *Document, template bool) (any, error)
```

#### BlueprintProcessor
```go
type BlueprintProcessor struct {
    blueprint *blueprint.Blueprint
    jobName   string
    suffix    string
    opts      []chatter.Opt
}
```

**Strengths:**
- ✅ Executes complete workflows
- ✅ Supports both Execute() and Run()
- ✅ Metadata templating
- ✅ Multiple output formats

**Concerns:**
- ⚠️ **Similar to AgentProcessor** - Code duplication in metadata handling
- ⚠️ **No workflow state visibility** - Can't see intermediate steps
- ⚠️ **No partial results** - If workflow fails midway, lose everything
- ⚠️ **No workflow timeout** - Could run forever

**Potential Improvements:**
```go
type BlueprintConfig struct {
    Suffix       string
    Options      []chatter.Opt
    Timeout      time.Duration
    SaveState    bool          // Save workflow state on error
    StepProgress func(step string, output any) // Per-step callback
}

// Extract common metadata logic
func prepareWorkflowInput(doc *Document, template bool) (any, error)

// Workflow state recovery
func (p *BlueprintProcessor) SaveState(path string) error
func (p *BlueprintProcessor) LoadState(path string) error
```

---

## Test Coverage Assessment

### Current Test Results (49 tests, 47 passing, 2 deferred)

```
Package: internal/iosystem (10 tests)
✅ Document creation and metadata
✅ Pipeline basic flow
✅ Pipeline with chunking
✅ Pipeline with multiple processors
✅ Pipeline error modes (FailFast, SkipError)
✅ Pipeline progress callback
✅ Pipeline metrics callback
✅ Pipeline stats tracking
✅ SpoolPipeline (9 subtests)

Package: internal/iosystem/processor (7 tests)
⏸️ AgentProcessor (DEFERRED to Phase 5)
⏸️ BlueprintProcessor (DEFERRED to Phase 5)
✅ IdentityProcessor
✅ ChunkerProcessor (5 subtests)

Package: internal/iosystem/sink (12 tests)
✅ FSSink (9 subtests - various scenarios)
✅ StdoutSink (3 subtests)

Package: internal/iosystem/source (11 tests)
✅ FileSource
✅ FileSeqSource
✅ MergedSource (7 subtests)
✅ StdinSource
✅ WalkSource (3 subtests - DEPRECATED but tested)
```

### Coverage Gaps

**Missing Unit Tests:**
1. ❌ **Document.Close()** - No test for resource cleanup
2. ❌ **Pipeline cancellation** - Context cancel mid-processing
3. ❌ **Pipeline resource cleanup** - Source/Processor/Sink Close() calls
4. ❌ **Error propagation** - Ensure errors include context
5. ❌ **Large file handling** - Test with >100MB files
6. ❌ **Concurrent access** - Multiple pipelines using same source
7. ❌ **Memory leaks** - Test with many small documents
8. ❌ **S3 integration** - Real S3 tests (currently only local)

**Missing Integration Tests:**
1. ❌ **AgentProcessor with real LLM** - Deferred to Phase 5
2. ❌ **BlueprintProcessor with workflows** - Deferred to Phase 5
3. ❌ **SpoolPipeline with S3** - Only tested with local filesystem
4. ❌ **End-to-end workflows** - Source → Processor → Sink
5. ❌ **Error recovery** - What happens when disk full, network error, etc.
6. ❌ **Performance benchmarks** - Throughput, latency, memory

**Testing Improvements Needed:**
```go
// Add benchmark tests
func BenchmarkPipeline_SmallFiles(b *testing.B)
func BenchmarkPipeline_LargeFiles(b *testing.B)
func BenchmarkChunker_Various(b *testing.B)

// Add stress tests
func TestPipeline_ManySmallFiles(t *testing.T) // 10,000 files
func TestPipeline_LargeFile(t *testing.T)      // 1GB file
func TestPipeline_Cancellation(t *testing.T)   // Cancel mid-run

// Add error injection tests
func TestPipeline_DiskFull(t *testing.T)
func TestPipeline_NetworkError(t *testing.T)
func TestPipeline_LLMTimeout(t *testing.T)
```

---

## Critical Design Decisions

### 1. Dual Pipeline Architecture ✅

**Decision:** Generic `Pipeline` for most cases, `SpoolPipeline` for transactional directory processing.

**Rationale:**
- Generic Pipeline: Simple, flexible, works for stdin/stdout/files
- SpoolPipeline: Necessary for transactional semantics with `spool.ForEach`

**Trade-offs:**
- ✅ Correct transactional behavior
- ✅ Simple mental model (one exception case)
- ⚠️ Two implementations to maintain
- ⚠️ Confusion about when to use which

**Validation Needed:**
- [ ] Are there other cases needing transactions? (Database writes, multi-file operations)
- [ ] Can we unify the interfaces more? (Both have similar config/stats)
- [ ] Should we create a `TransactionalSink` interface instead?

### 2. Document Uses io.Reader (not []byte) ✅

**Decision:** Document.Content is `io.Reader`, not `[]byte`.

**Rationale:**
- Streaming: Don't load entire files into memory
- Zero-copy: Pass readers between pipeline stages
- Scalability: Handle gigabyte-sized files

**Trade-offs:**
- ✅ Memory efficient
- ✅ Streaming processing
- ⚠️ Can't rewind/replay content
- ⚠️ Must read sequentially
- ⚠️ Error handling harder (read errors vs processing errors)

**Validation Needed:**
- [ ] Do any processors need to read content multiple times? (Add buffering if yes)
- [ ] Should we add `Document.Bytes()` helper for small documents?
- [ ] How do we handle read errors in processors?

### 3. AllowEmptyOutput Flag ✅

**Decision:** `SpoolPipeline.AllowEmptyOutput` permits processors to produce no output.

**Rationale:**
- User request: Some LLM agents filter/validate without output
- Flexibility: Not all processors generate output

**Trade-offs:**
- ✅ Supports filtering use cases
- ✅ User-requested feature
- ⚠️ Unclear semantics (is empty output success or failure?)
- ⚠️ Only on SpoolPipeline, not generic Pipeline

**Validation Needed:**
- [ ] Should generic Pipeline also support this?
- [ ] Should we have explicit "skip this document" signal?
- [ ] How do we distinguish "no output" from "forgot to write"?

### 4. Metadata as map[string]any ⚠️

**Decision:** Document.Metadata is `map[string]any`.

**Rationale:**
- Flexibility: Can store any metadata
- No schema: Easy to add new fields

**Trade-offs:**
- ✅ Very flexible
- ⚠️ No type safety - requires type assertions everywhere
- ⚠️ No discovery - can't enumerate available metadata
- ⚠️ Easy to misspell keys
- ⚠️ Hard to validate

**Alternative Approaches:**
```go
// Option 1: Type-safe metadata struct
type DocumentMetadata struct {
    SourcePath string
    Size       int64
    MimeType   string
    CreatedAt  time.Time
    Custom     map[string]any // For extensions
}

// Option 2: Typed getters
type Metadata map[string]any
func (m Metadata) String(key string) (string, error)
func (m Metadata) Int64(key string) (int64, error)
func (m Metadata) Time(key string) (time.Time, error)

// Option 3: Registry pattern
var metadataRegistry = map[string]reflect.Type{
    "size": reflect.TypeOf(int64(0)),
    "created_at": reflect.TypeOf(time.Time{}),
}
```

**Recommendation:** Add Option 2 (typed getters) for safety while keeping flexibility.

### 5. Processor Returns []*Document (not single) ✅

**Decision:** `Processor.Process()` returns slice of documents, not single document.

**Rationale:**
- Chunking: One document → many chunks
- Filtering: Zero documents output
- Expansion: One input → multiple outputs (e.g., extract images from PDF)

**Trade-offs:**
- ✅ Supports 1→N transformations
- ✅ Enables filtering (return empty slice)
- ⚠️ More complex error handling
- ⚠️ Memory allocation for slice

**Validation Needed:**
- [ ] Do we need streaming processors that yield documents incrementally?
- [ ] Should we have separate `FilterProcessor` interface?
- [ ] How do we handle partial failures (some docs succeed, some fail)?

---

## Issues and Improvements

### High Priority Issues

#### 1. Resource Leaks 🔴

**Problem:** Documents hold `io.Reader` but no Close() method.

**Impact:** File handles, network connections may leak.

**Example:**
```go
// Current: No way to close document
doc, _ := source.Next(ctx)
// If processing fails, doc.Content reader never closed

// Proposed:
type Document struct {
    Path    string
    Content io.ReadCloser  // Change to ReadCloser
    ...
}

// Usage:
doc, _ := source.Next(ctx)
defer doc.Close()  // Always close
```

**Fix Required:** Yes - Change Content to `io.ReadCloser`, update all sources.

#### 2. Error Context Missing 🔴

**Problem:** Errors don't include document path or processing stage.

**Impact:** Hard to debug which file failed and why.

**Example:**
```go
// Current error:
return fmt.Errorf("processing failed: %w", err)

// Improved error:
return fmt.Errorf("processing document %q at stage %q: %w", 
    doc.Path, processorName, err)

// Or use structured errors:
type ProcessingError struct {
    DocumentPath string
    Stage        string
    Processor    string
    Err          error
}
```

**Fix Required:** Yes - Add error wrapping with context.

#### 3. No Progress for Large Files 🟡

**Problem:** Pipeline only reports progress per-document, not within-document.

**Impact:** Processing a 1GB file shows no progress until complete.

**Example:**
```go
// Current: Progress only when document complete
pipeline.Run(ctx) // No updates for 10 minutes

// Improved: Progress during reading
type Document struct {
    Content io.ReadCloser
    Size    int64  // Known size
}

// Pipeline tracks bytes read, reports percentage
```

**Fix Required:** Maybe - Add byte-level progress if users need it.

#### 4. SpoolPipeline Special Case 🟡

**Problem:** `SpoolPipeline` is completely separate from generic `Pipeline`.

**Impact:** Two implementations, duplication, confusion.

**Alternatives:**
```go
// Option 1: Make Sink transactional
type TransactionalSink interface {
    Sink
    Begin(ctx context.Context) (Transaction, error)
}

type Transaction interface {
    Write(ctx context.Context, doc *Document) error
    Commit() error
    Rollback() error
}

// Option 2: Make Pipeline support transactions
type Pipeline struct {
    transactional bool  // If true, wrap each doc in transaction
}

// Option 3: Keep separate (current approach)
// Simplest, but most duplication
```

**Recommendation:** Keep separate for now, revisit if more transactional cases emerge.

#### 5. Metadata Type Safety 🟡

**Problem:** `map[string]any` requires type assertions everywhere.

**Impact:** Runtime panics, no compile-time checking.

**Fix:** Add typed helper methods (see decision #4 above).

### Medium Priority Issues

#### 6. No Retry Logic

**Problem:** Any failure is immediate error, no retries.

**Impact:** Transient network/LLM errors fail entire pipeline.

**Solution:** Add retry configuration to processors and sinks.

#### 7. No Batch Operations

**Problem:** Sinks write one document at a time.

**Impact:** Could be inefficient for high-throughput scenarios (1000s of small files).

**Solution:** Add optional `BatchSink` interface.

#### 8. Limited Metrics

**Problem:** Only basic stats (count, bytes). No timing breakdown, throughput, etc.

**Impact:** Hard to identify bottlenecks.

**Solution:** Enhanced `PipelineStats` (see analysis above).

#### 9. No Compression Support

**Problem:** Can't automatically compress output (e.g., write .gz files).

**Impact:** Wastes storage for text-heavy outputs.

**Solution:** Add compression option to `FSSink`.

#### 10. MergedSource Not Streaming

**Problem:** Loads all sources into memory before merging.

**Impact:** Can't merge large files efficiently.

**Solution:** Use `io.MultiReader` for streaming merge (see analysis above).

### Low Priority Issues

#### 11. No Schema Validation

Processors can't validate input/output schemas.

#### 12. No Discovery API

Can't enumerate available sources/processors/sinks at runtime.

#### 13. No Plugin System

Can't load custom processors from external packages.

#### 14. No Rate Limiting

HTTP sources/sinks have no rate limiting.

#### 15. No Caching

Repeated processing of same input isn't cached.

---

## Phase 5 Readiness

### What's Complete ✅

1. **Core interfaces** - Source, Processor, Sink, Pipeline
2. **Basic implementations** - stdin, stdout, file, chunker
3. **Filesystem abstraction** - FSSink with stream library
4. **Transactional processing** - SpoolPipeline
5. **Blueprint integration** - AgentProcessor, BlueprintProcessor
6. **Error handling** - FailFast, SkipError modes
7. **Progress tracking** - Callbacks and stats
8. **Comprehensive tests** - 47 passing tests

### What's Needed Before Phase 5 🔧

**Must Fix (Blockers):**
1. ❌ **Resource leaks** - Add Close() to Document (#1 above)
2. ❌ **Error context** - Add document path to errors (#2 above)
3. ❌ **Integration tests** - Test AgentProcessor and BlueprintProcessor with real commands

**Should Fix (Important):**
4. ⚠️ **Metadata helpers** - Add typed getters for safety (#5 above)
5. ⚠️ **MergedSource streaming** - Use io.MultiReader (#10 above)
6. ⚠️ **Better docs** - Document when to use SpoolPipeline vs Pipeline

**Nice to Have (Not Blockers):**
7. ✨ **Enhanced stats** - Timing breakdown, throughput (#8 above)
8. ✨ **Retry logic** - For transient errors (#6 above)
9. ✨ **Compression** - FSSink auto-compression (#9 above)

### Phase 5 Migration Risks

**Commands to Migrate:**
- `cmd/tell.go` - Simple stdin/stdout, should be easy
- `cmd/ask.go` - Uses spool, needs SpoolPipeline
- `cmd/run.go` - Uses spool, needs SpoolPipeline
- `cmd/task.go` - Uses blueprint, needs BlueprintProcessor

**Risk Areas:**
1. **Backward compatibility** - Must preserve all CLI behavior
2. **Chunking** - Complex logic in current reader, must preserve
3. **Mutable mode** - Critical for queue pattern, must work exactly as before
4. **Error handling** - Current commands have specific error handling
5. **Progress output** - Users expect certain progress messages

**Migration Strategy:**
1. Start with `tell.go` (simplest, stdin→stdout)
2. Then `task.go` (blueprint integration)
3. Then `ask.go` (SpoolPipeline, no chunking)
4. Finally `run.go` (SpoolPipeline + chunking, most complex)
5. Keep old code temporarily, A/B test with flag
6. Remove old code once validated

---

## Recommendations

### Before Phase 5

**Critical Fixes (Do Now):**

1. **Add Document.Close()**
   ```go
   type Document struct {
       Path    string
       Content io.ReadCloser  // Change from io.Reader
       ...
   }
   ```

2. **Add Error Context**
   ```go
   type ProcessingError struct {
       DocumentPath string
       Stage        string
       Cause        error
   }
   ```

3. **Add Metadata Helpers**
   ```go
   func (m Metadata) GetString(key string) (string, error)
   func (m Metadata) GetInt64(key string) (int64, error)
   ```

4. **Fix MergedSource Streaming**
   Use `io.MultiReader` instead of buffering.

5. **Write Migration Guide**
   Document for Phase 5: when to use SpoolPipeline vs Pipeline.

**Documentation Needed:**

1. Architecture decision record (why dual pipelines)
2. When to use SpoolPipeline vs Pipeline (with flowchart)
3. How to write custom processors (with examples)
4. Error handling best practices
5. Testing guide (unit vs integration)

### During Phase 5

**Testing Strategy:**

1. **Parallel implementation** - Keep old code, add new alongside
2. **Feature flag** - `--use-new-io` to test new system
3. **Comparison tests** - Run both, compare outputs
4. **Gradual rollout** - One command at a time
5. **Fallback plan** - Keep old code until fully validated

**Metrics to Track:**

1. Performance: Old vs new throughput
2. Memory: Peak memory usage
3. Errors: Failure rates and types
4. Compatibility: Any behavior changes

---

## Conclusion

**Overall Assessment: 🟢 Ready for Phase 5 with minor fixes**

**Strengths:**
- ✅ Solid architecture foundation
- ✅ Solves transactional semantics problem
- ✅ Clean interface design
- ✅ Good test coverage
- ✅ Blueprint integration working

**Weaknesses:**
- ⚠️ Resource leak potential (no Document.Close)
- ⚠️ Missing error context
- ⚠️ Metadata type safety concerns
- ⚠️ SpoolPipeline special case complexity

**Recommendation:**
1. **Fix critical issues** (#1, #2, #3, #4 above) - ~1-2 days
2. **Write migration guide** - ~1 day
3. **Begin Phase 5 with tell.go** - ~2-3 days
4. **Validate before proceeding** - ~1 day
5. **Continue with remaining commands** - ~1-2 weeks

**Total Estimated Time to Phase 5 Completion:** 2-3 weeks

---

## Next Steps

1. **Review this document** with team
2. **Prioritize fixes** (which to do before Phase 5)
3. **Update IO_SYSTEM_DESIGN.md** with decisions
4. **Create Phase 5 detailed plan**
5. **Begin implementation**

---

**Questions for Discussion:**

1. Should we fix all critical issues before Phase 5, or fix incrementally?
2. Do we need more integration tests before migrating commands?
3. Should we keep old code during migration (feature flag approach)?
4. What's acceptable performance regression (if any)?
5. Do we need external review before Phase 5?
