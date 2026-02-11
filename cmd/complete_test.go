package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatIssueCompletions_Empty(t *testing.T) {
	t.Parallel()
	comps := formatIssueCompletions(nil)
	if comps != nil {
		t.Errorf("expected nil for empty input, got %v", comps)
	}
}

func TestFormatIssueCompletions_Format(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "First issue", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-2", Title: "Second issue", StateName: "Todo", StateType: "unstarted", Priority: 1},
	}

	comps := formatIssueCompletions(issues)

	// 1 ActiveHelp header + 2 issue entries
	if len(comps) != 3 {
		t.Fatalf("expected 3 entries (1 header + 2 issues), got %d: %v", len(comps), comps)
	}

	// Header should contain all column titles including IDENTIFIER
	header := comps[0]
	for _, col := range []string{"IDENTIFIER", "STATUS", "PRIORITY", "TITLE"} {
		if !strings.Contains(header, col) {
			t.Errorf("header should contain %q, got %q", col, header)
		}
	}

	// Issue entries should have identifier before \t and status+priority+title after
	for i, issue := range issues {
		entry := comps[i+1]
		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("entry %d should have value\\tdescription format, got %q", i, entry)
		}
		if parts[0] != issue.Identifier {
			t.Errorf("entry %d value = %q, want %q", i, parts[0], issue.Identifier)
		}
		desc := parts[1]
		if !strings.Contains(desc, issue.StateName) {
			t.Errorf("entry %d description should contain state %q, got %q", i, issue.StateName, desc)
		}
		if !strings.Contains(desc, issue.Title) {
			t.Errorf("entry %d description should contain title %q, got %q", i, issue.Title, desc)
		}
	}

	// Verify priority labels are present
	if !strings.Contains(comps[1], "High") {
		t.Errorf("entry 0 should contain priority label 'High', got %q", comps[1])
	}
	if !strings.Contains(comps[2], "Urgent") {
		t.Errorf("entry 1 should contain priority label 'Urgent', got %q", comps[2])
	}
}

func TestFormatIssueCompletions_Alignment(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "Short", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-2", Title: "Longer title", StateName: "Todo", StateType: "unstarted", Priority: 4},
	}

	comps := formatIssueCompletions(issues)

	// Extract descriptions (after \t)
	desc1 := strings.SplitN(comps[1], "\t", 2)[1]
	desc2 := strings.SplitN(comps[2], "\t", 2)[1]

	// Strip ANSI codes for length comparison
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

	plain1 := strip(desc1)
	plain2 := strip(desc2)

	// "Short" and "Longer title" start at the same column position
	titleIdx1 := strings.Index(plain1, "Short")
	titleIdx2 := strings.Index(plain2, "Longer title")
	if titleIdx1 != titleIdx2 {
		t.Errorf("titles should be aligned: %q starts at %d, %q starts at %d\nplain1: %q\nplain2: %q",
			"Short", titleIdx1, "Longer title", titleIdx2, plain1, plain2)
	}
}

func TestFormatIssueCompletions_HeaderAlignment(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "AIS-147", Title: "Security", StateName: "Backlog", StateType: "backlog", Priority: 3},
		{Identifier: "AIS-264", Title: "Auth middleware", StateName: "In Progress", StateType: "started", Priority: 2},
	}

	comps := formatIssueCompletions(issues)

	// Strip ActiveHelp prefix and ANSI codes.
	stripAnsi := func(s string) string {
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

	// The header is an ActiveHelp; strip the _activeHelp_ prefix.
	header := stripAnsi(comps[0])
	header = strings.TrimPrefix(header, "_activeHelp_ ")

	// Simulate zsh rendering: "VALUE  -- DESCRIPTION"
	// maxID = len("AIS-264") = 7; zsh pads to 7 then adds "  -- " (5 chars).
	parts1 := strings.SplitN(comps[1], "\t", 2)
	rendered := fmt.Sprintf("%-7s  -- %s", parts1[0], stripAnsi(parts1[1]))

	// STATUS column should start at the same position in both header and rendered row.
	headerStatusIdx := strings.Index(header, "STATUS")
	renderedStatusIdx := strings.Index(rendered, "Backlog")
	if headerStatusIdx != renderedStatusIdx {
		t.Errorf("STATUS column misaligned: header at %d, data at %d\nheader:   %q\nrendered: %q",
			headerStatusIdx, renderedStatusIdx, header, rendered)
	}

	// TITLE column should also align.
	headerTitleIdx := strings.Index(header, "TITLE")
	renderedTitleIdx := strings.Index(rendered, "Security")
	if headerTitleIdx != renderedTitleIdx {
		t.Errorf("TITLE column misaligned: header at %d, data at %d\nheader:   %q\nrendered: %q",
			headerTitleIdx, renderedTitleIdx, header, rendered)
	}
}

func TestFormatIssueCompletions_NilState(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "No state issue", Priority: 0},
	}

	comps := formatIssueCompletions(issues)

	if len(comps) != 2 {
		t.Fatalf("expected 2 entries (1 header + 1 issue), got %d", len(comps))
	}

	if !strings.Contains(comps[1], "ENG-1") {
		t.Errorf("should contain identifier, got %q", comps[1])
	}
	if !strings.Contains(comps[1], "None") {
		t.Errorf("should contain priority label 'None', got %q", comps[1])
	}
}

func TestUserCompletionEntry_Normal(t *testing.T) {
	t.Parallel()

	got := userCompletionEntry("Marc Dupont", "Marc Dupont")
	want := "marc\tMarc Dupont"
	if got != want {
		t.Errorf("userCompletionEntry(\"Marc Dupont\", \"Marc Dupont\") = %q, want %q", got, want)
	}
}

func TestUserCompletionEntry_EmptyDisplayName(t *testing.T) {
	t.Parallel()

	got := userCompletionEntry("", "Full Name")
	want := "\tFull Name"
	if got != want {
		t.Errorf("userCompletionEntry(\"\", \"Full Name\") = %q, want %q", got, want)
	}
}
