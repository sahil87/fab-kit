package main

import (
	"github.com/spf13/cobra"
)

func paneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Tmux pane operations",
		Long:  "Tmux pane operations: map, capture, send, process",
	}

	cmd.PersistentFlags().StringP("server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.")

	cmd.AddCommand(
		paneMapCmd(),
		paneCaptureCmd(),
		paneSendCmd(),
		paneProcessCmd(),
	)

	return cmd
}
