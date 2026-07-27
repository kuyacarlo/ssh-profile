package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ssh-profiles/git-ssh/internal/apply"
	"github.com/ssh-profiles/git-ssh/internal/remoteurl"
	"github.com/spf13/cobra"
)

func (r *Root) cloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone [profile] [repo|owner/repo|url] [directory]",
		Short: "Clone a GitHub repo with a profile key (Orca-safe)",
		Long: multiline(
			"Clones over git@github.com (not Host aliases) using the profile's",
			"private key via GIT_SSH_COMMAND, then applies the profile in the",
			"new repo so push/fetch keep working under Orca/ADE.",
			"",
			"Target forms:",
			"  demo-repo             → git@github.com:<github_user>/demo-repo.git",
			"  example-org/demo-repo → git@github.com:example-org/demo-repo.git",
			"  git@github.com:o/r.git (unchanged host)",
		),
		Example: multiline(
			`git-ssh clone alice private-repo`,
			`git-ssh clone alice example-org/private-repo`,
			`git-ssh clone alice example-org/private-repo ~/src/private-repo`,
		),
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !r.checkProfiles(cmd) {
				return nil
			}
			name := args[0]
			target := args[1]
			p, ok := r.cfg.Lookup(name)
			if !ok {
				return missingProfileError(name)
			}

			url, dirHint, err := cloneURL(p.GithubUser, target)
			if err != nil {
				return err
			}
			dest := ""
			if len(args) == 3 {
				dest = args[2]
			} else {
				dest = dirHint
			}

			sshCmd, err := apply.SSHCommandFor(p.IdentityFile)
			if err != nil {
				return err
			}

			gitArgs := []string{"clone", url}
			if dest != "" {
				gitArgs = append(gitArgs, dest)
			}
			c := exec.Command("git", gitArgs...)
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
			if err := c.Run(); err != nil {
				return fmt.Errorf("git clone: %w", err)
			}

			repoDir := dest
			if repoDir == "" {
				repoDir = dirHint
			}
			abs, err := filepath.Abs(repoDir)
			if err != nil {
				return err
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := os.Chdir(abs); err != nil {
				return err
			}
			defer func() { _ = os.Chdir(wd) }()

			result, err := apply.Use(r.git, name, p, apply.Options{})
			if err != nil {
				return fmt.Errorf("cloned but failed to apply profile: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cloned and applied `%s`.\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  path: %s\n", abs)
			fmt.Fprintf(cmd.OutOrStdout(), "  ssh: %s\n", result.SSHCommand)
			if result.RemoteURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  origin: %s\n", result.RemoteURL)
			}
			return nil
		},
	}
}

func cloneURL(defaultOwner, target string) (url, dirHint string, err error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", fmt.Errorf("empty clone target")
	}
	// Full git/ssh/https URLs: keep host as-is (caller already pinned the key).
	if strings.Contains(target, "://") || strings.HasPrefix(target, "git@") {
		owner, repo, ok := remoteurl.ParseOwnerRepo(target)
		if ok {
			u, err := remoteurl.OriginURL(owner, repo)
			if err != nil {
				return "", "", err
			}
			return u, repo, nil
		}
		base := filepath.Base(strings.TrimSuffix(target, ".git"))
		return target, base, nil
	}
	u, err := remoteurl.ResolveTarget(defaultOwner, target)
	if err != nil {
		return "", "", err
	}
	_, repo, ok := remoteurl.ParseOwnerRepo(u)
	if !ok {
		repo = remoteurl.RepoNameFromPath(target)
	}
	return u, repo, nil
}
