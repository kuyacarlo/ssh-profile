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
			"git-ssh is a companion to git-profile.",
			"Profiles store which SSH key + GitHub user to use.",
			"`use` sets core.sshCommand and wires origin to",
			"git@github.com:<user>/<repo>.git so Orca/ADE keeps working.",
			"",
			"Typical pairing:",
			"  git-profile use alice",
			"  git-ssh use alice",
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
		r.unuseCmd(),
		r.currentCmd(),
		r.backupCmd(),
		r.restoreCmd(),
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
		Use:   "add <profile>",
		Short: "Add or update a profile",
		Example: multiline(
			`git-ssh add alice --identity ~/.ssh/alice/id_ed25519 --github-user alice`,
			`git-ssh add bob --identity ~/.ssh/bob/id_ed25519 --github-user bob`,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if profile.IdentityFile == "" {
				return fmt.Errorf("--identity is required for new profiles")
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
			if err := r.cfg.Store(name, profile); err != nil {
				return err
			}
			if err := r.saveConfig(); err != nil {
				return err
			}
			if err := include.WriteProfile(name, profile); err != nil {
				return fmt.Errorf("profile saved but include write failed: %w", err)
			}
			action := "Updated"
			if !exists {
				action = "Added"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s profile %q\n", action, name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&identity, "identity", "i", "", "path to private key")
	cmd.Flags().StringVarP(&githubUser, "github-user", "u", "", "GitHub username/owner for origin remotes")
	cmd.Flags().StringVar(&alias, "alias", "", "optional SSH Host alias (not github.com)")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "extra SSH config Key=Value")
	return cmd
}

func (r *Root) listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := r.cfg.Names()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles.")
				return nil
			}
			current := ""
			if r.git.IsRepository() {
				if name, err := apply.Current(r.git); err == nil {
					current = name
				}
			}
			for _, name := range names {
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
		Use:   "show <profile>",
		Short: "Show a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := r.cfg.Lookup(args[0])
			if !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\n", args[0])
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
		Use:     "del <profile>",
		Aliases: []string{"rm", "delete"},
		Short:   "Delete a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !r.cfg.DeleteProfile(name) {
				return fmt.Errorf("profile %q not found", name)
			}
			if err := r.saveConfig(); err != nil {
				return err
			}
			_ = include.RemoveProfile(name)
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted profile %q\n", name)
			return nil
		},
	}
}

func (r *Root) useCmd() *cobra.Command {
	var noRemote bool

	cmd := &cobra.Command{
		Use:   "use <profile> [repo|owner/repo]",
		Short: "Apply profile key + origin remote to this git repo",
		Long: multiline(
			"Sets core.sshCommand to the profile's private key (Orca-safe).",
			"",
			"Remote target (optional 2nd arg):",
			"  demo-repo              → git@github.com:<github_user>/demo-repo.git",
			"  example-org/demo-repo    → git@github.com:example-org/demo-repo.git",
			"",
			"With no 2nd arg: create origin from directory name, or normalize an",
			"existing github.com / *.github.com remote onto github.com.",
		),
		Example: multiline(
			`git-ssh use alice`,
			`git-ssh use alice demo-repo`,
			`git-ssh use alice example-org/demo-repo`,
			`git-ssh use alice --no-remote`,
		),
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			target := ""
			if len(args) == 2 {
				target = args[1]
			}
			p, ok := r.cfg.Lookup(name)
			if !ok {
				return fmt.Errorf("profile %q not found", name)
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
			fmt.Fprintf(cmd.OutOrStdout(), "Using profile %q\n", name)
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
			return nil
		},
	}
	cmd.Flags().BoolVar(&noRemote, "no-remote", false, "only set SSH key; do not create/update origin")
	return cmd
}

func (r *Root) unuseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unuse",
		Short: "Clear git-ssh settings from this git repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := apply.Unuse(r.git); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Cleared git-ssh profile from this repository")
			return nil
		},
	}
}

func (r *Root) currentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show active profile in this git repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, err := apply.Current(r.git)
			if err != nil || name == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No git-ssh profile active in this repository")
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
