//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package codec

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Register PNG decoder
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
		// Re-encode all images to JPEG
		return reencodeToJPEG(w, v, c.contentType)
	default:
		return fmt.Errorf("image codec: expected []byte, got %T", data)
	}
}

// reencodeToJPEG decodes an image (PNG or JPEG) and re-encodes it as JPEG.
// This is mandatory for all image outputs.
func reencodeToJPEG(w io.Writer, data []byte, inputMimeType string) error {
	// Decode the input image (supports both PNG and JPEG via registered decoders)
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decode image (format: %s): %w", format, err)
	}

	// Re-encode as JPEG with high quality
	opts := &jpeg.Options{Quality: 95}
	if err := jpeg.Encode(w, img, opts); err != nil {
		return fmt.Errorf("failed to encode image as JPEG: %w", err)
	}

	return nil
}
