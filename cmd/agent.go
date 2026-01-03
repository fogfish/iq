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

	snk "github.com/fogfish/iq/internal/iosystem/sink"
	src "github.com/fogfish/iq/internal/iosystem/source"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(agentCmd)
	fagent.apply(agentCmd)
	finput.apply(agentCmd)
	freply.apply(agentCmd)

	agentCmd.AddCommand(agentBatchCmd)
	fspool.apply(agentBatchCmd)

	agentCmd.AddCommand(agentServeCmd)
	fservmcp.apply(agentServeCmd)

}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run an LLM agent or prompt according to the defined workflow.",
	Long: `
The agent command executes a defined workflow, processing input through
a configured Large Language Model and producing output according to the
specified prompt or workflow configuration.

The command supports multiple input sources:
- Direct file arguments
- Standard input (stdin) for piped content
- Chunks
- etc.

For batch processing of multiple files, use the 'batch' subcommand.
For running as an MCP server, use the 'serve' subcommand.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq agent -f <yml>
	iq agent -f <yml> FILE1 FILE2 ...
	iq agent -f <yml> --array data.json
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          agent,
}

func agent(cmd *cobra.Command, args []string) error {
	reporter := fglobal.reporter()

	// Report workflow loading
	if fagent.file != "" {
		reporter.WorkflowLoading(fagent.file)
	}

	llm, err := fmodel.build(reporter)
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm, reporter)
	if err != nil {
		reporter.WorkflowError(err)
		return err
	}

	src, err := finput.build(args)
	if err != nil {
		return err
	}

	snk, err := freply.build()
	if err != nil {
		return err
	}

	_, err = srv.Run(context.Background(), src, snk)
	if err != nil {
		return err
	}

	return nil
}

//------------------------------------------------------------------------------

var agentBatchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Process multiple files through the defined LLM workflow.",
	Long: `
batch command treats a mounted directory of files as a processing queue—reading
from an input directory, applying each file content as input to the workflow,
and writing the results to an output directory. This batch-oriented processing
is ideal for transformation, summarization or enhanced file processing at
scale—with minimal setup and full traceability of inputs and outputs.

The command support mounting of AWS S3 bucket. Use s3:// prefix
prefix to direct the utility (e.g. s3://bucket/path).

  iq agent batch -p <prompt> -I s3://... -O s3://...

Processing a large number of files may require the ability to start, stop, and
resume the utility reliably. To support this, you can use the --mutable flag,
which removes each input file immediately after it has been successfully processed.
This enables fault-tolerant, resumable execution by ensuring already-processed
files are skipped on subsequent runs.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq agent batch -f <prompt> -I <dir> -O <dir>
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          agentBatch,
}

func agentBatch(cmd *cobra.Command, args []string) error {
	if len(finput.dir) == 0 && len(freply.dir) == 0 {
		return fmt.Errorf("batch processing requires input and output directories")
	}

	reporter := fglobal.reporter()

	// Report batch processing start
	reporter.BatchStart(finput.dir, freply.dir, fspool.mutable)

	// Report workflow loading
	if fagent.file != "" {
		reporter.WorkflowLoading(fagent.file)
	}

	llm, err := fmodel.build(reporter)
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm, reporter)
	if err != nil {
		reporter.WorkflowError(err)
		return err
	}

	q, err := fspool.build()
	if err != nil {
		return err
	}

	return q.ForEach(context.Background(), "/",
		func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
			_, err := srv.Run(ctx, src.NewReader(path, r), snk.NewWriter(w))
			return err
		})
}

//------------------------------------------------------------------------------

var agentServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "start workflow as a server",
	Long: `
Start iq as an MCP server exposing the defined workflow as a tool to other agents.
`,
	Example: `
	iq agent serve -f myagent.yml 
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          agentServe,
}

func agentServe(cmd *cobra.Command, args []string) error {
	reporter := fglobal.reporter()

	llm, err := fmodel.build(reporter)
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm, reporter)
	if err != nil {
		return err
	}

	mcp, err := fservmcp.build()
	if err != nil {
		return err
	}

	// The ConduitWithReporter embeds *conduit.Conduit, so we can pass it directly
	if err := mcp.Run(context.Background(), srv.Conduit); err != nil {
		return err
	}

	return nil
}
