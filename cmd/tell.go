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

func init() {
	rootCmd.AddCommand(tellCmd)
}

var tellCmd = &cobra.Command{
	Use:   "tell",
	Short: "prompts LLM",
	Long: `
Send a standalone prompt and receive an immediate response from LLM. It is ideal
for asking questions, drafting text, or running quick ideas before building the
workflow pipelines.

The command takes prompt either from file -p, --prompt flag or STDIN.

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
			spinner.Describe(fmt.Sprintf("prompting with %s", ellipses(filepath.Base(path))))
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
