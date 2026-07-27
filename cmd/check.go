package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (r *Root) checkRepo(cmd *cobra.Command) bool {
	if !r.git.IsRepository() {
		fmt.Fprintln(cmd.ErrOrStderr(), "The current working directory is not a valid git repository.")
		return false
	}
	return true
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

func (r *Root) exitMissingProfile(cmd *cobra.Command, name string) {
	fmt.Fprintf(cmd.ErrOrStderr(), "There is no profile with `%s` name\n", name)
	os.Exit(0)
}
