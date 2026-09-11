package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/atomicfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Shared operator state-file IO. Every `fab operator` state subcommand reads
// the whole file tolerantly (unknown TOP-LEVEL keys survive a read-modify-write
// — the tick-start posture, so a legacy hand-drifted file never wedges the
// operator) and re-marshals the OWNED sections (tracked, branch_map,
// clock_override) from typed structs on mutation — an invented field inside an
// owned section can neither be introduced nor survive a mutation of that
// section. All writes go through atomicfile.WriteFile; all timestamps are
// computed here (RFC3339 UTC) — no subcommand accepts one. The legacy
// sections (monitored/watches/autopilot/notes/notes_seq) exist only for the
// first-touch conversion in operator_migrate.go.

// branchMapEntry is one `branch_map` value ({ branch, repo }).
type branchMapEntry struct {
	Branch string `yaml:"branch"`
	Repo   string `yaml:"repo"`
}

// nowRFC3339 is the single timestamp source for every state mutation.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// operatorStatePath resolves the state-file path, honoring the test seam.
func operatorStatePath() (string, error) {
	if operatorStatePathOverride != "" {
		return operatorStatePathOverride, nil
	}
	// server "" → query the operator's own (current) tmux server socket.
	p, err := StatePath("")
	if err != nil {
		return "", fmt.Errorf("cannot determine operator state path: %w", err)
	}
	return p, nil
}

// loadOperatorState reads the whole state file tolerantly. A missing file is
// an empty map (not an error); an unparsable file is a hard error.
func loadOperatorState(path string) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", path, err)
		}
	}
	return data, nil
}

// saveOperatorState writes the whole state file atomically (temp+rename — the
// operator state file is a cold path, so the always-fsync variant is fine).
func saveOperatorState(path string, data map[string]interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal %s: %w", path, err)
	}
	if err := atomicfile.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// mutateOperatorState is the read-modify-write skeleton every mutation verb
// runs: load (tolerant) → legacy conversion (R5) → fn applies typed edits →
// save (atomic). fn errors abort without writing. After a successful save it
// runs the clock side effects (operator_clock.go): when the mutation flipped
// the tracked predicate, the operator-tick cron entry is muted
// (tracked→untracked) or unmuted (untracked→tracked) via `rk cron mute`, then
// the derived schedule is reconciled via `rk cron edit` (B3). The side
// effects are edge-triggered / on-change-only and fail-silent — an
// absent/failing rk never surfaces here, never changes the verb's exit code
// or stdout, and a failed save issues no rk call (the clock never diverges
// from a state that was not persisted).
func mutateOperatorState(fn func(data map[string]interface{}) error) error {
	return mutateOperatorStateClock(fn, true)
}

// mutateOperatorStateClock is mutateOperatorState with the clock side effects
// switchable: tick-start --diff disables the edge trigger and runs the
// level-wise reconciles instead (muteOperatorClockIfUntracked +
// reconcileOperatorSchedule), so one invocation never issues two mutes.
func mutateOperatorStateClock(fn func(data map[string]interface{}) error, syncClock bool) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	data, err := loadOperatorState(path)
	if err != nil {
		return err
	}
	// Legacy files convert on the first read-modify-write, landing in the same
	// atomic write as fn's own mutation (R5); a running autopilot queue refuses.
	// The second pass converts retired kind: fab-change items in a
	// tracked-present file (kind: pane + seeded scope.change/pane_pid).
	if err := convertLegacyOperatorState(data); err != nil {
		return err
	}
	if _, err := convertFabChangeItems(data); err != nil {
		return err
	}
	before := operatorTracked(data)
	if err := fn(data); err != nil {
		return err
	}
	// An expired clock_override is removed in this same write (R13) — the
	// schedule reconcile then derives (and applies) the reverted value.
	expireClockOverride(data, time.Now())
	after := operatorTracked(data)
	if err := saveOperatorState(path, data); err != nil {
		return err
	}
	if syncClock {
		syncOperatorClock(before, after)
		reconcileOperatorSchedule(data)
	}
	return nil
}

// operatorSection decodes an owned top-level section into its typed form
// (dropping any in-section drift). A missing/null section leaves out zero.
func operatorSection(data map[string]interface{}, key string, out interface{}) error {
	raw, ok := data[key]
	if !ok || raw == nil {
		return nil
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("cannot decode %s section: %w", key, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("cannot decode %s section: %w", key, err)
	}
	return nil
}

// validStage gates stage-valued flags against the six pipeline stage names.
func validStage(s string) bool {
	for _, st := range status.AllStages() {
		if st == s {
			return true
		}
	}
	return false
}

// emptyOperatorState is the skeleton `state` persists when the file is missing.
func emptyOperatorState() map[string]interface{} {
	return map[string]interface{}{
		"tracked":    []interface{}{},
		"branch_map": map[string]interface{}{},
	}
}

func operatorStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Print the operator state file (creates the empty skeleton when missing)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorState,
	}
	cmd.Flags().Bool("all", false, "deprecated no-op (the notes section is gone; kind: note items always print)")
	cmd.Flags().Bool("json", false, "print the state as JSON instead of YAML")
	return cmd
}

func runOperatorState(cmd *cobra.Command, args []string) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		// Skeleton-on-missing: persist the empty state, then print it — the
		// binary owns the "create if missing" init step.
		skeleton := emptyOperatorState()
		if err := saveOperatorState(path, skeleton); err != nil {
			return err
		}
		raw, err = yaml.Marshal(skeleton)
		if err != nil {
			return fmt.Errorf("cannot marshal %s: %w", path, err)
		}
	} else {
		// A legacy-shaped file converts on this read-modify-write like on any
		// other verb (R5), refusing while an autopilot queue is running; the
		// read then continues from the converted file. The same convert-and-
		// save fires for a tracked-present file still holding retired
		// kind: fab-change items.
		var probe map[string]interface{}
		if err := yaml.Unmarshal(raw, &probe); err != nil {
			return fmt.Errorf("cannot parse %s: %w", path, err)
		}
		if legacyOperatorState(probe) || hasLegacyFabChangeItems(probe) {
			if _, err := loadOperatorStateUpgraded(path); err != nil {
				return err
			}
			if raw, err = os.ReadFile(path); err != nil {
				return fmt.Errorf("cannot read %s: %w", path, err)
			}
		}
	}

	w := cmd.OutOrStdout()
	asJSON, _ := cmd.Flags().GetBool("json")

	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("cannot parse %s: %w", path, err)
	}
	items, err := decodeTrackedItems(data)
	if err != nil {
		return err
	}
	notes := []trackedItem{}
	for _, it := range items {
		if it.Kind == kindNote {
			notes = append(notes, it)
		}
	}

	if !asJSON {
		// OPEN NOTES header — human output only: comment-prefixed so stdout
		// stays parseable YAML for yq consumers. Omitted when no kind: note
		// items exist.
		if len(notes) > 0 {
			now := time.Now().UTC()
			fmt.Fprintf(w, "# OPEN NOTES (%d)\n", len(notes))
			for _, n := range notes {
				fmt.Fprintf(w, "# %s\n", formatTrackNoteLine(n, now))
			}
		}
		_, err = w.Write(raw)
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot convert %s to JSON: %w", path, err)
	}
	fmt.Fprintln(w, string(out))
	return nil
}

func operatorBranchMapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch-map",
		Short: "Manage the operator state file's branch_map",
	}
	cmd.AddCommand(operatorBranchMapRmCmd())
	return cmd
}

func operatorBranchMapRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [<change-id>]",
		Short: "Remove a branch_map entry (or all with --all)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runOperatorBranchMapRm,
	}
	cmd.Flags().Bool("all", false, "clear the entire branch_map")
	return cmd
}

func runOperatorBranchMapRm(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")
	if all && len(args) > 0 {
		return fmt.Errorf("branch-map rm takes either a change-id or --all, not both")
	}
	if !all && len(args) == 0 {
		return fmt.Errorf("branch-map rm requires a change-id or --all")
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		if all {
			data["branch_map"] = map[string]interface{}{}
			return nil
		}
		bm := map[string]branchMapEntry{}
		if err := operatorSection(data, "branch_map", &bm); err != nil {
			return err
		}
		id := args[0]
		if _, ok := bm[id]; !ok {
			return fmt.Errorf("no branch_map entry for %s", id)
		}
		delete(bm, id)
		data["branch_map"] = bm
		return nil
	})
}
