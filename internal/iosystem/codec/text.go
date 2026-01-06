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
)

type TextCodec struct{}

func NewTextCodec() Codec {
	return &TextCodec{}
}

func (c *TextCodec) ContentType() string {
	return ContentText
}

func (c *TextCodec) Extensions() []string {
	return []string{".txt", ".md"}
}

func (c *TextCodec) Decode(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *TextCodec) Encode(w io.Writer, data any) error {
	switch v := data.(type) {
	case string:
		_, err := w.Write([]byte(v))
		return err
	case []byte:
		_, err := w.Write(v)
		return err
	default:
		_, err := w.Write([]byte(fmt.Sprintf("%v", v)))
		return err
	}
}
