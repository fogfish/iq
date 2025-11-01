//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package llm

// Example usage of the LLM builder:
//
// Basic usage:
//
//	llm, err := llm.New().
//	    Profile("bedrock/claude-3-sonnet").
//	    Build()
//	if err != nil {
//	    return err
//	}
//
// With all features enabled:
//
//	llm, err := llm.New().
//	    Profile("bedrock/claude-3-sonnet").
//	    Debug(true).       // Enable debug logging to stderr
//	    Think(true).       // Print thinking content to stderr
//	    MaxEpoch(10).      // Limit to 10 epochs
//	    MaxTokens(4000).   // Limit output tokens
//	    Build()
//
// Mock mode for testing:
//
//	llm, err := llm.New().
//	    Profile("mock").   // Use mock LLM that echoes input
//	    Build()
//
// The builder applies decorators in this order:
//  1. Base LLM (from autoconfig or mock)
//  2. Thinking decorator (if Think is enabled)
//  3. Debug decorator (if Debug is enabled)
//  4. Quota decorator (if MaxEpoch or MaxTokens is set)
//
// CLI integration pattern:
//
//	func myCmd() *cobra.Command {
//	    var profile string
//	    var debug, think bool
//	    var maxEpoch, maxTokens int
//
//	    cmd := &cobra.Command{
//	        RunE: func(cmd *cobra.Command, args []string) error {
//	            llm, err := llm.New().
//	                Profile(profile).
//	                Debug(debug).
//	                Think(think).
//	                MaxEpoch(maxEpoch).
//	                MaxTokens(maxTokens).
//	                Build()
//	            if err != nil {
//	                return err
//	            }
//	            // Use llm...
//	        },
//	    }
//
//	    cmd.Flags().StringVar(&profile, "llm-profile", "", "LLM profile")
//	    cmd.Flags().BoolVar(&debug, "debug", false, "Debug mode")
//	    cmd.Flags().BoolVar(&think, "think", false, "Thinking mode")
//	    cmd.Flags().IntVar(&maxEpoch, "max-epoch", 0, "Max epochs")
//	    cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Max tokens")
//
//	    return cmd
//	}
