# Parser → AST → Compiler Architecture Implementation

## Overview

Successfully implemented a clean separation of concerns for workflow processing with three distinct phases:
1. **Parser**: YAML → AST
2. **AST**: Immutable intermediate representation
3. **Compiler**: AST → Executable functions

## Architecture

```
YAML Files → Parser → AST → Compiler → Executable Workflow
                      (validate)      (execute)
```

## Components

### 1. AST Package (`internal/blueprint/ast/`)

Defines immutable data structures representing the workflow:

- **AST**: Root container for blueprint and agents
- **BlueprintNode**: Workflow definition with jobs
- **JobNode**: Individual job with steps
- **StepNode**: Interface for agent and router steps
  - **AgentStepNode**: Simple agent execution
  - **RouterStepNode**: Conditional routing with CEL expressions
- **AgentNode**: Agent definition with prompt, schema, servers
- **ServerNode**: MCP server configuration
- **RouteNode**: CEL condition + target job

### 2. Parser Package (`internal/blueprint/parser/`)

Converts YAML to AST:

- **Parser**: Main parsing logic
- Handles blueprint YAML files
- Parses agent files (frontmatter + prompt)
- Supports both formats:
  - Simple: Just prompt text
  - Full: YAML frontmatter + prompt separated by `\n---\n`
- Resolves relative file paths
- No validation or compilation logic

### 3. Compiler Package (`internal/blueprint/compiler/`)

Validates and compiles AST to executable workflow:

- **Compiler**: Main compilation logic
- **Workflow**: Compiled workflow with jobs
- **Job**: Executable job (implements `AI` interface)
- **Step**: Interface for executable steps
  - **AgentStep**: Executes agent
  - **RouterStep**: Evaluates CEL, routes to appropriate job

**Compilation phases**:
1. Semantic validation
   - Route references point to existing jobs
   - CEL expressions are syntactically valid
   - All agent files are present
2. Agent initialization (Run with LLM)
3. CEL expression compilation
4. Reference resolution (routes → jobs)

### 4. Blueprint Package (`internal/blueprint/`)

Public API that orchestrates Parser → Compiler:

```go
// Load and compile blueprint
bp, err := blueprint.New(ctx, "workflow.yml", factory)

// Execute specific job
result, err := bp.Execute(ctx, "job-name", input)

// Get compiled job (implements AI interface)
job, err := bp.GetJob("job-name")
```

## Key Features

### CEL-Based Routing

Router steps use Common Expression Language for conditions:

```yaml
steps:
  - name: classifier
    uses: choice.md
    switch:
      - when: choice == "life"
        route: life-handler
      - when: choice == "none"
        route: fea-handler
    default: unknown-handler
```

**Benefits**:
- Industry standard (Kubernetes, Google Cloud)
- Expressive: `choice.contains("text") && confidence > 0.8`
- Safe: Sandboxed execution
- Type-aware: Works with structured data

### Proper Error Handling

Errors are categorized by phase:

- **Parse errors**: Syntax errors, missing files, malformed YAML
- **Compile errors**: Invalid references, bad CEL expressions, type mismatches
- **Runtime errors**: LLM failures, CEL evaluation errors

### Clean Interfaces

- `Factory`: Provides LLM instances
- `AI`: Common interface for agents, jobs, and steps
- All compiled components implement `Prompt(ctx, input) (output, error)`

### Path Resolution

Parser handles relative paths correctly:
- Blueprint file location is the base directory
- Agent files resolved relative to blueprint

## Migration from Old Code

### Old API:
```go
bp, _ := blueprint.New("file.yml")
app, _ := bp.Run(ctx, factory)
result, _ := app["job"].Prompt(ctx, input)
```

### New API:
```go
bp, _ := blueprint.New(ctx, "file.yml", factory)
result, _ := bp.Execute(ctx, "job", input)
// OR get job directly:
job, _ := bp.GetJob("job")
result, _ := job.Prompt(ctx, input)
```

### Factory Changes:

**Old**:
```go
type Factory interface {
    LLM(name string) (chatter.Chatter, error)
    Agent(name, spec string) (*agent.Agent, error)
}
```

**New**:
```go
type Factory interface {
    LLM(name string) (chatter.Chatter, error)
    // Agent creation is internal to compiler
}
```

### YAML Format Changes:

**Old** (map-based routes):
```yaml
switch:
  life: life-handler
  none: fea-handler
```

**New** (CEL-based routes):
```yaml
switch:
  - when: choice == "life"
    route: life-handler
  - when: choice == "none"
    route: fea-handler
default: unknown-handler
```

## Example Usage

See `internal/blueprint/main/main.go`:

```go
func main() {
    ctx := context.Background()
    
    bp, err := blueprint.New(ctx, "bp.yml", &factory{})
    if err != nil {
        log.Fatalf("failed to load blueprint: %v", err)
    }

    result, err := bp.Execute(ctx, "test", nil)
    log.Printf("==> %v\n%v\n", err, result)
}
```

## Testing

Each component can be tested independently:

- **Parser**: Test YAML → AST conversion
- **AST**: Test tree structure and validation
- **Compiler**: Test AST → executable compilation with mock factory
- **Integration**: Test full Parser → Compiler → Execute flow

## Implemented Features

### 1. JSON Schema Validation ✅

Agents support optional `input_schema` and `output_schema` for runtime type checking:

```yaml
---
name: parser
input_schema:
  type: string
output_schema:
  type: object
  required: [title, content]
  properties:
    title: {type: string}
    content: {type: string}
---
Parse the document: {{.current}}
```

Validation happens:
- **Before execution**: Input validated against `input_schema`
- **After execution**: Output validated against `output_schema`
- **Error reporting**: Clear messages indicating which field failed validation

### 2. Workflow State Management ✅

Workflows maintain execution context accessible to all steps:

**Context Structure:**
```go
type Context struct {
    Input   any            // Original workflow input
    State   map[string]any // Shared key-value store
    Steps   map[string]any // Named step outputs
    Current any            // Most recent step output
}
```

**YAML Syntax:**
```yaml
steps:
  - name: extract
    uses: extractor.md
    output: entities  # Store output with name

  - name: enrich
    uses: enricher.md
    # Can reference previous outputs in template
```

**Template Access:**
Agent prompts can access full context:

```markdown
---
name: enricher
---
Enrich these entities: {{.steps.entities}}

Original input was: {{.input}}
Previous step output: {{.current}}
```

**Available in templates:**
- `.input` - Original workflow input
- `.current` - Most recent step output (default if no output name)
- `.steps.stepname` - Named step output
- `.state.key` - Shared workflow state

## Future Enhancements

Based on the initial design review, remaining additions:

1. ~~**Type Safety** (Option 3): Add JSON Schema validation in agent definitions~~ ✅ DONE
2. **Error Recovery**: Retry policies, fallback routes
3. ~~**State Management**: Named outputs, workflow context~~ ✅ DONE  
4. **Parallel Execution**: DAG-based job scheduling
5. **Observability**: Execution tracing, timing, token usage
6. **Dry-run Mode**: Validate without executing

## File Structure

```
internal/blueprint/
├── ast/
│   └── ast.go              # AST node definitions
├── parser/
│   └── parser.go           # YAML → AST conversion
├── compiler/
│   ├── compiler.go         # AST validation & compilation
│   └── workflow.go         # Executable workflow structures
├── blueprint.go            # Public API
└── factory.go              # Factory adapter
```

## Dependencies Added

- `github.com/google/cel-go` v0.26.1 - CEL expression evaluation
