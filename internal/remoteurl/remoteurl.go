package remoteurl

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const DefaultHost = "github.com"

var (
	scpRemote = regexp.MustCompile(`(?i)^(?:git@)?([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
)

// NormalizeHost returns a lowercased host, defaulting to github.com.
func NormalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return DefaultHost
	}
	return host
}

// IsGitHubHost reports github.com or *.github.com aliases.
func IsGitHubHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == DefaultHost || strings.HasSuffix(host, ".github.com")
}

// CompatibleHost reports whether remoteHost should normalize onto preferred.
// GitHub family collapses onto github.com; other hosts only match exactly.
func CompatibleHost(remoteHost, preferred string) bool {
	preferred = NormalizeHost(preferred)
	remoteHost = strings.ToLower(strings.TrimSpace(remoteHost))
	if preferred == DefaultHost {
		return IsGitHubHost(remoteHost)
	}
	return remoteHost == preferred
}

// OriginURL builds git@<host>:owner/repo.git (GitHub and Gitea-compatible).
func OriginURL(host, owner, repo string) (string, error) {
	host = NormalizeHost(host)
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, ".git")
	if owner == "" {
		return "", fmt.Errorf("owner is required")
	}
	if repo == "" {
		return "", fmt.Errorf("repository name is required")
	}
	return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repo), nil
}

// ResolveTarget turns "demo-repo" or "owner/repo" into an origin URL on host.
func ResolveTarget(host, defaultOwner, target string) (string, error) {
	host = NormalizeHost(host)
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
		return OriginURL(host, owner, repo)
	}
	return OriginURL(host, defaultOwner, target)
}

// ParseRemote extracts host/owner/repo from common SSH/HTTPS remote forms.
func ParseRemote(remote string) (host, owner, repo string, ok bool) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", "", "", false
	}

	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil {
			return "", "", "", false
		}
		h := strings.ToLower(u.Host)
		if strings.Contains(h, ":") { // strip port
			h = strings.Split(h, ":")[0]
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || h == "" {
			return "", "", "", false
		}
		return h, parts[0], strings.TrimSuffix(parts[1], ".git"), true
	}

	m := scpRemote.FindStringSubmatch(remote)
	if m == nil {
		return "", "", "", false
	}
	return strings.ToLower(m[1]), m[2], m[3], true
}

// ParseOwnerRepo extracts owner/repo when the remote host is GitHub-family
// or matches preferred (empty preferred => github.com only).
func ParseOwnerRepo(remote string, preferred ...string) (owner, repo string, ok bool) {
	pref := DefaultHost
	if len(preferred) > 0 && strings.TrimSpace(preferred[0]) != "" {
		pref = preferred[0]
	}
	host, owner, repo, ok := ParseRemote(remote)
	if !ok || !CompatibleHost(host, pref) {
		return "", "", false
	}
	return owner, repo, true
}

// Normalize rewrites a compatible remote onto git@<preferred>:owner/repo.git.
// Incompatible remotes are left unchanged (ok=false).
func Normalize(remote, preferred string) (string, bool) {
	preferred = NormalizeHost(preferred)
	host, owner, repo, ok := ParseRemote(remote)
	if !ok || !CompatibleHost(host, preferred) {
		return remote, false
	}
	out, err := OriginURL(preferred, owner, repo)
	if err != nil {
		return remote, false
	}
	return out, true
}

// NormalizeGitHub rewrites GitHub-family remotes onto git@github.com:owner/repo.git.
func NormalizeGitHub(remote string) (string, bool) {
	return Normalize(remote, DefaultHost)
}

// RepoNameFromPath returns the directory base name for a repo path.
func RepoNameFromPath(repoPath string) string {
	return path.Base(strings.TrimRight(repoPath, "/"))
}
