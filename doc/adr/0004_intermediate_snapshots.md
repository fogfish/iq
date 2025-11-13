# ADR 0004: Intermediate Result Snapshots and Multi-Deliverable Workflows

**Status:** Analysis (Decision Pending)  
**Date:** 2025-11-12  
**Context:** Supporting workflows with multiple valuable outputs requiring snapshots/persistence

---

## Context and Problem Statement

Users need to execute workflows where multiple intermediate stages produce output that should be persisted. The motivating use case is combining images after text generating where both output needs to be preserved.


### Current State Analysis

The `iq` tool has:

1. **YAML-driven workflow engine** with jobs, steps, routing, foreach, and state management
2. **IOSystem** with Source → Processor → Sink pipeline architecture
3. **Context propagation** via `WorkflowContext` containing `Input`, `Current`, `Steps`, `State`
4. **Batch processing** via `spool.ForEach` with transactional semantics (input removed only after output written)
5. **Shell command execution** via `RunStepNode` supporting bash/zsh/sh commands
6. **Filesystem abstraction** supporting local and S3 via `github.com/fogfish/stream`

### Core Challenge

How to persist intermediate workflow outputs while:
- Maintaining workflow composability and simplicity
- Supporting human-in-the-loop approvals
- Avoiding over-engineering at this stage
- Preserving clean architecture boundaries
- Handling original input propagation through multi-stage pipelines

---

## Decision Drivers

- **Simplicity**: Avoid complex abstractions at this stage of development
- **Flexibility**: Support both simple and complex use cases
- **Composability**: Maintain clean separation between workflow logic and I/O
- **Human-in-the-loop**: Consider approval/review workflows
- **Architectural consistency**: Align with existing ADR decisions (0001, 0002, 0003)
- **Extensibility**: Enable future enhancements without breaking changes

---

## Options Analysis

### Option 1: Status Quo - Multiple Pipelines (No Changes)

**Approach**: Users manage separate workflow files for each deliverable stage.

**Example**:
```yaml
# gentxt.yml
jobs:
  main:
    steps:
      - uses: prompts/gentxt.md
```

```yaml
# genimg.yml
jobs:
  main:
    steps:
      - uses: prompts/genimg.md
```

```bash
# Execution
iq agent batch -f gentxt.yml -I doc/ -O txt/
iq agent batch -f genimg.yml -I txt/ -O img/
```

**Pros**:
- ✅ Zero implementation effort
- ✅ Maximum flexibility (different LLMs, retry strategies per stage)
- ✅ Clean separation of concerns
- ✅ Easy to understand and debug
- ✅ Natural checkpoint for human review between stages
- ✅ Transactional semantics via existing spool mechanism
- ✅ Independent scaling (run write and illustrate in parallel on different machines)

 
**Cons**:
- ❌ No answer how to pass the original document needs to flow through the separated pipelines.
- ❌ User must manage pipeline orchestration manually
- ❌ Original input propagation requires explicit handling
- ❌ More complex directory management
- ❌ No atomic "all stages succeed or fail" guarantee
- ❌ Verbose for simple cases (two files instead of one)

**Human-in-the-Loop**:
- Natural checkpoint: review `txt/` directory before running gen img
- Can edit outputs manually before next stage
- Resume processing with `--mutable` flag after approvals

---

### Option 2: Snapshot via Shell Commands

**Approach**: Use existing `run:` step to orchestrate external snapshots.

**Example**:
```yaml
name: content-creation
jobs:
  main:
    steps:
      # Stage 1: Generate text
      - uses: prompts/gentxt.md
        output: txt
      
      # Snapshot to filesystem
      - run: |
          echo "{{.steps.text}}" > output/articles/{{.document.filename}}.md
      
      # Stage 2: Generate image
      - uses: prompts/genimg.md
        output: img
      
      # Snapshot to filesystem
      - run: |
          echo "{{.steps.img}}" > output/images/{{.document.filename}}.png
```

In this scenario the original input is fed through `{{.document}}` and `{{.spets.key}}`


**Pros**:
- ✅ No code changes required (already implemented)
- ✅ Full control via shell commands
- ✅ Supports S3 via `aws s3 cp` or custom scripts
- ✅ Can trigger external hooks (webhooks, notifications)
- ✅ Template engine provides context access
- ✅ Original input always available via `{{.document}}`

**Cons**:
- ❌ Platform-dependent (bash/zsh commands)
- ❌ Escaping complexity for multi-line outputs
- ❌ No structured error handling (command failures)
- ❌ Security concerns (shell injection if inputs not sanitized)
- ❌ Limited to shell command capabilities (no structured S3 operations)
- ❌ Verbose and error-prone for complex outputs
- ❌ Binary/structured data (images) difficult to handle via echo
- ❌ No automatic path management or collision detection

**Human-in-the-Loop**:
- Can insert approval steps via external scripts
- Example:
  ```yaml
  - run: |
      ./scripts/request-approval.sh "{{.steps.article}}"
      # Blocks until approval received
  ```
- Limited: no built-in approval UI or state management

**Technical Limitations**:
- Shell commands execute in subprocess, no access to workflow state beyond templates
- Cannot modify `WorkflowContext` from shell commands
- Binary output requires base64 encoding or file I/O workarounds

---

### Option 3: Native Sink Step in Workflow

**Approach**: Add first-class `sink:` step type to workflow DSL.

**Proposed Syntax**:
```yaml
name: content-creation
jobs:
  main:
    steps:
      # Stage 1: Generate text
      - uses: prompts/gentxt.md
      
      # Snapshot
      - sink: output/articles/
        format: markdown
      
      # Stage 2: Generate image
      - uses: prompts/genimg.md
      
      # Snapshot illustration
      - sink: output/images/
        format: binary
```

**Feeding Original Input**:
- Original input automatically available as `{{.document}}` in all steps
- No special handling needed—context propagation handles it


**Pros**:
- ✅ Declarative, fits workflow DSL philosophy
- ✅ Type-safe and validated at compile time
- ✅ Reuses existing `iosystem.Sink` infrastructure (supports S3 automatically)
- ✅ Clean integration with progress reporting
- ✅ Consistent with ADR-0001 architecture (AST → Compiler → Runtime)
- ✅ Template engine handles path generation safely
- ✅ Original input propagation automatic via context

**Cons**:
- ⚠️ New step type (~300-400 lines of code)
- ⚠️ Extends DSL surface area (maintenance burden)
- ⚠️ May encourage complex workflows better split into pipelines
- ⚠️ Unclear semantics: should sink steps block? Affect `{{.current}}`?
- ⚠️ Adds I/O operations to workflow execution (side effects)
- ⚠️ Binary data handling (images) requires additional format support

**Human-in-the-Loop**:
- Can combine with approval step:
  ```yaml
  - uses: prompts/write.md
    output: article
  - sink: drafts/
    from: article
  - uses: prompts/approve.md  # Could call external approval API
    when: approval_required == true
  - sink: published/
    from: article
  ```
- Still limited: no built-in approval workflow, requires external integration

**Design Questions**:
1. **State mutation**: Should sink update `{{.current}}`? Proposal: No, sinks are side effects
2. **Error handling**: Should sink failure abort workflow? Proposal: Yes, fail fast
3. **Transactional semantics**: How do multiple sinks compose? Proposal: Sequential, no rollback (consistent with current spool)
4. **Binary data**: How to handle images/PDFs? Proposal: Accept `[]byte` in context, add `format: binary` flag

---

### Option 4: Workflow Composition (Pipeline of Pipelines)

**Approach**: Enable workflows to call other workflows, passing outputs as inputs.

**Example**:
```yaml
# master.yml
name: content-pipeline
jobs:
  main:
    steps:
      - workflow: gentxt.yml

      - workflow: genimg.yml
```

**Feeding Original Input**:
- Explicit via `input:` parameter to sub-workflow
- Master workflow maintains full context

**Pros**:
- ✅ Maximum composability and reusability
- ✅ Clear separation of concerns
- ✅ Sub-workflows can be tested independently
- ✅ Enables workflow library/marketplace

**Cons**:
- ❌ Most complex implementation (~500+ lines)
- ❌ Requires workflow-as-processor abstraction
- ❌ Nested context management complexity
- ❌ Still needs sink step (Option 3) for snapshots
- ❌ May be over-engineering for current use cases

---

## Comparison Matrix

| Criteria                       | Option 1: Status Quo  | Option 2: Shell Commands    | Option 3: Sink Step        | Option 5: Event-Driven |
| ------------------------------ | --------------------- | --------------------------- | -------------------------- | ---------------------- |
| **Implementation Effort**      | None                  | None (exists)               | Medium (~400 lines)        | High (~1000 lines)     |
| **Complexity**                 | Low                   | Medium                      | Medium                     | High                   |
| **Original Input Propagation** | Manual                | Automatic (`{{.document}}`) | Automatic                  | Automatic (in events)  |
| **Declarative**                | ✅                     | ⚠️ (imperative shell)        | ✅                          | ✅                      |
| **S3 Support**                 | ✅ (via batch)         | ⚠️ (via aws cli)             | ✅ (native)                 | ✅                      |
| **Binary Data**                | ✅                     | ❌                           | ⚠️ (needs impl)             | ⚠️                      |
| **Human-in-Loop**              | ✅ (manual)            | ⚠️ (scripting)               | ⚠️ (requires approval step) | ✅ (event-driven)       |
| **Single-Workflow Automation** | ❌ (requires 2-3 runs) | ✅                           | ✅                          | ✅                      |
| **SQS/Event Integration**      | ❌                     | ⚠️ (via aws cli)             | ❌                          | ✅ (native)             |
| **Production Fleet Support**   | ⚠️ (manual)            | ❌                           | ⚠️                          | ✅                      |
| **Error Handling**             | ✅ (transactional)     | ⚠️ (shell exit codes)        | ✅ (Go errors)              | ✅                      |
| **Testability**                | ✅                     | ❌                           | ✅                          | ⚠️ (needs mocking)      |
| **Security**                   | ✅                     | ⚠️ (shell injection)         | ✅                          | ✅                      |
| **Architectural Fit**          | ✅                     | ⚠️ (breaks I/O boundaries)   | ✅                          | ✅                      |

---

## Recommendations

### ~~Recommended Approach: **Hybrid (Option 1 + Option 3 for future)**~~ (Superseded)

**See Decision section below for final recommendation**

~~**Phase 1 (Immediate)**: **Option 1 - Status Quo**~~
- Document patterns for multi-pipeline workflows
- Provide examples showing original input propagation strategies
- Create helper scripts/templates for common patterns
- Focus on making multi-pipeline workflows easy to manage

**Rationale**:
1. **No over-engineering**: Avoid premature abstraction
2. **Natural human-in-the-loop**: Directory checkpoints for review
3. **Maximum flexibility**: Different LLMs, retry strategies, parallel execution
4. **Proven transactional semantics**: Existing spool mechanism works well
5. **Time to learn**: Observe user patterns before adding features

**Phase 2 (Future)**: **Option 3 - Sink Step** (when usage patterns emerge)
- Implement sink step if user demand validates need
- Add binary data support (images, PDFs)
- Integrate with progress reporting
- Consider approval step type

**Why not Option 2 (Shell Commands)?**
- Security concerns (shell injection)
- Platform dependence (Windows support poor)
- Binary data handling problematic
- Breaks architectural boundaries (I/O in workflow logic)
- Difficult to test and debug

**Why not Option 4 (Composition)?**
- Over-engineering for current use cases
- Can be added later without breaking changes
- Still needs sink step (so implements Option 3 + more)

---

## Original Input Propagation Patterns (Option 1)

### Pattern A: Embedded Metadata

**write.md**:
```markdown
---
format: json
---
Write article based on: {{.document}}

Output format:
{
  "brief": "original brief",
  "article": "written content"
}
```

**illustrate.md**:
```markdown
Create illustration for:
Brief: {{.input.brief}}
Article: {{.input.article}}
```

**Pros**: Self-contained, no external state
**Cons**: Increases token usage, redundant data

### Pattern B: Parallel Directories

```
workspace/
  briefs/
    project1.txt
    project2.txt
  articles/  (output from write.yml)
    project1.txt
    project2.txt
  images/    (output from illustrate.yml)
    project1.png
    project2.png
```

Use filename matching to correlate inputs:
```yaml
# illustrate.yml with custom source that reads both directories
```

**Pros**: Clean separation, no redundancy
**Cons**: Requires filename discipline, manual correlation

### Pattern C: Composite Input Generation

Create intermediate job that bundles inputs:
```yaml
# prepare-illustration.yml
jobs:
  main:
    steps:
      - run: |
          jq -n --arg brief "$(cat briefs/{{.document.filename}})" \
                --arg article "$(cat articles/{{.document.filename}})" \
                '{brief: $brief, article: $article}'
```

Then pipe to illustrate.yml

**Pros**: Explicit, debuggable
**Cons**: Requires scripting, complexity

---

## Human-in-the-Loop Patterns

### Pattern 1: Directory Review (Status Quo)
```bash
iq agent batch -f write.yml -I briefs/ -O articles/
# Human reviews articles/ directory, edits files
iq agent batch -f illustrate.yml -I articles/ -O images/
```

### Pattern 2: Approval Scripts (Option 2)
```yaml
steps:
  - uses: prompts/write.md
    output: article
  - run: ./scripts/slack-approval.sh "{{.steps.article}}"
  - uses: prompts/publish.md
```

### Pattern 3: Future Approval Step (Option 3 + extension)
```yaml
steps:
  - uses: prompts/write.md
    output: article
  - sink: drafts/
    from: article
  - approval:
      type: slack
      channel: "#content-review"
      timeout: 24h
  - sink: published/
    from: article
```

---

## Implementation Plan (if Option 3 chosen)

### Phase 1: Core Sink Step
1. Add `SinkStepNode` to AST (~50 lines)
2. Extend parser to recognize `sink:` keyword (~100 lines)
3. Implement `SinkStep` in compiler (~150 lines)
4. Add integration tests (~100 lines)
5. Update documentation and examples (~50 lines)

**Total effort**: ~450 lines, 2-3 days

### Phase 2: Enhanced Features
1. Binary data support (`format: binary`)
2. S3 path validation and error messages
3. Conditional sinks (`when:` expressions)
4. Sink aggregation (multiple sources)

**Total effort**: ~300 lines, 1-2 days

### Phase 3: Approval Integration
1. Design approval step API
2. Implement webhook-based approval
3. Add timeout and fallback handling

**Total effort**: ~500 lines, 3-4 days

---

## Questions for Alignment

Before proceeding with implementation, clarify:

1. **Urgency**: Is this a blocker for users now, or nice-to-have?
2. **User feedback**: Have users requested this? What patterns do they use?
3. **Risk tolerance**: Comfortable with Option 1 (status quo) + good documentation?
4. **Future vision**: Are complex multi-stage workflows core to `iq`'s mission?
5. **Approval workflows**: Is human-in-the-loop a priority feature?

---

## Decision

**DECISION: Option 1**
- Document multi-pipeline patterns for simple cases
- Manual HITL via directory review between pipeline runs
- **Target users**: Development, small teams, simple workflows

### Phase 2: Q1 2026 (Event-Driven Sink)
- Implement sink step with optional event emission
- Support SQS, EventBridge, and webhook targets
- Enable single-workflow execution with external HITL orchestration
- **Target users**: Production deployments, fleet-based processing, event-driven architectures

### Rationale

**Why Phase 1 first**:
- Not currently blocking users
- Validates assumptions and usage patterns
- Simpler mental model for getting started

**Why Phase 2 is justified**:
- **Real production use case**: Fleet of pipelines with SQS-based HITL
- **Automation benefit**: Single workflow execution vs. 2-3 manual pipeline runs
- **Event-driven architecture**: Fits modern cloud-native patterns
- **Maintains original input context**: Event payload includes full workflow state
- **Scalability**: Supports high-throughput production workloads

---

## Consequences

### If Option 1 (Status Quo + Documentation) - Phase 1

**Positive**:
- ✅ Zero technical debt
- ✅ Users learn composable patterns
- ✅ Time to validate assumptions
- ✅ Simple mental model for getting started
- ✅ Works well for development and small teams

**Negative**:
- ⚠️ Manual orchestration for multi-stage pipelines
- ⚠️ Doesn't support automated event-driven HITL
- ⚠️ Requires 2-3 manual commands for content → write → illustrate flow
- ⚠️ Original input tracking requires discipline

**Mitigation**:
- Provide excellent examples and templates
- Create helper scripts for common patterns
- Document event-driven patterns for future
- Monitor user feedback for production use cases

### If Option 5 (Event-Driven Sink) - Phase 2

**Positive**:
- ✅ Single workflow execution (automation)
- ✅ Native SQS/EventBridge integration for HITL fleets
- ✅ Event payload preserves full context (original input + deliverables)
- ✅ Scales to production workloads
- ✅ Declarative event emission in workflow DSL
- ✅ Foundation for complex orchestration patterns
- ✅ Testable via webhooks (no AWS required locally)

**Negative**:
- ⚠️ AWS SDK dependency (~several MB, but likely needed for S3 anyway)
- ⚠️ DSL complexity increases (sink + emit sections)
- ⚠️ Testing requires mocking or LocalStack
- ⚠️ IAM configuration complexity for users
- ⚠️ May encourage overly complex workflows

**Mitigation**:
- Keep implementation minimal and focused
- Provide webhook emitter for local testing
- Document when to use event-driven vs. manual patterns
- Clear error messages for IAM/configuration issues
- Examples showing both simple and complex patterns
- Optional feature (workflows work without emit:)

