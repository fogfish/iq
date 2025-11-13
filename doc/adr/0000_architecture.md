# 0000 - CLI Application Architecture

## In the context of

Building a command-line tool (`iq`) that processes documents through LLM-powered workflows with diverse input sources (stdin, files, directories, S3), output destinations, and processing modes (streaming, batch, server). Users interact exclusively through CLI flags and YAML configuration files—no Go programming required. Must support simple prompts (`iq tell`), batch processing (`iq agent batch`), and MCP server mode (`iq agent serve`).

## Facing Concern

**Flag explosion problem**: Multiple orthogonal concerns (LLM configuration, agent/workflow setup, input sources, output destinations, chunking strategies) create combinatorial flag complexity. Each command needs consistent flags but different defaults.

**Configuration validation timing**: Flags must be validated and converted to runtime objects (LLM clients, sources, sinks, processors) with clear error messages before execution starts. Invalid combinations (e.g., batch without directories) should fail early.

**Layering and dependency direction**: Need clear separation between CLI concerns (flag parsing, help text) and business logic (workflow execution, I/O processing) without circular dependencies. Commands must compose lower-level abstractions without understanding their implementation.

**Reusability across commands**: Core logic (LLM creation, source/sink building, workflow execution) should be reusable across `agent`, `batch`, `serve` commands without duplication.

**Builder pattern complexity**: Converting CLI flags to complex objects (LLM with decorators, pipelines with processors, sources with merge mode) requires incremental construction with error propagation.

## We decided for

**Four-layer architecture with strict dependency hierarchy**:

```
┌─────────────────────────────────────────────────────────┐
│  Layer 1: cmd/                                          │
│  - Cobra command definitions (root, agent, batch, etc) │
│  - Flag declarations using opts structs                 │
│  - Command execution logic (thin wrappers)              │
│  - Depends on: internal/service (builders)              │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Layer 2: internal/service/                             │
│  - Builder pattern for flag-to-object conversion        │
│  - llm.Builder: Creates configured LLM instances        │
│  - source.Builder: Creates iosystem.Source from flags   │
│  - sink.Builder: Creates iosystem.Sink from flags       │
│  - worker.Builder: Creates conduit.Conduit from flags   │
│  - Depends on: internal/iosystem, internal/blueprint    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Layer 3: internal/iosystem/ + internal/blueprint/      │
│  - Core abstractions (Source, Processor, Sink, Conduit) │
│  - Blueprint parser/compiler/executor                   │
│  - No knowledge of CLI flags or commands                │
│  - Pure business logic, fully testable                  │
│  - Depends on: external libraries (stream, scanner)     │
└─────────────────────────────────────────────────────────┘
```

**Opts pattern for flag management**: Each concern encapsulated in typed struct with methods:
- `optsLLM`: LLM provider configuration (profile, model, tokens, debug, think)
- `optsAgent`: Agent/workflow configuration (file path, chunking strategy, JSON output)
- `optsInput`: Input source configuration (directory, merge mode, stdin detection)
- `optsReply`: Output destination configuration (directory, file, quiet/silent modes)

Each opts struct provides:
- `apply(cmd *cobra.Command)`: Declares flags on command (called at init time)
- `build() (T, error)`: Converts flags to runtime object (called at run time)

**Builder pattern with method chaining**: Service layer builders enable fluent configuration:

```go
// LLM Builder (internal/service/llm)
llm, err := llm.New().
    Profile("iq", "claude-3-sonnet").
    Debug(true).
    Think(true).
    Quota(maxEpoch, maxUsage).
    Build()

// Worker Builder (internal/service/worker)
conduit, err := worker.New().
    Runtime().
    Splitter(chunkConfig).
    Workflow(agentFile, llm).
    Jsonify(true).
    Build()

// Source Builder (internal/service/source)
src, err := source.New().
    Files(".", args...).
    Path(dir).
    Merge(merge).
    Stdin().
    None().
    Build()

// Sink Builder (internal/service/sink)
snk, err := sink.New().
    File(file).
    Path(dir).
    Stdout(enabled).
    Build()
```

**Decorator pattern for LLM capabilities**: LLM instance wrapped with decorators applied during build:
- Base: `autoconfig.FromNetRC` or `Mock`
- `thinking` decorator: Prints reasoning to stderr
- `aio.NewJsonLogger` decorator: Debug logging
- `aio.NewQuota` decorator: Token/epoch limits

Each decorator wraps `chatter.Chatter` interface, enabling composition without tight coupling.

**Three command patterns**:
1. **Simple streaming** (`agent`): stdin/files → conduit → stdout/file
2. **Batch processing** (`agent batch`): `spool.ForEach` with reader/writer adapters
3. **MCP server** (`agent serve`): Workflow exposed as MCP tool, requires name and schemas

**Persistent flags inheritance**: All opts applied to `rootCmd.PersistentFlags()`, automatically inherited by subcommands. Eliminates flag redeclaration, ensures consistency.

**Lazy source creation**: Source builder methods check `b.src != nil` before creating, implementing precedence: explicit files > path > stdin > none. First valid source wins, rest ignored.

**Error propagation in builders**: Each builder stores first error in `err` field, subsequent methods become no-ops if `err != nil`. Forces error checking at `Build()`, prevents partial construction.

## Neglected

**Alternative CLI frameworks rejected**:

- **Direct flag.FlagSet**: More control but loses Cobra's help generation, subcommands, completion. Would require manual help text, flag inheritance, and validation.
- **Viper for config files**: Adds complexity, unclear precedence (CLI vs file), yet another config format. YAML workflows provide configuration, flags sufficient for runtime options.
- **Custom DSL**: Could create simplified syntax like `iq "prompt.yml | stdin > output.txt"`. Rejected because requires parser, less discoverable than flags, harder to script.

**Alternative flag patterns rejected**:

- **Global variables**: Simple but untestable, unclear ownership, race conditions in parallel tests. Rejected for testability.
- **Single config struct**: All flags in one struct passed everywhere. Rejected because violates separation of concerns, creates tight coupling.
- **Functional options at CLI level**: Could use `iq tell --llm-profile=x --llm-model=y` with functional options. Rejected because opts structs more structured, easier to document.

**Alternative builder patterns rejected**:

- **Direct construction**: `llm.New(profile, model, debug, think, maxEpoch, maxUsage)`. Rejected because parameter explosion, unclear optional parameters, poor extensibility.
- **Config structs**: `llm.New(Config{Profile: "iq", Debug: true})`. Rejected because requires all-or-nothing configuration, less fluent than chaining.
- **Mutable setters**: `llm.SetDebug(true); llm.SetThink(true)`. Rejected because mutable state, unclear initialization, error handling awkward.

**Alternative layering approaches rejected**:

- **Commands directly use iosystem**: Could skip service layer, commands call `iosystem.NewPipeline` directly. Rejected because duplicates flag-to-object logic across commands, violates DRY.
- **Fat service layer**: Could put all business logic in service. Rejected because business logic should be in iosystem/blueprint, service is just adapters.
- **Single "application" object**: Could have one `App` struct with all state. Rejected because monolithic, hard to test parts independently.

## To Achieve

**Discoverability**: Cobra's auto-generated help text, command completion, and flag descriptions make tool self-documenting. Users discover features through `iq help`, `iq agent --help`, and shell completion.

**Testability**: Clean layering enables testing at each level:
- Layer 1 (cmd): Test flag parsing, help text, error messages
- Layer 2 (service): Test builders with mock dependencies
- Layer 3 (iosystem/blueprint): Test pure business logic independently

**Consistency**: Persistent flags ensure all commands have same LLM, I/O, and agent options. Opts pattern ensures same flags work identically across commands.

**Composability**: Builders enable command-specific combinations—`agent` uses simple source→sink, `batch` adds spool iteration, `serve` integrates MCP server. Same underlying abstractions, different orchestration.

**Error clarity**: Builders validate incrementally, return descriptive errors with context (e.g., "failed to mount input dir /invalid: no such file"). Flag validation happens before expensive operations (LLM initialization).

**Extensibility**: Adding new command requires:
1. Create command in `cmd/` using existing opts
2. Call existing builders from `internal/service`
3. Orchestrate with appropriate pattern (stream/batch/server)
No changes to lower layers needed.

**Performance**: Lazy evaluation—sources/sinks created only if used, LLM initialized once, processors reused across documents. Builder pattern supports optimization (e.g., mock LLM for testing).

**Single binary**: All functionality in one executable, no plugins or extensions needed. Flag-driven configuration sufficient for all use cases.

## Accepting

**Cobra dependency**: Framework lock-in for CLI parsing. Acceptable because mature, stable, widely used. Would require significant rewrite to change, but no compelling reason to do so.

**Opts boilerplate**: Each opts struct duplicates pattern (apply/build methods, error handling, nil checks). Acceptable because consistent, type-safe, enables testing. Could use reflection/code generation but adds complexity.

**Builder verbosity**: Chained method calls verbose compared to direct construction. Acceptable because readable, self-documenting, enables optional configuration. Alternative (many parameters) worse.

**Service layer indirection**: Extra layer between commands and iosystem. Acceptable because enables reuse, isolates flag concerns. Commands stay thin, business logic remains in iosystem/blueprint.

**Multiple builder types**: Separate builders for llm, source, sink, worker. Acceptable because single-responsibility, each builder focused on one concern. Unified builder would be monolithic.

**No configuration file for CLI flags**: All configuration via flags or YAML workflows. Users requested config file for repeated flag combinations (e.g., `~/.iqrc` for default LLM). Deferred because workflows provide configuration, environment variables available for common settings.

**Persistent flag inheritance limitations**: Subcommands inherit all persistent flags even if irrelevant (e.g., `config` command has LLM flags). Acceptable because Cobra limitation, flags ignored if unused, low confusion risk.

**Error handling in builders stops at first error**: Can't collect all validation errors, reports only first. Acceptable because most errors block subsequent steps anyway (can't create source if directory invalid), early failure clear enough.

**No undo/rollback in batch processing**: If batch fails halfway, already-processed files remain in output, already-consumed inputs removed (if mutable). Acceptable because spool provides transaction semantics per file, resume capability handles failures.

**Mock LLM special case**: Profile/model "mock" creates mock instead of real LLM. Acceptable for testing, but special case in production code. Alternative (separate test-only path) more complex. Clearly documented, low risk.

