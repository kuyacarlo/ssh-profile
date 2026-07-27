package remoteurl

import (
	"testing"

	"github.com/matryer/is"
)

func TestResolveTarget(t *testing.T) {
	is := is.New(t)

	u, err := ResolveTarget(DefaultHost, "alice", "demo-repo")
	is.NoErr(err)
	is.Equal(u, "git@github.com:alice/demo-repo.git")

	u, err = ResolveTarget(DefaultHost, "alice", "example-org/demo-repo")
	is.NoErr(err)
	is.Equal(u, "git@github.com:example-org/demo-repo.git")

	_, err = ResolveTarget(DefaultHost, "", "demo-repo")
	is.True(err != nil)

	_, err = ResolveTarget(DefaultHost, "alice", "a/b/c")
	is.True(err != nil)

	u, err = ResolveTarget("forge.example.com", "alice", "demo-repo")
	is.NoErr(err)
	is.Equal(u, "git@forge.example.com:alice/demo-repo.git")
}

func TestParseAndNormalize(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		in         string
		preferred  string
		owner      string
		repo       string
		normalized string
		ok         bool
	}{
		{"git@github.com:alice/test1.git", DefaultHost, "alice", "test1", "git@github.com:alice/test1.git", true},
		{"git@alice.github.com:alice/test1.git", DefaultHost, "alice", "test1", "git@github.com:alice/test1.git", true},
		{"alice.github.com:alice/test1", DefaultHost, "alice", "test1", "git@github.com:alice/test1.git", true},
		{"github.com:alice/test1", DefaultHost, "alice", "test1", "git@github.com:alice/test1.git", true},
		{"https://github.com/alice/test1.git", DefaultHost, "alice", "test1", "git@github.com:alice/test1.git", true},
		{"ssh://git@forge.example.com/alice/other-repo.git", DefaultHost, "", "", "", false},
		{"ssh://git@forge.example.com/alice/other-repo.git", "forge.example.com", "alice", "other-repo", "git@forge.example.com:alice/other-repo.git", true},
		{"git@forge.example.com:alice/other-repo.git", "forge.example.com", "alice", "other-repo", "git@forge.example.com:alice/other-repo.git", true},
	}

	for _, c := range cases {
		owner, repo, ok := ParseOwnerRepo(c.in, c.preferred)
		is.Equal(ok, c.ok)
		if !c.ok {
			out, ok2 := Normalize(c.in, c.preferred)
			is.True(!ok2)
			is.Equal(out, c.in)
			continue
		}
		is.Equal(owner, c.owner)
		is.Equal(repo, c.repo)
		out, ok := Normalize(c.in, c.preferred)
		is.True(ok)
		is.Equal(out, c.normalized)
	}
}

func TestEffectiveHostHelpers(t *testing.T) {
	is := is.New(t)
	is.Equal(NormalizeHost(""), DefaultHost)
	is.True(IsGitHubHost("github.com"))
	is.True(IsGitHubHost("alice.github.com"))
	is.True(!IsGitHubHost("forge.example.com"))
	is.True(CompatibleHost("alice.github.com", DefaultHost))
	is.True(CompatibleHost("forge.example.com", "forge.example.com"))
	is.True(!CompatibleHost("forge.example.com", DefaultHost))
}
