package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

func dispatchStatusCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "status <change> <stage>",
		Short: "Report the dispatch state: running / done / failed / failed (no-result) / orphaned",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchStatus(cmd, args[0], args[1], jsonFlag)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// dispatchStatusJSON is the --json output shape for `fab dispatch status`.
type dispatchStatusJSON struct {
	Change string `json:"change"`
	Stage  string `json:"stage"`
	State  string `json:"state"`
	PID    int    `json:"pid"`
	PGID   int    `json:"pgid"`
	Exit   *int   `json:"exit,omitempty"`
}

func runDispatchStatus(cmd *cobra.Command, changeArg, stage string, jsonFlag bool) error {
	dir, id, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	rec, err := dispatch.Load(dir, stage)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dispatch for %s/%s (run `fab dispatch start` first)", changeArg, stage)
		}
		return err
	}

	exitPresent, exitCode, err := dispatch.ReadExit(dir, stage)
	if err != nil {
		return err
	}
	resultPresent := dispatch.ResultPresent(dir, stage)
	alive := dispatch.Alive(rec.PID)
	state := dispatch.DeriveState(exitPresent, exitCode, resultPresent, alive)

	if jsonFlag {
		out := dispatchStatusJSON{
			Change: id,
			Stage:  stage,
			State:  string(state),
			PID:    rec.PID,
			PGID:   rec.PGID,
		}
		if exitPresent {
			out.Exit = &exitCode
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(state))
	return nil
}
