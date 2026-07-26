package apply

import (
	"fmt"
	"strings"

	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/ssh-profiles/git-ssh/internal/file"
	"github.com/ssh-profiles/git-ssh/internal/sshconfig"
)

const (
	// ProfileKey records which git-ssh profile is active in the repo.
	ProfileKey = "git-ssh.profile"
	SSHCommand = "core.sshCommand"
)

// VCS is the git local-config surface used by use/unuse/current.
type VCS interface {
	IsRepository() bool
	Get(key string) (string, error)
	Set(key string, value string) error
	Unset(key string) error
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
	// Quote path for spaces; IdentitiesOnly avoids agent key order bugs.
	return fmt.Sprintf(`ssh -i %s -o IdentitiesOnly=yes`, shellQuote(path)), nil
}

// Use applies profile to the current repository via core.sshCommand.
func Use(vcs VCS, name string, profile config.Profile) error {
	if !vcs.IsRepository() {
		return fmt.Errorf("not a git repository")
	}
	cmd, err := SSHCommandFor(profile.IdentityFile)
	if err != nil {
		return err
	}
	if err := vcs.Set(SSHCommand, cmd); err != nil {
		return err
	}
	return vcs.Set(ProfileKey, name)
}

// Unuse clears git-ssh local config from the repository.
func Unuse(vcs VCS) error {
	if !vcs.IsRepository() {
		return fmt.Errorf("not a git repository")
	}
	if err := vcs.Unset(SSHCommand); err != nil {
		return err
	}
	return vcs.Unset(ProfileKey)
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
