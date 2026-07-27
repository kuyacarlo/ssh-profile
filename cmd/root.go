package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ssh-profiles/git-ssh/internal/apply"
	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/ssh-profiles/git-ssh/internal/git"
	"github.com/ssh-profiles/git-ssh/internal/include"
	"github.com/ssh-profiles/git-ssh/internal/keys"
	"github.com/ssh-profiles/git-ssh/internal/remoteurl"
	"github.com/ssh-profiles/git-ssh/internal/ui"
	"github.com/spf13/cobra"
)

// Root is the git-ssh CLI.
type Root struct {
	cobra.Command

	Version     string
	CommitHash  string
	CompileDate string

	filename string
	cfg      *config.Config
	git      *git.Git
}

// New builds the root command with subcommands.
func New() *Root {
	r := &Root{
		Version:     "0.1.0",
		CommitHash:  "Unknown",
		CompileDate: "Unknown",
		cfg:         config.New(),
		git:         git.New(),
	}

	r.Command = cobra.Command{
		Use:   "git-ssh",
		Short: "Per-repo SSH identity profiles for GitHub (Orca-safe)",
		Long: multiline(
			"Per-repo SSH keys for GitHub without Host aliases (Orca-safe).",
			"",
			"Layout:",
			"  ~/.config/git-ssh/config.json     profile store",
			"  ~/.ssh/git-ssh/<profile>/id_ed25519   managed keys",
			"",
			"github_user is the default GitHub owner used when wiring origin",
			"(bare `demo-repo` → git@github.com:<github_user>/demo-repo.git).",
			"It defaults to the profile name.",
			"",
			"Lifetime defaults (once per account, then forever):",
			"  git-ssh add alice",
			"  git-ssh clone alice private-repo",
			"  git-ssh use alice                 # existing checkout",
			"",
			"Pairs with git-profile under the same name:",
			"  git-profile use alice",
			"  git-ssh use alice",
			"Markers: current-profile.name (git-profile), current-profile.ssh (git-ssh).",
		),
		SilenceUsage: true,
	}

	r.PersistentFlags().StringVarP(&r.filename, "config", "c", "", "config file (default: ~/.config/git-ssh/config.json)")

	r.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		switch cmd.Name() {
		case "version", "help", "completion":
			return nil
		}
		if cmd == &r.Command {
			return nil
		}
		return r.loadConfig()
	}

	r.AddCommand(
		r.addCmd(),
		r.listCmd(),
		r.showCmd(),
		r.delCmd(),
		r.useCmd(),
		r.cloneCmd(),
		r.unuseCmd(),
		r.currentCmd(),
		r.exportCmd(),
		r.importCmd(),
		r.backupCmd(),
		r.restoreCmd(),
		r.completionCmd(),
		r.versionCmd(),
	)

	r.SetOut(os.Stdout)
	r.SetErr(os.Stderr)
	return r
}

// Execute runs the CLI.
func (r *Root) Execute() {
	if err := r.Command.Execute(); err != nil {
		os.Exit(1)
	}
}

func (r *Root) loadConfig() error {
	filename := r.filename
	if filename == "" {
		path, err := config.DefaultPath()
		if err != nil {
			return err
		}
		filename = path
	}
	if err := r.cfg.Load(filename); err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}
	r.filename = filename
	return nil
}

func (r *Root) saveConfig() error {
	return r.cfg.Save(r.filename)
}

func (r *Root) addCmd() *cobra.Command {
	var identity string
	var alias string
	var githubUser string
	var sets []string

	cmd := &cobra.Command{
		Use:     "add [profile]",
		Aliases: []string{"set"},
		Short:   "Add or update a profile (auto key + github_user)",
		Long: multiline(
			"Defaults that stick:",
			"  github_user  = profile name (override with --github-user)",
			"  identity     = ~/.ssh/git-ssh/<profile>/id_ed25519",
			"                 (created with ssh-keygen -t ed25519 if missing)",
			"",
			"Pass --identity only when reusing an existing key outside the managed tree.",
		),
		Example: multiline(
			`git-ssh add alice`,
			`git-ssh add work --github-user acme-bot`,
			`git-ssh add alice --identity ~/.ssh/other/id_ed25519`,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return r.addInteractive(cmd)
			}
			name := args[0]
			profile, exists := r.cfg.Lookup(name)
			if identity != "" {
				profile.IdentityFile = identity
			}
			if alias != "" {
				profile.HostAlias = alias
			}
			if githubUser != "" {
				profile.GithubUser = githubUser
			}
			createdKey, err := r.fillProfileDefaults(name, &profile)
			if err != nil {
				return err
			}
			if len(sets) > 0 {
				if profile.Config == nil {
					profile.Config = map[string]string{}
				}
				for _, item := range sets {
					key, value, ok := strings.Cut(item, "=")
					if !ok || strings.TrimSpace(key) == "" {
						return fmt.Errorf("invalid --set %q (want Key=Value)", item)
					}
					profile.Config[strings.TrimSpace(key)] = strings.TrimSpace(value)
				}
			}
			return r.saveProfile(cmd, name, profile, !exists, createdKey)
		},
	}

	cmd.Flags().StringVarP(&identity, "identity", "i", "", "private key path (default: managed ~/.ssh/git-ssh/<profile>/id_ed25519)")
	cmd.Flags().StringVarP(&githubUser, "github-user", "u", "", "GitHub owner for origin remotes (default: profile name)")
	cmd.Flags().StringVar(&alias, "alias", "", "optional SSH Host alias (not github.com)")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "extra SSH config Key=Value")
	return cmd
}

func (r *Root) addInteractive(cmd *cobra.Command) error {
	name, err := ui.PromptProfileName(cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		if ui.IsAborted(err) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Interactive add cancelled.")
			return nil
		}
		return err
	}
	existing, _ := r.cfg.Lookup(name)
	githubDefault := existing.GithubUser
	if githubDefault == "" {
		githubDefault = name
	}
	form, err := ui.PromptProfileFields(ui.ProfileFormData{
		Profile:      name,
		IdentityFile: existing.IdentityFile,
		GithubUser:   githubDefault,
		HostAlias:    existing.HostAlias,
	}, cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		if ui.IsAborted(err) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Interactive add cancelled.")
			return nil
		}
		return err
	}
	profile := config.Profile{
		IdentityFile: form.IdentityFile,
		GithubUser:   form.GithubUser,
		HostAlias:    form.HostAlias,
		Config:       existing.Config,
	}
	createdKey, err := r.fillProfileDefaults(name, &profile)
	if err != nil {
		return err
	}
	_, existed := r.cfg.Lookup(name)
	return r.saveProfile(cmd, name, profile, !existed, createdKey)
}

// fillProfileDefaults sets github_user from the profile name and ensures a
// managed ed25519 key when identity_file is empty.
func (r *Root) fillProfileDefaults(name string, profile *config.Profile) (createdKey bool, err error) {
	if strings.TrimSpace(profile.GithubUser) == "" {
		profile.GithubUser = name
	}
	if strings.TrimSpace(profile.IdentityFile) != "" {
		return false, nil
	}
	path, created, err := keys.EnsureEd25519(name)
	if err != nil {
		return false, err
	}
	profile.IdentityFile = path
	return created, nil
}

func (r *Root) saveProfile(cmd *cobra.Command, name string, profile config.Profile, isNew, createdKey bool) error {
	if err := r.cfg.Store(name, profile); err != nil {
		return err
	}
	if err := r.saveConfig(); err != nil {
		return err
	}
	if err := include.WriteProfile(name, profile); err != nil {
		return fmt.Errorf("profile saved but include write failed: %w", err)
	}
	action := "Successfully updated"
	if isNew {
		action = "Successfully added"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s `%s` profile.\n", action, name)
	fmt.Fprintf(cmd.OutOrStdout(), "  github_user: %s\n", profile.GithubUser)
	fmt.Fprintf(cmd.OutOrStdout(), "  identity: %s\n", profile.IdentityFile)
	if createdKey {
		fmt.Fprintln(cmd.OutOrStdout(), "  key: created (ed25519, empty passphrase)")
	}
	if pub := publicKeyLine(name, profile.IdentityFile); pub != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "  public key (add to GitHub → SSH keys):")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", pub)
	}
	return nil
}

func publicKeyLine(profile, identityFile string) string {
	if pub, err := keys.ReadPublicKey(profile); err == nil && pub != "" {
		return pub
	}
	body, err := os.ReadFile(identityFile + ".pub") //nolint:gosec
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func (r *Root) listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !r.checkProfiles(cmd) {
				return nil
			}
			current := ""
			if r.git.IsRepository() {
				if name, err := apply.Current(r.git); err == nil {
					current = name
				}
			}
			for _, name := range r.cfg.Names() {
				p, _ := r.cfg.Lookup(name)
				mark := " "
				if name == current {
					mark = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\t%s\t%s\n", mark, name, p.GithubUser, p.IdentityFile)
			}
			return nil
		},
	}
}

func (r *Root) showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [profile]",
		Short: "Show a profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !r.checkProfiles(cmd) {
				return nil
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				selected, err := ui.SelectProfile(r.cfg.Names(), cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					if ui.IsAborted(err) {
						return nil
					}
					return err
				}
				name = selected
			}
			p, ok := r.cfg.Lookup(name)
			if !ok {
				return missingProfileError(name)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "identity_file: %s\n", p.IdentityFile)
			if p.GithubUser != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "github_user: %s\n", p.GithubUser)
			}
			if p.HostAlias != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "host_alias: %s\n", p.HostAlias)
			}
			for _, key := range sortedKeys(p.Config) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", key, p.Config[key])
			}
			return nil
		},
	}
}

func (r *Root) delCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "del [profile]",
		Aliases: []string{"rm", "delete"},
		Short:   "Delete a profile",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !r.checkProfiles(cmd) {
				return nil
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				selected, err := ui.SelectProfile(r.cfg.Names(), cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					if ui.IsAborted(err) {
						fmt.Fprintln(cmd.ErrOrStderr(), "Interactive del cancelled.")
						return nil
					}
					return err
				}
				name = selected
			}
			if !r.cfg.DeleteProfile(name) {
				return missingProfileError(name)
			}
			if err := r.saveConfig(); err != nil {
				return err
			}
			_ = include.RemoveProfile(name)
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted `%s` profile.\n", name)
			return nil
		},
	}
}

func (r *Root) useCmd() *cobra.Command {
	var noRemote bool

	cmd := &cobra.Command{
		Use:     "use [profile] [repo|owner/repo]",
		Aliases: []string{"u"},
		Short:   "Apply profile key + origin remote to this git repo",
		Long: multiline(
			"Sets core.sshCommand to the profile's private key (Orca-safe).",
			"Records current-profile.ssh (pairs with git-profile's current-profile.name).",
			"",
			"Remote target (optional 2nd arg):",
			"  demo-repo             → git@github.com:<github_user>/demo-repo.git",
			"  example-org/demo-repo → git@github.com:example-org/demo-repo.git",
			"",
			"With no args: interactive profile select.",
			"With no 2nd arg: origin from directory name, or normalize existing GitHub remote.",
		),
		Example: multiline(
			`git-ssh use`,
			`git-ssh use alice`,
			`git-ssh use alice demo-repo`,
			`git-ssh use alice example-org/demo-repo`,
			`git-ssh use alice --no-remote`,
		),
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := r.checkRepo(); err != nil {
				return err
			}
			if !r.checkProfiles(cmd) {
				return nil
			}

			name := ""
			target := ""
			switch len(args) {
			case 0:
				selected, err := ui.SelectProfile(r.cfg.Names(), cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					if ui.IsAborted(err) {
						fmt.Fprintln(cmd.ErrOrStderr(), "Interactive use cancelled.")
						return nil
					}
					return fmt.Errorf("unable to select a profile: %w", err)
				}
				name = selected
			case 1:
				name = args[0]
			case 2:
				name = args[0]
				target = args[1]
			}

			p, ok := r.cfg.Lookup(name)
			if !ok {
				return missingProfileError(name)
			}
			result, err := apply.Use(r.git, name, p, apply.Options{
				Target:   target,
				NoRemote: noRemote,
			})
			if err != nil {
				return err
			}
			if err := include.WriteProfile(name, p); err != nil {
				return fmt.Errorf("applied locally but include write failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully applied `%s` profile to current git repository.\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  ssh: %s\n", result.SSHCommand)
			switch result.RemoteAction {
			case "added":
				fmt.Fprintf(cmd.OutOrStdout(), "  origin: added %s\n", result.RemoteURL)
			case "updated":
				fmt.Fprintf(cmd.OutOrStdout(), "  origin: updated %s\n", result.RemoteURL)
			case "unchanged":
				fmt.Fprintf(cmd.OutOrStdout(), "  origin: %s\n", result.RemoteURL)
			case "skipped":
				fmt.Fprintln(cmd.OutOrStdout(), "  origin: skipped")
			}
			warnNonGitHubOrigin(cmd, name, result.RemoteURL, result.RemoteAction)
			return nil
		},
	}
	cmd.Flags().BoolVar(&noRemote, "no-remote", false, "only set SSH key; do not create/update origin")
	return cmd
}

func warnNonGitHubOrigin(cmd *cobra.Command, profile, remoteURL, action string) {
	if action == "skipped" || strings.TrimSpace(remoteURL) == "" {
		return
	}
	if _, _, ok := remoteurl.ParseOwnerRepo(remoteURL); ok {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"warning: origin is not GitHub (%s); Orca will report no GitHub source.\n", remoteURL)
	fmt.Fprintf(cmd.ErrOrStderr(),
		"  retarget: git-ssh use %s <repo|owner/repo>\n", profile)
}

func (r *Root) unuseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "unuse",
		Aliases: []string{"uu"},
		Short:   "Clear git-ssh settings from this git repo",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := r.checkRepo(); err != nil {
				return err
			}
			if err := apply.Unuse(r.git); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Successfully removed git-ssh profile from current git repository.")
			return nil
		},
	}
}

func (r *Root) currentCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "current",
		Aliases: []string{"c"},
		Short:   "Show active profile in this git repo",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := r.checkRepo(); err != nil {
				return err
			}
			name, err := apply.Current(r.git)
			if err != nil || name == "" {
				// Mirror git-profile: empty => default
				fmt.Fprintln(cmd.OutOrStdout(), "default")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), name)
			return nil
		},
	}
}

func (r *Root) backupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup [file]",
		Short: "Backup profiles to a JSON file",
		Long: multiline(
			"Writes the sidecar profile store to a JSON file.",
			"If no path is given, creates a timestamped file under",
			"~/.config/git-ssh/backups/.",
		),
		Example: multiline(
			`git-ssh backup`,
			`git-ssh backup ./git-ssh-backup.json`,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) == 1 {
				path = args[0]
			} else {
				p, err := config.BackupPath()
				if err != nil {
					return err
				}
				path = p
			}
			if err := r.cfg.Save(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Backed up %d profile(s) to %s\n", r.cfg.Len(), path)
			return nil
		},
	}
}

func (r *Root) restoreCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <file>",
		Short: "Restore profiles from a backup JSON file",
		Long: multiline(
			"Replaces the current sidecar config with the backup,",
			"then rewrites owned Include fragments under ~/.ssh/git-ssh.d/.",
		),
		Example: `git-ssh restore ~/.config/git-ssh/backups/git-ssh-20260727-093000.json`,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]
			incoming := config.New()
			if err := incoming.LoadExisting(backupPath); err != nil {
				return err
			}
			if !force && r.cfg.Len() > 0 {
				return fmt.Errorf("current config has %d profile(s); pass --force to overwrite", r.cfg.Len())
			}

			// Drop include fragments for profiles that will disappear.
			oldNames := r.cfg.Names()
			incomingNames := map[string]struct{}{}
			for _, name := range incoming.Names() {
				incomingNames[name] = struct{}{}
			}
			for _, name := range oldNames {
				if _, keep := incomingNames[name]; !keep {
					_ = include.RemoveProfile(name)
				}
			}

			r.cfg.ReplaceAll(incoming)
			if err := r.saveConfig(); err != nil {
				return err
			}
			for _, name := range r.cfg.Names() {
				p, _ := r.cfg.Lookup(name)
				if err := include.WriteProfile(name, p); err != nil {
					return fmt.Errorf("restored config but include write failed for %q: %w", name, err)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restored %d profile(s) from %s\n", r.cfg.Len(), backupPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite non-empty current config")
	return cmd
}

func (r *Root) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "git-ssh %s (%s) %s\n", r.Version, r.CommitHash, r.CompileDate)
		},
	}
}

func multiline(lines ...string) string {
	return strings.Join(lines, "\n")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
