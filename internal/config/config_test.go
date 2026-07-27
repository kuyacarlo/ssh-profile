package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestStoreAndLookup(t *testing.T) {
	is := is.New(t)
	cfg := New()

	err := cfg.Store("work", Profile{IdentityFile: "~/.ssh/work"})
	is.NoErr(err)
	is.Equal(cfg.Len(), 1)

	p, ok := cfg.Lookup("work")
	is.True(ok)
	is.Equal(p.IdentityFile, "~/.ssh/work")

	err = cfg.Store("", Profile{IdentityFile: "x"})
	is.True(err != nil)

	err = cfg.Store("bad", Profile{})
	is.True(err != nil)
}

func TestDeleteProfile(t *testing.T) {
	is := is.New(t)
	cfg := New()
	_ = cfg.Store("home", Profile{IdentityFile: "~/.ssh/id"})

	is.True(!cfg.DeleteProfile("missing"))
	is.True(cfg.DeleteProfile("home"))
	is.Equal(cfg.Len(), 0)
}

func TestNamesSorted(t *testing.T) {
	is := is.New(t)
	cfg := New()
	_ = cfg.Store("zeta", Profile{IdentityFile: "a"})
	_ = cfg.Store("alpha", Profile{IdentityFile: "b"})
	is.Equal(cfg.Names(), []string{"alpha", "zeta"})
}

func TestSaveLoadRoundTrip(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := New()
	_ = cfg.Store("kc", Profile{
		IdentityFile: "~/.ssh/alice/id_ed25519",
		HostAlias:    "git-ssh.kc",
		Config:       map[string]string{"IdentitiesOnly": "yes"},
	})
	is.NoErr(cfg.Save(path))

	loaded := New()
	is.NoErr(loaded.Load(path))
	p, ok := loaded.Lookup("kc")
	is.True(ok)
	is.Equal(p.IdentityFile, "~/.ssh/alice/id_ed25519")
	is.Equal(p.HostAlias, "git-ssh.kc")
	is.Equal(p.Config["IdentitiesOnly"], "yes")
}

func TestEffectiveRemoteHost(t *testing.T) {
	is := is.New(t)
	cfg := New()
	is.Equal(cfg.EffectiveRemoteHost(Profile{}), "github.com")

	cfg.RemoteHost = "forge.example.com"
	is.Equal(cfg.EffectiveRemoteHost(Profile{}), "forge.example.com")
	is.Equal(cfg.EffectiveRemoteHost(Profile{RemoteHost: "github.com"}), "github.com")
}

func TestLoadCreatesMissingFile(t *testing.T) {
	is := is.New(t)
	path := filepath.Join(t.TempDir(), "missing", "config.json")
	cfg := New()
	is.NoErr(cfg.Load(path))
	_, err := os.Stat(path)
	is.NoErr(err)
}

func TestSaveLoadRemoteHost(t *testing.T) {
	is := is.New(t)
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := New()
	cfg.RemoteHost = "forge.example.com"
	_ = cfg.Store("alice", Profile{IdentityFile: "key", RemoteHost: "github.com"})
	is.NoErr(cfg.Save(path))

	loaded := New()
	is.NoErr(loaded.Load(path))
	is.Equal(loaded.RemoteHost, "forge.example.com")
	p, ok := loaded.Lookup("alice")
	is.True(ok)
	is.Equal(p.RemoteHost, "github.com")
}
