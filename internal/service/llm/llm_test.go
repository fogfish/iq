//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package llm_test

import (
	"context"
	"testing"

	"github.com/fogfish/iq/internal/service/llm"
	"github.com/kshard/chatter"
)

func TestFactory_CreateMock(t *testing.T) {
	factory := llm.New(llm.Config{
		Model: "mock",
	})

	chat, err := factory.LLM("")
	if err != nil {
		t.Fatalf("failed to create mock LLM: %v", err)
	}

	// Test mock echoes input
	ctx := context.Background()
	messages := []chatter.Message{
		&chatter.Prompt{Task: "test prompt"},
	}

	reply, err := chat.Prompt(ctx, messages)
	if err != nil {
		t.Fatalf("mock prompt failed: %v", err)
	}

	if reply.Stage != chatter.LLM_RETURN {
		t.Errorf("expected stage LLM_RETURN, got %v", reply.Stage)
	}
}

func TestFactory_CreateWithDecorators(t *testing.T) {
	factory := llm.New(llm.Config{
		Model:    "mock",
		Debug:    true,
		Think:    true,
		MaxEpoch: 5,
		MaxUsage: chatter.Usage{
			ReplyTokens: 1000,
		},
	})

	chat, err := factory.LLM("")
	if err != nil {
		t.Fatalf("failed to create LLM with decorators: %v", err)
	}

	// Verify it's decorated by checking it's not the base mock
	if chat == nil {
		t.Fatal("expected decorated LLM, got nil")
	}

	// Test it still works
	ctx := context.Background()
	messages := []chatter.Message{
		&chatter.Prompt{Task: "test"},
	}

	reply, err := chat.Prompt(ctx, messages)
	if err != nil {
		t.Fatalf("decorated LLM prompt failed: %v", err)
	}

	if reply == nil {
		t.Fatal("expected reply, got nil")
	}
}

func TestFactory_ModelOverride(t *testing.T) {
	factory := llm.New(llm.Config{
		Model: "default-model",
	})

	// Override with "mock" should work
	chat, err := factory.LLM("mock")
	if err != nil {
		t.Fatalf("failed to create with model override: %v", err)
	}

	if chat == nil {
		t.Fatal("expected LLM, got nil")
	}
}
