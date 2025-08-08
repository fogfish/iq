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
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/jdxcode/netrc"
	"github.com/spf13/cobra"
)

var (
	configBedrock  bool
	configOpenAI   bool
	configLMStudio bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolVar(&configBedrock, "bedrock", false, "configure Amazon Bedrock usage")
	configCmd.Flags().BoolVar(&configOpenAI, "openai", false, "configure OpenAI usage, secret key is required")
	configCmd.Flags().BoolVar(&configLMStudio, "lmstudio", false, "connect to a local LM Studio instance")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "configure utility",
	Long: `
Configure 'iq' to connect with your preferred LLM providers.

Once configured, 'iq' will remember your settings at ~/.netrc

Profiles:
  You can manage multiple configurations by setting different profile names.
  The credentials are read from ~/.netrc under the specified profile.

	iq ask -c aws
	iq ask --config gpt4o

See more info https://github.com/fogfish/iq
	`,
	Example: `
  iq config --openai <secret-key>     configure OpenAI usage
  iq config --bedrock                 configure Amazon Bedrock (Converse API) usage
  iq config --lmstudio                connect to a local LM Studio instance

  iq config --bedrock --config aws --llm us.meta.llama3-3-70b-instruct-v1:0
  `,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          config,
}

func config(cmd *cobra.Command, args []string) error {
	usr, err := user.Current()
	if err != nil {
		return err
	}

	file := filepath.Join(usr.HomeDir, ".netrc")
	n, err := netrc.Parse(file)
	if err != nil {
		return err
	}

	machine := n.Machine(rootLLM.Profile)
	if machine != nil {
		fPrintf(os.Stdout, "\n ✅ All good — you're set up and ready to go!\n")
		return nil
	}

	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if !configBedrock && !configOpenAI && !configLMStudio {
		return fmt.Errorf("provider is not defined")
	}

	if configBedrock {
		if len(rootLLM.Model) == 0 {
			rootLLM.Model = "us.anthropic.claude-3-7-sonnet-20250219-v1:0"
		}
		if err := converse(f); err != nil {
			return err
		}
	}

	if configOpenAI {
		if len(rootLLM.Model) == 0 {
			rootLLM.Model = "gpt-4o"
		}
		secret := "<secret>"
		if len(args) > 0 {
			secret = args[0]
		}
		if err := openai(f, secret); err != nil {
			return err
		}
	}

	if configLMStudio {
		if len(rootLLM.Model) == 0 {
			rootLLM.Model = "gemma-3-27b-it"
		}
		if err = lmstudio(f); err != nil {
			return err
		}
	}

	fPrintf(os.Stdout, "\n ✅ All good — you're set up and ready to go!\n")
	fPrintf(os.Stdout, "    %s is default model, use -m, --llm flags to override it.\n", rootLLM.Model)
	fPrintf(os.Stdout, "    You might need to adjust config at ~/.netrc later, based on your setup.\n")
	return nil
}

func converse(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider provider:bedrock/foundation/converse
        model %s
        region us-west-2

`, rootLLM.Profile, rootLLM.Model)
	return err
}

func openai(w io.Writer, secret string) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider provider:openai/foundation/gpt
        model %s
        host https://api.openai.com
        secret %s

`, rootLLM.Profile, rootLLM.Model, secret)
	return err
}

func lmstudio(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider provider:openai/foundation/gpt
        model %s
        host http://localhost:1234
        timeout 30

`, rootLLM.Profile, rootLLM.Model)
	return err
}
