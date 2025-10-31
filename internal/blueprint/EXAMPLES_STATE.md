# Example: State Management in Workflows

This example demonstrates how to use state management to pass data between workflow steps.

## Agent Definitions

### extract-entities.md
```markdown
---
name: entity-extractor
format: json
output_schema:
  type: object
  required: [entities]
  properties:
    entities:
      type: array
      items:
        type: object
        properties:
          name: {type: string}
          type: {type: string}
---
Extract named entities from the following text. Return as JSON with format: 
{"entities": [{"name": "...", "type": "..."}]}

Text: {{.current}}
```

### summarize.md
```markdown
---
name: summarizer
---
Create a summary of this text: {{.input}}

Previously extracted entities: {{range .steps.extracted.entities}}
- {{.name}} ({{.type}})
{{end}}
```

## Workflow Definition

```yaml
name: document-processor

jobs:
  process:
    steps:
      # Step 1: Extract entities and store result
      - name: extract
        uses: extract-entities.md
        output: extracted
      
      # Step 2: Summarize using original input and extracted entities
      - name: summarize
        uses: summarize.md
```

## How It Works

1. **Initial input**: Raw text document
2. **Step 1** (`extract`):
   - Receives: `{{.current}}` = raw text
   - Output stored as: `{{.steps.extracted}}`
3. **Step 2** (`summarize`):
   - Can access: 
     - `{{.input}}` = original text
     - `{{.steps.extracted}}` = entities from step 1
     - `{{.current}}` = last step output (entities)

## Benefits

- **No data loss**: Original input always available
- **Named references**: Clear which data you're using
- **Composability**: Build complex pipelines from simple steps
- **Debugging**: Each step's output is preserved and inspectable
