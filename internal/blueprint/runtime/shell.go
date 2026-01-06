package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"text/template"

	"github.com/fogfish/iq/internal/blueprint/ast"
	"github.com/kshard/chatter"
)

type Shell struct {
	Command string // Compiled template for the command
	Shell   string // Shell to use (sh, bash, zsh, etc.)
}

var _ Prompter = (*Shell)(nil)

func NewShell(shell, command string) (*Shell, error) {
	return &Shell{Shell: shell, Command: command}, nil
}

func (s *Shell) Config(map[string]*Job) error {
	return nil
}

func (s *Shell) Prompt(ctx context.Context, in Event, opts ...chatter.Opt) (Event, error) {
	// Render command template with workflow context
	tmpl, err := template.New("run").Parse(s.Command)
	if err != nil {
		return in, fmt.Errorf("failed to parse command template: %w", err)
	}

	var cmdBuf bytes.Buffer
	err = tmpl.Execute(&cmdBuf, map[string]any{
		ast.ContextKeyDocument: in.Document,
		ast.ContextKeyInput:    in.Current,
		ast.ContextKeyCurrent:  in.Current,
		ast.ContextKeySteps:    in.Steps,
	})
	if err != nil {
		return in, fmt.Errorf("failed to render command template: %w", err)
	}
	rendered := cmdBuf.String()

	// Execute command using exec package
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, s.Shell, "-c", rendered)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return in, fmt.Errorf("command failed: %w\nCommand: %s\nStderr: %s", err, rendered, stderrStr)
		}
		return in, fmt.Errorf("command failed: %w\nCommand: %s", err, rendered)
	}

	return in.copy(in.Key, Text(stdout.String())), nil
}
