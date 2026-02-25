package cmd

// ShellQuote is an exported wrapper for testing.
func ShellQuote(s string) string {
	return shellQuote(s)
}

// BuildFzfReloadCmd is an exported wrapper for testing.
func BuildFzfReloadCmd(self, cycle, statusFilter, labelFilter, user, sortBy, columnFlag string, limit int) string {
	return buildFzfReloadCmd(self, cycle, statusFilter, labelFilter, user, sortBy, columnFlag, limit)
}
