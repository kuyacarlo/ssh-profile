package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// BackupPath returns a timestamped path under ~/.config/git-ssh/backups/.
func BackupPath() (string, error) {
	keep, err := xdg.ConfigFile("git-ssh/backups/.keep")
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(keep)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("git-ssh-%s.json", time.Now().Format("20060102-150405"))
	return filepath.Join(dir, name), nil
}

// ReplaceAll replaces profiles and top-level remote_host with those from other.
func (c *Config) ReplaceAll(other *Config) {
	c.RemoteHost = other.RemoteHost
	c.Profiles = make(map[string]Profile, len(other.Profiles))
	for name, profile := range other.Profiles {
		c.Profiles[name] = profile
	}
}

// LoadExisting reads JSON from an existing file (does not create it).
func (c *Config) LoadExisting(filename string) error {
	body, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return err
	}
	tmp := New()
	if len(body) > 0 {
		if err := json.Unmarshal(body, tmp); err != nil {
			return fmt.Errorf("invalid backup JSON: %w", err)
		}
	}
	if tmp.Profiles == nil {
		tmp.Profiles = make(map[string]Profile)
	}
	c.ReplaceAll(tmp)
	return nil
}
