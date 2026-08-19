package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// operatorStatePathOverride is used in tests to redirect operator state-file
// I/O to a temp file instead of the real server-keyed XDG state path. It holds
// a full file path (not a directory).
var operatorStatePathOverride string

func operatorTickStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tick-start",
		Short: "Increment tick_count and record last_tick_at in the server-keyed operator state file",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTickStart,
	}
}

func runOperatorTickStart(cmd *cobra.Command, args []string) error {
	yamlPath, err := operatorStatePath()
	if err != nil {
		return err
	}

	// Tolerant whole-file read (missing → empty map); unknown top-level keys
	// (and the owned sections, untouched here) survive the write-back.
	data, err := loadOperatorState(yamlPath)
	if err != nil {
		return err
	}

	// Increment tick_count
	tickCount := 0
	if v, ok := data["tick_count"]; ok {
		switch n := v.(type) {
		case int:
			tickCount = n
		case int64:
			tickCount = int(n)
		case float64:
			tickCount = int(n)
		}
	}
	tickCount++

	// Capture time once so last_tick_at and stdout are consistent
	now := time.Now()

	data["tick_count"] = tickCount
	data["last_tick_at"] = now.UTC().Format(time.RFC3339)

	// Write back atomically via temp+rename (shared helper).
	if err := saveOperatorState(yamlPath, data); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tick: %d\nnow: %s\n", tickCount, now.Format("15:04"))
	return nil
}
