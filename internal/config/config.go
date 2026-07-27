package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/xdg"
)

// Profile is a named SSH identity for git remotes.
// `use` applies the key via core.sshCommand and can create/normalize origin to
// git@<remote_host>:<github_user>/<repo>.git (default host: github.com).
type Profile struct {
	IdentityFile string            `json:"identity_file"`
	GithubUser   string            `json:"github_user,omitempty"`
	RemoteHost   string            `json:"remote_host,omitempty"` // e.g. github.com or forge.example.com
	HostAlias    string            `json:"host_alias,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
}

// Config is the sidecar profile store.
type Config struct {
	// RemoteHost is the default git SSH host for origin URLs (default github.com).
	// Profiles may override with profile.remote_host.
	RemoteHost string             `json:"remote_host,omitempty"`
	Profiles   map[string]Profile `json:"profiles"`
}

// New returns an empty config.
func New() *Config {
	return &Config{Profiles: make(map[string]Profile)}
}

// DefaultPath is ~/.config/git-ssh/config.json (XDG).
func DefaultPath() (string, error) {
	return xdg.ConfigFile("git-ssh/config.json")
}

// Len returns the number of profiles.
func (c *Config) Len() int {
	return len(c.Profiles)
}

// Lookup returns a profile by name.
func (c *Config) Lookup(name string) (Profile, bool) {
	p, ok := c.Profiles[name]
	return p, ok
}

// EffectiveRemoteHost returns profile.remote_host, else config.remote_host,
// else github.com.
func (c *Config) EffectiveRemoteHost(p Profile) string {
	if h := strings.TrimSpace(p.RemoteHost); h != "" {
		return strings.ToLower(h)
	}
	if c != nil {
		if h := strings.TrimSpace(c.RemoteHost); h != "" {
			return strings.ToLower(h)
		}
	}
	return "github.com"
}

// Names returns sorted profile names.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Store upserts a profile.
func (c *Config) Store(name string, profile Profile) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if strings.TrimSpace(profile.IdentityFile) == "" {
		return fmt.Errorf("identity_file is required")
	}
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	c.Profiles[name] = profile
	return nil
}

// DeleteProfile removes a profile.
func (c *Config) DeleteProfile(name string) bool {
	if _, ok := c.Profiles[name]; !ok {
		return false
	}
	delete(c.Profiles, name)
	return true
}

// Save writes the config as indented JSON.
func (c *Config) Save(filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filename, data, 0o644) //nolint:gosec
}

// Load reads profiles from JSON, creating an empty file if missing.
func (c *Config) Load(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if c.Profiles == nil {
			c.Profiles = make(map[string]Profile)
		}
		return c.Save(filename)
	}

	body, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		c.Profiles = make(map[string]Profile)
		return nil
	}
	if err := json.Unmarshal(body, c); err != nil {
		return err
	}
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	return nil
}
