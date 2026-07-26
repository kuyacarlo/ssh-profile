package include

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/ssh-profiles/git-ssh/internal/file"
)

const (
	includeMarker = "# Managed by git-ssh — do not edit"
	includeGlob   = "git-ssh.d/*.conf"
)

// Dir is ~/.ssh/git-ssh.d
func Dir() (string, error) {
	home := xdg.Home
	if home == "" {
		return "", fmt.Errorf("home directory not found")
	}
	return filepath.Join(home, ".ssh", "git-ssh.d"), nil
}

// SSHConfigPath is ~/.ssh/config
func SSHConfigPath() (string, error) {
	home := xdg.Home
	if home == "" {
		return "", fmt.Errorf("home directory not found")
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// EnsureIncludeLine appends Include ~/.ssh/git-ssh.d/*.conf if missing.
func EnsureIncludeLine() error {
	sshConfig, err := SSHConfigPath()
	if err != nil {
		return err
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(sshConfig), 0o700); err != nil {
		return err
	}

	includeLine := "Include " + filepath.Join(filepath.Dir(sshConfig), includeGlob)

	body := ""
	if data, err := os.ReadFile(sshConfig); err == nil { //nolint:gosec
		body = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	if hasInclude(body, includeGlob) {
		return nil
	}

	var b strings.Builder
	if strings.TrimSpace(body) != "" {
		b.WriteString(strings.TrimRight(body, "\n"))
		b.WriteString("\n\n")
	}
	b.WriteString(includeMarker)
	b.WriteString("\n")
	b.WriteString(includeLine)
	b.WriteString("\n")

	return os.WriteFile(sshConfig, []byte(b.String()), 0o600) //nolint:gosec
}

func hasInclude(body, glob string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Include") && strings.Contains(fields[1], glob) {
			return true
		}
	}
	return false
}

// WriteProfile writes ~/.ssh/git-ssh.d/<name>.conf for the profile.
// Host github.com is never written (would break multi-account + Orca).
// Optional HostAlias becomes Host <alias> HostName github.com.
func WriteProfile(name string, profile config.Profile) error {
	if err := EnsureIncludeLine(); err != nil {
		return err
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	identity, err := file.ParseFilePath(profile.IdentityFile)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(includeMarker)
	b.WriteString("\n")
	b.WriteString("# Profile: ")
	b.WriteString(name)
	b.WriteString("\n")
	b.WriteString("# Applied per-repo via: git-ssh use ")
	b.WriteString(name)
	b.WriteString("\n")
	b.WriteString("# (core.sshCommand keeps remotes on github.com for Orca/ADE)\n")
	b.WriteString("# IdentityFile ")
	b.WriteString(identity)
	b.WriteString("\n")

	alias := strings.TrimSpace(profile.HostAlias)
	if alias == "" {
		alias = "git-ssh." + sanitize(name)
	}
	if alias != "" && !strings.EqualFold(alias, "github.com") {
		b.WriteString("\n")
		b.WriteString("Host ")
		b.WriteString(alias)
		b.WriteString("\n")
		b.WriteString("  HostName github.com\n")
		b.WriteString("  User git\n")
		b.WriteString("  IdentityFile ")
		b.WriteString(identity)
		b.WriteString("\n")
		b.WriteString("  IdentitiesOnly yes\n")

		for key, value := range profile.Config {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" || value == "" {
				continue
			}
			if strings.EqualFold(key, "HostName") || strings.EqualFold(key, "User") ||
				strings.EqualFold(key, "IdentityFile") || strings.EqualFold(key, "IdentitiesOnly") {
				continue
			}
			b.WriteString("  ")
			b.WriteString(key)
			b.WriteString(" ")
			b.WriteString(value)
			b.WriteString("\n")
		}
	}

	path := filepath.Join(dir, sanitize(name)+".conf")
	return os.WriteFile(path, []byte(b.String()), 0o600) //nolint:gosec
}

// RemoveProfile deletes the Include fragment for a profile.
func RemoveProfile(name string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, sanitize(name)+".conf")
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func sanitize(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "..", ".")
	return replacer.Replace(name)
}
