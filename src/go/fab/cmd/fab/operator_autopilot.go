package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// validAutopilotModes are the accepted --mode values for `autopilot start`;
// the first entry is the default and the back-compat fill for an absent
// `mode` field in a pre-existing state file.
var validAutopilotModes = []string{"stack-then-review", "merge-on-complete", "stacked-prs"}

func validAutopilotMode(mode string) bool {
	for _, m := range validAutopilotModes {
		if m == mode {
			return true
		}
	}
	return false
}

// Autopilot lifecycle verbs (fab-operator.md §6 Autopilot). start sets the
// queue running; pause/resume flip state; advance promotes the next entry
// (retaining queue/completed on exhaustion so the completion summary can still
// read them); stop clears the whole block to `autopilot: null`. Verbs other
// than start error when no queue is active.

func operatorAutopilotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autopilot",
		Short: "Manage the operator autopilot queue",
	}
	start := &cobra.Command{
		Use:   "start",
		Short: "Start an autopilot queue (--queue id,id,...)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorAutopilotStart,
	}
	start.Flags().StringSlice("queue", nil, "comma-separated change IDs, in order (required)")
	_ = start.MarkFlagRequired("queue")
	start.Flags().String("mode", validAutopilotModes[0], "merge mode: "+strings.Join(validAutopilotModes, " | "))

	advance := &cobra.Command{
		Use:   "advance",
		Short: "Complete (or --skip) the current change and promote the next queue entry",
		Args:  cobra.NoArgs,
		RunE:  runOperatorAutopilotAdvance,
	}
	advance.Flags().Bool("skip", false, "drop the current change without recording it as completed")

	cmd.AddCommand(
		start,
		autopilotSimpleCmd("pause", "Pause the active autopilot queue", runOperatorAutopilotPause),
		autopilotSimpleCmd("resume", "Resume a paused autopilot queue", runOperatorAutopilotResume),
		advance,
		autopilotSimpleCmd("stop", "Clear the autopilot block entirely", runOperatorAutopilotStop),
	)
	return cmd
}

func autopilotSimpleCmd(use, short string, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs, RunE: run}
}

// loadAutopilot decodes the autopilot section; requireActive errors when no
// queue is active (block absent or state cleared). An absent `mode` field in
// a pre-existing state file reads as the default mode (tolerant read — the
// typed re-marshal writes the field on the next mutation).
func loadAutopilot(data map[string]interface{}, requireActive bool) (*autopilotState, error) {
	ap := &autopilotState{}
	if err := operatorSection(data, "autopilot", ap); err != nil {
		return nil, err
	}
	if ap.Mode == "" {
		ap.Mode = validAutopilotModes[0]
	}
	if requireActive && (data["autopilot"] == nil || ap.State == nil) {
		return nil, fmt.Errorf("no autopilot queue is active")
	}
	return ap, nil
}

func runOperatorAutopilotStart(cmd *cobra.Command, args []string) error {
	queue, _ := cmd.Flags().GetStringSlice("queue")
	if len(queue) == 0 {
		return fmt.Errorf("--queue must name at least one change ID")
	}
	mode, _ := cmd.Flags().GetString("mode")
	if !validAutopilotMode(mode) {
		return fmt.Errorf("unknown --mode %q (valid: %s)", mode, strings.Join(validAutopilotModes, ", "))
	}
	running := "running"
	current := queue[0]
	return mutateOperatorState(func(data map[string]interface{}) error {
		data["autopilot"] = &autopilotState{
			Queue:     queue,
			Current:   &current,
			Completed: []string{},
			State:     &running,
			Mode:      mode,
		}
		return nil
	})
}

func runOperatorAutopilotPause(cmd *cobra.Command, args []string) error {
	return mutateOperatorState(func(data map[string]interface{}) error {
		ap, err := loadAutopilot(data, true)
		if err != nil {
			return err
		}
		paused := "paused"
		ap.State = &paused
		data["autopilot"] = ap
		return nil
	})
}

func runOperatorAutopilotResume(cmd *cobra.Command, args []string) error {
	return mutateOperatorState(func(data map[string]interface{}) error {
		ap, err := loadAutopilot(data, true)
		if err != nil {
			return err
		}
		running := "running"
		ap.State = &running
		data["autopilot"] = ap
		return nil
	})
}

func runOperatorAutopilotAdvance(cmd *cobra.Command, args []string) error {
	skip, _ := cmd.Flags().GetBool("skip")
	return mutateOperatorState(func(data map[string]interface{}) error {
		ap, err := loadAutopilot(data, true)
		if err != nil {
			return err
		}
		if ap.Current == nil {
			return fmt.Errorf("no autopilot queue is active")
		}
		if !skip {
			ap.Completed = append(ap.Completed, *ap.Current)
		}
		// Promote the next queue entry; on exhaustion retain queue/completed
		// (the completion summary reads them) with current/state null.
		next := -1
		for i, id := range ap.Queue {
			if id == *ap.Current && i+1 < len(ap.Queue) {
				next = i + 1
				break
			}
		}
		if next >= 0 {
			n := ap.Queue[next]
			ap.Current = &n
		} else {
			ap.Current = nil
			ap.State = nil
		}
		data["autopilot"] = ap
		return nil
	})
}

func runOperatorAutopilotStop(cmd *cobra.Command, args []string) error {
	return mutateOperatorState(func(data map[string]interface{}) error {
		if data["autopilot"] == nil {
			return fmt.Errorf("no autopilot queue is active")
		}
		data["autopilot"] = nil
		return nil
	})
}
