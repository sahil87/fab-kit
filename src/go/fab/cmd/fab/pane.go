package main

import (
	"errors"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

// paneValidationExitCode maps a pane.ValidatePane failure to the pane-family
// exit-code scheme shared with window-name's tmuxExitCode: 2 = pane missing,
// 3 = any other tmux failure (dead server, bad socket). Classification rides
// on the error value (pane.PaneNotFoundError) — no string matching.
func paneValidationExitCode(err error) int {
	var nf *pane.PaneNotFoundError
	if errors.As(err, &nf) {
		return 2
	}
	return 3
}

func paneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Tmux pane operations",
		Long:  "Tmux pane operations: map, capture, send, process, window-name",
	}

	cmd.PersistentFlags().StringP("server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.")

	cmd.AddCommand(
		paneMapCmd(),
		paneCaptureCmd(),
		paneSendCmd(),
		paneProcessCmd(),
		paneWindowNameCmd(),
	)

	return cmd
}
