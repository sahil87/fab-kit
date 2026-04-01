package main

import (
	"github.com/spf13/cobra"
)

func paneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Tmux pane observation and interaction",
	}

	cmd.AddCommand(
		paneMapCmd(),
		paneCaptureCmd(),
		paneSendCmd(),
		paneProcessCmd(),
	)

	return cmd
}
