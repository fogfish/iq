# Example 03: Workflow State and Context

This example demonstrates how to access different parts of the workflow context in agent prompts.

## Workflow Visualization

```
Input: "The laptop features 3.5 GHz processor, 16GB RAM, 1TB SSD"
  ↓
  ├─ Stored as: {{.document}}
  │
  ▼
┌─────────────────────────────────────────────────────────┐
│ Step 1: Extract                                         │
│ - Receives: {{.input}} (= document)                     │
│ - Returns: {specs: "...", category: "laptop"}          │
│ - Stored as: {{.steps.extract}}                         │
└─────────────────────────────────────────────────────────┘
  ↓
  ▼
┌─────────────────────────────────────────────────────────┐
│ Step 2: Transform                                       │
│ - Receives: {{.input}} (= steps.extract)                │
│ - Accesses: {{.input.specs}}                            │
│ - Returns: {cpu: "3.5 GHz", memory: "16GB", ...}       │
│ - Stored as: {{.steps.specs}}                           │
└─────────────────────────────────────────────────────────┘
  ↓
  ▼
┌─────────────────────────────────────────────────────────┐
│ Step 3: Summarize                                       │
│ - Accesses: {{.document}} (original)                    │
│ - Accesses: {{.steps.extract.category}} ("laptop")      │
│ - Accesses: {{.steps.specs.*}} (all fields)            │
│ - Returns: Final summary text                          │
└─────────────────────────────────────────────────────────┘
```

## Context Keys Available in Templates

- `{{.document}}` - Original workflow input (never changes)
- `{{.input}}` - Agent's input (current step value)
- `{{.current}}` - Alias for `.input` (workflow perspective)
- `{{.steps.name}}` - Access output from named steps
- `{{.state.key}}` - Shared workflow state (for custom data)

## Example Flow

1. **Extract** (`extract.md`):
   - Receives: `{{.input}}` = original document text
   - Returns: `{specs: "...", category: "..."}`
   - Stored as: `{{.steps.extract}}`

2. **Transform** (`transform.md`):
   - Receives: `{{.input}}` = output from extract step
   - Accesses: `{{.input.specs}}`
   - Returns: `{cpu: "...", memory: "...", storage: "..."}`
   - Stored as: `{{.steps.specs}}`

3. **Summarize** (`summarize.md`):
   - Accesses: `{{.document}}` - original text
   - Accesses: `{{.steps.extract.category}}` - from step 1
   - Accesses: `{{.steps.specs.cpu}}` - from step 2
   - Creates final summary using all context

## Key Concepts

### Agent Schema vs Template Variables

Agents define schemas for validation:
```yaml
schema:
  input:  # Validates what the agent receives
  reply:  # Validates what the agent returns
```

Templates reference context:
```markdown
{{.input}}           # Agent's input (matches schema.input)
{{.document}}        # Original workflow input
{{.steps.name}}      # Previous step outputs
```

### Named Outputs

Use `output:` to store step results:
```yaml
steps:
  - name: extract
    uses: extract.md
    output: extraction    # Store as {{.steps.extraction}}
```

### Accessing Nested Data

JSON objects can be accessed with dot notation:
```markdown
{{.steps.extract.category}}
{{.input.specs}}
{{.steps.specs.cpu}}
```

## Running the Example

```bash
iq task -a examples/03_state/run.yml examples/03_state/content/doc.txt
```

This demonstrates how agents can collaborate by storing and sharing data through the workflow context.
