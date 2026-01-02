# ADR 0005: Document Splitting and Collection Processing

**Status:** Proposed  
**Date:** 2026-01-02  
**Context:** Standardizing document splitting from CLI/processor to workflow directive with array-based processing

---

## Context and Problem Statement

Currently, document splitting (chunking) has inconsistent implementations across different parts of the system:

1. **CLI-level chunking**: `--splitter` flag prepares arrays before pipeline execution
2. **Processor-level chunking**: `ChunkerProcessor` in pipeline splits documents during processing
3. **No explicit collection handling**: After splitting, no clear pattern for collecting/merging results

This creates several problems:

- **Inconsistent semantics**: Split can happen at CLI (pre-pipeline) or processor (mid-pipeline)
- **No standardized collection**: Split outputs must be manually collected/merged
- **Complex orchestration**: Multi-stage workflows requiring split → process → merge need multiple pipeline runs
- **Unclear ownership**: Split responsibility unclear (CLI vs pipeline vs workflow)

**Motivating Use Case:**
```
Large document (20KB) → split into 1KB chunks → process each chunk with LLM → collect results
```

---

## Decision Drivers

- **Single responsibility**: Split should have one canonical implementation location
- **Workflow-level control**: Split configuration belongs in workflow definition, not CLI. It shields workflow for the overhead. 
- **Array semantics**: Split produces array, foreach processes array (map+reduce pattern)
- **Backward compatibility**: Existing `--splitter` CLI flag must continue working
- **Clear ownership**: Workflow defines split, source/processor/CLI only execute
- **Composability**: Split + foreach should compose naturally with other workflow constructs

---

## Decision

### **Split as Workflow Directive with Array Output**

Split becomes a **first-class workflow step** that produces an array of documents. The array is then processed using existing `foreach` construct for map+reduce operations.

---

## Design

### **1. Split Step in Workflow**

**Syntax:**

```yaml
jobs:
  process-large-doc:
    steps:
      # Split step: produces array of document chunks
      - split:
          strategy: paragraph  # or: sentence, chunk, tag
          size: 1024          # chunk size in bytes (for chunk strategy)
        output: chunks        # Variable name for array
      
      # Foreach: map+reduce over chunks (using the selector feature)
      - foreach:
          selector: chunks
          job: analyze
        output: analyzed      # Collected array of results
      
      # Next step receives array
      - uses: prompts/summarize.md
        # Template can access: {{.current}} = array
```

**Split Strategies:**
- `none`: No splitting (pass-through)
- `sentence`: Split on sentence boundaries, the split tokes are configurable, by default is .!?
- `paragraph`: Split on paragraph breaks, the split tokes are configurable, by default is new line.
- `chunk`: Split into fixed-size chunks with overlap
- `tag`: Split on custom delimiter tags

---

### **2. Split Execution Model**

**Split is in-memory operation:**

```go
package compiler

// SplitStep splits document into array of chunks
type SplitStep struct {
    strategy string
    size     int
    overlap  int    // For chunk strategy
}

func (s *SplitStep) Execute(ctx context.Context, doc *Document) ([]Document, error) {
    // Read document content
    content, err := io.ReadAll(doc.Reader)
    if err != nil {
        return nil, err
    }
    
    // Split content based on strategy
    parts := s.splitContent(content)
    
    // Create array of documents (in-memory, no key changes)
    chunks := make([]Document, len(parts))
    for i, part := range parts {
        chunks[i] = Document{
            Key:      doc.Key,  // All chunks share same key
            Metadata: doc.Metadata,
            Reader:   io.NopCloser(bytes.NewReader(part)),
        }
        // Add chunk metadata
        chunks[i].Metadata.Custom["chunk_index"] = strconv.Itoa(i)
        chunks[i].Metadata.Custom["chunk_total"] = strconv.Itoa(len(parts))
    }
    
    return chunks, nil
}

func (s *SplitStep) splitContent(content []byte) [][]byte {
    switch s.strategy {
    case "paragraph":
        return s.splitByParagraph(content)
    case "sentence":
        return s.splitBySentence(content)
    case "chunk":
        return s.splitBySize(content)
    case "tag":
        return s.splitByTag(content)
    default:
        return [][]byte{content} // No split
    }
}
```

**Key Characteristics:**
- ✅ Chunks are **in-memory arrays**, not individual files/keys
- ✅ All chunks **share same URN key** (no key derivation)
- ✅ Chunk metadata stored in `Metadata.Custom` map
- ✅ Split is **stateless** (pure function: document → array)

---

### **3. Foreach = Map + Reduce**

**Existing foreach handles array processing:**

```yaml
- foreach:
    selector: chunks
    job: analyze
  output: analyzed
```

**Foreach Semantics:**
1. **Input**: Array of documents from `split` output
2. **Map**: Execute `job` steps on each document independently
3. **Reduce**: Collect all results into output array, the reduce strategy is configurable. It reduce results either to json, jsonnl or plain text. 
4. **Output**: Array passed to next step via `{{.current}}`

**Implementation (existing):**

```go
// ForeachStep processes array (map+reduce)
type ForeachStep struct {
    items  string   // Variable name: "chunks"
    steps  []Step   // Processing steps
    output string   // Output variable name
}

func (f *ForeachStep) Execute(ctx context.Context, wctx *WorkflowContext) error {
    // Get input array
    items := wctx.State[f.items].([]Document)
    
    // MAP: Process each item
    results := make([]Document, len(items))
    for i, item := range items {
        result := item
        for _, step := range f.steps {
            var err error
            result, err = step.Execute(ctx, result)
            if err != nil {
                return fmt.Errorf("foreach item %d: %w", i, err)
            }
        }
        results[i] = result
    }
    
    // REDUCE: Store results array
    wctx.State[f.output] = results
    wctx.Current = results  // Also set as current for next step
    
    return nil
}
```

---

### **5. Compatibility Mode: CLI-Level Split**

**Current behavior with `--splitter` flag:**

```bash
# Current implementation
iq agent batch -f workflow.yml -I ./input -O ./output --splitter paragraph
```

**What happens now:**
1. CLI reads input file
2. CLI splits into chunks (using `github.com/fogfish/scanner`)
3. CLI feeds chunks into pipeline as separate documents
4. Each chunk processed independently
5. Each chunk written as separate output file
6. The file name of output file is file.seq-0000.txt

**Compatibility Mode (Deprecated):**

```go
package cmd

func executeBatch(cmd *cobra.Command, args []string) error {
    // Check for legacy --splitter flag
    if splitterStrategy != "" {
        // DEPRECATED: Print warning
        log.Warn("--splitter flag is deprecated, use workflow split: directive")
        
        // Create implicit split in workflow
        workflow = wrapWithSplit(workflow, splitterStrategy)
    }
    
    // Execute workflow normally
    return executeWorkflow(workflow)
}

// wrapWithSplit injects split step at beginning of first job
func wrapWithSplit(workflow *Workflow, strategy string) *Workflow {
    // Insert split step before first step in main job
    splitStep := &SplitStep{
        strategy: strategy,
        output:   "_chunks",
    }
    
    // Wrap remaining steps in foreach
    foreachStep := &ForeachStep{
        items:  "_chunks",
        steps:  workflow.Jobs["main"].Steps,
        output: "results",
    }
    
    workflow.Jobs["main"].Steps = []Step{splitStep, foreachStep}
    return workflow
}
```

**Migration Path:**

```bash
# Old (deprecated, still works)
iq agent batch -f workflow.yml --splitter paragraph

# New (recommended)
# Add split: to workflow.yml instead
```

**Deprecation Timeline:**
- v0.2: Add warning for `--splitter` flag
- v0.3: Remove `--splitter` flag support
- Users must migrate to workflow-level `split:`

---

## Complete Example

**Workflow with Split:**

```yaml
name: analyze-large-document

jobs:
  main:
    steps:
      # Step 1: Split document into chunks
      - split:
          strategy: paragraph
          size: 1024
        output: chunks
      
      # Step 2: Process each chunk (map+reduce)
      - foreach:
          selector: chunks
          job: analyze
        output: analyzed
      
      # Step 3: Emit chunked results (optional checkpoint)
      - emit: chunk-analysis
      
      # Step 4: Summarize all chunks
      - uses: prompts/summarize-all.md
        output: summary
      
      # Step 5: Emit final summary
      - emit: summary

  analyze:
    - uses: prompts/analyze-chunk.md
```

**Prompt Template for Array Input:**

```markdown
---
format: text
schema:
  input:
    type: array
    items:
      type: object
      properties:
        content: {type: string}
        metadata: {type: object}
---
Create a summary from these analyzed chunks:

{{range $i, $chunk := .current}}
## Chunk {{$i}}
{{$chunk.content}}

{{end}}

Provide a cohesive summary combining insights from all chunks.
```

**Execution Flow:**

```
Input: urn:iq:doc:document.txt (20KB)

Step 1 (split):
  Output: [chunk0, chunk1, chunk2, ...] (in-memory array)

Step 2 (foreach):
  Map: Process each chunk with LLM
  Reduce: Collect results → [result0, result1, result2, ...]

Step 3 (emit):
  Write: urn:iq:doc:chunk-analysis:document.txt (JSONL array)

Step 4 (summarize):
  Input: array from foreach
  Output: single summary document

Step 5 (emit):
  Write: urn:iq:doc:summary:chunk-analysis:document.txt
```

**Output Files:**

```
./output/
  chunk-analysis/document.txt     # JSONL: all chunk analyses
  summary/chunk-analysis/document.txt  # Final summary
```

---

## Implementation Plan

### Phase 1: AST and Parser (1-2 days)
1. Add `SplitStepNode` to AST (`internal/blueprint/ast/`)
2. Extend parser to recognize `split:` directive
3. Add validation rules (strategy required, size for chunk strategy)
4. Add tests for split syntax parsing

### Phase 2: Compiler (2-3 days)
1. Implement `SplitStep` in compiler
2. Integrate `github.com/fogfish/scanner` for splitting strategies
3. Add array type handling in workflow context
4. Update `ForeachStep` to accept arrays from split
5. Add tests for split execution

### Phase 3: Emit Array Support (1-2 days)
1. Update `EmitStep` to detect array input
2. Implement JSONL serialization for arrays
3. Add tests for array emit/load roundtrip

### Phase 4: CLI Compatibility Mode (1 day)
1. Add deprecation warning for `--splitter` flag
2. Implement `wrapWithSplit()` for backward compatibility
3. Update CLI help text with migration guidance
4. Add tests for compatibility mode

### Phase 5: Documentation (1-2 days)
1. Update user guide with split examples
2. Add migration guide for `--splitter` users
3. Update all examples to use split directive
4. Document array processing patterns

**Total Implementation Time:** ~1.5 weeks

---

## Alternatives Considered

### Alternative 1: Keep Split as CLI Flag Only

**Rejected:** Cannot support complex multi-stage workflows where split configuration varies per job. Also, split configuration belongs in workflow definition for version control.

### Alternative 2: Split as Processor in Pipeline

**Rejected:** Processors should be stateless transformations. Split creates array semantic that requires special handling. Workflow directive is more explicit.

### Alternative 3: Automatic Split Based on File Size

**Rejected:** Implicit behavior is magical and unpredictable. Explicit workflow directive is clearer and more maintainable.

### Alternative 4: Split Creates Individual Keys

**Rejected:** Creates complex collection completion signaling problem. Array semantic is simpler and atomic.

---

## Consequences

### Positive

✅ **Single source of truth**: Split defined in workflow, not scattered across CLI/processor  
✅ **Explicit control**: User explicitly defines split strategy per job  
✅ **Array semantics**: Natural map+reduce pattern with foreach  
✅ **Atomic collections**: Array = single unit, no partial results  
✅ **Composability**: Split + foreach + emit compose cleanly  
✅ **Version controlled**: Split configuration in workflow YAML  
✅ **Testability**: Split step independently testable  
✅ **Backward compatible**: CLI flag still works (deprecated)

### Negative

⚠️ **Breaking change**: Users with `--splitter` must migrate (deprecation path provided)  
⚠️ **Memory usage**: Large documents with many chunks held in memory as arrays  
⚠️ **Learning curve**: Users must understand split + foreach pattern  
⚠️ **Workflow verbosity**: Simple split requires 2 steps (split + foreach)

### Mitigation

- Provide clear migration guide with before/after examples
- Add memory usage warnings to documentation
- Create workflow templates for common split patterns
- Consider streaming chunking for very large documents (future enhancement)

---

## Related ADRs

- **ADR 0001**: Workflow Blueprint Architecture (workflow step types)
- **ADR 0002**: I/O System (document processing pipeline)
- **ADR 0005**: Key/Value I/O System (URN identities, emit directive)

---

## Open Questions

1. **Memory limits**: Should split enforce maximum chunk count or total size?
2. **Streaming split**: For 100MB+ documents, should we support streaming chunks instead of arrays?
3. **Parallel foreach**: Should foreach support parallel processing of chunks (with concurrency limit)?
4. **Custom split strategies**: Should users define custom split functions in workflow?

---

## Acceptance Criteria

- [ ] `split:` directive parsed in workflow YAML
- [ ] `SplitStep` implemented with strategies: sentence, paragraph, chunk, tag
- [ ] Split produces array of documents
- [ ] Foreach processes arrays from split
- [ ] Emit serializes arrays as JSONL
- [ ] CLI `--splitter` flag deprecated with warning
- [ ] Compatibility mode wraps legacy split usage
- [ ] Documentation updated with split examples
- [ ] Migration guide published
- [ ] All existing tests pass with new split implementation
