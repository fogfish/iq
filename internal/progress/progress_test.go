//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package progress

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestReporter_Basic(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithWriter(buf, false)

	r.WorkflowLoading("test.yml")
	output := buf.String()

	if !strings.Contains(output, "📋 Loading workflow from test.yml") {
		t.Errorf("Expected workflow loading message, got: %s", output)
	}
}

func TestReporter_Quiet(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithWriter(buf, true)

	r.WorkflowLoading("test.yml")
	r.DocumentStart("doc.txt", 1.5)
	r.StepStart("main", "step1", 1, 3)

	if buf.Len() > 0 {
		t.Errorf("Expected no output in quiet mode, got: %s", buf.String())
	}
}

func TestReporter_DocumentLifecycle(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithWriter(buf, false)

	r.DocumentStart("doc.txt", 2.5)
	r.DocumentComplete("doc.txt", time.Second*3)

	output := buf.String()
	if !strings.Contains(output, "📄 Processing: doc.txt (2.5 KB)") {
		t.Errorf("Expected document start message")
	}
	if !strings.Contains(output, "✅ Completed: doc.txt") {
		t.Errorf("Expected document complete message")
	}
}

func TestReporter_StepExecution(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithWriter(buf, false)

	r.StepStart("main", "extract", 1, 3)
	r.StepComplete("main", "extract", 1, 3, time.Second*2, 1234)

	output := buf.String()
	if !strings.Contains(output, "Step 1/3") {
		t.Errorf("Expected step progress")
	}
	if !strings.Contains(output, "1234 tokens") {
		t.Errorf("Expected token count")
	}
}

func TestReporter_Context(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithWriter(buf, false)

	ctx := context.Background()
	ctx = WithReporter(ctx, r)

	retrieved := FromContext(ctx)
	if retrieved != r {
		t.Errorf("Expected to retrieve same reporter from context")
	}
}

func TestReporter_StepInfo(t *testing.T) {
	ctx := context.Background()
	info := StepInfo{
		JobName:  "test",
		StepName: "step1",
		StepID:   1,
		JobSize:  3,
	}

	ctx = WithStepInfo(ctx, info)
	retrieved := GetStepInfo(ctx)

	if retrieved == nil {
		t.Fatal("Expected to retrieve step info")
	}
	if retrieved.JobName != "test" || retrieved.StepID != 1 {
		t.Errorf("Step info mismatch: %+v", retrieved)
	}
}

func TestReporter_Stats(t *testing.T) {
	r := New(false)

	r.UpdateTokens(1000, 500)
	r.UpdateTokens(2000, 1000)

	stats := r.GetStats()
	if stats.TokensInput != 3000 {
		t.Errorf("Expected 3000 input tokens, got %d", stats.TokensInput)
	}
	if stats.TokensOutput != 1500 {
		t.Errorf("Expected 1500 output tokens, got %d", stats.TokensOutput)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{100, "100"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1000000, "1.0M"},
		{1234567, "1.2M"},
	}

	for _, tt := range tests {
		result := formatNumber(tt.input)
		if result != tt.expected {
			t.Errorf("formatNumber(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		contains string
	}{
		{time.Millisecond * 500, "500ms"},
		{time.Second * 2, "2.0s"},
		{time.Second * 90, "1m 30s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("formatDuration(%v) = %s, should contain %s", tt.input, result, tt.contains)
		}
	}
}
