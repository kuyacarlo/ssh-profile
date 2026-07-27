package git

import (
	"errors"
	"os/exec"
	"strings"
)

// gitConfigKeyNotFound is the exit code git returns from
// `git config --unset` when the key is not present.
const gitConfigKeyNotFound = 5
const gitRemoteKeyNotFound = 2

// Git is a vcs
type Git struct {
	exec func(name string, arg ...string) *exec.Cmd
}

// New initializes and returns a new Git
func New() *Git {
	return &Git{
		exec: exec.Command,
	}
}

// IsRepository checks that current directory is a git repository
func (g *Git) IsRepository() bool {
	_, err := g.exec("git", "rev-parse", "--git-dir").CombinedOutput()

	return err == nil
}

// Get returns the value stored in git local config
func (g *Git) Get(key string) (string, error) {
	out, err := g.exec("git", "config", "--local", key).CombinedOutput()

	return strings.TrimSpace(string(out)), err
}

// Set sets the value for a key in git local config
func (g *Git) Set(key string, value string) error {
	_, err := g.exec("git", "config", "--local", key, value).CombinedOutput()

	return err
}

// Unset removes a key from git local config.
// A missing key is not treated as an error.
func (g *Git) Unset(key string) error {
	_, err := g.exec("git", "config", "--local", "--unset", key).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == gitConfigKeyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// TopLevel returns the repository root path.
func (g *Git) TopLevel() (string, error) {
	out, err := g.exec("git", "rev-parse", "--show-toplevel").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (g *Git) GetRemote(key string) (string, error) {
	out, err := g.exec("git", "remote", "get-url", key).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// HasRemote reports whether a named remote exists.
func (g *Git) HasRemote(name string) bool {
	_, err := g.GetRemote(name)
	return err == nil
}

// AddRemote creates a new remote.
func (g *Git) AddRemote(name string, value string) error {
	_, err := g.exec("git", "remote", "add", name, value).CombinedOutput()
	return err
}

func (g *Git) SetRemote(key string, value string) error {
	_, err := g.exec("git", "remote", "set-url", key, value).CombinedOutput()

	return err
}

// EnsureRemote adds or updates a remote URL.
func (g *Git) EnsureRemote(name string, value string) error {
	if g.HasRemote(name) {
		return g.SetRemote(name, value)
	}
	return g.AddRemote(name, value)
}

func (g *Git) UnsetRemote(key string) error {
	_, err := g.exec("git", "remote", "remove", key).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == gitRemoteKeyNotFound {
			return nil
		}

		return err
	}

	return nil
}
