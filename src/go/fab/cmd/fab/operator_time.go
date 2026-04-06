package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func operatorTimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "Print current time and optionally next-tick time",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTime,
	}
	cmd.Flags().String("interval", "", "Duration until next tick (e.g. 3m). If given, outputs next: HH:MM")
	return cmd
}

func runOperatorTime(cmd *cobra.Command, args []string) error {
	interval, _ := cmd.Flags().GetString("interval")

	var d time.Duration
	if interval != "" {
		var err error
		d, err = time.ParseDuration(interval)
		if err != nil {
			return fmt.Errorf("invalid --interval %q: %v", interval, err)
		}
	}

	now := time.Now()
	fmt.Fprintf(cmd.OutOrStdout(), "now: %s\n", now.Format("15:04"))

	if interval == "" {
		return nil
	}

	next := now.Add(d)
	fmt.Fprintf(cmd.OutOrStdout(), "next: %s\n", next.Format("15:04"))
	return nil
}
