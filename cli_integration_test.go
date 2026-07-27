package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Integration smoke against a real git repo + built binary.
func TestCLIUseTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration in short mode")
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// tests may run from module root
	bin := filepath.Join(t.TempDir(), "git-ssh")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	keyDir := t.TempDir()
	key := filepath.Join(keyDir, "id_ed25519")
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(key, []byte(pem), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := filepath.Join(t.TempDir(), "config.json")
	home := t.TempDir()

	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+home,
			"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return string(out)
	}

	run(root, "-c", cfg, "add", "alice",
		"--identity", key,
		"--github-user", "alice",
	)

	repo := filepath.Join(t.TempDir(), "random-folder-name")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	out := run(repo, "-c", cfg, "use", "alice", "demo-repo")
	if !strings.Contains(out, "git@github.com:alice/demo-repo.git") {
		t.Fatalf("expected demo-repo remote in output:\n%s", out)
	}

	url, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(url)); got != "git@github.com:alice/demo-repo.git" {
		t.Fatalf("origin=%q", got)
	}

	out = run(repo, "-c", cfg, "use", "alice", "example-org/demo-repo")
	if !strings.Contains(out, "git@github.com:example-org/demo-repo.git") {
		t.Fatalf("expected org remote:\n%s", out)
	}
	url, err = exec.Command("git", "-C", repo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(url)); got != "git@github.com:example-org/demo-repo.git" {
		t.Fatalf("origin=%q", got)
	}

	sshCmd, err := exec.Command("git", "-C", repo, "config", "--local", "--get", "core.sshCommand").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sshCmd), key) {
		t.Fatalf("core.sshCommand=%q", sshCmd)
	}
}
