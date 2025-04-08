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
	iq draft
	echo "What are the colors of rainbow?" | iq draft
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

	llm, err := createLLM("")
	if err != nil {
		return err
	}

	var prompt chatter.Prompt
	prompt.WithTask(`
	  You are a prompt engineer tasked with creating effective instructions
		for language models. Your task is to write a prompt that clearly defines
		a specific task for another LLM to perform: %s.
	`, subj)

	prompt.With(
		chatter.Rules(
			`The prompt you write should be:`,

			`Define task, guidelines and requirments`,
			`Clear and unambiguous`,
			`Purpose-driven (have a clear goal)`,
			`Engaging and motivating`,
			`Scalable for variations of the task`,
			`The output should be YAML, text under prompt key`,
			`Please include the prompt only, without any explanation or commentary.
			The task should be moderately challenging and involve reasoning,
			creativity, or structured output.`,
		),
	)

	prompt.With(
		chatter.Example{
			Input: "What are the colors of rainbow?",
			Reply: "prompt: |\nDefine the sequence of colors in a rainbow...",
		},
	)

	reply, err := llm.Prompt(context.Background(), prompt.ToSeq())
	if err != nil {
		return err
	}

	os.Stdout.Write([]byte(reply.Text))
	return nil
}

func teler() string {
	return `
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

`
}
