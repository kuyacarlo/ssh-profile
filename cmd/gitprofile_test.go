package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matryer/is"
)

// fakeRunner records git-profile invocations.
type fakeRunner struct {
	available bool
	addCalls  int
	useCalls  []string
}

func (f *fakeRunner) Available() bool { return f.available }

func (f *fakeRunner) RunAdd(in io.Reader, out, errOut io.Writer) error {
	f.addCalls++
	return nil
}

func (f *fakeRunner) RunUse(name string, in io.Reader, out, errOut io.Writer) error {
	f.useCalls = append(f.useCalls, name)
	return nil
}

// forceTerminal builds a Root whose git-profile offer always runs (terminal
// path forced, confirm answered by the given function).
func forceTerminal(t *testing.T, runner *fakeRunner, confirm func(string) bool) *Root {
	t.Helper()
	root := New()
	root.gitProfile = runner
	root.isTerminal = func(io.Reader) bool { return true }
	root.confirm = func(title, affirmative, negative string, in io.Reader, out io.Writer) (bool, error) {
		return confirm(title), nil
	}
	return root
}

func TestOfferGitProfileAddNotOfferedWhenUnavailable(t *testing.T) {
	is := is.New(t)
	runner := &fakeRunner{available: false}
	root := forceTerminal(t, runner, func(string) bool { return true })

	cfg := filepath.Join(t.TempDir(), "config.json")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-c", cfg, "add", "alice"})
	root.Execute()
	is.Equal(runner.addCalls, 0)
}

func TestOfferGitProfileAddRunsOnConfirm(t *testing.T) {
	is := is.New(t)
	runner := &fakeRunner{available: true}
	root := forceTerminal(t, runner, func(string) bool { return true })

	cfg := filepath.Join(t.TempDir(), "config.json")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-c", cfg, "add", "alice"})
	root.Execute()
	is.Equal(runner.addCalls, 1)
	is.True(strings.Contains(out.String(), "Running `git-profile add`"))
}

func TestOfferGitProfileAddSkipsOnDecline(t *testing.T) {
	is := is.New(t)
	runner := &fakeRunner{available: true}
	root := forceTerminal(t, runner, func(string) bool { return false })

	cfg := filepath.Join(t.TempDir(), "config.json")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-c", cfg, "add", "alice"})
	root.Execute()
	is.Equal(runner.addCalls, 0)
}

func TestOfferGitProfileUseRunsOnConfirm(t *testing.T) {
	is := is.New(t)
	runner := &fakeRunner{available: true}
	root := forceTerminal(t, runner, func(string) bool { return true })

	cfg := filepath.Join(t.TempDir(), "config.json")
	key := writeTestKey(t, t.TempDir())
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-c", cfg, "add", "alice", "--identity", key, "--github-user", "alice"})
	root.Execute()
	is.Equal(runner.addCalls, 1)

	repo := filepath.Join(t.TempDir(), "demo-repo")
	is.NoErr(os.MkdirAll(repo, 0o755))
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	wd, err := os.Getwd()
	is.NoErr(err)
	is.NoErr(os.Chdir(repo))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	root = forceTerminal(t, runner, func(string) bool { return true })
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-c", cfg, "use", "alice"})
	root.Execute()
	is.Equal(len(runner.useCalls), 1)
	is.Equal(runner.useCalls[0], "alice")
}
