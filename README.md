<p align="center">
  <img src="./doc/iq-v3.svg" height="256" />
  <p align="center"><strong>AI workflows for the shell — no code required</strong></p>
</p>

<p align="center">
  <a href="https://github.com/fogfish/iq/releases">
    <img src="https://img.shields.io/github/v/tag/fogfish/iq?label=version" />
  </a>
  <a href="https://github.com/fogfish/iq/actions/">
    <img src="https://github.com/fogfish/iq/workflows/build/badge.svg" />
  </a>
  <a href="http://github.com/fogfish/iq">
    <img src="https://img.shields.io/github/last-commit/fogfish/iq.svg" />
  </a>
  <a href="https://coveralls.io/github/fogfish/iq?branch=main">
    <img src="https://coveralls.io/repos/github/fogfish/iq/badge.svg?branch=main" />
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" />
  </a>
</p>

---

`iq` brings multi-step autonomous AI workflows to your shell — no coding required. Describe what you want to achieve in simple YAML, and `iq` handles the logic, execution, state, and recovery for you.


## Features

* **No-code workflow design** describe automations in simple, declarative YAML no programming required.
* **Goal-driven workflows** orchestrates multiple AI-powered steps into coherent agentic workflow.
* **Chaining, Routing, Iterating and State management** is natively supported using flexible, rule-based language.
* **Native Model Context Protocol (MCP) integration and server modes** connect and orchestrate external tools directly within your workflows, and optionally run a workflow as an MCP server.
* **Fail-safe execution** recover from errors with built-in supervisor and fallback strategies.
* **Automatic caching** reduces costs and speeds up execution by caching LLM responses with intelligent invalidation.
* **Flexible I/O** seamless input/output into stdio, files, directories, S3, SQS and many other.   
* **Support multiple LLM providers** works with OpenAI, AWS Bedrock, and local models.



## Installation

<details>
<summary>macOS - Homebrew (Recommended)</summary>

```bash
brew tap fogfish/iq https://github.com/fogfish/iq
brew install iq
```

Upgrade to latest version 

```bash
brew upgrade iq
```

</details>


<details>
<summary>Direct binaries download (Linux, Windows)</summary>

Download the executable from the [latest release](https://github.com/fogfish/iq/releases/latest) and add it to your PATH.

</details>


<details>
<summary>Build from sources</summary>

Requires Go [installed](https://go.dev/doc/install).

```bash
go install github.com/fogfish/iq@latest
```

</details>



## Configuration

The utility requires access to LLMs for execution. LLM access config and credentials are stored in `~/.iqrc`. 

<details>
<summary>AWS Bedrock (Recommended)</summary>

Requires AWS account with access to [AWS Bedrock service](https://aws.amazon.com/bedrock/).

```bash
iq config --bedrock
```

The default config uses `global.anthropic.claude-sonnet-4-5-20250929-v1:0` inference profile. Use `-m`, `--llm-id` flags to override the default model.

```bash
iq config --bedrock -m amazon.nova-pro-v1:0
```

</details>

<details>
<summary>Open AI</summary>

Requires account at [Open AI platform](https://platform.openai.com) and access key for api usage.

```bash
iq config --openai <api-key>
```

The default config uses `gpt-5` model. Use `-m`, `--llm-id` flags to override the default model.

```bash
iq config --openai <api-key> -m gpt-4o
```

</details>

<details>
<summary>Local AI, on Your Computer</summary>

[LM Studio](https://lmstudio.ai) is recommended approach to run LLMs locally. After LM Studio installation and installation of desired model, you can configure `iq`

```bash
iq config --lmstudio
```

The default config uses `gemma-3-27b-it` model. Use `-m`, `--llm-id` flags to override the default model.

```bash
iq config --lmstudio -m gpt-oss
```

</details>


## Quick Start

Let's build your first AI workflow with `iq`.
Here;s a minimal example that researches a topic and summarizes it using two AI agents:

```yaml
name: research
jobs:
  main:
    steps:
      - prompt: |
          Find three key facts about the topic:
          {{.input}}.

      - prompt: |
          Summarize following facts clearly in 2–3 sentences:
          {{.input}}
```

Run the workflow from your shell, pass the topic you want to research via stdio:

```bash
echo "singularity" | iq agent -f research.yml
```

**Enable caching** to save time and costs on repeated runs:

```bash
# First run - executes workflow and caches results
echo "singularity" | iq agent -f research.yml --cache-dir .cache

# Second run - instant, uses cached results
echo "singularity" | iq agent -f research.yml --cache-dir .cache
```

Caching is automatic—no changes to your workflow YAML needed. Cache entries are invalidated automatically when prompts change.

See [**User Guide**](./doc/user-guide.md) about the workflow syntax. 


## Usage

```bash
# run workflow
iq agent -f <yaml> FILE1, FILE2, ...

# draft prompts markdown and agent workflows
iq draft
iq draft agent
```

Use `iq help` for details

### Options

**Input**

- *(stdin)* Read input data from standard input 
- `FILE ...` One or more input files (local or s3:// paths)
- `-I <dir>`, `--input-dir` Process all files in a directory (local or s3:// paths)

**Input modifiers**
- `--json` Display output as formatted, colored JSON
- `--merge` Combine all input files into a single document before processing

**Output**

- *(stdout)* Write output data to standard output
- `-o <file>`, `--file` Write output to a single file (supports s3:// paths)
- `-O <dir>`, `--output-dir` Write each result to a file in a directory or S3 bucket 
 

### Batch processing

`batch` command treats a mounted directory of files as a processing queue—reading from an input directory, applying each file content as input to the workflow, and writing the results to an output directory. This batch-oriented processing is ideal for transformation, summarization or enhanced file processing at scale—with minimal setup and full traceability of inputs and outputs.

The command support mounting of AWS S3 bucket. Use `s3://` prefix prefix to direct the utility.

```bash
iq agent batch -f <prompt> -I s3://... -O s3://...
```

Processing a large number of files may require the ability to start, stop, and resume the utility reliably. To support this, you can use the `--mutable` flag, which removes each input file immediately after it has been successfully processed. This enables fault-tolerant, resumable execution by ensuring already-processed files are skipped on subsequent runs.

Use `--strict` to fail fast, terminating the processing on the first error.


### Chunking Large Files

Process files in chunks:

```bash
# Split by sentences
iq agent -f workflow.yml --splitter sentence large-doc.txt

# Split by paragraphs
iq agent -f workflow.yml --splitter paragraph large-doc.txt

# Fixed-size chunks
iq agent -f workflow.yml --splitter chunk --splitter-chunk 2048 large-doc.txt
```

**Sentences**: A sentence is defined as a punctuation mark (., !, or ?) followed by a whitespace character. The default punctuation marks are overwritten with `--splitter-chars` flag.

**Paragraphs**: A paragraph is defined as a block of text separated by an empty line (essentially using `\n\n` as delimiter). The default delimiter is overwritten with `--splitter-chars` flag.

**Fixed chunks**: A fixed chunk has a defined size limit, which is extended to include the end of the nearest sentence (it prevents loss of context). The chunk size is configured with `--splitter-chunk` flag and `--splitter-chars` are used to define punctuation marks similar to sentence split.


### Servers

TBD.

#### Model-Context-Protocol

Expose your workflow as a Model Context Protocol server:

```bash
iq agent serve -a workflow.yml
```

Requires workflow with `name` and schemas defined.

**Connect external tools to your workflows**

```markdown
---
servers:
  - type: stdio
    name: filesystem
    command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "./"]
---
Using the filesystem tool, read the file {{.input}} and summarize it.
```


## Examples

The [`examples/`](examples/) directory contains complete workflow examples.

## Documentation

- **[User Guide](doc/user-guide.md)** — Complete workflow design guide


## Contributing

`iq` is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request


## License

[![See LICENSE](https://img.shields.io/github/license/fogfish/iq.svg?style=for-the-badge)](LICENSE)

