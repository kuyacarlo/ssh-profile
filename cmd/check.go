package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (r *Root) checkRepo(cmd *cobra.Command) error {
	if !r.git.IsRepository() {
		return fmt.Errorf("the current working directory is not a valid git repository")
	}
	return nil
}

func (r *Root) checkProfiles(cmd *cobra.Command) bool {
	if r.cfg.Len() == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "There are no available profiles.")
		fmt.Fprintln(cmd.OutOrStdout(), "To add a new profile, use the following command:")
		fmt.Fprintln(cmd.OutOrStdout(), "$ git-ssh add")
		return false
	}
	return true
}

func missingProfileError(name string) error {
	return fmt.Errorf("there is no profile with `%s` name", name)
}
