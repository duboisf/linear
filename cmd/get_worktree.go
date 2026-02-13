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
	BranchExists(branch string) (bool, error)
	FetchBranch(remote, branch string) error
	CreateWorktree(path, branch, startPoint string) error
	PostCreate(dir string) error
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

func (g *execGitWorktreeCreator) BranchExists(branch string) (bool, error) {
	err := exec.CommandContext(g.ctx, "git", "rev-parse", "--verify", "refs/heads/"+branch).Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false, nil
	}
	return false, fmt.Errorf("checking branch %q: %w", branch, err)
}

func (g *execGitWorktreeCreator) FetchBranch(remote, branch string) error {
	if err := exec.CommandContext(g.ctx, "git", "fetch", remote, branch).Run(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", remote, branch, err)
	}
	return nil
}

func (g *execGitWorktreeCreator) CreateWorktree(path, branch, startPoint string) error {
	var args []string
	if startPoint == "" {
		args = []string{"worktree", "add", path, branch}
	} else {
		args = []string{"worktree", "add", "-b", branch, path, startPoint}
	}
	if err := exec.CommandContext(g.ctx, "git", args...).Run(); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	return nil
}

func (g *execGitWorktreeCreator) PostCreate(dir string) error {
	if _, err := exec.LookPath("mise"); err != nil {
		return nil
	}
	cmd := exec.CommandContext(g.ctx, "mise", "trust")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running mise trust: %w", err)
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

	worktreePath := filepath.Join(filepath.Dir(repoRoot), strings.ToLower(identifier), filepath.Base(repoRoot))

	exists, err := git.BranchExists(branchName)
	if err != nil {
		return err
	}

	if exists {
		if err := git.CreateWorktree(worktreePath, branchName, ""); err != nil {
			return err
		}
		fmt.Fprintf(w, "Reusing existing branch %q\n", branchName)
	} else {
		if err := git.FetchBranch("origin", "main"); err != nil {
			return err
		}
		if err := git.CreateWorktree(worktreePath, branchName, "origin/main"); err != nil {
			return err
		}
		fmt.Fprintf(w, "Created new branch %q from origin/main\n", branchName)
	}

	if err := git.PostCreate(worktreePath); err != nil {
		return err
	}

	fmt.Fprintln(w, worktreePath)
	return nil
}
