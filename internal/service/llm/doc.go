//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package llm

// Example usage of the LLM factory:
//
// Basic usage:
//
//	factory := llm.NewFactory(llm.Config{
//	    Profile: "bedrock",
//	    Model:   "claude-3-sonnet",
//	})
//	chat, err := factory.Create("")
//	if err != nil {
//	    return err
//	}
//
// With all features enabled:
//
//	factory := llm.NewFactory(llm.Config{
//	    Profile:  "bedrock",
//	    Model:    "claude-3-sonnet",
//	    Debug:    true,  // Enable debug logging to stderr
//	    Think:    true,  // Print thinking content to stderr
//	    MaxEpoch: 10,    // Limit to 10 epochs
//	    MaxUsage: chatter.Usage{
//	        ReplyTokens: 4000,  // Limit output tokens
//	    },
//	})
//	chat, err := factory.Create("")
//
// Mock mode for testing:
//
//	factory := llm.NewFactory(llm.Config{
//	    Model: "mock",  // Use mock LLM that echoes input
//	})
//	chat, err := factory.Create("")
//
// Override model at creation time:
//
//	factory := llm.NewFactory(llm.Config{
//	    Profile: "bedrock",
//	    Model:   "claude-3-sonnet",  // Default model
//	})
//	// Use different model for this instance
//	chat, err := factory.Create("claude-3-opus")
//
// The factory applies decorators in this order:
//  1. Base LLM (from autoconfig or mock)
//  2. Thinking decorator (if Think is enabled)
//  3. Debug decorator (if Debug is enabled)
//  4. Quota decorator (if MaxEpoch or MaxUsage is set)
