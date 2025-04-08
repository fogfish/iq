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
	taskWithBash   bool
	taskWithGolang bool
	taskWithPython bool
	taskWorkDir    string
)

func init() {
	rootCmd.AddCommand(taskCmd)

	taskCmd.Flags().BoolVar(&taskWithBash, "bash", false, "enable bash in the command registry")
	taskCmd.Flags().BoolVar(&taskWithGolang, "golang", false, "enable golang in the command registry")
	taskCmd.Flags().BoolVar(&taskWithPython, "python", false, "enable python in the command registry")
	taskCmd.Flags().StringVar(&taskWorkDir, "workdir", "", "enable custom workdir")
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "execute task (workflow) with external commands",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq task -p mytask.yml
	iq task -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq task --python
	`,
	SilenceUsage: true,
	RunE:         task,
}

func task(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	agt, req, err := agentForTasks(taskWorkDir)
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
		os.Stderr.WriteString("\n")
		spinner.Reset()

		os.Stdout.Write(reply)
		os.Stdout.WriteString("\n")
	}

	return nil
}
