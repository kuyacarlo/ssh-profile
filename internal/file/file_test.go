package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/matryer/is"
)

func TestParseFilePath(t *testing.T) {
	is := is.New(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	xdg.Reload()

	got, err := ParseFilePath("~")
	is.NoErr(err)
	is.Equal(got, home)

	got, err = ParseFilePath("~/keys/id_ed25519")
	is.NoErr(err)
	is.Equal(got, filepath.Join(home, "keys", "id_ed25519"))

	_, err = ParseFilePath("")
	is.True(err != nil)

	_, err = ParseFilePath("~other/key")
	is.True(err != nil)

	got, err = ParseFilePath("/tmp/../tmp/key")
	is.NoErr(err)
	is.Equal(got, "/tmp/key")
}

func TestExists(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	is.NoErr(os.WriteFile(path, []byte("x"), 0o600))

	got, err := Exists(path)
	is.NoErr(err)
	is.Equal(got, path)

	_, err = Exists(filepath.Join(dir, "missing"))
	is.True(err != nil)

	_, err = Exists(dir)
	is.True(err != nil)
}
