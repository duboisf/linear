package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
	"github.com/duboisf/linear/internal/keyring"
)

// Options holds injectable dependencies for all commands.
type Options struct {
	// NewAPIClient creates a GraphQL client from an API key.
	NewAPIClient func(apiKey string) graphql.Client
	// KeyringProvider resolves API keys.
	KeyringProvider keyring.Provider
	// Prompter handles interactive API key prompts.
	Prompter keyring.Prompter
	// NativeStore is the platform-specific credential store.
	NativeStore keyring.Provider
	// FileStore is the file-based fallback credential store.
	FileStore keyring.Provider
	// GitWorktreeCreator abstracts git worktree operations.
	GitWorktreeCreator GitWorktreeCreator
	// Cache provides file-based caching for issue details.
	Cache *cache.Cache
	// Stdin for interactive input.
	Stdin io.Reader
	// Stdout for command output.
	Stdout io.Writer
	// Stderr for error output.
	Stderr io.Writer
}

// NewRootCmd creates the root cobra command with all subcommands wired up.
func NewRootCmd(opts Options) *cobra.Command {
	root := &cobra.Command{
		Use:           "linear",
		Short:         "CLI for the Linear issue tracker",
		SilenceUsage:  true,
		SilenceErrors: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)

	root.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands:"},
		&cobra.Group{ID: "setup", Title: "Setup Commands:"},
	)

	createCmd := newCreateCmd(opts)
	createCmd.GroupID = "core"
	issueCmd := newIssueCmd(opts)
	issueCmd.GroupID = "core"
	userCmd := newUserCmd(opts)
	userCmd.GroupID = "core"

	cacheCmd := newCacheCmd(opts)
	cacheCmd.GroupID = "setup"
	completionCmd := newCompletionCmd()
	completionCmd.GroupID = "setup"
	versionCmd := newVersionCmd()
	versionCmd.GroupID = "setup"

	root.SetHelpCommand(&cobra.Command{Hidden: true})

	root.AddCommand(
		createCmd,
		issueCmd,
		userCmd,
		cacheCmd,
		completionCmd,
		versionCmd,
	)
	return root
}

// Execute creates the root command with default options and runs it.
func Execute() error {
	opts := DefaultOptions()
	return NewRootCmd(opts).ExecuteContext(context.Background())
}

// nativeKeyringProvider returns the platform-specific keyring provider.
func nativeKeyringProvider() keyring.Provider {
	switch runtime.GOOS {
	case "darwin":
		return &keyring.KeychainProvider{}
	default:
		return &keyring.SecretToolProvider{}
	}
}

// DefaultOptions returns production-ready Options with platform-appropriate
// keyring, standard I/O, and the default API client.
func DefaultOptions() Options {
	native := nativeKeyringProvider()
	file := &keyring.FileProvider{}
	cacheDir := filepath.Join(os.TempDir(), "linear-cache")
	if d, err := os.UserCacheDir(); err == nil {
		cacheDir = filepath.Join(d, "linear")
	}
	return Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return api.NewClient(apiKey, "")
		},
		KeyringProvider: &keyring.ChainProvider{
			Providers: []keyring.Provider{
				&keyring.EnvProvider{},
				native,
				file,
			},
		},
		Prompter:           &keyring.InteractivePrompter{},
		NativeStore:        native,
		FileStore:          file,
		GitWorktreeCreator: &execGitWorktreeCreator{ctx: context.Background()},
		Cache:              cache.New(cacheDir, 5*time.Minute),
		Stdin:              os.Stdin,
		Stdout:             os.Stdout,
		Stderr:             os.Stderr,
	}
}

// resolveClient resolves an API key and returns an authenticated GraphQL client.
func resolveClient(cmd *cobra.Command, opts Options) (graphql.Client, error) {
	apiKey, err := keyring.Resolve(keyring.ResolveOptions{
		Provider:    opts.KeyringProvider,
		Prompter:    opts.Prompter,
		NativeStore: opts.NativeStore,
		FileStore:   opts.FileStore,
		Stdin:       opts.Stdin,
		MsgWriter:   opts.Stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving API key: %w", err)
	}
	return opts.NewAPIClient(apiKey), nil
}
