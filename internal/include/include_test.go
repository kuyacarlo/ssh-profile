package include

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/matryer/is"
)

func TestHasInclude(t *testing.T) {
	is := is.New(t)
	is.True(hasInclude("Include ~/.ssh/git-ssh.d/*.conf\n", includeGlob))
	is.True(!hasInclude("Include ~/.ssh/other.d/*.conf\n", includeGlob))
}

func TestWriteProfileAndEnsure(t *testing.T) {
	is := is.New(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	xdg.Reload()

	sshDir := filepath.Join(home, ".ssh")
	is.NoErr(os.MkdirAll(sshDir, 0o700))
	sshConfig := filepath.Join(sshDir, "config")
	is.NoErr(os.WriteFile(sshConfig, []byte("Host bastion\n  User alice\n"), 0o600))

	err := WriteProfile("kc", config.Profile{
		IdentityFile: filepath.Join(home, "key"),
		HostAlias:    "git-ssh.kc",
		Config:       map[string]string{"ForwardAgent": "yes"},
	})
	is.NoErr(err)

	body, err := os.ReadFile(sshConfig)
	is.NoErr(err)
	is.True(strings.Contains(string(body), "git-ssh.d/*.conf"))
	is.True(strings.Contains(string(body), "Host bastion"))

	frag, err := os.ReadFile(filepath.Join(sshDir, "git-ssh.d", "kc.conf"))
	is.NoErr(err)
	text := string(frag)
	is.True(strings.Contains(text, "Host git-ssh.kc"))
	is.True(strings.Contains(text, "HostName github.com"))
	is.True(!strings.Contains(text, "\nHost github.com\n"))
	is.True(strings.Contains(text, "ForwardAgent yes"))

	is.NoErr(RemoveProfile("kc"))
	_, err = os.Stat(filepath.Join(sshDir, "git-ssh.d", "kc.conf"))
	is.True(os.IsNotExist(err))
}
