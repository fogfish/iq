//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package codec

import (
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

type YAMLCodec struct{}

func NewYAMLCodec() Codec {
	return &YAMLCodec{}
}

func (c *YAMLCodec) ContentType() string {
	return ContentYAML
}

func (c *YAMLCodec) Extensions() []string {
	return []string{".yaml", ".yml"}
}

func (c *YAMLCodec) Decode(r io.Reader) (any, error) {
	var result map[string]any
	if err := yaml.NewDecoder(r).Decode(&result); err != nil {
		return nil, fmt.Errorf("yaml decode: %w", err)
	}
	return result, nil
}

func (c *YAMLCodec) Encode(w io.Writer, data any) error {
	return yaml.NewEncoder(w).Encode(data)
}
