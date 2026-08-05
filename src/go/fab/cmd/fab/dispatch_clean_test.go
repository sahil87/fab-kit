package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

func runClean(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchCleanCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// makeDispatchDir creates .fab-dispatch/{id}/ with a marker file so removal is
// observable.
func makeDispatchDir(t *testing.T, repoRoot, id string) string {
	t.Helper()
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, filepath.Join(dir, "apply.yaml"), "pid: 1\n")
	return dir
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestDispatchClean_NamedChange(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := makeDispatchDir(t, repoRoot, id)

	if _, err := runClean(t, "abcd"); err != nil {
		t.Fatalf("clean <change>: %v", err)
	}
	if exists(dir) {
		t.Error("named change's dispatch dir should be removed")
	}
}

func TestDispatchClean_All(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	live := makeDispatchDir(t, repoRoot, id)
	other := makeDispatchDir(t, repoRoot, "zzzz")

	if _, err := runClean(t); err != nil {
		t.Fatalf("clean (all): %v", err)
	}
	if exists(live) || exists(other) {
		t.Error("clean (no arg) should remove all dispatch dirs")
	}
}

func TestDispatchClean_Orphans(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	live := makeDispatchDir(t, repoRoot, id)       // resolves to active change
	orphan := makeDispatchDir(t, repoRoot, "zzzz") // no such change

	if _, err := runClean(t, "--orphans"); err != nil {
		t.Fatalf("clean --orphans: %v", err)
	}
	if !exists(live) {
		t.Error("live change's dispatch dir must survive --orphans")
	}
	if exists(orphan) {
		t.Error("orphaned dispatch dir must be pruned by --orphans")
	}
}

// TestDispatchClean_PaneDispatchNeedsNoModeAwareness pins the documented
// pane-mode `clean` behavior: clean removes state dirs and never inspects a
// record's mode, so a pane dispatch's dir is removed exactly like a headless
// one's — and (as with a live headless process) the worker itself is NOT killed,
// which is why `kill` exists as the separate verb.
func TestDispatchClean_PaneDispatchNeedsNoModeAwareness(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	dir := seedPaneDispatch(t, repoRoot, id, "apply", "%42", "")

	if _, err := runClean(t, "abcd"); err != nil {
		t.Fatalf("clean over a pane dispatch: %v", err)
	}
	if exists(dir) {
		t.Error("a pane dispatch's state dir should be removed like any other")
	}
}

func TestDispatchClean_NoState(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	// No .fab-dispatch/ dir at all.
	out, err := runClean(t)
	if err != nil {
		t.Fatalf("clean with no state should not error: %v", err)
	}
	if out == "" {
		t.Error("expected an informational no-state message")
	}
}
