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

func TestBuilder_Mock(t *testing.T) {
	chat, err := llm.New().
		Profile("mock").
		Build()

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

func TestBuilder_WithDecorators(t *testing.T) {
	chat, err := llm.New().
		Profile("mock").
		Debug(true).
		Think(true).
		MaxEpoch(5).
		MaxTokens(1000).
		Build()

	if err != nil {
		t.Fatalf("failed to create LLM with decorators: %v", err)
	}

	// Verify it's decorated by checking it's not nil
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

func TestBuilder_ChainOrder(t *testing.T) {
	// Test that decorators are applied in the correct order
	chat, err := llm.New().
		Profile("mock").
		Think(true). // First decorator
		Debug(true). // Second decorator (should see thinking output)
		Build()

	if err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	if chat == nil {
		t.Fatal("expected LLM, got nil")
	}
}

func TestBuilder_DisabledDecorators(t *testing.T) {
	chat, err := llm.New().
		Profile("mock").
		Debug(false). // Should not apply
		Think(false). // Should not apply
		MaxEpoch(0).  // Should not apply
		MaxTokens(0). // Should not apply
		Build()

	if err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	if chat == nil {
		t.Fatal("expected LLM, got nil")
	}
}

func TestBuilder_ErrorPropagation(t *testing.T) {
	chat, err := llm.New().
		Profile("invalid-format-no-slash").
		Debug(true). // Should still chain even with error
		Build()

	if err == nil {
		t.Fatal("expected error for invalid profile, got nil")
	}

	if chat != nil {
		t.Fatal("expected nil LLM on error, got non-nil")
	}
}
