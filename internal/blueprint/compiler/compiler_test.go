//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq/internal/blueprint/compiler/compiler_test.go
//

package compiler

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// ============================================================================
// Content Hash Generation Tests
// ============================================================================

func TestContentHashGeneration(t *testing.T) {
	tests := []struct {
		name     string
		prompt1  string
		prompt2  string
		samehash bool
	}{
		{
			name:     "identical prompts",
			prompt1:  "Extract key information",
			prompt2:  "Extract key information",
			samehash: true,
		},
		{
			name:     "different prompts",
			prompt1:  "Extract key information",
			prompt2:  "Transform to JSON",
			samehash: false,
		},
		{
			name:     "whitespace sensitive",
			prompt1:  "Extract information",
			prompt2:  "Extract  information",
			samehash: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := generateHash(tt.prompt1)
			hash2 := generateHash(tt.prompt2)

			if tt.samehash {
				if hash1 != hash2 {
					t.Errorf("Expected same hash for identical prompts, got %s and %s", hash1, hash2)
				}
			} else {
				if hash1 == hash2 {
					t.Errorf("Expected different hash for different prompts, got %s for both", hash1)
				}
			}

			// Verify hash is 6 characters (first 6 hex chars = 3 bytes)
			if len(hash1) != 6 {
				t.Errorf("Expected hash length 6, got %d: %s", len(hash1), hash1)
			}
		})
	}
}

func TestHashDeterminism(t *testing.T) {
	prompt := "Extract structured data from the document."

	// Generate hash multiple times
	hash1 := generateHash(prompt)
	hash2 := generateHash(prompt)
	hash3 := generateHash(prompt)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("Hash generation is not deterministic: %s, %s, %s", hash1, hash2, hash3)
	}
}

// generateHash mimics the compiler's hash generation
func generateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash[:3])
}
