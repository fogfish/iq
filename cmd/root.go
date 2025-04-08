//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fogfish/iq/internal/prompt"
	"github.com/fogfish/iq/internal/service"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/command"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute(vsn string) {
	rootCmd.Version = vsn

	if err := rootCmd.Execute(); err != nil {
		e := err.Error()
		fmt.Println(strings.ToUpper(e[:1]) + e[1:])
		os.Exit(1)
	}
}

var (
	rootProfile string
	rootLLM     string
	rootPrompt  string
	rootDebug   bool
	rootSilent  bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootProfile, "config", "c", "iq", "access profile to LLM provider at ~/.netrc")
	rootCmd.PersistentFlags().StringVarP(&rootLLM, "llm", "m", "", "overrides LLM model defined at ~/.netrc")
	rootCmd.PersistentFlags().StringVarP(&rootPrompt, "prompt", "p", "", "path to prompt yaml file")
	rootCmd.PersistentFlags().BoolVar(&rootDebug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVarP(&rootSilent, "silent", "s", false, "not output")
}

var rootCmd = &cobra.Command{
	Use:   "iq",
	Short: "a fast and lightweight CLI tool for running LLM-powered agents",
	Long: `
iq is a fast and lightweight CLI tool for running LLM-powered agents.

TBD

See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq
	`,
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) { cmd.Help() },
}

//------------------------------------------------------------------------------

func configLLM(llmid string) (chatter.Chatter, error) {
	llm, err := createLLM(llmid)
	if err != nil {
		return nil, err
	}

	if !rootDebug {
		return llm, nil
	}

	return aio.NewLogger(os.Stderr, llm), nil
}

func createLLM(llmid string) (chatter.Chatter, error) {
	if len(llmid) != 0 {
		return autoconfig.New(rootProfile, llmid)
	}

	if len(rootLLM) != 0 {
		return autoconfig.New(rootProfile, rootLLM)
	}

	return autoconfig.New(rootProfile)
}

//------------------------------------------------------------------------------

func parsePrompt() (*viper.Viper, error) {
	if len(rootPrompt) != 0 {
		return prompt.ParseFile(rootPrompt)
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Size() > 0 {
		return prompt.Parse(os.Stdin)
	}

	return nil, fmt.Errorf("undefined prompt")
}

func agentForPrompts() (*service.Prompter, *viper.Viper, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	llm, err := configLLM(in.GetString("llm"))
	if err != nil {
		return nil, nil, err
	}

	agt, err := service.NewPrompter(llm, in)
	if err != nil {
		return nil, nil, err
	}

	return agt, in, nil
}

func agentForTasks(workdir string) (*service.Worker, *viper.Viper, error) {
	in, err := parsePrompt()
	if err != nil {
		return nil, nil, err
	}

	llm, err := configLLM(in.GetString("llm"))
	if err != nil {
		return nil, nil, err
	}

	if taskWithBash || taskWithGolang || taskWithPython {
		registry := []string{}
		if taskWithBash {
			registry = append(registry, command.BASH)
		}
		if taskWithGolang {
			registry = append(registry, command.GOLANG)
		}
		if taskWithPython {
			registry = append(registry, command.PYTHON)
		}

		in.Set(prompt.YAML_REGISTRY, registry)
	}

	agt, err := service.NewWorker(llm, in, workdir)
	if err != nil {
		return nil, nil, err
	}

	return agt, in, nil
}

//------------------------------------------------------------------------------

type spinner interface {
	Describe(string)
	Finish() error
	Reset()
}

type nospinner int

func (nospinner) Describe(string) {}
func (nospinner) Finish() error   { return nil }
func (nospinner) Reset()          {}

func createSpinner() spinner {
	if rootSilent {
		return nospinner(0)
	}

	return progressbar.NewOptions(-1,
		progressbar.OptionShowBytes(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSpinnerType(11),
	)
}
