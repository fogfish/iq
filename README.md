<p align="center">
  <img src="./doc/iq.svg" height="128" />
  <h3 align="center">iq</h3>
  <p align="center"><strong>Intelligent Query is a lightweight command-line LLM-powered file processor.</strong></p>

  <p align="center">
    <!-- Version -->
    <a href="https://github.com/fogfish/iq/releases">
      <img src="https://img.shields.io/github/v/tag/fogfish/iq?label=version" />
    </a>
    <!-- Build Status  -->
    <a href="https://github.com/fogfish/iq/actions/">
      <img src="https://github.com/fogfish/iq/workflows/build/badge.svg" />
    </a>
    <!-- GitHub -->
    <a href="http://github.com/fogfish/iq">
      <img src="https://img.shields.io/github/last-commit/fogfish/iq.svg" />
    </a>
    <!-- Coverage -->
    <a href="https://coveralls.io/github/fogfish/iq?branch=main">
      <img src="https://coveralls.io/repos/github/fogfish/iq/badge.svg?branch=main" />
    </a>
  </p>
</p>

--- 

Intelligent Query `iq` is a lightweight command-line LLM-powered file processor. The tool is designed to simplify running LLM-powered agents in the batch against mounted file systems, whether they are simple prompting agents or more complex system that dynamically direct their own processes and tool to accomplish task.

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [Prompt Engineering](#prompt-engineering)
- [Examples](#examples)
  - [Basic usage](#basic-usage)
  - [Processing file](#processing-file)
  - [Basic agent](#basic-agent)
  - [Processing files with agent](#processing-files-with-agent)
  - [Classification](#classification)
  - [Processing large files](#processing-large-files)
  - [Working with AWS S3](#working-with-aws-s3)
  - [STDIN/STDOUT](#stdinstdout)
- [How To Contribute](#how-to-contribute)
- [License](#license)


## Features

* Apply prompt-based agents to files within a directory.
* Execute workflow-driven tools on directory-based files.
* Enable LLMs to utilize Bash, Golang, and Python for executing tasks.
* Support prompt templates and meta-prompting (prompts generated dynamically by the LLM).
* Enable STDIN/STDOUT for shell scripting integration.
* Support OpenAI, OpenAI-compatible APIs, AWS Bedrock, and LM Studio as LLM backends.


## Installation

Install utility using either from [Homebrew](https://brew.sh) (MacOS) or [GitHub Binary Releases](https://github.com/fogfish/iq/releases) (other platforms).

```bash
## Install using brew
brew tap fogfish/iq https://github.com/fogfish/iq
brew install -q iq

## use `brew upgrade` to upgrade to latest version 
```

Alternatively, install from source code, it requires Golang 1.24
```bash
go install github.com/fogfish/iq@latest
```

Before using `iq`, you need to configure it with access to an LLM provider. The tool supports multiple backends, including Amazon Bedrock, OpenAI, OpenAI-compatible APIs, and local LM Studio instances. The tool keeps the config at the `.netrc` file that generally resides in the user's home directory.

It is your responsibility to obtain access to LLMs either creating the necessary accounts or installed any required software (such as LM Studio), and verified access to the target model.


Use following commands to autoconfig `.netrc` record.
```bash
iq config --bedrock
iq config --lmstudio
iq config --openai <secret-key>
```

> [!TIP]
> My personal recommendation is usage of Amazon Bedrock. 

## Quick Start

```bash
echo "What are the colors of rainbow?" | iq tell
```

> [!TIP]
> use -m, --llm flags to override the default model

## Usage

```bash
iq help

Available Commands:
  ask         process files in mounted dir with LLM
  config      configure utility
  draft       generate prompt template
  run         process files in mounted dir with LLM agent
  tag         classify files in the current directory using LLM
  task        execute LLM-agent with prompt instructions
  tell        send a prompt to LLM

Flags:
  -c, --config string   config profile at ~/.netrc about LLM provider (default "iq")
      --input string    override prompt input
  -m, --llm string      overrides LLM model defined at ~/.netrc
  -p, --prompt string   path to prompt yaml file

Use "iq [command] --help" for more information about a command.
```

### Prompt Engineering

Prompts are a core component of how the `iq` utility operates.  It expects prompts to be defined in YAML files, with the `prompt` key containing the actual prompt text.

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

`iq` support [prompt templating](./examples/prompt/03_withargs.yml) throught structured `input` key and [Golang templating syntax](https://www.digitalocean.com/community/tutorials/how-to-use-templates-in-go). 

```yaml
input:
  name: rainbow

prompt: |
  What are the colors of {{.name}}?
```

In addtion, you can "try" to force [LLM responding in JSON](./examples/prompt/02_jsonify.yml) and build reusable prompt via [include directive](./examples/prompt/04_include.yml).

> [!TIP]
> Meta-prompting is possible as well. The command below uses 
> input text to generate a prompt
> 
> `echo "What are the colors of rainbow?" | iq draft`


## Examples

### Basic usage

`iq tell` send a standalone prompt and receive an immediate response, ideal for asking questions, drafting text, or running quick ideas past the model.

For example, running the prompt [What are the colors of the rainbow](./examples/prompt/01_basic.yml)

```bash
iq tell -p ./examples/prompt/01_basic.yml
```


### Processing file

`iq ask` command treats a mounted directory of files as a processing queue—reading from an input directory, applying LLM-powered prompts to each file's content, and writing the results to an output directory. This batch-oriented processing is ideal for transformation, summarization or enhanced file processing at scale—with minimal setup and full traceability of inputs and outputs.

For example, running the prompt [What are the colors of the thing](./examples/prompt/05_overfile.yml) over [files](./examples/prompt/doc/) containing "earth", "sun", etc would produce the color pallete about each concept.  

```bash
iq ask -p ./examples/prompt/05_overfile.yml -d ./examples/prompt/doc -o /tmp
```

> [!IMPORTANT]
> Use `--mutable` flag to remove input file right after it is processed.
> This controversial advice allows a queue-like system implementation to
> deal with errors. Use this option with caution — it modifies your input
> data and is best suited for temporary or disposable file queues.


### Basic agent

We assume LLM-powered agent is a system that dynamically direct their own processes and available tool to accomplish task. Unlike a single prompt, a workflow allows the model to reason through multiple steps — such as reading, generating, modifying, and combining files — to accomplish a complex goal. Each step builds on the last, forming a coherent process that mimics how a human might complete a task using a shell or scripting environment.

For example, a workflow might involve generating content into files, transforming or replacing parts of that content, aggregating results, and applying formatting — all before producing a final output. By giving the LLM both a plan and access to command-level operations, workflows unlock a higher level of automation and creativity within your file system.

`iq task` empowers LLMs with **Bash**, **Golang** and **Python** as available tools (must be installed in your environment) to accomplish defined task. [Create an issue](https://github.com/fogfish/iq/issues) if your use-case requires other tools.

For example, [the task about colors](./examples/task/01_basic.yml).


```bash
iq task -p ./examples/task/01_basic.yml
```

### Processing files with agent

`iq run` command treats a mounted directory of files as a processing queue—reading from an input directory, evaluating agent with each file's content, and writing the results to an output directory.

For example, running the prompt [What are the colors of the thing](./examples/task/02_processor.yml) over [files](./examples/prompt/doc/) containing "earth", "sun", etc would produce the color pallete about each concept.  

```bash
iq run -p ./examples/task/02_processor.yml -d ./examples/prompt/doc -o /tmp
```

### Classification

`iq tag` command uses an LLM to classify files based on their content and organize them accordingly. It processes each file, runs a prompt designed to extract metadata or labels (e.g., category, topic, sentiment, priority), and returns a structured response — typically in JSON. This metadata is used to move or copy files into specific directories, effectively sorting your input set into meaningful buckets. `iq tag` is ideal for organizing large, unstructured datasets, triaging documents, or preparing inputs for downstream workflows.

```bash
iq tag -p ./examples/prompt/05_overfile.yml -d ./examples/prompt/doc -o /tmp
```

### Processing large files

`iq` provides smart input splitting to ensure files fit within the context window limits of LLMs. By default, it includes the largest possible portion of the file directly in the prompt. Splitting behavior is configurable — you can divide input by sentences, paragraphs, or fixed-size chunks. 

```bash
iq help

  --splitter string         split input file into sentence, paragraph or chunk (default "none")
  --splitter-chars string   sequence of charates used by splitter as delimiter
  --splitter-chunk int      chunk size for splitter (default 1024)
```

**Sentences**: A sentence is defined as a punctuation mark (., !, or ?) followed by a whitespace character. The default punctuation marks are overwritten with `--splitter-chars` flag.

```bash
iq tell \
  --splitter sentence --splitter-chars . \
  ...
```

**Paragraphs**: A paragraph is defined as a block of text separated by an empty line (essentially using `\n\n` as delimiter). The default delimiter is overwritten with `--splitter-chars` flag.

```bash
iq tell \
  --splitter paragraph --splitter-chars $'\x0a\x0a' \
  ...
```

**Fixed chunks**: A fixed chunk has a defined size limit, which is extended to include the end of the nearest sentence (it prevents loss of context). The chunk size is configured with `--splitter-chunk` flag and `--splitter-chars` are used to define punctuation marks similar to sentence split.

```bash
iq tell \
  --splitter chunk --splitter-chunk 4096 \
  ...
```

### Working with AWS S3

`iq` supports both local file system and AWS S3 buckets. Use `s3://` prefix to direct the utility (e.g. s3://bucket/path).

```bash
echo "What ..." | iq ask -d s3://my/example -o s3://my/result`
```

### STDIN/STDOUT

`id` enables STDIN/STDOUT for shell scripting integration.
* It ONLY reads prompt from STDIN unless `-p` flag is used.
* It output request of processing to STDOUT for `draft`, `tell` and `task` commands.


## How To Contribute

`iq` is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request


## License

[![See LICENSE](https://img.shields.io/github/license/fogfish/iq.svg?style=for-the-badge)](LICENSE)

