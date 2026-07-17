package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsArchivable_HydrateDone(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  intake: done
  apply: done
  review: done
  hydrate: done
`), 0o644)

	if !isArchivable(statusPath) {
		t.Error("expected archivable for hydrate: done")
	}
}

func TestIsArchivable_HydrateSkipped(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  hydrate: skipped
`), 0o644)

	if !isArchivable(statusPath) {
		t.Error("expected archivable for hydrate: skipped")
	}
}

func TestIsArchivable_HydratePending(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  hydrate: pending
`), 0o644)

	if isArchivable(statusPath) {
		t.Error("expected not archivable for hydrate: pending")
	}
}

func TestIsArchivable_MissingFile(t *testing.T) {
	if isArchivable("/nonexistent/.status.yaml") {
		t.Error("expected not archivable for missing file")
	}
}

func TestIsArchivable_HydrateKeyOutsideProgressBlock(t *testing.T) {
	// The deleted regex matched a `hydrate:` key at any indentation anywhere
	// in the document. statusfile semantics only honor progress.hydrate.
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  hydrate: pending
stage_metrics:
  hydrate: done
`), 0o644)

	if isArchivable(statusPath) {
		t.Error("hydrate key outside progress: must not make a change archivable")
	}
}

func TestIsArchivable_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte("not: [valid: yaml"), 0o644)

	if isArchivable(statusPath) {
		t.Error("expected not archivable for unparseable .status.yaml")
	}
}

func TestAllArchivableNames(t *testing.T) {
	dir := t.TempDir()

	// Archivable change
	changeDir := filepath.Join(dir, "260401-ab12-done-change")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("progress:\n  hydrate: done\n"), 0o644)

	// Non-archivable change
	pendingDir := filepath.Join(dir, "260401-cd34-pending-change")
	os.MkdirAll(pendingDir, 0o755)
	os.WriteFile(filepath.Join(pendingDir, ".status.yaml"), []byte("progress:\n  hydrate: pending\n"), 0o644)

	// Archive directory (should be excluded)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	names := allArchivableNames(dir)
	if len(names) != 1 {
		t.Fatalf("expected 1 archivable, got %d", len(names))
	}
	if names[0] != "260401-ab12-done-change" {
		t.Errorf("name = %q, want %q", names[0], "260401-ab12-done-change")
	}
}

func TestAllArchivableNames_NoEligible(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "260401-ab12-pending")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("progress:\n  hydrate: active\n"), 0o644)

	names := allArchivableNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 archivable, got %d", len(names))
	}
}

func TestListArchivable(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "260401-ab12-done")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("progress:\n  hydrate: done\n"), 0o644)

	var buf bytes.Buffer
	listArchivable(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("260401-ab12-done")) {
		t.Error("expected change name in output")
	}
}

func TestListArchivable_None(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	listArchivable(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("(none)")) {
		t.Error("expected (none) in output")
	}
}

func TestArchiveLoop(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	changesDir := filepath.Join(fabRoot, "changes")

	// Helper to create an archivable change folder (intake + hydrate: done).
	makeChange := func(folder, title string) {
		cd := filepath.Join(changesDir, folder)
		os.MkdirAll(cd, 0o755)
		os.WriteFile(filepath.Join(cd, ".status.yaml"), []byte("progress:\n  hydrate: done\n"), 0o644)
		os.WriteFile(filepath.Join(cd, "intake.md"), []byte("# Intake: "+title+"\n"), 0o644)
	}

	makeChange("260401-aa11-first-change", "First change")
	makeChange("260401-bb22-second-change", "Second change")

	// A third change that is already archived (destination exists) → skipped.
	thirdFolder := "260401-cc33-third-change"
	makeChange(thirdFolder, "Third change")
	archivedDest := filepath.Join(changesDir, "archive", "2026", "04")
	os.MkdirAll(filepath.Join(archivedDest, thirdFolder), 0o755)

	var out, errOut bytes.Buffer
	resolved := []string{
		"260401-aa11-first-change",
		"260401-bb22-second-change",
		thirdFolder,
	}
	archived, skipped, failed := archiveLoop(&out, &out, &errOut, fabRoot, resolved)

	if archived != 2 {
		t.Errorf("archived = %d, want 2", archived)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (already-archived third change)", skipped)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}

	// Both archivable changes moved under archive/2026/04/.
	for _, f := range []string{"260401-aa11-first-change", "260401-bb22-second-change"} {
		moved := filepath.Join(archivedDest, f)
		if _, err := os.Stat(moved); os.IsNotExist(err) {
			t.Errorf("%s not found in archive/2026/04/", f)
		}
		orig := filepath.Join(changesDir, f)
		if _, err := os.Stat(orig); !os.IsNotExist(err) {
			t.Errorf("%s should be removed from changes/ after archive", f)
		}
	}

	// Footer reflects the counts.
	if !strings.Contains(out.String(), "Archived 2, skipped 1, failed 0.") {
		t.Errorf("footer missing or wrong, got:\n%s", out.String())
	}
	// Already-archived third change is reported as skipped, not failed.
	if !strings.Contains(out.String(), thirdFolder+" — already archived, skipping") {
		t.Errorf("expected third change reported as skipped, got:\n%s", out.String())
	}
}

func TestBatchArchiveCmd_Structure(t *testing.T) {
	cmd := batchArchiveCmd()
	if cmd.Use != "archive [change...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "archive [change...]")
	}

	if cmd.Flags().Lookup("yes") == nil {
		t.Error("missing --yes flag")
	}
	if cmd.Flags().ShorthandLookup("y") == nil {
		t.Error("missing -y shorthand for --yes")
	}
	if cmd.Flags().Lookup("dry-run") == nil {
		t.Error("missing --dry-run flag")
	}
	if cmd.Flags().ShorthandLookup("d") != nil {
		t.Error("--dry-run must not have a short alias")
	}
	if cmd.Flags().Lookup("all") != nil {
		t.Error("--all flag must be removed")
	}
	if cmd.Flags().Lookup("list") != nil {
		t.Error("--list flag must be removed")
	}
	if cmd.Flags().Lookup("quiet") == nil {
		t.Error("missing --quiet flag")
	}
	if cmd.Flags().ShorthandLookup("q") == nil {
		t.Error("missing -q shorthand for --quiet")
	}
}

// forceTTY overrides the package-level isStdinTTY seam for the duration of a
// test so the prompt / non-TTY-guard branches are exercised deterministically.
func forceTTY(t *testing.T, tty bool) {
	t.Helper()
	orig := isStdinTTY
	isStdinTTY = func(io.Reader) bool { return tty }
	t.Cleanup(func() { isStdinTTY = orig })
}

// makeArchivable writes an archivable (hydrate: done) change folder under
// {root}/fab/changes/{folder}.
func makeArchivable(t *testing.T, root, folder string) string {
	t.Helper()
	changeDir := filepath.Join(root, "fab", "changes", folder)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("name: "+folder+"\nprogress:\n  hydrate: done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte("# Intake: "+folder+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return changeDir
}

// --- Empty-set and archived-name exit semantics (k4ge / 753q) ---

func TestRunBatchArchive_EmptyYesSetIsBenignNoOp(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	// --yes over an empty set must be a benign no-op (exit 0) — the empty-set
	// check runs before the prompt/guard.
	if err := runBatchArchive(cmd, nil, true, false, false); err != nil {
		t.Fatalf("empty --yes set must be a benign no-op (exit 0), got: %v", err)
	}
	if !strings.Contains(out.String(), "No archivable changes found.") {
		t.Errorf("missing notice, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 0, skipped 0, failed 0.") {
		t.Errorf("missing zero footer, got:\n%s", out.String())
	}
}

func TestRunBatchArchive_EmptyBareSetIsBenignNoOpBeforeGuard(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{})
	// Even with a non-TTY stdin and no --yes, the empty-set no-op must win
	// over the non-TTY guard (the F49 check happens first).
	forceTTY(t, false)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(""))

	if err := runBatchArchive(cmd, nil, false, false, false); err != nil {
		t.Fatalf("empty bare set must be a benign no-op (exit 0) before the guard, got: %v", err)
	}
	if !strings.Contains(out.String(), "No archivable changes found.") {
		t.Errorf("missing notice, got:\n%s", out.String())
	}
}

func TestRunBatchArchive_ArchivedNameSoftSkips(t *testing.T) {
	root := t.TempDir()
	folder := "260310-abcd-my-change"
	archivedDir := filepath.Join(root, "fab", "changes", "archive", "2026", "03", folder)
	os.MkdirAll(archivedDir, 0o755)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	// Explicit-args path: an archived name soft-skips with no prompt.
	if err := runBatchArchive(cmd, []string{folder}, false, false, false); err != nil {
		t.Fatalf("archived name must soft-skip (exit 0), got: %v", err)
	}
	if !strings.Contains(out.String(), folder+" — already archived, skipping") {
		t.Errorf("expected soft-skip line, got:\n%s\nstderr:\n%s", out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Archived 0, skipped 1, failed 0.") {
		t.Errorf("footer should count it as skipped, got:\n%s", out.String())
	}
}

// TestRunBatchArchive_DryRunListsOnly: --dry-run lists the archivable set,
// prompts nothing, and archives nothing (replaces the old --list).
func TestRunBatchArchive_DryRunListsOnly(t *testing.T) {
	root := t.TempDir()
	folder := "260310-abcd-my-change"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, nil, false, true, false); err != nil {
		t.Fatalf("--dry-run must list, got error: %v", err)
	}
	if !strings.Contains(out.String(), "Archivable changes") {
		t.Errorf("--dry-run must print the archivable list, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), folder) {
		t.Errorf("list must include the archivable change, got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("--dry-run must not prompt, got:\n%s", out.String())
	}
	// Nothing archived: the change folder is still in place.
	if _, err := os.Stat(changeDir); err != nil {
		t.Errorf("--dry-run must not archive anything: %v", err)
	}
}

// TestRunBatchArchive_DryRunYesMutuallyExclusive: --dry-run --yes errors.
func TestRunBatchArchive_DryRunYesMutuallyExclusive(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	err := runBatchArchive(cmd, nil, true, true, false)
	if err == nil {
		t.Fatal("expected error for --dry-run --yes (mutually exclusive)")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRunBatchArchive_YesArchivesAllNoPrompt: --yes archives every archivable
// change with no prompt (replaces the old --all).
func TestRunBatchArchive_YesArchivesAllNoPrompt(t *testing.T) {
	root := t.TempDir()
	fabRoot := filepath.Join(root, "fab")
	f1 := makeArchivable(t, root, "260401-aa11-first")
	f2 := makeArchivable(t, root, "260401-bb22-second")
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, nil, true, false, false); err != nil {
		t.Fatalf("--yes must archive all, got error: %v\nstderr:\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("--yes must not prompt, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 2, skipped 0, failed 0.") {
		t.Errorf("expected both archived, got:\n%s", out.String())
	}
	// Both source folders gone (moved into archive/).
	for _, d := range []string{f1, f2} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("%s should be removed from changes/ after --yes archive", d)
		}
	}
	_ = fabRoot
}

// TestRunBatchArchive_BarePromptYesArchives: bare + TTY, answering "y" archives
// all archivable changes.
func TestRunBatchArchive_BarePromptYesArchives(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, true)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("y\n"))

	if err := runBatchArchive(cmd, nil, false, false, false); err != nil {
		t.Fatalf("bare prompt + 'y' must archive, got error: %v\nstderr:\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Archive these 1? [y/N]") {
		t.Errorf("expected prompt, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 1, skipped 0, failed 0.") {
		t.Errorf("expected the change archived, got:\n%s", out.String())
	}
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Errorf("%s should be removed after a 'y' confirm", changeDir)
	}
}

// TestRunBatchArchive_BarePromptYesWordArchives: the full word "yes" also
// confirms (case-insensitive).
func TestRunBatchArchive_BarePromptYesWordArchives(t *testing.T) {
	root := t.TempDir()
	makeArchivable(t, root, "260401-aa11-first")
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, true)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("YES\n"))

	if err := runBatchArchive(cmd, nil, false, false, false); err != nil {
		t.Fatalf("bare prompt + 'YES' must archive, got error: %v", err)
	}
	if !strings.Contains(out.String(), "Archived 1, skipped 0, failed 0.") {
		t.Errorf("expected the change archived, got:\n%s", out.String())
	}
}

// TestRunBatchArchive_BarePromptEnterAborts: bare + TTY, a bare Enter (default
// No) aborts — exit 0, nothing archived.
func TestRunBatchArchive_BarePromptEnterAborts(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, true)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("\n")) // bare Enter

	if err := runBatchArchive(cmd, nil, false, false, false); err != nil {
		t.Fatalf("bare prompt + Enter must abort with exit 0, got error: %v", err)
	}
	if strings.Contains(out.String(), "Archived 1") {
		t.Errorf("Enter must not archive anything, got:\n%s", out.String())
	}
	if _, err := os.Stat(changeDir); err != nil {
		t.Errorf("Enter must leave the change in place: %v", err)
	}
}

// TestRunBatchArchive_BarePromptNoAborts: an explicit "n" aborts too.
func TestRunBatchArchive_BarePromptNoAborts(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, true)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("n\n"))

	if err := runBatchArchive(cmd, nil, false, false, false); err != nil {
		t.Fatalf("'n' must abort with exit 0, got error: %v", err)
	}
	if _, err := os.Stat(changeDir); err != nil {
		t.Errorf("'n' must leave the change in place: %v", err)
	}
}

// TestRunBatchArchive_NonTTYWithoutYesRefuses: non-TTY stdin + no --yes must
// refuse with guidance and a non-zero exit (it must NOT prompt or hang).
func TestRunBatchArchive_NonTTYWithoutYesRefuses(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, false)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("")) // non-interactive, would EOF

	err := runBatchArchive(cmd, nil, false, false, false)
	if err == nil {
		t.Fatal("non-TTY without --yes must refuse with a non-zero exit")
	}
	// The refusal guidance is carried on the returned error (which main()
	// prints once, prefixed with "ERROR:"), not emitted to stderr by the
	// handler — so the one failure path produces a single ERROR: line.
	if !strings.Contains(err.Error(), "refusing to prompt for confirmation on a non-interactive stdin") {
		t.Errorf("expected refusal guidance in returned error, got:\n%v", err)
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("guidance must point at --yes, got:\n%v", err)
	}
	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("non-TTY path must NOT prompt, got:\n%s", out.String())
	}
	if _, err := os.Stat(changeDir); err != nil {
		t.Errorf("refusal must archive nothing: %v", err)
	}
}

// TestRunBatchArchive_ExplicitArgsNoPrompt: explicit-args archiving never
// prompts and never consults the TTY guard, even on a non-TTY stdin.
func TestRunBatchArchive_ExplicitArgsNoPrompt(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, false) // explicit args must ignore the guard

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, []string{folder}, false, false, false); err != nil {
		t.Fatalf("explicit args must archive without prompting, got error: %v\nstderr:\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "[y/N]") {
		t.Errorf("explicit args must not prompt, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 1, skipped 0, failed 0.") {
		t.Errorf("expected the named change archived, got:\n%s", out.String())
	}
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Errorf("%s should be removed after explicit-args archive", changeDir)
	}
}

// TestRunBatchArchive_AmbiguousNameSurfaces: jznd (d) end-to-end guard. An
// ambiguous override (matches 2+ live changes) must surface a distinct
// ambiguity warning on stderr and NOT be misreported as already-archived
// (the ErrNotFound soft-skip path) — the resolve sentinels let runBatchArchive
// branch ErrAmbiguous → warn vs ErrNotFound → maybe-soft-skip. Unit-level
// classification is covered in internal/resolve/resolve_test.go; this asserts
// the call-site behavior through runBatchArchive.
func TestRunBatchArchive_AmbiguousNameSurfaces(t *testing.T) {
	root := t.TempDir()
	changesDir := filepath.Join(root, "fab", "changes")
	// Two live changes sharing the "report" substring → an "report" override
	// is ambiguous.
	for _, folder := range []string{"260401-aa11-report-alpha", "260401-bb22-report-beta"} {
		cd := filepath.Join(changesDir, folder)
		os.MkdirAll(cd, 0o755)
		os.WriteFile(filepath.Join(cd, ".status.yaml"), []byte("progress:\n  hydrate: done\n"), 0o644)
	}
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	// The only target is ambiguous → nothing resolves → RunE returns the
	// no-valid-changes error (exit 1), but the ambiguity must be reported
	// distinctly, NOT as "could not resolve" or as an already-archived skip.
	err := runBatchArchive(cmd, []string{"report"}, false, false, false)
	if err == nil {
		t.Fatal("expected error: the sole target is ambiguous, nothing resolves")
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "Multiple changes match") {
		t.Errorf("ambiguous name must surface the ambiguity warning, got stderr:\n%s", stderr)
	}
	if strings.Contains(stderr, "could not resolve") {
		t.Errorf("ambiguous name must NOT be misreported as a plain resolve failure, got stderr:\n%s", stderr)
	}
	if strings.Contains(out.String(), "already archived, skipping") {
		t.Errorf("ambiguous name must NOT be soft-skipped as already-archived, got stdout:\n%s", out.String())
	}
}

// TestRunBatchArchive_NoValidChangesReturnsError: the explicit-targets
// nothing-resolves path returns an error through RunE (previously
// os.Exit(1)) — stderr becomes `ERROR: No valid changes to archive.`.
func TestRunBatchArchive_NoValidChangesReturnsError(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	err := runBatchArchive(cmd, []string{"zzzz-nope"}, false, false, false)
	if err == nil {
		t.Fatal("expected error when no target resolves")
	}
	if !strings.Contains(err.Error(), "No valid changes to archive.") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- --quiet (o5f9): suppress per-change progress; retain footer, data, stderr ---

// TestRunBatchArchive_QuietYesFooterOnly: --quiet --yes over a non-empty set
// prints ONLY the summary footer — no "Archiving N changes..." preamble and no
// per-change "  {name} — archived" lines.
func TestRunBatchArchive_QuietYesFooterOnly(t *testing.T) {
	root := t.TempDir()
	makeArchivable(t, root, "260401-aa11-first")
	makeArchivable(t, root, "260401-bb22-second")
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, nil, true, false, true); err != nil {
		t.Fatalf("--quiet --yes must archive all, got error: %v\nstderr:\n%s", err, errOut.String())
	}
	got := out.String()
	if strings.Contains(got, "Archiving 2 changes...") {
		t.Errorf("--quiet must suppress the 'Archiving N changes...' preamble, got:\n%s", got)
	}
	if strings.Contains(got, "— archived") {
		t.Errorf("--quiet must suppress per-change lines, got:\n%s", got)
	}
	if !strings.Contains(got, "Archived 2, skipped 0, failed 0.") {
		t.Errorf("--quiet must retain the summary footer, got:\n%s", got)
	}
	// The footer, once its leading blank line is trimmed, is the ENTIRE stdout.
	if strings.TrimSpace(got) != "Archived 2, skipped 0, failed 0." {
		t.Errorf("--quiet --yes stdout must be footer-only, got:\n%q", got)
	}
}

// TestRunBatchArchive_QuietExplicitArgsFooterOnly: --quiet with explicit args
// suppresses the same per-change lines and keeps the footer.
func TestRunBatchArchive_QuietExplicitArgsFooterOnly(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, []string{folder}, false, false, true); err != nil {
		t.Fatalf("--quiet explicit args must archive, got error: %v\nstderr:\n%s", err, errOut.String())
	}
	got := out.String()
	if strings.Contains(got, "— archived\n") && strings.Contains(got, folder+" — archived") {
		t.Errorf("--quiet must suppress the per-change line, got:\n%s", got)
	}
	if strings.TrimSpace(got) != "Archived 1, skipped 0, failed 0." {
		t.Errorf("--quiet explicit-args stdout must be footer-only, got:\n%q", got)
	}
}

// TestRunBatchArchive_QuietEmptySetRetainsNoOp: --quiet over an empty set still
// prints the empty-set no-op output (it is the run's outcome, not per-change
// progress) and exits 0 (F49 preserved).
func TestRunBatchArchive_QuietEmptySetRetainsNoOp(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchArchive(cmd, nil, true, false, true); err != nil {
		t.Fatalf("--quiet empty --yes set must be a benign no-op (exit 0), got: %v", err)
	}
	if !strings.Contains(out.String(), "No archivable changes found.") {
		t.Errorf("--quiet must retain the empty-set no-op notice, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 0, skipped 0, failed 0.") {
		t.Errorf("--quiet must retain the zero footer, got:\n%s", out.String())
	}
}

// TestRunBatchArchive_QuietRetainsStderr: --quiet must never touch stderr — an
// unresolvable name still surfaces its warning while stdout carries only the
// footer.
func TestRunBatchArchive_QuietRetainsStderr(t *testing.T) {
	root := t.TempDir()
	makeArchivable(t, root, "260401-aa11-first")
	chdirTestEnv(t, root, map[string]string{})

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	// Mix a resolvable change with an unresolvable one: the good one archives,
	// the bad one warns on stderr — stderr must be untouched by --quiet.
	if err := runBatchArchive(cmd, []string{"260401-aa11-first", "zzzz-nope"}, false, false, true); err != nil {
		t.Fatalf("warn-and-skip must not error, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "could not resolve 'zzzz-nope'") {
		t.Errorf("--quiet must retain stderr warnings, got stderr:\n%s", errOut.String())
	}
	if strings.Contains(out.String(), "— archived") {
		t.Errorf("--quiet must still suppress the per-change stdout line, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Archived 1, skipped 0, failed 0.") {
		t.Errorf("footer must still print, got:\n%s", out.String())
	}
}

// TestRunBatchArchive_QuietDoesNotImplyYes: --quiet is orthogonal to consent —
// a bare --quiet on a (forced) TTY still lists the set and prompts [y/N]; it
// does NOT archive without a confirming answer.
func TestRunBatchArchive_QuietDoesNotImplyYes(t *testing.T) {
	root := t.TempDir()
	folder := "260401-aa11-first"
	changeDir := makeArchivable(t, root, folder)
	chdirTestEnv(t, root, map[string]string{})
	forceTTY(t, true)

	cmd := batchArchiveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader("\n")) // bare Enter = default No

	if err := runBatchArchive(cmd, nil, false, false, true); err != nil {
		t.Fatalf("bare --quiet + Enter must abort with exit 0, got error: %v", err)
	}
	// The consent listing + prompt are NOT progress — they survive --quiet.
	if !strings.Contains(out.String(), "Archive these 1? [y/N]") {
		t.Errorf("--quiet must NOT suppress the consent prompt (not implied --yes), got:\n%s", out.String())
	}
	// Default No aborted — nothing archived.
	if _, err := os.Stat(changeDir); err != nil {
		t.Errorf("--quiet must not imply --yes: the change must remain in place, got: %v", err)
	}
}
