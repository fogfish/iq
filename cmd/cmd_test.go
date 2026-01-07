//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"
)

func TestAgentWithPrompt(t *testing.T) {
	// iq agent -f testdata/prompt.md
	fmodel.profile = "mock"
	fagent.file = "testdata/prompt.md"

	out, err := sut(agentCmd, nil)

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Hello World!"),
	)
}

func TestAgentWithWorkflow(t *testing.T) {
	// iq agent -f testdata/workflow.yml
	fmodel.profile = "mock"
	fagent.file = "testdata/workflow.yml"

	out, err := sut(agentCmd, nil)

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Hello World!"),
	)
}

func TestAgentWithFile(t *testing.T) {
	// iq agent -f testdata/prompt.md FILE1
	fmodel.profile = "mock"
	fagent.file = "testdata/workflow.yml"

	out, err := sut(agentCmd, []string{"testdata/doc.txt"})

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Hello World!"),
		it.String(out).Contain("Content."),
	)
}

func TestAgentWithFiles(t *testing.T) {
	// iq agent -f testdata/prompt.md FILE1 FILE2
	fmodel.profile = "mock"
	fagent.file = "testdata/workflow.yml"

	out, err := sut(agentCmd, []string{"testdata/doc.txt", "testdata/doc.txt"})

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Hello World! Content.Hello World! Content."),
	)
}

func TestAgentWithFileMerge(t *testing.T) {
	// iq agent -f testdata/prompt.md --merge FILE1 FILE2
	fmodel.profile = "mock"
	fagent.file = "testdata/workflow.yml"
	fagent.merge = true

	out, err := sut(agentCmd, []string{"testdata/doc.txt", "testdata/doc.txt"})

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Hello World!"),
		it.String(out).Contain("Content. Content."),
	)
}

func TestDraft(t *testing.T) {
	// iq draft
	fmodel.profile = "mock"

	out, err := sut(draftCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("[Describe the task and goals clearly and concisely]."),
	)
}

func TestDraftYaml(t *testing.T) {
	// iq draft agent
	fmodel.profile = "mock"

	out, err := sut(draftYamlCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("uses: prompts/prompt.md"),
	)
}

//------------------------------------------------------------------------------

// helper utility to sut cobra.Command
func sut(cmd *cobra.Command, args []string) (string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	err := cmd.RunE(cmd, args)

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}
