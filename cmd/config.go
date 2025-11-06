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

	iq ask -p aws
	iq ask --profile gpt4o

See more info https://github.com/fogfish/iq
	`,
	Example: `
  iq config --openai <secret-key>     configure OpenAI usage
  iq config --bedrock                 configure Amazon Bedrock (Converse API) usage
  iq config --lmstudio                connect to a local LM Studio instance

  iq config --bedrock --profile aws --llm-id us.meta.llama3-3-70b-instruct-v1:0
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

	machine := n.Machine(fmodel.profile)
	if machine != nil {
		fmt.Fprintf(os.Stdout, "\n ✅ All good — you're set up and ready to go!\n")
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
		if len(fmodel.model) == 0 {
			fmodel.model = "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
		}
		if err := converse(f); err != nil {
			return err
		}
	}

	if configOpenAI {
		if len(fmodel.model) == 0 {
			fmodel.model = "gpt-5"
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
		if len(fmodel.model) == 0 {
			fmodel.model = "gemma-3-27b-it"
		}
		if err = lmstudio(f); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "\n ✅ All good — you're set up and ready to go!\n")
	fmt.Fprintf(os.Stdout, "    %s is default model, use -m, --llm-id flags to override it.\n", fmodel.model)
	fmt.Fprintf(os.Stdout, "    You might need to adjust config at ~/.netrc later, based on your setup.\n")
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

`, fmodel.profile, fmodel.model)
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

`, fmodel.profile, fmodel.model, secret)
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

`, fmodel.profile, fmodel.model)
	return err
}
