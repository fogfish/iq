//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package compiler_test

/*

import (
	"testing"

	"github.com/fogfish/iq/internal/blueprint/compiler"
	"github.com/fogfish/iq/internal/iosystem"
)

func TestApplyEmit(t *testing.T) {
	tests := []struct {
		name string
		emit string
		key  iosystem.Key
		want iosystem.Key
	}{
		{
			name: "with emit prefix",
			emit: "summary",
			key:  "a.txt",
			want: "summary/a.txt",
		},
		{
			name: "without emit",
			emit: "",
			key:  "a.txt",
			want: "a.txt",
		},
		{
			name: "subdirectory with emit",
			emit: "summary",
			key:  "sub/a.txt",
			want: "summary/sub/a.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compiler.ApplyEmit(tt.emit, tt.key)
			if got != tt.want {
				t.Errorf("ApplyEmit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyEmitWithCounters(t *testing.T) {
	tests := []struct {
		name     string
		emit     string
		key      iosystem.Key
		counters []int
		want     iosystem.Key
	}{
		{
			name:     "single counter",
			emit:     "research",
			key:      "a.txt",
			counters: []int{1},
			want:     "research/a.000001.txt",
		},
		{
			name:     "nested counters",
			emit:     "research",
			key:      "a.txt",
			counters: []int{1, 5},
			want:     "research/a.000001.000005.txt",
		},
		{
			name:     "no emit, with counter",
			emit:     "",
			key:      "a.txt",
			counters: []int{1},
			want:     "a.000001.txt",
		},
		{
			name:     "no counters",
			emit:     "research",
			key:      "a.txt",
			counters: []int{},
			want:     "research/a.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compiler.ApplyEmitWithCounters(tt.emit, tt.key, tt.counters)
			if got != tt.want {
				t.Errorf("ApplyEmitWithCounters() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmitContext_PushPop(t *testing.T) {
	ec := &compiler.EmitContext{}

	// Test push
	ec.PushCounter(1)
	if len(ec.Counters) != 1 || ec.Counters[0] != 1 {
		t.Errorf("After PushCounter(1), got %v, want [1]", ec.Counters)
	}

	ec.PushCounter(5)
	if len(ec.Counters) != 2 || ec.Counters[1] != 5 {
		t.Errorf("After PushCounter(5), got %v, want [1, 5]", ec.Counters)
	}

	// Test pop
	ec.PopCounter()
	if len(ec.Counters) != 1 || ec.Counters[0] != 1 {
		t.Errorf("After PopCounter(), got %v, want [1]", ec.Counters)
	}

	ec.PopCounter()
	if len(ec.Counters) != 0 {
		t.Errorf("After PopCounter(), got %v, want []", ec.Counters)
	}

	// Test pop on empty
	ec.PopCounter()
	if len(ec.Counters) != 0 {
		t.Errorf("After PopCounter() on empty, got %v, want []", ec.Counters)
	}
}

*/
