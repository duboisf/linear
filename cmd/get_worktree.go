package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Khan/genqlient/graphql"

	"github.com/duboisf/linear/internal/api"
)

// GitWorktreeCreator abstracts git operations for creating worktrees.
type GitWorktreeCreator interface {
	RepoRootDir() (string, error)
	FetchBranch(remote, branch string) error
	CreateWorktree(path, branch, startPoint string) error
}

// execGitWorktreeCreator implements GitWorktreeCreator using os/exec.
type execGitWorktreeCreator struct {
	ctx context.Context
}

func (g *execGitWorktreeCreator) RepoRootDir() (string, error) {
	out, err := exec.CommandContext(g.ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("getting repo root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *execGitWorktreeCreator) FetchBranch(remote, branch string) error {
	if err := exec.CommandContext(g.ctx, "git", "fetch", remote, branch).Run(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", remote, branch, err)
	}
	return nil
}

func (g *execGitWorktreeCreator) CreateWorktree(path, branch, startPoint string) error {
	if err := exec.CommandContext(g.ctx, "git", "worktree", "add", "-b", branch, path, startPoint).Run(); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	return nil
}

// runWorktreeCreate fetches the issue's branch name and creates a git worktree.
func runWorktreeCreate(ctx context.Context, client graphql.Client, identifier string, git GitWorktreeCreator, w io.Writer) error {
	resp, err := api.GetIssue(ctx, client, identifier)
	if err != nil {
		return fmt.Errorf("getting issue: %w", err)
	}

	if resp.Issue == nil {
		return fmt.Errorf("issue %s not found", identifier)
	}

	branchName := resp.Issue.BranchName
	if branchName == "" {
		return fmt.Errorf("issue %s has no branch name", identifier)
	}

	repoRoot, err := git.RepoRootDir()
	if err != nil {
		return err
	}

	worktreePath := filepath.Join(filepath.Dir(repoRoot), filepath.Base(repoRoot)+"--"+strings.ToLower(identifier))

	if err := git.FetchBranch("origin", "main"); err != nil {
		return err
	}

	if err := git.CreateWorktree(worktreePath, branchName, "origin/main"); err != nil {
		return err
	}

	fmt.Fprintln(w, worktreePath)
	return nil
}
