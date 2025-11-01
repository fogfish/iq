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
	"io"

	snk "github.com/fogfish/iq/internal/iosystem/sink"
	src "github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/stream/spool"
	"github.com/spf13/cobra"
)

var (
// execWithBash   bool
// execWithGolang bool
// execWithPython bool
// execWorkDir string
)

func init() {
	rootCmd.AddCommand(taskCmd)

	// execCmd.Flags().BoolVar(&execWithBash, "bash", false, "enable bash in the command registry")
	// execCmd.Flags().BoolVar(&execWithGolang, "golang", false, "enable golang in the command registry")
	// execCmd.Flags().BoolVar(&execWithPython, "python", false, "enable python in the command registry")
	// taskCmd.Flags().StringVar(&execWorkDir, "workdir", "", "work directory for tools and commands")
}

var taskCmd = &cobra.Command{
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
	RunE:          withUsage(task),
}

func task(cmd *cobra.Command, args []string) error {
	llm, err := fmodel.build()
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm)
	if err != nil {
		return err
	}

	if len(finput.dir) > 0 && len(freply.dir) > 0 {
		rfs, err := source.Mount(finput.dir)
		if err != nil {
			return err
		}
		wfs, err := sink.Mount(freply.dir)
		if err != nil {
			return err
		}
		sp := spool.New(rfs, wfs, spool.IsImmutable)
		return sp.ForEach(context.Background(), "/",
			func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
				_, err := srv.Run(ctx, src.NewReader(path, r), snk.NewWriter(w))
				return err
			})
	}

	src, err := finput.build(args)
	if err != nil {
		return err
	}

	snk, err := freply.build()
	if err != nil {
		return err
	}

	// spinner := createSpinner()
	// defer spinner.Finish()

	// if rootThink {
	// 	spinner.Finish()
	// }

	// srv, err := worker.New().
	// 	Blueprint(rootPrompt, llm).
	// 	// Progress(func(doc *iosystem.Document, err error) {
	// 	// 	fmt.Fprintf(os.Stderr, "==> %s\n", ellipses(filepath.Base(doc.Path)))
	// 	// 	// spinner.Describe(
	// 	// 	// 	fmt.Sprintf("processing with %s", ellipses(filepath.Base(doc.Path))),
	// 	// 	// )
	// 	// }).
	// 	Build()
	// if err != nil {
	// 	return err
	// }

	// src, err := source.New().
	// 	Files(".", args...).
	// 	Stdin().
	// 	Build()
	// if err != nil {
	// 	return err
	// }

	// snk, err := sink.New().
	// 	File("xxx.txt").
	// 	Stdout(true).
	// 	Build()
	// if err != nil {
	// 	return err
	// }

	// b, err := parseInputStdin()
	// if err != nil {
	// 	return err
	// }

	// bp, err := blueprint.New(context.Background(), rootPrompt, factory)
	// if err != nil {
	// 	return err
	// }
	_, err = srv.Run(context.Background(), src, snk)
	// reply, err := srv.Prompt(context.Background(), string(b))
	if err != nil {
		return err
	}

	// f := colorjson.NewFormatter()
	// f.Indent = 2

	// s, _ := f.Marshal(reply)

	// os.Stdout.Write(s)
	// os.Stdout.WriteString("\n")

	return nil

	// agt, req, err := agentForTasks(execWorkDir)
	// if err != nil {
	// 	return err
	// }

	// if len(args) == 0 {
	// 	spinner.Describe("processing ...")
	// 	reply, err := agt.PromptOnce(context.Background(), req)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	spinner.Finish()

	// 	os.Stdout.Write(reply)
	// 	os.Stdout.WriteString("\n")
	// 	return nil
	// }

	// q, err := createSpool("/", "stdout", false, true)
	// if err != nil {
	// 	return err
	// }

	// for i, arg := range args {
	// 	path, err := filepath.Abs(arg)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	args[i] = filepath.Clean(path)
	// }

	// err = q.ForEachPath(context.Background(), args,
	// 	func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
	// 		spinner.Describe(fmt.Sprintf("processing with %s", ellipses(filepath.Base(path))))
	// 		defer spinner.Reset()

	// 		fd := reader.New(rootScanner, rootScannerChars, rootScannerChunk, r)
	// 		return reader.Process(ctx, agt, req, fd, w)
	// 	},
	// )
	// if err != nil {
	// 	return err
	// }

	// return nil
}

// type factory struct{}

// func (f *factory) LLM(name string) (chatter.Chatter, error) {
// 	llm, err := autoconfig.FromNetRC("iq")
// 	if err != nil {
// 		return nil, err
// 	}
// 	llm = aio.NewJsonLogger(os.Stderr, llm)
// 	return llm, nil
// }
