package remoteurl

import (
	"testing"

	"github.com/matryer/is"
)

func TestResolveTarget(t *testing.T) {
	is := is.New(t)

	u, err := ResolveTarget("alice", "demo-repo")
	is.NoErr(err)
	is.Equal(u, "git@github.com:alice/demo-repo.git")

	u, err = ResolveTarget("alice", "example-org/demo-repo")
	is.NoErr(err)
	is.Equal(u, "git@github.com:example-org/demo-repo.git")

	_, err = ResolveTarget("", "demo-repo")
	is.True(err != nil)

	_, err = ResolveTarget("alice", "a/b/c")
	is.True(err != nil)
}

func TestParseAndNormalize(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		in            string
		owner         string
		repo          string
		normalized    string
		ok            bool
	}{
		{"git@github.com:alice/test1.git", "alice", "test1", "git@github.com:alice/test1.git", true},
		{"git@alice.github.com:alice/test1.git", "alice", "test1", "git@github.com:alice/test1.git", true},
		{"alice.github.com:alice/test1", "alice", "test1", "git@github.com:alice/test1.git", true},
		{"github.com:alice/test1", "alice", "test1", "git@github.com:alice/test1.git", true},
		{"ssh://git@forge.example.com/alice/other-repo.git", "", "", "", false},
	}

	for _, c := range cases {
		owner, repo, ok := ParseOwnerRepo(c.in)
		is.Equal(ok, c.ok)
		if !c.ok {
			continue
		}
		is.Equal(owner, c.owner)
		is.Equal(repo, c.repo)
		out, ok := NormalizeGitHub(c.in)
		is.True(ok)
		is.Equal(out, c.normalized)
	}
}
