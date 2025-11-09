//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents an MCP server exposing defined tools.
type Server interface {
	Run(context.Context, *conduit.Conduit) error
}

// Builder helps to construct an MCP server with defined tools.
type Builder struct {
	server  *mcp.Server
	conduit *conduit.Conduit
}

// New creates a new MCP server builder.
func New() *Builder {
	return &Builder{}
}

// Server sets up the MCP server with the given name.
func (b *Builder) Server(name string) *Builder {
	b.server = mcp.NewServer(&mcp.Implementation{Name: name}, nil)
	return b
}

// Conduit configures the MCP server with the given conduit (workflow).
func (b *Builder) Conduit(c *conduit.Conduit) *Builder {
	b.server = mcp.NewServer(&mcp.Implementation{Name: c.Name}, nil)
	return b
}

// Build finalizes the MCP server construction.
func (b *Builder) Build() (Server, error) {
	return b, nil
}

// Run starts the MCP server and exposes the defined tool.
func (b *Builder) Run(ctx context.Context, srv *conduit.Conduit) error {
	switch {
	case len(srv.Name) == 0:
		return fmt.Errorf("agent serve requires defined tool name")
	case srv.Input == nil:
		return fmt.Errorf("agent serve requires defined input schema")
	}

	b.conduit = srv

	if srv.Reply != nil {
		b.server.AddTool(
			&mcp.Tool{
				Name:         srv.Name,
				Description:  srv.About,
				InputSchema:  srv.Input,
				OutputSchema: srv.Reply,
			},
			b.Cmd,
		)
	} else {
		b.server.AddTool(
			&mcp.Tool{
				Name:        srv.Name,
				Description: srv.About,
				InputSchema: srv.Input,
				OutputSchema: map[string]any{
					"type":     "object",
					"required": []string{"reply"},
					"properties": map[string]any{
						"reply": map[string]any{
							"type": "string",
						},
					},
				},
			},
			b.UnsafeCmd,
		)
	}

	if err := b.server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return err
	}

	return nil
}

func (b *Builder) Cmd(ctx context.Context, ctr *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, err := b.conduit.Cmd(ctx, ctr.Params.Arguments)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: out.String()},
		},
		StructuredContent: out.Bytes(),
	}, nil
}

func (b *Builder) UnsafeCmd(ctx context.Context, ctr *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, err := b.conduit.Cmd(ctx, ctr.Params.Arguments)
	if err != nil {
		return nil, err
	}

	reply := map[string]any{"reply": out.String()}
	json, err := json.Marshal(reply)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(json)},
		},
		StructuredContent: json,
	}, nil
}
