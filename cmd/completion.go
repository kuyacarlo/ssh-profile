package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func (r *Root) completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for git-ssh.

Bash:

  $ source <(git-ssh completion bash)

Zsh:

  $ git-ssh completion zsh > ~/.zsh/completions/_git-ssh

Fish:

  $ git-ssh completion fish | source
`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		Run: func(_ *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = r.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				_ = r.GenZshCompletion(os.Stdout)
			case "fish":
				_ = r.GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = r.GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}
