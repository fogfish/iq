package compiler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os/exec"
	"strings"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Agent struct {
	Node     *ast.AgentNode
	manifold *agent.Manifold[any, any]
	servers  []Server
}

type Server struct {
	uid string
	cmd *mcp.CommandTransport
	cli *mcp.Client
	api *mcp.ClientSession
}

func (agt *Agent) compile(ctx context.Context, llm chatter.Chatter) error {
	agt.servers = make([]Server, 0, len(agt.Node.Servers))

	registry := command.NewRegistry()
	for i, srv := range agt.Node.Servers {
		cmd := exec.Command(srv.Command)
		rpc := &mcp.CommandTransport{Command: cmd}
		cli := mcp.NewClient(&mcp.Implementation{Name: srv.Name}, nil)
		api, err := cli.Connect(context.Background(), rpc, nil)
		if err != nil {
			return err
		}
		agt.servers[i] = Server{uid: srv.Name, cmd: rpc, cli: cli, api: api}
		if err := registry.Attach(srv.Name, api); err != nil {
			return err
		}
	}

	agt.manifold = agent.NewManifold(llm,
		codec.FromEncoder(agt.encode),
		codec.FromDecoder(agt.decode),
		registry,
	)

	return nil
}

func (agt *Agent) Prompt(ctx context.Context, input any, opt ...chatter.Opt) (any, error) {
	// Validate input against input_schema if defined
	if agt.Node.Schema != nil && agt.Node.Schema.Input != nil {
		if err := agt.validateSchema(input, agt.Node.Schema.Input, "input"); err != nil {
			return nil, fmt.Errorf("input validation failed for agent '%s': %w", agt.Node.Name, err)
		}
	}

	// Execute the agent
	reply, err := agt.manifold.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	// Validate output against output_schema if defined
	if agt.Node.Schema != nil && agt.Node.Schema.Reply != nil {
		if err := agt.validateSchema(reply, agt.Node.Schema.Reply, "output"); err != nil {
			return nil, fmt.Errorf("output validation failed for agent '%s': %w", agt.Node.Name, err)
		}
	}

	return reply, nil
}

// validateSchema validates data against a JSON Schema
func (agt *Agent) validateSchema(data any, schemaMap map[string]any, context string) error {
	// Convert schema map to JSON and unmarshal into Schema struct
	schemaBytes, err := json.Marshal(schemaMap)
	if err != nil {
		return fmt.Errorf("failed to marshal %s schema: %w", context, err)
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return fmt.Errorf("invalid %s schema: %w", context, err)
	}

	// Resolve schema (handles references)
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return fmt.Errorf("failed to resolve %s schema: %w", context, err)
	}

	// Validate data
	if err := resolved.Validate(data); err != nil {
		return fmt.Errorf("%s validation error: %w", context, err)
	}

	return nil
}

func (agt *Agent) encode(in any) (chatter.Message, error) {
	switch v := in.(type) {
	case map[string]any:
		return agt.encodeStruct(v)
	case string:
		return agt.encodeStruct(map[string]any{"input": v})
	case []byte:
		return agt.encodeStruct(map[string]any{"input": string(v)})
	case nil:
		return agt.encodeStruct(map[string]any{})
	default:
		return nil, fmt.Errorf("unsupported input type: %T", in)
	}
}

func (agt *Agent) encodeStruct(in any) (chatter.Message, error) {
	txt, err := template.New("").Parse(agt.Node.Prompt)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	err = txt.Execute(&sb, in)
	if err != nil {
		return nil, err
	}

	var prompt chatter.Prompt
	prompt.WithTask(sb.String())

	if agt.Node.Format == "json" {
		jsonify.Strings.Harden(&prompt)
	}

	return &prompt, nil
}

func (agt *Agent) decode(reply *chatter.Reply) (float64, any, error) {
	if agt.Node.Format == "json" {
		var obj any
		if err := jsonify.Strings.Decode(reply, &obj); err != nil {
			return 0.0, nil, err
		}

		return 1.0, obj, nil
	}

	return 1.0, reply.String(), nil
}
