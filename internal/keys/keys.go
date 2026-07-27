package keys

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

const keyName = "id_ed25519"

// BaseDir is ~/.ssh/git-ssh — managed keys live here; config stays in XDG.
func BaseDir() (string, error) {
	home := xdg.Home
	if home == "" {
		return "", fmt.Errorf("home directory not found")
	}
	return filepath.Join(home, ".ssh", "git-ssh"), nil
}

// ProfileDir is ~/.ssh/git-ssh/<profile>/.
func ProfileDir(profile string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, sanitize(profile)), nil
}

// PrivateKeyPath is ~/.ssh/git-ssh/<profile>/id_ed25519.
func PrivateKeyPath(profile string) (string, error) {
	dir, err := ProfileDir(profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, keyName), nil
}

// PublicKeyPath is the .pub sibling of the private key.
func PublicKeyPath(profile string) (string, error) {
	priv, err := PrivateKeyPath(profile)
	if err != nil {
		return "", err
	}
	return priv + ".pub", nil
}

// EnsureEd25519 returns the private key path, creating an ed25519 keypair
// under ~/.ssh/git-ssh/<profile>/ when missing. Existing keys are reused.
func EnsureEd25519(profile string) (privatePath string, created bool, err error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "", false, fmt.Errorf("profile name is required")
	}
	dir, err := ProfileDir(profile)
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", false, err
	}
	priv, err := PrivateKeyPath(profile)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(priv); err == nil {
		return priv, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, err
	}

	comment := "git-ssh:" + sanitize(profile)
	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-f", priv,
		"-N", "",
		"-C", comment,
		"-q",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", false, fmt.Errorf("ssh-keygen: %w\n%s", err, out)
	}
	_ = os.Chmod(priv, 0o600)
	if pub := priv + ".pub"; fileExists(pub) {
		_ = os.Chmod(pub, 0o644)
	}
	return priv, true, nil
}

// ReadPublicKey returns the public key line (trimmed), if present.
func ReadPublicKey(profile string) (string, error) {
	path, err := PublicKeyPath(profile)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sanitize(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "..", ".")
	return replacer.Replace(name)
}
