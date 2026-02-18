# 0010 - Streaming Pipeline Architecture

**Status**: Proposed  
**Date**: 2026-02-14  
**Deciders**: Architecture Team  
**Supersedes**: Aspects of ADR 0002 (I/O System), ADR 0006 (Key-Value I/O)

## Context

The application currently has two separate execution models that emerged organically:

1. **I/O Pipeline** (`conduit`): A `Source → Processor[] → Sink` pipeline that processes documents sequentially. The `Conduit` orchestrator pulls documents from a `Source` one at a time, threads them through a chain of `Processor` implementations (Chunker → Collector → Agent), and pushes results to a `Sink`. This is an imperative loop with manual EOF propagation, error accumulation, and stub concurrency support (`runConcurrent` is unimplemented).

2. **Blueprint Runtime**: A `Job → Step[]` tree where each `Job` sequentially invokes `Prompter.Prompt()` on an `Event`, threading the event through a decorator chain (Memento → Emitter → Repeater → Printer → Cache → Core). Control flow is modeled via recursive `Prompt()` calls — `Router` dispatches to sub-jobs, `ForEach` iterates over sub-jobs.

These two models are bridged by `processor.Agent`, which converts `Document` ↔ `Event` at the boundary. This creates several architectural problems:

- **Dual execution models**: The conduit loop and the runtime's recursive `Prompt()` chain have overlapping concerns (error handling, progress reporting, output writing) but different mechanisms.
- **No native concurrency**: The conduit's `runConcurrent` is a stub. The blueprint runtime is purely sequential. Adding parallelism requires rethinking both layers.
- **Complex I/O boundary**: The `processor.Agent` bridge performs codec conversion, type mapping (`any` → `Gist`), and format negotiation — complexity that exists solely because two different data models meet.
- **Rigid control flow**: Adding new control patterns (e.g., parallel job execution, streaming intermediate results, pipeline branching) requires modifications to both the conduit loop and the runtime's recursive structure.
- **EOF as data**: The conduit uses sentinel `EOF()` documents flowing through the processor chain — a fragile mechanism that processors must explicitly handle.

## Decision

Adopt a **streaming pipeline architecture** based on Go channels, using the algebra defined in [`github.com/fogfish/golem/pipe`](https://github.com/fogfish/golem/tree/main/pipe) to unify both execution models into a single channel-based processing pipeline.

### Streaming Algebra

The foundational algebra consists of four operations over typed channels (`<-chan T`):

| Operation  | Signature             | Purpose                                                                          |
| ---------- | --------------------- | -------------------------------------------------------------------------------- |
| **Unfold** | `() → <-chan T`       | Produce a stream from a source (file system walk, stdin, literal values)         |
| **Map**    | `<-chan A → <-chan B` | Transform each element (1:1 — LLM call, template rendering, codec conversion)    |
| **FMap**   | `<-chan A → <-chan B` | Transform with fan-out (1:N — document splitting, foreach expansion)             |
| **Fold**   | `<-chan T → T`        | Consume a stream to a terminal value (write to sink, collect results, aggregate) |

Supporting combinators from the library:

| Combinator           | Purpose                                                                |
| -------------------- | ---------------------------------------------------------------------- |
| **Filter**           | Predicate-based element selection (routing decisions)                  |
| **Partition**        | Split stream by predicate into two channels (router with two branches) |
| **Join**             | Merge multiple channels into one (parallel job fan-in)                 |
| **ForEach**          | Apply side-effect per element (progress reporting, caching)            |
| **Take / TakeWhile** | Limit stream consumption                                               |
| **Seq**              | Lift literal values into a channel                                     |
| **Throttle**         | Rate-limit channel throughput (API rate limiting)                      |

The library provides two function abstractions:
- `F[A, B]` — pure morphism `A → (B, error)` for `Map`, `Filter`, `Unfold`, `Fold`
- `FF[A, B]` — arrow/functor `(ctx, A, chan<- B) → error` for `FMap` (one input, many outputs)

Error handling is built into the algebra via two lifting strategies:
- `Lift` / `LiftF` — failure aborts the pipeline (fail-fast)
- `Try` / `TryF` — failure emits to error channel, pipeline continues (skip-error)

### Shared Memory Model

The channel element type is a **pointer to a memory cell** rather than the document content itself. This is the key architectural insight: channels carry references, not values.

```go
// Cell is the unit of data flowing through the pipeline.
// It is a reference to a mutable memory location within the workflow's
// address space. Channels pass *Cell pointers — lightweight, zero-copy.
type Cell struct {
    Key   iosystem.Key   // document identity
    Value Gist           // polymorphic content (Text, Json, List, Binary)
    Steps map[string]Gist // named outputs from prior steps (shared across cells)
    Env   map[string]any  // environment/metadata
}
```

The `Cell` unifies today's `Document` (I/O layer) and `Event` (runtime layer) into a single type. Because channels pass `*Cell` pointers, there is no serialization overhead between pipeline stages. Each stage reads/writes fields on the same memory location.

The `Steps` map provides the shared memory space: any stage can store named results that downstream stages reference via Go templates (`{{.steps.extract}}`). This replaces the current `Memento` decorator — naming is a write to shared memory, not a wrapper.

### Two-Phase Pipeline Compilation

Pipeline construction follows a **two-phase compilation** model, mirroring the existing Parser → AST → Compiler architecture but targeting channel topology instead of decorator chains.

The critical insight is that channel creation and ownership belong to the pipe functions — `pipe.Map`, `pipe.Partition`, `pipe.FMap` etc. each create and return their output channels internally. The AST carries **typed function references** directly in its nodes — `pipe.F[*Cell, *Cell]` for maps, `pipe.FF[*Cell, *Cell]` for fan-out, `func(*Cell) bool` for predicates, `monoid.Monoid[*Cell]` for folds. This makes the AST type-safe: the compiler cannot wire a predicate where a map is expected.

**Phase 1 — Build `vm.App` (functions first, then topology):**
The domain compiler walks the blueprint AST and creates concrete processing functions (LLM calls, shell commands, CEL predicates, retry/cache wrappers). These are assembled into a `vm.App` — a control-flow AST where every node carries its typed function reference directly.

```go
// Phase 1: build vm.App with typed function references
extractFn := pipe.Lift(func(cell *Cell) (*Cell, error) {
    reply, err := llm.Prompt(ctx, renderTemplate(extractPrompt, cell))
    if err != nil {
        return cell, err
    }
    cell.Value = decodeReply(reply)
    cell.Steps["extract"] = cell.Value
    return cell, nil
})

classifyPred := func(cell *Cell) bool {
    return evalCEL(cell, classifyExpr)
}

prog := &vm.App{
    Root: vm.Seq{
        Steps: []vm.Node{
            vm.UnfoldNode{F: sourceFn},
            vm.MapNode{F: extractFn},
            vm.PartitionNode{
                F:       classifyPred,
                Match:   vm.Seq{Steps: []vm.Node{vm.MapNode{F: technicalFn}}},
                Default: vm.Seq{Steps: []vm.Node{vm.MapNode{F: creativeFn}}},
            },
            vm.MapNode{F: formatFn},
            vm.FoldNode{M: sinkMonoid},
        },
    },
}
```

**Phase 2 — Instantiate Channel Topology (from `vm.App`):**
The VM compiler walks the `vm.App` and constructs the live channel topology by calling pipe functions with the function references stored in each node. Each `pipe.Map`, `pipe.FMap`, `pipe.Partition` call creates channels and spawns goroutines. No data flows yet because the source channel has not been fed.

```go
// Phase 2: walk vm.App, instantiate channel graph
ch1, errs1 := pipe.Map(ctx, ch0, node.F)           // MapNode.F is pipe.F[*Cell, *Cell]
ch2a, ch2b := pipe.Partition(ctx, ch1, node.F)      // PartitionNode.F is func(*Cell) bool
ch3a, errs3a := pipe.Map(ctx, ch2a, matchNode.F)
ch3b, errs3b := pipe.Map(ctx, ch2b, defaultNode.F)
ch4 := pipe.Join(ctx, ch3a, ch3b)
ch5, errs5 := pipe.Map(ctx, ch4, formatNode.F)
```

This produces a live channel topology:

```
Unfold ──→ ch0 ──Map──→ ch1 ──Partition──→ ch2a, ch2b
                                           │        │
                                          Map      Map
                                           │        │
                                          ch3a    ch3b
                                           └──Join──┘
                                               │
                                         ch4 ──Map──→ ch5 ──→ Fold
```

The type-safe function reference approach provides compile-time guarantees that the original name-based vtable pattern cannot: a `MapNode` always carries an `F[*Cell, *Cell]`, a `PartitionNode` always carries a `func(*Cell) bool` predicate, and a `FoldNode` always carries a `Monoid[*Cell]`. The compiler enforces correct wiring structurally rather than through runtime lookups.

### Mapping Current Constructs to Stream Operations

| Current Construct         | Stream Operation            | Notes                                                                                                                      |
| ------------------------- | --------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `Source.Next()` iterator  | `Unfold` / `Seq`            | Source becomes a channel producer. File walk → `Unfold`, literal input → `Seq`, stdin → single-element channel             |
| `Sink.Write()` consumer   | `Fold` / `ForEach`          | Sink becomes a channel consumer. File write → `ForEach` with side-effect, collect → `Fold` with monoid                     |
| `Processor.Process()`     | `Map`                       | 1:1 document transformation. The monadic `[]*Doc → []*Doc` collapses to `*Cell → *Cell` since fan-out is handled by `FMap` |
| `processor.Chunker`       | `FMap`                      | 1:N splitting: one document becomes many chunks via arrow `(ctx, *Cell, chan<- *Cell) → error`                             |
| `processor.Collector`     | `Fold`                      | N:1 aggregation: fold all cells into one using a list monoid                                                               |
| `processor.Agent`         | `Map`                       | LLM processing: eliminated as a bridge — the LLM call is a direct `Map` function                                           |
| `Job.Prompt()` sequential | Channel chain               | `ch0 → Map → ch1 → Map → ch2` — each step is a `Map` connected by channels                                                 |
| `Manifold` (LLM agent)    | `Map`                       | `F[*Cell, *Cell]` that renders template, calls LLM, decodes reply                                                          |
| `Router` (conditional)    | `Partition` / `Filter`      | CEL predicate partitions stream; sub-jobs are parallel channel sub-pipelines joined at the end                             |
| `ForEach` (iteration)     | `FMap` + sub-pipeline       | Arrow expands list into per-item channels, each processed by sub-job pipeline, results joined                              |
| `Shell` (command)         | `Map`                       | `F[*Cell, *Cell]` that renders template, execs command, captures stdout                                                    |
| `Memento` (naming)        | Write to `cell.Steps[name]` | No decorator needed — a post-Map side-effect writes to shared memory                                                       |
| `Emitter` (output)        | `Fold` side-effect          | Tap the stream, write to filesystem, continue passing downstream                                                           |
| `Printer` (progress)      | `Fold` side-effect          | Tap the stream, emit progress event, continue passing downstream                                                           |
| `Repeater` (retry)        | Retry within `F[A, B]`      | The function itself retries; error channel carries failures after exhaustion                                               |
| `Cache` (memoization)     | Guard within `F[A, B]`      | Check cache before LLM call, store after — internal to the Map function                                                    |
| `Conduit` orchestrator    | **Eliminated**              | The channel topology *is* the orchestrator. `Unfold ──→ pipeline ──→ Fold` replaces the imperative loop                    |
| `bufferingSink`           | `pipe.ToSeq()` / `Fold`     | Collect all results into a slice, then flush — natural fold operation                                                      |
| EOF propagation           | Channel close               | Closing a channel naturally signals completion to all downstream consumers. No sentinel documents needed                   |

### Concurrency Model

This ADR addresses **pipeline-level parallelism** only — the concurrent execution of steps within a single pipeline. Parallel execution of multiple pipelines per input document (data-level parallelism) is out of scope and will be addressed in a separate ADR.

**Pipeline parallelism via goroutine-per-stage**: Each `pipe.Map`, `pipe.FMap`, `pipe.Partition`, etc. spawns its own goroutine. A pipeline of N stages runs N goroutines connected by channels. While each goroutine processes one cell at a time, multiple stages operate concurrently — stage 2 processes cell A while stage 3 processes cell B that already passed through stage 2. Backpressure is automatic via channel capacity.

```
Time →
─────────────────────────────────────────────
Stage 1 (extract):  [docA] [docB] [docC] ...
Stage 2 (classify):        [docA] [docB] ...
Stage 3 (format):                 [docA] ...
─────────────────────────────────────────────
```

This is the natural concurrency model of Go channel pipelines. It preserves document ordering — cells flow through stages in FIFO order — and requires no additional synchronization.

**Fan-out/fan-in** for routing:
```go
// Router partitions stream — each branch runs in its own goroutine
techCh, otherCh := pipe.Partition(ctx, in, routerPredicate)

// Sub-pipelines run concurrently (each stage is a goroutine)
techOut := techPipeline(ctx, techCh)
otherOut := otherPipeline(ctx, otherCh)

// Fan-in merges results (ordering across branches is non-deterministic,
// but ordering within each branch is preserved)
merged := pipe.Join(ctx, techOut, otherOut)
```

**Note on `fork` sub-package**: The `golem/pipe/fork` package provides parallel variants (`fork.Map` with W workers) that process multiple cells concurrently within a single stage. This violates event ordering and is explicitly **not used** in this architecture. All stages use sequential `pipe.Map` / `pipe.FMap` to guarantee FIFO ordering.

### Error Handling

The `golem/pipe` library models errors as a separate channel (`<-chan error`) returned alongside the output channel. This maps cleanly to the existing error modes:

- **FailFast** (`pipe.Lift`): First error closes the pipeline. The error channel receives one error and the processing function returns `false` from `catch()`, triggering channel closure.
- **SkipError** (`pipe.Try`): Errors are sent to the error channel, processing continues. A `Fold` over the error channel collects all errors into `Stats.Errors`.

```go
// Error collection runs as a parallel fold
errCh := pipe.StdErr(mainPipeline)  // log errors to stderr
// or
allErrors := pipe.Fold(ctx, errCh, errorMonoid)  // collect all errors
```

### Decorator Elimination

The current decorator chain (Memento → Emitter → Repeater → Printer → Cache → Core) is replaced by composing concerns directly into the pipeline:

```go
// Before: nested decorators
step := Memento("extract",
    Emitter(sink,
        Repeater(3, delay,
            Printer(
                Cache(store,
                    Manifold(agent))))))

// After: composed pipeline functions  
fn := pipe.Try(func(cell *Cell) (*Cell, error) {
    // Cache check (was: Cache decorator)
    if cached := cache.Get(cell.cacheKey()); cached != nil {
        cell.Value = cached
        return cell, nil
    }

    // Core LLM call (was: Manifold)
    reply, err := retryCall(agent, cell, 3, delay) // retry is function composition
    if err != nil {
        return cell, err
    }

    // Progress (was: Printer) — or use ForEach tap
    reporter.StepComplete(cell.Key)

    // Cache store
    cache.Put(cell.cacheKey(), reply)
    cell.Value = reply

    // Memento (was: Memento decorator) — shared memory write
    cell.Steps["extract"] = reply

    return cell, nil
})

out, errs := pipe.Map(ctx, in, fn)

// Emitter as parallel tap (was: Emitter decorator)
emitted := pipe.ForEach(ctx, out, pipe.Pure(func(cell *Cell) *Cell {
    sink.Write(cell)
    return cell
}))
```

Alternatively, concerns can be composed as function wrappers (middleware pattern) without channel overhead:

```go
fn := withMemento("extract",
    withCache(store,
        withRetry(3, delay,
            agentCall(agent))))
```

This is a design choice per-step: lightweight concerns (memento, cache) are best as function composition; concerns that need stream-level visibility (progress, metrics) are best as `ForEach` taps on the channel.

### Package Structure

The architecture introduces a standalone **`vm`** (virtual machine) package that fully abstracts control flow. This package is domain-agnostic — it knows nothing about LLMs, prompts, documents, or codecs. It is a candidate for open-source as an independent module (e.g., `github.com/fogfish/vm`).

```
vm/                        # STANDALONE — open-source candidate
  ast.go                   # Control-flow AST (App, Seq, Map, FMap, Partition, Join, ...)
  cell.go                  # Cell type — generic shared memory unit
  vm.go                    # Compiler: walks vm.App → channel topology
  vm_test.go               # Pure control-flow tests (no LLM, no I/O)
```

The `vm` package defines:

1. **Control-Flow AST** — a type-safe AST where each node carries typed function references from `golem/pipe`. No domain concepts (agents, prompts, schemas). Just structural nodes with their processing functions:

```go
// Node represents a control-flow operation.
type Node interface{ HKT1(Node) }

// Seq executes a list of nodes sequentially as a channel chain.
type Seq struct {
    Steps []Node
}

// MapNode applies F[*Cell, *Cell] to each element.
type MapNode struct {
    F pipe.F[*Cell, *Cell]
}

// FMapNode applies FF[*Cell, *Cell] (fan-out) to each element.
type FMapNode struct {
    FF pipe.FF[*Cell, *Cell]
}

// PartitionNode splits stream by predicate into branches.
type PartitionNode struct {
    F       func(*Cell) bool
    Match   Node
    Default Node
}

// FoldNode terminates a stream.
type FoldNode struct {
    M monoid.Monoid[*Cell]
}

// UnfoldNode produces a stream.
type UnfoldNode struct {
    F pipe.F[*Cell, *Cell]
}
```

2. **VM compiler** — walks the `vm.App`, instantiates the live channel topology by calling pipe functions with the typed function references stored in each AST node:

```go
// App is the root of a control-flow AST.
type App struct {
    Root Node
}

// Compile walks vm.App and builds the channel topology.
func Compile(ctx context.Context, app *App) (<-chan *Cell, <-chan error)
```

3. **Cell** — the generic shared memory unit (as defined in the Shared Memory Model section).

The application integrates the `vm` package through an **AST transformation** layer:

```
internal/
  blueprint/
    ast/                 # unchanged — domain-specific AST (agents, prompts, schemas)
    parser/              # unchanged
    compiler/
      compiler.go        # MODIFIED — transforms blueprint AST → vm.App
      transform.go       # NEW — blueprint AST → vm control-flow AST transformation
    runtime/             # unchanged during transition — existing Prompter implementations
  iosystem/              # unchanged during transition
    codec/               # unchanged
    storage/             # unchanged
    conduit/             # unchanged during transition
    processor/           # unchanged during transition
    source/              # unchanged during transition
    sink/                # unchanged during transition
```

The blueprint compiler walks the domain AST (`BlueprintNode` → `JobNode` → `StepNode`), creates concrete domain functions (LLM calls, shell commands, CEL predicates, retry/cache wrappers), and assembles them into a `vm.App` with `vm.Seq`, `vm.MapNode`, `vm.PartitionNode`, etc. Domain details (prompt templates, CEL expressions, MCP configs) are captured in closures embedded directly in the AST nodes.

### Migration Strategy

The migration follows a strict **fail-fast integration** approach. Every phase produces a working, releasable application. The VM is integrated into the release build immediately in Phase 2 — not deferred to a late phase.

**Phase 1 — VM Package**

Implement the `vm` package as a standalone, domain-agnostic module. Define the control-flow AST with typed function references, the compiler (AST → channel topology), and `Cell` type. Test extensively with pure control-flow tests — no LLM, no I/O, no application dependencies.

This phase has **zero impact** on the existing application.

**Phase 2 — Replace Conduit with VM**

Remove the conduit imperative loop. The outer application pipeline becomes a VM pipeline: `VM[Unfold → Map(blueprint) → Fold]`. The blueprint runtime (`Job.Prompt()` with its decorator chain) continues to execute as-is — it is wrapped as a single `Map` function inside the VM.

```
BEFORE:  Conduit.runSequential { source.Next() → processor.Agent.Process() → sink.Write() }
AFTER:   VM[ Unfold(source) → Map(blueprint.Prompt) → Fold(sink) ]
```

**What changes:**
- `conduit.Conduit` and its `runSequential` loop are replaced by a VM pipeline with three nodes: `UnfoldNode` → `MapNode` → `FoldNode`
- `UnfoldNode.F` wraps existing `Source` implementations (filesystem, stdin, none)
- `FoldNode.M` wraps existing `Sink` implementations (stdout, file, filesystem)
- The `MapNode.F` performs the same work as `processor.Agent.Process()`: decode `Document` → `Event`, call `blueprint.Prompt()`, encode `Event` → `Document`
- `processor.Agent`, `processor.Chunker`, `processor.Collector`, EOF sentinel handling, `bufferingSink` are eliminated
- Chunker becomes an `FMapNode` before the blueprint Map; Collector becomes a `FoldNode` before the blueprint Map

**What stays the same:**
- `blueprint.Blueprint`, `runtime.Job`, `runtime.Manifold`, `runtime.Router`, `runtime.ForEach`, `runtime.Shell` — all untouched
- All decorators (Memento, Emitter, Repeater, Printer, Cache) — all untouched
- Parser, AST, Compiler — all untouched
- All existing tests of blueprint runtime pass without modification

**The application is fully functional after this phase.** The conduit is gone, VM is in production, but the internal blueprint runtime is unchanged.

**Phase 3 — Migrate Agents and Shell to Internal VMs**

Each leaf-node runtime component (`Manifold`, `Shell`) creates its own internal VM instance while preserving the `Prompter` interface. The decorator chain (cache, retry, printer, memento, emitter) is also preserved — it wraps the component as before.

The pipeline becomes nested VMs:

```
VM[ Unfold → Map(blueprint) → Fold ]
                  │
                  └─ Job.Prompt() sequentially calls:
                       Memento → Emitter → Repeater → Printer → Cache → Manifold
                                                                           │
                                                                    VM[ Unfold(cell) → Map(llm.Prompt) → Fold(decode) ]
```

**What changes per component:**

- **`Manifold.Prompt()`**: Internally constructs `VM[Unfold(input) → Map(llm) → Fold(decode)]`. The Unfold produces a single cell from the input Event. The Map calls `llm.Prompt()`. The Fold collects the result. Externally, `Manifold` still satisfies `Prompter` — `Job` calls `Prompt()` as before, decorators wrap it as before.

- **`Shell.Prompt()`**: Same pattern — `VM[Unfold(input) → Map(exec) → Fold(capture)]`. The Map renders the template and runs the command. Externally unchanged.

**What stays the same:**
- `Job.Prompt()` — still iterates `[]Prompter` sequentially
- `Router.Prompt()` — still evaluates CEL, dispatches to sub-job
- `ForEach.Prompt()` — still iterates list, calls sub-job per item
- All decorators — still wrap `Prompter` as before
- All existing tests pass without modification

**The application is fully functional after this phase.** Agents and Shell use channels internally, but the blueprint execution model is unchanged from the outside.

**Phase 4 — AST Transformation and Full Pipeline**

Transform the blueprint AST into a VM control-flow AST, node by node. Each transformation step is an independent PR that keeps the application functional. The decorator chain is absorbed into the VM pipeline as function composition within AST node function references.

#### Node-by-Node Transformation Analysis

The transformation replaces `compiler.compileStep()` → decorator chain with direct emission of VM AST nodes carrying their processing functions. Each node type below is an independent, shippable PR.

**PR 4.1 — AgentStepNode (simplest, most common)**

Current compilation in `compileStep()`:
```
AgentStepNode → Manifold → Cache(Printer(Repeater(Emitter(Memento(Manifold)))))
```

VM transformation:
```
AgentStepNode → MapNode{F: withMemento(name, withEmit(sink, withRetry(n, withCache(store, manifoldFn))))}
```

The `Manifold` is no longer wrapped in decorators — its concerns are composed as function wrappers embedded directly in the `MapNode`. The `MapNode` in the VM AST is a single node; the complexity lives in the `F` function reference.

**Impact**: Covers ~70% of workflow steps. All examples with sequential agent chains (`01_chain`, `02_json_schema`, `03_state`, `09_templates`) work via VM pipeline.

**PR 4.2 — RunStepNode (Shell)**

Current: `RunStepNode → Shell → decorators`

VM transformation:
```
RunStepNode → MapNode{F: withMemento(name, shellFn)}
```

Shell steps are rarely decorated (no cache, no retry typically). The transformation is straightforward.

**Impact**: Example `11_command` works via VM pipeline.

**PR 4.3 — Job (Seq composition)**

Current: `Job.Prompt()` iterates `[]Prompter` sequentially.

VM transformation: A `JobNode` becomes `Seq{Steps: [MapNode, MapNode, ...]}` in the VM AST. The sequential loop is replaced by the channel chain that `Seq` compiles to.

At this point the `runtime.Job` type is no longer needed — `Seq` handles sequential composition.

**Impact**: All multi-step workflows work via VM pipeline.

**PR 4.4 — RouterStepNode**

Current: `Router.Prompt()` optionally calls an LLM agent, then evaluates CEL conditions, then dispatches to a sub-job.

VM transformation:
```
RouterStepNode → Seq{
    MapNode{F: routerAgentFn},              // optional: LLM call to produce choice
    PartitionNode{
        F:       celPredicate0,              // CEL predicate for route 0
        Match:   Seq{...route-0-job...},
        Default: PartitionNode{              // nested: next condition
            F:       celPredicate1,
            Match:   Seq{...route-1-job...},
            Default: Seq{...default-job...},
        },
    },
}
```

Multi-route routers become nested `PartitionNode` chains (if/else-if/else). Each route's sub-job is a `Seq` of its own steps (already handled by PR 4.3).

**Complexity**: The router's CEL evaluation context needs `choice` available. The predicate function in the `PartitionNode` must access the shared memory (`cell.Steps["__choice__"]`) that the preceding `MapNode` wrote.

**Impact**: Example `04_routing` works via VM pipeline.

**PR 4.5 — ForeachStepNode**

Current: `ForEach.Prompt()` evaluates CEL selector, iterates list, calls sub-job per item, formats output.

VM transformation:
```
ForeachStepNode → FMapNode{FF: pipe.LiftF(func(ctx, cell, out) error {
    list := evalSelector(cell)
    for i, item := range list {
        subCell := cell.copy(item)
        // compile sub-job as inline VM or direct function call
        result := subJobFn(subCell)
        out <- result
    }
})}
```

ForEach is an `FMapNode` because it expands one cell into many. The sub-job execution can be either:
- (a) A direct function call to the compiled sub-job function (simpler, preserves current sequential behavior)
- (b) A nested VM pipeline (enables future per-item parallelism)

Option (a) is recommended for this PR to minimize risk.

**Impact**: Examples `06_foreach`, `10_foreach_selector` work via VM pipeline.

**PR 4.6 — Full Pipeline Integration**

With all node types transformed:
- Remove `runtime.Job` (replaced by VM `Seq`)
- Remove all decorator types (`Memento`, `Emitter`, `Repeater`, `Printer`, `Cache`) — their logic is absorbed into function composition within AST nodes
- Remove `processor.Agent` bridge (already gone from Phase 2)
- The blueprint compiler's `Compile()` now returns a `vm.Pipeline` instead of a `*Workflow` with `map[string]*runtime.Job`

**PR 4.7 — Cleanup**

- Remove `conduit/` package (already gone from Phase 2)
- Remove `processor/` package
- Remove `source/` and `sink/` packages (replaced by VM Unfold/Fold adapters)
- Simplify `iosystem/` to `codec/` + `storage/` + `document.go`
- Extract `vm` as independent open-source module

#### Transformation Order Summary

```
PR 4.1  AgentStepNode  → MapNode       (covers ~70% of steps, highest value)
PR 4.2  RunStepNode    → MapNode       (simple, low risk)
PR 4.3  Job            → Seq           (enables multi-step composition)
PR 4.4  RouterStepNode → PartitionNode (complex, requires CEL context handling)
PR 4.5  ForeachStepNode→ FMapNode      (complex, requires sub-job execution)
PR 4.6  Full pipeline integration       (remove old runtime types)
PR 4.7  Cleanup                         (remove conduit, processors, source/sink)
```

Each PR is independently testable: the existing example workflows serve as integration tests. After each PR, all examples that use the transformed node types run through the VM pipeline, while examples using not-yet-transformed node types continue to use the old runtime path. The compiler can dispatch per-step: transformed node types emit VM AST nodes, untransformed types fall back to the existing `Prompter` chain wrapped in a `MapNode`.

## Consequences

### Positive

- **Unified execution model**: One pipeline architecture replaces two (conduit + runtime). The channel topology *is* the program.
- **Pipeline parallelism**: Each stage runs in its own goroutine, enabling concurrent step execution. Go channels provide backpressure and cancellation (`context.Context`) without custom synchronization code.
- **Composability**: Stream algebra operations compose naturally — new control patterns (parallel jobs, conditional branches, stream merging) are combinations of existing operators.
- **Standalone VM**: The `vm` package is domain-agnostic and independently testable. It is a candidate for open-source extraction, reducing coupling between the application and the streaming infrastructure.
- **Non-breaking migration**: Each phase preserves the application's external behavior. Existing components adopt channels internally behind stable interfaces. No big-bang rewrite.
- **Simplified I/O**: Sources become `Unfold` functions, sinks become `Fold` functions. The `Conduit` orchestrator, `Processor` interface, EOF sentinels, and `processor.Agent` bridge are eventually eliminated.
- **Zero-copy data flow**: Channels pass `*Cell` pointers. No serialization between stages. Shared `Steps` map provides natural inter-step communication.
- **Type-safe pipeline**: The control-flow AST carries typed function references (`pipe.F`, `pipe.FF`, `monoid.Monoid`) — the compiler enforces correct wiring structurally at compile time.
- **Declarative pipeline**: The control-flow AST is a data structure that can be inspected, validated, visualized, and optimized independently of domain logic.
- **Error channels**: Errors flow through a parallel channel, enabling flexible error collection, logging, and fail-fast/skip-error modes without conditional logic in the processing path.
- **Graceful shutdown**: `context.Context` cancellation propagates through all channel operations naturally — the library handles this in every operator.

### Negative

- **Learning curve**: Channel-based pipelines require understanding of Go concurrency patterns and the `golem/pipe` algebra.
- **Debugging complexity**: Goroutine-per-stage means stack traces span multiple goroutines. Channel deadlocks are possible if topology is wired incorrectly.
- **No data-level parallelism**: This architecture processes documents sequentially through the pipeline (one cell per stage at a time). Parallel processing of multiple documents concurrently is deferred to a separate ADR.
- **Migration effort**: Multi-phase migration touches compiler, runtime, I/O system, and service layer. Dual code paths (old runtime + VM pipeline) must coexist during transition.
- **Two ASTs**: The domain AST and the VM control-flow AST are separate structures connected by a transformation layer. Changes to workflow semantics may require updates in both.

### Risks

- **Channel capacity tuning**: Under-sized channels cause blocking; over-sized channels waste memory. Needs benchmarking to find optimal defaults.
- **Goroutine leaks**: Incorrectly wired pipelines (unconsumed channels, missing close) leak goroutines. Mitigated by the library's consistent close propagation, but custom `FMap` arrows must be careful.
- **Memory pressure with FMap**: ForEach expansion can create many concurrent sub-pipelines. Need to bound concurrency for large arrays.

## References

- [golem/pipe — Type Safe Channels](https://github.com/fogfish/golem/tree/main/pipe): Streaming algebra implementation for Go channels
- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines): Foundation concepts
- [SRFI-41 Streams](http://srfi.schemers.org/srfi-41/srfi-41.html): Stream interface semantics that `golem/pipe` derives from
- ADR 0000: Architecture (four-layer CLI architecture)
- ADR 0001: Workflow Blueprint (Parser → AST → Compiler)
- ADR 0002: I/O System (Source → Processor → Sink, superseded by this ADR)
- ADR 0006: Key-Value I/O System (storage abstraction, retained)
- ADR 0008: Unified Codec System (codec registry, retained)
- ADR 0009: Automatic Caching (content-addressed cache, retained as middleware) 

## Implementation Plan

### Phase 1 — Standalone VM Package (zero app impact)

| Issue                                                                            | Description                                                                         |
| -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| [#96 — Add `golem/pipe` dependency](https://github.com/fogfish/iq/issues/96)     | Foundation library for channel algebra                                              |
| [#97 — Implement `Cell` type](https://github.com/fogfish/iq/issues/97)           | Shared memory unit flowing through channels                                         |
| [#98 — Implement VM control-flow AST](https://github.com/fogfish/iq/issues/98)   | `Seq`, `MapNode`, `FMapNode`, `PartitionNode`, `FoldNode`, `UnfoldNode`, `JoinNode` |
| [#99 — Implement VM compiler](https://github.com/fogfish/iq/issues/99)            | `Compile()` walks `vm.App` → channel topology, `Run()` → execution                   |

### Phase 2 — Replace Conduit with VM

| Issue                                                                                 | Description                                         |
| ------------------------------------------------------------------------------------- | --------------------------------------------------- |
| [#111 — Cell ↔ Event ↔ Document conversion](https://github.com/fogfish/iq/issues/111) | Bridge utilities for the transition period          |
| [#112 — Integration test suite](https://github.com/fogfish/iq/issues/112)             | Regression gate for all example workflows           |
| [#100 — Source → Unfold adapters](https://github.com/fogfish/iq/issues/100)           | Adapt `None`, `Reader`, `Storage` sources           |
| [#101 — Sink → Fold adapters](https://github.com/fogfish/iq/issues/101)               | Adapt `Writer`, `File`, `FileSystem` sinks          |
| [#102 — Replace conduit with VM pipeline](https://github.com/fogfish/iq/issues/102)   | Core Phase 2 — `VM[Unfold → Map(blueprint) → Fold]` |
| [#103 — Remove conduit/processor packages](https://github.com/fogfish/iq/issues/103)  | Dead code cleanup                                   |

### Phase 4 — Node-by-Node AST Transformation

| Issue                                                                                  | Description                                                                                |
| -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| [#104 — AgentStepNode → `vm.MapNode`](https://github.com/fogfish/iq/issues/104)        | Covers ~70% of steps; `withMemento`, `withCache`, `withRetry`, `withEmit` wrappers         |
| [#105 — RunStepNode → `vm.MapNode`](https://github.com/fogfish/iq/issues/105)          | Shell command steps                                                                        |
| [#106 — Job → `vm.Seq`](https://github.com/fogfish/iq/issues/106)                      | Sequential composition replaces `runtime.Job` loop                                         |
| [#107 — RouterStepNode → `vm.PartitionNode`](https://github.com/fogfish/iq/issues/107) | Nested if/else-if/else chain with CEL predicates                                           |
| [#108 — ForeachStepNode → `vm.FMapNode`](https://github.com/fogfish/iq/issues/108)     | 1:N expansion with per-item sub-job execution                                              |
| [#109 — Remove old runtime types](https://github.com/fogfish/iq/issues/109)            | Eliminate `Job`, `Memento`, `Emitter`, `Printer`, `Repeater`, `Cache`, `Router`, `ForEach` |
| [#110 — Final cleanup + `vm` docs](https://github.com/fogfish/iq/issues/110)           | Simplify package structure, document `vm` for extraction                                   |
