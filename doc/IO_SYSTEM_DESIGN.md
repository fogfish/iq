# Universal Input/Output System Design for IQ

## Executive Summary

This document outlines the design for a universal, extensible I/O abstraction system for the `iq` project that unifies input sources (stdin, files, directories, S3, HTTP) and output destinations while maintaining the existing chunking capabilities and integrating seamlessly with the blueprint workflow system.

**Key Design Decisions:**
1. **Use `github.com/fogfish/stream` for all filesystem I/O** - Single unified abstraction for local and S3
2. **Retire `internal/service/worker.go`** - Use `blueprint/compiler.Agent` instead
3. **Blueprint agents as first-class processors** - Workflows are central to architecture

## Current State Analysis

### Existing Components

1. **Input Sources** (Currently in `cmd/`)
   - **stdin**: Direct piping (`echo "text" | iq tell`)
   - **File arguments**: Individual files as CLI args (`iq tell file1.txt file2.txt`)
   - **Mounted directories**: Local filesystem (`-d ./input`)
   - **S3 buckets**: AWS S3 paths (`-d s3://bucket/path`)
   - **Merge mode**: Combine multiple files into single input (`--merge`)

2. **Output Destinations** (Currently in `cmd/root.go`)
   - **stdout**: Direct output (`-o stdout`)
   - **Local filesystem**: Write to directory (`-o ./output`)
   - **S3 buckets**: Write to S3 (`-o s3://bucket/path`)

3. **Processing Patterns** (Currently in `internal/reader/`)
   - **Chunking strategies**:
     - `none`: Process entire file as single unit
     - `sentence`: Split by sentence boundaries
     - `paragraph`: Split by paragraph markers
     - `chunk`: Split into fixed-size semantic chunks
   - **Stream processing**: Iterator pattern with `scanner.Scanner`
   - **LLM integration**: Process each chunk through agent/prompter

4. **Spool System** (`github.com/fogfish/stream/spool`)
   - **Queue-based processing**: Treats directories as processing queues
   - **Mutable mode**: Remove input files after processing (resume capability)
   - **Strict mode**: Fail-fast vs skip-error behavior
   - **ForEach pattern**: Process each file with callback

5. **Stream Library** (`github.com/fogfish/stream`)
   - **Unified filesystem interface**: Implements `fs.FS`, `fs.StatFS`, `fs.ReadDirFS`, `fs.GlobFS`
   - **Writable extensions**: `stream.CreateFS`, `stream.RemoveFS`, `stream.CopyFS`
   - **Transparent S3/Local**: `stream.NewFS` handles both with `s3://` prefix
   - **Local wrapper**: `lfs.New` for local filesystem with same interface

### Key Patterns to Preserve

1. ✅ **Chunking must be maintained** - Critical for large file processing
2. ✅ **Stream processing** - Memory efficient, handles large files
3. ✅ **Mutable/resume capability** - Fault-tolerant batch processing
4. ✅ **github.com/fogfish/stream usage** - Unified filesystem abstraction
5. ✅ **Blueprint integration** - Workflows as processors

## Problems with Current Design

1. **Tight Coupling**: I/O logic scattered across `cmd/` files
2. **Code Duplication**: Similar patterns repeated in `tell.go`, `ask.go`, `run.go`, `task.go`
3. **Limited Extensibility**: Adding new sources (HTTP server) requires modifying multiple files
4. **No Blueprint Integration**: Workflows can't leverage I/O abstractions directly
5. **Inconsistent Interfaces**: Different commands handle I/O differently
6. **Testing Difficulty**: Hard to test I/O logic in isolation
7. **Duplicate Agent Abstractions**: `service.Worker` vs `blueprint/compiler.Agent`

## Design Principles

1. **Leverage github.com/fogfish/stream**: Use existing filesystem abstraction
2. **Composition over Inheritance**: Combine simple components for complex behavior
3. **Interface-Driven**: Plugin architecture for sources/sinks
4. **Backward Compatible**: Existing commands continue to work
5. **Blueprint First**: Design with workflow integration in mind
6. **Streaming by Default**: Handle unbounded data efficiently
7. **Zero-Copy Where Possible**: Minimize data copying for performance

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│   (cmd/tell.go, cmd/ask.go, blueprint workflows)            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    I/O System Layer                          │
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   │
│  │   Source     │   │  Processor   │   │     Sink     │   │
│  │  (Input)     │──▶│  (Transform) │──▶│   (Output)   │   │
│  └──────────────┘   └──────────────┘   └──────────────┘   │
│         │                   │                   │           │
│         │                   │                   │           │
│  ┌──────▼──────┐   ┌────────▼────────┐   ┌─────▼──────┐  │
│  │ - Stdin     │   │ - Chunker       │   │ - Stdout   │  │
│  │ - File      │   │ - Reader        │   │ - File     │  │
│  │ - FS (Dir)  │   │ - Merger        │   │ - FS (Dir) │  │
│  │ - HTTP      │   │ - Agent         │   │ - HTTP     │  │
│  │             │   │ - Blueprint     │   │            │  │
│  └─────────────┘   └─────────────────┘   └────────────┘  │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              github.com/fogfish/stream Layer                 │
│     (Unified FS abstraction for local/S3)                   │
└─────────────────────────────────────────────────────────────┘
```

## Core Abstractions

### 1. Source Interface (Input)

```go
package iosystem

import (
    "context"
    "io"
)

// Document represents a single input document with metadata
type Document struct {
    Path     string            // Logical path (e.g., "dir/file.txt", "stdin")
    Reader   io.Reader         // Content stream
    Metadata map[string]string // Additional metadata (size, type, etc.)
}

// Source produces documents from an input source
type Source interface {
    // Next returns the next document or io.EOF when complete
    Next(ctx context.Context) (*Document, error)
    
    // Close releases resources
    Close() error
}

// SourceFactory creates sources from configuration
type SourceFactory func(config map[string]any) (Source, error)
```

**Implementations:**

```go
// StdinSource - reads from standard input
type StdinSource struct {
    read bool
}

func NewStdinSource() Source

// FileSource - reads single file using stream or lfs
type FileSource struct {
    fsys fs.FS  // From stream.NewFS or lfs.New
    path string
    read bool
}

func NewFileSource(path string) (Source, error)
// Uses stream.NewFS or lfs.New based on path prefix

// FilesSource - reads multiple files using stream or lfs
type FilesSource struct {
    fsys  fs.FS  // From stream.NewFS or lfs.New
    paths []string
    index int
}

func NewFilesSource(paths ...string) (Source, error)
// Uses stream.NewFS or lfs.New based on path prefix

// WalkSource - walks filesystem tree (local or S3)
// This REPLACES separate DirSource and S3Source
type WalkSource struct {
    fsys    spool.FileSystem  // Interface - implemented by stream or lfs
    root    string
    walker  func(fs.FS, string, fs.WalkDirFunc) error
    mutable bool              // Delete files after reading (queue mode)
    walked  bool
    queue   []*Document
}

type WalkConfig struct {
    Mutable bool   // Delete files after reading (queue mode)
    Pattern string // Glob pattern filter (optional)
}

func NewWalkSource(path string, config WalkConfig) (Source, error)
// If path starts with "s3://", uses stream.NewFS(path[5:])
// Otherwise uses lfs.New(path)
// Supports mutable mode for resume capability

// HTTPSource - embedded HTTP server that receives uploads
type HTTPSource struct {
    addr   string
    queue  chan *Document
    server *http.Server
}

func NewHTTPSource(addr string) (Source, error)
// Starts HTTP server accepting POST /upload
// Returns documents as they're uploaded

// MergedSource - combines multiple sources into single document
type MergedSource struct {
    sources []Source
}

func NewMergedSource(sources ...Source) Source
// Concatenates all source content into single document
```

### 2. Processor Interface (Transformation)

```go
// Processor transforms documents in a pipeline
type Processor interface {
    // Process takes input document and produces zero or more output documents
    Process(ctx context.Context, doc *Document) ([]*Document, error)
    
    // Close releases resources
    Close() error
}

// ProcessorFactory creates processors from configuration
type ProcessorFactory func(config map[string]any) (Processor, error)
```

**Implementations:**

```go
// ChunkerProcessor - splits documents by strategy
type ChunkerProcessor struct {
    strategy       string // "none", "sentence", "paragraph", "chunk"
    chunkSize      int
    delimiterChars string
}

func NewChunkerProcessor(strategy string, opts ...ChunkOption) Processor
// Returns multiple documents, one per chunk
// Preserves original path with chunk suffix: "file.txt#chunk1"

// AgentProcessor - processes through blueprint Agent
// REPLACES service.Prompter - uses blueprint/compiler.Agent instead
type AgentProcessor struct {
    agent *compiler.Agent  // From blueprint/compiler package
}

func NewAgentProcessor(agent *compiler.Agent) Processor
// Processes document content through blueprint agent
// Returns single document with agent response

// BlueprintProcessor - processes through compiled workflow
type BlueprintProcessor struct {
    blueprint *blueprint.Blueprint
    jobName   string
}

func NewBlueprintProcessor(bp *blueprint.Blueprint, job string) Processor
// Executes workflow job with document content as input
// Supports full workflow state management

// IdentityProcessor - pass-through (no transformation)
type IdentityProcessor struct{}

func NewIdentityProcessor() Processor
```

### 3. Sink Interface (Output)

```go
// Sink consumes processed documents
type Sink interface {
    // Write stores a document
    Write(ctx context.Context, doc *Document) error
    
    // Close finalizes output and releases resources
    Close() error
}

// SinkFactory creates sinks from configuration
type SinkFactory func(config map[string]any) (Sink, error)
```

**Implementations:**

```go
// StdoutSink - writes to standard output
type StdoutSink struct{}

func NewStdoutSink() Sink

// FileSink - writes to single file using stream or lfs
type FileSink struct {
    fsys spool.FileSystem  // Interface - implemented by stream or lfs
    path string
    file stream.File
}

func NewFileSink(path string) (Sink, error)
// Uses stream.NewFS or lfs.New based on path prefix

// FSSink - writes to filesystem, one file per document (local or S3)
// This REPLACES separate DirSink and S3Sink
type FSSink struct {
    fsys spool.FileSystem  // Interface - implemented by stream or lfs
    root string
}

func NewFSSink(path string) (Sink, error)
// If path starts with "s3://", uses stream.NewFS(path[5:])
// Otherwise uses lfs.New(path)
// Preserves document path structure under root

// HTTPSink - sends documents via HTTP POST
type HTTPSink struct {
    endpoint string
    client   *http.Client
}

func NewHTTPSink(endpoint string) Sink
// POST each document to endpoint

// MultiSink - writes to multiple sinks
type MultiSink struct {
    sinks []Sink
}

func NewMultiSink(sinks ...Sink) Sink
```

### 4. Pipeline Orchestrator

```go
// Pipeline coordinates source → processor → sink flow
type Pipeline struct {
    source     Source
    processors []Processor
    sink       Sink
    options    PipelineOptions
}

type PipelineOptions struct {
    Concurrency int           // Parallel processing workers
    ErrorMode   ErrorMode     // FailFast or SkipError
    Progress    ProgressFunc  // Progress callback
    Metrics     MetricsFunc   // Metrics callback
}

type ErrorMode int

const (
    FailFast ErrorMode = iota
    SkipError
)

type ProgressFunc func(doc *Document, err error)
type MetricsFunc func(stats PipelineStats)

// NewPipeline creates a processing pipeline
func NewPipeline(source Source, sink Sink, opts ...PipelineOption) *Pipeline

// AddProcessor adds a processing stage
func (p *Pipeline) AddProcessor(proc Processor) *Pipeline

// Run executes the pipeline
func (p *Pipeline) Run(ctx context.Context) error

// Stats returns processing statistics
func (p *Pipeline) Stats() PipelineStats

type PipelineStats struct {
    DocsProcessed int
    DocsSkipped   int
    BytesRead     int64
    BytesWritten  int64
    Errors        []error
    Duration      time.Duration
}
```

## Integration with Blueprint

### Blueprint as Processor

Blueprints become first-class processors in the pipeline:

```go
// In cmd/task.go - new approach
bp, err := blueprint.New(context.Background(), rootPrompt, &factory{})
if err != nil {
    return err
}

// Get the compiled agent from blueprint
job, err := bp.GetJob("main")
if err != nil {
    return err
}

// Extract agent from first step (assuming it's an AgentStep)
agentStep, ok := job.Steps[0].(*compiler.AgentStep)
if !ok {
    return fmt.Errorf("first step is not an agent")
}

// Create agent processor (using blueprint's Agent, not service.Prompter)
processor := iosystem.NewAgentProcessor(agentStep.Agent)

// Create pipeline
source := iosystem.NewStdinSource()
sink := iosystem.NewStdoutSink()

pipeline := iosystem.NewPipeline(source, sink)
pipeline.AddProcessor(processor)
pipeline.Run(ctx)
```

### Blueprint with Chunking

```go
// Process large files with chunking through blueprint
bp, _ := blueprint.New(ctx, "workflow.yml", &factory{})
job, _ := bp.GetJob("analyzer")
agentStep := job.Steps[0].(*compiler.AgentStep)

source, _ := iosystem.NewFileSource("large_doc.txt")
sink := iosystem.NewStdoutSink()

pipeline := iosystem.NewPipeline(source, sink)
pipeline.
    AddProcessor(iosystem.NewChunkerProcessor("chunk", 
        iosystem.WithChunkSize(2048))).
    AddProcessor(iosystem.NewAgentProcessor(agentStep.Agent))
pipeline.Run(ctx)

// Each chunk flows through the blueprint agent
// Results aggregated at output
```

### Blueprint Configuration for I/O

Extend blueprint YAML to declare I/O:

```yaml
name: document-processor
version: v1

input:
  source: fs
  path: s3://mybucket/inputs  # Automatically uses stream.NewFS
  chunking:
    strategy: chunk
    size: 2048

output:
  sink: fs
  path: s3://mybucket/outputs  # Automatically uses stream.NewFS

jobs:
  main:
    steps:
      - uses: extract.md
      - uses: transform.md
```

Parser reads I/O config, compiler creates pipeline automatically.

## Unified Filesystem Approach

### Key Insight from github.com/fogfish/stream

The `stream` library already provides everything we need for unified filesystem access:

```go
// spool.FileSystem is an INTERFACE that both stream and lfs implement
type FileSystem interface {
    fs.FS
    Create(path string, attr *struct{}) (File, error)
    Remove(path string) error
}

// Works for both local and S3
func mount(path string) (spool.FileSystem, error) {
    if strings.HasPrefix(path, "s3://") {
        return stream.NewFS(path[5:])  // Returns *stream.FileSystem (implements spool.FileSystem)
    }
    return lfs.New(path)  // Returns *lfs.FileSystem (implements spool.FileSystem)
}

// Now WalkSource and FSSink use this single interface
source, _ := iosystem.NewWalkSource("s3://bucket/path")  // Uses stream.NewFS
source, _ := iosystem.NewWalkSource("/local/path")       // Uses lfs.New
```

**Benefits:**
1. **No separate implementations** - ONE `WalkSource` for both local and S3
2. **No separate implementations** - ONE `FSSink` for both local and S3
3. **Standard interface** - `fs.FS` throughout
4. **Transparent switching** - Via path prefix only
5. **All stream features** - metadata, copy, remove, glob, etc.
6. **Battle-tested** - Already in production use

## Migration Strategy

### Phase 1: Core Abstractions (Week 1)

1. Create `internal/iosystem/` package
2. Define interfaces: `Source`, `Processor`, `Sink`, `Pipeline`
3. Implement basic sources: `StdinSource`, `FileSource`, `FilesSource`
4. Implement basic processors: `IdentityProcessor`, `ChunkerProcessor`
5. Implement basic sinks: `StdoutSink`, `FileSink`
6. Unit tests for each component

### Phase 2: Filesystem Sources/Sinks (Week 2)

1. Implement `WalkSource` using `stream.NewFS`/`lfs.New`
2. Implement `FSSink` using `stream.NewFS`/`lfs.New`
3. Implement `MergedSource`
4. Add mutable mode support to `WalkSource`
5. Integration tests with local filesystem (tmpdir)
6. Integration tests with S3 (localstack or mocks)

### Phase 3: Blueprint Integration (Week 3)

1. **Retire `internal/service/worker.go`** - no longer needed
2. **Update `internal/service/prompt.go`** to optionally use `blueprint/compiler.Agent`
3. Implement `AgentProcessor` using `compiler.Agent`
4. Implement `BlueprintProcessor`
5. Update `internal/reader/` to use new processors
6. Integration tests with LLM mocking

### Phase 4: Pipeline Orchestrator (Week 4)

1. Implement `Pipeline` with basic flow
2. Add concurrency support (NO NEEDS NOW)
3. Add error handling modes
4. Add progress/metrics callbacks
5. Performance testing

### Phase 5: Command Migration (Week 5)

1. Refactor `cmd/tell.go` to use pipeline
2. Refactor `cmd/ask.go` to use pipeline
3. Refactor `cmd/run.go` to use pipeline
4. Refactor `cmd/task.go` to use pipeline
5. Ensure backward compatibility
6. Update CLI help/docs

### Phase 6: HTTP Extensions (Week 6)

1. Implement `HTTPSource` (embedded server)
2. Implement `HTTPSink` (webhook)
3. Add new command: `iq serve` (HTTP endpoint for workflows)
4. Add authentication/authorization
5. API documentation

### Phase 7: Blueprint Integration (Week 7)

1. Extend blueprint parser for I/O declarations
2. Extend compiler to create pipelines from YAML
3. Add blueprint examples with I/O
4. Update ARCHITECTURE.md
5. End-to-end testing

## Backward Compatibility

All existing commands continue to work unchanged:

```bash
# These all work as-is
echo "text" | iq tell
iq tell -p prompt.yml file.txt
iq ask -d ./input -o ./output
iq run -p workflow.yml -d s3://bucket/in -o s3://bucket/out
cat doc.txt | iq task -p task.yml
```

Internal implementation migrates to new system without breaking CLI.

## Example Usage Patterns

### Pattern 1: Simple Stdin to Stdout

```go
source := iosystem.NewStdinSource()
sink := iosystem.NewStdoutSink()

pipeline := iosystem.NewPipeline(source, sink)
pipeline.AddProcessor(iosystem.NewAgentProcessor(agent))
pipeline.Run(ctx)
```

### Pattern 2: Local Directory with Chunking

```go
source, _ := iosystem.NewWalkSource("./input", iosystem.WalkConfig{Mutable: true})
sink, _ := iosystem.NewFSSink("./output")

pipeline := iosystem.NewPipeline(source, sink,
    iosystem.WithConcurrency(8),
)
pipeline.
    AddProcessor(iosystem.NewChunkerProcessor("chunk")).
    AddProcessor(iosystem.NewAgentProcessor(agent))
pipeline.Run(ctx)
```

### Pattern 3: S3 to S3 with Blueprint

```go
// Transparent S3 usage via stream.NewFS
source, _ := iosystem.NewWalkSource("s3://input-bucket/prefix", iosystem.WalkConfig{})
sink, _ := iosystem.NewFSSink("s3://output-bucket/results")

pipeline := iosystem.NewPipeline(source, sink)
pipeline.AddProcessor(iosystem.NewBlueprintProcessor(bp, "main"))
pipeline.Run(ctx)
```

### Pattern 4: HTTP Server Processing

```go
source, _ := iosystem.NewHTTPSource(":8080")
sink, _ := iosystem.NewFSSink("./processed")

pipeline := iosystem.NewPipeline(source, sink)
pipeline.AddProcessor(iosystem.NewBlueprintProcessor(bp, "upload-handler"))

// Starts HTTP server, processes uploads through blueprint
pipeline.Run(ctx)
```

### Pattern 5: Merge Multiple Files

```go
source := iosystem.NewMergedSource(
    iosystem.NewFileSource("doc1.txt"),
    iosystem.NewFileSource("doc2.txt"),
    iosystem.NewFileSource("doc3.txt"),
)

pipeline := iosystem.NewPipeline(source, iosystem.NewStdoutSink())
pipeline.AddProcessor(iosystem.NewAgentProcessor(agent))
pipeline.Run(ctx)
```

## Extension Points

### Custom Sources

```go
// Kafka source example
type KafkaSource struct {
    consumer *kafka.Consumer
    topic    string
}

func (s *KafkaSource) Next(ctx context.Context) (*Document, error) {
    msg, err := s.consumer.ReadMessage(ctx)
    if err != nil {
        return nil, err
    }
    
    return &Document{
        Path:   fmt.Sprintf("kafka://%s/%d", s.topic, msg.Offset),
        Reader: bytes.NewReader(msg.Value),
        Metadata: map[string]string{
            "partition": fmt.Sprintf("%d", msg.Partition),
            "offset":    fmt.Sprintf("%d", msg.Offset),
        },
    }, nil
}

// Register factory
iosystem.RegisterSource("kafka", func(config map[string]any) (Source, error) {
    return NewKafkaSource(config["topic"].(string)), nil
})
```

### Custom Processors

```go
// Markdown to HTML processor
type MarkdownProcessor struct{}

func (p *MarkdownProcessor) Process(ctx context.Context, doc *Document) ([]*Document, error) {
    content, _ := io.ReadAll(doc.Reader)
    html := markdown.ToHTML(content)
    
    return []*Document{{
        Path:     strings.Replace(doc.Path, ".md", ".html", 1),
        Reader:   bytes.NewReader(html),
        Metadata: doc.Metadata,
    }}, nil
}
```

### Custom Sinks

```go
// PostgreSQL sink (store results in database)
type PostgresSink struct {
    db *sql.DB
}

func (s *PostgresSink) Write(ctx context.Context, doc *Document) error {
    content, _ := io.ReadAll(doc.Reader)
    _, err := s.db.ExecContext(ctx,
        "INSERT INTO documents (path, content) VALUES ($1, $2)",
        doc.Path, content)
    return err
}
```

## Performance Considerations

1. **Streaming**: All components use `io.Reader` to avoid loading entire files into memory
2. **Concurrency**: Pipeline supports parallel processing with worker pools
3. **Buffering**: Strategic buffering in processors to reduce I/O overhead
4. **Connection Pooling**: S3 clients (via stream) reuse connections
5. **Zero-Copy**: Direct reader chaining where possible (no intermediate buffers)
6. **Stream Library**: Leverages optimized S3 multipart upload/download in `stream`

## Testing Strategy

1. **Unit Tests**: Each source/processor/sink tested independently with mocks
2. **Integration Tests**: Full pipelines with real filesystems (tmpdir via `lfs`)
3. **E2E Tests**: Complete workflows with S3 (localstack), HTTP servers
4. **Benchmark Tests**: Performance metrics for chunking, concurrency
5. **Chaos Tests**: Error injection to test error handling modes

## Security Considerations

1. **Path Traversal**: Validate all file paths, prevent `../` escapes
2. **Resource Limits**: Configurable max file size, max documents
3. **Authentication**: S3 uses AWS credentials (via stream), HTTP uses bearer tokens
4. **Input Validation**: Sanitize all user-provided paths and configurations
5. **Rate Limiting**: HTTP source has configurable rate limits

## Monitoring and Observability

```go
type PipelineMetrics struct {
    DocumentsIn      prometheus.Counter
    DocumentsOut     prometheus.Counter
    BytesProcessed   prometheus.Counter
    ProcessingTime   prometheus.Histogram
    ErrorsTotal      prometheus.Counter
    ActivePipelines  prometheus.Gauge
}

// Pipeline supports metrics injection
pipeline := iosystem.NewPipeline(source, sink,
    iosystem.WithMetrics(func(stats PipelineStats) {
        metrics.DocumentsIn.Add(float64(stats.DocsProcessed))
        metrics.BytesProcessed.Add(float64(stats.BytesRead))
    }),
)
```

## File Structure

```
internal/iosystem/
├── source.go           # Source interface and base types
├── sink.go             # Sink interface and base types
├── processor.go        # Processor interface and base types
├── pipeline.go         # Pipeline orchestrator
├── document.go         # Document type
├── registry.go         # Factory registry
│
├── source/
│   ├── stdin.go
│   ├── file.go
│   ├── fs.go          # Unified FS source (uses stream)
│   ├── http.go
│   └── merged.go
│
├── processor/
│   ├── identity.go
│   ├── chunker.go
│   ├── agent.go       # Uses blueprint/compiler.Agent
│   └── blueprint.go
│
├── sink/
│   ├── stdout.go
│   ├── file.go
│   ├── fs.go          # Unified FS sink (uses stream)
│   ├── http.go
│   └── multi.go
│
└── internal/
    ├── buffer.go       # Buffering utilities
    ├── pool.go         # Worker pools
    └── validation.go   # Path/input validation
```

## Key Design Decisions

### 1. Use github.com/fogfish/stream for All Filesystem I/O

**Rationale:**
- Already in use and proven in codebase
- Implements standard `fs.FS` interface
- Transparent S3/local filesystem abstraction
- Handles S3 multipart upload/download optimization
- Supports metadata, glob patterns, copy, remove operations

**Impact:**
- Only ONE filesystem source implementation needed (`WalkSource`)
- Only ONE filesystem sink implementation needed (`FSSink`)
- Simpler code, less duplication
- Better testing (can use `lfs.NewTempFS` for local tests)

### 2. Retire internal/service/worker.go

**Rationale:**
- `blueprint/compiler.Agent` provides same functionality
- Cleaner architecture - blueprints are self-contained
- Reduces maintenance burden
- Eliminates duplicate LLM agent abstractions

**Impact:**
- `AgentProcessor` uses `compiler.Agent` directly
- `service.Prompter` may still exist for simple prompting
- Blueprint workflows become primary agent abstraction
- Commands can use blueprint agents without service layer

### 3. Blueprint Agents as First-Class Processors

**Rationale:**
- Workflows are central to iq's value proposition
- Pipelines should treat agents and workflows uniformly
- Enables composability (chain multiple workflows)

**Impact:**
- `BlueprintProcessor` executes full workflow jobs
- `AgentProcessor` executes individual blueprint agents
- Can mix and match agents from different blueprints
- Commands become thin wrappers around pipelines

## Open Questions

1. **Chunking State**: How to reassemble chunked results? Add aggregator processor?
2. **Transactional Output**: Should sinks support rollback on error?
3. **Streaming Output**: Can sinks support streaming writes for large outputs?
4. **Schema Validation**: Should processors validate document schemas?
5. **Compression**: Transparent compression for sources/sinks?
6. **Encryption**: At-rest encryption for sensitive documents?

## Success Criteria

1. ✅ All existing commands work unchanged
2. ✅ Chunking functionality preserved with same performance
3. ✅ Blueprint workflows can use I/O system
4. ✅ New HTTP source/sink working with examples
5. ✅ Only ONE filesystem source/sink implementation (using stream)
6. ✅ `internal/service/worker.go` retired successfully
7. ✅ 90%+ code coverage for iosystem package
8. ✅ Performance within 10% of current implementation
9. ✅ Zero breaking changes to public API
10. ✅ Documentation complete with examples

## Next Steps

1. **✅ Review this design** with team/stakeholders - ADDRESSED feedback
2. **Prototype core interfaces** with simple implementations
3. **Test stream integration** with local and S3 filesystems
4. **Benchmark chunking performance** vs current implementation
5. **Begin Phase 1 implementation**
6. **Plan service.worker retirement** migration path
7. **Iterate based on feedback**
