# ADR 0008: Unified Codec System for Content Type Handling

**Status:** Proposed  
**Date:** 2026-01-06  
**Context:** Standardizing encoding/decoding of raw data across the application

---

## In the context of

Building a document processing pipeline that handles multiple content types (JSON, YAML, text, images, JSONL) across various components (sources, processors, sinks, blueprint runtime). Currently, encoding/decoding logic is scattered throughout the codebase with inconsistent patterns, making it difficult to add new content types without modifying multiple files.

**Current Pain Points:**

1. **Scattered encoding/decoding logic**: Each processor (Agent, Jsonify, Chunker) implements its own encode/decode methods
2. **Inconsistent type detection**: File extension-based detection in sources, MIME type-based in processors, mixed approaches in sinks
3. **Duplication**: Similar JSON marshal/unmarshal, YAML parsing, text conversion repeated across files
4. **Hard to extend**: Adding new content type (e.g., JSONL, CSV, XML, PNG) requires changes in 5+ locations
5. **No centralized registry**: Content type constants defined in multiple places, no single source of truth

**Files with encoding/decoding logic:**

```
internal/iosystem/document.go          - Content type constants, FilePath() extension logic
internal/iosystem/processor/agent.go   - decode() and encode() methods (lines 167-201)
internal/iosystem/processor/jsonify.go - JSON formatting (lines 75-115)
internal/iosystem/source/file.go       - File extension to content type (lines 57-64)
internal/iosystem/source/reader.go     - Content type initialization
internal/iosystem/sink/file.go         - Special handling for images (lines 49-52)
internal/blueprint/runtime/emitter.go  - Event encoding (lines 52-68)
internal/blueprint/compiler/context.go - anyToGist() conversion (lines 126-139)
```

## Facing Concern

**Extensibility**: Adding support for new content types (JSONL for streaming, CSV for tabular data, XML for structured documents, PNG/JPG for vision models) currently requires:
- Adding constants in `document.go`
- Updating extension detection in `file.go`
- Adding decode logic in `agent.go`
- Adding encode logic in `agent.go`
- Updating `jsonify.go` if formatting needed
- Updating `emitter.go` for blueprint runtime
- Potential changes in sink implementations

**Consistency**: Different components use different approaches:
- Agent processor: switch on `doc.Type` for decoding, switch on result type for encoding
- File source: switch on file extension
- Jsonify: checks `ContentType` field vs `Type` field
- Emitter: type switches on runtime types (`Text`, `Json`, `List`)

**Type safety**: No compile-time guarantees that all content types are handled in all locations. Easy to add constant but forget to handle it in decode/encode paths.

**Performance**: Repeated JSON marshal/unmarshal for same data. No opportunity for caching or optimization when patterns are scattered.

**Testing**: Difficult to test codec behavior comprehensively when logic distributed across multiple packages.

## We decided for

**Centralized Codec Registry** with pluggable encoder/decoder implementations following the strategy pattern. New package `internal/iosystem/codec` provides:

1. **Codec interface**: Unified encode/decode contract
2. **Registry**: Map of content type → Codec implementation
3. **Built-in codecs**: JSON, YAML, Text, Binary, JSONL, Image
4. **Helper functions**: `Decode(doc)`, `Encode(data, contentType)`, `DetectContentType(path)`

**Architecture:**

```
internal/iosystem/codec/
├── codec.go           # Codec interface, Registry, helper functions
├── json.go            # JSON codec (with optional formatting)
├── yaml.go            # YAML codec
├── text.go            # Plain text codec
├── binary.go          # Binary/octet-stream codec
├── jsonl.go           # JSON Lines codec (streaming)
├── image.go           # Image codecs (PNG, JPG)
└── codec_test.go      # Comprehensive codec tests
```

**Key Principles:**

1. **Single responsibility**: Each codec handles one content type
2. **Open/closed**: Easy to add new codecs without modifying existing code
3. **Dependency injection**: Components receive codecs via registry, not hardcoded switches
4. **Backward compatible**: Existing code continues working, migration gradual
5. **Document-centric**: Codecs work with `Document` type and `io.Reader`/`io.Writer`

## Design

### **1. Core Codec Interface**

**Location:** `internal/iosystem/codec/codec.go`

```go
package codec

import (
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

// Codec handles encoding/decoding for a specific content type.
type Codec interface {
	// ContentType returns the MIME type this codec handles (e.g., "application/json")
	ContentType() string

	// Decode reads from io.Reader and converts to Go value.
	// Returns:
	//   - string for text content
	//   - []byte for binary content
	//   - map[string]any for structured data (JSON, YAML)
	Decode(r io.Reader) (any, error)

	// Encode converts Go value to bytes and writes to io.Writer.
	// Accepts:
	//   - string: written as-is
	//   - []byte: written as-is
	//   - map[string]any: marshaled to format
	//   - struct: marshaled to format
	Encode(w io.Writer, data any) error

	// Extensions returns common file extensions for this type (e.g., [".json"])
	Extensions() []string
}

// Registry manages codec instances.
type Registry struct {
	codecs map[string]Codec      // contentType -> codec
	extMap map[string]string     // extension -> contentType
}

// Global default registry
var Default = NewRegistry()

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
	r.Register(NewImageCodec(iosystem.ContentPNG))
	r.Register(NewImageCodec(iosystem.ContentJPG))
	
	return r
}

// Register adds a codec to the registry.
func (r *Registry) Register(c Codec) {
	contentType := c.ContentType()
	r.codecs[contentType] = c
	
	// Map extensions to content type
	for _, ext := range c.Extensions() {
		r.extMap[ext] = contentType
	}
}

// Get returns codec for content type.
func (r *Registry) Get(contentType string) (Codec, bool) {
	c, ok := r.codecs[contentType]
	return c, ok
}

// GetByExtension returns codec for file extension.
func (r *Registry) GetByExtension(ext string) (Codec, bool) {
	contentType, ok := r.extMap[ext]
	if !ok {
		return nil, false
	}
	return r.Get(contentType)
}

// Decode decodes a document using appropriate codec.
func (r *Registry) Decode(doc *iosystem.Document) (any, error) {
	codec, ok := r.Get(doc.Type)
	if !ok {
		// Fallback to binary for unknown types
		codec = r.codecs[iosystem.ContentStream]
	}
	return codec.Decode(doc.Reader)
}

// Encode encodes data to a new document.
func (r *Registry) Encode(data any, contentType string) (*iosystem.Document, error) {
	codec, ok := r.Get(contentType)
	if !ok {
		// Fallback based on data type
		switch data.(type) {
		case string:
			contentType = iosystem.ContentText
			codec = r.codecs[contentType]
		case []byte:
			contentType = iosystem.ContentStream
			codec = r.codecs[contentType]
		default:
			contentType = iosystem.ContentJSON
			codec = r.codecs[contentType]
		}
	}
	
	var buf bytes.Buffer
	if err := codec.Encode(&buf, data); err != nil {
		return nil, err
	}
	
	doc := iosystem.NewDocument("", bytes.NewReader(buf.Bytes()))
	doc.Type = contentType
	return doc, nil
}

// DetectContentType detects content type from file path.
func (r *Registry) DetectContentType(path string) string {
	ext := filepath.Ext(path)
	contentType, ok := r.extMap[ext]
	if !ok {
		return iosystem.ContentText // Default fallback
	}
	return contentType
}
```

### **2. JSON Codec Implementation**

**Location:** `internal/iosystem/codec/json.go`

```go
package codec

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TylerBrock/colorjson"
	"github.com/fogfish/iq/internal/iosystem"
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
	return iosystem.ContentJSON
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
		// Use colorjson for formatted output
		f := colorjson.NewFormatter()
		f.Indent = c.indent
		formatted, err := f.Marshal(data)
		if err != nil {
			return fmt.Errorf("json color encode: %w", err)
		}
		_, err = w.Write(formatted)
		return err
	} else if c.pretty {
		// Standard pretty-print
		enc := json.NewEncoder(w)
		enc.SetIndent("", bytesIndent(c.indent))
		return enc.Encode(data)
	} else {
		// Compact JSON
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
```

### **3. YAML Codec Implementation**

**Location:** `internal/iosystem/codec/yaml.go`

```go
package codec

import (
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/goccy/go-yaml"
)

type YAMLCodec struct{}

func NewYAMLCodec() Codec {
	return &YAMLCodec{}
}

func (c *YAMLCodec) ContentType() string {
	return iosystem.ContentYAML
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
```

### **4. Text Codec Implementation**

**Location:** `internal/iosystem/codec/text.go`

```go
package codec

import (
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

type TextCodec struct{}

func NewTextCodec() Codec {
	return &TextCodec{}
}

func (c *TextCodec) ContentType() string {
	return iosystem.ContentText
}

func (c *TextCodec) Extensions() []string {
	return []string{".txt", ".md"}
}

func (c *TextCodec) Decode(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, nil // Return []byte, not string, for consistency
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
		// Convert to string representation
		_, err := w.Write([]byte(fmt.Sprintf("%v", v)))
		return err
	}
}
```

### **5. Binary Codec Implementation**

**Location:** `internal/iosystem/codec/binary.go`

```go
package codec

import (
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

type BinaryCodec struct{}

func NewBinaryCodec() Codec {
	return &BinaryCodec{}
}

func (c *BinaryCodec) ContentType() string {
	return iosystem.ContentStream
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
```

### **6. JSONL Codec Implementation**

**Location:** `internal/iosystem/codec/jsonl.go`

```go
package codec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

// ContentJSONL is the MIME type for JSON Lines
const ContentJSONL = "application/jsonl"

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
	// Expect array/slice
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
```

### **7. Image Codec Implementation**

**Location:** `internal/iosystem/codec/image.go`

```go
package codec

import (
	"fmt"
	"io"

	"github.com/fogfish/iq/internal/iosystem"
)

type ImageCodec struct {
	contentType string
	ext         string
}

func NewImageCodec(contentType string) Codec {
	var ext string
	switch contentType {
	case iosystem.ContentPNG:
		ext = ".png"
	case iosystem.ContentJPG:
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
	// Images are binary data
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
```

### **8. Migration Path**

**Phase 1: Create codec package (non-breaking)**
- Add `internal/iosystem/codec/` package with all codec implementations
- Add comprehensive tests
- Keep existing code unchanged

**Phase 2: Migrate processors (gradual)**
- Update `agent.go`: Replace `decode()`/`encode()` with `codec.Default.Decode()`/`Encode()`
- Update `jsonify.go`: Use `JSONCodec` directly
- Update `emitter.go`: Use codec for encoding

**Phase 3: Migrate sources and sinks**
- Update `file.go` source: Use `codec.DetectContentType()`
- Update `file.go` sink: Use codec for image handling
- Update other sources/sinks

**Phase 4: Cleanup**
- Remove old extension detection logic from `document.go`
- Consolidate content type constants
- Update tests to use codec package

### **9. Usage Examples**

**Adding a new content type (CSV):**

```go
// 1. Add constant
const ContentCSV = "text/csv"

// 2. Implement codec
type CSVCodec struct{}

func (c *CSVCodec) ContentType() string { return ContentCSV }
func (c *CSVCodec) Extensions() []string { return []string{".csv"} }

func (c *CSVCodec) Decode(r io.Reader) (any, error) {
	reader := csv.NewReader(r)
	return reader.ReadAll()
}

func (c *CSVCodec) Encode(w io.Writer, data any) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()
	// ... write rows
}

// 3. Register in codec.go init()
r.Register(NewCSVCodec())
```

**Using codec in custom processor:**

```go
type MyProcessor struct {
	registry *codec.Registry
}

func (p *MyProcessor) Process(ctx context.Context, docs []*Document) ([]*Document, error) {
	for _, doc := range docs {
		// Decode document
		data, err := p.registry.Decode(doc)
		if err != nil {
			return nil, err
		}
		
		// Transform data
		transformed := p.transform(data)
		
		// Encode back to same format
		outDoc, err := p.registry.Encode(transformed, doc.Type)
		if err != nil {
			return nil, err
		}
		
		results = append(results, outDoc)
	}
	return results, nil
}
```

## Neglected

**Alternative approaches rejected:**

1. **Type assertion everywhere**: Continue with scattered switch statements
   - **Rejected**: Violates DRY, hard to maintain, error-prone

2. **Interface on Document**: Add `doc.Decode()` / `doc.Encode()` methods
   - **Rejected**: Tight coupling, Document becomes too complex, hard to test codecs independently

3. **Codec as Document field**: `doc.Codec` field set at creation
   - **Rejected**: Bloats Document struct, unclear lifecycle, complicates serialization

4. **Separate encoder/decoder interfaces**: Split encode and decode
   - **Rejected**: Most codecs need both, extra complexity without benefit

5. **Content negotiation**: Automatic format conversion
   - **Rejected**: Over-engineering, users should be explicit about formats

6. **Streaming codecs**: Support for non-buffering decode
   - **Rejected**: Current architecture buffers documents, streaming adds complexity without benefit

## To Achieve

**Extensibility**: Add new content type by implementing Codec interface and registering. No changes to existing processors, sources, or sinks.

**Consistency**: All encoding/decoding goes through same interface. Guaranteed that all codecs handle content types uniformly.

**Testability**: Codec implementations testable in isolation. Mock registry for testing processors.

**Performance**: Opportunity for optimization (caching decoded documents, lazy decoding, format-specific optimizations) in single location.

**Maintainability**: Content type handling logic in one package. Adding support for new format requires 1 file, not 5+ files.

**Backward compatibility**: Existing code continues working. Migration can be gradual, component by component.

**Type safety**: Compile-time guarantee that codec handles its declared content type. Registry ensures no duplicate registrations.

**Documentation**: Single place to document supported formats, their capabilities, and limitations.

## Accepting

**Additional abstraction layer**: Introduces Codec interface and Registry. Adds indirection between raw data and Go types. Trade-off for maintainability.

**Memory overhead**: Registry holds codec instances. Minimal impact (~10 codecs × ~100 bytes each).

**Migration effort**: Requires updating multiple files to use codec system. Gradual migration reduces risk but increases transition period.

**Breaking API changes**: If we change Document structure or content type constants during migration. Can mitigate with deprecation warnings.

**Testing burden**: Need comprehensive codec tests. Each codec needs tests for encode/decode paths, edge cases, error handling.

**Dependency on registry**: Components depend on codec.Default global. Could complicate testing if not careful. Mitigate by allowing registry injection.

## Implementation Checklist

- [ ] Create `internal/iosystem/codec/` package structure
- [ ] Implement core `Codec` interface and `Registry`
- [ ] Implement built-in codecs (JSON, YAML, Text, Binary)
- [ ] Add JSONL codec for streaming formats
- [ ] Add Image codecs (PNG, JPG)
- [ ] Write comprehensive codec tests
- [ ] Update `agent.go` to use codec system
- [ ] Update `jsonify.go` to use JSONCodec
- [ ] Update `emitter.go` to use codec
- [ ] Update `file.go` source to use DetectContentType
- [ ] Update `file.go` sink to use codec
- [ ] Add codec usage examples to documentation
- [ ] Update ADR index with this proposal
- [ ] Plan gradual migration of remaining components

---

**Related ADRs:**
- ADR 0002: IO System (Document architecture)
- ADR 0007: Array Input (Output formatting needs)

**References:**
- Strategy Pattern: https://en.wikipedia.org/wiki/Strategy_pattern
- MIME Types: https://www.iana.org/assignments/media-types/media-types.xhtml
