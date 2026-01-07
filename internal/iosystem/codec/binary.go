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

type BinaryCodec struct{}

func NewBinaryCodec() Codec {
	return &BinaryCodec{}
}

func (c *BinaryCodec) ContentType() string {
	return ContentStream
}

func (c *BinaryCodec) Extensions() []string {
	return []string{".bin", ".dat"}
}

func (c *BinaryCodec) Decode(r io.Reader) (any, error) {
	return io.ReadAll(r)
}

func (c *BinaryCodec) Encode(w io.Writer, data any) error {
	switch v := data.(type) {
	case []byte:
		_, err := w.Write(v)
		return err
	default:
		return fmt.Errorf("binary codec: expected []byte, got %T", data)
	}
}
