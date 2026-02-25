package cmd

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
	"github.com/duboisf/linear/internal/format"
)

// editableField defines a field that can be edited interactively.
type editableField struct {
	Name    string
	Current string
}

// newIssueEditInteractiveCmd creates the hidden "issue edit-interactive"
// subcommand used by the fzf ctrl-e binding. It launches nested fzf pickers
// for field selection and value selection, then updates the issue via the API.
func newIssueEditInteractiveCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "edit-interactive [IDENTIFIER]",
		Short:  "Interactively edit an issue (used by fzf binding)",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			identifier := args[0]
			resp, err := api.GetIssue(cmd.Context(), client, identifier)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}
			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", identifier)
			}

			issue := resp.Issue

			field, err := fzfPickField(issue)
			if err != nil {
				return err
			}
			if field == "" {
				return nil // user cancelled
			}

			timeNow := opts.TimeNow
			if timeNow == nil {
				timeNow = time.Now
			}

			result, err := applyFieldEdit(cmd.Context(), client, opts.Cache, timeNow, issue, field)
			if err != nil {
				return err
			}
			if result == "" {
				return nil // user cancelled value picker
			}

			// Refresh cache so the fzf preview shows updated data.
			if opts.Cache != nil {
				refreshIssueCache(cmd.Context(), client, opts.Cache, identifier)
			}

			colorEnabled := format.ColorEnabled(os.Stderr)
			issueID := format.Colorize(colorEnabled, format.Bold, identifier)
			fmt.Fprintf(os.Stderr, "Updated %s: %s\n", issueID, result)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	return cmd
}

// fzfPickField presents a field picker for the given issue and returns the
// selected field name. Returns empty string if the user cancelled.
func fzfPickField(issue *api.GetIssueIssue) (string, error) {
	fields := buildEditableFields(issue)

	// Compute max name width for alignment.
	maxName := 0
	for _, f := range fields {
		if len(f.Name) > maxName {
			maxName = len(f.Name)
		}
	}

	lines := make([]string, len(fields))
	for i, f := range fields {
		lines[i] = fmt.Sprintf("%-*s  %s", maxName, f.Name, f.Current)
	}

	input := strings.Join(lines, "\n") + "\n"

	cmd := exec.Command("fzf",
		"--ansi",
		"--no-sort",
		"--layout=reverse",
		"--header", fmt.Sprintf("Edit %s: pick a field", issue.Identifier),
		"--header-first",
	)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = nil // let fzf use /dev/tty

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if fzfExitOK(err) {
			return "", nil
		}
		return "", fmt.Errorf("running fzf field picker: %w", err)
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return "", nil
	}
	// Extract the field name (first whitespace-delimited token).
	return strings.Fields(selected)[0], nil
}

// buildEditableFields returns the list of editable fields with their current values.
func buildEditableFields(issue *api.GetIssueIssue) []editableField {
	currentState := "None"
	if issue.State != nil {
		currentState = issue.State.Name
	}

	currentCycle := "None"
	if issue.Cycle != nil {
		currentCycle = fmt.Sprintf("Cycle %.0f", issue.Cycle.Number)
		if issue.Cycle.Name != nil && *issue.Cycle.Name != "" {
			currentCycle += " - " + *issue.Cycle.Name
		}
	}

	currentPriority := format.PriorityLabel(issue.Priority)

	currentLabels := "None"
	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		names := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			names[i] = l.Name
		}
		currentLabels = strings.Join(names, ", ")
	}

	currentAssignee := "Unassigned"
	if issue.Assignee != nil {
		currentAssignee = issue.Assignee.Name
	}

	currentProject := "None"
	if issue.Project != nil {
		currentProject = issue.Project.Name
	}

	fields := []editableField{
		{Name: "Status", Current: currentState},
		{Name: "Priority", Current: currentPriority},
		{Name: "Cycle", Current: currentCycle},
		{Name: "Assignee", Current: currentAssignee},
		{Name: "Project", Current: currentProject},
		{Name: "Title", Current: truncate(issue.Title, 50)},
		{Name: "Description", Current: "(opens $EDITOR)"},
	}

	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		fields = slices.Insert(fields, 3,
			editableField{Name: "Labels-Add", Current: currentLabels},
			editableField{Name: "Labels-Remove", Current: currentLabels},
		)
	} else {
		fields = slices.Insert(fields, 3,
			editableField{Name: "Labels-Add", Current: currentLabels},
		)
	}

	return fields
}

// truncate shortens s to maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// applyFieldEdit presents the appropriate value picker for the given field,
// performs the API update, and returns a human-readable result string.
// Returns empty string if the user cancelled.
func applyFieldEdit(ctx context.Context, client graphql.Client, c *cache.Cache, timeNow func() time.Time, issue *api.GetIssueIssue, field string) (string, error) {
	switch field {
	case "Status":
		return editStatus(ctx, client, issue)
	case "Priority":
		return editPriority(ctx, client, issue)
	case "Cycle":
		return editCycle(ctx, client, c, timeNow, issue)
	case "Labels-Add":
		return editLabelsAdd(ctx, client, c, issue)
	case "Labels-Remove":
		return editLabelsRemove(ctx, client, issue)
	case "Assignee":
		return editAssignee(ctx, client, c, issue)
	case "Project":
		return editProject(ctx, client, issue)
	case "Title":
		return editTitle(ctx, client, issue)
	case "Description":
		return editDescription(ctx, client, issue)
	default:
		return "", fmt.Errorf("unknown field %q", field)
	}
}

// editStatus presents a workflow state picker and updates the issue.
func editStatus(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	if issue.Team == nil {
		return "", fmt.Errorf("issue has no team")
	}

	resp, err := api.ListWorkflowStates(ctx, client, 50, issue.Team.Id)
	if err != nil {
		return "", fmt.Errorf("listing workflow states: %w", err)
	}
	if resp.WorkflowStates == nil || len(resp.WorkflowStates.Nodes) == 0 {
		return "", fmt.Errorf("no workflow states found")
	}

	// Group by type, sort within groups by position.
	typeOrder := []string{"started", "unstarted", "triage", "backlog", "completed", "canceled"}
	grouped := make(map[string][]*api.ListWorkflowStatesWorkflowStatesWorkflowStateConnectionNodesWorkflowState)
	for _, s := range resp.WorkflowStates.Nodes {
		grouped[s.Type] = append(grouped[s.Type], s)
	}
	for _, states := range grouped {
		slices.SortFunc(states, func(a, b *api.ListWorkflowStatesWorkflowStatesWorkflowStateConnectionNodesWorkflowState) int {
			return cmp.Compare(a.Position, b.Position)
		})
	}

	var lines []string
	currentStateID := ""
	if issue.State != nil {
		currentStateID = issue.State.Id
	}

	for _, stateType := range typeOrder {
		states := grouped[stateType]
		if len(states) == 0 {
			continue
		}
		for _, s := range states {
			marker := "  "
			if s.Id == currentStateID {
				marker = "* "
			}
			color := format.StateColor(s.Type)
			line := fmt.Sprintf("%s%s%s%s  %s", s.Id, "\t", marker, format.Colorize(true, color, s.Name), format.Colorize(true, format.Gray, s.Type))
			lines = append(lines, line)
		}
	}

	selected, err := fzfPickValue("Select status", lines, true)
	if err != nil || selected == "" {
		return "", err
	}

	// Extract state ID (first tab-delimited field).
	stateID, _, _ := strings.Cut(selected, "\t")

	input := &api.IssueUpdateInput{StateId: &stateID}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating status: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("status update was not successful")
	}

	// Find the name of the selected state.
	stateName := stateID
	for _, s := range resp.WorkflowStates.Nodes {
		if s.Id == stateID {
			stateName = s.Name
			break
		}
	}
	return fmt.Sprintf("status → %s", stateName), nil
}

// editPriority presents a priority picker and updates the issue.
func editPriority(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	priorities := []struct {
		Value int
		Label string
	}{
		{1, "Urgent"},
		{2, "High"},
		{3, "Normal"},
		{4, "Low"},
		{0, "No priority"},
	}

	currentPriority := int(issue.Priority)

	var lines []string
	for _, p := range priorities {
		marker := "  "
		if p.Value == currentPriority {
			marker = "* "
		}
		color := format.PriorityColor(float64(p.Value))
		lines = append(lines, fmt.Sprintf("%d\t%s%s", p.Value, marker, format.Colorize(true, color, p.Label)))
	}

	selected, err := fzfPickValue("Select priority", lines, true)
	if err != nil || selected == "" {
		return "", err
	}

	// Extract priority value (first tab-delimited field).
	valStr, _, _ := strings.Cut(selected, "\t")
	val, _ := strconv.Atoi(valStr)

	input := &api.IssueUpdateInput{Priority: &val}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating priority: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("priority update was not successful")
	}

	label := format.PriorityLabel(float64(val))
	return fmt.Sprintf("priority → %s", label), nil
}

// editCycle presents a cycle picker and updates the issue.
func editCycle(ctx context.Context, client graphql.Client, c *cache.Cache, timeNow func() time.Time, issue *api.GetIssueIssue) (string, error) {
	resp, err := listCyclesCached(ctx, client, c, timeNow)
	if err != nil {
		return "", fmt.Errorf("listing cycles: %w", err)
	}
	if resp.Cycles == nil || len(resp.Cycles.Nodes) == 0 {
		return "", fmt.Errorf("no cycles found")
	}

	currentCycleID := ""
	if issue.Cycle != nil {
		currentCycleID = issue.Cycle.Id
	}

	var lines []string
	for _, c := range resp.Cycles.Nodes {
		if c.IsPast && !c.IsPrevious {
			continue // skip old past cycles
		}

		marker := "  "
		if c.Id == currentCycleID {
			marker = "* "
		}

		label := fmt.Sprintf("#%.0f", c.Number)
		if c.Name != nil && *c.Name != "" {
			label += " " + *c.Name
		}

		var status string
		switch {
		case c.IsActive:
			status = format.Colorize(true, format.Green, "Active")
		case c.IsNext:
			status = format.Colorize(true, format.Yellow, "Next")
		case c.IsPrevious:
			status = format.Colorize(true, format.Gray, "Previous")
		case c.IsFuture:
			status = format.Colorize(true, format.Cyan, "Upcoming")
		}

		dates := formatCycleDateRange(c.StartsAt, c.EndsAt)
		if dates != "" {
			label += "  " + format.Colorize(true, format.Gray, dates)
		}

		lines = append(lines, fmt.Sprintf("%s\t%s%s  %s", c.Id, marker, label, status))
	}

	// Add "None" option to remove cycle.
	noneMarker := "  "
	if currentCycleID == "" {
		noneMarker = "* "
	}
	lines = append(lines, fmt.Sprintf("none\t%s%s", noneMarker, format.Colorize(true, format.Gray, "No cycle")))

	selected, err := fzfPickValue("Select cycle", lines, true)
	if err != nil || selected == "" {
		return "", err
	}

	cycleID, _, _ := strings.Cut(selected, "\t")

	if cycleID == "none" {
		// Remove cycle — set to empty string to unset.
		emptyStr := ""
		input := &api.IssueUpdateInput{CycleId: &emptyStr}
		updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
		if err != nil {
			return "", fmt.Errorf("removing cycle: %w", err)
		}
		if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
			return "", fmt.Errorf("cycle removal was not successful")
		}
		return "cycle → None", nil
	}

	input := &api.IssueUpdateInput{CycleId: &cycleID}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating cycle: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("cycle update was not successful")
	}

	// Find cycle name for display.
	cycleName := cycleID
	for _, c := range resp.Cycles.Nodes {
		if c.Id == cycleID {
			cycleName = fmt.Sprintf("Cycle %.0f", c.Number)
			if c.Name != nil && *c.Name != "" {
				cycleName += " - " + *c.Name
			}
			break
		}
	}
	return fmt.Sprintf("cycle → %s", cycleName), nil
}

// editLabelsAdd presents labels not on the issue for multi-selection and adds them.
func editLabelsAdd(ctx context.Context, client graphql.Client, c *cache.Cache, issue *api.GetIssueIssue) (string, error) {
	resp, err := labelsCached(ctx, client, c)
	if err != nil {
		return "", fmt.Errorf("listing labels: %w", err)
	}
	if resp.IssueLabels == nil || len(resp.IssueLabels.Nodes) == 0 {
		return "", fmt.Errorf("no labels found")
	}

	// Build set of current label IDs.
	currentIDs := make(map[string]bool)
	if issue.Labels != nil {
		for _, l := range issue.Labels.Nodes {
			currentIDs[l.Id] = true
		}
	}

	var lines []string
	for _, l := range resp.IssueLabels.Nodes {
		if currentIDs[l.Id] {
			continue // skip labels already on the issue
		}
		lines = append(lines, fmt.Sprintf("%s\t%s", l.Id, l.Name))
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("all labels are already assigned to the issue")
	}

	selected, err := fzfPickMultiValue("Add labels (TAB to select, ENTER to confirm)", lines, true)
	if err != nil || len(selected) == 0 {
		return "", err
	}

	var addIDs []string
	var addNames []string
	for _, line := range selected {
		id, name, _ := strings.Cut(line, "\t")
		addIDs = append(addIDs, id)
		addNames = append(addNames, name)
	}

	input := &api.IssueUpdateInput{AddedLabelIds: addIDs}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("adding labels: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("label add was not successful")
	}

	return fmt.Sprintf("labels added: %s", strings.Join(addNames, ", ")), nil
}

// editLabelsRemove presents labels on the issue for multi-selection and removes them.
func editLabelsRemove(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	if issue.Labels == nil || len(issue.Labels.Nodes) == 0 {
		return "", fmt.Errorf("issue has no labels to remove")
	}

	var lines []string
	for _, l := range issue.Labels.Nodes {
		lines = append(lines, fmt.Sprintf("%s\t%s", l.Id, l.Name))
	}

	selected, err := fzfPickMultiValue("Remove labels (TAB to select, ENTER to confirm)", lines, true)
	if err != nil || len(selected) == 0 {
		return "", err
	}

	var removeIDs []string
	var removeNames []string
	for _, line := range selected {
		id, name, _ := strings.Cut(line, "\t")
		removeIDs = append(removeIDs, id)
		removeNames = append(removeNames, name)
	}

	input := &api.IssueUpdateInput{RemovedLabelIds: removeIDs}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("removing labels: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("label removal was not successful")
	}

	return fmt.Sprintf("labels removed: %s", strings.Join(removeNames, ", ")), nil
}

// editAssignee presents a user picker and updates the issue.
func editAssignee(ctx context.Context, client graphql.Client, c *cache.Cache, issue *api.GetIssueIssue) (string, error) {
	resp, err := usersForCompletionCached(ctx, client, c)
	if err != nil {
		return "", fmt.Errorf("listing users: %w", err)
	}
	if resp.Users == nil || len(resp.Users.Nodes) == 0 {
		return "", fmt.Errorf("no users found")
	}

	currentAssigneeID := ""
	if issue.Assignee != nil {
		currentAssigneeID = issue.Assignee.Id
	}

	var lines []string
	// Add "Unassigned" option.
	unassignMarker := "  "
	if currentAssigneeID == "" {
		unassignMarker = "* "
	}
	lines = append(lines, fmt.Sprintf("none\t%s%s", unassignMarker, format.Colorize(true, format.Gray, "Unassigned")))

	for _, u := range resp.Users.Nodes {
		marker := "  "
		if u.Id == currentAssigneeID {
			marker = "* "
		}
		lines = append(lines, fmt.Sprintf("%s\t%s%s", u.Id, marker, u.DisplayName))
	}

	selected, err := fzfPickValue("Select assignee", lines, true)
	if err != nil || selected == "" {
		return "", err
	}

	userID, _, _ := strings.Cut(selected, "\t")

	if userID == "none" {
		// Unassign.
		emptyStr := ""
		input := &api.IssueUpdateInput{AssigneeId: &emptyStr}
		updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
		if err != nil {
			return "", fmt.Errorf("unassigning: %w", err)
		}
		if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
			return "", fmt.Errorf("unassign was not successful")
		}
		return "assignee → Unassigned", nil
	}

	input := &api.IssueUpdateInput{AssigneeId: &userID}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating assignee: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("assignee update was not successful")
	}

	// Find user name for display.
	userName := userID
	for _, u := range resp.Users.Nodes {
		if u.Id == userID {
			userName = u.DisplayName
			break
		}
	}
	return fmt.Sprintf("assignee → %s", userName), nil
}

// editProject presents a project picker and updates the issue.
func editProject(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	resp, err := api.ListProjects(ctx, client, 50)
	if err != nil {
		return "", fmt.Errorf("listing projects: %w", err)
	}
	if resp.Projects == nil || len(resp.Projects.Nodes) == 0 {
		return "", fmt.Errorf("no projects found")
	}

	currentProjectID := ""
	if issue.Project != nil {
		currentProjectID = issue.Project.Id
	}

	var lines []string
	// Add "None" option.
	noneMarker := "  "
	if currentProjectID == "" {
		noneMarker = "* "
	}
	lines = append(lines, fmt.Sprintf("none\t%s%s", noneMarker, format.Colorize(true, format.Gray, "No project")))

	for _, p := range resp.Projects.Nodes {
		marker := "  "
		if p.Id == currentProjectID {
			marker = "* "
		}
		lines = append(lines, fmt.Sprintf("%s\t%s%s", p.Id, marker, p.Name))
	}

	selected, err := fzfPickValue("Select project", lines, true)
	if err != nil || selected == "" {
		return "", err
	}

	projectID, _, _ := strings.Cut(selected, "\t")

	if projectID == "none" {
		emptyStr := ""
		input := &api.IssueUpdateInput{ProjectId: &emptyStr}
		updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
		if err != nil {
			return "", fmt.Errorf("removing project: %w", err)
		}
		if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
			return "", fmt.Errorf("project removal was not successful")
		}
		return "project → None", nil
	}

	input := &api.IssueUpdateInput{ProjectId: &projectID}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating project: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("project update was not successful")
	}

	projectName := projectID
	for _, p := range resp.Projects.Nodes {
		if p.Id == projectID {
			projectName = p.Name
			break
		}
	}
	return fmt.Sprintf("project → %s", projectName), nil
}

// editTitle opens $EDITOR to edit the issue title.
func editTitle(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	newTitle, err := editInEditor(issue.Title, "linear-title-*.txt")
	if err != nil {
		return "", err
	}
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" || newTitle == issue.Title {
		return "", nil // empty or no change
	}

	input := &api.IssueUpdateInput{Title: &newTitle}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating title: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("title update was not successful")
	}

	return fmt.Sprintf("title → %s", truncate(newTitle, 50)), nil
}

// editDescription opens $EDITOR to edit the issue description.
func editDescription(ctx context.Context, client graphql.Client, issue *api.GetIssueIssue) (string, error) {
	desc := ""
	if issue.Description != nil {
		desc = *issue.Description
	}

	newDesc, err := editInEditor(desc, "linear-description-*.md")
	if err != nil {
		return "", err
	}
	if newDesc == desc {
		return "", nil // no change
	}

	input := &api.IssueUpdateInput{Description: &newDesc}
	updateResp, err := api.UpdateIssue(ctx, client, issue.Id, input)
	if err != nil {
		return "", fmt.Errorf("updating description: %w", err)
	}
	if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
		return "", fmt.Errorf("description update was not successful")
	}

	return "description updated", nil
}

// fzfPickValue runs fzf for single-value selection. Lines are tab-delimited
// with the ID in the first field (hidden via --with-nth=2..). Returns the
// full selected line or empty string if cancelled.
func fzfPickValue(header string, lines []string, hideIDs bool) (string, error) {
	input := strings.Join(lines, "\n") + "\n"

	args := []string{
		"--ansi",
		"--no-sort",
		"--layout=reverse",
		"--header", header,
		"--header-first",
		"--delimiter", "\t",
	}
	if hideIDs {
		args = append(args, "--with-nth=2..")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = nil

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if fzfExitOK(err) {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// fzfPickMultiValue runs fzf for multi-value selection. Lines are tab-delimited
// with the ID in the first field. Returns the selected lines or nil if cancelled.
func fzfPickMultiValue(header string, lines []string, hideIDs bool) ([]string, error) {
	input := strings.Join(lines, "\n") + "\n"

	args := []string{
		"--ansi",
		"--no-sort",
		"--layout=reverse",
		"--multi",
		"--header", header,
		"--header-first",
		"--delimiter", "\t",
	}
	if hideIDs {
		args = append(args, "--with-nth=2..")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = nil

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if fzfExitOK(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("running fzf: %w", err)
	}

	raw := strings.TrimSpace(out.String())
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// editorCmd returns the editor command to use for editing text.
func editorCmd() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}

// editInEditor opens the given content in $EDITOR and returns the edited content.
// Returns empty string if the user saves an empty file or makes no changes.
func editInEditor(content, pattern string) (string, error) {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	parts := strings.Fields(editorCmd())
	cmd := exec.Command(parts[0], append(parts[1:], tmpFile.Name())...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor: %w", err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}

	return string(edited), nil
}
