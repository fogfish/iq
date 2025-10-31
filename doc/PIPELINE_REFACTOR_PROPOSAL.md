# Pipeline Refactor Proposal: Reusable Pipeline Pattern

## Problem Statement

Currently we have two separate pipeline implementations:
1. **Generic Pipeline** - Takes source/sink at construction
2. **SpoolPipeline** - Separate implementation for transactional directory processing

This creates:
- ❌ Code duplication
- ❌ Maintenance burden (two implementations to keep in sync)
- ❌ Inconsistent APIs
- ❌ Cannot reuse Pipeline for spool processing

## Proposed Solution

**Make Pipeline reusable by passing source/sink to `Run()` instead of constructor.**

### Current API (Before)

```go
// Pipeline is tied to specific source/sink at construction
pipeline := iosystem.NewPipeline(source, sink, config)
pipeline.AddProcessor(chunker)
pipeline.AddProcessor(agent)
pipeline.Run(ctx) // Uses source/sink from construction
```

### Proposed API (After)

```go
// Pipeline is a reusable processor chain
pipeline := iosystem.NewPipeline(config)
pipeline.AddProcessor(chunker)
pipeline.AddProcessor(agent)

// Run with different sources/sinks
pipeline.Run(ctx, source1, sink1) // First use
pipeline.Run(ctx, source2, sink2) // Reuse with different I/O
```

## Benefits

### 1. Single Pipeline Implementation ✅

```go
// Generic streaming use case
source := iosystem.NewStdinSource()
sink := iosystem.NewStdoutSink()
pipeline.Run(ctx, source, sink)

// Spool transactional use case - same pipeline!
spool.ForEach(ctx, root,
    func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
        // Create ephemeral source/sink from spool's r/w
        source := iosystem.NewReaderSource(path, r)
        sink := iosystem.NewWriterSink(w)
        
        // Reuse the same pipeline
        return pipeline.Run(ctx, source, sink)
    })
```

### 2. Pipeline Reusability ✅

```go
// Build pipeline once
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(chunker)
pipeline.AddProcessor(agent)

// Use with different inputs
for _, file := range files {
    source := iosystem.NewFileSource(file)
    sink := iosystem.NewFSSink(outputDir)
    pipeline.Run(ctx, source, sink)
}
```

### 3. Better Testability ✅

```go
// Test pipeline with mock sources/sinks
func TestPipeline(t *testing.T) {
    pipeline := iosystem.NewPipeline(nil)
    pipeline.AddProcessor(testProcessor)
    
    // Test with different inputs
    pipeline.Run(ctx, mockSource1, mockSink1)
    pipeline.Run(ctx, mockSource2, mockSink2)
}
```

### 4. Eliminate SpoolPipeline ✅

No need for separate SpoolPipeline - use regular Pipeline with ReaderSource/WriterSink.

## Implementation Details

### New Sources for Spool Integration

```go
// ReaderSource wraps an io.Reader as a single-document Source
type ReaderSource struct {
    path     string
    reader   io.Reader
    consumed bool
}

func NewReaderSource(path string, r io.Reader) *ReaderSource {
    return &ReaderSource{
        path:   path,
        reader: r,
    }
}

func (s *ReaderSource) Next(ctx context.Context) (*Document, error) {
    if s.consumed {
        return nil, io.EOF
    }
    s.consumed = true
    return NewDocument(s.path, s.reader), nil
}

func (s *ReaderSource) Close() error {
    // io.Reader doesn't have Close - spool manages lifecycle
    return nil
}
```

### New Sinks for Spool Integration

```go
// WriterSink wraps an io.Writer as a Sink
type WriterSink struct {
    writer io.Writer
}

func NewWriterSink(w io.Writer) *WriterSink {
    return &WriterSink{writer: w}
}

func (s *WriterSink) Write(ctx context.Context, doc *Document) error {
    _, err := io.Copy(s.writer, doc.Reader)
    return err
}

func (s *WriterSink) Close() error {
    // io.Writer doesn't have Close - spool manages lifecycle
    return nil
}
```

### Refactored Pipeline API

```go
// Pipeline coordinates document processing through processor stages.
// It is reusable - the same pipeline can process multiple source/sink pairs.
type Pipeline struct {
    processors []Processor
    config     PipelineConfig
}

// NewPipeline creates a reusable pipeline with the given configuration.
func NewPipeline(config *PipelineConfig) *Pipeline {
    if config == nil {
        config = &PipelineConfig{
            ErrorMode: FailFast,
        }
    }
    return &Pipeline{
        processors: make([]Processor, 0),
        config:     *config,
    }
}

// AddProcessor adds a processing stage to the pipeline.
func (p *Pipeline) AddProcessor(proc Processor) *Pipeline {
    p.processors = append(p.processors, proc)
    return p
}

// Run executes the pipeline with the given source and sink.
// The pipeline can be run multiple times with different sources/sinks.
func (p *Pipeline) Run(ctx context.Context, source Source, sink Sink) (*PipelineStats, error) {
    if source == nil {
        return nil, fmt.Errorf("source cannot be nil")
    }
    if sink == nil {
        return nil, fmt.Errorf("sink cannot be nil")
    }
    
    stats := &PipelineStats{StartTime: time.Now()}
    defer func() {
        stats.Duration = time.Since(stats.StartTime)
        if p.config.Metrics != nil {
            p.config.Metrics(*stats)
        }
    }()
    
    // Ensure cleanup
    defer source.Close()
    defer sink.Close()
    
    // Process all documents
    for {
        select {
        case <-ctx.Done():
            return stats, ctx.Err()
        default:
        }
        
        doc, err := source.Next(ctx)
        if err == io.EOF {
            return stats, nil // Success
        }
        if err != nil {
            stats.Errors = append(stats.Errors, err)
            if p.config.ErrorMode == FailFast {
                return stats, err
            }
            continue
        }
        
        if err := p.processDocument(ctx, doc, sink, stats); err != nil {
            stats.Errors = append(stats.Errors, err)
            if p.config.Progress != nil {
                p.config.Progress(doc, err)
            }
            if p.config.ErrorMode == FailFast {
                return stats, err
            }
            stats.DocsSkipped++
            continue
        }
        
        stats.DocsProcessed++
        if p.config.Progress != nil {
            p.config.Progress(doc, nil)
        }
    }
}

// processDocument runs a single document through all processors and to the sink.
func (p *Pipeline) processDocument(ctx context.Context, doc *Document, sink Sink, stats *PipelineStats) error {
    docs := []*Document{doc}
    
    // Apply processors in sequence
    for _, processor := range p.processors {
        var nextDocs []*Document
        for _, d := range docs {
            processed, err := processor.Process(ctx, d)
            if err != nil {
                return fmt.Errorf("processor error: %w", err)
            }
            nextDocs = append(nextDocs, processed...)
        }
        docs = nextDocs
        
        if len(docs) == 0 {
            return nil // Filtered out
        }
    }
    
    // Write all results to sink
    for _, d := range docs {
        if err := sink.Write(ctx, d); err != nil {
            return fmt.Errorf("sink error: %w", err)
        }
    }
    
    return nil
}

// Close closes all processors (but NOT source/sink - those are managed by caller).
func (p *Pipeline) Close() error {
    var errs []error
    for i, proc := range p.processors {
        if err := proc.Close(); err != nil {
            errs = append(errs, fmt.Errorf("processor %d: %w", i, err))
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("errors closing processors: %v", errs)
    }
    return nil
}
```

## Usage Examples

### Example 1: Simple Stdin/Stdout (tell command)

```go
// cmd/tell.go
pipeline := iosystem.NewPipeline(&iosystem.PipelineConfig{
    ErrorMode: iosystem.FailFast,
    Progress: func(doc *Document, err error) {
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        }
    },
})

// Add processors
agent := compiler.NewAgent(promptFile, factory)
pipeline.AddProcessor(iosystem.NewAgentProcessor(agent))

// Run once with stdin/stdout
source := iosystem.NewStdinSource()
sink := iosystem.NewStdoutSink()
stats, err := pipeline.Run(ctx, source, sink)
```

### Example 2: Directory Processing with Spool (ask command)

```go
// cmd/ask.go
// Build pipeline once
pipeline := iosystem.NewPipeline(&iosystem.PipelineConfig{
    ErrorMode: iosystem.SkipError, // Continue on errors
})

agent := compiler.NewAgent(promptFile, factory)
pipeline.AddProcessor(iosystem.NewAgentProcessor(agent))

// Mount filesystems
inFS, _ := mountFS(inputDir)
outFS, _ := mountFS(outputDir)

// Configure spool
spoolOpts := []opts.Option[spool.Spool]{spool.IsMutable, spool.WithSkipError}
sp := spool.New(inFS, outFS, spoolOpts...)

// Process all files transactionally
err := sp.ForEach(ctx, "/",
    func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
        // Create ephemeral source/sink from spool's r/w
        source := iosystem.NewReaderSource(path, r)
        sink := iosystem.NewWriterSink(w)
        
        // Run pipeline (reused for each file!)
        stats, err := pipeline.Run(ctx, source, sink)
        
        // Check if output was written (for AllowEmptyOutput logic)
        if stats.BytesWritten == 0 && !allowEmptyOutput {
            return fmt.Errorf("no output produced for %s", path)
        }
        
        return err
    })
```

### Example 3: Batch Processing Multiple Files (run command)

```go
// cmd/run.go
// Build pipeline once
pipeline := iosystem.NewPipeline(&iosystem.PipelineConfig{
    ErrorMode: iosystem.SkipError,
})

// Add chunking + agent processing
pipeline.
    AddProcessor(iosystem.NewChunkerProcessor("chunk", iosystem.WithChunkSize(2048))).
    AddProcessor(iosystem.NewAgentProcessor(agent))

// Process through spool (same as ask example)
inFS, _ := mountFS(inputDir)
outFS, _ := mountFS(outputDir)
sp := spool.New(inFS, outFS, spool.IsMutable)

err := sp.ForEach(ctx, "/",
    func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
        source := iosystem.NewReaderSource(path, r)
        sink := iosystem.NewWriterSink(w)
        _, err := pipeline.Run(ctx, source, sink)
        return err
    })
```

### Example 4: Task/Blueprint Processing (task command)

```go
// cmd/task.go
pipeline := iosystem.NewPipeline(&iosystem.PipelineConfig{
    ErrorMode: iosystem.FailFast,
})

// Execute blueprint workflow
bp, _ := blueprint.New(ctx, workflowFile, factory)
pipeline.AddProcessor(iosystem.NewBlueprintProcessor(bp, "main"))

// Run with stdin/stdout
source := iosystem.NewStdinSource()
sink := iosystem.NewStdoutSink()
stats, err := pipeline.Run(ctx, source, sink)
```

## Migration Path

### Phase 1: Add New Sources/Sinks
1. Implement `ReaderSource` and `WriterSink`
2. Add tests for new components
3. Keep existing Pipeline as-is

### Phase 2: Refactor Pipeline
1. Change `NewPipeline(source, sink, config)` → `NewPipeline(config)`
2. Change `Run(ctx)` → `Run(ctx, source, sink) (*PipelineStats, error)`
3. Update all Pipeline tests
4. Keep SpoolPipeline temporarily

### Phase 3: Migrate Commands
1. Update `cmd/tell.go` to new API
2. Update `cmd/task.go` to new API
3. Update `cmd/ask.go` to use Pipeline + ReaderSource/WriterSink
4. Update `cmd/run.go` to use Pipeline + ReaderSource/WriterSink
5. Integration tests for each command

### Phase 4: Remove SpoolPipeline
1. Delete `internal/iosystem/spool_pipeline.go`
2. Delete `internal/iosystem/spool_pipeline_test.go`
3. Update documentation

## Benefits Summary

| Aspect                   | Before                        | After                            |
| ------------------------ | ----------------------------- | -------------------------------- |
| Pipeline Implementations | 2 (Pipeline + SpoolPipeline)  | 1 (Pipeline)                     |
| Lines of Code            | ~550                          | ~350 (-36%)                      |
| Reusability              | Low (tied to source/sink)     | High (reuse with any I/O)        |
| Testability              | Moderate                      | High (mock sources/sinks easily) |
| Maintenance              | High (sync 2 implementations) | Low (single implementation)      |
| API Consistency          | Low (different patterns)      | High (uniform pattern)           |

## Open Questions

1. **Should Pipeline.Close() close processors?**
   - Current proposal: Yes (processors are owned by pipeline)
   - Source/Sink: No (owned by caller, closed in Run() defer)

2. **Should Run() return stats or modify internal stats?**
   - Current proposal: Return new stats (immutable, thread-safe)
   - Alternative: Accumulate in pipeline.stats (requires locking)

3. **What about AllowEmptyOutput flag?**
   - Current proposal: Caller checks `stats.BytesWritten` after Run()
   - Alternative: Add to PipelineConfig (but less flexible)

4. **Should we support Run() without source/sink (use stored ones)?**
   - No - simplicity over convenience
   - Can wrap in helper if needed: `pipeline.RunWith(source, sink)`

## Conclusion

**Recommendation: Proceed with refactor** ✅

This design:
- ✅ Eliminates code duplication (remove SpoolPipeline)
- ✅ Makes Pipeline truly reusable
- ✅ Simplifies maintenance (one implementation)
- ✅ Enables spool integration without special cases
- ✅ Improves testability
- ✅ Maintains all existing functionality

The migration is straightforward and can be done incrementally without breaking existing code until Phase 4.
