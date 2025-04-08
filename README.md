# iq

Intelligent Query `iq` is a fast and lightweight CLI for running LLM-powered agents.  Use it to run prompts and workflows on local files or S3 buckets.

The philosophy behind the tool is to provide two distinct modes of operation: batch processing and individual tasks. Commands like 'ask' and 'run' are designed for processing groups of files in bulk, whether they are stored locally or in S3 buckets. These commands allow you to apply prompts or run workflows across multiple files at once. On the other hand, commands like 'exec' and 'tell' are focused on isolated operations, where you perform a single task or send a one-off prompt to the LLM.

## Quick Start

Install utility from GitHub. (Homebrew is coming)

```bash
go install github.com/fogfish/iq@latest
```

You need to configure iq with access to an LLM provider before using it. It supports multiple backends, including Amazon Bedrock, OpenAI, OpenAI-compatible APIs, and local LM Studio instances. Before doing this you have to create respective accounts, install software (LM Studio) and make sure you have access to the model.

```bash
iq config --bedrock
iq config --lmstudio
iq config --openai <secret-key>
```

> [!TIP]
> My personal recommendation is usage of Amazon Bedrock. 


```bash
echo "What are the colors of rainbow?" | iq tell
```

> [!TIP]
> use -m, --llm flags to override the default model


## User Guide

### Prompt structure

`iq` recommends structured prompting using the [TELeR framework](https://aclanthology.org/2023.findings-emnlp.946.pdf) — a practical taxonomy that breaks prompts into clear components: Task, Environment, Learner, and Response. This approach helps you craft reusable prompts by clearly defining goals, constraints, tone, and expected outputs. Use it to improve prompt quality, automate workflows, and ensure consistent LLM behavior across files and tasks.

Use draft command to create a prompt YAML file:
```bash
iq draft
```

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

In addtion, you can "try" to force [LLM responding in JSON](./examples/prompt/02_jsonify.yml) and reuse the core prompt structure via [include directive](./examples/prompt/04_include.yml).

> [!TIP]
> Meta-prompting is possible as well. The command below uses 
> input text to generate a prompt
> 
> `echo "What are the colors of rainbow?" | iq draft`

### Processing file

`iq` treats files as a processing queue — reading from an input folder, applying LLM-powered prompts or workflows, and writing results to an output folder. The `ask` command runs a prompt against each file's content, while `run` executes a full task workflow defined in a prompt template. This batch-oriented design allows you to automate the transformation, summarization, classification, or enhancement of documents at scale — with minimal setup and full traceability of inputs and outputs.

For example, running the prompt [What are the colors of the thing](./examples/prompt/05_overfile.yml) over [files](./examples/prompt/doc/) containing "earth", "sun", etc would produce the color pallete about it.  

```bash
iq ask -p ./examples/prompt/05_overfile.yml -d ./examples/prompt/doc -o /tmp
```

> [!TIP]
> iq supports both local file system and AWS S3 buckets.
> Use s3:// prefix to direct the utility (e.g. s3://bucket/path).
>
> `echo "What ..." | iq ask -d s3://my/example -o s3://my/result`

> [!IMPORTANT]
> Use `--mutable` flag to remove input file right after it is processed.
> This controversial advice allows a queue-like system implementation to
> deal with errors. Use this option with caution — it modifies your input
> data and is best suited for temporary or disposable file queues.

### Workflow

A workflow in `iq` represents a structured sequence of operations that an LLM can execute autonomously using available commands and tools. Unlike a single prompt, a workflow allows the model to reason through multiple steps — such as reading, generating, modifying, and combining files — to accomplish a complex goal. Each step builds on the last, forming a coherent process that mimics how a human might complete a task using a shell or scripting environment. For example, a workflow might involve generating content into files, transforming or replacing parts of that content, aggregating results, and applying formatting — all before producing a final output. By giving the LLM both a plan and access to command-level operations, workflows unlock a higher level of automation and creativity within your file system. For example, [the workflow about colors](./examples/task/01_bash.yml).

`iq` support shell scripting specificly **Bash**, **Golang** and **Python** as available tools (must be installed in your environment). Create an issue if your use-case requires other tools. 

```bash
iq run -p ./examples/task/01_bash.yml -d ./examples/prompt/doc -o /tmp
```

### Classification

The 'iq tag' command uses an LLM to classify files based on their content and organize them accordingly. It processes each file, runs a prompt designed to extract metadata or labels (e.g., category, topic, sentiment, priority), and returns a structured response — typically in JSON. This metadata is used to move or copy files into specific directories, effectively sorting your input set into meaningful buckets. 'iq tag' is ideal for organizing large, unstructured datasets, triaging documents, or preparing inputs for downstream workflows.

```bash
iq tag -p ./examples/prompt/05_overfile.yml -d ./examples/prompt/doc -o /tmp
```

### On-demand interactions

The `tell` and `exec` commands are twins to `ask` and `run`. They are designed for single, on-demand interactions with an LLM — without involving file batches. Use `tell` to send a standalone prompt and receive an immediate response, ideal for asking questions, drafting text, or running quick ideas past the model. In contrast, `exec` is used to run a full workflow: a prompt that describes a multi-step task and leverages built-in tools to read, write, and manipulate files as needed.


## How To Contribute

`iq` is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request


## License

[![See LICENSE](https://img.shields.io/github/license/fogfish/iq.svg?style=for-the-badge)](LICENSE)

