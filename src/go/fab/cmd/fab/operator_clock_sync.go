package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// errOperatorClockUnresolved is `clock sync`'s one loud failure: the entry
// could neither be resolved nor seeded (rk broken, an unresolved tie, or a
// failed add). The reconcile side effects riding other verbs stay
// fail-silent; this verb's whole purpose is to answer "what is the clock",
// so it says when it cannot. Its non-zero exit is the operator skill's
// missing-entry STOP.
var errOperatorClockUnresolved = errors.New("could not resolve or seed the operator-tick cron entry")

// operatorClockDoc is the five-key document `clock sync` prints — exactly
// what the ready line and the frame header consume, in a fixed order.
type operatorClockDoc struct {
	ID              string `yaml:"id"`
	ScheduleSummary string `yaml:"schedule_summary"`
	Deliver         string `yaml:"deliver"`
	Muted           bool   `yaml:"muted"`
	MutedUntil      *int64 `yaml:"muted_until"`
}

func operatorClockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "The operator-tick cron entry: sync (seed-if-missing, reconcile, print)",
		Long: "Verbs over the operator-tick cron entry fab's clock reconcile owns end to\n" +
			"end. `sync` seeds the entry when the server has none, reconciles the mute\n" +
			"against the tracked set in both directions, applies the derived schedule,\n" +
			"and prints the resolved row — one command for the operator to heal and read.",
	}
	cmd.AddCommand(operatorClockSyncCmd())
	return cmd
}

func operatorClockSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Seed the operator-tick cron entry when missing, reconcile mute and schedule, print the row",
		Long: "Runs the clock reconcile once without touching the tick counter and prints the\n" +
			"resolved operator-tick entry as YAML: id, schedule_summary, deliver, muted,\n" +
			"muted_until (null unless a lease is live). Seeds the entry (one `rk cron add`\n" +
			"with the full steady-state shape) when `rk cron list --json` shows no\n" +
			"role:operator row; mutes an unmuted entry when nothing is tracked and clears a\n" +
			"standing mute or lease when work is tracked; applies the derived schedule via\n" +
			"one `rk cron edit` only when the live entry differs. Exits non-zero with one\n" +
			"stderr line when the entry can neither be resolved nor seeded.",
		Args: cobra.NoArgs,
		RunE: runOperatorClockSync,
	}
}

func runOperatorClockSync(cmd *cobra.Command, args []string) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	// Read the state as-is: this verb never creates the skeleton (Init step 1's
	// `fab operator state` does), so an absent file is a plain error rather
	// than a silently empty — hence untracked, hence muting — state.
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("operator state file not found at %s — run `fab operator state` first", path)
		}
		return fmt.Errorf("cannot stat %s: %w", path, err)
	}
	// Every `fab operator` verb converts a legacy-shaped file on read; a raw
	// load would see no `tracked` items in a v10 file and mute a live clock.
	data, err := loadOperatorStateUpgraded(path)
	if err != nil {
		return err
	}
	row, ok := ensureOperatorCronRow()
	if !ok {
		return errOperatorClockUnresolved
	}
	reconcileOperatorMute(data, row)
	reconcileOperatorSchedule(data)
	row, ok = resolveOperatorCronRow()
	if !ok {
		return errOperatorClockUnresolved
	}
	doc := operatorClockDoc{
		ID:              row.ID,
		ScheduleSummary: row.ScheduleSummary,
		Deliver:         row.Deliver,
		Muted:           row.Muted,
	}
	if row.MutedUntil != 0 {
		until := row.MutedUntil
		doc.MutedUntil = &until
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("cannot render the clock document: %w", err)
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), string(out))
	return err
}
