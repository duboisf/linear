package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Test doubles ---

type fakeIssue struct {
	identifier string
	title      string
	priority   float64
	updatedAt  string
}

func (f *fakeIssue) GetIdentifier() string { return f.identifier }
func (f *fakeIssue) GetTitle() string      { return f.title }
func (f *fakeIssue) GetPriority() float64  { return f.priority }
func (f *fakeIssue) GetUpdatedAt() string  { return f.updatedAt }

type fakeState struct {
	name     string
	stateType string
}

func (f *fakeState) GetName() string { return f.name }
func (f *fakeState) GetType() string { return f.stateType }

type fakeIssueWithState struct {
	*fakeIssue
	state *fakeState
}

func (f *fakeIssueWithState) GetState() *fakeState { return f.state }

func sampleIssues() []issueWithState {
	return []issueWithState{
		{
			Issue:     &fakeIssue{identifier: "ENG-1", title: "Fix login bug", priority: 2, updatedAt: "2025-06-15T10:00:00Z"},
			StateName: "In Progress",
			StateType: "started",
		},
		{
			Issue:     &fakeIssue{identifier: "ENG-2", title: "Add dark mode", priority: 3, updatedAt: "2025-06-14T08:00:00Z"},
			StateName: "Todo",
			StateType: "unstarted",
		},
		{
			Issue:     &fakeIssue{identifier: "ENG-3", title: "Refactor auth module", priority: 1, updatedAt: "2025-06-13T12:00:00Z"},
			StateName: "Backlog",
			StateType: "backlog",
		},
	}
}

// --- formatDate ---

func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "valid RFC3339",
			give: "2025-06-15T10:30:00Z",
			want: "2025-06-15",
		},
		{
			name: "valid with timezone offset",
			give: "2025-01-02T15:04:05-07:00",
			want: "2025-01-02",
		},
		{
			name: "invalid returns input",
			give: "not-a-date",
			want: "not-a-date",
		},
		{
			name: "empty returns empty",
			give: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDate(tt.give)
			if got != tt.want {
				t.Errorf("formatDate(%q) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}

// --- WrapActiveIssue / WrapIssues ---

func TestWrapActiveIssue(t *testing.T) {
	t.Parallel()

	t.Run("with state", func(t *testing.T) {
		t.Parallel()
		issue := &fakeIssueWithState{
			fakeIssue: &fakeIssue{identifier: "ENG-1", title: "Test"},
			state:     &fakeState{name: "In Progress", stateType: "started"},
		}
		wrapped := WrapActiveIssue[*fakeState](issue)
		if wrapped.StateName != "In Progress" {
			t.Errorf("StateName = %q, want %q", wrapped.StateName, "In Progress")
		}
		if wrapped.StateType != "started" {
			t.Errorf("StateType = %q, want %q", wrapped.StateType, "started")
		}
		if wrapped.GetIdentifier() != "ENG-1" {
			t.Errorf("GetIdentifier() = %q, want %q", wrapped.GetIdentifier(), "ENG-1")
		}
	})

	t.Run("nil state", func(t *testing.T) {
		t.Parallel()
		issue := &fakeIssueWithState{
			fakeIssue: &fakeIssue{identifier: "ENG-2", title: "Test"},
			state:     nil,
		}
		wrapped := WrapActiveIssue[*fakeState](issue)
		if wrapped.StateName != "" {
			t.Errorf("StateName = %q, want empty", wrapped.StateName)
		}
		if wrapped.StateType != "" {
			t.Errorf("StateType = %q, want empty", wrapped.StateType)
		}
	})
}

func TestWrapIssues(t *testing.T) {
	t.Parallel()

	issues := []*fakeIssueWithState{
		{fakeIssue: &fakeIssue{identifier: "A-1"}, state: &fakeState{name: "Todo", stateType: "unstarted"}},
		{fakeIssue: &fakeIssue{identifier: "A-2"}, state: &fakeState{name: "Done", stateType: "completed"}},
	}
	wrapped := WrapIssues[*fakeState](issues)
	if len(wrapped) != 2 {
		t.Fatalf("len = %d, want 2", len(wrapped))
	}
	if wrapped[0].GetIdentifier() != "A-1" {
		t.Errorf("[0].GetIdentifier() = %q, want %q", wrapped[0].GetIdentifier(), "A-1")
	}
	if wrapped[1].StateName != "Done" {
		t.Errorf("[1].StateName = %q, want %q", wrapped[1].StateName, "Done")
	}
}

// --- NewModel + applyFilter ---

func TestNewModel(t *testing.T) {
	t.Parallel()

	issues := sampleIssues()
	m := NewModel(issues, false)

	if len(m.filtered) != 3 {
		t.Errorf("filtered = %d, want 3", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.selected != "" {
		t.Errorf("selected = %q, want empty", m.selected)
	}
}

func TestApplyFilter(t *testing.T) {
	t.Parallel()

	issues := sampleIssues()
	m := NewModel(issues, false)

	// Simulate typing "auth" into the filter.
	m.filter.SetValue("auth")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(m.filtered))
	}
	if m.issues[m.filtered[0]].GetIdentifier() != "ENG-3" {
		t.Errorf("filtered issue = %q, want ENG-3", m.issues[m.filtered[0]].GetIdentifier())
	}
}

func TestApplyFilter_ByIdentifier(t *testing.T) {
	t.Parallel()

	issues := sampleIssues()
	m := NewModel(issues, false)

	m.filter.SetValue("eng-2")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(m.filtered))
	}
	if m.issues[m.filtered[0]].GetIdentifier() != "ENG-2" {
		t.Errorf("filtered issue = %q, want ENG-2", m.issues[m.filtered[0]].GetIdentifier())
	}
}

func TestApplyFilter_NoMatch(t *testing.T) {
	t.Parallel()

	issues := sampleIssues()
	m := NewModel(issues, false)

	m.filter.SetValue("nonexistent")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("filtered = %d, want 0", len(m.filtered))
	}
}

// --- Update: keyboard navigation ---

func sendKey(m Model, keyType tea.KeyType) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated.(Model)
}

func sendRune(m Model, r rune) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return updated.(Model)
}

func TestUpdate_CursorNavigation(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	m = sendKey(m, tea.KeyDown)
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	m = sendKey(m, tea.KeyDown)
	if m.cursor != 2 {
		t.Errorf("after down: cursor = %d, want 2", m.cursor)
	}

	// Should not go past last item.
	m = sendKey(m, tea.KeyDown)
	if m.cursor != 2 {
		t.Errorf("after down past end: cursor = %d, want 2", m.cursor)
	}

	m = sendKey(m, tea.KeyUp)
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}

	// Should not go below 0.
	m = sendKey(m, tea.KeyUp)
	m = sendKey(m, tea.KeyUp)
	if m.cursor != 0 {
		t.Errorf("after up past start: cursor = %d, want 0", m.cursor)
	}
}

func TestUpdate_VimNavigation(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)

	// With empty filter, j/k should navigate.
	m = sendRune(m, 'j')
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	m = sendRune(m, 'k')
	if m.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", m.cursor)
	}
}

func TestUpdate_CtrlNavigation(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)

	m = sendKey(m, tea.KeyCtrlJ)
	if m.cursor != 1 {
		t.Errorf("after ctrl+j: cursor = %d, want 1", m.cursor)
	}

	m = sendKey(m, tea.KeyCtrlK)
	if m.cursor != 0 {
		t.Errorf("after ctrl+k: cursor = %d, want 0", m.cursor)
	}
}

func TestUpdate_EnterSelectsIssue(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m = sendKey(m, tea.KeyDown) // move to ENG-2
	m = sendKey(m, tea.KeyEnter)

	if m.selected != "ENG-2" {
		t.Errorf("selected = %q, want %q", m.selected, "ENG-2")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestUpdate_EscCancels(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m = sendKey(m, tea.KeyEsc)

	if m.selected != "" {
		t.Errorf("selected = %q, want empty", m.selected)
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestUpdate_CtrlCCancels(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m = sendKey(m, tea.KeyCtrlC)

	if m.selected != "" {
		t.Errorf("selected = %q, want empty", m.selected)
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestUpdate_EnterOnEmptyFilteredList(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m.filter.SetValue("nonexistent")
	m.applyFilter()

	m = sendKey(m, tea.KeyEnter)

	if m.selected != "" {
		t.Errorf("selected = %q, want empty on empty list", m.selected)
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

// --- View ---

func TestView_ContainsIssues(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	output := m.View()

	for _, want := range []string{"ENG-1", "ENG-2", "ENG-3", "Fix login bug", "Add dark mode", "IDENTIFIER", "Filter:"} {
		if !strings.Contains(output, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestView_EmptyOnQuit(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m = sendKey(m, tea.KeyEsc)
	output := m.View()

	if output != "" {
		t.Errorf("View() after quit = %q, want empty", output)
	}
}

func TestView_NoMatchingIssues(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	m.filter.SetValue("zzz")
	m.applyFilter()

	output := m.View()
	if !strings.Contains(output, "No matching issues") {
		t.Error("View() missing 'No matching issues' message")
	}
}

// --- Selected ---

func TestSelected_DefaultEmpty(t *testing.T) {
	t.Parallel()

	m := NewModel(sampleIssues(), false)
	if m.Selected() != "" {
		t.Errorf("Selected() = %q, want empty", m.Selected())
	}
}
