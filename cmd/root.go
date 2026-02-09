package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
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
	// StoreProvider stores API keys after prompting.
	StoreProvider keyring.Provider
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
	}
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)
	root.AddCommand(
		newIssueCmd(opts),
		newCompletionCmd(),
	)
	return root
}

// Execute creates the root command with default options and runs it.
func Execute() error {
	opts := defaultOptions()
	return NewRootCmd(opts).ExecuteContext(context.Background())
}

func defaultOptions() Options {
	return Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return api.NewClient(apiKey, "")
		},
		KeyringProvider: &keyring.ChainProvider{
			Providers: []keyring.Provider{
				&keyring.EnvProvider{},
				&keyring.SecretToolProvider{},
			},
		},
		Prompter:      &keyring.InteractivePrompter{},
		StoreProvider: &keyring.SecretToolProvider{},
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	}
}

// resolveClient resolves an API key and returns an authenticated GraphQL client.
func resolveClient(cmd *cobra.Command, opts Options) (graphql.Client, error) {
	apiKey, err := keyring.Resolve(opts.KeyringProvider, opts.Prompter, opts.StoreProvider, opts.Stdin, opts.Stderr)
	if err != nil {
		return nil, fmt.Errorf("resolving API key: %w", err)
	}
	return opts.NewAPIClient(apiKey), nil
}
