//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package cmd

import (
	"context"
	"io"
	"os"

	"github.com/kshard/chatter"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(draftCmd)
	draftCmd.AddCommand(draftYamlCmd)
}

var draftCmd = &cobra.Command{
	Use:   "draft",
	Short: "generate prompt template",
	Long: `
iq recommends structured prompting using the TELeR framework — a practical
taxonomy that breaks prompts into clear components: Task, Environment, Learner,
and Response. This approach helps you craft reusable prompts by clearly defining
goals, constraints, tone, and expected outputs. Use it to improve prompt quality,
automate workflows, and ensure consistent LLM behavior across files and tasks.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	# Generate prompt template
	iq draft

	# Generate prompt with LLM, using stdin as input for sketching the task
	echo "What are the colors of rainbow?" | iq draft

	# Generate workflow for agent execution
	iq draft agent
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          draft,
}

func draft(cmd *cobra.Command, args []string) error {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		os.Stdout.Write([]byte(teler()))
		return nil
	}

	subj, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	llm, err := fmodel.build()
	if err != nil {
		return err
	}

	var prompt chatter.Prompt
	prompt.WithTask(`
	  You are a prompt engineer tasked with creating effective instructions
		for language models. Your task is to write a prompt that clearly defines
		a specific task for another LLM to perform: %s.
	`, subj)

	prompt.WithRules(
		`The prompt you write should be:`,

		`Define task, guidelines and requirments`,
		`Clear and unambiguous`,
		`Purpose-driven (have a clear goal)`,
		`Engaging and motivating`,
		`Scalable for variations of the task`,
		`The output should be Markdown text with YAML front matter as defined in example below.`,
		`Include the prompt only, without any explanation or commentary.
			The task should be moderately challenging and involve reasoning,
			creativity, or structured output.`,
	)

	prompt.With(
		chatter.Example{
			Input: "What are the colors of rainbow?",
			Reply: "---\nformat: text\n---\nDefine the sequence of colors in a rainbow...",
		},
	)

	reply, err := llm.Prompt(context.Background(), prompt.ToSeq())
	if err != nil {
		return err
	}

	os.Stdout.Write([]byte(reply.String()))
	return nil
}

func teler() string {
	return `---
format: text
schema:
  input:
    type: object
    required: [attr]
    properties:
      attr: {type: string}
  reply:
    type: object
    required: [attr]
    properties:
      attr: {type: string}
---
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

`
}

var draftYamlCmd = &cobra.Command{
	Use:   "agent",
	Short: "generate agent template",
	Long: `
xxx

See more info https://github.com/fogfish/iq
	`,
	Example: `
	# Generate workflow for agent execution
	iq draft agent
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          draftYaml,
}

func draftYaml(cmd *cobra.Command, args []string) error {
	os.Stdout.Write([]byte(yaml()))
	return nil
}

func yaml() string {
	return `name: draft
jobs:
  main:
    steps:
      - uses: prompts/prompt.md

      - name: foobar
        uses: prompts/prompt.md
        output: foobar

      - name: foobar
        uses: prompts/prompt.md
        retry:
          attempts: 3
          delay: 2
          yield: prompts/fallback.md
        output: foobar

      - uses: prompts/prompt.md
        switch:
          - when: choice == "choice" && steps.foobar.category == "foo"
            route: func
        default: unknown

      - uses: prompts/prompt.md
        foreach:
          job: func


  func:
    steps:
      - uses: prompts/prompt.md

  unknown:
    steps:
      - uses: prompts/prompt.md

`
}
