# IOSystem Package - Phase 1 Implementation

## Overview

The `iosystem` package provides a universal, extensible I/O abstraction system for processing documents through pipelines. It unifies input sources, transformation processors, and output destinations with a clean, composable interface.

## Phase 1 Deliverables ✅

### Core Interfaces
- **Document** - Represents a single document with path, reader, and metadata
- **Source** - Produces documents from input (stdin, files, etc.)
- **Processor** - Transforms documents in the pipeline
- **Sink** - Consumes processed documents and writes to output
- **Pipeline** - Orchestrates the flow: Source → Processor(s) → Sink

### Implemented Sources
- **StdinSource** - Reads from standard input
- **FileSource** - Reads a single file (local or S3 via `github.com/fogfish/stream`)
- **FileSeqSource** - Reads multiple files in sequence

### Implemented Processors
- **IdentityProcessor** - Pass-through processor (no transformation)
- **ChunkerProcessor** - Splits documents by strategy:
  - `none` - No splitting
  - `sentence` - Split by sentences
  - `paragraph` - Split by paragraphs  
  - `chunk` - Split into semantic chunks of specified size

### Implemented Sinks
- **StdoutSink** - Writes to standard output
- **FileSink** - Writes to a single file (local or S3)

## Architecture

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   Source     │──▶│  Processor   │──▶│     Sink     │
│  (Input)     │   │  (Transform) │   │   (Output)   │
└──────────────┘   └──────────────┘   └──────────────┘
```

## Usage Examples

### Example 1: Simple Pipeline

```go
package main

import (
    "context"
    "github.com/fogfish/iq/internal/iosystem"
    "github.com/fogfish/iq/internal/iosystem/source"
    "github.com/fogfish/iq/internal/iosystem/sink"
)

func main() {
    src := source.NewStdinSource()
    snk := sink.NewStdoutSink()

    pipeline := iosystem.NewPipeline(src, snk, nil)
    pipeline.Run(context.Background())
}
```

### Example 2: With Chunking

```go
package main

import (
    "context"
    "github.com/fogfish/iq/internal/iosystem"
    "github.com/fogfish/iq/internal/iosystem/source"
    "github.com/fogfish/iq/internal/iosystem/processor"
    "github.com/fogfish/iq/internal/iosystem/sink"
)

func main() {
    src, _ := source.NewFileSource("input.txt")
    snk := sink.NewStdoutSink()
    
    chunker := processor.NewChunkerProcessor(processor.ChunkConfig{
        Strategy:  processor.StrategySentence,
    })

    pipeline := iosystem.NewPipeline(src, snk, nil).
        AddProcessor(chunker)
    
    pipeline.Run(context.Background())
}
```

### Example 3: Multiple Processors

```go
package main

import (
    "context"
    "github.com/fogfish/iq/internal/iosystem"
    "github.com/fogfish/iq/internal/iosystem/source"
    "github.com/fogfish/iq/internal/iosystem/processor"
    "github.com/fogfish/iq/internal/iosystem/sink"
)

func main() {
    src, _ := source.NewFileSeqSource("doc1.txt", "doc2.txt", "doc3.txt")
    snk, _ := sink.NewFileSink("output.txt")
    
    chunker := processor.NewChunkerProcessor(processor.ChunkConfig{
        Strategy:  processor.StrategyChunk,
        ChunkSize: 2048,
    })
    identity := processor.NewIdentityProcessor()

    pipeline := iosystem.NewPipeline(src, snk, nil).
        AddProcessor(chunker).
        AddProcessor(identity)
    
    pipeline.Run(context.Background())
}
```

### Example 4: Progress Tracking

```go
package main

import (
    "context"
    "fmt"
    "github.com/fogfish/iq/internal/iosystem"
    "github.com/fogfish/iq/internal/iosystem/source"
    "github.com/fogfish/iq/internal/iosystem/sink"
)

func main() {
    src, _ := source.NewFileSeqSource("file1.txt", "file2.txt")
    snk := sink.NewStdoutSink()

    config := &iosystem.PipelineConfig{
        Progress: func(doc *iosystem.Document, err error) {
            if err != nil {
                fmt.Printf("Error: %v\n", err)
            } else {
                fmt.Printf("Processed: %s\n", doc.Path)
            }
        },
        Metrics: func(stats iosystem.PipelineStats) {
            fmt.Printf("Total: %d docs in %s\n", 
                stats.DocsProcessed, stats.Duration)
        },
    }
    
    pipeline := iosystem.NewPipeline(src, snk, config)
    pipeline.Run(context.Background())
}
```

## Testing

Run all tests:
```bash
go test ./internal/iosystem/...
```

Run specific test:
```bash
go test ./internal/iosystem -run TestPipeline_Simple
```

Run with coverage:
```bash
go test ./internal/iosystem/... -cover
```

Run the example:
```bash
go run ./internal/iosystem/examples/basic/main.go
```

## Key Design Features

### 1. Streaming by Default
All components use `io.Reader` to avoid loading entire files into memory. Large files can be processed efficiently.

### 2. Composable Architecture
Sources, processors, and sinks can be mixed and matched. Multiple processors can be chained.

### 3. Metadata Propagation
Documents carry metadata through the pipeline. Processors can add/modify metadata.

### 4. Chunking Integration
Uses existing `github.com/fogfish/scanner` library for proven chunking strategies.

### 5. Unified Filesystem Access
Uses `github.com/fogfish/stream` for transparent local/S3 file access.

### 6. Error Handling
Supports both fail-fast and skip-error modes.

### 7. Observability
Progress callbacks and metrics provide visibility into pipeline execution.

## Future Phases

### Phase 2: Filesystem Sources/Sinks (Week 2)
- WalkSource - walks directory trees (local/S3)
- FSSink - writes multiple files to directory
- MergedSource - combines multiple sources

### Phase 3: Blueprint Integration (Week 3)
- AgentProcessor - processes via blueprint agents
- BlueprintProcessor - executes workflow jobs
- Retire internal/service/worker.go

### Phase 4: Pipeline Orchestrator (Week 4)
- Concurrent processing with worker pools
- Advanced error handling modes
- Performance optimization

### Phase 5: Command Migration (Week 5)
- Refactor cmd/* to use iosystem
- Maintain backward compatibility

### Phase 6: HTTP Extensions (Week 6)
- HTTPSource - embedded HTTP server
- HTTPSink - webhook integration

### Phase 7: Blueprint Integration (Week 7)
- Extend parser for I/O declarations
- Compiler-driven pipeline creation

## File Structure

```
internal/iosystem/
├── document.go              # Document type
├── source.go                # Source interface
├── processor.go             # Processor interface
├── sink.go                  # Sink interface
├── pipeline.go              # Pipeline orchestrator
├── *_test.go               # Unit tests
│
├── source/
│   ├── stdin.go            # Stdin source
│   ├── file.go             # File/Files sources
│   └── *_test.go           # Source tests
│
├── processor/
│   ├── identity.go         # Identity processor
│   ├── chunker.go          # Chunking processor
│   └── *_test.go           # Processor tests
│
├── sink/
│   ├── stdout.go           # Stdout sink
│   ├── file.go             # File sink
│   └── *_test.go           # Sink tests
│
└── examples/
    └── basic/
        └── main.go         # Example usage
```

## Dependencies

- `github.com/fogfish/scanner` - Text chunking strategies
- `github.com/fogfish/stream` - Unified S3/local filesystem
- `github.com/fogfish/stream/lfs` - Local filesystem wrapper
- `github.com/fogfish/stream/spool` - Queue-based processing (Phase 2)

## Notes

- All components are goroutine-safe for single-goroutine use
- Concurrent processing will be added in Phase 4
- S3 support is present but requires AWS credentials configured
- Chunking strategies match existing `internal/reader` behavior

## Success Criteria

- ✅ All core interfaces defined
- ✅ Basic sources implemented (stdin, file, files)
- ✅ Basic processors implemented (identity, chunker)
- ✅ Basic sinks implemented (stdout, file)
- ✅ Pipeline orchestration working
- ✅ Comprehensive unit tests
- ✅ Integration tests
- ✅ Example code
- ✅ Documentation

## Review Checklist

- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Test coverage > 80%
- [ ] Example runs successfully
- [ ] Documentation is clear and complete
- [ ] Design aligns with overall architecture
- [ ] No breaking changes to existing code
- [ ] Ready for Phase 2 extension
