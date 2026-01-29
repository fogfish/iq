//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/fogfish/iq/internal/auth"
	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

/*

TODO: 6. Manifold MCP Server Cleanup


func (agt *Manifold) Close() error {
    for _, srv := range agt.servers {
        srv.cli.Close()  // ← No such method exists
    }
}

*/

// Manifold is an agent state machine bound with input/output event types.
type Manifold struct {
	Node     *ast.AgentNode
	manifold *agent.Manifold[Event, Gist]
	static   string
	prompt   *template.Template
	servers  []server
}

var _ Prompter = (*Manifold)(nil)

type server struct {
	uid string
	cmd mcp.Transport
	cli *mcp.Client
	api *mcp.ClientSession
}

// Create a new manifold instance
func NewManifold(node *ast.AgentNode, llm chatter.Chatter) (agt *Manifold, err error) {
	agt = &Manifold{Node: node}

	//
	// Prompt
	static, dynamic, found := strings.Cut(agt.Node.Prompt, "## INPUT")
	if found {
		agt.static = static
		agt.prompt, err = template.New("").Parse(dynamic)
		if err != nil {
			return
		}
	} else {
		agt.prompt, err = template.New("").Parse(agt.Node.Prompt)
		if err != nil {
			return
		}
	}

	//
	// MCP
	agt.servers = make([]server, len(agt.Node.Servers))
	registry := command.NewRegistry()
	for i, node := range agt.Node.Servers {
		agt.servers[i], err = connect(node)
		if err != nil {
			return
		}

		if err = registry.Attach(node.Name, agt.servers[i].api); err != nil {
			return
		}
	}

	agt.manifold = agent.NewManifold(llm,
		codec.FromEncoder(agt.encode),
		codec.FromDecoder(agt.decode),
		registry,
	)

	return
}

func connect(srv ast.ServerNode) (server, error) {
	switch {
	case len(srv.Command) > 0:
		cmd := exec.Command(srv.Command[0], srv.Command[1:]...)
		rpc := &mcp.CommandTransport{Command: cmd}
		cli := mcp.NewClient(&mcp.Implementation{Name: srv.Name}, nil)
		api, err := cli.Connect(context.Background(), rpc, nil)
		if err != nil {
			return server{}, err
		}
		return server{uid: srv.Name, cmd: rpc, cli: cli, api: api}, nil

	case len(srv.Url) > 0:
		rpc, err := auth.NewTransport(auth.Config{Endpoint: srv.Url})
		if err != nil {
			return server{}, err
		}

		cli := mcp.NewClient(&mcp.Implementation{Name: srv.Name}, nil)
		api, err := cli.Connect(context.Background(), rpc, nil)
		if err != nil {
			return server{}, err
		}
		return server{uid: srv.Name, cmd: rpc, cli: cli, api: api}, nil
	default:
		return server{}, fmt.Errorf("server '%s' type '%s' is not supported", srv.Name, srv.Type)
	}
}

func (agt *Manifold) Config(map[string]*Job) error {
	return nil
}

// Executes the agent with given input and returns the output
func (agt *Manifold) Prompt(ctx context.Context, in Event, opt ...chatter.Opt) (Event, error) {
	if agt.Node.RunsOn == "/dev/null" {
		var sb strings.Builder
		err := agt.prompt.Execute(&sb, in.ToState())
		if err != nil {
			return Event{}, err
		}

		return in.copy(in.Key, Text(sb.String())), nil
	}

	opt = append(opt, aio.Route(agt.Node.RunsOn))
	reply, err := agt.manifold.Prompt(ctx, in, opt...)
	if err != nil {
		return in, err
	}

	return in.copy(in.Key, reply), nil
}

func (agt *Manifold) encode(in Event) (chatter.Message, error) {
	if agt.Node.Schema.Input != nil {
		if err := agt.validateSchema(in, agt.Node.Schema.Input); err != nil {
			return nil, fmt.Errorf("input validation failed for agent '%s': %w", agt.Node.Name, err)
		}
	}

	// Note: this is the only place where event is converted to prompt
	var sb strings.Builder
	err := agt.prompt.Execute(&sb, in.ToState())
	if err != nil {
		return nil, err
	}

	var prompt chatter.Prompt
	if agt.static != "" {
		prompt.WithTask(agt.static)
		prompt.WithBlob("", sb.String())
	} else {
		prompt.WithBlob("", sb.String())
	}

	if agt.Node.Format == "json" {
		// Only harden with schema if reply schema is defined
		jsonify.Strings.Harden(&prompt, agt.Node.Schema.Reply)
	}

	return &prompt, nil
}

// validateSchema validates data against a JSON Schema
func (agt *Manifold) validateSchema(in Event, schema *jsonschema.Schema) error {
	// resolve schema references
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return fmt.Errorf("failed to resolve schema: %w", err)
	}

	if err := resolved.Validate(in.Current); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

func (agt *Manifold) decode(reply *chatter.Reply) (float64, Gist, error) {
	// Check if reply contains binary content (images)
	if len(reply.Content) > 0 {
		binaries := make([]Binary, 0)

		for _, c := range reply.Content {
			switch v := c.(type) {
			case *chatter.Binary:
				binaries = append(binaries, Binary{
					Type: v.Type,
					Data: v.Data,
				})
			}
		}

		// If we have binaries, return them
		if len(binaries) > 0 {
			if len(binaries) == 1 {
				return 1.0, binaries[0], nil
			}

			list := make([]any, len(binaries))
			for i, b := range binaries {
				list[i] = b
			}
			return 1.0, List(list), nil
		}
	}

	// JSON format handling
	if agt.Node.Format == "json" {
		var obj any
		// Decode with schema (will validate if schema is non-nil)
		if err := jsonify.Strings.Decode(reply, agt.Node.Schema.Reply, &obj); err != nil {
			return 0.0, nil, err
		}

		switch v := obj.(type) {
		case []any:
			return 1.0, List(v), nil
		case map[string]any:
			return 1.0, Json(v), nil
		default:
			b, err := json.Marshal(obj)
			if err != nil {
				return 0.0, nil, fmt.Errorf("unsupported reply shape: %+v", obj)
			}

			return 0.0, nil, fmt.Errorf("unsupported reply shape: %s", string(b))
		}
	}

	// Default text handling
	return 1.0, Text(reply.String()), nil
}
