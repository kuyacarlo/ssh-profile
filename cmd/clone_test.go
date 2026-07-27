package cmd

import (
	"testing"

	"github.com/matryer/is"
)

func TestCloneURL(t *testing.T) {
	is := is.New(t)

	url, dir, err := cloneURL("github.com", "alice", "private-repo")
	is.NoErr(err)
	is.Equal(url, "git@github.com:alice/private-repo.git")
	is.Equal(dir, "private-repo")

	url, dir, err = cloneURL("github.com", "alice", "example-org/private-repo")
	is.NoErr(err)
	is.Equal(url, "git@github.com:example-org/private-repo.git")
	is.Equal(dir, "private-repo")

	url, dir, err = cloneURL("github.com", "alice", "git@github.com:alice/x.git")
	is.NoErr(err)
	is.Equal(url, "git@github.com:alice/x.git")
	is.Equal(dir, "x")

	url, dir, err = cloneURL("forge.example.com", "alice", "private-repo")
	is.NoErr(err)
	is.Equal(url, "git@forge.example.com:alice/private-repo.git")
	is.Equal(dir, "private-repo")

	url, dir, err = cloneURL("forge.example.com", "alice", "ssh://git@forge.example.com/alice/x.git")
	is.NoErr(err)
	is.Equal(url, "git@forge.example.com:alice/x.git")
	is.Equal(dir, "x")
}
