package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
