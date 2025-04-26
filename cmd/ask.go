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
	"os"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	askDir     string
	askOut     string
	askMutable bool
	askStrict  bool
)

func init() {
	rootCmd.AddCommand(askCmd)

	askCmd.Flags().StringVarP(&askDir, "dir", "d", ".", "input directory or S3 path to read files from")
	askCmd.Flags().StringVarP(&askOut, "out", "o", "", "output directory or S3 path to write results")
	askCmd.Flags().BoolVar(&askMutable, "mutable", false, "enable mutable processing, input file is removed\nafter results are saved (allows implementation of fail safe pipelines)")
	askCmd.Flags().BoolVar(&askStrict, "strict", false, "enforce strict processing, failing on the first error")
}

var askCmd = &cobra.Command{
	Use:   "ask",
	Short: "process files in mounted dir with LLM",
	Long: `

ask command treats a mounted directory of files as a processing queue—reading
from an input directory, applying LLM-powered prompts to each file's content,
and writing the results to an output directory. This batch-oriented processing
is ideal for transformation, summarization or enhanced file processing at
scale—with minimal setup and full traceability of inputs and outputs.

The command support mounting of AWS S3 bucket. Use s3:// prefix
prefix to direct the utility (e.g. s3://bucket/path).

  echo "What ..." | iq ask -d s3://my/example -o s3://my/result

Processing a large number of files may require the ability to start, stop, and
resume the utility reliably. To support this, you can use the --mutable flag,
which removes each input file immediately after it has been successfully processed.
This enables fault-tolerant, resumable execution by ensuring already-processed
files are skipped on subsequent runs.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq ask -p prompt.yml -o ./output
	
	echo "What are colors of the thing in the attached document?" | iq ask -o ./output
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(ask),
}

func ask(cmd *cobra.Command, args []string) error {
	fPrintf(os.Stderr, " 📂 asking about files in %s\n", askDir)
	if askMutable {
		fPrintf(os.Stderr, " ‼️ removes each input file immediately after it has been processed.\n")
	}

	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool(askDir, askOut, askMutable, askStrict)
	if err != nil {
		return err
	}

	agt, req, err := agentForPrompts()
	if err != nil {
		return err
	}

	q.ForEachFile(context.Background(), "/",
		func(ctx context.Context, path string, b []byte) ([]byte, error) {
			spinner.Describe(ellipses(path))
			defer respinner(spinner)

			req.Set(spec.YAML_BLOB, string(b))
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return nil, err
			}

			return reply, nil
		})

	return nil
}
