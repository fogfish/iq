//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package reader_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/fogfish/iq/internal/reader"
	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/spf13/viper"
)

func TestReaderNone(t *testing.T) {
	b := &bytes.Buffer{}
	r := strings.NewReader("Hello World.\n")
	s := reader.New(reader.STRATEGY_NONE, "", 0, r)

	req := viper.New()
	req.SetConfigType("yaml")

	err := reader.Process(context.Background(), mock{}, req, s, b)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(b.String(), "Hello World.\n"),
	)
}

func TestReaderSentence(t *testing.T) {
	b := &bytes.Buffer{}
	r := strings.NewReader("Hello. World.\n")
	s := reader.New(reader.STRATEGY_SENTENCE, "", 0, r)

	req := viper.New()
	req.SetConfigType("yaml")

	err := reader.Process(context.Background(), mock{}, req, s, b)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(b.String(), "Hello.\nWorld.\n"),
	)
}

func TestReaderParagraph(t *testing.T) {
	b := &bytes.Buffer{}
	r := strings.NewReader("Hello.\nWorld.\n")
	s := reader.New(reader.STRATEGY_PARAGRAPH, ".\n", 0, r)

	req := viper.New()
	req.SetConfigType("yaml")

	err := reader.Process(context.Background(), mock{}, req, s, b)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(b.String(), "Hello\nWorld\n"),
	)
}

func TestReaderChunk(t *testing.T) {
	b := &bytes.Buffer{}
	r := strings.NewReader("Hello. World.")
	s := reader.New(reader.STRATEGY_CHUNK, "", 2, r)

	req := viper.New()
	req.SetConfigType("yaml")

	err := reader.Process(context.Background(), mock{}, req, s, b)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(b.String(), "Hello.\nWorld.\n"),
	)
}

type mock struct{}

func (mock) PromptOnce(ctx context.Context, req *viper.Viper, opts ...chatter.Opt) ([]byte, error) {
	blob := req.GetString(spec.YAML_BLOB)
	return []byte(blob), nil
}
