package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// hookTestEnv isolates env vars and cwd for a hook test and restores them on
// t.Cleanup.
func hookTestEnv(t *testing.T, cwd string, envOverrides map[string]string) {
	t.Helper()
	origEnv := map[string]string{}
	for k := range envOverrides {
		origEnv[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
	for k, v := range envOverrides {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	// Chdir for resolve.FabRoot()
	origWd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
}

// setupHookRepoRoot creates a temp repo with a fab/ subdir and returns
// (repoRoot, fabRoot).
func setupHookRepoRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	fabRoot := filepath.Join(root, "fab")
	if err := os.MkdirAll(fabRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, fabRoot
}

// --- session-scoped hooks are now no-op shims (ioku) ---
//
// fab no longer PRODUCES agent lifecycle state. The stop / user-prompt /
// session-start handlers that wrote `.fab-runtime.yaml` `_agents` entries were
// divested — fab is a pure consumer of the `@rk_agent_state` tmux pane-option
// convention (see internal/pane). The handlers survive one release as silent
// no-op shims for un-migrated settings; they MUST exit 0, emit nothing on
// stdout/stderr, and never create a `.fab-runtime.yaml`. The 2.13.6-to-2.14.0
// migration removes the settings entries.

func TestSessionScopedHookShims_SilentNoOp(t *testing.T) {
	cases := []struct {
		name    string
		cmd     func() *cobra.Command
		payload string
	}{
		{"stop", hookStopCmd, `{"session_id":"uuid-stop","transcript_path":"/tmp/t.jsonl","hook_event_name":"Stop"}`},
		{"session-start", hookSessionStartCmd, `{"session_id":"uuid-ss","hook_event_name":"SessionStart"}`},
		{"user-prompt", hookUserPromptCmd, `{"session_id":"uuid-up","hook_event_name":"UserPromptSubmit"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot, _ := setupHookRepoRoot(t)
			// Populate tmux env to prove the shim ignores it (the old handler
			// would have written an entry with these fields).
			hookTestEnv(t, repoRoot, map[string]string{
				"TMUX":      "/tmp/tmux-1001/fabKit,8671,0",
				"TMUX_PANE": "%15",
			})

			cmd := tc.cmd()
			cmd.SetIn(strings.NewReader(tc.payload))
			var out, errOut bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("%s shim must exit 0, got: %v", tc.name, err)
			}
			if out.Len() != 0 {
				t.Errorf("%s shim must emit nothing on stdout, got: %q", tc.name, out.String())
			}
			if errOut.Len() != 0 {
				t.Errorf("%s shim must emit nothing on stderr, got: %q", tc.name, errOut.String())
			}
			if _, err := os.Stat(filepath.Join(repoRoot, ".fab-runtime.yaml")); !os.IsNotExist(err) {
				t.Errorf("%s shim must NOT create .fab-runtime.yaml; stat err=%v", tc.name, err)
			}
		})
	}
}

func TestSessionScopedHookShims_ToleratesJunkInput(t *testing.T) {
	for _, mk := range []func() *cobra.Command{hookStopCmd, hookSessionStartCmd, hookUserPromptCmd} {
		cmd := mk()
		cmd.SetIn(strings.NewReader("not-json"))
		var out, errOut bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s shim must exit 0 on junk input, got: %v", cmd.Use, err)
		}
		if out.Len() != 0 || errOut.Len() != 0 {
			t.Errorf("%s shim must emit nothing, got stdout=%q stderr=%q", cmd.Use, out.String(), errOut.String())
		}
	}
}

// --- fab hook sync exit contract (k4ge): all hook subcommands exit 0 ---

func TestHookSync_FailureSurfacedButExitsZero(t *testing.T) {
	repoRoot, _ := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{})

	// Block the settings write: a directory at the settings.local.json path
	// makes hooklib.Sync fail.
	settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")
	if err := os.MkdirAll(settingsPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync must exit 0 on failure, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "hook sync:") {
		t.Errorf("failure should be surfaced on stderr, got: %q", errOut.String())
	}
}

func TestHookSync_NoFabRootExitsZero(t *testing.T) {
	// A directory tree with no fab/ anywhere up to root would be needed to
	// make resolve.FabRoot fail; t.TempDir() is safe as long as no ancestor
	// has a fab/ dir — guard by skipping if resolution unexpectedly succeeds.
	dir := t.TempDir()
	hookTestEnv(t, dir, map[string]string{})

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync must exit 0 even without a fab root, got: %v", err)
	}
}

// TestHookSync_RegistersNoSessionHooks confirms the divestment: with an empty
// DefaultMappings, a first sync writes a settings file that registers ZERO
// session-scoped hook entries. `Sync` is retained one release but now fully
// inert — it registers nothing and no longer rewrites legacy on-*.sh scripts,
// so it never adds SessionStart/Stop/UserPromptSubmit.
func TestHookSync_RegistersNoSessionHooks(t *testing.T) {
	repoRoot, _ := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{})

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync failed: %v", err)
	}

	// `sync` swallows failures (surfaces them on stderr, exits 0), so an
	// os.IsNotExist early-return could mask a silent no-write. Assert the
	// success signal directly: empty stderr + a "Created"/"hooks: OK" stdout
	// line proves sync actually ran to a write, not that it silently bailed.
	if errOut.Len() != 0 {
		t.Fatalf("hook sync stderr must be empty on success, got: %s", errOut.String())
	}
	stdout := out.String()
	if !strings.Contains(stdout, "hooks: OK") && !strings.Contains(stdout, "Created") {
		t.Fatalf("hook sync stdout must report the OK/Created line, got: %q", stdout)
	}

	settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// No settings file at all is an acceptable "nothing registered" outcome.
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read settings: %v", err)
	}
	for _, event := range []string{"SessionStart", "Stop", "UserPromptSubmit"} {
		if strings.Contains(string(data), event) {
			t.Errorf("settings must not register %s after divestment, got: %s", event, string(data))
		}
	}
}

// --- artifact-write no-op shim ---
//
// Artifact bookkeeping is no longer hook-owned: the change_type/confidence/
// plan-count recompute moved to the pull-based `fab status refresh`
// (internal/refresh, tested in internal/refresh/refresh_test.go) and is
// self-healed at the transition seams. `fab hook artifact-write` is retained
// for one release as a silent no-op shim for un-migrated projects whose
// settings still register it — it MUST exit 0 and emit nothing on stdout (a
// PostToolUse entry parses stdout as additionalContext JSON, so any output
// would be noisy on an un-migrated project).

func TestHookArtifactWrite_ShimIsSilentNoOp(t *testing.T) {
	cmd := hookArtifactWriteCmd()
	// Feed a well-formed PostToolUse payload that the old hook would have acted
	// on; the shim must ignore it entirely.
	cmd.SetIn(strings.NewReader(`{"tool_name":"Write","tool_input":{"file_path":"fab/changes/260310-abcd-x/intake.md"}}`))
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact-write shim must exit 0, got: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("shim must emit nothing on stdout (PostToolUse parses it as additionalContext JSON), got: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("shim must emit nothing on stderr, got: %q", errOut.String())
	}
}

func TestHookArtifactWrite_ShimToleratesJunkInput(t *testing.T) {
	cmd := hookArtifactWriteCmd()
	cmd.SetIn(strings.NewReader("not-json"))
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact-write shim must exit 0 on junk input, got: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("shim must emit nothing on stdout, got: %q", out.String())
	}
}
