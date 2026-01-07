//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package codec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

type JSONLCodec struct{}

func NewJSONLCodec() Codec {
	return &JSONLCodec{}
}

func (c *JSONLCodec) ContentType() string {
	return ContentJSONL
}

func (c *JSONLCodec) Extensions() []string {
	return []string{".jsonl", ".ndjson"}
}

func (c *JSONLCodec) Decode(r io.Reader) (any, error) {
	var results []any
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var item any
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("jsonl decode line: %w", err)
		}
		results = append(results, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("jsonl scan: %w", err)
	}

	return results, nil
}

func (c *JSONLCodec) Encode(w io.Writer, data any) error {
	switch v := data.(type) {
	case []any:
		for _, item := range v {
			if err := json.NewEncoder(w).Encode(item); err != nil {
				return fmt.Errorf("jsonl encode: %w", err)
			}
		}
	case []map[string]any:
		for _, item := range v {
			if err := json.NewEncoder(w).Encode(item); err != nil {
				return fmt.Errorf("jsonl encode: %w", err)
			}
		}
	default:
		return fmt.Errorf("jsonl codec: expected array, got %T", data)
	}
	return nil
}
