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
	"fmt"
	"os"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(tellCmd)
}

var tellCmd = &cobra.Command{
	Use:   "tell",
	Short: "send a prompt to LLM",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq tell -p myprompt.yml
	iq tell -p myprompt.yml FILE1 FILE2 ...
	echo "What are the colors of rainbow?" | iq tell
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(tell),
}

func tell(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	agt, req, err := agentForPrompts()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		spinner.Describe("prompting ...")
		reply, err := agt.PromptOnce(context.Background(), req)
		if err != nil {
			return err
		}
		spinner.Finish()

		os.Stdout.Write(reply)
		os.Stdout.WriteString("\n")
		return nil
	}

	for _, file := range args {
		spinner.Describe(fmt.Sprintf("prompting with %s ...", file))
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		req.Set(spec.YAML_BLOB, string(b))
		reply, err := agt.PromptOnce(context.Background(), req)
		if err != nil {
			return err
		}
		spinner.Reset()

		os.Stdout.Write(reply)
		os.Stdout.WriteString("\n")
	}

	return nil
}
