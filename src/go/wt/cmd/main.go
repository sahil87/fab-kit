package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	wt "github.com/wvrdz/fab-kit/src/go/wt/internal/worktree"
)

func main() {
	root := &cobra.Command{
		Use:   "wt",
		Short: "Git worktree management — create, list, open, delete worktrees",
		Long: `Git worktree management — create, list, open, delete worktrees.

Shell wrapper (recommended):
  To enable the "Open here" menu option (cd into a worktree in the current
  shell), add this function to your shell profile (~/.bashrc or ~/.zshrc):

    wt() {
      local line last rc
      while IFS= read -r line; do
        printf '%s\n' "$line"
        last=$line
      done < <(command wt "$@")
      rc=$?
      if [[ "$last" == cd\ * ]]; then
        eval "$last"
      fi
      return $rc
    }`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		createCmd(),
		listCmd(),
		openCmd(),
		deleteCmd(),
		initCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(wt.ExitGeneralError)
	}
}
