# ADR 0006: Key/Value I/O System with Path-Based Keys

**Status:** Analysis (Decision Pending)  
**Date:** 2026-01-02  
**Context:** Refactoring I/O system to support intermediate results, skip-if-exists, and unified multi-stage workflows

---

## Context and Problem Statement

The current I/O system (ADR 0002) uses file paths as document identities and tightly couples processing to filesystem operations. This creates several limitations:

1. **No intermediate result preservation**: Multi-stage workflows (chunk → process → summarize → facts → validate → research) require multiple pipeline runs orchestrated via bash scripts
2. **No skip-if-exists**: Failed pipelines must reprocess expensive LLM operations from scratch
3. **Limited storage backends**: Filesystem and S3 using github.com/fogfish/stream but no other key/value stores
4. **Complex orchestration**: Workflows requiring 4-6 separate pipeline executions cannot be unified
5. **No explicit output control**: Steps cannot define where their output is written

**Motivating Use Case:**
```
1. Large document (10-20KB) → split into 1KB chunks (ADR 0005)
2. Process each chunk with LLM
3. Merge chunks → write summary file
4. Summary → extract N facts (1:N expansion)
5. Summary → validate uncertainty (parallel branch)
6. Each fact → make research (N:N processing)
```

Current implementation requires 4 separate pipelines with bash orchestration. Goal: single unified workflow.

---

## Decision Drivers

- **Unified workflows**: Support multi-stage processing in single workflow execution
- **Intermediate results**: Preserve outputs at any stage for inspection, debugging, and skip logic
- **Skip-if-exists**: Avoid expensive LLM reprocessing by checking cached results
- **Simple path-based keys**: Use filesystem paths as keys
- **Storage abstraction**: Pure key/value interface supporting FS, S3, and future backends
- **Explicit output control**: Each step declares where output is written via `emit:` attribute
- **Source responsibility**: Key construction isolated to source layer, pipeline remains agnostic
- **Cloud productization**: Enable distributed processing with shared storage (S3)

---

## Decision

### **Core Architecture: Pure Key/Value Storage with Path-Based Keys**

Use simple filesystem-style paths as document keys, with pure key/value storage abstraction implemented via `github.com/fogfish/stream` for local filesystem and S3. Each processing step explicitly declares output via `emit:` attribute.

---

## Design

### **1. Simple Path-Based Keys**

**Key Structure: Filesystem Path (Relative to Base)**

```go
package iosystem

// Key is a simple filesystem-style path
// Paths are relative to storage base directory
type Key string

// Examples:
// "a.txt"           - file in root
// "sub/a.txt"       - file in subdirectory
// "summary/a.txt"   - file with prefix from emit
```

**Key Semantics:**
- **Path**: Simple string path, filesystem-compatible
- **Relative**: Always relative to storage base directory
- **Separator**: Forward slash `/` (portable across systems)

**Examples:**

```
Input filesystem (base: ./input):
  ./input/a.txt           → Key: "a.txt"
  ./input/sub/a.txt       → Key: "sub/a.txt"
  ./input/sub/b.txt       → Key: "sub/b.txt"

After step with emit: "summary":
  Input:  "a.txt"
  Output: "summary/a.txt"

After step with emit: "summary" on subdirectory file:
  Input:  "sub/a.txt"
  Output: "summary/sub/a.txt"

Within foreach the sequence (itteration id is recorded, it is automatically added to the key extension, sequence id are stacked for the nested foreach )
After foreach processing array with emit: "posts":
  Input:  "facts/summary/a.txt" (each element)
  Output: "research/theories/summary/a.seq-0001.txt"
  Output: "research/theories/summary/a.seq-0002.txt"
  Output: "research/theories/summary/a.seq-0003.txt"
```

**Benefits:**
- ✅ Simple, intuitive path structure
- ✅ Filesystem-compatible (no URN parsing)
- ✅ Natural prefix matching for batch operations
- ✅ Easy debugging (paths are human-readable)
- ✅ No external dependencies

---

### **2. Document Structure**

```go
package iosystem

// Document represents a processing unit with identity and content
type Document struct {
    // Key is the simple path-based identity
    Key Key  // string type: "sub/a.txt"
    
    // Metadata holds document attributes (separate from identity)
    Metadata Metadata
    
    // Reader provides streaming access (auto-closing)
    Reader io.ReadCloser
}

// Metadata holds document attributes (NOT part of key identity)
type Metadata struct {
    ContentType string            // MIME type (e.g., "text/plain", "application/json")
    Extension   string            // File extension for output (e.g., ".txt", ".md")
    Size        int64             // Document size in bytes
    Custom      map[string]string // User-defined attributes
}

func NewDocument(key Key, reader io.ReadCloser) *Document {
    return &Document{
        Key:      key,
        Metadata: Metadata{Custom: make(map[string]string)},
        Reader:   reader,
    }
}
```

**Design Principles:**
- Identity (Key) is simple string path
- Auto-closing reader prevents resource leaks
- Metadata includes file extension, not embedded in key
- Content type determined from extension or explicitly set

---

### **3. Pure Key/Value Storage Interface**

```go
package storage

import (
    "context"
    "io"
    "github.com/fogfish/iq/internal/iosystem"
)

// Storage provides pure key/value operations
// Implementations use github.com/fogfish/stream for FS and S3
type Storage interface {
    // Put writes value to key (overwrites if exists)
    Put(ctx context.Context, key iosystem.Key, value io.Reader) error
    
    // Get reads value from key (returns auto-closing reader)
    Get(ctx context.Context, key iosystem.Key) (io.ReadCloser, error)
        
    // Has checks if key exists (cheap operation for skip-if-exists logic)
    Has(ctx context.Context, key iosystem.Key) (bool, error)

    // Walk all keys matching prefix pattern
    Walk(ctx context.Context, prefix iosystem.Key, visitor func(Document) error) error
}
```

**Implementation via `github.com/fogfish/stream`:**

```go
// FSStorage wraps stream library for filesystem and S3
type FSStorage struct {
    fs stream.CreateFS[struct{}]
}

func NewFSStorage(path string) (*FSStorage, error) {
    // Supports local paths and s3:// URLs via stream library
    fs := stream.NewFS[struct{}](path)
    return &FSStorage{fs: fs}, nil
}

func (s *FSStorage) Put(ctx context.Context, key iosystem.Key, value io.Reader) error {
    // Key is already a path, use directly
    file, err := s.fs.Create(string(key), nil)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = io.Copy(file, value)
    if err != nil {
        file.Cancel()
        return err
    }
    return nil
}

func (s *FSStorage) Get(ctx context.Context, key iosystem.Key) (io.ReadCloser, error) {
    // Key is already a path, use directly
    return s.fs.Open(string(key))
}

func (s *FSStorage) Has(ctx context.Context, key iosystem.Key) (bool, error) {
    _, err := s.fs.Stat(string(key))
    if err != nil {
        if os.IsNotExist(err) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (s *FSStorage) Walk(ctx context.Context, prefix iosystem.Key, visitor func(Document) error) error {
    // Use stream library's prefix matching
    var keys []iosystem.Key
    
    // Walk filesystem with prefix
    err := fs.WalkDir(s.fs, string(prefix), func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if !d.IsDir() {
            keys = append(keys, iosystem.Key(path))
        }

        return visitor()
    })
    
    return keys, err
}
```

**Supported Backends:**
- ✅ Local filesystem (via `stream`)
- ✅ S3 (via `stream` with `s3://` prefix)
- ⏸️ DynamoDB (deferred until FS/S3 validated)

---

### **4. Source: Key Construction Responsibility**

**Source is solely responsible for constructing document keys.** Pipeline and processors remain key-agnostic.

```go
package source

// FSSource walks filesystem and constructs simple path keys
type FSSource struct {
    storage   Storage
    baseDir   string
    recursive bool
    pattern   string // Optional glob pattern (Issue #13)
}

func NewFS(storage Storage, baseDir string, opts ...Option) (*FSSource, error) {
    return &FSSource{
        storage:   storage,
        baseDir:   baseDir,
        recursive: false,
    }, nil
}

// WithRecursive enables recursive directory traversal (Issue #18)
func WithRecursive() Option {
    return func(s *FSSource) {
        s.recursive = true
    }
}

// WithPattern sets file glob pattern (Issue #13)
func WithPattern(pattern string) Option {
    return func(s *FSSource) {
        s.pattern = pattern
    }
}

// Next yields documents with properly constructed path keys
func (s *FSSource) Next(ctx context.Context) (*iosystem.Document, error) {
    // Walk filesystem
    fsPath := s.nextFile() // e.g., "/base/sub/a.txt"
    
    // Construct key from filesystem path (relative to base)
    key := s.fsPathToKey(fsPath)
    
    // Read content from storage
    reader, err := s.storage.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    
    return iosystem.NewDocument(key, reader), nil
}

// fsPathToKey converts filesystem path to relative key
// "/base/sub/a.txt" → "sub/a.txt"
// "/base/a.txt" → "a.txt"
func (s *FSSource) fsPathToKey(fsPath string) iosystem.Key {
    clean := filepath.Clean(fsPath)
    
    // Remove base directory prefix
    relPath, err := filepath.Rel(s.baseDir, clean)
    if err != nil {
        // Fallback: strip prefix manually
        relPath = strings.TrimPrefix(clean, s.baseDir)
        relPath = strings.TrimPrefix(relPath, "/")
    }
    
    // Convert to forward slashes (portable)
    relPath = filepath.ToSlash(relPath)
    
    return iosystem.Key(relPath)
}
```

**Key Characteristics:**
- Source owns key construction logic
- Filesystem paths directly map to relative keys
- Pipeline processes documents without key knowledge
- Supports recursive traversal (Issue #18)
- Supports glob patterns (Issue #13)

---

### **5. Emit: Step-Level Output Control**

**Emit is an attribute on any processing step** that defines the output key prefix.

**Workflow Syntax:**

```yaml
jobs:
  process:
    steps:
      # Step 1: Process files with emit prefix
      - uses: prompts/summarize.md
        emit: summary      

      # Step 2: Extract facts (1:N expansion)
      - uses: prompts/facts.md
        emit: theories

      # Step 3: Process each theory with foreach
      - foreach:
          job: research
          emit: posts
```

**Emit Semantics:**

```go
package blueprint

// Step represents a workflow step with optional emit
type Step struct {
    Name      string
    Emit      string     // Optional: output key prefix
    Processor Processor
    Foreach   *Foreach   // Optional: array processing
}

// ProcessStep executes step and applies emit prefix
func (w *Workflow) ProcessStep(ctx context.Context, step *Step, doc *iosystem.Document) error {
    // Execute processor
    output, err := step.Processor.Process(ctx, doc)
    if err != nil {
        return err
    }
    
    // Apply emit prefix to output key
    outputKey := w.applyEmit(step.Emit, doc.Key)
    
    // Write to storage
    return w.storage.Put(ctx, outputKey, output.Reader)
}

// applyEmit adds emit prefix to input key
// emit="summary", key="sub/a.txt" → "summary/sub/a.txt"
// emit="", key="sub/a.txt" → "sub/a.txt" (no change)
func (w *Workflow) applyEmit(emit string, key iosystem.Key) iosystem.Key {
    if emit == "" {
        return key
    }
    return iosystem.Key(emit + "/" + string(key))
}
```

**Emit + Foreach:**

When `foreach` processes arrays, emit prefix is applied to each element:

```go
// ProcessForeach handles array processing with emit
func (w *Workflow) ProcessForeach(ctx context.Context, step *Step, arrayKey iosystem.Key) error {
    // Read JSONL array (see ADR 0006)
    array, err := w.readArray(ctx, arrayKey)
    if err != nil {
        return err
    }
    
    // Process each element
    for i, elem := range array {
        // Process element
        output, err := step.Foreach.Processor.Process(ctx, elem)
        if err != nil {
            return err
        }
        
        // Construct output key with emit prefix and array index
        // emit="posts", arrayKey="theories/summary/a.txt", i=0
        // → "posts/theories/summary/a.seq-0001.txt"
        outputKey := w.applyEmitWithIndex(step.Emit, arrayKey, i)
        
        // Write to storage
        err = w.storage.Put(ctx, outputKey, output.Reader)
        if err != nil {
            return err
        }
    }
    
    return nil
}

// applyEmitWithIndex adds emit prefix and array index
// emit="posts", key="theories/summary/a.txt", index=0
// → "posts/theories/summary/a.seq-0001.txt"
func (w *Workflow) applyEmitWithIndex(emit string, key iosystem.Key, index int) iosystem.Key {
    suffix := fmt.Sprintf("-%06d", index+1)
    
    if emit == "" {
        return iosystem.Key(string(key) + suffix)
    }
    
    return iosystem.Key(emit + "/" + string(key) + suffix)
}
```

**Benefits:**
- ✅ Simple attribute syntax (not separate directive)
- ✅ Explicit output control per step
- ✅ No complex key derivation logic
- ✅ Natural foreach integration
- ✅ Each emit defines unique key space

---

### **6. Skip-If-Exists: CLI Flag**

**Skip-if-exists is implemented as a CLI flag**, not a workflow directive. It checks if output keys exist before processing.

**CLI Usage:**

```bash
# Run workflow, skip documents with existing output
iq run workflow.yml --skip-if-exists

# Without flag: always process (overwrite existing)
iq run workflow.yml
```

**Implementation:**

```go
package cmd

// RunCommand executes workflow with optional skip-if-exists
type RunCommand struct {
    WorkflowPath string
    SkipIfExists bool  // Set via --skip-if-exists flag
}

func (c *RunCommand) Execute(ctx context.Context) error {
    // Load workflow
    workflow, err := blueprint.Load(c.WorkflowPath)
    if err != nil {
        return err
    }
    
    // Get input documents from source
    docs := workflow.Source.List(ctx)
    
    for _, doc := range docs {
        // Check skip-if-exists before processing
        if c.SkipIfExists {
            exists, err := c.checkOutputExists(ctx, workflow, doc.Key)
            if err != nil {
                return err
            }
            if exists {
                log.Printf("Skipping %s (output exists)", doc.Key)
                continue
            }
        }
        
        // Execute workflow
        err := workflow.Execute(ctx, doc)
        if err != nil {
            return err
        }
    }
    
    return nil
}

// checkOutputExists verifies if anchor output key exists
// Anchor key = emit prefix from LAST step in workflow
func (c *RunCommand) checkOutputExists(ctx context.Context, workflow *blueprint.Workflow, inputKey iosystem.Key) (bool, error) {
    // Get last step (anchor)
    lastStep := workflow.Jobs[0].Steps[len(workflow.Jobs[0].Steps)-1]
    
    // Compute expected output key
    outputKey := c.computeOutputKey(lastStep, inputKey)
    
    // Check storage
    return workflow.Storage.Has(ctx, outputKey)
}

// computeOutputKey calculates output key for anchor step
func (c *RunCommand) computeOutputKey(step *blueprint.Step, inputKey iosystem.Key) iosystem.Key {
    // Handle foreach case (check for array file)
    if step.Foreach != nil {
        // Array output: emit/input.key (JSONL file)
        if step.Emit != "" {
            return iosystem.Key(step.Emit + "/" + string(inputKey))
        }
        return inputKey
    }
    
    // Regular step: apply emit prefix
    if step.Emit != "" {
        return iosystem.Key(step.Emit + "/" + string(inputKey))
    }
    
    return inputKey
}
```

**Benefits:**
- ✅ CLI-level control (not workflow concern)
- ✅ Simple anchor key check (last step emit)
- ✅ Cheap Has() operation (no full read)
- ✅ Enables incremental pipeline recovery

**Example:**

```yaml
# workflow.yml
jobs:
  process:
    steps:
      - name: summarize
        emit: summary
        processor: { type: llm, prompt: summarize.md }
      
      - name: theorize
        emit: theories
        processor: { type: llm, prompt: theorize.md }
      
      - name: post
        emit: posts        # ← Anchor step (last)
        foreach:
          processor: { type: llm, prompt: post.md }
```

```bash
# First run: processes all files
iq run workflow.yml

# Fails on file sub/b.txt at step "post"
# Some files have outputs: posts/summary/a.txt, posts/theories/a.txt

# Recovery run: skips files with existing anchor key
iq run workflow.yml --skip-if-exists
# Skips: a.txt (posts/a.txt exists)
# Processes: sub/b.txt (posts/sub/b.txt missing)
```

---

## Implementation Plan

### **Phase 1: Core Infrastructure (Week 1-2)**

**Goals:**
- ✅ Define Key as string type
- ✅ Implement Storage interface with FSStorage
- ✅ Update Document to use Key
- ✅ Implement source key construction

**Tasks:**
1. Define `Key` type as `string` in `internal/iosystem/types.go`
2. Implement `Storage` interface in `internal/storage/storage.go`
3. Implement `FSStorage` using `github.com/fogfish/stream`
4. Update `Document` struct to use new Key type
5. Refactor `FSSource` to construct relative path keys
6. Unit tests for storage operations

**Acceptance Criteria:**
- Source yields documents with correct relative path keys
- FSStorage Put/Get/Has/Match operations work with simple paths
- S3 storage works via stream library (`s3://` prefix)

### **Phase 2: Emit Attribute (Week 2)**

**Goals:**
- ✅ Add `Emit` attribute to Step
- ✅ Implement emit prefix logic in workflow execution
- ✅ Handle foreach + emit combination

**Tasks:**
1. Add `Emit string` field to `Step` struct
2. Implement `applyEmit()` key transformation
3. Update workflow executor to apply emit to outputs
4. Implement `applyEmitWithIndex()` for foreach arrays
5. Integration tests with emit prefix

**Acceptance Criteria:**
- Steps without emit write to input key location
- Steps with emit write to `emit/input-key` location
- Foreach with emit writes to `emit/array-key-NNNNNN`

### **Phase 3: Skip-If-Exists (Week 3)**

**Goals:**
- ✅ Implement --skip-if-exists CLI flag
- ✅ Implement anchor key checking logic
- ✅ Integrate with workflow execution

**Tasks:**
1. Add `--skip-if-exists` flag to run command
2. Implement `checkOutputExists()` anchor logic
3. Implement `computeOutputKey()` for last step
4. Handle foreach anchor case (array file check)
5. Integration tests with skip scenarios

**Acceptance Criteria:**
- `--skip-if-exists` skips documents with existing anchor
- Without flag, all documents processed (overwrite)
- Correct anchor computed for regular and foreach steps

### **Phase 4: Testing & Documentation (Week 3-4)**

**Goals:**
- ✅ Comprehensive integration tests
- ✅ Update user documentation
- ✅ Migration guide for existing workflows

**Tasks:**
1. Integration tests for multi-stage workflows
2. Test skip-if-exists recovery scenarios
3. Test S3 storage backend
4. Update user guide with emit examples
5. Document migration from old system
6. Performance testing with large file sets

**Acceptance Criteria:**
- All examples work with new system
- Documentation complete and accurate
- Performance comparable to current implementation

---

## Consequences

### **Positive**

- ✅ **Simple path-based keys**: No URN library dependency, easier debugging
- ✅ **Explicit output control**: Each step declares output via emit attribute
- ✅ **No complex derivation**: Emit prefix is added directly, no logic needed
- ✅ **Natural foreach integration**: Array processing with emit works intuitively
- ✅ **Simplified skip logic**: Anchor key = last step emit + input key
- ✅ **Storage abstraction**: Clean key/value interface via github.com/fogfish/stream
- ✅ **Filesystem semantics**: Paths work naturally with FS and S3
- ✅ **CLI-level skip**: Skip-if-exists orthogonal to workflow definition

### **Neutral**

- ⚖️ **Emit prefix accumulation**: Keys grow with workflow depth (e.g., `posts/theories/summary/a.txt`)
  - **Mitigation**: Natural workflow progression, human-readable
- ⚖️ **Foreach array indexing**: Suffix format `-NNNNNN` chosen for uniqueness
  - **Mitigation**: Consistent with filesystem semantics (no URI fragments)

### **Negative**

- ❌ **No hierarchical typing**: Keys are strings, no type-safe segments
  - **Mitigation**: Simplicity outweighs type safety for paths
- ❌ **Prefix-based matching only**: No semantic querying (e.g., "all theories")
  - **Mitigation**: Prefix matching sufficient for workflow patterns

---

## Related

- **ADR 0002**: I/O System (being superseded)
- **ADR 0004**: Intermediate Snapshots (emit solves this)
- **ADR 0006**: Document Splitting (split + emit + foreach integration)
- **Issue #18**: Recursive directory traversal
- **Issue #13**: File pattern selection
- **Issue #35**: Chunk path naming improvements
- **Issue #36**: Document identity unification
- **Issue #48**: Immediate chunk emission

---

## References

- [github.com/fogfish/stream](https://github.com/fogfish/stream) - FS/S3 abstraction library
- ADR 0006 - Document Splitting (foreach array processing patterns)
