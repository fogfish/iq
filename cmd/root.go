//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/fogfish/iq/internal/adapter"
	"github.com/fogfish/iq/internal/core"
	"github.com/fogfish/iq/internal/prompt"
	"github.com/fogfish/iq/internal/service"
	"github.com/fogfish/opts"
	"github.com/fogfish/stream"
	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
	"github.com/kshard/chatter"
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
	rootLLM          adapter.LLM
	gLLM             chatter.Chatter
	rootPrompt       string
	rootInput        string
	rootDebug        bool
	rootThink        bool
	rootSilent       bool
	rootScanner      string
	rootScannerChunk int
	rootScannerChars string
)

func init() {
	fmodel.apply(rootCmd)
	fagent.apply(rootCmd)
	finput.apply(rootCmd)
	freply.apply(rootCmd)

	// rootCmd.PersistentFlags().StringVarP(&rootLLM.Profile, "config", "c", "iq", "config profile at ~/.netrc about LLM provider")
	// rootCmd.PersistentFlags().StringVarP(&rootLLM.Model, "llm", "m", "", "overrides LLM model defined at ~/.netrc")
	// rootCmd.PersistentFlags().IntVar(&rootLLM.MaxEpoch, "max-epoch", 0, "max number of attempts (epoch) to refine the task before give up")
	// rootCmd.PersistentFlags().IntVar(&rootLLM.MaxUsage.InputTokens, "max-input-tokens", 0, "max number of input tokens to consume before give up")
	// rootCmd.PersistentFlags().IntVar(&rootLLM.MaxUsage.ReplyTokens, "max-reply-tokens", 0, "max number of reply tokens to consume before give up")

	// TODO: agent -a
	//rootCmd.PersistentFlags().StringVar(&rootPrompt, "prompt", "", "path to prompt yaml file")
	rootCmd.PersistentFlags().StringVar(&rootInput, "input", "", "override prompt input")

	// rootCmd.PersistentFlags().BoolVar(&rootDebug, "debug", false, "enable debug output")
	// rootCmd.PersistentFlags().BoolVar(&rootThink, "think", false, "enable thinking output")
	// rootCmd.PersistentFlags().BoolVarP(&rootSilent, "silent", "s", false, "enable silent behaviour")

	// rootCmd.PersistentFlags().StringVar(&rootScanner, "splitter", "none", "split input file into sentence, paragraph or chunk")
	// rootCmd.PersistentFlags().IntVar(&rootScannerChunk, "splitter-chunk", 1024, "chunk size for splitter")
	// rootCmd.PersistentFlags().StringVar(&rootScannerChars, "splitter-chars", "", "sequence of charates used by splitter as delimiter")
}

var rootCmd = &cobra.Command{
	Use:   "iq",
	Short: "a fast and lightweight CLI tool for running LLM-powered agents",
	Long: `
 
(_)__ _      
| / _' |   A lightweight command-line LLM-powered file processor.
|_\__, |    
     |_|

The tool is designed to simplify running LLM-powered agents in the batch against
mounted file systems, whether they are simple prompting agents or more complex
system that dynamically direct their own processes and tool to accomplish task.

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
	iq task -p mytask.yml
	iq task -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq task --python

	## Process files with LLM
	iq ask -p prompt.yml -o ./output
	echo "What are colors of the thing in the attached document?" | iq ask -o ./output
	`,
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) { cmd.Help() },
}

//------------------------------------------------------------------------------

func parsePrompt() (in *viper.Viper, err error) {
	if len(rootPrompt) == 0 {
		in, err = parsePromptStdin()
	} else {
		in, err = prompt.ParseFile(rootPrompt)
	}

	if err != nil {
		return
	}

	if len(rootInput) == 0 {
		return
	}

	args := in.GetStringMap(prompt.YAML_INPUT)
	if args == nil {
		args = map[string]any{}
	}

	for _, input := range strings.Split(rootInput, ",") {
		kv := strings.Split(input, "=")
		if len(kv) != 2 {
			continue
		}
		args[kv[0]] = kv[1]
	}
	in.Set(prompt.YAML_INPUT, args)

	return in, nil
}

func parsePromptStdin() (*viper.Viper, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("no prompt data in stdin")
	}

	in, err := prompt.Parse(os.Stdin)
	if err != nil {
		return nil, err
	}

	return in, nil
}

func parseInputStdin() ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func agentForPrompts() (*service.Prompter, *core.Prompt, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	req, err := adapter.DecodeViperToPrompt(in)
	if err != nil {
		return nil, nil, err
	}

	if len(rootPrompt) != 0 {
		doc, err := parseInputStdin()
		if err != nil {
			return nil, nil, err
		}
		req.Blob = string(doc)
	}

	gLLM, err = rootLLM.Create(in.GetString("llm"), rootDebug, rootThink)
	if err != nil {
		return nil, nil, err
	}

	agt, err := service.NewPrompter(gLLM)
	if err != nil {
		return nil, nil, err
	}

	return agt, req, nil
}

func agentForTasks(workdir string) (*service.Worker, *core.Prompt, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	req, err := adapter.DecodeViperToPrompt(in)
	if err != nil {
		return nil, nil, err
	}

	if len(rootPrompt) != 0 {
		doc, err := parseInputStdin()
		if err != nil {
			return nil, nil, err
		}
		req.Blob = string(doc)
	}

	gLLM, err = rootLLM.Create(in.GetString("llm"), rootDebug, rootThink)
	if err != nil {
		return nil, nil, err
	}

	if len(workdir) == 0 {
		workdir, err = os.MkdirTemp(os.TempDir(), "iq-")
		if err != nil {
			return nil, nil, err
		}
	}

	registry, err := adapter.DecodeViperToRegistry(in, workdir)
	if err != nil {
		return nil, nil, err
	}

	// if execWithBash || execWithGolang || execWithPython {
	// 	if execWithBash {
	// 		registry.Register(command.Bash("", workdir))
	// 	}
	// 	if execWithGolang {
	// 		registry.Register(command.Golang(workdir))
	// 	}
	// 	if execWithPython {
	// 		registry.Register(command.Python(workdir))
	// 	}
	// }

	agt, err := service.NewWorker(gLLM, registry)
	if err != nil {
		return nil, nil, err
	}

	return agt, req, nil
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

	const std = "stdout"
	if path == std {
		return stdout{}, nil
	}

	return lfs.New(path)
}

type fout struct {
	io.Writer
}

func (fout) Stat() (fs.FileInfo, error) { return nil, nil }
func (fout) Close() error               { return nil }
func (fout) Cancel() error              { return nil }

type stdout struct{}

func (stdout) Open(name string) (fs.File, error) { return nil, nil }
func (stdout) Remove(path string) error          { return nil }
func (stdout) Create(path string, attr *struct{}) (stream.File, error) {
	return fout{Writer: os.Stdout}, nil
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

		// usage := gLLM.Usage()
		// fmt.Fprintf(os.Stderr, "\n\n 💡 Tokens used: %d (input: %d, reply: %d)\n", usage.InputTokens+usage.ReplyTokens, usage.InputTokens, usage.ReplyTokens)

		return nil
	}
}

func fPrintf(w io.Writer, format string, a ...any) (n int, err error) {
	if rootSilent {
		return 0, nil
	}
	return fmt.Fprintf(w, format, a...)
}
