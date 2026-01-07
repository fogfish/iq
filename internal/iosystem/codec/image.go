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

type ImageCodec struct {
	contentType string
	ext         string
}

func NewImageCodec(contentType string) Codec {
	var ext string
	switch contentType {
	case ContentPNG:
		ext = ".png"
	case ContentJPG:
		ext = ".jpg"
	default:
		ext = ".img"
	}

	return &ImageCodec{
		contentType: contentType,
		ext:         ext,
	}
}

func (c *ImageCodec) ContentType() string {
	return c.contentType
}

func (c *ImageCodec) Extensions() []string {
	return []string{c.ext}
}

func (c *ImageCodec) Decode(r io.Reader) (any, error) {
	return io.ReadAll(r)
}

func (c *ImageCodec) Encode(w io.Writer, data any) error {
	switch v := data.(type) {
	case []byte:
		_, err := w.Write(v)
		return err
	default:
		return fmt.Errorf("image codec: expected []byte, got %T", data)
	}
}
