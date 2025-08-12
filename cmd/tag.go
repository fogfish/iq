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
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/fogfish/iq/internal/core"
	"github.com/spf13/cobra"
)

var (
	tagDir     string
	tagOut     string
	tagMutable bool
	tagStrict  bool
)

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.Flags().StringVarP(&tagDir, "dir", "d", ".", "input directory or S3 path to read files from")
	tagCmd.Flags().StringVarP(&tagOut, "out", "o", "", "output directory or S3 path to write results")
	tagCmd.PersistentFlags().BoolVar(&tagMutable, "mutable", false, "enable mutable processing, input file is removed\nafter results are saved (allows implementation of fail safe pipelines)")
	tagCmd.PersistentFlags().BoolVar(&tagStrict, "strict", false, "enforce strict processing, failing on the first error")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "classify files in the current directory using LLM",
	Long: `
The 'iq tag' command uses an LLM to classify files based on their content and
organize them accordingly. It processes each file, runs a prompt designed to
extract metadata or labels (e.g., category, topic, sentiment, priority), and
returns a structured response — typically in JSON. This metadata is used to
move or copy files into specific directories, effectively sorting your input
set into meaningful buckets. 'iq tag' is ideal for organizing large,
unstructured datasets, triaging documents, or preparing inputs for downstream
workflows.


See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq tag -p classify.yml -o ./output
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(tag),
}

func tag(cmd *cobra.Command, args []string) error {
	fPrintf(os.Stderr, " 📂 taging files in %s\n", tagDir)
	if tagMutable {
		fPrintf(os.Stderr, " ‼️ removes each input file immediately after it has been processed.\n")
	}

	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool(tagDir, tagOut, tagMutable, tagStrict)
	if err != nil {
		return err
	}

	agt, req, err := agentForPrompts()
	if err != nil {
		return err
	}
	req.Format = core.FORMAT_JSON

	err = q.Partition(context.Background(), "/",
		func(ctx context.Context, path string, r io.Reader) (string, error) {
			spinner.Describe(ellipses(path))
			defer respinner(spinner)

			b, err := io.ReadAll(r)
			if err != nil {
				return "", err
			}

			req.Blob = string(b)
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return "", err
			}

			var seq []string
			if err := json.Unmarshal(reply, &seq); err != nil {
				return "", err
			}

			var class = "none"
			if len(seq) > 0 {
				class = seq[0]
			}

			shard := strings.ReplaceAll(strings.ToLower(class), " ", "_")
			return shard, nil
		})
	if err != nil {
		return err
	}

	return nil
}
