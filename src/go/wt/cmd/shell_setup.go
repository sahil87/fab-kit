package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// ShellWrapperFunc is the shell wrapper function output by shell-setup.
// Defined as a constant to avoid duplication between the subcommand and tests.
const ShellWrapperFunc = `wt() {
  local _wt_cd _wt_rc
  _wt_cd=$(mktemp "${TMPDIR:-/tmp}/wt-cd.XXXXXX")
  WT_CD_FILE="$_wt_cd" command wt "$@"
  _wt_rc=$?
  if [[ -s "$_wt_cd" ]]; then
    local _wt_dir
    _wt_dir=$(<"$_wt_cd")
    cd -- "$_wt_dir" || true
  fi
  rm -f "$_wt_cd"
  return $_wt_rc
}
export WT_WRAPPER=1
`

func shellSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell-setup",
		Short: "Output shell wrapper function for eval",
		Long: `Output a shell wrapper function suitable for eval in your shell profile.

Usage:
  eval "$(wt shell-setup)"

Add the above line to your ~/.bashrc or ~/.zshrc to enable the "Open here"
menu option, which changes the current shell's working directory to the
selected worktree.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := filepath.Base(os.Getenv("SHELL"))

			switch shell {
			case "bash", "zsh", ".", "":
				// Supported shells or unset SHELL — output without warning
			default:
				fmt.Fprintf(os.Stderr, "warning: unsupported shell %q — outputting bash/zsh wrapper\n", shell)
			}

			os.Stdout.WriteString(ShellWrapperFunc)
			return nil
		},
	}

	return cmd
}
