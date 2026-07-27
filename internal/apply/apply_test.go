package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/matryer/is"
)

type fakeVCS struct {
	repo    bool
	top     string
	kv      map[string]string
	remotes map[string]string
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

func (f *fakeVCS) TopLevel() (string, error) { return f.top, nil }

func (f *fakeVCS) HasRemote(name string) bool {
	_, ok := f.remotes[name]
	return ok
}

func (f *fakeVCS) GetRemote(name string) (string, error) {
	return f.remotes[name], nil
}

func (f *fakeVCS) EnsureRemote(name string, value string) error {
	if f.remotes == nil {
		f.remotes = map[string]string{}
	}
	f.remotes[name] = value
	return nil
}

func writeEd25519Private(t *testing.T, path string) {
	t.Helper()
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestUseAddsOrigin(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{
		repo:    true,
		top:     filepath.Join(dir, "test1"),
		kv:      map[string]string{},
		remotes: map[string]string{},
	}
	profile := config.Profile{
		IdentityFile: key,
		GithubUser:   "alice",
	}

	result, err := Use(vcs, "alice", profile, Options{})
	is.NoErr(err)
	is.Equal(result.RemoteAction, "added")
	is.Equal(result.RemoteURL, "git@github.com:alice/test1.git")
	is.Equal(vcs.remotes["origin"], "git@github.com:alice/test1.git")
	is.Equal(vcs.kv[ProfileKey], "alice")
}

func TestUseExplicitRepoTarget(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{
		repo:    true,
		top:     filepath.Join(dir, "some-folder"),
		kv:      map[string]string{},
		remotes: map[string]string{},
	}
	profile := config.Profile{IdentityFile: key, GithubUser: "alice"}

	result, err := Use(vcs, "alice", profile, Options{Target: "demo-repo"})
	is.NoErr(err)
	is.Equal(result.RemoteAction, "added")
	is.Equal(result.RemoteURL, "git@github.com:alice/demo-repo.git")
}

func TestUseOrgRepoTargetUpdatesOrigin(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{
		repo: true,
		top:  dir,
		kv:   map[string]string{},
		remotes: map[string]string{
			"origin": "git@github.com:alice/test1.git",
		},
	}
	result, err := Use(vcs, "alice", config.Profile{IdentityFile: key, GithubUser: "alice"}, Options{Target: "example-org/demo-repo"})
	is.NoErr(err)
	is.Equal(result.RemoteAction, "updated")
	is.Equal(vcs.remotes["origin"], "git@github.com:example-org/demo-repo.git")
}

func TestUseNormalizesAliasRemote(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{
		repo: true,
		top:  dir,
		kv:   map[string]string{},
		remotes: map[string]string{
			"origin": "git@alice.github.com:alice/old.git",
		},
	}
	result, err := Use(vcs, "alice", config.Profile{IdentityFile: key, GithubUser: "alice"}, Options{})
	is.NoErr(err)
	is.Equal(result.RemoteAction, "updated")
	is.Equal(vcs.remotes["origin"], "git@github.com:alice/old.git")
}

func TestUseNoRemoteFlag(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{repo: true, top: dir, kv: map[string]string{}, remotes: map[string]string{}}
	result, err := Use(vcs, "x", config.Profile{IdentityFile: key, GithubUser: "alice"}, Options{NoRemote: true})
	is.NoErr(err)
	is.Equal(result.RemoteAction, "skipped")
	is.Equal(len(vcs.remotes), 0)
}

func TestUseRequiresGithubUserWhenNoOrigin(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	writeEd25519Private(t, key)

	vcs := &fakeVCS{repo: true, top: dir, kv: map[string]string{}, remotes: map[string]string{}}
	_, err := Use(vcs, "x", config.Profile{IdentityFile: key}, Options{})
	is.True(err != nil)
}

func TestUnuse(t *testing.T) {
	is := is.New(t)
	vcs := &fakeVCS{repo: true, kv: map[string]string{ProfileKey: "x", SSHCommand: "ssh"}, remotes: map[string]string{"origin": "git@github.com:a/b.git"}}
	is.NoErr(Unuse(vcs))
	is.Equal(vcs.kv[ProfileKey], "")
	is.Equal(vcs.remotes["origin"], "git@github.com:a/b.git")
}

func TestCurrent(t *testing.T) {
	is := is.New(t)
	vcs := &fakeVCS{repo: true, kv: map[string]string{ProfileKey: "alice"}}
	name, err := Current(vcs)
	is.NoErr(err)
	is.Equal(name, "alice")

	_, err = Current(&fakeVCS{repo: false})
	is.True(err != nil)
}

func TestSSHCommandForQuotesSpaces(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "my keys")
	is.NoErr(os.MkdirAll(keyDir, 0o700))
	key := filepath.Join(keyDir, "id_ed25519")
	writeEd25519Private(t, key)

	cmd, err := SSHCommandFor(key)
	is.NoErr(err)
	is.True(strings.Contains(cmd, `"`+key+`"`))
	is.True(strings.Contains(cmd, "IdentitiesOnly=yes"))
}
