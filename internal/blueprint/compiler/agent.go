//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler

/*
import (
	"context"
	"fmt"
	"html/template"
	"os/exec"
	"strings"

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

// Agent represents a compiled agent bound with a thinker state machine.
type Agent struct {
	Node     *ast.AgentNode
	manifold *agent.Manifold[any, any]
	prompt   *template.Template
	servers  []Server
}

// active MCP server session
type Server struct {
	uid string
	cmd mcp.Transport
	cli *mcp.Client
	api *mcp.ClientSession
}

// create an instance of the agent from its AST node
func (agt *Agent) compile(ctx context.Context, llm chatter.Chatter) error {
	agt.servers = make([]Server, len(agt.Node.Servers))

	//
	// MCP

	registry := command.NewRegistry()
	for i, node := range agt.Node.Servers {
		srv, err := agt.server(node)
		if err != nil {
			return err
		}

		agt.servers[i] = srv
		if err := registry.Attach(node.Name, srv.api); err != nil {
			return err
		}
	}

	//
	// Prompt

	prompt, err := template.New("").Parse(agt.Node.Prompt)
	if err != nil {
		return err
	}
	agt.prompt = prompt

	//
	// Agent State Machine

	agt.manifold = agent.NewManifold(llm,
		codec.FromEncoder(agt.encode),
		codec.FromDecoder(agt.decode),
		registry,
	)

	return nil
}

func (agt *Agent) server(srv ast.ServerNode) (Server, error) {
	switch {
	case len(srv.Command) > 0:
		cmd := exec.Command(srv.Command[0], srv.Command[1:]...)
		rpc := &mcp.CommandTransport{Command: cmd}
		cli := mcp.NewClient(&mcp.Implementation{Name: srv.Name}, nil)
		api, err := cli.Connect(context.Background(), rpc, nil)
		if err != nil {
			return Server{}, err
		}
		return Server{uid: srv.Name, cmd: rpc, cli: cli, api: api}, nil

	case len(srv.Url) > 0:
		rpc, err := auth.NewTransport(auth.Config{Endpoint: srv.Url})
		if err != nil {
			return Server{}, err
		}

		cli := mcp.NewClient(&mcp.Implementation{Name: srv.Name}, nil)
		api, err := cli.Connect(context.Background(), rpc, nil)
		if err != nil {
			return Server{}, err
		}
		return Server{uid: srv.Name, cmd: rpc, cli: cli, api: api}, nil
	default:
		return Server{}, fmt.Errorf("server '%s' type '%s' is not supported", srv.Name, srv.Type)
	}
}

// Prompt executes the agent with given input and returns the output
func (agt *Agent) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error) {
	opt = append(opt, aio.Route(agt.Node.RunsOn))
	reply, err := agt.manifold.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

// validateSchema validates data against a JSON Schema
func (agt *Agent) validateSchema(data any, schema *jsonschema.Schema) error {
	// Resolve schema (handles references)
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return fmt.Errorf("failed to resolve schema: %w", err)
	}

	// Validate data
	if err := resolved.Validate(data); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

func (agt *Agent) encode(in any) (chatter.Message, error) {
	switch v := in.(type) {
	case map[string]any:
		return agt.encodeStruct(v)
	case string:
		return agt.encodeStruct(map[string]any{
			ast.ContextKeyInput:   v,
			ast.ContextKeyCurrent: v,
		})
	case []byte:
		return agt.encodeStruct(map[string]any{
			ast.ContextKeyInput:   string(v),
			ast.ContextKeyCurrent: string(v),
		})
	case nil:
		return agt.encodeStruct(map[string]any{})
	default:
		return nil, fmt.Errorf("unsupported input type: %T", in)
	}
}

func (agt *Agent) encodeStruct(in map[string]any) (chatter.Message, error) {
	// Add .input as alias for .current (agent's perspective)
	// This allows templates to use {{.input}} which semantically matches schema.input
	if current, hasCurrent := in[ast.ContextKeyCurrent]; hasCurrent {
		// Only set .input if not already present (don't overwrite standalone usage)
		if _, hasInput := in[ast.ContextKeyInput]; !hasInput {
			in[ast.ContextKeyInput] = current
		}
	}

	// Validate agent's input against schema
	if agt.Node.Schema.Input != nil {
		if input, hasInput := in[ast.ContextKeyInput]; hasInput {
			if err := agt.validateSchema(input, agt.Node.Schema.Input); err != nil {
				return nil, fmt.Errorf("input validation failed for agent '%s': %w", agt.Node.Name, err)
			}
		}
	}

	var sb strings.Builder
	err := agt.prompt.Execute(&sb, in)
	if err != nil {
		return nil, err
	}

	var prompt chatter.Prompt
	prompt.WithTask(sb.String())

	if agt.Node.Format == "json" {
		// Only harden with schema if reply schema is defined
		jsonify.Strings.Harden(&prompt, agt.Node.Schema.Reply)
	}

	return &prompt, nil
}

func (agt *Agent) decode(reply *chatter.Reply) (float64, any, error) {
	if agt.Node.Format == "json" {
		var obj any
		// Decode with schema (will validate if schema is non-nil)
		if err := jsonify.Strings.Decode(reply, agt.Node.Schema.Reply, &obj); err != nil {
			return 0.0, nil, err
		}

		return 1.0, obj, nil
	}

	return 1.0, reply.String(), nil
}
*/
