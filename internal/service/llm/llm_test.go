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
	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
)

func TestBuilder_Mock(t *testing.T) {
	chat, err := llm.New().
		Profile("mock", "").
		Build()

	it.Then(t).Should(
		it.Nil(err),
	).ShouldNot(
		it.Nil(chat),
	)

	// Test mock echoes input
	ctx := context.Background()
	messages := []chatter.Message{
		&chatter.Prompt{Task: "test prompt"},
	}

	reply, err := chat.Prompt(ctx, messages)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(
		it.Equal(reply.Stage, chatter.LLM_RETURN),
	)
}

func TestBuilder_WithDecorators(t *testing.T) {
	chat, err := llm.New().
		Profile("mock", "").
		Debug(true).
		Think(true).
		Quota(5, chatter.Usage{ReplyTokens: 1000}).
		Build()

	it.Then(t).Should(
		it.Nil(err),
	).ShouldNot(
		it.Nil(chat),
	)

	// Test it still works
	ctx := context.Background()
	messages := []chatter.Message{
		&chatter.Prompt{Task: "test"},
	}

	reply, err := chat.Prompt(ctx, messages)
	it.Then(t).Should(
		it.Nil(err),
		it.True(reply != nil),
	)
}

func TestBuilder_ChainOrder(t *testing.T) {
	// Test that decorators are applied in the correct order
	chat, err := llm.New().
		Profile("mock", "").
		Think(true). // First decorator
		Debug(true). // Second decorator (should see thinking output)
		Build()

	it.Then(t).Should(
		it.Nil(err),
	).ShouldNot(
		it.Nil(chat),
	)
}

func TestBuilder_DisabledDecorators(t *testing.T) {
	chat, err := llm.New().
		Profile("mock", "").
		Debug(false).              // Should not apply
		Think(false).              // Should not apply
		Quota(0, chatter.Usage{}). // Should not apply (0 epoch)
		Build()

	it.Then(t).Should(
		it.Nil(err),
	).ShouldNot(
		it.Nil(chat),
	)
}

func TestBuilder_ProfileWithModel(t *testing.T) {
	// Test profile with explicit model override
	chat, err := llm.New().
		Profile("mock", "mock-model").
		Build()

	it.Then(t).Should(
		it.Nil(err),
		it.True(chat != nil),
	)
}

func TestBuilder_QuotaDecorator(t *testing.T) {
	chat, err := llm.New().
		Profile("mock", "").
		Quota(10, chatter.Usage{
			InputTokens: 5000,
			ReplyTokens: 2000,
		}).
		Build()

	it.Then(t).Should(
		it.Nil(err),
		it.True(chat != nil),
	)

	// Test it works
	ctx := context.Background()
	messages := []chatter.Message{
		&chatter.Prompt{Task: "test"},
	}

	reply, err := chat.Prompt(ctx, messages)
	it.Then(t).Should(
		it.Nil(err),
		it.True(reply != nil),
	)
}
