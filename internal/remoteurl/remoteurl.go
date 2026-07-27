package remoteurl

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var (
	scpGitHub = regexp.MustCompile(`(?i)^(?:git@)?([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
	// host aliases like alice.github.com still carry owner/repo
)

// OriginURL builds an Orca-safe GitHub SSH remote.
func OriginURL(githubUser, repo string) (string, error) {
	user := strings.TrimSpace(githubUser)
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, ".git")
	if user == "" {
		return "", fmt.Errorf("github user is required")
	}
	if repo == "" {
		return "", fmt.Errorf("repository name is required")
	}
	return fmt.Sprintf("git@github.com:%s/%s.git", user, repo), nil
}

// ResolveTarget turns "demo-repo" or "example-org/demo-repo" into an origin URL.
// Bare repo names use defaultOwner (profile github_user).
func ResolveTarget(defaultOwner, target string) (string, error) {
	target = strings.TrimSpace(target)
	target = strings.TrimSuffix(target, ".git")
	if target == "" {
		return "", fmt.Errorf("empty remote target")
	}
	if strings.Contains(target, "/") {
		owner, repo, ok := strings.Cut(target, "/")
		owner = strings.TrimSpace(owner)
		repo = strings.TrimSpace(repo)
		if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
			return "", fmt.Errorf("invalid target %q (want repo or owner/repo)", target)
		}
		return OriginURL(owner, repo)
	}
	return OriginURL(defaultOwner, target)
}

// ParseOwnerRepo extracts owner/repo from common GitHub remote forms.
// Accepts github.com and *.github.com Host aliases.
func ParseOwnerRepo(remote string) (owner, repo string, ok bool) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", "", false
	}

	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", false
		}
		host := strings.ToLower(u.Host)
		if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
			return "", "", false
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", false
		}
		return parts[0], strings.TrimSuffix(parts[1], ".git"), true
	}

	m := scpGitHub.FindStringSubmatch(remote)
	if m == nil {
		return "", "", false
	}
	host := strings.ToLower(m[1])
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
		return "", "", false
	}
	return m[2], m[3], true
}

// NormalizeGitHub rewrites a GitHub-like remote onto git@github.com:owner/repo.git.
// Non-GitHub remotes are left unchanged (ok=false).
func NormalizeGitHub(remote string) (string, bool) {
	owner, repo, ok := ParseOwnerRepo(remote)
	if !ok {
		return remote, false
	}
	out, err := OriginURL(owner, repo)
	if err != nil {
		return remote, false
	}
	return out, true
}

// RepoNameFromPath returns the directory base name for a repo path.
func RepoNameFromPath(repoPath string) string {
	return path.Base(strings.TrimRight(repoPath, "/"))
}
