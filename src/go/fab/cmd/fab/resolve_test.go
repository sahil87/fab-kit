package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

const resolveTestFolder = "260401-ab12-add-feature"

// resolveTestRepo creates a temp repo with one change folder and chdirs into
// it (env/cwd restored on cleanup).
func resolveTestRepo(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes", resolveTestFolder), 0o755)
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})
}

// runResolveCmd executes a fresh resolveCmd with the given args, returning
// (stdout, error).
func runResolveCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := resolveCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestResolveOutputFlagsMutuallyExclusive: conflicting output-mode flags fail
// loudly instead of being silently resolved by a priority chain (F25).
func TestResolveOutputFlagsMutuallyExclusive(t *testing.T) {
	resolveTestRepo(t)

	_, err := runResolveCmd(t, "--status", "--folder", "ab12")
	if err == nil {
		t.Fatal("expected mutual-exclusion error for --status --folder")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("expected cobra flags-group error, got: %v", err)
	}

	// --id is part of the group too (it was previously a dead flag outside
	// the chain — `--id --pane` silently printed the pane).
	_, err = runResolveCmd(t, "--id", "--pane", "ab12")
	if err == nil {
		t.Fatal("expected mutual-exclusion error for --id --pane")
	}
}

// TestResolveIDFlagWired: --id is a real explicit-default flag — it is read
// and selects the ID output mode (previously registered but never read).
func TestResolveIDFlagWired(t *testing.T) {
	resolveTestRepo(t)

	explicit, err := runResolveCmd(t, "--id", "ab12")
	if err != nil {
		t.Fatalf("--id resolve failed: %v", err)
	}
	if explicit != "ab12\n" {
		t.Errorf("--id output = %q, want %q", explicit, "ab12\n")
	}

	implicit, err := runResolveCmd(t, "ab12")
	if err != nil {
		t.Fatalf("default resolve failed: %v", err)
	}
	if implicit != explicit {
		t.Errorf("default output %q differs from explicit --id output %q", implicit, explicit)
	}
}

// TestChangeResolveSharesImplementation: `fab change resolve` is a thin
// wrapper over the same runResolve implementation as `fab resolve --folder`
// (F27) — identical stdout, identical error strings.
func TestChangeResolveSharesImplementation(t *testing.T) {
	resolveTestRepo(t)

	folderOut, err := runResolveCmd(t, "--folder", "ab12")
	if err != nil {
		t.Fatalf("resolve --folder failed: %v", err)
	}

	cmd := changeResolveCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"ab12"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("change resolve failed: %v", err)
	}

	if out.String() != folderOut {
		t.Errorf("change resolve output %q != resolve --folder output %q", out.String(), folderOut)
	}
	if out.String() != resolveTestFolder+"\n" {
		t.Errorf("change resolve output = %q, want %q", out.String(), resolveTestFolder+"\n")
	}

	// Error strings flow through the same shared path.
	_, errA := runResolveCmd(t, "--folder", "zzzz")
	cmdB := changeResolveCmd()
	cmdB.SetOut(&bytes.Buffer{})
	cmdB.SetArgs([]string{"zzzz"})
	errB := cmdB.Execute()
	if errA == nil || errB == nil {
		t.Fatal("expected resolution errors for unknown change")
	}
	if errA.Error() != errB.Error() {
		t.Errorf("error strings drifted:\n  resolve --folder: %q\n  change resolve:   %q", errA.Error(), errB.Error())
	}
}

// TestResolvePaneNoTmuxReturnsError: the $TMUX guard returns an error through
// RunE (previously os.Exit(1)) when --server is not given (F30/R13).
func TestResolvePaneNoTmuxReturnsError(t *testing.T) {
	resolveTestRepo(t)

	_, err := runResolveCmd(t, "--pane", "ab12")
	if err == nil {
		t.Fatal("expected error for --pane outside tmux")
	}
	if !strings.Contains(err.Error(), "not inside a tmux session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestResolveServerFlagRegistered: --server/-L is plumbed through fab resolve
// for the pane mode (F27), matching the pane family's flag shape.
func TestResolveServerFlagRegistered(t *testing.T) {
	cmd := resolveCmd()
	f := cmd.Flags().Lookup("server")
	if f == nil {
		t.Fatal("missing --server flag")
	}
	if f.Shorthand != "L" {
		t.Errorf("--server shorthand = %q, want %q", f.Shorthand, "L")
	}
}

// resolveTestRepoWith creates a temp repo with the given change folders — each
// WITH a .status.yaml, so bare resolution counts them as candidates — and
// chdirs into it (no .fab-status.yaml symlink: no change is active).
func resolveTestRepoWith(t *testing.T, folders ...string) {
	t.Helper()
	root := t.TempDir()
	for _, f := range folders {
		dir := filepath.Join(root, "fab", "changes", f)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, ".status.yaml"), []byte("stage: intake\n"), 0o644)
	}
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})
}

// TestResolveOrNoneNotFound: --or-none maps ErrNotFound to "(none)" + success
// for BOTH bare resolution and an explicit override argument (dow0 R2/R3).
func TestResolveOrNoneNotFound(t *testing.T) {
	// resolveTestRepo's folder has no .status.yaml, so bare resolution finds
	// zero candidates (not-found) while override resolution can still match.
	resolveTestRepo(t)

	for _, args := range [][]string{
		{"--or-none"},         // bare: no active change
		{"--or-none", "zzzz"}, // explicit override: no match
	} {
		out, err := runResolveCmd(t, args...)
		if err != nil {
			t.Fatalf("resolve %v failed: %v", args, err)
		}
		if out != "(none)\n" {
			t.Errorf("resolve %v output = %q, want %q", args, out, "(none)\n")
		}
	}
}

// TestResolveOrNoneBareAmbiguous: bare ambiguous ("multiple changes exist,
// none active") IS the no-active-change state → "(none)" + success (dow0 R2).
func TestResolveOrNoneBareAmbiguous(t *testing.T) {
	resolveTestRepoWith(t, "260401-ab12-add-feature", "260402-cd34-add-widget")

	out, err := runResolveCmd(t, "--or-none")
	if err != nil {
		t.Fatalf("bare ambiguous --or-none failed: %v", err)
	}
	if out != "(none)\n" {
		t.Errorf("output = %q, want %q", out, "(none)\n")
	}
}

// TestResolveOrNoneOverrideAmbiguousStillErrors: a named-but-multi-matching
// override is a real user error — never mapped, message unchanged (dow0 R2).
func TestResolveOrNoneOverrideAmbiguousStillErrors(t *testing.T) {
	resolveTestRepoWith(t, "260401-ab12-add-feature", "260402-cd34-add-widget")

	out, err := runResolveCmd(t, "--or-none", "add")
	if err == nil {
		t.Fatalf("expected ambiguous-override error, got output %q", out)
	}
	if !strings.Contains(err.Error(), "Multiple changes match") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestResolveOrNoneInfrastructureErrorStillErrors: a missing fab/ root is an
// infrastructure error, not a state sentinel — never mapped (dow0 R2).
func TestResolveOrNoneInfrastructureErrorStillErrors(t *testing.T) {
	root := t.TempDir()
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})
	if _, err := resolve.FabRoot(); err == nil {
		t.Skip("a fab/ directory exists above the temp dir; cannot simulate a missing root")
	}

	_, err := runResolveCmd(t, "--or-none")
	if err == nil {
		t.Fatal("expected fab/-root error despite --or-none")
	}
	if !strings.Contains(err.Error(), "fab/ directory not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestResolveOrNoneOutputModeComposition: --or-none composes with the output
// modes — "(none)" replaces the mode-specific output on the none path, and
// the flag is a no-op on the success path (dow0 R1/R3).
func TestResolveOrNoneOutputModeComposition(t *testing.T) {
	resolveTestRepo(t)

	for _, mode := range []string{"--folder", "--id"} {
		out, err := runResolveCmd(t, mode, "--or-none", "zzzz")
		if err != nil {
			t.Fatalf("%s --or-none failed: %v", mode, err)
		}
		if out != "(none)\n" {
			t.Errorf("%s --or-none output = %q, want %q", mode, out, "(none)\n")
		}
	}

	// Success path: the flag changes nothing.
	out, err := runResolveCmd(t, "--folder", "--or-none", "ab12")
	if err != nil {
		t.Fatalf("success-path --or-none failed: %v", err)
	}
	if out != resolveTestFolder+"\n" {
		t.Errorf("success output = %q, want %q", out, resolveTestFolder+"\n")
	}
}

// TestResolveWithoutOrNoneUnchanged: flag absent → absence stays an error on
// both the bare and override paths (regression pin for the opt-in contract).
func TestResolveWithoutOrNoneUnchanged(t *testing.T) {
	resolveTestRepo(t)

	_, err := runResolveCmd(t)
	if err == nil || !strings.Contains(err.Error(), "No active change") {
		t.Errorf("bare resolve error = %v, want a No-active-change error", err)
	}

	_, err = runResolveCmd(t, "zzzz")
	if err == nil || !strings.Contains(err.Error(), "No change matches") {
		t.Errorf("override resolve error = %v, want a No-change-matches error", err)
	}
}

// TestChangeResolveHasNoOrNoneFlag: the thin wrapper stays flag-free — the
// query flags (incl. --or-none) live on top-level fab resolve only (dow0 R4).
func TestChangeResolveHasNoOrNoneFlag(t *testing.T) {
	resolveTestRepo(t)

	cmd := changeResolveCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--or-none", "ab12"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("expected unknown-flag error, got: %v", err)
	}
}
