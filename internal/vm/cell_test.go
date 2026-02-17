//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package vm_test

import (
	"testing"

	"github.com/fogfish/iq/internal/vm"
)

// testGist is a minimal Gist for testing
type testGist string

func (g testGist) ContentType() string { return "text/plain" }

func TestNewCell(t *testing.T) {
	c := vm.NewCell("doc.txt", testGist("hello"))
	if c.Key != "doc.txt" {
		t.Errorf("expected key doc.txt, got %s", c.Key)
	}
	if c.Value.ContentType() != "text/plain" {
		t.Errorf("unexpected content type")
	}
	if c.Steps == nil {
		t.Error("Steps map should be initialized")
	}
	if c.Env == nil {
		t.Error("Env map should be initialized")
	}
}

func TestCellCopy(t *testing.T) {
	c := vm.NewCell("doc.txt", testGist("hello"))
	c.Steps["extract"] = testGist("result")

	c2 := c.Copy(testGist("world"))
	if c2.Key != "doc.txt" {
		t.Errorf("copy should preserve key")
	}
	if c2.Value.(testGist) != "world" {
		t.Errorf("copy should have new value")
	}
	// Shared Steps map
	if _, ok := c2.Steps["extract"]; !ok {
		t.Error("copy should share Steps map")
	}
}
