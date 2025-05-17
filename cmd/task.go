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
	"io"
	"os"
	"path/filepath"

	"github.com/fogfish/iq/internal/reader"
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
	Short: "execute LLM-agent with prompt instructions",
	Long: `
We assume LLM-powered agent is a system that dynamically direct their own
processes and available tool to accomplish task. Unlike a single prompt,
a workflow allows the model to reason through multiple steps — such as reading,
generating, modifying, and combining files — to accomplish a complex goal.
Each step builds on the last, forming a coherent process that mimics how a human
might complete a task using a shell or scripting environment.

It empowers LLMs with Bash, Golang and Python as available tools to accomplish
defined task.


See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq task -p mytask.yml
	iq task -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq task --python
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(exec),
}

func exec(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	if rootThink {
		spinner.Finish()
	}

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

	q, err := createSpool("/", "stdout", false, true)
	if err != nil {
		return err
	}

	for i, arg := range args {
		path, err := filepath.Abs(arg)
		if err != nil {
			return err
		}
		args[i] = filepath.Clean(path)
	}

	err = q.ForEachPath(context.Background(), args,
		func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
			spinner.Describe(fmt.Sprintf("processing with %s", ellipses(filepath.Base(path))))
			defer spinner.Reset()

			fd := reader.New(rootScanner, rootScannerChars, rootScannerChunk, r)
			return reader.Process(ctx, agt, req, fd, w)
		},
	)
	if err != nil {
		return err
	}

	return nil
}
