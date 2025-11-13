# 0002 - IO System and Blueprint Integration

## In the context of

Building a universal I/O abstraction system for processing documents through LLM-powered workflows. The tool must handle diverse input sources (stdin, files, directories, S3) and output destinations while supporting chunking strategies (sentence, paragraph, semantic chunks) for large files. Must integrate seamlessly with blueprint workflows and provide transactional semantics for directory-based batch processing.

## Facing Concern

**Unified abstraction complexity**: Need single interface covering stdin (streaming), single files, multiple files, directory trees, and S3 buckets—each with different access patterns, error modes, and performance characteristics.

**Transactional semantics for batch processing**: When processing directory queues with mutable mode (removing input after success), must guarantee input files removed only after output successfully written. Original `WalkSource` approach broke `spool.ForEach` transactions by faking output filesystem, causing premature input deletion.

**Chunking integration**: Large files must be split using existing `github.com/fogfish/scanner` strategies (sentence/paragraph/chunk) while maintaining document context and enabling result aggregation.

**Blueprint workflow integration**: Must support both individual agents (`blueprint/compiler.Agent`) and complete workflows (`Blueprint`) as processors, retiring duplicate abstraction in `internal/service/worker.go`.

**Filesystem abstraction choice**: Need unified local/S3 access without implementing separate sources/sinks. Must leverage existing `github.com/fogfish/stream` library with `fs.FS` interface.

## We decided for

**Source → Processor → Sink pipeline architecture** with three core interfaces:

1. **Source interface**: Iterator pattern with `Next(ctx) (*Document, error)` returning `io.EOF` when exhausted. Implementations: `StdinSource`, `FileSource`, `FSSource`, `ReaderSource` (for spool integration). Document carries path, `io.Reader`, content type, and metadata map.

2. **Processor interface**: Transformation with `Process(ctx, doc) ([]*Document, error)` supporting 1:N splitting (chunking), 1:1 transformation (agents), and 1:0 filtering. Implementations: `IdentityProcessor`, `ChunkerProcessor` (integrates `github.com/fogfish/scanner`), `AgentProcessor` (wraps `blueprint/compiler.Agent`), `BlueprintProcessor` (executes workflow jobs).

3. **Sink interface**: Consumer with `Write(ctx, doc) error` and `Close() error` for finalization. Implementations: `StdoutSink`, `FileSink`, `FSSink` (directory writes using `github.com/fogfish/stream`), `WriterSink` (wraps `io.Writer` for spool).

**Conduit orchestrator**: Reusable pipeline created with `New(config)`, accumulates processors via `AddProcessor(proc)`, executes with `Run(ctx, source, sink)` returning `*Stats`. Pipeline takes source/sink at runtime (not construction), enabling reuse across multiple input/output pairs.

**Spool integration pattern**: For transactional directory processing, use `spool.ForEach` with `ReaderSource`/`WriterSink` wrappers. Pipeline runs inside ForEach callback, ensuring transaction completes before input removed. Pattern:
```go
spool.ForEach(ctx, "/", func(ctx, path, r, w) error {
    src := source.NewReaderSource(path, r)
    snk := sink.NewWriterSink(w)
    _, err := pipeline.Run(ctx, src, snk)
    return err
})
```

**Blueprint as processors**: `AgentProcessor` wraps `compiler.Agent`, `BlueprintProcessor` wraps entire `Blueprint` executing specific job. Retired `internal/service/worker.go` in favor of blueprint agents as primary abstraction.

**Filesystem via github.com/fogfish/stream**: All filesystem operations use `stream.NewFS` (handles S3 with `s3://` prefix) or `lfs.New` (local wrapper). Single `FSSink` implementation works for both local and S3 by detecting path prefix.

**Chunking via scanner library**: `ChunkerProcessor` integrates `github.com/fogfish/scanner` with strategies: `none` (pass-through), `sentence` (split by punctuation), `paragraph` (split by double newline), `chunk` (semantic chunks with configurable size). Each chunk becomes separate document with path suffix (`file.txt#chunk1`).

## Neglected

**Alternative architectures rejected**:

- **Separate WalkSource for directories**: Original design attempted directory source that walked filesystem. Abandoned because breaks spool transactions—cannot guarantee input deletion timing when source controls file discovery. Spool must control iteration.

- **SpoolPipeline as primary abstraction**: Created specialized pipeline for spool transactions. Marked deprecated after refactoring Pipeline to accept source/sink at runtime. Removed ~200 lines of duplicate code. Pattern now handled by ReaderSource/WriterSink adapters.

- **Service.Worker as agent abstraction**: Duplicate abstraction for running LLM agents. Retired in favor of `blueprint/compiler.Agent` as single agent representation. Reduces maintenance burden, unifies architecture around blueprints.

- **Custom filesystem implementations**: Could implement separate `DirSource`, `S3Source`, `DirSink`, `S3Sink`. Rejected because `github.com/fogfish/stream` already provides unified `fs.FS` interface with transparent local/S3 switching. Would duplicate tested code.

- **Push-based processing**: Could use channels and goroutines for parallelism. Rejected in favor of simple iterator pattern. Concurrency added later without interface changes—`Conduit` has placeholder for `runConcurrent`.

- **Complex chunking strategies**: Could add ML-based semantic chunking, sliding windows, overlapping chunks. Rejected because existing scanner library provides proven strategies matching current behavior. Extensible via custom processors.

**Alternative spool integration patterns rejected**:

- **FakeFS pattern**: Attempted to provide fake output filesystem to spool while buffering writes. Complex, breaks transaction semantics, unreliable.

- **Buffer-then-write pattern**: Buffer entire output in memory, write after success. Rejected because defeats streaming design, fails for large outputs, high memory usage.

## To Achieve

**Reusability**: Conduit created once with processors, runs multiple times with different source/sink pairs. Example: same chunker+agent pipeline processes stdin, files, or directories without rebuilding.

**Composability**: Processors chain naturally—chunker splits one document to many, agent processes each chunk, sink aggregates results. Interface allows custom processors without framework changes.

**Transactional batch processing**: Spool integration ensures fault tolerance—input removed only after output committed. Supports resume after failures (mutable mode with queue semantics).

**Streaming efficiency**: All components use `io.Reader`/`io.Writer`, never load entire files into memory. Handles multi-GB files with constant memory footprint. Zero-copy path when processors don't buffer.

**Unified filesystem access**: Single code path for local and S3 via stream library. `FSSink` detects `s3://` prefix, creates appropriate filesystem. No S3-specific sources/sinks needed.

**Blueprint integration**: Workflows as processors enables composition—chain multiple blueprint jobs, mix agents from different workflows, use same pipeline for simple prompts or complex routing.

**Clear error handling**: Two modes—`FailFast` stops on first error, `SkipError` continues collecting errors. Progress callbacks provide per-document status. Stats struct tracks successes, skips, errors.

**Testability**: Small interfaces enable focused unit tests. Mock sources/sinks for processor testing. Integration tests use `lfs.NewTempFS` for filesystem operations without I/O.

**Observability**: `Stats` tracks documents processed/skipped, bytes read/written, error list, timing. Optional `Progress` callback per document, `Metrics` callback per pipeline run.

## Accepting

**Refactored pipeline API**: Changed from `NewPipeline(source, sink, config)` to `New(config)` + `Run(ctx, source, sink)`. Required updating all call sites, but gains reusability. Marked `SpoolPipeline` deprecated, will remove in Phase 5.

**Spool integration indirection**: Requires `ReaderSource`/`WriterSink` adapter wrappers instead of direct pipeline usage. Two extra object allocations per file, but necessary for transaction semantics. Alternative (SpoolPipeline) required ~200 lines of duplicate code.

**No WalkSource**: Cannot provide directory source that integrates cleanly with spool transactions. Users must call `spool.ForEach` directly with reader/writer adapters. More verbose but explicit about transactional boundaries.

**Single-threaded processing**: `Conduit.Run` processes sequentially despite `Concurrency` config parameter. Parallel processing deferred to future work. Sufficient for current CLI use cases, framework supports concurrency without breaking changes.

**Scanner dependency for chunking**: Ties chunking behavior to `github.com/fogfish/scanner` library. Cannot easily swap chunking implementations without processor replacement. Acceptable because scanner proven in production, matches existing behavior.

**Document path conventions**: Chunks use path suffixes (`file.txt#chunk1`). No standardized separator—`#` chosen ad-hoc. Could conflict with actual filenames. Acceptable risk because chunks are ephemeral within pipeline.

**Stream library S3 prefix convention**: Detection via `s3://` string prefix is fragile—string manipulation vulnerable to edge cases. Alternative (explicit filesystem type parameter) rejected as more verbose. Stream library handles edge cases, proven reliable.

**No output aggregation**: Chunked documents written separately to sink. Downstream processors must handle chunk sequences. Missing built-in aggregator processor for combining chunk outputs. Deferred because aggregation strategy varies by use case (concatenate, merge, deduplicate).

