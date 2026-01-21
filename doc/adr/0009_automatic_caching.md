# 0009 - Automatic Caching System

## In the context of

Building an LLM-powered workflow system where multiple steps process documents through expensive API calls. Users need to minimize costs and improve execution speed when running workflows repeatedly or iterating on workflow development. The system must cache LLM responses transparently without requiring manual configuration in workflow YAML files.

## Facing Concern

**Cost efficiency**: Every LLM API call incurs costs. During development, testing, or batch processing, workflows may process the same inputs multiple times with identical prompts, leading to unnecessary expenses.

**Developer experience**: Explicit cache key management in YAML files creates cognitive overhead and boilerplate. Users must manually decide what to cache and maintain cache keys across workflow updates.

**Cache invalidation complexity**: When prompts change, caches with manually-specified keys remain valid, potentially returning stale results. Users must remember to update cache keys when modifying prompts.

**Iteration context handling**: Foreach loops create multiple executions of the same step with different inputs. Each iteration needs unique cache entries, but the system already has a Key mechanism (`in.Key`) that tracks iteration context.

**Step type diversity**: Different step types (Agent, Router, Foreach, Run) have different caching needs:
- Agent steps always execute prompts → cacheable
- Router steps may or may not execute prompts → conditionally cacheable
- Foreach steps are containers → not directly cacheable (nested jobs are cached)
- Run steps execute shell commands → not cacheable (cheap + side effects)

## We decided for

**Automatic caching with content-based invalidation and single-flag activation**:

### Core Principles

1. **Opt-in via CLI flag**: Caching is controlled entirely by `--cache-dir` flag. No YAML configuration needed.
2. **Automatic wrapping**: Compiler automatically wraps cacheable steps with Cache decorator when cache storage is available.
3. **Content-based keys**: Cache keys include SHA256 hash (6 hex chars) of prompt content for automatic invalidation.
4. **Key reuse**: Use existing `iosystem.Key` mechanism that already handles foreach iteration context via `SeqID`.

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│  CLI Layer (cmd/)                                        │
│  --cache-dir flag → Creates cache storage                │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│  Compiler (internal/blueprint/compiler/)                 │
│  - Checks if c.cache != nil                              │
│  - Extracts prompt content for cacheable steps           │
│  - Computes SHA256 hash (first 6 hex chars)              │
│  - Wraps step with Cache(workflow, job, step, hash, .)   │
│  - Pass-through for non-cacheable steps                  │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────────────┐
│  Runtime Cache Decorator (internal/blueprint/runtime/)   │
│  - Stores: workflow, job, step, hash, cache, Prompter   │
│  - generateKey(): workflow/job/step-hash/in.Key          │
│  - Prompt(): Check cache → Hit: return | Miss: execute   │
│  - Storage: .md files with YAML front matter            │
└──────────────────────────────────────────────────────────┘
```

### Step Type Handling

```go
// Compiler determines cacheability
func (c *Compiler) extractStepContent(tree *ast.AST, node ast.StepNode) string {
    switch v := node.(type) {
    case *ast.AgentStepNode:
        // Always has uses (required) → return agent.Prompt
        if agent, ok := tree.Agents[v.Uses]; ok {
            return agent.Prompt
        }
    case *ast.RouterStepNode:
        // Only cacheable if has uses (router agent)
        if v.Uses != "" {
            if agent, ok := tree.Agents[v.Uses]; ok {
                return agent.Prompt
            }
        }
    // ForeachStepNode, RunStepNode → return ""
    }
    return ""
}
```

### Cache Key Structure

**Format**: `workflow/job/step-name-hash/doc-key`

**Components**:
- `workflow`: From Blueprint `name` field
- `job`: Current job name
- `step-name`: YAML `name` field or `step-{index}` default
- `hash`: First 6 hex chars of SHA256(prompt content)
- `doc-key`: Input document key (contains `.SeqID(i+1)` for foreach)

**Examples**:
```
research/main/extract-a3f2b1/report.txt
research/main/step-2-c4d5e6/data.json
pipeline/process/transform-7e8f9a/docs.0001.txt  ← foreach iteration
```

### Iteration Handling (Foreach)

**No explicit iteration context needed**. The existing Key mechanism automatically handles this:

1. **Foreach runtime** creates unique keys per iteration:
   ```go
   // in foreach.go
   key := in.Key.SeqID(i + 1)  // docs.txt → docs.0001.txt
   ```

2. **Cache uses Key directly**:
   ```go
   // in cache.go
   func (c *Cache) generateKey(docKey iosystem.Key) iosystem.Key {
       stepKey := fmt.Sprintf("%s-%s", c.step, c.hash)
       return iosystem.Key(filepath.Join(c.workflow, c.job, stepKey, string(docKey)))
   }
   ```

3. **Nested foreach** works automatically via sequential SeqID calls:
   ```
   doc.txt → doc.0001.txt → doc.0001.0002.txt
   ```

### Storage Format

Cache files use Markdown with YAML front matter for human readability:

```markdown
---
key: "research/main/extract-a3f2b1/doc.txt"
workflow: "research"
job: "main"
step: "extract-a3f2b1"
timestamp: "2026-01-21T10:30:00Z"
---

[Cached LLM response content]
```

### Cache Invalidation

**Automatic via content hash**:
1. Prompt modified → SHA256 changes → New hash suffix
2. New cache key generated → Old cache not found → Step executes
3. New result stored with new key
4. Old cache remains on disk but unused (can be cleaned manually)

**No version management needed**. Hash directly represents content version.

## To achieve

### Benefits

1. **Zero configuration**: No `cache:` fields in YAML. Single `--cache-dir` flag enables everything.

2. **Automatic invalidation**: Prompt changes create new cache keys. No manual cache clearing needed.

3. **Cost savings**: Repeated workflow runs with identical prompts reuse cached responses.

4. **Development speed**: Instant re-runs during prompt iteration for unchanged steps.

5. **Transparent operation**: Cache is invisible to workflow logic. Add/remove flag without YAML changes.

6. **Inspectable cache**: Markdown format makes cache entries debuggable and auditable.

7. **Simplified architecture**: Leverages existing Key mechanism for iteration handling.

### Breaking Changes

**For Users**:
- Remove `cache:` declarations from YAML files (no longer supported)
- Replace `--llm-cache` flag with `--cache-dir`
- Old JSON cache format incompatible (rebuild cache)

**Migration Path**:
```bash
# 1. Remove cache: keys from YAML
sed -i '' '/cache:/d' workflow.yml

# 2. Update scripts
# OLD: iq agent -f workflow.yml --llm-cache .cache
# NEW: iq agent -f workflow.yml --cache-dir .cache

# 3. Delete old cache
rm -rf .cache
```

### Implementation Phases

**Phase 1 - Core Infrastructure** (Features 1-7):
- Remove legacy Cache field from AST
- Extract cacheable content in compiler
- Define runtime Cache with compile-time hash
- Implement simple key generation using in.Key
- Update constructor to accept workflow/job/step/hash
- Pass workflow/job context through compiler
- Enable automatic cache detection

**Phase 2 - Format & Robustness** (Features 4, 8-9):
- Update storage format to Markdown with YAML front matter
- Handle cache misses and invalidation
- Remove `--llm-cache` flag

**Phase 3 - Quality & Documentation** (Features 10-12):
- Add comprehensive tests
- Update user documentation
- Cleanup obsolete references

### Design Decisions

**Why 6-character hash?**
- 16^6 = 16.7M combinations → sufficient uniqueness for most projects
- Human-readable cache keys for debugging
- Balance between collision resistance and key length

**Why not cache Run steps?**
- Shell commands are cheap to execute (< 1ms typically)
- Commands often have side effects (file writes, API calls)
- Command output may change based on external state

**Why not cache Foreach directly?**
- Foreach is a container/control flow step, not an execution step
- Each iteration calls a job whose steps are cached normally
- Key.SeqID mechanism already provides iteration context

**Why Markdown format?**
- Human-readable for debugging
- YAML front matter for structured metadata
- Familiar format for developers
- Easy to inspect with standard tools (cat, less, grep)

**Why use in.Key instead of separate iteration tracking?**
- Avoids duplicating iteration context logic
- Key already encodes nested context from SeqID
- Simplifies Cache implementation (no iteration parameters needed)
- Foreach naturally creates unique keys per iteration

## Related Decisions

- **0002 - I/O System**: Defines Key type and SeqID mechanism used for iteration tracking
- **0001 - Workflow Blueprint**: Defines AST and compiler structure modified for caching
- **0000 - CLI Application Architecture**: Service layer creates cache storage from --cache-dir flag

## Status

Accepted (2026-01-21)

## Consequences

### Positive

- Dramatically simplified workflow authoring (no cache boilerplate)
- Reduced API costs during development and repeated executions
- Faster iteration during prompt engineering
- Automatic cache invalidation prevents stale results
- Inspectable cache aids debugging

### Negative

- Breaking change requires YAML updates for existing workflows
- Old cache format incompatible (one-time migration cost)
- Cache directory grows over time (manual cleanup needed for old hashes)
- 6-char hash has theoretical collision risk (extremely low probability)

### Neutral

- Cache is opt-in (backward compatible for users not using caching)
- Requires external cache management (no automatic eviction policy)
- Users control cache location and lifetime via filesystem
