package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ssh-profiles/git-ssh/internal/config"
	"github.com/ssh-profiles/git-ssh/internal/include"
	"github.com/spf13/cobra"
)

func (r *Root) exportCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "export [profile]",
		Aliases: []string{"e"},
		Short:   "Export a profile as JSON",
		Args:    cobra.ExactArgs(1),
		Example: `git-ssh export alice`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := r.cfg.Lookup(args[0])
			if !ok {
				r.exitMissingProfile(cmd, args[0])
				return nil
			}
			data, err := json.Marshal(p)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func (r *Root) importCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "import [profile] [json-values]",
		Aliases: []string{"i"},
		Short:   "Import a profile from JSON",
		Args:    cobra.ExactArgs(2),
		Example: `git-ssh import alice '{"identity_file":"~/.ssh/alice/id_ed25519","github_user":"alice"}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var profile config.Profile
			if err := json.Unmarshal([]byte(args[1]), &profile); err != nil {
				return fmt.Errorf("unable to decode profile values: %w", err)
			}
			if err := r.cfg.Store(args[0], profile); err != nil {
				return err
			}
			if err := r.saveConfig(); err != nil {
				return err
			}
			if err := include.WriteProfile(args[0], profile); err != nil {
				return fmt.Errorf("imported but include write failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully imported `%s` profile.\n", args[0])
			return nil
		},
	}
}