//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package codec

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TylerBrock/colorjson"
)

type JSONCodec struct {
	pretty bool
	color  bool
	indent int
}

func NewJSONCodec(pretty bool) Codec {
	return &JSONCodec{
		pretty: pretty,
		color:  false,
		indent: 2,
	}
}

func NewPrettyJSONCodec(color bool, indent int) Codec {
	return &JSONCodec{
		pretty: true,
		color:  color,
		indent: indent,
	}
}

func (c *JSONCodec) ContentType() string {
	return ContentJSON
}

func (c *JSONCodec) Extensions() []string {
	return []string{".json"}
}

func (c *JSONCodec) Decode(r io.Reader) (any, error) {
	var result map[string]any
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return result, nil
}

func (c *JSONCodec) Encode(w io.Writer, data any) error {
	if c.pretty && c.color {
		f := colorjson.NewFormatter()
		f.Indent = c.indent
		formatted, err := f.Marshal(data)
		if err != nil {
			return fmt.Errorf("json color encode: %w", err)
		}
		_, err = w.Write(formatted)
		return err
	} else if c.pretty {
		enc := json.NewEncoder(w)
		enc.SetIndent("", bytesIndent(c.indent))
		return enc.Encode(data)
	} else {
		return json.NewEncoder(w).Encode(data)
	}
}

func bytesIndent(n int) string {
	indent := make([]byte, n)
	for i := range indent {
		indent[i] = ' '
	}
	return string(indent)
}
