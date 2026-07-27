package apply

import (
	"fmt"
	"strings"

	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/ssh-profiles/git-ssh/internal/file"
	"github.com/ssh-profiles/git-ssh/internal/remoteurl"
	"github.com/ssh-profiles/git-ssh/internal/sshconfig"
)

const (
	// ProfileKey records which git-ssh profile is active (pairs with
	// git-profile's current-profile.name).
	ProfileKey = "current-profile.ssh"
	SSHCommand = "core.sshCommand"
	Origin     = "origin"
)

// VCS is the git surface used by use/unuse/current.
type VCS interface {
	IsRepository() bool
	Get(key string) (string, error)
	Set(key string, value string) error
	Unset(key string) error
	TopLevel() (string, error)
	HasRemote(name string) bool
	GetRemote(name string) (string, error)
	EnsureRemote(name string, value string) error
}

// Options controls remote wiring during Use.
type Options struct {
	// Target is "demo-repo" or "owner/repo" (e.g. example-org/demo-repo).
	// When set, origin is created/updated to that GitHub URL.
	Target string
	// NoRemote skips origin create/update/normalize.
	NoRemote bool
}

// Result describes what Use changed.
type Result struct {
	SSHCommand   string
	RemoteURL    string
	RemoteAction string // "added", "updated", "unchanged", "skipped"
}

// SSHCommandFor builds an Orca-safe ssh command that pins one identity
// while remotes stay on github.com.
func SSHCommandFor(identityFile string) (string, error) {
	path, err := file.ParseFilePath(identityFile)
	if err != nil {
		return "", err
	}
	if _, err := file.Exists(path); err != nil {
		return "", fmt.Errorf("identity file: %w", err)
	}
	kind, err := sshconfig.ClassifyKey(path)
	if err != nil {
		return "", err
	}
	if kind != sshconfig.PrivateKey {
		return "", fmt.Errorf("identity file is not a private key: %s", path)
	}
	return fmt.Sprintf(`ssh -i %s -o IdentitiesOnly=yes`, shellQuote(path)), nil
}

// Use applies profile identity (+ optional origin remote) to the repo.
func Use(vcs VCS, name string, profile config.Profile, opts Options) (Result, error) {
	var result Result
	if !vcs.IsRepository() {
		return result, fmt.Errorf("not a git repository")
	}
	cmd, err := SSHCommandFor(profile.IdentityFile)
	if err != nil {
		return result, err
	}
	if err := vcs.Set(SSHCommand, cmd); err != nil {
		return result, err
	}
	if err := vcs.Set(ProfileKey, name); err != nil {
		return result, err
	}
	result.SSHCommand = cmd

	if opts.NoRemote {
		result.RemoteAction = "skipped"
		return result, nil
	}

	remoteURL, action, err := ensureOrigin(vcs, profile, opts.Target)
	if err != nil {
		return result, err
	}
	result.RemoteURL = remoteURL
	result.RemoteAction = action
	return result, nil
}

func ensureOrigin(vcs VCS, profile config.Profile, target string) (string, string, error) {
	target = strings.TrimSpace(target)

	// Explicit target always wins (folder name irrelevant).
	if target != "" {
		url, err := remoteurl.ResolveTarget(profile.GithubUser, target)
		if err != nil {
			return "", "", err
		}
		had := vcs.HasRemote(Origin)
		if had {
			current, err := vcs.GetRemote(Origin)
			if err != nil {
				return "", "", err
			}
			if current == url {
				return url, "unchanged", nil
			}
		}
		if err := vcs.EnsureRemote(Origin, url); err != nil {
			return "", "", err
		}
		if had {
			return url, "updated", nil
		}
		return url, "added", nil
	}

	if vcs.HasRemote(Origin) {
		current, err := vcs.GetRemote(Origin)
		if err != nil {
			return "", "", err
		}
		normalized, ok := remoteurl.NormalizeGitHub(current)
		if !ok {
			return current, "unchanged", nil
		}
		if normalized == current {
			return current, "unchanged", nil
		}
		if err := vcs.EnsureRemote(Origin, normalized); err != nil {
			return "", "", err
		}
		return normalized, "updated", nil
	}

	user := strings.TrimSpace(profile.GithubUser)
	if user == "" {
		return "", "skipped", fmt.Errorf("no origin remote and profile has no github_user; re-add with --github-user or pass a repo target")
	}

	top, err := vcs.TopLevel()
	if err != nil {
		return "", "", err
	}
	url, err := remoteurl.OriginURL(user, remoteurl.RepoNameFromPath(top))
	if err != nil {
		return "", "", err
	}
	if err := vcs.EnsureRemote(Origin, url); err != nil {
		return "", "", err
	}
	return url, "added", nil
}

// Unuse clears git-ssh local config from the repository.
// Remotes are left alone.
func Unuse(vcs VCS) error {
	if !vcs.IsRepository() {
		return fmt.Errorf("not a git repository")
	}
	if err := vcs.Unset(SSHCommand); err != nil {
		return err
	}
	if err := vcs.Unset(ProfileKey); err != nil {
		return err
	}
	// Legacy marker from earlier git-ssh builds.
	_ = vcs.Unset("git-ssh.profile")
	return nil
}

// Current returns the active profile name, if any.
func Current(vcs VCS) (string, error) {
	if !vcs.IsRepository() {
		return "", fmt.Errorf("not a git repository")
	}
	name, err := vcs.Get(ProfileKey)
	if err != nil || strings.TrimSpace(name) == "" {
		return "", err
	}
	return name, nil
}

func shellQuote(path string) string {
	if strings.ContainsAny(path, " \t\"'\\") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return path
}
