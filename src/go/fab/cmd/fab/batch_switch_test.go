package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllChangeNames(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "260401-cd34-fix-bug"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	names := allChangeNames(dir)
	if len(names) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(names))
	}

	// Should not include "archive"
	for _, name := range names {
		if name == "archive" {
			t.Error("archive should be excluded")
		}
	}
}

func TestAllChangeNames_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := allChangeNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 changes, got %d", len(names))
	}
}

// (getBranchPrefix was retired in 260612-ye8r — branch_prefix now comes from
// the shared internal/config accessor, tested in internal/config.)

// TestRunBatchSwitch_NoTmuxReturnsError verifies the $TMUX guard returns an
// error through RunE (previously os.Exit(1)) — stderr becomes
// `ERROR: not inside a tmux session` via main.go's single formatter.
func TestRunBatchSwitch_NoTmuxReturnsError(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})

	cmd := batchSwitchCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	err := runBatchSwitch(cmd, []string{"whatever"}, false, false)
	if err == nil {
		t.Fatal("expected error outside tmux")
	}
	if !strings.Contains(err.Error(), "not inside a tmux session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRunBatchSwitch_UnresolvableWarnsAndSkips verifies in-process resolution
// (resolve.ToFolder, no `fab change resolve` subprocess): an unresolvable
// name warns with the resolver's SPECIFIC error and the loop continues
// (exit 0).
func TestRunBatchSwitch_UnresolvableWarnsAndSkips(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes", "260401-ab12-add-feature"), 0o755)
	chdirTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})

	cmd := batchSwitchCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchSwitch(cmd, []string{"zzzz-no-such-change"}, false, false); err != nil {
		t.Fatalf("warn-and-skip path must not error, got: %v", err)
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "could not resolve 'zzzz-no-such-change'") {
		t.Errorf("missing warn-and-skip warning, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "No change matches") {
		t.Errorf("warning must surface the resolver's specific error, got:\n%s", stderr)
	}
}

func TestListChanges(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	var buf bytes.Buffer
	listChanges(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("260401-ab12-add-feature")) {
		t.Error("expected change name in output")
	}
	if bytes.Contains([]byte(output), []byte("archive")) && !bytes.Contains([]byte(output), []byte("Available changes")) {
		t.Error("archive should not appear in list")
	}
}

func TestBatchSwitchCmd_Structure(t *testing.T) {
	cmd := batchSwitchCmd()
	if cmd.Use != "switch [change...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "switch [change...]")
	}

	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}

// batchSwitchFixture creates a fab root with one change folder and a
// fab/project/config.yaml carrying the given providers.claude.session_command
// (the default tier's provider), chdirs into it (via chdirTestEnv, TMUX set), and
// returns the resolvable change name.
func batchSwitchFixture(t *testing.T, sessionCommand string) (root, change string) {
	t.Helper()
	root = t.TempDir()
	change = "260401-ab12-add-feature"
	if err := os.MkdirAll(filepath.Join(root, "fab", "changes", change), 0o755); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "providers:\n  claude:\n    session_command: \"" + sessionCommand + "\"\n"
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})
	return root, change
}

// stubBatchSwitchTmuxCapture stubs `wt` (echoing a worktree path), a `tmux`
// that appends its full argument list to a capture file, and a `git` that reports
// the branch missing (so branchExists routes to the positional and never touches
// the network), all prepended to $PATH. runBatchSwitch invokes each via
// exec.Command / pane.RunCmd (PATH-resolved). Returns the tmux capture file path.
// These tests assert the tmux spawn command, not wt routing — the git stub only
// keeps the probe hermetic.
func stubBatchSwitchTmuxCapture(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	capture := filepath.Join(t.TempDir(), "tmux-args")
	scripts := map[string]string{
		"wt":   "echo /fake/worktrees/switch",
		"tmux": `printf '%s\n' "$@" >> ` + capture,
		"git":  `case "$1" in show-ref) exit 1 ;; ls-remote) exit 0 ;; *) exit 0 ;; esac`,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return capture
}

// stubBatchSwitchRouting stubs `git` (the branchExists probe), an argv-capturing
// `wt`, and a no-op `tmux`, all prepended to $PATH. The `git` stub dispatches on
// its first argument: `show-ref` exits per showRefExit (0 = branch exists locally),
// `ls-remote` prints lsRemoteOut then exits per lsRemoteExit (non-empty stdout with
// exit 0 = branch exists remotely). The `wt` stub appends its full argv to a capture
// file, prints a fake worktree path, and exits per wtExit. Returns the wt-argv
// capture file path. NEVER invokes the real installed wt (whose OLD dual semantics
// differ from the migrated --checkout contract).
func stubBatchSwitchRouting(t *testing.T, showRefExit, lsRemoteExit int, lsRemoteOut string, wtExit int) string {
	t.Helper()
	bin := t.TempDir()
	wtCapture := filepath.Join(t.TempDir(), "wt-args")
	gitBody := `case "$1" in
  show-ref) exit ` + itoa(showRefExit) + ` ;;
  ls-remote) printf '%s' "` + lsRemoteOut + `"; exit ` + itoa(lsRemoteExit) + ` ;;
  *) exit 0 ;;
esac`
	wtBody := `printf '%s\n' "$@" >> ` + wtCapture + `
echo /fake/worktrees/switch
exit ` + itoa(wtExit)
	scripts := map[string]string{
		"git":  gitBody,
		"wt":   wtBody,
		"tmux": "exit 0",
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return wtCapture
}

// itoa is a tiny local int→string helper so the stub bodies read cleanly without
// pulling strconv into the test's import set.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// runBatchSwitchOnce is the shared driver for the routing tests: it builds the
// command, runs runBatchSwitch for the single change, and returns captured stderr.
func runBatchSwitchOnce(t *testing.T, change string) (stderr string, err error) {
	t.Helper()
	cmd := batchSwitchCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err = runBatchSwitch(cmd, []string{change}, false, false)
	return errOut.String(), err
}

// TestRunBatchSwitch_Routing verifies branchExists probe-and-route per wt's 2af2
// contract: existing local branch → --checkout form; remote-only branch → --checkout
// form; missing branch (both probes fail) → positional form; offline ls-remote →
// positional form. The default tier's branch is the change folder name (no prefix
// configured in the fixture).
func TestRunBatchSwitch_Routing(t *testing.T) {
	t.Run("existing local branch routes through --checkout", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude")
		// show-ref exits 0 (local branch exists); ls-remote must NOT be consulted.
		capture := stubBatchSwitchRouting(t, 0, 1, "", 0)

		if stderr, err := runBatchSwitchOnce(t, change); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}
		args, readErr := os.ReadFile(capture)
		if readErr != nil {
			t.Fatalf("reading wt capture: %v", readErr)
		}
		got := string(args)
		if !strings.Contains(got, "--checkout\n"+change) {
			t.Errorf("expected --checkout %s form, got wt argv:\n%s", change, got)
		}
		// The positional (bare change name as the trailing arg, no --checkout before it)
		// must NOT appear — verify --checkout precedes the branch name.
		if !strings.Contains(got, "--reuse") || !strings.Contains(got, "--worktree-name") {
			t.Errorf("expected --reuse --worktree-name retained, got:\n%s", got)
		}
	})

	t.Run("remote-only branch routes through --checkout", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude")
		// show-ref exits 1 (no local branch); ls-remote prints a matching ref, exit 0.
		capture := stubBatchSwitchRouting(t, 1, 0, "abc123\trefs/heads/"+change, 0)

		if stderr, err := runBatchSwitchOnce(t, change); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}
		args, readErr := os.ReadFile(capture)
		if readErr != nil {
			t.Fatalf("reading wt capture: %v", readErr)
		}
		if !strings.Contains(string(args), "--checkout\n"+change) {
			t.Errorf("expected --checkout %s for remote-only branch, got:\n%s", change, string(args))
		}
	})

	t.Run("missing branch routes through positional", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude")
		// show-ref exits 1, ls-remote exits 0 with EMPTY output → branch missing.
		capture := stubBatchSwitchRouting(t, 1, 0, "", 0)

		if stderr, err := runBatchSwitchOnce(t, change); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}
		args, readErr := os.ReadFile(capture)
		if readErr != nil {
			t.Fatalf("reading wt capture: %v", readErr)
		}
		got := string(args)
		if strings.Contains(got, "--checkout") {
			t.Errorf("expected positional form (no --checkout) for missing branch, got:\n%s", got)
		}
		// The change name must be the trailing positional arg.
		if !strings.HasSuffix(strings.TrimRight(got, "\n"), change) {
			t.Errorf("expected trailing positional %s, got:\n%s", change, got)
		}
	})

	t.Run("offline ls-remote degrades to positional", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude")
		// show-ref exits 1 (no local); ls-remote exits non-zero (offline) → not-remote.
		capture := stubBatchSwitchRouting(t, 1, 2, "", 0)

		if stderr, err := runBatchSwitchOnce(t, change); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}
		args, readErr := os.ReadFile(capture)
		if readErr != nil {
			t.Fatalf("reading wt capture: %v", readErr)
		}
		if strings.Contains(string(args), "--checkout") {
			t.Errorf("offline ls-remote must degrade to positional, got:\n%s", string(args))
		}
	})
}

// TestRunBatchSwitch_WtFailureSurfacesStderr verifies that a wt create failure is
// warn-and-skipped (loop continues, no error returned) AND the child stderr is
// surfaced via pane.StderrError in the warning line — the migration signal wt's
// typed exit-2 error carries, which the old .Output() call discarded.
func TestRunBatchSwitch_WtFailureSurfacesStderr(t *testing.T) {
	_, change := batchSwitchFixture(t, "claude")
	bin := t.TempDir()
	// git: branch missing → positional; wt: exit 2 writing a diagnostic to stderr.
	scripts := map[string]string{
		"git":  "case \"$1\" in show-ref) exit 1 ;; ls-remote) exit 0 ;; *) exit 0 ;; esac",
		"wt":   "echo \"Branch 'x' already exists: use --checkout\" >&2\nexit 2",
		"tmux": "exit 0",
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	stderr, err := runBatchSwitchOnce(t, change)
	if err != nil {
		t.Fatalf("wt failure must warn-and-skip (no returned error), got: %v", err)
	}
	if !strings.Contains(stderr, "failed to create worktree for '"+change+"'") {
		t.Errorf("missing warn-and-skip line, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "use --checkout") {
		t.Errorf("wt child stderr not surfaced (pane.StderrError), got:\n%s", stderr)
	}
}

// TestRunBatchSwitch_SpawnCommandProfileInjection verifies that the worker spawn
// command carries the default tier's {model}/{effort} PROFILE — substituted into a
// templated session_command (no literal braces reach tmux), or appended as
// --model/--effort to a non-templated command. The default tier resolves to
// claude/claude-fable-5/xhigh.
func TestRunBatchSwitch_SpawnCommandProfileInjection(t *testing.T) {
	t.Run("templated session_command substituted with the default profile", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "codex -m {model} -c model_reasoning_effort={effort}")
		capture := stubBatchSwitchTmuxCapture(t)

		cmd := batchSwitchCmd()
		var out, errOut bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		if err := runBatchSwitch(cmd, []string{change}, false, false); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, errOut.String())
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		got := string(args)
		if strings.Contains(got, "{model}") || strings.Contains(got, "{effort}") {
			t.Errorf("literal placeholder braces reached tmux:\n%s", got)
		}
		if !strings.Contains(got, "codex -m claude-fable-5 -c model_reasoning_effort=xhigh '/fab-switch") {
			t.Errorf("templated session_command not substituted with the default profile:\n%s", got)
		}
	})

	t.Run("non-templated session_command has the profile appended", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude --dangerously-skip-permissions")
		capture := stubBatchSwitchTmuxCapture(t)

		cmd := batchSwitchCmd()
		var out, errOut bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		if err := runBatchSwitch(cmd, []string{change}, false, false); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, errOut.String())
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		if !strings.Contains(string(args), "claude --dangerously-skip-permissions --model claude-fable-5 --effort xhigh '/fab-switch") {
			t.Errorf("non-templated session_command missing the appended default profile:\n%s", string(args))
		}
	})
}
