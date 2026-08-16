package gitprofile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withFakeGitProfile writes an executable `git-profile` into a temp dir and
// prepends it to PATH for the duration of the test.
func withFakeGitProfile(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "git-profile")
	script := "#!/bin/sh\nprintf 'git-profile ran: %s\\n' \"$@\" >&2\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestAvailableTrueWhenPresent(t *testing.T) {
	withFakeGitProfile(t)
	if !New().Available() {
		t.Fatal("expected Available() true with fake git-profile on PATH")
	}
}

func TestAvailableFalseWhenMissing(t *testing.T) {
	l := &Linker{lookPath: func(string) (string, error) {
		return "", exec.ErrNotFound
	}}
	if l.Available() {
		t.Fatal("expected Available() false with no git-profile")
	}
}

func TestRunAddAndRunUseExecFake(t *testing.T) {
	withFakeGitProfile(t)
	l := New()

	var out strings.Builder
	var errOut strings.Builder
	if err := l.RunAdd(strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	if !strings.Contains(errOut.String(), "add") {
		t.Fatalf("RunAdd did not execute `git-profile add`: %q", errOut.String())
	}

	errOut.Reset()
	if err := l.RunUse("alice", strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("RunUse: %v", err)
	}
	if !strings.Contains(errOut.String(), "alice") {
		t.Fatalf("RunUse did not pass profile name: %q", errOut.String())
	}
}

func TestSkipRunnerNeverAvailable(t *testing.T) {
	skip := Skip{}
	if skip.Available() {
		t.Fatal("Skip should report unavailable")
	}
}
