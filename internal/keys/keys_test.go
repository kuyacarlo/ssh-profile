package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/matryer/is"
)

func TestEnsureEd25519CreatesAndReuses(t *testing.T) {
	is := is.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	xdg.Reload()

	priv, created, err := EnsureEd25519("alice")
	is.NoErr(err)
	is.True(created)
	is.True(strings.HasSuffix(priv, filepath.Join(".ssh", "git-ssh", "alice", "id_ed25519")))
	_, err = os.Stat(priv)
	is.NoErr(err)
	_, err = os.Stat(priv + ".pub")
	is.NoErr(err)

	pub, err := ReadPublicKey("alice")
	is.NoErr(err)
	is.True(strings.HasPrefix(pub, "ssh-ed25519 "))

	priv2, created2, err := EnsureEd25519("alice")
	is.NoErr(err)
	is.True(!created2)
	is.Equal(priv2, priv)
}

func TestEnsureEd25519RequiresName(t *testing.T) {
	is := is.New(t)
	_, _, err := EnsureEd25519("  ")
	is.True(err != nil)
}
