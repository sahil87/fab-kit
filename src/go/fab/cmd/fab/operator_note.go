package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Note verbs (fab-operator.md §4 Notes). Notes are the operator's owned
// surface for cross-cutting narrative state — phase-plan progress, peer
// agreements, corrections, merge-gate dependency waits. The binary owns the
// ids, timestamps, caps, and pruning; the operator states intent through
// flags and never hand-writes the YAML.

const (
	// noteTextCap is the binary-enforced cap on note text (in runes) — notes
	// re-enter operator context on every state read, so unbounded prose
	// defeats the point.
	noteTextCap = 500
	// resolvedNotesCap bounds the resolved-note history; the oldest resolved
	// notes are pruned past it. Open notes are never pruned — notes are
	// decisions, not a dedupe cache.
	resolvedNotesCap = 50
	// openNotesWarnCap is the "do I even need a note?" nudge made mechanical:
	// list warns on stderr above this many open notes.
	openNotesWarnCap = 25
	// noteStaleThreshold flags a note stale (display-only) when its
	// updated_at age exceeds it.
	noteStaleThreshold = 14 * 24 * time.Hour
)

// noteKinds is the kind enum — deliberately no `lesson` kind: durable process
// lessons route via `idea` → a fab change into docs/memory, and the absence
// of the kind is the guard against notes degrading into a reflexive
// scratchpad.
var noteKinds = []string{"dependency_wait", "phase_plan", "coordination", "correction"}

// noteEntry is one `notes` entry — the fab-operator.md §4 Notes schema,
// byte-compatibly. `notes` is a list in creation order, so resolve-time
// pruning (oldest-first) is well-defined without sorting timestamps.
type noteEntry struct {
	ID         string   `yaml:"id" json:"id"`
	Kind       string   `yaml:"kind" json:"kind"`
	Text       string   `yaml:"text" json:"text"`
	Refs       []string `yaml:"refs,omitempty" json:"refs,omitempty"`
	CreatedAt  string   `yaml:"created_at" json:"created_at"`
	UpdatedAt  string   `yaml:"updated_at" json:"updated_at"`
	Resolved   bool     `yaml:"resolved" json:"resolved"`
	ResolvedAt *string  `yaml:"resolved_at" json:"resolved_at"`
}

func validNoteKind(kind string) bool {
	for _, k := range noteKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// noteSeq reads the persisted top-level notes_seq counter (ids are assigned
// from it and never reused after prune), tolerant of YAML int decoding. A
// missing key is 0.
func noteSeq(data map[string]interface{}) int {
	switch v := data["notes_seq"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// readNotes decodes the notes section into its typed creation-order list
// (dropping any in-section drift). A missing/null section yields an empty
// list.
func readNotes(data map[string]interface{}) ([]noteEntry, error) {
	notes := []noteEntry{}
	if err := operatorSection(data, "notes", &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// formatNoteAge renders a note age with day capability — pane.FormatIdleDuration
// caps at hours by contract with its pane callers, while note ages span weeks
// (`⚠ 21d`).
func formatNoteAge(d time.Duration) string {
	secs := int64(d.Seconds())
	if secs < 0 {
		secs = 0
	}
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm", secs/60)
	case secs < 86400:
		return fmt.Sprintf("%dh", secs/3600)
	default:
		return fmt.Sprintf("%dd", secs/86400)
	}
}

// noteAge is the note's age from updated_at; an unparseable timestamp is 0.
func noteAge(n noteEntry, now time.Time) time.Duration {
	t, err := time.Parse(time.RFC3339, n.UpdatedAt)
	if err != nil {
		return 0
	}
	return now.Sub(t)
}

// formatNoteLine renders the shared one-line note shape used by `note list`
// and the `state` OPEN NOTES header: id · kind · age · first line of text. A
// stale note (updated_at older than noteStaleThreshold) carries the
// display-only `⚠ <age>` flag in place of the plain age.
func formatNoteLine(n noteEntry, now time.Time) string {
	age := formatNoteAge(noteAge(n, now))
	if noteAge(n, now) > noteStaleThreshold {
		age = "⚠ " + age
	}
	first, _, _ := strings.Cut(n.Text, "\n")
	return fmt.Sprintf("%s %s %s %s", n.ID, n.Kind, age, first)
}

func operatorNoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Manage narrative-state notes in the operator state file",
	}
	cmd.AddCommand(
		operatorNoteAddCmd(),
		operatorNoteResolveCmd(),
		operatorNoteUpdateCmd(),
		operatorNoteListCmd(),
	)
	return cmd
}

func operatorNoteAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Create a note (binary-assigned n<N> id from the persisted notes_seq)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorNoteAdd,
	}
	cmd.Flags().String("kind", "", "note kind: dependency_wait|phase_plan|coordination|correction (required)")
	cmd.Flags().StringArray("ref", nil, "reference (change id / repo); repeatable")
	_ = cmd.MarkFlagRequired("kind")
	return cmd
}

func runOperatorNoteAdd(cmd *cobra.Command, args []string) error {
	kind, _ := cmd.Flags().GetString("kind")
	if !validNoteKind(kind) {
		return fmt.Errorf("invalid --kind %q (valid: %s)", kind, strings.Join(noteKinds, ", "))
	}
	text := args[0]
	if len([]rune(text)) > noteTextCap {
		return fmt.Errorf("note text over the %d-character cap", noteTextCap)
	}
	refs, _ := cmd.Flags().GetStringArray("ref")

	var id string
	err := mutateOperatorState(func(data map[string]interface{}) error {
		notes, err := readNotes(data)
		if err != nil {
			return err
		}
		seq := noteSeq(data) + 1
		id = fmt.Sprintf("n%d", seq)
		now := nowRFC3339()
		notes = append(notes, noteEntry{
			ID:        id,
			Kind:      kind,
			Text:      text,
			Refs:      refs,
			CreatedAt: now,
			UpdatedAt: now,
		})
		data["notes"] = notes
		data["notes_seq"] = seq
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), id)
	return nil
}

func operatorNoteResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id>",
		Short: "Mark a note resolved (idempotent; prunes resolved notes past the 50-cap, oldest-first)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorNoteResolve,
	}
}

func runOperatorNoteResolve(cmd *cobra.Command, args []string) error {
	id := args[0]
	return mutateOperatorState(func(data map[string]interface{}) error {
		notes, err := readNotes(data)
		if err != nil {
			return err
		}
		found := false
		for i := range notes {
			if notes[i].ID == id {
				found = true
				if !notes[i].Resolved {
					now := nowRFC3339()
					notes[i].Resolved = true
					notes[i].ResolvedAt = &now
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("no note %s", id)
		}
		resolved := 0
		for _, n := range notes {
			if n.Resolved {
				resolved++
			}
		}
		if resolved > resolvedNotesCap {
			// Oldest-first in creation (list) order; open notes are never pruned.
			drop := resolved - resolvedNotesCap
			kept := notes[:0]
			for _, n := range notes {
				if n.Resolved && drop > 0 {
					drop--
					continue
				}
				kept = append(kept, n)
			}
			notes = kept
		}
		data["notes"] = notes
		return nil
	})
}

func operatorNoteUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <id> <text>",
		Short: "Replace a note's text in place (refreshes updated_at)",
		Args:  cobra.ExactArgs(2),
		RunE:  runOperatorNoteUpdate,
	}
}

func runOperatorNoteUpdate(cmd *cobra.Command, args []string) error {
	id, text := args[0], args[1]
	if len([]rune(text)) > noteTextCap {
		return fmt.Errorf("note text over the %d-character cap", noteTextCap)
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		notes, err := readNotes(data)
		if err != nil {
			return err
		}
		for i := range notes {
			if notes[i].ID == id {
				notes[i].Text = text
				notes[i].UpdatedAt = nowRFC3339()
				data["notes"] = notes
				return nil
			}
		}
		return fmt.Errorf("no note %s", id)
	})
}

func operatorNoteListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notes (open by default; --all includes resolved)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorNoteList,
	}
	cmd.Flags().Bool("open", false, "list open notes only (the default)")
	cmd.Flags().Bool("all", false, "include resolved notes")
	cmd.Flags().Bool("json", false, "emit the notes as JSON")
	return cmd
}

func runOperatorNoteList(cmd *cobra.Command, args []string) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	data, err := loadOperatorState(path)
	if err != nil {
		return err
	}
	notes, err := readNotes(data)
	if err != nil {
		return err
	}
	all, _ := cmd.Flags().GetBool("all")
	asJSON, _ := cmd.Flags().GetBool("json")

	openCount := 0
	shown := []noteEntry{}
	for _, n := range notes {
		if !n.Resolved {
			openCount++
		}
		if n.Resolved && !all {
			continue
		}
		shown = append(shown, n)
	}

	if asJSON {
		out, err := json.MarshalIndent(shown, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		w := cmd.OutOrStdout()
		now := time.Now().UTC()
		for _, n := range shown {
			fmt.Fprintln(w, formatNoteLine(n, now))
		}
	}
	if openCount > openNotesWarnCap {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d open notes (>%d) — do you need all of them?\n", openCount, openNotesWarnCap)
	}
	return nil
}
