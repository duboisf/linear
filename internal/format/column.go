package format

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/duboisf/linear/internal/api"
)

type issueListNode = api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue

// ColumnDef defines how to render a single column in the issue list table.
type ColumnDef struct {
	Header string
	Value  func(issue *issueListNode) string
	Color  func(issue *issueListNode) string
}

// _columnRegistry maps column names to their definitions.
var _columnRegistry = map[string]ColumnDef{
	"id": {
		Header: "IDENTIFIER",
		Value:  func(issue *issueListNode) string { return issue.Identifier },
		Color:  func(_ *issueListNode) string { return "" },
	},
	"status": {
		Header: "STATUS",
		Value: func(issue *issueListNode) string {
			if issue.State != nil {
				return issue.State.Name
			}
			return ""
		},
		Color: func(issue *issueListNode) string {
			if issue.State != nil {
				return StateColor(issue.State.Type)
			}
			return ""
		},
	},
	"priority": {
		Header: "PRIORITY",
		Value:  func(issue *issueListNode) string { return PriorityLabel(issue.Priority) },
		Color:  func(issue *issueListNode) string { return PriorityColor(issue.Priority) },
	},
	"labels": {
		Header: "LABELS",
		Value:  func(issue *issueListNode) string { return issueLabels(issue) },
		Color:  func(_ *issueListNode) string { return Cyan },
	},
	"title": {
		Header: "TITLE",
		Value:  func(issue *issueListNode) string { return issue.Title },
		Color:  func(_ *issueListNode) string { return "" },
	},
	"updated": {
		Header: "UPDATED",
		Value: func(issue *issueListNode) string {
			t, err := time.Parse(time.RFC3339, issue.UpdatedAt)
			if err != nil {
				return issue.UpdatedAt
			}
			return t.Format("2006-01-02")
		},
		Color: func(_ *issueListNode) string { return Gray },
	},
	"created": {
		Header: "CREATED",
		Value: func(issue *issueListNode) string {
			t, err := time.Parse(time.RFC3339, issue.CreatedAt)
			if err != nil {
				return issue.CreatedAt
			}
			return t.Format("2006-01-02")
		},
		Color: func(_ *issueListNode) string { return Gray },
	},
	"cycle": {
		Header: "CYCLE",
		Value: func(issue *issueListNode) string {
			if issue.Cycle == nil {
				return ""
			}
			return fmt.Sprintf("%.0f", issue.Cycle.Number)
		},
		Color: func(_ *issueListNode) string { return "" },
	},
	"assignee": {
		Header: "ASSIGNEE",
		Value: func(issue *issueListNode) string {
			if issue.Assignee == nil {
				return ""
			}
			return issue.Assignee.Name
		},
		Color: func(_ *issueListNode) string { return "" },
	},
	"project": {
		Header: "PROJECT",
		Value: func(issue *issueListNode) string {
			if issue.Project == nil {
				return ""
			}
			return issue.Project.Name
		},
		Color: func(_ *issueListNode) string { return "" },
	},
	"estimate": {
		Header: "ESTIMATE",
		Value: func(issue *issueListNode) string {
			if issue.Estimate == nil {
				return ""
			}
			return fmt.Sprintf("%.0f", *issue.Estimate)
		},
		Color: func(_ *issueListNode) string { return "" },
	},
	"duedate": {
		Header: "DUE DATE",
		Value: func(issue *issueListNode) string {
			if issue.DueDate == nil {
				return ""
			}
			return *issue.DueDate
		},
		Color: func(_ *issueListNode) string { return Gray },
	},
}

// ColumnNames lists all known column names in canonical order.
var ColumnNames = []string{"id", "status", "priority", "labels", "title", "updated", "created", "cycle", "assignee", "project", "estimate", "duedate"}

// _defaultColumnNames is the full default column set (base for additive mode).
var _defaultColumnNames = []string{"id", "status", "priority", "labels", "title"}

// DefaultColumns returns the default column set, including labels only if any
// issue has labels. Used when no --column flag is specified.
func DefaultColumns(issues []*issueListNode) []string {
	for _, issue := range issues {
		if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
			return []string{"id", "status", "priority", "labels", "title"}
		}
	}
	return []string{"id", "status", "priority", "title"}
}

// ParseColumns parses a --column flag value into a list of column names.
//
// Syntax:
//   - "id,status,title"  replacement: show exactly these columns
//   - "+updated"         additive: append to defaults
//   - "+updated:2"       additive: insert at position 2 (1-based)
//
// All entries must be either additive (+) or replacement (no +), not mixed.
func ParseColumns(spec string) ([]string, error) {
	parts := strings.Split(spec, ",")

	hasAdditive := false
	hasReplacement := false
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "+") {
			hasAdditive = true
		} else {
			hasReplacement = true
		}
	}
	if hasAdditive && hasReplacement {
		return nil, fmt.Errorf("cannot mix additive (+col) and replacement (col) syntax in --column")
	}
	if !hasAdditive && !hasReplacement {
		return nil, fmt.Errorf("--column requires at least one column name")
	}

	if hasReplacement {
		return parseReplacementColumns(parts)
	}
	return parseAdditiveColumns(parts)
}

func parseReplacementColumns(parts []string) ([]string, error) {
	var columns []string
	seen := make(map[string]bool)
	for _, p := range parts {
		name := strings.TrimSpace(strings.ToLower(p))
		if name == "" {
			continue
		}
		if _, ok := _columnRegistry[name]; !ok {
			return nil, fmt.Errorf("unknown column %q (available: %s)", name, strings.Join(ColumnNames, ", "))
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate column %q", name)
		}
		seen[name] = true
		columns = append(columns, name)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("--column requires at least one column name")
	}
	return columns, nil
}

func parseAdditiveColumns(parts []string) ([]string, error) {
	columns := make([]string, len(_defaultColumnNames))
	copy(columns, _defaultColumnNames)

	seen := make(map[string]bool, len(columns))
	for _, c := range columns {
		seen[c] = true
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "+")

		name := p
		pos := 0
		if colName, posStr, ok := strings.Cut(p, ":"); ok {
			name = colName
			n, err := strconv.Atoi(posStr)
			if err != nil {
				return nil, fmt.Errorf("invalid position in +%s: %w", p, err)
			}
			if n < 1 {
				return nil, fmt.Errorf("position must be >= 1, got %d", n)
			}
			pos = n
		}

		name = strings.ToLower(name)
		if _, ok := _columnRegistry[name]; !ok {
			return nil, fmt.Errorf("unknown column %q (available: %s)", name, strings.Join(ColumnNames, ", "))
		}
		if seen[name] {
			continue // already in defaults, skip silently
		}
		seen[name] = true

		if pos == 0 {
			columns = append(columns, name)
		} else {
			idx := min(pos-1, len(columns))
			columns = slices.Insert(columns, idx, name)
		}
	}

	return columns, nil
}
