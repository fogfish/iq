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
	"strings"

	"github.com/fogfish/iq/internal/prompt"
	"github.com/fogfish/iq/internal/service"
	"github.com/fogfish/opts"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/command"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute(vsn string) {
	rootCmd.Version = vsn

	if err := rootCmd.Execute(); err != nil {
		e := err.Error()
		fmt.Fprintf(os.Stderr, "\n ❌ Something went wrong. Check the error below for details.\n   Run `iq help` for guidance.\n\n   %s\n\n", strings.ToUpper(e[:1])+e[1:])
		os.Exit(1)
	}
}

var (
	rootProfile  string
	rootLLM      string
	rootPrompt   string
	rootMaxEpoch int
	rootDebug    bool
	rootSilent   bool
	gLLM         chatter.Chatter
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootProfile, "config", "c", "iq", "config profile at ~/.netrc about LLM provider")
	rootCmd.PersistentFlags().StringVarP(&rootLLM, "llm", "m", "", "overrides LLM model defined at ~/.netrc")
	rootCmd.PersistentFlags().StringVarP(&rootPrompt, "prompt", "p", "", "path to prompt yaml file")
	rootCmd.PersistentFlags().IntVarP(&rootMaxEpoch, "epoch", "e", 5, "max number of attempts (epoch) to refine the task before give up")
	rootCmd.PersistentFlags().BoolVar(&rootDebug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVarP(&rootSilent, "silent", "s", false, "not output")
}

var rootCmd = &cobra.Command{
	Use:   "iq",
	Short: "a fast and lightweight CLI tool for running LLM-powered agents",
	Long: `
 _
(_)__ _      
| / _' |    a fast and lightweight CLI for running LLM-powered agents.
|_\__, |    Use it to run prompts and workflows on local files or S3 buckets.
     |_|

The philosophy behind the tool is to provide two distinct modes of operation:
batch processing and individual tasks. Commands like 'ask' and 'run' are designed
for processing groups of files in bulk, whether they are stored locally or in
S3 buckets. These commands allow you to apply prompts or run workflows across
multiple files at once. On the other hand, commands like 'exec' and 'tell' are
focused on isolated operations, where you perform a single task or send a
one-off prompt to the LLM.

For example, below it process files in current directory returning the color
palletes for things depicted by file.

echo "What are colors of the thing in the attached document?" | iq ask -o /tmp

Run 'iq help' for guidance.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	## Send a prompt to LLM
	iq tell -p myprompt.yml
	iq tell -p myprompt.yml FILE1 ...

	## Execute a workflow using LLM instructions
	iq exec -p mytask.yml
	iq exec -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq exec --python

	## Process files with LLM
	iq ask -p prompt.yml -o ./output
	echo "What are colors of the thing in the attached document?" | iq ask -o ./output
	`,
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) { cmd.Help() },
}

//------------------------------------------------------------------------------

func configLLM(llmid string) (chatter.Chatter, error) {
	llm, err := createLLM(llmid)
	if err != nil {
		return nil, err
	}

	if !rootDebug {
		return llm, nil
	}

	return aio.NewLogger(os.Stderr, llm), nil
}

func createLLM(llmid string) (chatter.Chatter, error) {
	if len(llmid) != 0 {
		return autoconfig.New(rootProfile, llmid)
	}

	if len(rootLLM) != 0 {
		if rootLLM == "mock" {
			return llmock(0), nil
		}
		return autoconfig.New(rootProfile, rootLLM)
	}

	return autoconfig.New(rootProfile)
}

type llmock int

func (llmock) UsedInputTokens() int { return 0 }
func (llmock) UsedReplyTokens() int { return 0 }

func (llmock) Prompt(ctx context.Context, prompt []fmt.Stringer, opt ...chatter.Opt) (chatter.Reply, error) {
	seq := make([]string, len(prompt))
	for i, s := range prompt {
		seq[i] = s.String()
	}
	reply := strings.Join(seq, " ")

	return chatter.Reply{Text: reply, UsedInputTokens: len(reply), UsedReplyTokens: len(reply)}, nil
}

//------------------------------------------------------------------------------

func parsePrompt() (*viper.Viper, error) {
	if len(rootPrompt) != 0 {
		return prompt.ParseFile(rootPrompt)
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("no prompt data in stdin")
	}

	return prompt.Parse(os.Stdin)
}

func agentForPrompts() (*service.Prompter, *viper.Viper, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	gLLM, err = configLLM(in.GetString("llm"))
	if err != nil {
		return nil, nil, err
	}

	agt, err := service.NewPrompter(gLLM, in, rootMaxEpoch)
	if err != nil {
		return nil, nil, err
	}

	return agt, in, nil
}

func agentForTasks(workdir string) (*service.Worker, *viper.Viper, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	gLLM, err = configLLM(in.GetString("llm"))
	if err != nil {
		return nil, nil, err
	}

	if execWithBash || execWithGolang || execWithPython {
		registry := []string{}
		if execWithBash {
			registry = append(registry, command.BASH)
		}
		if execWithGolang {
			registry = append(registry, command.GOLANG)
		}
		if execWithPython {
			registry = append(registry, command.PYTHON)
		}

		in.Set(prompt.YAML_REGISTRY, registry)
	}

	agt, err := service.NewWorker(gLLM, in, workdir, rootMaxEpoch)
	if err != nil {
		return nil, nil, err
	}

	return agt, in, nil
}

//------------------------------------------------------------------------------

func createSpool(pathDir, pathOut string, mutable, strict bool) (*spool.Spool, error) {
	dir, err := mount(pathDir)
	if err != nil {
		return nil, fmt.Errorf("unable to mount input dir (--dir, -d): %w", err)
	}

	out, err := mount(pathOut)
	if err != nil {
		return nil, fmt.Errorf("unable to mount output dir (--out, -o): %w", err)
	}

	opt := []opts.Option[spool.Spool]{}
	if mutable {
		opt = append(opt, spool.IsMutable)
	} else {
		opt = append(opt, spool.IsImmutable)
	}

	if strict {
		opt = append(opt, spool.WithStrict)
	} else {
		opt = append(opt, spool.WithSkipError)
	}

	q := spool.New(dir, out, opt...)

	return q, nil
}

func mount(path string) (spool.FileSystem, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("undefined mount point")
	}

	const s3pfx = "s3://"
	if strings.HasPrefix(path, s3pfx) {
		return stream.NewFS(path[len(s3pfx):])
	}

	return lfs.New(path)
}

//------------------------------------------------------------------------------

type spinner interface {
	Describe(string)
	Finish() error
	Reset()
}

type nospinner int

func (nospinner) Describe(string) {}
func (nospinner) Finish() error   { return nil }
func (nospinner) Reset()          {}

func createSpinner() spinner {
	if rootSilent {
		return nospinner(0)
	}

	return progressbar.NewOptions(-1,
		progressbar.OptionShowBytes(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSpinnerType(11),
	)
}

func ellipses(txt string) string {
	if len(txt) > 80 {
		return txt[:80] + "..."
	}
	return txt
}

func respinner(s spinner) {
	if rootSilent {
		return
	}

	os.Stderr.WriteString("\n")
	s.Reset()
}

//------------------------------------------------------------------------------

func withUsage(f func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := f(cmd, args)
		if err != nil {
			return err
		}

		if rootSilent {
			return nil
		}

		it := gLLM.UsedInputTokens()
		rt := gLLM.UsedReplyTokens()
		fmt.Fprintf(os.Stderr, "\n\n 💡 Tokens used: %d (input: %d, reply: %d)\n", it+rt, it, rt)

		return nil
	}
}

func fPrintf(w io.Writer, format string, a ...any) (n int, err error) {
	if rootSilent {
		return 0, nil
	}
	return fmt.Fprintf(w, format, a...)
}
