package cmd

import (
	"context"
	"fmt"
	"io"

	snk "github.com/fogfish/iq/internal/iosystem/sink"
	src "github.com/fogfish/iq/internal/iosystem/source"
	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/stream/spool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(agentCmd)
	fagent.apply(rootCmd)
	finput.apply(rootCmd)
	freply.apply(rootCmd)

	agentCmd.AddCommand(agentBatchCmd)
	agentCmd.AddCommand(agentServeCmd)
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run an LLM agent or prompt according to the defined workflow.",
	Long: `
xxx


See more info https://github.com/fogfish/iq
	`,
	Example: `
	# No input
	iq agent -p mytask.yml
	iq agent -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq agent --python
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          agent,
}

func agent(cmd *cobra.Command, args []string) error {
	llm, err := fmodel.build()
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm)
	if err != nil {
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
xxx


See more info https://github.com/fogfish/iq
	`,
	Example: `
	# No input
	iq agent -p mytask.yml
	iq agent -p mytask.yml FILE1 FILE2 ...
	echo "Using available tools draw the rainbow?" | iq agent --python
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          batch,
}

func batch(cmd *cobra.Command, args []string) error {
	if len(finput.dir) == 0 && len(freply.dir) == 0 {
		return fmt.Errorf("batch processing requires input and output directories")
	}

	llm, err := fmodel.build()
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm)
	if err != nil {
		return err
	}

	rfs, err := source.Mount(finput.dir)
	if err != nil {
		return err
	}

	wfs, err := sink.Mount(freply.dir)
	if err != nil {
		return err
	}

	// TODO: spool flags
	sp := spool.New(rfs, wfs, spool.IsImmutable)
	return sp.ForEach(context.Background(), "/",
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
xxx	
`,
	Example: `
	# Start iq in MCP server
	iq agent serve -a myagent.yml 
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          serve,
}

func serve(cmd *cobra.Command, args []string) error {
	llm, err := fmodel.build()
	if err != nil {
		return err
	}

	srv, err := fagent.build(llm)
	if err != nil {
		return err
	}

	switch {
	case len(srv.Name) == 0:
		return fmt.Errorf("agent serve requires defined tool name")
	case srv.Input == nil:
		return fmt.Errorf("agent serve requires defined input schema")
	case srv.Reply == nil:
		return fmt.Errorf("agent serve requires defined reply schema")
	}

	server := mcp.NewServer(&mcp.Implementation{Name: fagent.agent}, nil)
	server.AddTool(
		&mcp.Tool{
			Name:         srv.Name,
			Description:  srv.About,
			InputSchema:  srv.Input,
			OutputSchema: srv.Reply,
		},
		srv.RunAsCmd,
	)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return err
	}

	return nil
}
