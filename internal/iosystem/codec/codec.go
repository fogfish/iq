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
	"io"
	"path/filepath"
)

// Content type constants
const (
	ContentStream = "application/octet-stream"
	ContentJSON   = "application/json"
	ContentJSONL  = "application/jsonl"
	ContentYAML   = "application/x-yaml"
	ContentPNG    = "image/png"
	ContentJPG    = "image/jpeg"
	ContentText   = "text/plain"
	ContentEOF    = "application/x-eof" // Signals end of stream
)

// Codec handles encoding/decoding for a specific content type.
type Codec interface {
	// ContentType returns the MIME type this codec handles
	ContentType() string

	// Decode reads from io.Reader and converts to Go value
	Decode(r io.Reader) (any, error)

	// Encode converts Go value to bytes and writes to io.Writer
	Encode(w io.Writer, data any) error

	// Extensions returns common file extensions for this type
	Extensions() []string
}

// Registry manages codec instances.
type Registry struct {
	codecs map[string]Codec
	extMap map[string]string
}

// Default is the global default registry
var Default = NewRegistry()

// NewRegistry creates a new codec registry
func NewRegistry() *Registry {
	r := &Registry{
		codecs: make(map[string]Codec),
		extMap: make(map[string]string),
	}

	// Register built-in codecs
	r.Register(NewJSONCodec(false))
	r.Register(NewYAMLCodec())
	r.Register(NewTextCodec())
	r.Register(NewBinaryCodec())
	r.Register(NewJSONLCodec())
	r.Register(NewImageCodec(ContentPNG))
	r.Register(NewImageCodec(ContentJPG))

	return r
}

// Register adds a codec to the registry
func (r *Registry) Register(c Codec) {
	contentType := c.ContentType()
	r.codecs[contentType] = c

	for _, ext := range c.Extensions() {
		r.extMap[ext] = contentType
	}
}

// Get returns codec for content type
func (r *Registry) Get(contentType string) (Codec, bool) {
	c, ok := r.codecs[contentType]
	return c, ok
}

// GetByExtension returns codec for file extension
func (r *Registry) GetByExtension(ext string) (Codec, bool) {
	contentType, ok := r.extMap[ext]
	if !ok {
		return nil, false
	}
	return r.Get(contentType)
}

// Decode decodes a document using appropriate codec
func (r *Registry) Decode(doc io.Reader, contentType string) (any, error) {
	codec, ok := r.Get(contentType)
	if !ok {
		codec = r.codecs[ContentStream]
		if codec == nil {
			return nil, fmt.Errorf("no codec found for content type %s", contentType)
		}
	}
	return codec.Decode(doc)
}

// Encode encodes data to a new document
func (r *Registry) Encode(data any, contentType string) (io.Reader, error) {
	codec, ok := r.Get(contentType)
	if !ok {
		switch data.(type) {
		case string:
			contentType = ContentText
			codec = r.codecs[contentType]
		case []byte:
			contentType = ContentStream
			codec = r.codecs[contentType]
		default:
			contentType = ContentJSON
			codec = r.codecs[contentType]
		}
	}

	if codec == nil {
		return nil, fmt.Errorf("no codec found for content type %s", contentType)
	}

	var buf bytes.Buffer
	if err := codec.Encode(&buf, data); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

// DetectContentType detects content type from file path
func (r *Registry) DetectContentType(path string) string {
	ext := filepath.Ext(path)
	contentType, ok := r.extMap[ext]
	if !ok {
		return ContentStream
	}
	return contentType
}

// GetExtension returns the preferred file extension for a content type.
// Returns empty string if content type is not registered.
func (r *Registry) GetExtension(contentType string) string {
	codec, ok := r.Get(contentType)
	if !ok {
		return ""
	}
	
	exts := codec.Extensions()
	if len(exts) == 0 {
		return ""
	}
	
	// Return the first (preferred) extension
	return exts[0]
}
