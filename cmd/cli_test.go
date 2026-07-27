package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func writeTestKey(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "id_ed25519")
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVersionCommand(t *testing.T) {
	is := is.New(t)
	root := New()
	root.Version = "9.9.9"
	root.CommitHash = "abc"
	root.CompileDate = "today"

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	is.NoErr(root.Command.Execute())
	is.True(strings.Contains(buf.String(), "git-ssh 9.9.9 (abc) today"))
}

func TestAddListExportMissing(t *testing.T) {
	is := is.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	cfg := filepath.Join(t.TempDir(), "config.json")
	key := writeTestKey(t, t.TempDir())

	run := func(args ...string) (string, error) {
		root := New()
		out := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		root.SetOut(out)
		root.SetErr(errBuf)
		root.SetArgs(append([]string{"-c", cfg}, args...))
		err := root.Command.Execute()
		return out.String() + errBuf.String(), err
	}

	out, err := run("add", "alice", "--identity", key, "--github-user", "alice")
	is.NoErr(err)
	is.True(strings.Contains(out, "Successfully added"))

	out, err = run("list")
	is.NoErr(err)
	is.True(strings.Contains(out, "alice"))

	out, err = run("export", "alice")
	is.NoErr(err)
	is.True(strings.Contains(out, `"identity_file"`))
	is.True(strings.Contains(out, `"github_user":"alice"`))

	_, err = run("export", "missing")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "no profile"))

	_, err = run("show", "missing")
	is.True(err != nil)
}

func TestUseRequiresGitRepo(t *testing.T) {
	is := is.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	cfg := filepath.Join(t.TempDir(), "config.json")
	key := writeTestKey(t, t.TempDir())

	root := New()
	root.SetArgs([]string{"-c", cfg, "add", "alice", "--identity", key, "--github-user", "alice"})
	is.NoErr(root.Command.Execute())

	workdir := t.TempDir()
	root = New()
	root.SetArgs([]string{"-c", cfg, "use", "alice"})
	// Execute from a non-repo directory.
	wd, err := os.Getwd()
	is.NoErr(err)
	is.NoErr(os.Chdir(workdir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	err = root.Command.Execute()
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "not a valid git repository"))
}

func TestUseAndCurrentInRepo(t *testing.T) {
	is := is.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	cfg := filepath.Join(t.TempDir(), "config.json")
	key := writeTestKey(t, t.TempDir())

	root := New()
	root.SetArgs([]string{"-c", cfg, "add", "alice", "--identity", key, "--github-user", "alice"})
	is.NoErr(root.Command.Execute())

	repo := filepath.Join(t.TempDir(), "demo-repo")
	is.NoErr(os.MkdirAll(repo, 0o755))
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	wd, err := os.Getwd()
	is.NoErr(err)
	is.NoErr(os.Chdir(repo))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	out := &bytes.Buffer{}
	root = New()
	root.SetOut(out)
	root.SetArgs([]string{"-c", cfg, "use", "alice", "demo-repo"})
	is.NoErr(root.Command.Execute())
	is.True(strings.Contains(out.String(), "git@github.com:alice/demo-repo.git"))

	out.Reset()
	root = New()
	root.SetOut(out)
	root.SetArgs([]string{"-c", cfg, "current"})
	is.NoErr(root.Command.Execute())
	is.Equal(strings.TrimSpace(out.String()), "alice")
}
