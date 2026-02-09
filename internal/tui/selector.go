package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/duboisf/linear/internal/format"
)

// IssueState represents a workflow state with name and type.
type IssueState interface {
	GetName() string
	GetType() string
}

// Issue is the interface that issue types must satisfy for the selector.
type Issue interface {
	GetIdentifier() string
	GetTitle() string
	GetPriority() float64
	GetUpdatedAt() string
}

// issueWithState wraps an Issue together with its state info, since the
// concrete state types differ between Active and All issue queries.
type issueWithState struct {
	Issue
	StateName string
	StateType string
}

// WrapActiveIssue wraps an issue that has a GetState() returning a state pointer.
// S should be the pointer type (e.g., *WorkflowState) since GetName/GetType are pointer receivers.
func WrapActiveIssue[S interface {
	comparable
	GetName() string
	GetType() string
}, T interface {
	Issue
	GetState() S
}](issue T) issueWithState {
	iws := issueWithState{Issue: issue}
	var zero S
	if s := issue.GetState(); s != zero {
		iws.StateName = s.GetName()
		iws.StateType = s.GetType()
	}
	return iws
}

// WrapIssues converts a slice of typed issues into []issueWithState using WrapActiveIssue.
// S should be the pointer type (e.g., *WorkflowState) since GetName/GetType are pointer receivers.
func WrapIssues[S interface {
	comparable
	GetName() string
	GetType() string
}, T interface {
	Issue
	GetState() S
}](issues []T) []issueWithState {
	result := make([]issueWithState, len(issues))
	for i, issue := range issues {
		result[i] = WrapActiveIssue[S, T](issue)
	}
	return result
}

// Column widths (minimums).
const (
	colIdentifier = 12
	colState      = 14
	colPriority   = 10
	colUpdated    = 12
	minTitleWidth = 10
)

// Model implements tea.Model for the interactive issue selector.
type Model struct {
	issues   []issueWithState
	filtered []int // indices into issues
	filter   textinput.Model
	cursor   int
	selected string // identifier of the selected issue, empty if cancelled
	quitting bool
	width    int
	height   int
	color    bool
}

// NewModel creates a new selector model.
func NewModel(issues []issueWithState, color bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 100

	m := Model{
		issues: issues,
		filter: ti,
		color:  color,
	}
	m.applyFilter()
	return m
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter.Value())
	m.filtered = m.filtered[:0]
	for i, issue := range m.issues {
		if query == "" ||
			strings.Contains(strings.ToLower(issue.GetIdentifier()), query) ||
			strings.Contains(strings.ToLower(issue.GetTitle()), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.issues[m.filtered[m.cursor]].GetIdentifier()
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyUp, tea.KeyCtrlK:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown, tea.KeyCtrlJ:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyRunes:
			if len(msg.Runes) == 1 {
				switch msg.Runes[0] {
				case 'k':
					if m.filter.Value() == "" {
						if m.cursor > 0 {
							m.cursor--
						}
						return m, nil
					}
				case 'j':
					if m.filter.Value() == "" {
						if m.cursor < len(m.filtered)-1 {
							m.cursor++
						}
						return m, nil
					}
				}
			}
		}
	}

	prevValue := m.filter.Value()
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prevValue {
		m.applyFilter()
	}
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Filter input
	b.WriteString(" Filter: ")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	width := m.width
	if width == 0 {
		width = 80
	}

	// Calculate title width: total minus fixed columns minus padding/markers
	// "   " (3 marker) + colIdentifier + "  " + colState + "  " + colPriority + "  " + title + "  " + colUpdated
	fixedWidth := 3 + colIdentifier + 2 + colState + 2 + colPriority + 2 + 2 + colUpdated
	titleWidth := width - fixedWidth
	if titleWidth < minTitleWidth {
		titleWidth = minTitleWidth
	}

	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	_ = selectedStyle

	// Header
	header := fmt.Sprintf("   %-*s  %-*s  %-*s  %-*s  %-*s",
		colIdentifier, "IDENTIFIER",
		colState, "STATE",
		colPriority, "PRIORITY",
		titleWidth, "TITLE",
		colUpdated, "UPDATED",
	)
	if m.color {
		header = headerStyle.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Visible rows: leave room for filter (2 lines), header (1 line), help (2 lines)
	visibleRows := m.height - 5
	if visibleRows < 3 {
		visibleRows = len(m.filtered)
	}

	// Scroll offset
	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(m.filtered) {
		endIdx = len(m.filtered)
	}

	noColor := !m.color

	for i := startIdx; i < endIdx; i++ {
		idx := m.filtered[i]
		issue := m.issues[idx]
		isSelected := i == m.cursor

		marker := "  "
		if isSelected {
			marker = " \u25b8"
		}

		identifier := issue.GetIdentifier()
		stateName := issue.StateName
		stateType := issue.StateType
		priorityLabel := format.PriorityLabel(issue.GetPriority())
		title := issue.GetTitle()
		updatedAt := formatDate(issue.GetUpdatedAt())

		// Truncate title if needed
		if len(title) > titleWidth {
			title = title[:titleWidth-1] + "\u2026"
		}

		// Pad to fixed width, then apply color (so ANSI codes don't break alignment)
		stateCol := format.PadColor(m.color, format.StateColor(stateType), stateName, colState)
		priorityCol := format.PadColor(m.color, format.PriorityColor(issue.GetPriority()), priorityLabel, colPriority)

		row := fmt.Sprintf("%s %-*s  %s  %s  %-*s  %-*s",
			marker,
			colIdentifier, identifier,
			stateCol,
			priorityCol,
			titleWidth, title,
			colUpdated, updatedAt,
		)

		if isSelected && !noColor {
			row = lipgloss.NewStyle().Bold(true).Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString("   No matching issues.\n")
	}

	// Help line
	b.WriteString("\n")
	helpText := "up/down navigate | enter select | esc quit | type to filter"
	if !noColor {
		helpText = lipgloss.NewStyle().Faint(true).Render(helpText)
	}
	b.WriteString(" " + helpText)

	return b.String()
}

// formatDate parses an RFC3339 time string and returns YYYY-MM-DD.
func formatDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02")
}

// Selected returns the identifier of the selected issue, or empty if cancelled.
func (m Model) Selected() string {
	return m.selected
}

// RunSelector launches the interactive issue selector and returns the selected identifier.
// Returns empty string if the user cancelled.
func RunSelector(issues []issueWithState, in io.Reader, out io.Writer) (string, error) {
	if len(issues) == 0 {
		return "", fmt.Errorf("no issues to select from")
	}

	_, noColor := os.LookupEnv("NO_COLOR")
	color := !noColor

	m := NewModel(issues, color)

	opts := []tea.ProgramOption{
		tea.WithOutput(out),
	}
	if in != nil {
		opts = append(opts, tea.WithInput(in))
	}

	p := tea.NewProgram(m, opts...)
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running selector: %w", err)
	}

	final := result.(Model)
	return final.Selected(), nil
}
