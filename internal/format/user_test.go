package format_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func TestRoleLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		admin bool
		want  string
	}{
		{name: "admin", admin: true, want: "Admin"},
		{name: "member", admin: false, want: "Member"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.RoleLabel(tt.admin)
			if got != tt.want {
				t.Errorf("RoleLabel(%v) = %q, want %q", tt.admin, got, tt.want)
			}
		})
	}
}

func TestStatusLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		active bool
		want   string
	}{
		{name: "active", active: true, want: "Active"},
		{name: "disabled", active: false, want: "Disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.StatusLabel(tt.active)
			if got != tt.want {
				t.Errorf("StatusLabel(%v) = %q, want %q", tt.active, got, tt.want)
			}
		})
	}
}

func TestFormatUserList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		users  []*api.ListUsersUsersUserConnectionNodesUser
		color  bool
		checks func(t *testing.T, got string)
	}{
		{
			name:  "empty list shows header only",
			users: nil,
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "NAME") {
					t.Error("expected header to contain NAME")
				}
				if !strings.Contains(got, "DISPLAY NAME") {
					t.Error("expected header to contain DISPLAY NAME")
				}
				if !strings.Contains(got, "EMAIL") {
					t.Error("expected header to contain EMAIL")
				}
				if !strings.Contains(got, "ROLE") {
					t.Error("expected header to contain ROLE")
				}
				if !strings.Contains(got, "STATUS") {
					t.Error("expected header to contain STATUS")
				}
				lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
				if len(lines) != 1 {
					t.Errorf("expected 1 line (header only), got %d", len(lines))
				}
			},
		},
		{
			name: "single active admin user",
			users: []*api.ListUsersUsersUserConnectionNodesUser{
				{
					Name:        "Jane Doe",
					DisplayName: "jane",
					Email:       "jane@example.com",
					Active:      true,
					Admin:       true,
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Jane Doe") {
					t.Error("expected output to contain Jane Doe")
				}
				if !strings.Contains(got, "jane") {
					t.Error("expected output to contain jane")
				}
				if !strings.Contains(got, "jane@example.com") {
					t.Error("expected output to contain jane@example.com")
				}
				if !strings.Contains(got, "Admin") {
					t.Error("expected output to contain Admin")
				}
				if !strings.Contains(got, "Active") {
					t.Error("expected output to contain Active")
				}
			},
		},
		{
			name: "multiple users",
			users: []*api.ListUsersUsersUserConnectionNodesUser{
				{
					Name:        "Jane Doe",
					DisplayName: "jane",
					Email:       "jane@example.com",
					Active:      true,
					Admin:       true,
				},
				{
					Name:        "John Smith",
					DisplayName: "john",
					Email:       "john@example.com",
					Active:      false,
					Admin:       false,
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Jane Doe") {
					t.Error("expected output to contain Jane Doe")
				}
				if !strings.Contains(got, "John Smith") {
					t.Error("expected output to contain John Smith")
				}
				if !strings.Contains(got, "Admin") {
					t.Error("expected output to contain Admin")
				}
				if !strings.Contains(got, "Member") {
					t.Error("expected output to contain Member")
				}
				if !strings.Contains(got, "Active") {
					t.Error("expected output to contain Active")
				}
				if !strings.Contains(got, "Disabled") {
					t.Error("expected output to contain Disabled")
				}
				lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
				if len(lines) != 3 {
					t.Errorf("expected 3 lines (header + 2 users), got %d", len(lines))
				}
			},
		},
		{
			name: "with color enabled",
			users: []*api.ListUsersUsersUserConnectionNodesUser{
				{
					Name:        "Jane Doe",
					DisplayName: "jane",
					Email:       "jane@example.com",
					Active:      true,
					Admin:       true,
				},
			},
			color: true,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, format.Bold) {
					t.Error("expected bold ANSI codes in header")
				}
				if !strings.Contains(got, format.Green) {
					t.Error("expected green ANSI code for active status")
				}
			},
		},
		{
			name: "disabled user has red color",
			users: []*api.ListUsersUsersUserConnectionNodesUser{
				{
					Name:        "John Smith",
					DisplayName: "john",
					Email:       "john@example.com",
					Active:      false,
					Admin:       false,
				},
			},
			color: true,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, format.Red) {
					t.Error("expected red ANSI code for disabled status")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatUserList(tt.users, tt.color)
			tt.checks(t, got)
		})
	}
}

func TestFormatUserDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		user   *api.GetUserByDisplayNameUsersUserConnectionNodesUser
		color  bool
		checks func(t *testing.T, got string)
	}{
		{
			name: "active admin user",
			user: &api.GetUserByDisplayNameUsersUserConnectionNodesUser{
				Name:        "Jane Doe",
				DisplayName: "jane",
				Email:       "jane@example.com",
				Active:      true,
				Admin:       true,
				IsMe:        false,
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				expectations := []string{
					"Name: Jane Doe",
					"Display Name: jane",
					"Email: jane@example.com",
					"Role: Admin",
					"Status: Active",
				}
				for _, exp := range expectations {
					if !strings.Contains(got, exp) {
						t.Errorf("expected output to contain %q", exp)
					}
				}
				if strings.Contains(got, "Is Me:") {
					t.Error("expected no Is Me field when IsMe is false")
				}
			},
		},
		{
			name: "disabled member user",
			user: &api.GetUserByDisplayNameUsersUserConnectionNodesUser{
				Name:        "John Smith",
				DisplayName: "john",
				Email:       "john@example.com",
				Active:      false,
				Admin:       false,
				IsMe:        false,
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Role: Member") {
					t.Error("expected Role: Member")
				}
				if !strings.Contains(got, "Status: Disabled") {
					t.Error("expected Status: Disabled")
				}
			},
		},
		{
			name: "is me user",
			user: &api.GetUserByDisplayNameUsersUserConnectionNodesUser{
				Name:        "Fred",
				DisplayName: "fred",
				Email:       "fred@example.com",
				Active:      true,
				Admin:       false,
				IsMe:        true,
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Is Me: Yes") {
					t.Error("expected Is Me: Yes")
				}
			},
		},
		{
			name: "with color enabled",
			user: &api.GetUserByDisplayNameUsersUserConnectionNodesUser{
				Name:        "Jane Doe",
				DisplayName: "jane",
				Email:       "jane@example.com",
				Active:      true,
				Admin:       true,
				IsMe:        false,
			},
			color: true,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, format.Bold) {
					t.Error("expected bold ANSI codes in field labels")
				}
				if !strings.Contains(got, format.Green) {
					t.Error("expected green ANSI code for active status")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatUserDetail(tt.user, tt.color)
			tt.checks(t, got)
		})
	}
}
