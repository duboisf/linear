package keyring_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

// --- Helper process for subprocess-based exec.Command mocking ---

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("GO_HELPER_STDOUT"))
	code, _ := strconv.Atoi(os.Getenv("GO_HELPER_EXIT_CODE"))
	os.Exit(code)
}

func fakeCommandRunner(stdout string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_STDOUT="+stdout,
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
		)
		return cmd
	}
}

// --- Mock provider for testing ChainProvider and Resolve ---

type mockProvider struct {
	getKey   string
	getErr   error
	storeErr error
	stored   string
}

func (m *mockProvider) GetAPIKey() (string, error) {
	return m.getKey, m.getErr
}

func (m *mockProvider) StoreAPIKey(key string) error {
	m.stored = key
	return m.storeErr
}

var _ keyring.Provider = (*mockProvider)(nil)

// --- Mock prompter ---

type mockPrompter struct {
	key string
	err error
}

func (m *mockPrompter) PromptForAPIKey(_ io.Reader, _ io.Writer) (string, error) {
	return m.key, m.err
}

var _ keyring.Prompter = (*mockPrompter)(nil)
