package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// runNoteCmd executes a note/state command capturing stdout and stderr
// alongside the error.
func runNoteCmd(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

// readNotesList decodes the notes section into typed entries.
func readNotesList(t *testing.T, path string) []noteEntry {
	t.Helper()
	notes, err := readNotes(readStateFile(t, path))
	if err != nil {
		t.Fatalf("decode notes: %v", err)
	}
	return notes
}

// makeNotesYAML seeds a state file with notes_seq and resolvedCount resolved
// notes (n1..nR) followed by openCount open notes (nR+1..nR+O), all stamped
// 2026-01-01.
func makeNotesYAML(seq, resolvedCount, openCount int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "notes_seq: %d\nnotes:\n", seq)
	for i := 1; i <= resolvedCount+openCount; i++ {
		resolved := i <= resolvedCount
		fmt.Fprintf(&b, "  - id: n%d\n    kind: phase_plan\n    text: note %d\n", i, i)
		b.WriteString("    created_at: \"2026-01-01T00:00:00Z\"\n    updated_at: \"2026-01-01T00:00:00Z\"\n")
		fmt.Fprintf(&b, "    resolved: %t\n", resolved)
		if resolved {
			b.WriteString("    resolved_at: \"2026-01-01T00:00:00Z\"\n")
		} else {
			b.WriteString("    resolved_at: null\n")
		}
	}
	return b.String()
}

func TestOperatorNoteAdd_AssignsPersistedSeqIDs(t *testing.T) {
	path := withOperatorState(t, "")

	out, _, err := runNoteCmd(t, operatorNoteAddCmd(),
		"--kind", "coordination", "--ref", "s2gw", "--ref", "fab-kit", "merge sequence pos 1/4")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if out != "n1\n" {
		t.Errorf("stdout = %q, want %q", out, "n1\n")
	}
	out, _, err = runNoteCmd(t, operatorNoteAddCmd(), "--kind", "phase_plan", "second note")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if out != "n2\n" {
		t.Errorf("stdout = %q, want %q", out, "n2\n")
	}

	data := readStateFile(t, path)
	if data["notes_seq"] != 2 {
		t.Errorf("notes_seq = %v, want 2", data["notes_seq"])
	}
	notes := readNotesList(t, path)
	if len(notes) != 2 {
		t.Fatalf("notes len = %d, want 2", len(notes))
	}
	n := notes[0]
	if n.ID != "n1" || n.Kind != "coordination" || n.Text != "merge sequence pos 1/4" {
		t.Errorf("note identity fields wrong: %+v", n)
	}
	if len(n.Refs) != 2 || n.Refs[0] != "s2gw" || n.Refs[1] != "fab-kit" {
		t.Errorf("refs = %v, want [s2gw fab-kit]", n.Refs)
	}
	if n.Resolved || n.ResolvedAt != nil {
		t.Errorf("new note must be unresolved: %+v", n)
	}
	for _, ts := range []string{n.CreatedAt, n.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("timestamp %q not RFC3339: %v", ts, err)
		}
	}
}

func TestOperatorNoteAdd_ValidationErrors(t *testing.T) {
	path := withOperatorState(t, "")

	if _, _, err := runNoteCmd(t, operatorNoteAddCmd(), "--kind", "lesson", "durable lesson"); err == nil {
		t.Error("unknown kind (lesson is deliberately absent) must error")
	}
	overCap := strings.Repeat("x", noteTextCap+1)
	if _, _, err := runNoteCmd(t, operatorNoteAddCmd(), "--kind", "phase_plan", overCap); err == nil {
		t.Error("over-cap text must error")
	}
	// Validation failures write nothing.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("state file must not be created on validation failure: %v", err)
	}
	// Exactly at the cap succeeds.
	if _, _, err := runNoteCmd(t, operatorNoteAddCmd(), "--kind", "phase_plan", strings.Repeat("x", noteTextCap)); err != nil {
		t.Errorf("at-cap text must succeed: %v", err)
	}
}

func TestOperatorNoteResolve_IdempotentPruneAndUnknown(t *testing.T) {
	path := withOperatorState(t, makeNotesYAML(80, 50, 30))

	if _, _, err := runNoteCmd(t, operatorNoteResolveCmd(), "n51"); err != nil {
		t.Fatalf("resolve n51: %v", err)
	}
	notes := readNotesList(t, path)
	// 50 pre-resolved + this resolve = 51 > 50-cap → the oldest resolved note
	// (n1) is pruned: 79 entries, all 30 open surviving.
	if len(notes) != 79 {
		t.Fatalf("notes len = %d, want 79 (oldest resolved pruned)", len(notes))
	}
	resolved, open := 0, 0
	ids := map[string]bool{}
	var n51 noteEntry
	for _, n := range notes {
		ids[n.ID] = true
		if n.Resolved {
			resolved++
		} else {
			open++
		}
		if n.ID == "n51" {
			n51 = n
		}
	}
	if resolved != 50 || open != 29 {
		t.Errorf("resolved/open = %d/%d, want 50/29 (30 open − n51, now resolved) — open notes must never be pruned", resolved, open)
	}
	if ids["n1"] {
		t.Error("oldest resolved note n1 must be pruned past the 50-cap")
	}
	for i := 52; i <= 80; i++ {
		if !ids[fmt.Sprintf("n%d", i)] {
			t.Errorf("open note n%d must survive pruning", i)
		}
	}
	if !n51.Resolved || n51.ResolvedAt == nil {
		t.Fatalf("n51 must be resolved with resolved_at: %+v", n51)
	}
	if seq := readStateFile(t, path)["notes_seq"]; seq != 80 {
		t.Errorf("notes_seq = %v, want 80 (untouched by prune)", seq)
	}

	// Idempotent re-resolve: exit 0, resolved_at unchanged.
	firstResolvedAt := *n51.ResolvedAt
	if _, _, err := runNoteCmd(t, operatorNoteResolveCmd(), "n51"); err != nil {
		t.Fatalf("re-resolve n51: %v", err)
	}
	for _, n := range readNotesList(t, path) {
		if n.ID == "n51" && (n.ResolvedAt == nil || *n.ResolvedAt != firstResolvedAt) {
			t.Errorf("resolved_at changed on idempotent re-resolve: %v", n.ResolvedAt)
		}
	}

	if _, _, err := runNoteCmd(t, operatorNoteResolveCmd(), "n999"); err == nil {
		t.Error("unknown id must error")
	}

	// Ids are never reused after prune: the next add is n81, not n1.
	out, _, err := runNoteCmd(t, operatorNoteAddCmd(), "--kind", "correction", "post-prune note")
	if err != nil {
		t.Fatalf("post-prune add: %v", err)
	}
	if out != "n81\n" {
		t.Errorf("post-prune id = %q, want n81 (pruned ids never reused)", out)
	}
}

func TestOperatorNoteUpdate(t *testing.T) {
	path := withOperatorState(t, `notes_seq: 1
notes:
  - id: n1
    kind: phase_plan
    text: phase 1 of 3
    created_at: "2026-01-01T00:00:00Z"
    updated_at: "2026-01-01T00:00:00Z"
    resolved: false
    resolved_at: null
`)

	if _, _, err := runNoteCmd(t, operatorNoteUpdateCmd(), "n1", "phase 2 of 3"); err != nil {
		t.Fatalf("update: %v", err)
	}
	n := readNotesList(t, path)[0]
	if n.Text != "phase 2 of 3" {
		t.Errorf("text = %q, want replaced", n.Text)
	}
	if n.CreatedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at must be untouched: %q", n.CreatedAt)
	}
	if n.UpdatedAt == "2026-01-01T00:00:00Z" {
		t.Error("updated_at must move forward on update")
	}
	if _, _, err := runNoteCmd(t, operatorNoteUpdateCmd(), "n999", "x"); err == nil {
		t.Error("unknown id must error")
	}
	if _, _, err := runNoteCmd(t, operatorNoteUpdateCmd(), "n1", strings.Repeat("x", noteTextCap+1)); err == nil {
		t.Error("over-cap text must error")
	}
}

func TestOperatorNoteList(t *testing.T) {
	stale := time.Now().UTC().Add(-21 * 24 * time.Hour).Format(time.RFC3339)
	fresh := time.Now().UTC().Format(time.RFC3339)
	withOperatorState(t, fmt.Sprintf(`notes_seq: 3
notes:
  - id: n1
    kind: dependency_wait
    text: "merge-gate wait\nsecond line never renders"
    created_at: "%[1]s"
    updated_at: "%[1]s"
    resolved: false
    resolved_at: null
  - id: n2
    kind: phase_plan
    text: phase 2 of 3
    created_at: "%[2]s"
    updated_at: "%[2]s"
    resolved: false
    resolved_at: null
  - id: n3
    kind: correction
    text: fixed earlier conclusion
    created_at: "%[2]s"
    updated_at: "%[2]s"
    resolved: true
    resolved_at: "%[2]s"
`, stale, fresh))

	// Default is --open: resolved excluded; stale note flagged; first line only.
	out, _, err := runNoteCmd(t, operatorNoteListCmd())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("default list lines = %d, want 2 (open only):\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "n1 dependency_wait ⚠ 21d merge-gate wait") {
		t.Errorf("stale note line wrong: %q", lines[0])
	}
	if strings.Contains(lines[0], "second line") {
		t.Errorf("only the text's first line may render: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "n2 phase_plan ") || strings.Contains(lines[1], "⚠") {
		t.Errorf("fresh note line wrong: %q", lines[1])
	}

	out, _, err = runNoteCmd(t, operatorNoteListCmd(), "--all")
	if err != nil {
		t.Fatalf("list --all: %v", err)
	}
	if len(strings.Split(strings.TrimRight(out, "\n"), "\n")) != 3 || !strings.Contains(out, "n3 correction") {
		t.Errorf("--all must include resolved:\n%s", out)
	}

	// --json: clean machine output — parses, no decoration.
	out, _, err = runNoteCmd(t, operatorNoteListCmd(), "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var decoded []noteEntry
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("--json output must parse as JSON: %v\n%s", err, out)
	}
	if len(decoded) != 2 || decoded[0].ID != "n1" || decoded[1].ID != "n2" {
		t.Errorf("--json default must be the open notes: %+v", decoded)
	}
}

func TestOperatorNoteList_WarnsAboveOpenCap(t *testing.T) {
	withOperatorState(t, makeNotesYAML(openNotesWarnCap+1, 0, openNotesWarnCap+1))

	out, errOut, err := runNoteCmd(t, operatorNoteListCmd())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(errOut, "26 open notes") {
		t.Errorf("stderr must warn above %d open notes, got %q", openNotesWarnCap, errOut)
	}
	if strings.Contains(out, "warning") {
		t.Errorf("stdout must stay clean (warning is stderr-only):\n%s", out)
	}
	if got := len(strings.Split(strings.TrimRight(out, "\n"), "\n")); got != 26 {
		t.Errorf("stdout lines = %d, want 26", got)
	}
}

// TestOperatorNote_UnknownTopLevelKeysSurvive pins the tolerant-read posture
// for the note verbs: a legacy hand-written plan_queue key (and any other
// unknown top-level key) round-trips every verb unchanged.
func TestOperatorNote_UnknownTopLevelKeysSurvive(t *testing.T) {
	path := withOperatorState(t, "plan_queue:\n  - s2gw\n  - c2auto\ncustom_top: keep-me\nnotes_seq: 0\n")

	if _, _, err := runNoteCmd(t, operatorNoteAddCmd(), "--kind", "phase_plan", "queue tracking"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, _, err := runNoteCmd(t, operatorNoteUpdateCmd(), "n1", "queue tracking v2"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, _, err := runNoteCmd(t, operatorNoteResolveCmd(), "n1"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, _, err := runNoteCmd(t, operatorNoteListCmd()); err != nil {
		t.Fatalf("list: %v", err)
	}

	data := readStateFile(t, path)
	queue, ok := data["plan_queue"].([]interface{})
	if !ok || len(queue) != 2 || queue[0] != "s2gw" || queue[1] != "c2auto" {
		t.Errorf("legacy plan_queue did not round-trip: %v", data["plan_queue"])
	}
	if data["custom_top"] != "keep-me" {
		t.Errorf("unknown top-level key not preserved: %v", data["custom_top"])
	}
}

func TestOperatorState_OpenNotesHeader(t *testing.T) {
	updated := time.Now().UTC().Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	seed := fmt.Sprintf(`monitored: {}
notes_seq: 2
notes:
  - id: n1
    kind: coordination
    text: merge sequence pos 1/4
    created_at: "%[1]s"
    updated_at: "%[1]s"
    resolved: false
    resolved_at: null
  - id: n2
    kind: correction
    text: fixed earlier conclusion
    created_at: "%[1]s"
    updated_at: "%[1]s"
    resolved: true
    resolved_at: "%[1]s"
`, updated)

	// bodyNotes parses stdout (the `# ` header lines are valid YAML comments)
	// and returns the ids present in the body's notes list.
	bodyNotes := func(t *testing.T, out string) []string {
		t.Helper()
		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(out), &data); err != nil {
			t.Fatalf("stdout body must parse as YAML under the header comments: %v\n%s", err, out)
		}
		notes, err := readNotes(data)
		if err != nil {
			t.Fatalf("decode notes: %v", err)
		}
		ids := []string{}
		for _, n := range notes {
			ids = append(ids, n.ID)
		}
		return ids
	}

	t.Run("human default: header + resolved excluded", func(t *testing.T) {
		path := withOperatorState(t, seed)
		out, _, err := runNoteCmd(t, operatorStateCmd())
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		if !strings.HasPrefix(out, "# OPEN NOTES (1)\n# n1 coordination 2d merge sequence pos 1/4\n") {
			t.Errorf("missing comment-prefixed OPEN NOTES header:\n%s", out)
		}
		if ids := bodyNotes(t, out); len(ids) != 1 || ids[0] != "n1" {
			t.Errorf("default body notes = %v, want [n1] (resolved excluded)", ids)
		}
		if data := readStateFile(t, path); len(readNotesList(t, path)) != 2 || data["notes_seq"] != 2 {
			t.Error("a read must never rewrite the file on disk")
		}
	})

	t.Run("human --all: resolved included", func(t *testing.T) {
		withOperatorState(t, seed)
		out, _, err := runNoteCmd(t, operatorStateCmd(), "--all")
		if err != nil {
			t.Fatalf("state --all: %v", err)
		}
		if !strings.HasPrefix(out, "# OPEN NOTES (1)\n") {
			t.Errorf("header is open-only even under --all:\n%s", out)
		}
		if ids := bodyNotes(t, out); len(ids) != 2 {
			t.Errorf("--all body notes = %v, want both", ids)
		}
	})

	t.Run("json: no header, resolved excluded", func(t *testing.T) {
		withOperatorState(t, seed)
		out, _, err := runNoteCmd(t, operatorStateCmd(), "--json")
		if err != nil {
			t.Fatalf("state --json: %v", err)
		}
		if strings.Contains(out, "#") || strings.Contains(out, "OPEN NOTES") {
			t.Errorf("--json output must be header-free:\n%s", out)
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(out), &data); err != nil {
			t.Fatalf("--json output must parse as JSON: %v\n%s", err, out)
		}
		notes, err := readNotes(data)
		if err != nil {
			t.Fatalf("decode notes: %v", err)
		}
		if len(notes) != 1 || notes[0].ID != "n1" {
			t.Errorf("--json notes = %+v, want [n1] (resolved excluded)", notes)
		}
	})

	t.Run("json --all: resolved included", func(t *testing.T) {
		withOperatorState(t, seed)
		out, _, err := runNoteCmd(t, operatorStateCmd(), "--json", "--all")
		if err != nil {
			t.Fatalf("state --json --all: %v", err)
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(out), &data); err != nil {
			t.Fatalf("--json --all output must parse as JSON: %v\n%s", err, out)
		}
		notes, err := readNotes(data)
		if err != nil {
			t.Fatalf("decode notes: %v", err)
		}
		if len(notes) != 2 {
			t.Errorf("--json --all notes = %+v, want both", notes)
		}
	})

	t.Run("no open notes: header omitted", func(t *testing.T) {
		withOperatorState(t, `notes:
  - id: n1
    kind: correction
    text: done
    created_at: "2026-01-01T00:00:00Z"
    updated_at: "2026-01-01T00:00:00Z"
    resolved: true
    resolved_at: "2026-01-01T00:00:00Z"
`)
		out, _, err := runNoteCmd(t, operatorStateCmd())
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		if strings.Contains(out, "OPEN NOTES") {
			t.Errorf("header must be omitted when nothing is open:\n%s", out)
		}
		if ids := bodyNotes(t, out); len(ids) != 0 {
			t.Errorf("resolved-only default body notes = %v, want none", ids)
		}
	})

	t.Run("no notes: byte-identical raw output", func(t *testing.T) {
		raw := "monitored: {}\ntick_count: 47\n"
		withOperatorState(t, raw)
		out, _, err := runNoteCmd(t, operatorStateCmd())
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		if out != raw {
			t.Errorf("a note-less file must print byte-identically (raw verbatim, no header):\ngot  %q\nwant %q", out, raw)
		}
	})

	t.Run("open-only notes: raw body preserved under the header", func(t *testing.T) {
		raw := fmt.Sprintf(`notes:
  - id: n1
    kind: phase_plan
    text: open only
    created_at: "%[1]s"
    updated_at: "%[1]s"
    resolved: false
    resolved_at: null
`, updated)
		withOperatorState(t, raw)
		out, _, err := runNoteCmd(t, operatorStateCmd())
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		if !strings.HasPrefix(out, "# OPEN NOTES (1)\n") || !strings.HasSuffix(out, raw) {
			t.Errorf("nothing filtered → raw body must print verbatim under the header:\n%s", out)
		}
	})
}

func TestOperatorState_SkeletonHasNotes(t *testing.T) {
	path := withOperatorState(t, "")

	out, _, err := runNoteCmd(t, operatorStateCmd())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if strings.Contains(out, "OPEN NOTES") {
		t.Errorf("skeleton output must have no header:\n%s", out)
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("skeleton output must parse: %v\n%s", err, out)
	}
	notes, ok := data["notes"].([]interface{})
	if !ok || len(notes) != 0 {
		t.Errorf("skeleton notes = %v, want []", data["notes"])
	}
	// The skeleton is persisted, not just printed.
	if _, ok := readStateFile(t, path)["notes"].([]interface{}); !ok {
		t.Error("persisted skeleton missing notes: []")
	}
}
