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
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func Execute(vsn string) {
	rootCmd.Version = vsn

	if err := rootCmd.Execute(); err != nil {
		e := err.Error()
		fmt.Fprintf(os.Stderr, "\n ❌ Something went wrong. Check the error below for details.\n   Run `iq help` for guidance.\n\n   %s\n\n", strings.ToUpper(e[:1])+e[1:])
		os.Exit(1)
	}
}

func init() {
	fmodel.apply(rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   "iq",
	Short: "AI workflows for the shell - no code required.",
	Long: `
 
(_)__ _      
| / _' |   AI workflows for the shell - no code required.
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

// func createSpool(pathDir, pathOut string, mutable, strict bool) (*spool.Spool, error) {
// 	dir, err := mount(pathDir)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to mount input dir (--dir, -d): %w", err)
// 	}

// 	out, err := mount(pathOut)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to mount output dir (--out, -o): %w", err)
// 	}

// 	opt := []opts.Option[spool.Spool]{}
// 	if mutable {
// 		opt = append(opt, spool.IsMutable)
// 	} else {
// 		opt = append(opt, spool.IsImmutable)
// 	}

// 	if strict {
// 		opt = append(opt, spool.WithStrict)
// 	} else {
// 		opt = append(opt, spool.WithSkipError)
// 	}

// 	q := spool.New(dir, out, opt...)

// 	return q, nil
// }

// func mount(path string) (spool.FileSystem, error) {
// 	if len(path) == 0 {
// 		return nil, fmt.Errorf("undefined mount point")
// 	}

// 	const s3pfx = "s3://"
// 	if strings.HasPrefix(path, s3pfx) {
// 		return stream.NewFS(path[len(s3pfx):])
// 	}

// 	const std = "stdout"
// 	if path == std {
// 		return stdout{}, nil
// 	}

// 	return lfs.New(path)
// }

// type fout struct {
// 	io.Writer
// }

// func (fout) Stat() (fs.FileInfo, error) { return nil, nil }
// func (fout) Close() error               { return nil }
// func (fout) Cancel() error              { return nil }

// type stdout struct{}

// func (stdout) Open(name string) (fs.File, error) { return nil, nil }
// func (stdout) Remove(path string) error          { return nil }
// func (stdout) Create(path string, attr *struct{}) (stream.File, error) {
// 	return fout{Writer: os.Stdout}, nil
// }

// //------------------------------------------------------------------------------

// type spinner interface {
// 	Describe(string)
// 	Finish() error
// 	Reset()
// }

// type nospinner int

// func (nospinner) Describe(string) {}
// func (nospinner) Finish() error   { return nil }
// func (nospinner) Reset()          {}

// func createSpinner() spinner {
// 	if rootSilent {
// 		return nospinner(0)
// 	}

// 	return progressbar.NewOptions(-1,
// 		progressbar.OptionShowBytes(false),
// 		progressbar.OptionClearOnFinish(),
// 		progressbar.OptionSetWriter(os.Stderr),
// 		progressbar.OptionShowDescriptionAtLineEnd(),
// 		progressbar.OptionSpinnerType(11),
// 	)
// }

// func ellipses(txt string) string {
// 	if len(txt) > 80 {
// 		return txt[:80] + "..."
// 	}
// 	return txt
// }

// func respinner(s spinner) {
// 	if rootSilent {
// 		return
// 	}

// 	os.Stderr.WriteString("\n")
// 	s.Reset()
// }

// //------------------------------------------------------------------------------

// func withUsage(f func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
// 	return func(cmd *cobra.Command, args []string) error {
// 		err := f(cmd, args)
// 		if err != nil {
// 			return err
// 		}

// 		if rootSilent {
// 			return nil
// 		}

// 		// usage := gLLM.Usage()
// 		// fmt.Fprintf(os.Stderr, "\n\n 💡 Tokens used: %d (input: %d, reply: %d)\n", usage.InputTokens+usage.ReplyTokens, usage.InputTokens, usage.ReplyTokens)

// 		return nil
// 	}
// }

// func fPrintf(w io.Writer, format string, a ...any) (n int, err error) {
// 	if rootSilent {
// 		return 0, nil
// 	}
// 	return fmt.Fprintf(w, format, a...)
// }
