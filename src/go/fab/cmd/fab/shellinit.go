package main

import (
	"fmt"

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
			out := cmd.OutOrStdout()
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return fmt.Errorf("unsupported shell %q: must be one of %v", args[0], shellInitSupportedShells)
			}
		},
	}
}
