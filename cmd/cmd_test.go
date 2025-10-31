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

func TestTell(t *testing.T) {
	// iq tell -s -m ... -p ...
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/prompt.yml"

	out, err := sut(tellCmd, nil)

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("What are the colors of rainbow?"),
	)
}

func TestTellWithFiles(t *testing.T) {
	// iq tell -s -m ... -p ... FILE1 FILE2
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/prompt.yml"

	out, err := sut(tellCmd, []string{"../examples/prompt/doc/sun.txt", "../examples/prompt/doc/sky.txt"})
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("What are the colors of rainbow?"),
		it.String(out).Contain("sun"),
		it.String(out).Contain("sky"),
	)
}

func TestTellWithMergedFiles(t *testing.T) {
	// iq tell -s -m ... -p ... --merge FILE1 FILE2
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/prompt.yml"
	tellMergeFiles = true

	out, err := sut(tellCmd, []string{"../examples/prompt/doc/sun.txt", "../examples/prompt/doc/sky.txt"})
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("What are the colors of rainbow?"),
		it.String(out).Contain("sun"),
		it.String(out).Contain("sky"),
	)
}

func TestTellWithArgs(t *testing.T) {
	// iq tell -s -m ... -p ...
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/args.yml"

	out, err := sut(tellCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("What are the colors of rainbow?"),
	)
}

func TestTellWithArgsOverride(t *testing.T) {
	// iq tell -s -m ... -p ... --input "key1=value1,key2=value2"
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/args.yml"
	rootInput = "key1=value1,key2,key3=,name=sun"

	out, err := sut(tellCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("What are the colors of sun?"),
	)
}

func TestExec(t *testing.T) {
	// iq exec -s -m ... -p ...
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/task.yml"

	out, err := sut(taskCmd, nil)

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Use available tools to complete the workflow"),
	)
}

func TestExecWithFiles(t *testing.T) {
	// iq exec -s -m ... -p ... FILE1 FILE2
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/task.yml"

	out, err := sut(taskCmd, []string{"../examples/prompt/doc/sun.txt", "../examples/prompt/doc/sky.txt"})
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Use available tools to complete the workflow"),
	)
}

func TestAsk(t *testing.T) {
	// iq ask -s -m ... -p ... -d ... -o ...
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/prompt.yml"
	askDir = "../examples/prompt/doc"
	askOut = "/tmp"

	_, err := sut(askCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
	)
}

func TestRun(t *testing.T) {
	// iq run -s -m ... -p ... -d ... -o ...
	rootSilent = true
	rootLLM.Model = "mock"
	rootPrompt = "./testdata/task.yml"
	runDir = "../examples/prompt/doc"
	runOut = "/tmp"

	_, err := sut(runCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
	)
}

func TestDraft(t *testing.T) {
	// iq draft
	rootSilent = true
	rootLLM.Model = "mock"

	out, err := sut(draftCmd, nil)
	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("[Describe the task and goals clearly and concisely]."),
	)
}

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
