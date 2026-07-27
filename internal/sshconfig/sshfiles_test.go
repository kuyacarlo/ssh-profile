package sshconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestClassifyKey(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()

	priv := filepath.Join(dir, "id_ed25519")
	is.NoErr(os.WriteFile(priv, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"), 0o600))
	kind, err := ClassifyKey(priv)
	is.NoErr(err)
	is.Equal(kind, PrivateKey)

	pub := filepath.Join(dir, "id_ed25519.pub")
	is.NoErr(os.WriteFile(pub, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyData comment\n"), 0o644))
	kind, err = ClassifyKey(pub)
	is.NoErr(err)
	is.Equal(kind, PublicKey)

	unknown := filepath.Join(dir, "note.txt")
	is.NoErr(os.WriteFile(unknown, []byte("hello"), 0o644))
	kind, err = ClassifyKey(unknown)
	is.NoErr(err)
	is.Equal(kind, Unknown)

	_, err = ClassifyKey(filepath.Join(dir, "missing"))
	is.True(err != nil)
}
