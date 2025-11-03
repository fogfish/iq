# Pipeline Quick Reference

## When to Use What

### Use Generic Pipeline When:
- ✅ Processing stdin to stdout
- ✅ Single file input/output
- ✅ HTTP source/sink
- ✅ Any non-directory I/O
- ✅ No transactional semantics needed

### Use Pipeline + Spool When:
- ✅ Processing directory of files
- ✅ Need transactional semantics (commit/rollback)
- ✅ Queue-based processing (mutable mode)
- ✅ Resume capability after failures
- ✅ S3 batch processing

## Quick Patterns

### Pattern 1: Stdin → Stdout
```go
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(myProcessor)

src := source.NewStdinSource()
snk := sink.NewStdoutSink()

stats, err := pipeline.Run(ctx, src, snk)
```

### Pattern 2: File → File
```go
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(myProcessor)

src, _ := source.NewFileSource("input.txt")
snk, _ := sink.NewFileSink("output.txt")

stats, err := pipeline.Run(ctx, src, snk)
```

### Pattern 3: Directory → Directory (Spool)
```go
// Build reusable pipeline
pipeline := iosystem.NewPipeline(&iosystem.PipelineConfig{
    ErrorMode: iosystem.SkipError,
})
pipeline.AddProcessor(myProcessor)

// Mount filesystems
inFS, _ := lfs.New("./input")
outFS, _ := lfs.New("./output")

// Create spool
sp := spool.New(inFS, outFS, spool.IsMutable)

// Process all files
sp.ForEach(ctx, "/", func(ctx, path, r, w) error {
    src := source.NewReaderSource(path, r)
    snk := sink.NewWriterSink(w)
    _, err := pipeline.Run(ctx, src, snk)
    return err
})
```

### Pattern 4: S3 → S3 (Spool)
```go
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(myProcessor)

// S3 paths automatically detected by stream library
inFS, _ := stream.NewFS("my-bucket", stream.WithPrefix("input/"))
outFS, _ := stream.NewFS("my-bucket", stream.WithPrefix("output/"))

sp := spool.New(inFS, outFS, spool.IsImmutable)

sp.ForEach(ctx, "/", func(ctx, path, r, w) error {
    src := source.NewReaderSource(path, r)
    snk := sink.NewWriterSink(w)
    _, err := pipeline.Run(ctx, src, snk)
    return err
})
```

### Pattern 5: With Chunking
```go
pipeline := iosystem.NewPipeline(nil)

// Add chunker first
chunker := processor.NewChunkerProcessor(processor.ChunkConfig{
    Strategy: processor.StrategySentence,
})
pipeline.AddProcessor(chunker)

// Then other processors
pipeline.AddProcessor(myProcessor)

// Use with any source/sink
stats, err := pipeline.Run(ctx, source, sink)
```

### Pattern 6: With Progress Tracking
```go
config := &iosystem.PipelineConfig{
    ErrorMode: iosystem.SkipError,
    Progress: func(doc *iosystem.Document, err error) {
        if err != nil {
            log.Printf("❌ %s: %v", doc.Path, err)
        } else {
            log.Printf("✅ %s", doc.Path)
        }
    },
}

pipeline := iosystem.NewPipeline(config)
pipeline.AddProcessor(myProcessor)

stats, err := pipeline.Run(ctx, source, sink)
```

### Pattern 7: Reuse Pipeline for Multiple Inputs
```go
// Build once
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(processor1)
pipeline.AddProcessor(processor2)

// Use many times
for _, inputFile := range files {
    src, _ := source.NewFileSource(inputFile)
    snk, _ := sink.NewFSSink(outputDir)
    
    stats, err := pipeline.Run(ctx, src, snk)
    if err != nil {
        log.Printf("Failed %s: %v", inputFile, err)
    }
}
```

## Configuration Options

### PipelineConfig
```go
type PipelineConfig struct {
    Concurrency int          // Number of parallel workers (1 = sequential)
    ErrorMode   ErrorMode    // FailFast or SkipError
    Progress    ProgressFunc // Called after each document
    Metrics     MetricsFunc  // Called with final stats
}
```

### ErrorMode
```go
iosystem.FailFast  // Stop on first error
iosystem.SkipError // Continue processing on errors
```

### Spool Options
```go
spool.IsMutable    // Remove input files after success (queue mode)
spool.IsImmutable  // Keep input files (read-only mode)
spool.WithStrict   // Fail on first error
spool.WithSkipError // Continue on errors
```

## Return Values

### PipelineStats
```go
type PipelineStats struct {
    DocsProcessed int           // Successfully processed documents
    DocsSkipped   int           // Skipped due to errors
    BytesRead     int64         // Total bytes read
    BytesWritten  int64         // Total bytes written
    Errors        []error       // All errors encountered
    Duration      time.Duration // Total processing time
    StartTime     time.Time     // When processing started
}
```

## Common Mistakes to Avoid

### ❌ Don't create source/sink inside spool callback
```go
// WRONG - wasteful
sp.ForEach(ctx, "/", func(ctx, path, r, w) error {
    pipeline := iosystem.NewPipeline(nil) // ❌ Creates new pipeline each time!
    pipeline.AddProcessor(proc)
    // ...
})
```

### ✅ Create pipeline once, reuse
```go
// CORRECT - efficient
pipeline := iosystem.NewPipeline(nil)
pipeline.AddProcessor(proc)

sp.ForEach(ctx, "/", func(ctx, path, r, w) error {
    src := source.NewReaderSource(path, r)
    snk := sink.NewWriterSink(w)
    _, err := pipeline.Run(ctx, src, snk) // ✅ Reuses pipeline
    return err
})
```

### ❌ Don't forget error handling
```go
// WRONG - ignores stats
_, err := pipeline.Run(ctx, src, snk)
if err != nil {
    return err
}
```

### ✅ Check stats for empty output
```go
// CORRECT - checks stats
stats, err := pipeline.Run(ctx, src, snk)
if err != nil {
    return err
}

if stats.BytesWritten == 0 && !allowEmptyOutput {
    return fmt.Errorf("no output produced")
}
```

## Performance Tips

1. **Reuse pipelines** - Create once, use many times
2. **Use SkipError mode** for batch processing
3. **Set appropriate concurrency** (future feature)
4. **Use streaming** - Don't buffer entire files in memory
5. **Close resources** - Pipeline closes source/sink automatically in Run()

## Testing Tips

```go
func TestMyProcessor(t *testing.T) {
    // Create mock source
    src := newMockSource("test content")
    
    // Create mock sink
    snk := newMockSink()
    
    // Create pipeline with processor
    pipeline := iosystem.NewPipeline(nil)
    pipeline.AddProcessor(myProcessor)
    
    // Run and assert
    stats, err := pipeline.Run(context.Background(), src, snk)
    
    it.Then(t).Should(
        it.Nil(err),
        it.Equal(stats.DocsProcessed, 1),
        it.Equal(snk.docs[0].Path, "expected.txt"),
    )
}
```

## Migration from SpoolPipeline

See `doc/PIPELINE_REFACTOR_PROPOSAL.md` for detailed migration guide.

**Quick version:**
1. Change `NewSpoolPipeline()` → `NewPipeline()`
2. Mount filesystems with `lfs.New()` or `stream.NewFS()`
3. Create spool with `spool.New()`
4. Wrap in `sp.ForEach()` with `ReaderSource`/`WriterSink`

Done! ✅
