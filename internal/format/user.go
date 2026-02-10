package format

import (
	"fmt"
	"strings"

	"github.com/duboisf/linear/internal/api"
)

// RoleLabel returns a human-readable role string.
func RoleLabel(admin bool) string {
	if admin {
		return "Admin"
	}
	return "Member"
}

// StatusLabel returns a human-readable status string.
func StatusLabel(active bool) string {
	if active {
		return "Active"
	}
	return "Disabled"
}

// StatusColor returns the ANSI color code for the given active status.
func StatusColor(active bool) string {
	if active {
		return Green
	}
	return Red
}

// FormatUserList formats a slice of users as an aligned table for terminal output.
func FormatUserList(users []*api.ListUsersUsersUserConnectionNodesUser, color bool) string {
	const gap = "  "

	// Compute max visible widths per column.
	maxName := len("NAME")
	maxDisplay := len("DISPLAY NAME")
	maxEmail := len("EMAIL")
	maxRole := len("ROLE")
	maxStatus := len("STATUS")
	for _, u := range users {
		if len(u.Name) > maxName {
			maxName = len(u.Name)
		}
		if len(u.DisplayName) > maxDisplay {
			maxDisplay = len(u.DisplayName)
		}
		if len(u.Email) > maxEmail {
			maxEmail = len(u.Email)
		}
		if l := len(RoleLabel(u.Admin)); l > maxRole {
			maxRole = l
		}
		if l := len(StatusLabel(u.Active)); l > maxStatus {
			maxStatus = l
		}
	}

	var buf strings.Builder

	// Header
	buf.WriteString(PadColor(color, Bold, "NAME", maxName))
	buf.WriteString(gap)
	buf.WriteString(PadColor(color, Bold, "DISPLAY NAME", maxDisplay))
	buf.WriteString(gap)
	buf.WriteString(PadColor(color, Bold, "EMAIL", maxEmail))
	buf.WriteString(gap)
	buf.WriteString(PadColor(color, Bold, "ROLE", maxRole))
	buf.WriteString(gap)
	buf.WriteString(Colorize(color, Bold, "STATUS"))
	buf.WriteByte('\n')

	// Rows
	for _, u := range users {
		buf.WriteString(fmt.Sprintf("%-*s", maxName, u.Name))
		buf.WriteString(gap)
		buf.WriteString(fmt.Sprintf("%-*s", maxDisplay, u.DisplayName))
		buf.WriteString(gap)
		buf.WriteString(fmt.Sprintf("%-*s", maxEmail, u.Email))
		buf.WriteString(gap)
		buf.WriteString(fmt.Sprintf("%-*s", maxRole, RoleLabel(u.Admin)))
		buf.WriteString(gap)

		buf.WriteString(Colorize(color, StatusColor(u.Active), StatusLabel(u.Active)))
		buf.WriteByte('\n')
	}

	return buf.String()
}

// FormatUserDetail formats a single user in a detailed key-value format.
func FormatUserDetail(user *api.GetUserByDisplayNameUsersUserConnectionNodesUser, color bool) string {
	var buf strings.Builder

	field := func(label, value string) {
		fmt.Fprintf(&buf, "%s %s\n", Colorize(color, Bold, label+":"), value)
	}

	field("Name", user.Name)
	field("Display Name", user.DisplayName)
	field("Email", user.Email)
	field("Role", RoleLabel(user.Admin))

	field("Status", Colorize(color, StatusColor(user.Active), StatusLabel(user.Active)))

	if user.IsMe {
		field("Is Me", "Yes")
	}

	return buf.String()
}
