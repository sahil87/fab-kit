package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

func dispatchLogsCmd() *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "logs <change> <stage>",
		Short: "Print the dispatch log (combined stdout+stderr)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchLogs(cmd, args[0], args[1], tail)
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 0, "Print only the last N lines (0 = all)")
	return cmd
}

func runDispatchLogs(cmd *cobra.Command, changeArg, stage string, tail int) error {
	dir, _, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(dispatch.LogPath(dir, stage))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dispatch log for %s/%s", changeArg, stage)
		}
		return fmt.Errorf("read dispatch log: %w", err)
	}

	if tail > 0 {
		data = dispatch.Tail(data, tail)
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}
