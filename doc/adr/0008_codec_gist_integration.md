# Codec System Integration with Gist/Event Architecture

**Context:** How the unified codec system (ADR 0008) integrates with the dual-document architecture

---

## The Dual Document System

Your application maintains two parallel representations:

### 1. **iosystem.Document** - External I/O Layer
```go
type Document struct {
    Key      Key           // File path identity
    Type     string        // MIME type (application/json, text/plain, etc.)
    Reader   io.Reader     // Raw bytes stream
    Metadata Metadata      // File metadata
}
```
- **Purpose**: External file I/O (read from disk/S3, write to disk/S3)
- **Format**: Streaming bytes with MIME type
- **Layer**: Sources, Sinks, Processors (I/O boundary)

### 2. **runtime.Event** with **Gist** - Internal Processing Layer
```go
type Event struct {
    Key      iosystem.Key
    Document Gist              // Original input
    Current  Gist              // Current processing state
    Steps    map[string]Gist   // Named step outputs
}

type Gist interface{ HKT1(Gist) }  // Higher-kinded type marker

// Concrete implementations:
type Text string              // Plain text
type Json map[string]any     // JSON object
type List []any               // JSON array
```
- **Purpose**: Workflow processing (blueprint runtime, LLM interactions)
- **Format**: Typed Go values (Text, Json, List)
- **Layer**: Blueprint compiler, workflow steps, LLM prompts

---

## Current Conversion Points

### 1. **Document → Event (Agent.decode)**
Location: `internal/iosystem/processor/agent.go:167-177`

```go
func (p *Agent) decode(doc *iosystem.Document) (any, error) {
    content, err := io.ReadAll(doc.Reader)
    
    switch doc.Type {
    case iosystem.ContentJSON:
        var input map[string]any
        json.Unmarshal(content, &input)
        return input, nil  // → becomes Json Gist
    case iosystem.ContentYAML:
        var input map[string]any
        yaml.Unmarshal(content, &input)
        return input, nil  // → becomes Json Gist
    default:
        return content, nil  // []byte → becomes Text Gist
    }
}
```

### 2. **Event → Document (Agent.encode)**
Location: `internal/iosystem/processor/agent.go:179-201`

```go
func (p *Agent) encode(reply any) (*iosystem.Document, error) {
    switch v := reply.(type) {
    case string:
        return &iosystem.Document{
            Reader: bytes.NewReader([]byte(v)),
            Type:   iosystem.ContentText,
        }, nil
    case []byte:
        return &iosystem.Document{
            Reader: bytes.NewReader(v),
            Type:   iosystem.ContentStream,
        }, nil
    default:  // map[string]any, struct, etc.
        data, err := json.Marshal(v)
        return &iosystem.Document{
            Reader: bytes.NewReader(data),
            Type:   iosystem.ContentJSON,
        }, nil
    }
}
```

### 3. **any → Gist (compiler context)**
Location: `internal/blueprint/compiler/context.go:129-142`

```go
func anyToGist(in any) runtime.Gist {
    switch v := in.(type) {
    case string:        return runtime.Text(v)
    case []byte:        return runtime.Text(v)
    case map[string]any: return runtime.Json(v)
    case runtime.Text:  return v
    case runtime.Json:  return v
    default:
        panic(fmt.Sprintf("unsupported runtime.Gist type: %T", v))
    }
}
```

### 4. **Gist → bytes (runtime emitter)**
Location: `internal/blueprint/runtime/emitter.go:52-68`

```go
func (e *Emitter) Prompt(...) (Event, error) {
    // ... get Event from blueprint
    
    var buf bytes.Buffer
    switch v := val.Current.(type) {
    case Text:
        buf.Write([]byte(v))
    case Json, List:
        enc := json.NewEncoder(&buf)
        enc.SetIndent("", "  ")
        enc.Encode(v)
    }
    
    e.snapshot.Put(ctx, key, &buf)
}
```

---

## How Codec System Integrates

### **Option A: Codec at I/O Boundary (RECOMMENDED)**

Keep Gist conversion separate, use codecs only for Document ↔ bytes:

```
┌─────────────────────────────────────────────────────────────┐
│                        Application Flow                       │
└─────────────────────────────────────────────────────────────┘

Source (File/S3)
    │
    ├─> Document {Type: ContentJSON, Reader: io.Reader}
    │
    v
[CODEC: Decode Document → any]  ← Uses codec.Registry
    │
    ├─> any (string | []byte | map[string]any)
    │
    v
[anyToGist: any → Gist]        ← Simple type assertion
    │
    ├─> Gist (Text | Json | List)
    │
    v
Blueprint Runtime (Event processing)
    │
    ├─> Gist transformations
    │
    v
[Gist → any]                   ← Simple conversion
    │
    ├─> any (string | map[string]any)
    │
    v
[CODEC: Encode any → Document] ← Uses codec.Registry
    │
    ├─> Document {Type: ContentJSON, Reader: io.Reader}
    │
    v
Sink (File/S3)
```

**Implementation:**

```go
// internal/iosystem/processor/agent.go

func (p *Agent) decode(doc *iosystem.Document) (any, error) {
    // Use codec registry to decode Document → any
    return codec.Default.Decode(doc)
}

func (p *Agent) encode(reply any) (*iosystem.Document, error) {
    // Determine content type from reply type
    var contentType string
    switch reply.(type) {
    case string, []byte:
        contentType = iosystem.ContentText
    default:
        contentType = iosystem.ContentJSON
    }
    
    // Use codec registry to encode any → Document
    return codec.Default.Encode(reply, contentType)
}
```

**Pros:**
- ✅ Clean separation: Codecs handle I/O, Gist handles logic
- ✅ No changes to Gist types (Text, Json, List remain simple)
- ✅ No changes to Event/workflow runtime
- ✅ Codec system focused on one responsibility (serialization)
- ✅ Easy to add new I/O formats (JSONL, CSV, XML) without touching Gist

**Cons:**
- Still need `anyToGist()` function (but it's trivial type assertion)

---

### **Option B: Gist-Aware Codecs**

Make codecs understand Gist types directly:

```go
// internal/iosystem/codec/gist.go

type GistCodec struct {
    baseCodec Codec
}

func (c *GistCodec) DecodeToGist(doc *iosystem.Document) (runtime.Gist, error) {
    any, err := c.baseCodec.Decode(doc.Reader)
    if err != nil {
        return nil, err
    }
    return runtime.ToGist(any)
}

func (c *GistCodec) EncodeFromGist(gist runtime.Gist, contentType string) (*iosystem.Document, error) {
    var data any
    switch g := gist.(type) {
    case runtime.Text:
        data = string(g)
    case runtime.Json:
        data = map[string]any(g)
    case runtime.List:
        data = []any(g)
    }
    
    return codec.Default.Encode(data, contentType)
}
```

**Pros:**
- ✅ Direct Document ↔ Gist conversion
- ✅ One-step conversion

**Cons:**
- ❌ Couples codec system to Gist (violates separation of concerns)
- ❌ Codec package now depends on runtime package
- ❌ Harder to test codecs independently
- ❌ Gist becomes I/O concern (it should be pure logic)

---

## Recommended Approach

**Use Option A: Keep codec at I/O boundary**

### Updated Agent Processor:

```go
// internal/iosystem/processor/agent.go

import "github.com/fogfish/iq/internal/iosystem/codec"

type Agent struct {
    w       Worker
    config  AgentConfig
    codecs  *codec.Registry  // NEW: codec registry
}

func NewAgent(w Worker, config *AgentConfig) *Agent {
    return &Agent{
        w:      w,
        config: *config,
        codecs: codec.Default,  // Use global registry
    }
}

func (p *Agent) decode(doc *iosystem.Document) (any, error) {
    // Step 1: Codec decodes Document → any
    return p.codecs.Decode(doc)
}

func (p *Agent) encode(reply any) (*iosystem.Document, error) {
    // Step 1: Determine content type
    var contentType string
    switch reply.(type) {
    case string, []byte:
        contentType = iosystem.ContentText
    default:
        contentType = iosystem.ContentJSON
    }
    
    // Step 2: Codec encodes any → Document
    return p.codecs.Encode(reply, contentType)
}

func (p *Agent) Process(ctx context.Context, docs []*iosystem.Document) ([]*iosystem.Document, error) {
    // ... existing code ...
    
    // Decode documents using codec
    items := make([]any, 0, len(docs))
    for _, doc := range docs {
        content, err := p.decode(doc)  // Uses codec
        if err != nil {
            return nil, err
        }
        items = append(items, content)
    }
    
    // anyToGist conversion happens in Worker.Prompt()
    // (inside blueprint/compiler where it belongs)
    input := items
    if len(items) == 1 {
        input = items[0]
    }
    
    result, err := p.w.Prompt(ctx, key, input, p.config.Options...)
    
    // Encode result using codec
    reply, err := p.encode(result)  // Uses codec
    // ...
}
```

### Flow with Codecs:

```
File (file.json)
    ↓
Source reads → Document{Type: ContentJSON, Reader: [raw bytes]}
    ↓
Agent.decode()
    ↓
codec.JSONCodec.Decode() → map[string]any
    ↓
Worker.Prompt() internally calls:
    compiler.anyToGist() → runtime.Json
    ↓
    Blueprint processes Event{Current: Json{...}}
    ↓
    Returns Event{Current: Json{...}}
    ↓
    Converts back to map[string]any
    ↓
Agent.encode()
    ↓
codec.JSONCodec.Encode() → Document{Type: ContentJSON, Reader: [json bytes]}
    ↓
Sink writes → File (output.json)
```

---

## Migration Strategy

### Phase 1: Add codec to Agent processor
```go
// internal/iosystem/processor/agent.go
// Replace decode()/encode() to use codec.Default
```

### Phase 2: Add codec to Emitter
```go
// internal/blueprint/runtime/emitter.go
// Replace switch statement with codec.Encode()
```

### Phase 3: Keep anyToGist() as-is
```go
// internal/blueprint/compiler/context.go
// No changes needed - this is pure type conversion
```

### Phase 4: Future extension
When you add new I/O formats (CSV, XML, Parquet):
- Add codec implementation (e.g., CSVCodec)
- Register in codec.Default
- No changes to Gist, Event, or workflow logic
- CSV automatically maps to Json Gist ([]map[string]any)

---

## Summary

**Answer to your question:**

> Shall I implement Event To Document and back and then use this proposal as is?

**No, keep them separate:**

1. **Codec system**: Handles `Document ↔ any` (I/O boundary)
2. **Gist conversion**: Handles `any ↔ Gist` (logic boundary)

**The codec system works at a lower level than Gist:**

```
Document (I/O layer - files, bytes, MIME types)
    ↕ [CODEC: serialization/deserialization]
any (Go values - string, []byte, map[string]any)
    ↕ [anyToGist: type assertion]
Gist (Logic layer - Text, Json, List)
    ↕ [Workflow processing]
Event (Blueprint runtime)
```

**Key insight:** 
- Codec doesn't need to know about Gist
- Gist doesn't need to know about codecs
- Agent processor sits at the boundary and uses both:
  - Uses codec to cross I/O boundary
  - Uses anyToGist (existing) to cross logic boundary

This keeps concerns separated and makes both systems independently testable and extensible.
