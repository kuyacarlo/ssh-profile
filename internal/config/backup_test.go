package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestBackupRestoreRoundTrip(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	live := filepath.Join(dir, "config.json")
	backup := filepath.Join(dir, "backup.json")

	cfg := New()
	_ = cfg.Store("alice", Profile{IdentityFile: "~/.ssh/alice/id_ed25519"})
	_ = cfg.Store("bob", Profile{IdentityFile: "~/.ssh/bob/id_ed25519"})
	is.NoErr(cfg.Save(live))
	is.NoErr(cfg.Save(backup))

	empty := New()
	is.NoErr(empty.Save(live))
	is.Equal(empty.Len(), 0)

	restored := New()
	is.NoErr(restored.LoadExisting(backup))
	is.Equal(restored.Len(), 2)
	p, ok := restored.Lookup("bob")
	is.True(ok)
	is.Equal(p.IdentityFile, "~/.ssh/bob/id_ed25519")

	_, err := os.Stat(backup)
	is.NoErr(err)
}

func TestLoadExistingMissing(t *testing.T) {
	is := is.New(t)
	cfg := New()
	err := cfg.LoadExisting(filepath.Join(t.TempDir(), "nope.json"))
	is.True(err != nil)
}
