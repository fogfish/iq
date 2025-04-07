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
	"os"
	"strings"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/fogfish/opts"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
	"github.com/spf13/cobra"
)

var (
	spoolDir     string
	spoolOut     string
	spoolMutable bool
	spoolStrict  bool
)

func init() {
	rootCmd.AddCommand(spoolCmd)
	spoolCmd.PersistentFlags().StringVarP(&spoolDir, "dir", "d", "", "input directory or s3 path")
	spoolCmd.PersistentFlags().StringVarP(&spoolOut, "out", "o", "", "output directory or s3 path")
	spoolCmd.PersistentFlags().BoolVar(&spoolMutable, "mutable", false, "mutable spool")
	spoolCmd.PersistentFlags().BoolVar(&spoolStrict, "strict", false, "strict spool")

	spoolCmd.AddCommand(spoolPromptCmd)
	spoolCmd.AddCommand(spoolTaskCmd)
	spoolCmd.AddCommand(spoolClassifyCmd)
}

var spoolCmd = &cobra.Command{
	Use:   "spool",
	Short: "process files with LLM",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq
	`,
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) { cmd.Help() },
}

//------------------------------------------------------------------------------

var spoolClassifyCmd = &cobra.Command{
	Use:   "classify",
	Short: "classify each file",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq
	`,
	SilenceUsage: true,
	RunE:         spoolClassify,
}

func spoolClassify(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool()
	if err != nil {
		return err
	}

	agt, req, err := agentForPrompts()
	if err != nil {
		return err
	}
	req.Set(spec.YAML_FORMAT, "json")

	q.PartitionFile(context.Background(), "/",
		func(ctx context.Context, path string, b []byte) (string, error) {
			if len(path) > 80 {
				spinner.Describe(path[:80])
			} else {
				spinner.Describe(path)
			}
			defer func() {
				os.Stderr.WriteString("\n")
				spinner.Reset()
			}()

			req.Set(spec.YAML_BLOB, string(b))
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return "", err
			}

			var seq []string
			if err := json.Unmarshal(reply, &seq); err != nil {
				return "", err
			}

			shard := strings.ReplaceAll(strings.ToLower(seq[0]), " ", "_")
			return shard, nil
		})

	return nil
}

//------------------------------------------------------------------------------

var spoolPromptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "apply prompt for each file",
	Long: `
xxx

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq
	`,
	SilenceUsage: true,
	RunE:         spoolPrompt,
}

func spoolPrompt(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool()
	if err != nil {
		return err
	}

	agt, req, err := agentForPrompts()
	if err != nil {
		return err
	}

	q.ForEachFile(context.Background(), "/",
		func(ctx context.Context, path string, b []byte) ([]byte, error) {
			if len(path) > 80 {
				spinner.Describe(path[:80])
			} else {
				spinner.Describe(path)
			}
			defer func() {
				os.Stderr.WriteString("\n")
				spinner.Reset()
			}()

			req.Set(spec.YAML_BLOB, string(b))
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return nil, err
			}

			return reply, nil
		})

	return nil
}

var spoolTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "evaluate task for each file",
	Long: `
TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq
	`,
	SilenceUsage: true,
	RunE:         spoolTask,
}

func spoolTask(cmd *cobra.Command, args []string) error {
	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool()
	if err != nil {
		return err
	}

	agt, req, err := agentForTasks()
	if err != nil {
		return err
	}

	q.ForEachFile(context.Background(), "/",
		func(ctx context.Context, path string, b []byte) ([]byte, error) {
			if len(path) > 80 {
				spinner.Describe(path[:80])
			} else {
				spinner.Describe(path)
			}
			defer func() {
				os.Stderr.WriteString("\n")
				spinner.Reset()
			}()

			req.Set(spec.YAML_BLOB, string(b))
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return nil, err
			}

			return reply, nil
		})

	return nil
}

func createSpool() (*spool.Spool, error) {
	dir, err := mount(spoolDir)
	if err != nil {
		return nil, err
	}

	out, err := mount(spoolOut)
	if err != nil {
		return nil, err
	}

	opt := []opts.Option[spool.Spool]{}
	if spoolMutable {
		opt = append(opt, spool.IsMutable)
	} else {
		opt = append(opt, spool.IsImmutable)
	}

	if spoolStrict {
		opt = append(opt, spool.WithStrict)
	} else {
		opt = append(opt, spool.WithSkipError)
	}

	q := spool.New(dir, out, opt...)

	return q, nil
}

func mount(path string) (spool.FileSystem, error) {
	const s3pfx = "s3://"
	if strings.HasPrefix(path, s3pfx) {
		return stream.NewFS(path[len(s3pfx):])
	}

	return lfs.New(path)
}
