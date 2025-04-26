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
  iq config --bedrock                 configure Amazon Bedrock usage
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

	machine := n.Machine(rootProfile)
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
		if len(rootLLM) == 0 {
			rootLLM = "us.meta.llama3-3-70b-instruct-v1:0"
		}
		if err := bedrock(f); err != nil {
			return err
		}
	}

	if configOpenAI {
		if len(rootLLM) == 0 {
			rootLLM = "gpt-4o"
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
		if len(rootLLM) == 0 {
			rootLLM = "gemma-3-27b-it"
		}
		if err = lmstudio(f); err != nil {
			return err
		}
	}

	fPrintf(os.Stdout, "\n ✅ All good — you're set up and ready to go!\n")
	fPrintf(os.Stdout, "    %s is default model, use -m, --llm flags to override it.\n", rootLLM)
	fPrintf(os.Stdout, "    You might need to adjust config at ~/.netrc later, based on your setup.\n")
	return nil
}

func bedrock(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider bedrock
        region us-west-2
        family llama3
        model %s
`, rootProfile, rootLLM)
	return err
}

func openai(w io.Writer, secret string) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider openai
        host https://api.openai.com
        model %s
        secret %s
`, rootProfile, rootLLM, secret)
	return err
}

func lmstudio(w io.Writer) error {
	_, err := fmt.Fprintf(w, `
#
# added by iq
machine %s
        provider openai
        host http://localhost:1234
        model %s
        timeout 30
`, rootProfile, rootLLM)
	return err
}
