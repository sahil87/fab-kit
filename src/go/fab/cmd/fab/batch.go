package main

import (
	"github.com/spf13/cobra"
)

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Multi-target batch operations",
	}

	cmd.AddCommand(
		batchNewCmd(),
		batchSwitchCmd(),
		batchArchiveCmd(),
	)

	return cmd
}
