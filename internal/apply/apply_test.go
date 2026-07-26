package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/matryer/is"
)

type fakeVCS struct {
	repo bool
	kv   map[string]string
}

func (f *fakeVCS) IsRepository() bool { return f.repo }

func (f *fakeVCS) Get(key string) (string, error) {
	return f.kv[key], nil
}

func (f *fakeVCS) Set(key string, value string) error {
	if f.kv == nil {
		f.kv = map[string]string{}
	}
	f.kv[key] = value
	return nil
}

func (f *fakeVCS) Unset(key string) error {
	delete(f.kv, key)
	return nil
}

func writeEd25519Private(t *testing.T, path string) {
	t.Helper()
	// Minimal PEM wrapper — ClassifyKey only checks PEM framing.
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestUseUnuseCurrent(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{repo: true, kv: map[string]string{}}
	profile := config.Profile{IdentityFile: key}

	is.NoErr(Use(vcs, "kc", profile))
	is.Equal(vcs.kv[ProfileKey], "kc")
	is.True(vcs.kv[SSHCommand] != "")
	is.True(filepath.Base(vcs.kv[SSHCommand]) != "") // smoke

	name, err := Current(vcs)
	is.NoErr(err)
	is.Equal(name, "kc")

	is.NoErr(Unuse(vcs))
	is.Equal(vcs.kv[ProfileKey], "")
	is.Equal(vcs.kv[SSHCommand], "")
}

func TestUseRequiresRepo(t *testing.T) {
	is := is.New(t)
	err := Use(&fakeVCS{repo: false}, "x", config.Profile{IdentityFile: "y"})
	is.True(err != nil)
}

func TestSSHCommandForRejectsMissing(t *testing.T) {
	is := is.New(t)
	_, err := SSHCommandFor(filepath.Join(t.TempDir(), "nope"))
	is.True(err != nil)
}
