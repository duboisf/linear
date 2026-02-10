package cmd

import (
	"strings"
	"testing"
)

func TestFormatFzfLines_Empty(t *testing.T) {
	t.Parallel()

	header, lines := formatFzfLines(nil)
	if header != "" {
		t.Errorf("expected empty header for nil input, got %q", header)
	}
	if lines != nil {
		t.Errorf("expected nil lines for nil input, got %v", lines)
	}
}

func TestFormatFzfLines_Format(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "First issue", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-2", Title: "Second issue", StateName: "Todo", StateType: "unstarted", Priority: 3},
	}

	header, lines := formatFzfLines(issues)

	// Header should contain column titles
	for _, col := range []string{"IDENTIFIER", "STATUS", "PRIORITY", "TITLE"} {
		if !strings.Contains(header, col) {
			t.Errorf("header should contain %q, got %q", col, header)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Lines should contain identifiers
	if !strings.Contains(lines[0], "ENG-1") {
		t.Errorf("line 0 should contain ENG-1, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "ENG-2") {
		t.Errorf("line 1 should contain ENG-2, got %q", lines[1])
	}

	// Lines should contain titles
	if !strings.Contains(lines[0], "First issue") {
		t.Errorf("line 0 should contain title, got %q", lines[0])
	}

	// Lines should contain priority labels
	if !strings.Contains(lines[0], "High") {
		t.Errorf("line 0 should contain priority 'High', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Normal") {
		t.Errorf("line 1 should contain priority 'Normal', got %q", lines[1])
	}
}

func TestFormatFzfLines_Alignment(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "Short", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-200", Title: "Longer title", StateName: "Todo", StateType: "unstarted", Priority: 4},
	}

	_, lines := formatFzfLines(issues)

	// Strip ANSI codes for alignment comparison
	strip := func(s string) string {
		var b strings.Builder
		inEsc := false
		for _, r := range s {
			if r == '\033' {
				inEsc = true
				continue
			}
			if inEsc {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					inEsc = false
				}
				continue
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	plain0 := strip(lines[0])
	plain1 := strip(lines[1])

	// Titles should be at the same column position
	titleIdx0 := strings.Index(plain0, "Short")
	titleIdx1 := strings.Index(plain1, "Longer title")
	if titleIdx0 != titleIdx1 {
		t.Errorf("titles should be aligned: 'Short' at %d, 'Longer title' at %d\nline0: %q\nline1: %q",
			titleIdx0, titleIdx1, plain0, plain1)
	}
}

func TestFormatFzfLines_NilState(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "No state issue", Priority: 0},
	}

	header, lines := formatFzfLines(issues)
	if header == "" {
		t.Error("expected non-empty header")
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "ENG-1") {
		t.Errorf("line should contain ENG-1, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "None") {
		t.Errorf("line should contain priority 'None', got %q", lines[0])
	}
}
