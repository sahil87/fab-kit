package main

// Cobra-execution tests for the low-coverage change-lifecycle RunE bodies
// (260612-tb6f, F46): changeArchiveCmd, changeArchiveListCmd, changeSwitchCmd,
// changeRestoreCmd, changeListCmd, changeRenameCmd, logReviewCmd, and
// listPendingItems. Follows the in-package execution pattern of
// memory_index_test.go (temp fab repo + os.Chdir + SetArgs + assertions on
// stdout and the filesystem) and asserts the EXACT stdout shapes skills
// parse: the archive command's structured YAML, the `already archived:`
// soft-skip line, and the `index: failed` print-then-error contract.

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const execTestStatusYAML = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: done
  apply: active
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

// setupChangeRepo builds a repo with one change (active via symlink) and
// chdirs into it so resolve.FabRoot() resolves.
func setupChangeRepo(t *testing.T) (repoRoot, folder string) {
	t.Helper()
	repoRoot = t.TempDir()
	folder = "260310-abcd-my-change"
	changeDir := filepath.Join(repoRoot, "fab", "changes", folder)
	mustMkdir(t, changeDir)
	mustWrite(t, filepath.Join(changeDir, ".status.yaml"), execTestStatusYAML)
	mustWrite(t, filepath.Join(changeDir, "intake.md"), "# Intake: My Change\n")
	if err := os.Symlink("fab/changes/"+folder+"/.status.yaml", filepath.Join(repoRoot, ".fab-status.yaml")); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	return repoRoot, folder
}

// execCapture executes a cobra command with args, capturing os.Stdout (the
// change-lifecycle RunE bodies print via fmt.Println, not cmd.OutOrStdout).
func execCapture(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	execErr := cmd.Execute()
	os.Stdout = orig
	w.Close()
	data, _ := io.ReadAll(r)
	return string(data), execErr
}

// --- fab change archive ---

func TestChangeArchiveCmd_EmitsStructuredYAML(t *testing.T) {
	_, folder := setupChangeRepo(t)

	out, err := execCapture(t, changeArchiveCmd(), "abcd")
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	// The structured YAML block skills parse — exact keys, one per line.
	for _, want := range []string{
		"name: " + folder,
		"move: moved",
		"index: created",
		"pointer: cleared",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("archive output missing %q:\n%s", want, out)
		}
	}
	for _, key := range []string{"action:", "backlog:"} {
		if !strings.Contains(out, key) {
			t.Errorf("archive output missing key %q:\n%s", key, out)
		}
	}
}

func TestChangeArchiveCmd_AlreadyArchivedSoftSkips(t *testing.T) {
	setupChangeRepo(t)

	if _, err := execCapture(t, changeArchiveCmd(), "abcd"); err != nil {
		t.Fatalf("first archive: %v", err)
	}
	out, err := execCapture(t, changeArchiveCmd(), "abcd")
	if err != nil {
		t.Fatalf("re-archiving a genuinely archived change must soft-skip (exit 0), got: %v", err)
	}
	if !strings.Contains(out, "already archived: abcd") {
		t.Errorf("expected 'already archived:' soft-skip line, got:\n%s", out)
	}
}

func TestChangeArchiveCmd_IndexFailedPrintsYAMLAndErrors(t *testing.T) {
	repoRoot, _ := setupChangeRepo(t)

	// A directory at the index path makes the index update fail after the
	// move succeeds — the YAML must still print (with index: failed) AND the
	// command must exit non-zero (hv7t's print-then-error contract).
	archiveDir := filepath.Join(repoRoot, "fab", "changes", "archive")
	mustMkdir(t, filepath.Join(archiveDir, "index.md"))

	out, err := execCapture(t, changeArchiveCmd(), "abcd")
	if err == nil {
		t.Fatal("expected non-zero exit when the index write fails")
	}
	if !strings.Contains(out, "index: failed") {
		t.Errorf("expected 'index: failed' in printed YAML, got:\n%s", out)
	}
	if !strings.Contains(out, "move: moved") {
		t.Errorf("YAML must still report the successful move, got:\n%s", out)
	}
}

// --- fab change restore ---

func TestChangeRestoreCmd_RoundTrip(t *testing.T) {
	repoRoot, folder := setupChangeRepo(t)

	if _, err := execCapture(t, changeArchiveCmd(), "abcd"); err != nil {
		t.Fatalf("archive: %v", err)
	}
	out, err := execCapture(t, changeRestoreCmd(), "abcd")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, want := range []string{"action: restore", "name: " + folder, "move: restored", "index: removed"} {
		if !strings.Contains(out, want) {
			t.Errorf("restore output missing %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "fab", "changes", folder, ".status.yaml")); err != nil {
		t.Errorf("restored change folder missing: %v", err)
	}
}

// --- fab change archive-list ---

func TestChangeArchiveListCmd_ListsArchivedFolders(t *testing.T) {
	_, folder := setupChangeRepo(t)

	// Empty archive: no output, no error.
	out, err := execCapture(t, changeArchiveListCmd())
	if err != nil {
		t.Fatalf("archive-list (empty): %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("empty archive should list nothing, got:\n%s", out)
	}

	if _, err := execCapture(t, changeArchiveCmd(), "abcd"); err != nil {
		t.Fatalf("archive: %v", err)
	}
	out, err = execCapture(t, changeArchiveListCmd())
	if err != nil {
		t.Fatalf("archive-list: %v", err)
	}
	if !strings.Contains(out, folder) {
		t.Errorf("archive-list missing %q:\n%s", folder, out)
	}
}

// --- fab change switch ---

func TestChangeSwitchCmd_SwitchUpdatesPointer(t *testing.T) {
	repoRoot, _ := setupChangeRepo(t)

	// A second change to switch to.
	other := "260311-wxyz-other-change"
	otherDir := filepath.Join(repoRoot, "fab", "changes", other)
	mustMkdir(t, otherDir)
	mustWrite(t, filepath.Join(otherDir, ".status.yaml"),
		strings.ReplaceAll(execTestStatusYAML, "260310-abcd-my-change", other))

	out, err := execCapture(t, changeSwitchCmd(), "wxyz")
	if err != nil {
		t.Fatalf("switch: %v", err)
	}
	if !strings.Contains(out, other) {
		t.Errorf("switch output should name the activated change:\n%s", out)
	}
	target, err := os.Readlink(filepath.Join(repoRoot, ".fab-status.yaml"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if want := "fab/changes/" + other + "/.status.yaml"; target != want {
		t.Errorf("pointer = %q, want %q", target, want)
	}
}

func TestChangeSwitchCmd_NoneDeactivates(t *testing.T) {
	repoRoot, _ := setupChangeRepo(t)

	out, err := execCapture(t, changeSwitchCmd(), "--none")
	if err != nil {
		t.Fatalf("switch --none: %v", err)
	}
	if !strings.Contains(out, "No active change.") {
		t.Errorf("expected deactivation message, got:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, ".fab-status.yaml")); !os.IsNotExist(err) {
		t.Error("pointer symlink should be removed by --none")
	}

	// Idempotent re-run reports the already-deactivated state, exit 0.
	out, err = execCapture(t, changeSwitchCmd(), "--none")
	if err != nil {
		t.Fatalf("second switch --none: %v", err)
	}
	if !strings.Contains(out, "already deactivated") {
		t.Errorf("expected already-deactivated message, got:\n%s", out)
	}
}

func TestChangeSwitchCmd_NoArgsErrors(t *testing.T) {
	setupChangeRepo(t)

	if _, err := execCapture(t, changeSwitchCmd()); err == nil {
		t.Fatal("switch without <name> or --none must error")
	}
}

// --- fab change list ---

func TestChangeListCmd_ListsActiveAndArchived(t *testing.T) {
	repoRoot, folder := setupChangeRepo(t)

	out, err := execCapture(t, changeListCmd())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// The exact row format skills parse: name:display_stage:display_state:score.
	if !strings.Contains(out, folder+":apply:active:0.0") {
		t.Errorf("list missing %q row:\n%s", folder+":apply:active:0.0", out)
	}

	// --archive scans fab/changes/archive top-level entries (the flat,
	// pre-yyyy/mm layout — still a supported archive shape).
	archived := "260309-zz99-old-change"
	archivedDir := filepath.Join(repoRoot, "fab", "changes", "archive", archived)
	mustMkdir(t, archivedDir)
	mustWrite(t, filepath.Join(archivedDir, ".status.yaml"),
		strings.ReplaceAll(execTestStatusYAML, "260310-abcd-my-change", archived))

	out, err = execCapture(t, changeListCmd(), "--archive")
	if err != nil {
		t.Fatalf("list --archive: %v", err)
	}
	if !strings.Contains(out, archived+":apply:active:0.0") {
		t.Errorf("list --archive missing %q row:\n%s", archived, out)
	}
	if strings.Contains(out, folder) {
		t.Errorf("list --archive must not include live changes:\n%s", out)
	}
}

// --- fab change rename ---

func TestChangeRenameCmd_RenamesSlugKeepsPrefix(t *testing.T) {
	repoRoot, folder := setupChangeRepo(t)

	out, err := execCapture(t, changeRenameCmd(), "--folder", folder, "--slug", "renamed-change")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	want := "260310-abcd-renamed-change"
	if !strings.Contains(out, want) {
		t.Errorf("rename should print the new folder name %q, got:\n%s", want, out)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "fab", "changes", want)); err != nil {
		t.Errorf("renamed folder missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "fab", "changes", folder)); !os.IsNotExist(err) {
		t.Error("old folder name should be gone after rename")
	}
}

func TestChangeRenameCmd_MissingFlagsError(t *testing.T) {
	setupChangeRepo(t)

	if _, err := execCapture(t, changeRenameCmd(), "--slug", "x"); err == nil {
		t.Fatal("rename without --folder must error")
	}
	if _, err := execCapture(t, changeRenameCmd(), "--folder", "260310-abcd-my-change"); err == nil {
		t.Fatal("rename without --slug must error")
	}
}

// --- fab log review ---

func TestLogReviewCmd_AppendsHistoryEntry(t *testing.T) {
	repoRoot, folder := setupChangeRepo(t)

	if _, err := execCapture(t, logReviewCmd(), "abcd", "passed"); err != nil {
		t.Fatalf("log review: %v", err)
	}
	if _, err := execCapture(t, logReviewCmd(), "abcd", "failed", "fix-code"); err != nil {
		t.Fatalf("log review with rework: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, "fab", "changes", folder, ".history.jsonl"))
	if err != nil {
		t.Fatalf("history file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("history has %d lines, want 2:\n%s", len(lines), data)
	}
	if !strings.Contains(lines[0], `"result":"passed"`) {
		t.Errorf("first entry missing passed result: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"result":"failed"`) || !strings.Contains(lines[1], `"rework":"fix-code"`) {
		t.Errorf("second entry missing failed result + rework: %s", lines[1])
	}
}

func TestLogReviewCmd_UnknownChangeErrors(t *testing.T) {
	setupChangeRepo(t)

	if _, err := execCapture(t, logReviewCmd(), "zzzz", "passed"); err == nil {
		t.Fatal("log review on an unknown change must error")
	}
}

// --- listPendingItems (fab batch new --list) ---

func TestListPendingItems_FormatsAndFilters(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("x", 90)
	backlog := "# Backlog\n\n" +
		"- [ ] [ab12] 2026-06-01: first pending item\n" +
		"- [x] [cd34] 2026-06-01: done item must not list\n" +
		"- [ ] [ef56] " + long + "\n"
	backlogPath := filepath.Join(dir, "backlog.md")
	if err := os.WriteFile(backlogPath, []byte(backlog), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := listPendingItems(&buf, backlogPath); err != nil {
		t.Fatalf("listPendingItems: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Pending backlog items:") {
		t.Errorf("missing header:\n%s", out)
	}
	if !strings.Contains(out, "[ab12]") {
		t.Errorf("missing pending item ab12:\n%s", out)
	}
	if strings.Contains(out, "cd34") {
		t.Errorf("checked item cd34 must not be listed:\n%s", out)
	}
	// Long descriptions are truncated to 80 chars.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "[ef56]") {
			desc := strings.TrimSpace(strings.SplitN(line, "]", 2)[1])
			if len(desc) > 80 {
				t.Errorf("ef56 description not truncated to 80 chars (got %d): %q", len(desc), desc)
			}
		}
	}
}

func TestListPendingItems_MissingBacklogErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := listPendingItems(&buf, filepath.Join(t.TempDir(), "absent.md")); err == nil {
		t.Fatal("missing backlog file must error")
	}
}
