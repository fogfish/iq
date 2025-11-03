# 0001 - Workflow Blueprint Architecture

## In the context of

Building an LLM-powered file processing tool (`iq`) that enables users to define complex multi-step workflows involving conditional routing, state management, and tool integration. Users need to chain multiple LLM operations together, route execution based on LLM outputs, and maintain context across steps—all without writing Go code.

## Facing Concern

**Multi-phase compilation complexity**: Workflows require parsing YAML definitions, validating cross-references between jobs and routes, compiling CEL expressions, initializing LLM agents with templates, and resolving job dependencies—all while providing clear error messages at each phase.

**Type safety vs flexibility tradeoff**: Need runtime validation of structured LLM outputs (JSON schemas) and CEL expressions without sacrificing the flexibility of dynamic routing and template-based prompts.

**Context propagation challenge**: Workflow steps must access original input, previous step outputs, and shared state without polluting function signatures or creating thread-safety issues.

## We decided for

**Three-phase Parser → AST → Compiler architecture** with intermediate representation:

1. **Parser** (`internal/blueprint/parser/`):
   - Converts YAML workflow files to immutable AST
   - Handles both blueprint files and agent markdown files (with optional frontmatter)
   - Supports two formats: pure markdown or YAML frontmatter + prompt separated by `\n---\n`
   - Resolves relative file paths from blueprint directory
   - Zero validation—pure syntax transformation

2. **AST** (`internal/blueprint/ast/`):
   - Immutable intermediate representation with typed nodes
   - Three step types: `AgentStepNode` (simple execution), `RouterStepNode` (CEL-based routing), `ForeachStepNode` (array iteration)
   - Separates data structures from logic—AST is pure data
   - Well-known context keys defined as constants: `document`, `input`, `current`, `steps`, `state`

3. **Compiler** (`internal/blueprint/compiler/`):
   - Validates semantic correctness: route references, CEL syntax, agent file existence
   - Compiles CEL expressions using `github.com/google/cel-go`
   - Initializes agents with Go template engine
   - Resolves job references to executable function pointers
   - Returns executable `Workflow` with compiled `Job` and `Step` objects

**CEL for conditional routing**: Using Common Expression Language (Kubernetes standard) for route conditions instead of simple string matching, enabling complex expressions like `choice.contains("urgent") && confidence > 0.8`.

**Context via `context.Context`**: Using Go's idiomatic `context.WithValue` pattern for workflow state propagation instead of explicit `*Context` parameters. Private `contextKey` type prevents collisions. `WorkflowContext` struct contains `Input`, `State`, `Steps`, and `Current` maps.

**JSON Schema validation**: Optional `input_schema` and `output_schema` in agent frontmatter, validated at runtime (before execution for input, after for output) using `github.com/google/jsonschema-go`.

## Neglected

**Alternative architectures rejected**:

- **Single-pass interpreter**: Would mix parsing, validation, and execution, making errors ambiguous and testing difficult
- **Code generation**: Would require build step and lose runtime flexibility
- **Simple string-based routing**: Cannot express complex conditions; CEL provides industry-standard expression evaluation

**Alternative context patterns rejected**:

- **Explicit `wtfCtx *Context` parameter**: Clutters function signatures, not idiomatic Go, breaks compatibility with `chatter.Chatter` interface
- **Global state**: Thread-unsafe, prevents concurrent workflow execution
- **Per-step context copying**: Performance overhead, complexity in managing updates

**Alternative validation strategies rejected**:

- **Schema-first approach**: Too rigid for exploratory LLM interactions
- **Runtime-only validation**: Fails late, poor developer experience
- **No validation**: Type errors caught too late, difficult debugging

## To Achieve

**Clear error reporting**: Three distinct error phases (parse, compile, runtime) with specific error messages indicating file location, line number, and nature of failure.

**Reusability and composability**: Compiled workflows are immutable and reusable. Same `Blueprint` can execute different jobs with different inputs. Jobs implement common `AI` interface (`Prompt(ctx, input) (output, error)`).

**Testability**: Each phase independently testable—parser with mock files, compiler with mock factory, runtime with mock LLM. AST can be constructed programmatically for tests.

**Extensibility**: Adding new step types requires only: (1) AST node definition, (2) parser support, (3) compiler implementation, (4) validation rules. No changes to existing code paths.

**Type safety where needed**: JSON schemas provide optional type checking for structured data while preserving flexibility for unstructured prompts.

**Industry-standard tooling**: CEL expressions compatible with Kubernetes validation, Google Cloud, and other cloud-native tools. Go templates familiar to Go developers.

**Performance**: CEL programs compiled once, reused for each execution. Templates parsed once per agent initialization. Workflow context passed by reference through `context.Context`.

## Accepting

**CEL dependency overhead**: Adds `github.com/google/cel-go` (~several MB) but provides battle-tested expression evaluation with proper sandboxing. Simpler alternatives (eval, simple parsers) would be less secure and less capable.

**Context.Value pattern limitations**: Type assertions required to extract workflow context. Must use typed accessor (`GetWorkflowContext(ctx)`) to avoid panics. Lost compile-time type checking for context contents.

**AST verbosity**: Three-phase architecture creates more code (~1000 lines) vs single-pass interpreter (~400 lines), but gains clear separation of concerns and better error messages.

**Runtime schema validation overhead**: JSON schema validation adds latency to each agent call, but provides crucial type safety for complex workflows. Can be disabled by omitting schema definitions.

**Template engine constraints**: Go templates less powerful than Jinja2/Liquid, but integrates natively without external dependencies. Learning curve for users unfamiliar with `{{.field}}` syntax.

**Markdown frontmatter format**: Using `\n---\n` separator is convention but not universal standard. Could conflict with markdown files containing horizontal rules in first lines, though mitigated by parser logic checking for YAML validity.
