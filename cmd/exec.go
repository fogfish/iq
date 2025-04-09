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

var (
	execWithBash   bool
	execWithGolang bool
	execWithPython bool
	execWorkDir    string
)

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().BoolVar(&execWithBash, "bash", false, "enable bash in the command registry")
	execCmd.Flags().BoolVar(&execWithGolang, "golang", false, "enable golang in the command registry")
	execCmd.Flags().BoolVar(&execWithPython, "python", false, "enable python in the command registry")
	execCmd.Flags().StringVar(&execWorkDir, "workdir", "", "work directory for tools and commands")
}

var execCmd = &cobra.Command{
	Use:   "task",
	Short: "execute a workflow using LLM instructions",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq exec -p mytask.yml
	iq exec -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq exec --python
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(exec),
}

func exec(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	agt, req, err := agentForTasks(execWorkDir)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		spinner.Describe("processing ...")
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
		spinner.Describe(fmt.Sprintf("processing with %s ...", file))
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
