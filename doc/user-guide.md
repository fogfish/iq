# User Guide

Build intelligent, multi-step agentic workflows with declarative YAML blueprints.

---

## Table of Contents

- [User Guide](#user-guide)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
    - [Key Features](#key-features)
  - [Core Concepts](#core-concepts)
    - [Workflow](#workflow)
    - [Jobs](#jobs)
    - [Steps](#steps)
    - [Agents](#agents)
    - [Context \& State](#context--state)
  - [Getting Started](#getting-started)
    - [Your First Workflow](#your-first-workflow)
  - [Blueprint Syntax Reference](#blueprint-syntax-reference)
    - [Blueprint Structure](#blueprint-structure)
    - [Job Definition](#job-definition)
    - [Step Types](#step-types)
      - [Prompting Step](#prompting-step)
      - [Command Step](#command-step)
      - [Routing Step](#routing-step)
      - [Iterating Step](#iterating-step)
    - [Agent Format](#agent-format)
      - [JSON Schema Validation](#json-schema-validation)
      - [Remote MCP Servers Integration](#remote-mcp-servers-integration)
      - [Remote MCP Servers Authentication](#remote-mcp-servers-authentication)
  - [Workflow Patterns](#workflow-patterns)
  - [Best Practices](#best-practices)
    - [Workflow Design](#workflow-design)
    - [Prompt Engineering](#prompt-engineering)
    - [Maintenance](#maintenance)
  - [Troubleshooting](#troubleshooting)
    - [Common Issues](#common-issues)
    - [Debugging Tips](#debugging-tips)
    - [Getting Help](#getting-help)

---

## Introduction

`iq` brings multi-step autonomous AI workflows to your shell — no coding required. Describe what you want to achieve in simple YAML, and `iq` handles the logic, execution, state, and recovery for you.


### Key Features

* **No-code workflow design** describe automations in simple, declarative YAML no programming required.
* **Goal-driven workflows** orchestrates multiple AI-powered steps into coherent agentic workflow.
* **Chaining, Routing, Iterating and State management** is natively supported using flexible, rule-based language.
* **Native Model Context Protocol (MCP) integration and server modes** connect and orchestrate external tools directly within your workflows, and optionally run a workflow as an MCP server.
* **Fail-safe execution** recover from errors with built-in supervisor and fallback strategies.


## Core Concepts

### Workflow

A **workflow** is a YAML file that defines your agent. Besides various metadata, it specifies one or more jobs to be executed. The YAML format conceptually similar to GitHub Actions.

```yaml
name: my-workflow
about: A description of what this workflow does
entrypoint: main  # Optional: defaults to "main"

jobs:
  main:
    steps:
      - uses: prompts/step1.md
      - uses: prompts/step2.md
```

### Jobs

A **job** is a sequence of steps that execute in order. Jobs can:

- Be called from other jobs (via routing or foreach)
- Accept input from the previous step or router
- Return output to the caller
- Execute on specific LLM models (via `runs-on`) (no supported yet)

```yaml
jobs:
  main:
    steps:
      - uses: prompts/classify.md
      
  handler:
    steps:
      - uses: prompts/process.md
```

### Steps

A **step** is a single unit of work powered by LLM:
* **prompting** executes an autonomous step depicted by prompt and tools;
* **roting** executes the adaptive decision making, branching the workflow to other jobs;
* **iterating** executes a job for array of sub-tasks. 

```yaml
steps:
  # prompting
  - uses: prompts/extract.md
    output: data
    
  # routing  
  - uses: prompts/classify.md
    switch:
      - when: choice == "urgent"
        route: urgent-handler
    default: normal-handler
    
  # itearting
  - uses: prompts/generate-list.md
    foreach:
      job: process-item
```

### Agents

Each step is "nano agent" depicted by prompt in a markdown format. The prompt file consists of an optional YAML frontmatter with configuration and prompt itself with placeholders.

**Simple agent** (prompt only):
```markdown
Extract the key information from: {{.input}}
```

**Full agent** (with configuration):
```markdown
---
format: json
schema:
  reply:
    type: object
    properties:
      category: {type: string}
---
Classify the following text: {{.input}}
```

### Context & State

The workflow engine maintains execution context accessible to all steps:
- `{{.document}}` original workflow input (immutable)
- `{{.input}}` current input to the agent either text or structure 
- `{{.current}}` output from the most recent step
- `{{.steps.name}}` named outputs from previous steps (aka global state)

Accessing the workflow contex from prompt is build on simple templating language:
```markdown
{{.input}} equals {{.document}} equals {{.current}}

Title: {{.steps.extract.title}}
Date: {{.steps.extract.date}}

{{range .steps.items}}
- {{.name}}: {{.value}}
{{end}}
```


## Getting Started

### Your First Workflow

Let's create a simple two-step workflow that extracts information and then summarizes it.

**1. Create the workflow** (`workflow.yml`):

```yaml
name: extract-and-summarize
about: Extract information from text and create a summary

jobs:
  main:
    steps:
      - name: extract
        uses: prompts/extract.md
        output: data
        
      - name: summarize
        uses: prompts/summarize.md
```

**2. Create the extraction agent** (`prompts/extract.md`):

```markdown
---
format: json
schema:
  reply:
    type: object
    properties:
      facts: 
        type: array
        items: {type: string}
---
Extract key facts from the following text as a JSON array: {{.input}}
```

**3. Create the summary agent** (`prompts/summarize.md`):

```markdown
Create a brief summary based on these facts:

{{range .steps.extract.facts}}
- {{.}}
{{end}}
```

**4. Run the workflow**:

```bash
iq agent -a workflow.yml "The laptop features a 3.5 GHz processor..."
```

(See [full example of this workflow](../examples/09_templates/))


## Blueprint Syntax Reference

### Blueprint Structure

```yaml
# Required: workflow name
name: my-workflow

# Optional: description
about: What this workflow does

# Optional: default job to run (defaults to "main")
entrypoint: main

# Optional: default LLM for all jobs (Not Supported)
runs-on: gpt-4

# Optional: input/output schemas for the entire workflow
schema:
  input:
    type: string
  reply:
    type: object
    properties:
      result: {type: string}

# Required: job definitions
jobs:
  job-name:
    # Job definition here
```

### Job Definition

```yaml
jobs:
  my-job:
    # Optional: override workflow's default LLM (Not Supported)
    runs-on: claude-3

    # Required: list of steps
    steps:
      - name: step1
        uses: prompts/agent.md
        output: result1
        
      - name: step2
        uses: prompts/agent2.md
```

### Step Types

#### Prompting Step

Execute an LLM agent:

```yaml
- name: extract        # Optional: step name
  uses: prompts/extract.md  # Required: path to agent file
  output: extracted    # Optional: store result with this name
  retry:               # Optional: retry configuration
    attempts: 3
    delay: 2
    yield: prompts/fallback.md
```


#### Command Step

Execute shell commands directly in your workflow:

```yaml
- name: fetch-page       # Optional: step name
  run: curl -sL $(echo '{{.current}}' | tr -d '\n')  # Shell command with template variables
  output: html           # Optional: store result with this name
  retry:                 # Optional: retry configuration
    attempts: 3
    delay: 2
  runs-on: bash          # Optional: shell to use (default: sh)
```

**Template Variables in Commands:**
- `{{.current}}` - Output from previous step (or input for first step)
- `{{.document}}` - Original workflow input
- `{{.steps.name}}` - Named outputs from previous steps
- `{{.state.key}}` - Workflow state values

**Examples:**

```yaml
# Simple command
- name: list-files
  run: ls -la

# With template variable
- name: fetch
  run: curl -sL '{{.current}}'
  output: content

# Multi-line command
- name: process
  run: |
    echo '{{.current}}' | \
    grep -oP 'href="\K[^"]+' | \
    head -10
  output: links

# Using previous step output
- name: transform
  run: echo '{{.steps.data}}' | jq '.results[]'
```

**Notes:**
- Commands execute in `sh` by default (configure with `runs-on: bash`)
- Stdout is captured as the step output
- Non-zero exit codes trigger retries or failures
- Stderr is included in error messages
- Wrap template variables in quotes to handle special characters
- Input may include newlines - trim with `tr -d '\n'` if needed


#### Routing Step

Conditional routing based on LLM output:

```yaml
- name: classifier
  uses: prompts/classify.md  # Agent that produces output to evaluate, can be skipped
  output: classification     # Optional: store the classification result
  switch:
    - when: choice == "category-a"    # CEL expression
      route: handle-category-a        # Target job name
    - when: choice == "category-b"
      route: handle-category-b
    - when: choice.contains("urgent")
      route: urgent-handler
  default: default-handler   # Fallback if no route matches
  retry:                     # Optional: retry the classification
    attempts: 2
```
Router conditions use [CEL (Common Expression Language)](https://github.com/google/cel-spec):

**Available variables**:
- `choice`: Output from the router agent
- `steps`: Named step outputs (`steps.extract.category`)

**Examples**:

```yaml
# Simple equality
- when: choice == "urgent"
  route: urgent-handler

# String operations
- when: choice.contains("error")
  route: error-handler

# Numeric comparison
- when: steps.score.value > 0.8
  route: high-confidence

# Complex conditions
- when: choice == "laptop" && steps.price.amount < 1000
  route: budget-laptop

# Type checking
- when: choice.startsWith("category:")
  route: categorized
```


#### Iterating Step

Process arrays by executing a job for each item:

```yaml
- name: process-all
  uses: prompts/generate-list.md  # Optional: agent to create the array
  foreach:
    job: process-item        # Job to execute per item
  output: results            # Optional: store array of results
  retry:                     # Optional: retry array generation
    attempts: 2
```

If `uses` is omitted, the current value must be an array:

```yaml
- name: process-current-array
  foreach:
    job: process-item
```

Use `selector` to extract arrays from JSON objects:

```yaml
- name: process-users
  foreach:
    selector: "current.users"           # Extract from current context
    job: process-user

- name: process-items
  foreach:
    selector: "steps.data.items"     # Extract from step results
    job: process-item
```

### Agent Format


```markdown
---
# Optional: force JSON output
format: json

# Optional: input/output validation
schema:
  input:
    type: string
  reply:
    type: object
    properties:
      result: {type: string}

# Optional: MCP servers for tool access
servers:
  - name: my-tool
    command:
      - ./tool
      - --serve
---
Your prompt template here: {{.input}}
```

#### JSON Schema Validation

Define input and output schemas for type safety:

```yaml
---
format: json
schema:
  input:
    type: object
    required: [query, max_results]
    properties:
      query: {type: string, minLength: 1}
      max_results: {type: integer, minimum: 1, maximum: 100}
  reply:
    type: object
    required: [results, count]
    properties:
      results:
        type: array
        items:
          type: object
          properties:
            title: {type: string}
            score: {type: number}
      count: {type: integer}
---
Search for: {{.input.query}}
Limit: {{.input.max_results}}
```

#### Remote MCP Servers Integration

Enable tool access via Model Context Protocol servers:

```markdown
---
servers:
  - name: filesystem
    command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "./"]

  - name: weather
    command: ["python", "weather_server.py"]

  - name: calculator
    command: ["./calc-service", "--port", "8080"]
---
Get weather for San Francisco and calculate temperature in Celsius.
```

The LLM can discover and invoke tools provided by these servers.

#### Remote MCP Servers Authentication 

For MCP servers that require authentication (such as OAuth2-protected endpoints), configure Bearer token credentials in your `~/.iqrc` file using netrc format:

```bash
machine api.example.com
  secret your-bearer-token-here
```

Then reference the authenticated server by URL:

```markdown
---
servers:
  - name: github
    url: https://api.githubcopilot.com/mcp/
---
Use the authenticated API to fetch data for: {{.input}}
```

The system automatically adds `Authorization: Bearer <token>` headers to requests to the specified host. Use the `secret` field for the Bearer token value.


## Workflow Patterns

See examples about possible patterns:

* [Sequential Processing](../examples/01_chain/run.yml)
* [Conditional Routing](../examples/04_routing/run.ymls)
* [Iteration](../examples/06_foreach/run.yml)
* [Error Recovery](../examples/07_retry/run.yml)
* [Global state](../examples/03_state/run.yml)
* [JSON Schema validaton](../examples/02_json_schema/run.yml)
* [MCP tools and server](../examples/08_mcp/run.yml)
* [Shell Commands](../examples/11_command/run.yml)


## Best Practices

### Workflow Design

1. **Keep jobs focused**: Each job should have a single responsibility
2. **Use descriptive names**: Name steps and outputs clearly (`extract`, not `step1`)
3. **Minimize router complexity**: Prefer simple conditions over complex CEL expressions
4. **Design for failure**: Add retry logic to steps that interact with external services
5. **Validate early**: Use JSON Schema for structured data extraction

### Prompt Engineering

1. **Be specific**: Clear, detailed prompts produce better results
2. **Provide examples**: Show expected input/output format in prompts
3. **Use templates wisely**: Reference only the context you need
4. **Request structured output**: Use `format: json` with schema for parsing
5. **Test incrementally**: Build workflows step by step, testing each addition

> [!TIP]
> [TELeR framework](https://aclanthology.org/2023.findings-emnlp.946.pdf) — a practical taxonomy that breaks prompts into clear components: Task, Environment, Learner, and Response. This approach helps you craft reusable prompts by clearly defining goals, constraints, tone, and expected outputs. Use it to improve prompt quality, automate workflows, and ensure consistent LLM behavior across files and tasks.
> 

Use `iq draft` command to create an empty structured prompt YAML file:

```yaml
prompt: |
  [Describe the task and goals clearly and concisely].
  
  Guidelines:
    (1) [High-level principles or approach to follow.]
    (2) ...

  Strictly adhere to the following requirements when generating a response.
  Do not deviate, ignore, or modify any aspect of them:
    1. [Concrete requirement]
    2. [Another specific rule]
    ...

  Example Input:
  [Show an example of what the input might look like.]

  Expected Output:
  [Demonstrate the ideal format or structure of the response.]

  Additional Context:
    - [Relevant detail #1]
    - [Constraint or domain knowledge #2]
    - ...

  Input:
    [Insert the actual input here]
```


### Maintenance

1. **Version your workflows**: Track changes to blueprints and prompts
2. **Document complex logic**: Add comments in YAML and prompt files
3. **Use consistent structure**: Organize prompts in directories by purpose
4. **Review generated outputs**: Spot-check results to catch prompt drift


## Troubleshooting

### Common Issues

**"entrypoint job not found"**
- Ensure you have a job named `main` or specify `entrypoint: your-job-name`
- Check for typos in job names

**"agent file not found"**
- Verify the path in `uses:` is relative to the blueprint file
- Check file extension (`.md` or `.markdown`)

**"no matching route"**
- Add a `default:` job to router steps
- Check CEL expressions with simpler conditions
- Verify router agent returns expected output format

**"validation failed"**
- Review JSON Schema definition for typos
- Check LLM output format matches schema
- Use `format: json` to request structured output

**"context not found"**
- Ensure named outputs match step names
- Check template variable syntax (`{{.steps.name}}`)
- Verify step executed before being referenced

### Debugging Tips

1. **Test agents individually**: Run single-agent workflows first
2. **Inspect intermediate outputs**: Add `output:` names to examine step results
3. **Simplify routing**: Start with 2 routes, add complexity gradually
4. **Validate schemas separately**: Test schema with known-good data
5. **Check LLM selection**: Ensure model supports required features (tools, JSON mode)

### Getting Help

- Review the [examples](../examples/) directory
- Check the [architecture documentation](../internal/blueprint/ARCHITECTURE.md)
- Open an issue on GitHub with:
  - Blueprint YAML
  - Agent prompts
  - Error messages
  - Expected vs actual behavior

