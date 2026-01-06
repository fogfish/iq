# ADR 0007: Array Input and Foreach Output Formatting

**Status:** Accepted  
**Date:** 2026-01-03  
**Context:** Enabling array-based input and configurable output formatting for map/reduce workflows

---

## In the context of

Building LLM-powered workflows that need to process collections of items (JSON arrays, split documents, newline-delimited JSON) and aggregate results with specific output formats. The tool excels at transformation (map) through existing `foreach` construct, but lacks native support for:

1. **Array input injection**: Cannot start workflows with JSON array or collection as input
2. **Split-to-array conversion**: `--splitter` produces individual documents, not arrays consumable by `foreach`
3. **Configurable output formatting**: `foreach` always collects results as Go arrays, but users need JSON, JSONL, or text output with custom delimiters

**Motivating Use Cases:**

```yaml
# Use Case 1: Process JSON array from file
# Input: [{"name": "Alice"}, {"name": "Bob"}]
# Workflow processes each item, aggregates results

# Use Case 2: Split large document, process chunks, merge results
# Input: large-doc.txt (20KB)
# CLI: --splitter paragraph --array
# Workflow: foreach processes chunks, next step gets aggregated output

# Use Case 3: Multiple formats needed
# Same workflow needs JSON array (structured), JSONL (streaming), or text (summaries)
```

## Facing Concern

**Minimal invasiveness**: Solution must extend existing `ForeachStep.Prompt()` (line 346, `workflow.go`) without architectural rewrites. ADR 0005 declined due to complexity—avoid similar fate. No new service abstractions.

**Streaming preservation**: Normal mode (without `--array`) MUST continue streaming documents through pipeline on-demand. Loading all documents into memory unacceptable for non-array mode. Only array mode can buffer in memory.

**Context injection point**: `ForeachStep` extracts arrays from `WorkflowContext.document` and processes items. Need mechanism to populate context with arrays from CLI input (files, stdin, split results) while maintaining streaming for normal workflows.

**Leverage existing architecture**: Must use Conduit/Processor pipeline, not create new service abstractions. Array collection should be a processor in the pipeline.

**Format vs reduce confusion**: Previous terminology ("reduce") implies aggregation function. Actual need is output serialization/formatting (codec). Delimiter configuration required for text format.

**Backward compatibility**: Existing `--splitter` behavior (separate documents with `.seq-NNNN` suffix), `--merge` flag, and `foreach` with `uses:` agents must continue working unchanged.

## We decided for

**CLI-level array injection via `--array` flag** that converts input sources (files, splits, stdin) into arrays collected by new `ArrayCollector` processor. Arrays injected into `WorkflowContext` as both `document` (original workflow input) and `input` (current agent input), accessible via `selector: document`.

**Configurable output formatting per foreach** using interface-based `Formatter` system (NOT reduce functions—this is serialization). `ForeachStep` gains optional `format:` attribute specifying output codec: `json`, `jsonl`, or `text` with configurable delimiters.

**Processor-based array collection**: New `ArrayCollector` processor added as first stage in Conduit pipeline when `--array` flag used. Maintains streaming for normal mode, buffers only in array mode.

**No workflow-level split step** (keeping ADR 0005 declined). Split remains CLI/processor concern via `--splitter` flag. When combined with `--array`, split results collected into array by `ArrayCollector`.

## Design

### **1. CLI Flag: `--array`**

New flag enables array mode, adding `ArrayCollector` processor to pipeline:

```bash
# Example 1: JSON array file
iq agent -f workflow.yml input.json --array

# Example 2: Multiple files as array
iq agent -f workflow.yml file1.txt file2.txt --array

# Example 3: Split + array
iq agent -f workflow.yml large.txt --splitter paragraph --array

# Example 4: Normal mode (streaming, no array)
iq agent -f workflow.yml large.txt --splitter paragraph  # NO --array
```

**Behavior:**
- **Without `--array`**: Normal streaming (each document flows through pipeline independently)
- **With `--array`**: Array mode (all documents collected by ArrayCollector, injected as array in context)

---

### **2. Monadic Processor Interface**

**Location:** `internal/iosystem/types.go` (modify existing Processor interface)

```go
// Processor transforms documents in a pipeline.
// Processors are monadic: accept array of documents, return array of documents.
//
// A processor can:
//   - Transform documents (map)
//   - Split documents (flatMap - one doc becomes many)
//   - Filter documents (return empty slice)
//   - Collect/aggregate documents (ArrayCollector pattern)
//
// Implementations should:
//   - Be stateless where possible
//   - Return errors for processing failures
//   - Release resources in Close()
type Processor interface {
    // Process takes input documents and produces zero or more output documents.
    // Monadic signature enables:
    //   - Single doc processing: len(docs)==1 (normal case)
    //   - Array processing: len(docs)>1 (array mode after collection)
    //   - EOF signal: docs[0].Type == ContentEOF (end of stream)
    // Return empty slice to filter out documents.
    // Return error for processing failures.
    Process(ctx context.Context, docs []*Document) ([]*Document, error)

    // Close releases any resources held by the processor.
    Close() error
}
```

**Rationale:** Monadic interface (`[]*Document → []*Document`) eliminates double encoding:
- ArrayCollector emits `[]*Document` array directly (no JSON encoding)
- AgentProcessor receives `[]*Document` array directly (no JSON decoding)
- Normal processors receive single-element array: `[doc]`
- EOF signal flows as document with `ContentEOF` type

**ContentEOF constant:**

```go
const (
    ContentText ContentType = "text/plain"
    ContentJSON ContentType = "application/json"
    ContentEOF  ContentType = "application/x-eof"  // NEW: Signals end of stream
)
```

---

### **3. Array Collector Processor**

**Location:** `internal/iosystem/processor/collector.go` (new processor implementing standard Processor)

```go
package processor

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"

    "github.com/fogfish/iq/internal/iosystem"
)

// ArrayCollector collects all input documents and emits them as array.
// NO encoding/decoding - passes documents directly (monadic).
type ArrayCollector struct {
    collected []*iosystem.Document  // Collected documents
}

// NewArrayCollector creates processor that collects documents into array.
func NewArrayCollector() *ArrayCollector {
    return &ArrayCollector{
        collected: make([]*iosystem.Document, 0),
    }
}

// Process collects documents or emits array on EOF signal.
// Normal documents: collected, return empty
// EOF document: emit collected []*Document array (NO JSON encoding)
func (p *ArrayCollector) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
    // Check for EOF signal
    if len(docs) > 0 && docs[0].Type == iosystem.ContentEOF {
        // Return collected documents as array (monadic - no encoding!)
        result := p.collected
        p.collected = nil  // Clear for reuse
        return result, nil
    }
    
    // Collect all input documents
    for _, doc := range docs {
        p.collected = append(p.collected, doc)
    }

    // Return empty - continue collecting until EOF
    return []*iosystem.Document{}, nil
}

// Close finalizes collection
func (p *ArrayCollector) Close() error {
    return nil
}
```

**Integration with Agent Processor:**

Agent processor updated to accept document arrays:

```go
// internal/iosystem/processor/agent.go (signature change)

func (p *Agent) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
    // Array mode: len(docs) > 1, pass array to workflow
    // Normal mode: len(docs) == 1, pass single doc
    // EOF passthrough: docs[0].Type == ContentEOF, ignore
    
    if len(docs) == 0 || (len(docs) == 1 && docs[0].Type == iosystem.ContentEOF) {
        return docs, nil  // Passthrough EOF or empty
    }
    
    var input any
    
    if len(docs) == 1 {
        // Normal mode: single document
        doc := docs[0]
        content, err := io.ReadAll(doc.Reader)
        if err != nil {
            return nil, fmt.Errorf("failed to read document: %w", err)
        }
        
        if doc.Type == iosystem.ContentJSON {
            if err := json.Unmarshal(content, &input); err != nil {
                return nil, fmt.Errorf("failed to parse JSON: %w", err)
            }
        } else {
            input = string(content)
        }
    } else {
        // Array mode: multiple documents - NO JSON decoding!
        // Build array from documents directly
        items := make([]any, 0, len(docs))
        for _, doc := range docs {
            content, err := io.ReadAll(doc.Reader)
            if err != nil {
                return nil, fmt.Errorf("failed to read document: %w", err)
            }
            
            if doc.Type == iosystem.ContentJSON {
                var item any
                if err := json.Unmarshal(content, &item); err != nil {
                    return nil, fmt.Errorf("failed to parse JSON: %w", err)
                }
                items = append(items, item)
            } else {
                items = append(items, string(content))
            }
        }
        input = items  // Array of items, no double encoding!
    }
    } else {
        input = string(content)
    }

    // ArrayCollector emits JSON array document
    // Parsed above as input (Go array/slice type)
    // No special handling needed!

    // Create workflow context with input
    ctx = compiler.NewWorkflowContext(ctx, input)

    // Execute workflow
    result, err := p.job.Prompt(ctx, input)
    if err != nil {
        return nil, err
    }

    // Convert result to document
    // ... existing serialization logic ...
}
```

---

### **4. Conduit Integration for Monadic Processors**

**Location:** `internal/iosystem/conduit/conduit.go` (update to use monadic Process)

```go
// runSequential processes documents one at a time.
func (p *Conduit) runSequential(ctx context.Context, source iosystem.Source, sink iosystem.Sink, stats *Stats) error {
    // Process documents until source exhausted
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        doc, err := source.Next(ctx)
        if err == io.EOF {
            // NEW: Inject EOF document into pipeline
            // ArrayCollector detects this and emits collected []*Document array
            eofDoc := &iosystem.Document{
                Type: iosystem.ContentEOF,
                Path: "<eof>",
            }
            _ = p.processDocument(ctx, []*iosystem.Document{eofDoc}, sink, stats)
            break
        }
        if err != nil {
            stats.Errors = append(stats.Errors, err)
            if p.config.ErrorMode == FailFast {
                return err
            }
            continue
        }

        if err := p.processDocument(ctx, []*iosystem.Document{doc}, sink, stats); err != nil {
            stats.Errors = append(stats.Errors, err)
            if p.config.Progress != nil {
                p.config.Progress(doc, err)
            }
            if p.config.ErrorMode == FailFast {
                return err
            }
            stats.DocsSkipped++
            continue
        }

        stats.DocsProcessed++
        if p.config.Progress != nil {
            p.config.Progress(doc, nil)
        }
    }
    
    return nil
}

// processDocument runs documents through all processors (monadic flow)
// UPDATED: accepts []*Document, processors use monadic Process([]*Document)
func (p *Conduit) processDocument(ctx context.Context, docs []*Document, sink Sink, stats *Stats) error {
    // Pass documents through processor chain
    // Each processor can transform, filter, split, or collect
    currentDocs := docs
    
    for _, processor := range p.processors {
        // Monadic call: []*Document -> []*Document
        nextDocs, err := processor.Process(ctx, currentDocs)
        if err != nil {
            return err
        }
        currentDocs = nextDocs
    }
    
    // Write final documents to sink
    for _, d := range currentDocs {
        if d.Type == iosystem.ContentEOF {
            continue  // Don't write EOF markers to output
        }
        if err := sink.Write(ctx, d); err != nil {
            return err
        }
    }
    
    return nil
}
```

**Key changes:**
- `processDocument` accepts `[]*Document` instead of single document
- Processors called with array: `processor.Process(ctx, currentDocs)`
- Normal mode: array has single element `[doc]`
- Array mode: ArrayCollector emits array after EOF, AgentProcessor receives full array
- No loops over results - monadic flow handles batching naturally

---

### **5. Foreach Output Format Configuration**

**AST Extension:** Add `Format` field to `ForeachStepNode`

```go
// internal/blueprint/ast/ast.go

type ForeachStepNode struct {
    Name      string
    RunsOn    string
    Uses      string
    Selector  string
    Job       string
    Output    string
    Format    *FormatNode  // NEW: output serialization format
    Retry     *RetryNode
}

// FormatNode configures output serialization after foreach
type FormatNode struct {
    Type      string  // "json", "jsonl", "text" (default: "json")
    Delimiter string  // For text format (default: "\n")
}
```

**Parser Extension:** Parse `format:` attribute from YAML

```go
// internal/blueprint/parser/parser.go

type foreachYAML struct {
    Uses     string      `yaml:"uses,omitempty"`
    Selector string      `yaml:"selector,omitempty"`
    Job      string      `yaml:"job"`
    Format   *formatYAML `yaml:"format,omitempty"`  // NEW
}

type formatYAML struct {
    Type      string `yaml:"type"`           // "json", "jsonl", "text"
    Delimiter string `yaml:"delim,omitempty"`  // For text format
}

func (p *Parser) parseForeachStep(step stepYAML) (*ast.ForeachStepNode, error) {
    // ... existing validation ...
    
    var format *ast.FormatNode
    if step.Foreach.Format != nil {
        ftype := step.Foreach.Format.Type
        if ftype == "" {
            ftype = "json"  // Default
        }
        
        if ftype != "json" && ftype != "jsonl" && ftype != "text" {
            return nil, fmt.Errorf("invalid format type: %s (must be json, jsonl, or text)", ftype)
        }
        
        delim := step.Foreach.Format.Delimiter
        if delim == "" {
            delim = "\n"  // Default newline
        }
        
        format = &ast.FormatNode{
            Type:      ftype,
            Delimiter: delim,
        }
    } else {
        // Default: JSON array
        format = &ast.FormatNode{
            Type:      "json",
            Delimiter: "\n",
        }
    }
    
    return &ast.ForeachStepNode{
        // ... existing fields ...
        Format: format,
    }, nil
}
```

---

### **6. Formatter Interface**

**Location:** `internal/blueprint/compiler/formatter.go` (new file)

```go
package compiler

import (
    "bytes"
    "encoding/json"
    "fmt"
    "strings"
    
    "github.com/fogfish/iq/internal/blueprint/ast"
)

// Formatter serializes foreach results into output format.
// This is a CODEC for result serialization, NOT a reduce function.
type Formatter interface {
    // Format converts array of results into serialized output
    Format(results []any) (any, error)
}

//------------------------------------------------------------------------------

// JSONFormatter returns results as JSON array (default)
type JSONFormatter struct{}

func (f *JSONFormatter) Format(results []any) (any, error) {
    return results, nil  // Already array, JSON-encoded downstream
}

//------------------------------------------------------------------------------

// JSONLFormatter returns results as newline-delimited JSON
type JSONLFormatter struct{}

func (f *JSONLFormatter) Format(results []any) (any, error) {
    var buf bytes.Buffer
    encoder := json.NewEncoder(&buf)
    
    for i, result := range results {
        if err := encoder.Encode(result); err != nil {
            return nil, fmt.Errorf("failed to encode result %d: %w", i, err)
        }
    }
    
    return strings.TrimSpace(buf.String()), nil
}

//------------------------------------------------------------------------------

// TextFormatter concatenates results with configurable delimiter
type TextFormatter struct {
    Delimiter string
}

func (f *TextFormatter) Format(results []any) (any, error) {
    var buf bytes.Buffer
    
    for i, result := range results {
        if i > 0 {
            buf.WriteString(f.Delimiter)
        }
        
        // Convert result to string
        switch v := result.(type) {
        case string:
            buf.WriteString(v)
        case []byte:
            buf.Write(v)
        default:
            // Marshal to JSON if not string
            data, err := json.Marshal(v)
            if err != nil {
                return nil, fmt.Errorf("failed to marshal result %d: %w", i, err)
            }
            buf.Write(data)
        }
    }
    
    return buf.String(), nil
}

//------------------------------------------------------------------------------

// NewFormatter creates formatter based on format node
func NewFormatter(format *ast.FormatNode) (Formatter, error) {
    if format == nil {
        return &JSONFormatter{}, nil
    }
    
    switch format.Type {
    case "json":
        return &JSONFormatter{}, nil
    case "jsonl":
        return &JSONLFormatter{}, nil
    case "text":
        return &TextFormatter{Delimiter: format.Delimiter}, nil
    default:
        return nil, fmt.Errorf("unknown format type: %s", format.Type)
    }
}
```

---

### **7. ForeachStep Integration**

**Modify `ForeachStep` to use formatter:**

```go
// internal/blueprint/compiler/workflow.go

type ForeachStep struct {
    UsesAgent  *Agent
    Selector   cel.Program
    Job        *Job
    JobName    string
    OutputName string
    Retry      *Retry
    Formatter  Formatter  // NEW: output format serializer
}

func (step *ForeachStep) Prompt(ctx context.Context, opt ...chatter.Opt) error {
    wfCtx := GetWorkflowContext(ctx)
    if wfCtx == nil {
        return fmt.Errorf("workflow context not found in context")
    }

    reporter := progress.FromContext(ctx)
    startTime := time.Now()

    // ... existing array extraction logic (lines 300-365) ...
    
    // Execute job for each item (existing code)
    results := make([]any, 0, len(items))
    successCount := 0
    for i, item := range items {
        // ... existing per-item execution (lines 370-395) ...
        results = append(results, result)
    }

    if reporter != nil {
        reporter.SetForeachMode(false)
        reporter.ForeachComplete(successCount, len(items), time.Since(startTime))
    }

    // NEW: Apply output formatter
    finalResult, err := step.Formatter.Format(results)
    if err != nil {
        return fmt.Errorf("format failed: %w", err)
    }

    // Store formatted results in context
    if step.OutputName != "" {
        wfCtx.SetStepOutput(step.OutputName, finalResult)
    } else {
        wfCtx.Current = finalResult
    }

    return nil
}
```

---

### **8. Compiler Integration**

**Update compiler to create formatter:**

```go
// internal/blueprint/compiler/compiler.go

func (c *Compiler) compileForeachStep(node *ast.ForeachStepNode, job *Job) (*ForeachStep, error) {
    // ... existing compilation logic ...
    
    // Create formatter
    formatter, err := NewFormatter(node.Format)
    if err != nil {
        return nil, err
    }
    
    return &ForeachStep{
        UsesAgent:  usesAgent,
        Selector:   selector,
        Job:        targetJob,
        JobName:    node.Job,
        OutputName: node.Output,
        Retry:      retry,
        Formatter:  formatter,  // NEW
    }, nil
}
```

---

### **9. Worker Builder Integration**

**Add `--array` flag and ArrayCollector processor:**

```go
// cmd/opts.go

type optsAgent struct {
    file     string
    job      string
    json     bool
    array    bool     // NEW
    splitter string
    // ... other fields ...
}

func (opts *optsAgent) apply(cmd *cobra.Command) {
    f := cmd.PersistentFlags()
    
    f.StringVarP(&opts.file, "file", "f", "",
        "Path to workflow blueprint YAML file")
    
    f.BoolVar(&opts.array, "array", false,  // NEW
        "Collect all inputs into array for batch processing")
    
    // ... other flags ...
}

func (opts *optsAgent) build(llm chatter.Chatter, reporter progress.Reporter) (*worker.ConduitWithReporter, error) {
    return worker.New().
        Reporter(reporter).
        Runtime().
        ArrayMode(opts.array).       // NEW: adds ArrayCollector if enabled
        Splitter(processor.ChunkConfig{
            Strategy: opts.splitter,
            // ...,
        }).
        Workflow(opts.file, llm).
        Jsonify(opts.json).
        Build()
}
```

**Update worker builder:**

```go
// internal/service/worker/worker.go

func (b *Builder) ArrayMode(enable bool) *Builder {
    if b.err != nil || b.runtime == nil || !enable {
        return b
    }
    
    // Create and add ArrayCollector as FIRST processor
    // Implements BatchProcessor - emits array after source EOF
    b.runtime.AddProcessor(processor.NewArrayCollector())
    
    return b
}

// No need to pass reference to AgentProcessor!
// ArrayCollector emits document naturally through pipeline
```

---

## Complete Examples

### **Example 1: JSON Array Input**

**Input file (`users.json`):**
```json
[
  {"name": "Alice", "age": 30},
  {"name": "Bob", "age": 25}
]
```

**Workflow (`process-users.yml`):**
```yaml
name: process-users
jobs:
  main:
    steps:
      - foreach:
          selector: document  # document is array from --array input
          job: analyze-user
        output: analyzed
        format:           # Output formatting
          type: jsonl     # Newline-delimited JSON
      
      - uses: prompts/summarize.md
  
  analyze-user:
    steps:
      - uses: prompts/analyze.md
```

**CLI:**
```bash
iq agent -f process-users.yml users.json --array
```

**What happens:**
1. CLI reads `users.json`, Source emits each array element as document
2. Conduit calls `ArrayCollector.Process([doc])` for each document
3. ArrayCollector collects documents, returns empty array
4. Source exhausted, Conduit injects EOF: `ArrayCollector.Process([EOF doc])`
5. ArrayCollector detects EOF, returns collected `[]*Document` array (NO JSON encoding!)
6. Conduit calls `AgentProcessor.Process([doc1, doc2, ...])` with array
7. AgentProcessor sees len(docs) > 1, builds input array from documents (NO JSON decoding!)
8. Workflow context gets array as `document` and `input`
9. Foreach uses `selector: document` to iterate
10. Each user processed by `analyze-user` job
11. Results formatted via `format: {type: jsonl}` as newline-delimited JSON
12. Summary step receives JSONL string as input

**Key benefit:** No double encoding! ArrayCollector emits `[]*Document`, AgentProcessor receives `[]*Document` directly.

---

### **Example 2: Split + Array + Format**

**Input file (`large-doc.txt`):** 20KB document

**Workflow (`process-chunks.yml`):**
```yaml
name: process-chunks
jobs:
  main:
    steps:
      - foreach:
          selector: document
          job: analyze-chunk
        format:
          type: text
          delim: "\n\n"  # Double newline separator
      
      - uses: prompts/final-summary.md
  
  analyze-chunk:
    steps:
      - uses: prompts/extract-facts.md
```

**CLI:**
```bash
iq agent -f process-chunks.yml large-doc.txt --splitter paragraph --array
```

**What happens:**
1. `ChunkerProcessor.Process([doc])` splits document, returns `[chunk1, chunk2, ...]`
2. Conduit calls `ArrayCollector.Process([chunk1, chunk2, ...])` with all chunks
3. ArrayCollector collects chunks, returns empty
4. After splitter finishes, source exhausted, Conduit: `ArrayCollector.Process([EOF doc])`
5. ArrayCollector detects EOF, returns collected `[]*Document` array (documents, not JSON!)
6. `AgentProcessor.Process([chunk1, chunk2, ...])` receives document array
7. AgentProcessor builds input array from documents (single decoding pass)
8. Workflow context gets array as `document`
9. `foreach` iterates over chunks via `selector: document`
10. Each chunk processed by `analyze-chunk` job
11. Results formatted via `text` with double-newline delimiter
12. Final summary step receives concatenated text

---

### **Example 3: Multiple Files as Array**

**CLI:**
```bash
iq agent -f workflow.yml file1.txt file2.txt file3.txt --array
```

**What happens:**
1. Source emits three documents: `doc1`, `doc2`, `doc3`
2. Conduit: `ArrayCollector.Process([doc1])`, `Process([doc2])`, `Process([doc3])`
3. ArrayCollector collects: `[doc1, doc2, doc3]`, returns empty each time
4. After source EOF, Conduit: `ArrayCollector.Process([EOF doc])`
5. ArrayCollector emits: `[doc1, doc2, doc3]` (documents directly, no encoding!)
6. `AgentProcessor.Process([doc1, doc2, doc3])` receives array
7. Workflow processes array via `foreach: {selector: document, ...}`

---

### **Example 4: Backward Compatibility - Split Without Array (Streaming)**

**CLI (existing behavior):**
```bash
iq agent batch -f workflow.yml -I ./input -O ./output --splitter paragraph
```

**What happens:**
1. Each input file split into chunks
2. Each chunk streams through pipeline independently (NO ArrayCollector)
3. Each chunk produces separate output file (`.seq-0000` suffix)
4. **Memory efficient** - only one chunk in memory at a time

---

## Neglected

**Alternative architectures rejected:**

- **Workflow-level split step** (ADR 0005 approach): Declined due to complexity. Would require new step type, AST changes, multiple execution paths. Violates "small extension" constraint.

- **Automatic JSON array detection**: Could auto-detect `[...]` and inject as array without `--array` flag. Rejected because ambiguous—user might want to process JSON array as single document (pass entire array to LLM). Explicit `--array` flag clearer.

- **Format as separate workflow step**: Could add `format:` step after `foreach`. Rejected because formatting is inherently part of foreach output semantics. Separate step adds unnecessary verbosity.

- **Custom format functions**: Could allow users to define format logic via CEL expressions or scripts. Rejected as over-engineered for initial implementation. Three built-in formats (json/jsonl/text) cover 95% of use cases.

- **Streaming format**: Could process foreach results as stream without collecting in memory. Rejected because foreach already executes sequentially, and formatting needs all results to serialize correctly (especially for JSON array).

- **Service layer array injection**: Initial ADR draft proposed loading all documents in service layer before pipeline. Rejected per feedback because breaks streaming semantics for normal mode, causes memory bloat, and creates new abstraction instead of leveraging Conduit/Processor architecture.

- **JSONL auto-detection by extension**: Could detect `.jsonl` files and automatically parse as array. Deferred as lower priority—explicit `--array` flag sufficient for initial implementation.

---

## To Achieve

**Minimal invasiveness**: Solution extends existing `ForeachStep` and modifies Conduit/Processor architecture minimally. Core changes:
- Processor interface signature: `Process(ctx, doc *Document)` → `Process(ctx, docs []*Document)` (monadic)
- New `ContentEOF` document type constant (~1 line in types.go)
- New `ArrayCollector` processor (~30 lines - simple collection logic)
- Conduit: inject EOF document (~3 lines), update processDocument signature (~10 lines)
- AgentProcessor: handle document arrays (~20 lines)
- `Formatter` interface (~120 lines)
- AST/parser extensions (~50 lines)
- Total: ~230 lines of actual implementation

**Streaming preserved**: Normal mode (without `--array`) continues streaming through pipeline—documents processed on-demand as single-element arrays `[doc]`. No memory accumulation. Array mode explicitly opt-in via CLI flag. ArrayCollector only active when flag used.

**Clean lifecycle**: EOF signal explicit via special document type flowing through existing Processor.Process(). ArrayCollector detects EOF document, emits collected array. No weak "ready" checks or GetArray() methods. Natural monadic document flow.

**No abstraction leaks**: AgentProcessor doesn't check for ArrayCollector presence. Receives document array via monadic interface. No special handling needed. EOF document is regular document with special type.

**Zero double encoding**: **Critical improvement** - monadic interface eliminates double JSON encoding/decoding:
- ArrayCollector emits `[]*Document` array directly (documents, not JSON bytes)
- AgentProcessor receives `[]*Document` array via monadic signature
- Single JSON decode pass when reading document contents
- Efficient: no marshal → unmarshal round-trip

**Use case coverage**: Handles all motivating scenarios:
1. JSON array input: `--array` + `selector: document`
2. Split + aggregate: `--splitter` + `--array` + `format:`
3. Multiple output formats: JSON/JSONL/text with delimiter control

**Flexible formatting**: Interface-based formatter system allows easy extension. Adding new formats (e.g., CSV, XML) requires single new `Formatter` implementation, no workflow changes. Delimiter configurable per-format.

**Backward compatibility**: Existing workflows unchanged. New functionality opt-in via:
- `--array` flag (CLI)
- `format:` attribute (workflow YAML)
- `selector: document` (workflow logic)

**Clear semantics**: 
- Without `--array`: streaming mode (each document → full pipeline run independently)
- With `--array`: batch mode (all documents collected → array → one pipeline run)
- `--splitter` without `--array`: independent chunks (streaming, separate outputs)
- `--splitter` with `--array`: chunks collected as array (batch mode)

**Context clarity**: 
- `document`: original workflow input (array when `--array` used)
- `input`: current agent input (available as `{{.input}}` in templates, initially same as `document`)
- `current`: most recent step output

**Debuggability**: Array collection happens at processor boundary, visible in pipeline logs. Format strategy explicit in workflow YAML. Users understand what's happening.

**Testability**: Each component independently testable:
- `Formatter` interface: unit tests for each format type
- `ArrayCollector`: processor unit tests
- Integration: end-to-end CLI tests

---

## Accepting

**In-memory array requirement**: `--array` mode loads all documents into memory via `ArrayCollector`. Not streaming-friendly for large datasets (e.g., 1000 files × 10MB each). Acceptable because:
- Array mode opt-in via explicit CLI flag
- Normal mode preserves streaming (no memory accumulation)
- Mitigated by splitting large files into chunks
- Future optimization: streaming array iterator (deferred to ADR 0006 outcomes)

**EOF document flows through pipeline**: Special `ContentEOF` document type injected by Conduit after source exhaustion. All processors receive it via Process(). Acceptable because:
- Leverages existing Processor interface (no new interface needed)
- Processors ignore EOF document by default (return empty or pass through)
- Only ArrayCollector cares about EOF (uses as emission signal)
- Natural document flow—no special batch API
- Consistent with existing document-oriented architecture

**Array collection silently drops documents**: `ArrayCollector.Process()` returns empty slice for normal documents—collected items don't flow through remaining pipeline until EOF. Acceptable because:
- Expected behavior for batch collection
- Alternative (buffering in pipeline) more complex
- Clear semantics: collection phase, then emission phase on EOF
- Only applies when --array flag explicit

**Format configuration in AST**: Adding `Format` field to `ForeachStepNode` couples workflow syntax to output serialization. Acceptable because:
- Formatting is intrinsic to foreach output (not separate concern)
- Three formats + delimiter cover common cases without excessive YAML complexity
- Extensible via interface without YAML changes
- Terminology correct: "format" is codec/serialization, not "reduce" function

**No nested array support**: Cannot handle arrays of arrays (e.g., `[[1,2], [3,4]]`) with flatten semantics. `selector: document` on nested array iterates outer array only. Acceptable because:
- Edge case, not in motivating use cases
- Workaround: use selector CEL expression to flatten
- Future enhancement without breaking changes

**Delimiter configuration limited**: Text formatter only supports single delimiter string. No support for prefix/suffix, padding, or complex separators. Acceptable because:
- Covers 95% of use cases (newline, double-newline, comma, etc.)
- Complex formatting can be done in post-processing step
- Future enhancement: template-based formatting

---

## Migration Path

**Phase 1: Core Implementation** (this ADR)
1. Add `--array` flag and `ArrayCollector` processor
2. Implement `Formatter` interface with three formats
3. Extend AST/parser for `format:` attribute
4. Update `ForeachStep` to use formatter
5. Add worker builder integration

**Phase 2: Documentation & Examples**
1. Update user guide with array mode examples
2. Create example workflows in `examples/13_array_input/`
3. Document format strategies in reference docs
4. Add troubleshooting section for memory usage

**Phase 3: Field Testing**
1. Test with real workflows (JSON APIs, document splitting)
2. Gather feedback on format strategies and delimiter needs
3. Identify ADR 0006 requirements based on usage patterns

**Phase 4: Optimization** (future ADR)
1. Lazy array evaluation (memory optimization if needed)
2. Streaming format (incremental serialization if needed)
3. Custom format functions (if use cases emerge)
4. JSONL auto-detection by file extension

---

## Open Questions

**Q1: Should `--array` auto-detect JSONL (newline-delimited JSON)?**

**Answer:** Deferred. Initial implementation requires explicit `--array` flag. JSONL auto-detection by file extension (`.jsonl`) can be added as enhancement after field testing, without breaking changes.

**Q2: What if user specifies `format:` but foreach doesn't produce array?**

**Answer:** Not an error. Format applied to empty array, produces empty result according to format type:
- `json`: `[]`
- `jsonl`: `""` (empty string)
- `text`: `""` (empty string)

**Q3: Should `selector: document` be implicit in array mode?**

**Answer:** No. Explicit better than implicit. User must write:
```yaml
- foreach:
    selector: document  # Required
    job: process
```
This makes array source clear in workflow definition.

**Q4: How does this interact with `--merge` flag?**

**Answer:** Incompatible combination. CLI validation:
```go
if opts.merge && opts.array {
    return fmt.Errorf("--merge and --array are mutually exclusive")
}
```
- `--merge`: multiple files → one text document (Union source)
- `--array`: multiple files/chunks → array (ArrayCollector processor)

**Q5: Can foreach use `selector: input` instead of `selector: document`?**

**Answer:** Yes, but semantically equivalent in array mode. Both `document` and `input` initially contain the same array value. Convention: use `document` for clarity (indicates original workflow input).

---

## Implementation Checklist

- [ ] Add `ContentEOF` constant to iosystem types (`internal/iosystem/types.go`)
- [ ] Add `--array` CLI flag (`cmd/opts.go`)
- [ ] Implement `ArrayCollector` processor with EOF detection (`internal/iosystem/processor/collector.go`)
- [ ] Inject EOF document in Conduit.runSequential() after source exhaustion (~3 lines)
- [ ] Create `Formatter` interface and implementations (`internal/blueprint/compiler/formatter.go`)
- [ ] Extend AST with `Format` field and `FormatNode` (`internal/blueprint/ast/ast.go`)
- [ ] Update parser for `format:` attribute (`internal/blueprint/parser/parser.go`)
- [ ] Modify `ForeachStep` to use formatter (`internal/blueprint/compiler/workflow.go`)
- [ ] Update compiler to create formatters (`internal/blueprint/compiler/compiler.go`)
- [ ] Add `ArrayMode()` to worker builder (`internal/service/worker/worker.go`)
- [ ] Add CLI validation for flag conflicts (`--merge` vs `--array`)
- [ ] Write unit tests for EOF document handling
- [ ] Write unit tests for all three formatters
- [ ] Write unit tests for ArrayCollector with EOF signal
- [ ] Write integration tests for array mode with various input types
- [ ] Update user documentation with array mode examples
- [ ] Create example workflows (`examples/13_array_input/`)
- [ ] Add troubleshooting guide for memory considerations

---

## Success Criteria

1. ✅ JSON array file processed via `--array` flag
2. ✅ Split document chunks aggregated with `--splitter --array`
3. ✅ Foreach format strategies (json/jsonl/text) work correctly
4. ✅ Text format delimiter configuration works
5. ✅ Normal mode (without `--array`) maintains streaming behavior
6. ✅ Backward compatibility: existing workflows unchanged
7. ✅ No architectural rewrites (extension only via Conduit/Processor)
8. ✅ Clear error messages for invalid configurations
9. ✅ Memory usage acceptable for array mode, efficient for streaming mode

---

## Design Rationale: Monadic Processor Interface

**Challenge Addressed:** Initial designs had:
1. Weak "ready" signal (GetArray/IsEmpty checks)
2. Abstraction leaks (AgentProcessor checking for ArrayCollector)
3. **Double encoding/decoding** - ArrayCollector JSON-encodes array, AgentProcessor JSON-decodes it

**Solution:** Made `Processor.Process()` monadic - accepting and returning document arrays:

```go
// OLD: Process(ctx context.Context, doc *Document) ([]*Document, error)
// NEW: Process(ctx context.Context, docs []*Document) ([]*Document, error)
```

Combined with `ContentEOF` document type for end-of-stream signal. This provides:

1. **Zero Double Encoding**: ArrayCollector emits `[]*Document` directly, AgentProcessor receives `[]*Document` directly
   - OLD flow: Documents → ArrayCollector → `json.Marshal(items)` → JSON bytes → AgentProcessor → `json.Unmarshal()` → items
   - NEW flow: Documents → ArrayCollector → `[]*Document` → AgentProcessor → read documents
   - Eliminates wasteful marshal/unmarshal round-trip

2. **Clean EOF Signal**: EOF document flows through monadic interface naturally

3. **No New Interface**: Modified existing `Processor` interface to be monadic

4. **No Abstraction Leaks**: AgentProcessor receives arrays via standard interface, no special checks

5. **Monadic Composition**: All processors compose naturally:
   - Normal: `Process([doc]) → [doc]` (identity for most processors)
   - Split: `Process([doc]) → [doc1, doc2, ...]` (flatMap)
   - Collect: `Process([doc]) → []` (collect), then `Process([EOF]) → [doc1, doc2, ...]` (emit)
   - Filter: `Process([docs]) → []` (drop)

**Pipeline Flow:**

```
Array Mode:
  Source → doc → Conduit: ArrayCollector.Process([doc]) → [] (collect)
  Source → doc → Conduit: ArrayCollector.Process([doc]) → [] (collect)
  Source EOF
  ↓
  Conduit → EOF doc → ArrayCollector.Process([EOF]) → [doc1, doc2, ...] (emit array)
  ↓
  AgentProcessor.Process([doc1, doc2, ...]) [array input, NO decoding!]
  ↓
  Result → Sink

Normal Mode:
  Source → doc → Conduit: Processor1.Process([doc]) → [doc]
  → Processor2.Process([doc]) → [doc] → Sink
  (streaming, single-element arrays)
```

**Benefits:**
- **Performance**: No double encoding/decoding of JSON arrays
- **Simplicity**: Single Processor interface, monadic composition
- **Consistency**: Documents flow naturally as arrays through pipeline
- **Minimal change**: Only signature change, behavior consistent
- **Backward compatible**: Existing processors work with single-element arrays
