package cmd

import (
	"bytes"
	"context"

	snk "github.com/fogfish/iq/internal/iosystem/sink"
	src "github.com/fogfish/iq/internal/iosystem/source"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start workflow as a server",
	Long: `
xxx	
`,
	Example: `
	# Start iq in MCP mode
	iq serve --mcp
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(serve),
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

	server := mcp.NewServer(&mcp.Implementation{Name: fagent.agent}, nil)
	server.AddTool(
		&mcp.Tool{
			Name:        srv.Name,
			Description: srv.About,
			InputSchema: srv.Input,
			// map[string]any{
			// 	"type": "object",
			// 	"properties": map[string]any{
			// 		"name": map[string]any{
			// 			"type": "string",
			// 		},
			// 	},
			// },
			OutputSchema: srv.Reply,
			// map[string]any{
			// 	"type": "object",
			// 	"properties": map[string]any{
			// 		"message": map[string]any{
			// 			"type": "string",
			// 		},
			// 	},
			// },
		},
		func(ctx context.Context, ctr *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// var in any
			// if ctr.Params.Arguments != nil {
			// 	if err := json.Unmarshal(ctr.Params.Arguments, &in); err != nil {
			// 		return nil, fmt.Errorf("%w: %v", jsonrpc2.ErrInvalidParams, err)
			// 	}
			// }

			r := src.NewReaderJSON("", bytes.NewBuffer(ctr.Params.Arguments))

			var b bytes.Buffer
			w := snk.NewWriter(&b)

			_, err := srv.Run(ctx, r, w)
			if err != nil {
				return nil, err
			}
			// .Prompt(ctx, in)

			// v := map[string]any{"message": "Hello from iq server!"}
			// b, _ := json.Marshal(v)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: b.String()},
				},
				StructuredContent: b.Bytes(),
				// StructuredContent: map[string]any{"message": "Hello from iq server!"},
			}, nil
		},
	)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return err
	}

	return nil
}
