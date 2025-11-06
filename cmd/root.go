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

	"github.com/spf13/cobra"
)

func Execute(vsn string) {
	rootCmd.Version = vsn

	if err := rootCmd.Execute(); err != nil {
		e := err.Error()
		fmt.Fprintf(os.Stderr, "\n ❌ Something went wrong. Check the error below for details.\n   Run `iq help` for guidance.\n\n   %s\n\n", strings.ToUpper(e[:1])+e[1:])
		os.Exit(1)
	}
}

func init() {
	fglobal.apply(rootCmd)
	fmodel.apply(rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   "iq",
	Short: "AI workflows for the shell - no code required.",
	Long: `
 
(_)__ _      
| / _' |   AI workflows for the shell - no code required.
|_\__, |    
     |_|

The tool is designed to simplify running multi-step autonomous AI workflows
inside the shell, using YAML as a configuration language. 

Run 'iq help' for guidance.

See more info https://github.com/fogfish/iq
	`,
	Example: `
	# run workflow
	iq agent -f <yaml> FILE1, FILE2, ...

	# draft prompts markdown and agent workflows
	iq draft
	iq draft agent

	# batch processing
	iq agent batch -f <prompt> -I s3://... -O s3://...
	`,
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) { cmd.Help() },
}
