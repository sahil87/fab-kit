package main

import (
	"github.com/spf13/cobra"
)

// shellInitSupportedShells enumerates the shells `shell-init` can emit
// completion scripts for. Kept in one place so the validator and the
// dispatch switch agree.
var shellInitSupportedShells = []string{"bash", "zsh", "fish"}

func shellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "shell-init <bash|zsh|fish>",
		Short:     "Emit shell completion script for sourcing (alias for 'completion <shell>')",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: shellInitSupportedShells,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Args validation above (cobra.OnlyValidArgs + ValidArgs) guarantees
			// args[0] is one of shellInitSupportedShells before RunE is invoked,
			// so no default branch is needed.
			out := cmd.OutOrStdout()
			root := cmd.Root()
			// Cobra's built-in `completion <shell>` subcommand uses
			// GenBashCompletionV2 (with descriptions enabled) for bash; using
			// GenBashCompletion (v1) here would diverge from `fab completion
			// bash`. zsh and fish use the only public generator each exposes.
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			}
			return nil
		},
	}
}
