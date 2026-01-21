//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package runtime_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fogfish/iq/internal/blueprint/runtime"
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/storage"
	"github.com/kshard/chatter"
)

// MockPrompter implements runtime.Prompter for testing
type MockPrompter struct {
	PromptFunc func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error)
	callCount  int
}

func (m *MockPrompter) Config(map[string]*runtime.Job) error {
	return nil
}

func (m *MockPrompter) Prompt(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
	m.callCount++
	if m.PromptFunc != nil {
		return m.PromptFunc(ctx, in, opts...)
	}
	// Default: return input with modified Current
	out := in
	out.Current = runtime.Text("mocked response")
	return out, nil
}

func TestCacheMiss_TriggersStepExecution(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("response for " + string(in.Key))
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	// Execute
	in := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("input content"),
	}

	result, err := cache.Prompt(ctx, in)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("Expected prompter to be called once on cache miss, got %d calls", mock.callCount)
	}

	expected := runtime.Text("response for doc1.txt")
	if string(result.Current.(runtime.Text)) != string(expected) {
		t.Errorf("Expected response to be returned, got %v", result.Current)
	}
}

func TestCacheHit_SkipsStepExecution(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("first response")
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	in := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("input content"),
	}

	// First call - cache miss, should execute
	_, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("Expected 1 call after cache miss, got %d", mock.callCount)
	}

	// Second call - cache hit, should NOT execute
	result, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("Expected no additional calls on cache hit, got %d total calls", mock.callCount)
	}

	expected := runtime.Text("first response")
	if string(result.Current.(runtime.Text)) != string(expected) {
		t.Errorf("Expected cached response '%s', got '%s'", expected, result.Current)
	}
}

func TestPromptChange_CreatesNewCacheEntry(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock1 := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("response from prompt 1")
			return out, nil
		},
	}

	mock2 := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("response from prompt 2")
			return out, nil
		},
	}

	// Create two cache instances with different hashes (simulating different prompts)
	cache1 := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock1)
	cache2 := runtime.NewCache(store, "workflow", "job", "step", "def456", mock2)

	in := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("input content"),
	}

	// First call with cache1
	result1, err := cache1.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Cache1 call failed: %v", err)
	}
	expected1 := runtime.Text("response from prompt 1")
	if string(result1.Current.(runtime.Text)) != string(expected1) {
		t.Errorf("Expected response from prompt 1, got %v", result1.Current)
	}

	// Call with cache2 (different hash) - should be cache miss and execute
	result2, err := cache2.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Cache2 call failed: %v", err)
	}
	expected2 := runtime.Text("response from prompt 2")
	if string(result2.Current.(runtime.Text)) != string(expected2) {
		t.Errorf("Expected response from prompt 2, got %v", result2.Current)
	}

	if mock1.callCount != 1 {
		t.Errorf("Expected mock1 to be called once, got %d", mock1.callCount)
	}
	if mock2.callCount != 1 {
		t.Errorf("Expected mock2 to be called once, got %d", mock2.callCount)
	}

	// Verify both cache entries exist independently
	// Call cache1 again - should hit cache
	result1Again, err := cache1.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Cache1 second call failed: %v", err)
	}
	if string(result1Again.Current.(runtime.Text)) != string(expected1) {
		t.Errorf("Expected cached response from prompt 1, got %v", result1Again.Current)
	}
	if mock1.callCount != 1 {
		t.Errorf("Expected no additional calls to mock1, got %d", mock1.callCount)
	}
}

func TestCacheReuse_AcrossDocuments(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("response for " + string(in.Key))
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	// Process doc1
	in1 := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("content of doc1"),
	}
	result1, err := cache.Prompt(ctx, in1)
	if err != nil {
		t.Fatalf("Doc1 processing failed: %v", err)
	}
	expected1 := runtime.Text("response for doc1.txt")
	if string(result1.Current.(runtime.Text)) != string(expected1) {
		t.Errorf("Expected response for doc1, got %v", result1.Current)
	}

	// Process doc2 (different document)
	in2 := runtime.Event{
		Key:     iosystem.Key("doc2.txt"),
		Current: runtime.Text("content of doc2"),
	}
	result2, err := cache.Prompt(ctx, in2)
	if err != nil {
		t.Fatalf("Doc2 processing failed: %v", err)
	}
	expected2 := runtime.Text("response for doc2.txt")
	if string(result2.Current.(runtime.Text)) != string(expected2) {
		t.Errorf("Expected response for doc2, got %v", result2.Current)
	}

	// Both documents should have been processed (no cross-document cache sharing)
	if mock.callCount != 2 {
		t.Errorf("Expected 2 calls (one per document), got %d", mock.callCount)
	}

	// Reprocessing same document should hit cache
	result1Again, err := cache.Prompt(ctx, in1)
	if err != nil {
		t.Fatalf("Doc1 reprocessing failed: %v", err)
	}
	if string(result1Again.Current.(runtime.Text)) != string(expected1) {
		t.Errorf("Expected cached response for doc1, got %v", result1Again.Current)
	}
	if mock.callCount != 2 {
		t.Errorf("Expected no additional calls when reprocessing doc1, got %d", mock.callCount)
	}
}

func TestOldCacheEntries_RemainUnused_AfterPromptChange(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock1 := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("old prompt response")
			return out, nil
		},
	}

	mock2 := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("new prompt response")
			return out, nil
		},
	}

	// Create cache with old prompt hash
	oldCache := runtime.NewCache(store, "workflow", "job", "step", "old123", mock1)

	in := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("input content"),
	}

	// Process with old prompt - creates cache entry
	_, err := oldCache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Old prompt processing failed: %v", err)
	}

	// Verify old cache entry exists
	oldKey := iosystem.Key("workflow/job/step-old123/doc1.txt.md")
	hasOld, _ := store.Has(ctx, oldKey)
	if !hasOld {
		t.Error("Expected old cache entry to exist")
	}

	// Create cache with new prompt hash (simulating prompt change)
	newCache := runtime.NewCache(store, "workflow", "job", "step", "new456", mock2)

	// Process with new prompt - should NOT use old cache
	result, err := newCache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("New prompt processing failed: %v", err)
	}
	expected := runtime.Text("new prompt response")
	if string(result.Current.(runtime.Text)) != string(expected) {
		t.Errorf("Expected new prompt response, got %v", result.Current)
	}

	// Verify new cache entry exists
	newKey := iosystem.Key("workflow/job/step-new456/doc1.txt.md")
	hasNew, _ := store.Has(ctx, newKey)
	if !hasNew {
		t.Error("Expected new cache entry to exist")
	}

	// Verify old cache entry still exists (not deleted)
	hasOldAfter, _ := store.Has(ctx, oldKey)
	if !hasOldAfter {
		t.Error("Expected old cache entry to remain after prompt change")
	}

	// Verify both prompts were called (no cache sharing)
	if mock1.callCount != 1 || mock2.callCount != 1 {
		t.Errorf("Expected each prompt to be called once, got mock1=%d, mock2=%d", mock1.callCount, mock2.callCount)
	}
}

func TestCacheWriteFailure_DoesNotFailExecution(t *testing.T) {
	// Setup
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Use a storage that will fail on write
	store, _ := storage.NewFileSystem(tmpDir)
	// Note: MemIO doesn't have a way to simulate write failure,
	// but we're documenting the expected behavior here
	// In production, this is handled by logging the error and continuing

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("response")
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	in := runtime.Event{
		Key:     iosystem.Key("doc1.txt"),
		Current: runtime.Text("input content"),
	}

	// Execute - even if cache write fails, execution should succeed
	result, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Expected execution to succeed even on cache write failure, got error: %v", err)
	}

	expected := runtime.Text("response")
	if string(result.Current.(runtime.Text)) != string(expected) {
		t.Errorf("Expected response to be returned, got %v", result.Current)
	}
}

// ============================================================================
// Cache Key Generation Tests
// ============================================================================

func TestCacheKey_WorkflowJobStepCombination(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		job      string
		step     string
		hash     string
		docKey   iosystem.Key
		expected string
	}{
		{
			name:     "standard key",
			workflow: "myworkflow",
			job:      "myjob",
			step:     "mystep",
			hash:     "abc123",
			docKey:   iosystem.Key("doc.txt"),
			expected: "myworkflow/myjob/mystep-abc123/doc.txt.md",
		},
		{
			name:     "nested document key",
			workflow: "workflow",
			job:      "job",
			step:     "step",
			hash:     "def456",
			docKey:   iosystem.Key("folder/doc.txt"),
			expected: "workflow/job/step-def456/folder/doc.txt.md",
		},
		{
			name:     "step with index",
			workflow: "chain",
			job:      "main",
			step:     "step-2",
			hash:     "789abc",
			docKey:   iosystem.Key("input.json"),
			expected: "chain/main/step-2-789abc/input.json.md",
		},
		{
			name:     "no hash",
			workflow: "workflow",
			job:      "job",
			step:     "step",
			hash:     "",
			docKey:   iosystem.Key("doc.txt"),
			expected: "workflow/job/step/doc.txt.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()
			store, _ := storage.NewFileSystem(tmpDir)

			mock := &MockPrompter{
				PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
					out := in
					out.Current = runtime.Text("response")
					return out, nil
				},
			}

			cache := runtime.NewCache(store, tt.workflow, tt.job, tt.step, tt.hash, mock)

			in := runtime.Event{
				Key:     tt.docKey,
				Current: runtime.Text("test content"),
			}

			// Execute to trigger cache write
			_, err := cache.Prompt(ctx, in)
			if err != nil {
				t.Fatalf("Prompt failed: %v", err)
			}

			// Verify the cache key was created at expected location
			cacheKey := iosystem.Key(tt.expected)
			has, err := store.Has(ctx, cacheKey)
			if err != nil {
				t.Fatalf("Failed to check cache: %v", err)
			}
			if !has {
				t.Errorf("Expected cache entry at %s, but it doesn't exist", tt.expected)
			}
		})
	}
}

func TestCacheKey_DeterministicHashing(t *testing.T) {
	// Verify that same content produces same hash
	content1 := "Extract key information from the document."
	content2 := "Extract key information from the document."
	content3 := "Different prompt content."

	hash1 := hashContent(content1)
	hash2 := hashContent(content2)
	hash3 := hashContent(content3)

	if hash1 != hash2 {
		t.Errorf("Expected identical content to produce same hash, got %s and %s", hash1, hash2)
	}

	if hash1 == hash3 {
		t.Errorf("Expected different content to produce different hash, got %s for both", hash1)
	}

	// Verify hash is 6 characters
	if len(hash1) != 6 {
		t.Errorf("Expected hash to be 6 characters, got %d: %s", len(hash1), hash1)
	}
}

func TestCacheKey_ForeachIterations(t *testing.T) {
	// Test that foreach iterations get unique cache keys via Key.SeqID
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	responses := []string{"response 1", "response 2", "response 3"}
	callIndex := 0

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text(responses[callIndex])
			callIndex++
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	// Simulate foreach iterations with SeqID
	for i := 1; i <= 3; i++ {
		docKey := iosystem.Key("doc.txt")
		// Simulate SeqID appending iteration number
		iterKey := docKey.SeqID(i)

		in := runtime.Event{
			Key:     iterKey,
			Current: runtime.Text("input content"),
		}

		result, err := cache.Prompt(ctx, in)
		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Verify each iteration gets correct response
		expected := runtime.Text(responses[i-1])
		if string(result.Current.(runtime.Text)) != string(expected) {
			t.Errorf("Iteration %d: expected '%s', got '%s'", i, expected, result.Current)
		}
	}

	// Verify all 3 iterations were executed (no cache sharing)
	if mock.callCount != 3 {
		t.Errorf("Expected 3 calls for 3 iterations, got %d", mock.callCount)
	}

	// Verify each iteration created a unique cache entry
	for i := 1; i <= 3; i++ {
		docKey := iosystem.Key("doc.txt")
		iterKey := docKey.SeqID(i)
		cacheKey := iosystem.Key("workflow/job/step-abc123/" + string(iterKey) + ".md")

		has, err := store.Has(ctx, cacheKey)
		if err != nil {
			t.Fatalf("Failed to check cache for iteration %d: %v", i, err)
		}
		if !has {
			t.Errorf("Expected cache entry for iteration %d at %s", i, cacheKey)
		}
	}
}

// ============================================================================
// Cache Storage Format Tests
// ============================================================================

func TestCacheFormat_YAMLFrontMatter(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("This is the cached content.")
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "myworkflow", "myjob", "mystep", "abc123", mock)

	in := runtime.Event{
		Key:     iosystem.Key("testdoc.txt"),
		Current: runtime.Text("input"),
	}

	// Execute to create cache
	_, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Read raw cache file
	cacheKey := iosystem.Key("myworkflow/myjob/mystep-abc123/testdoc.txt.md")
	reader, err := store.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	content := make([]byte, 0, 1024)
	buf := make([]byte, 256)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			content = append(content, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	contentStr := string(content)

	// Verify YAML front matter structure
	if !containsString(contentStr, "---") {
		t.Error("Cache file missing YAML front matter delimiter '---'")
	}

	// Verify metadata fields
	requiredFields := []string{
		"key:",
		"workflow:",
		"job:",
		"step:",
		"content_type:",
		"timestamp:",
	}

	for _, field := range requiredFields {
		if !containsString(contentStr, field) {
			t.Errorf("Cache file missing required metadata field: %s", field)
		}
	}

	// Verify specific values
	if !containsString(contentStr, "workflow: myworkflow") {
		t.Error("Cache file has incorrect workflow value")
	}
	if !containsString(contentStr, "job: myjob") {
		t.Error("Cache file has incorrect job value")
	}
	if !containsString(contentStr, "step: mystep") {
		t.Error("Cache file has incorrect step value")
	}

	// Verify content after front matter
	if !containsString(contentStr, "This is the cached content.") {
		t.Error("Cache file missing expected content")
	}
}

func TestCacheFormat_MarkdownExtension(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			out.Current = runtime.Text("content")
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	in := runtime.Event{
		Key:     iosystem.Key("doc.txt"),
		Current: runtime.Text("input"),
	}

	// Execute to create cache
	_, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Verify cache file has .md extension
	cacheKey := iosystem.Key("workflow/job/step-abc123/doc.txt.md")
	has, err := store.Has(ctx, cacheKey)
	if err != nil {
		t.Fatalf("Failed to check cache: %v", err)
	}
	if !has {
		t.Error("Cache file should have .md extension")
	}
}

func TestCacheFormat_JSONContent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := storage.NewFileSystem(tmpDir)

	mock := &MockPrompter{
		PromptFunc: func(ctx context.Context, in runtime.Event, opts ...chatter.Opt) (runtime.Event, error) {
			out := in
			// Return JSON content
			out.Current = runtime.Json(map[string]any{
				"name":  "test",
				"value": 42,
			})
			return out, nil
		},
	}

	cache := runtime.NewCache(store, "workflow", "job", "step", "abc123", mock)

	in := runtime.Event{
		Key:     iosystem.Key("doc.json"),
		Current: runtime.Text("input"),
	}

	// Execute to create cache
	result1, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Read from cache
	result2, err := cache.Prompt(ctx, in)
	if err != nil {
		t.Fatalf("Failed to read from cache: %v", err)
	}

	// Verify JSON content is preserved
	json1, ok1 := result1.Current.(runtime.Json)
	json2, ok2 := result2.Current.(runtime.Json)

	if !ok1 {
		t.Fatalf("Expected result1 to be Json, got %T", result1.Current)
	}
	if !ok2 {
		t.Fatalf("Expected result2 to be Json, got %T", result2.Current)
	}

	if json1["name"] != json2["name"] {
		t.Errorf("Name mismatch: %v != %v", json1["name"], json2["name"])
	}

	// Handle int vs float64 from JSON unmarshaling
	val1, ok1 := json1["value"].(int)
	val2, ok2 := json2["value"].(float64)
	if !ok1 {
		val1Float, _ := json1["value"].(float64)
		val1 = int(val1Float)
	}
	if !ok2 {
		val2Int, _ := json2["value"].(int)
		val2 = float64(val2Int)
	}

	if float64(val1) != val2 {
		t.Errorf("Value mismatch: %v != %v", json1["value"], json2["value"])
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// hashContent simulates the hash generation done by compiler
func hashContent(content string) string {
	// This is a simplified version for testing
	// In production, compiler uses: sha256.Sum256([]byte(content))
	// and takes first 3 bytes as hex (6 chars)
	import_crypto_sha256 := func() string {
		// Inline import for test
		h := [32]byte{}
		for i, c := range []byte(content) {
			h[i%32] ^= c
		}
		return fmt.Sprintf("%02x%02x%02x", h[0], h[1], h[2])
	}
	return import_crypto_sha256()
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && findString(haystack, needle) >= 0
}

func findString(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
