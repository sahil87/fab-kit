package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"gopkg.in/yaml.v3"
)

func TestGitRepoRoot_ReturnsPath(t *testing.T) {
	// This test runs inside the fab-kit repo, so gitRepoRoot should succeed
	root, err := gitRepoRoot()
	if err != nil {
		t.Skipf("not in a git repo: %v", err)
	}
	if root == "" {
		t.Error("gitRepoRoot() returned empty string")
	}
}

func TestOperatorCmd_Structure(t *testing.T) {
	cmd := operatorCmd()
	if cmd.Use != "operator" {
		t.Errorf("Use = %q, want %q", cmd.Use, "operator")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	if cmd.Flags().Lookup("workers") == nil {
		t.Error("missing --workers flag")
	}

	// Verify tick-start and time subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}
	if !subNames["tick-start"] {
		t.Error("operator command missing tick-start subcommand")
	}
	if !subNames["time"] {
		t.Error("operator command missing time subcommand")
	}
}

// stubNoRK forces the built-in launcher arm for tests that exercise it: the
// delegation probe answers false regardless of what the host machine carries
// on PATH. Without the pin, a dev box with a capable rk installed would take
// the delegation branch and syscall.Exec would replace the TEST PROCESS.
func stubNoRK(t *testing.T) {
	t.Helper()
	prev := rkOperatorPath
	rkOperatorPath = func() (string, bool) { return "", false }
	t.Cleanup(func() { rkOperatorPath = prev })
}

// TestRunOperator_NoTmux verifies the $TMUX guard returns an error through
// RunE (previously os.Exit(1), which killed the test process) so main.go's
// central handler formats it as `ERROR: not inside a tmux session`.
func TestRunOperator_NoTmux(t *testing.T) {
	stubNoRK(t)
	t.Setenv("TMUX", "")

	cmd := operatorCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when $TMUX is unset, got nil")
	}
	if err.Error() != "not inside a tmux session" {
		t.Errorf("error = %q, want %q", err.Error(), "not inside a tmux session")
	}
}

func TestRunOperator_WorkersOverride(t *testing.T) {
	stubNoRK(t)
	root := t.TempDir()
	chdirTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})

	bin := t.TempDir()
	capture := filepath.Join(t.TempDir(), "tmux-args")
	scripts := map[string]string{
		"git": "if [ \"$1\" = rev-parse ]; then printf '%s\\n' \"$PWD\"; fi",
		"tmux": `if [ "$1" = list-windows ]; then exit 0; fi
printf '%s\n' "$@" >> ` + capture,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmd := operatorCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--workers", "co'dex; $(touch nope)"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("operator --workers: %v", err)
	}

	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("reading tmux capture: %v", err)
	}
	want := "FAB_AGENT_WORKERS='co'\\''dex; $(touch nope)' "
	if !strings.Contains(string(args), want) || !strings.Contains(string(args), "'/fab-operator'") {
		t.Errorf("tmux command missing safely quoted workers prefix %q:\n%s", want, args)
	}

	// Interactive spawn grammar: the new-window shell-command argument (the
	// LAST captured line of the single new-window call) must end with the
	// shell fallback suffix, with the FAB_AGENT_WORKERS= prefix still leading.
	lines := strings.Split(strings.TrimRight(string(args), "\n"), "\n")
	last := lines[len(lines)-1]
	if !strings.HasSuffix(last, `; exec "$SHELL"`) {
		t.Errorf("new-window shell command missing shell fallback suffix, last arg = %q", last)
	}
	if !strings.HasPrefix(last, "FAB_AGENT_WORKERS=") {
		t.Errorf("FAB_AGENT_WORKERS= prefix must still lead the composed command, last arg = %q", last)
	}
}

func TestRunOperator_WorkersOverrideDoesNotRelaunchExistingSingleton(t *testing.T) {
	stubNoRK(t)
	root := t.TempDir()
	chdirTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})

	bin := t.TempDir()
	capture := filepath.Join(t.TempDir(), "tmux-args")
	tmuxScript := `if [ "$1" = list-windows ]; then
  printf '@7\toperator\n'
  exit 0
fi
printf '%s\n' "$@" >> ` + capture
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte("#!/bin/sh\n"+tmuxScript+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out bytes.Buffer
	cmd := operatorCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--workers", "codex"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("operator --workers with existing singleton: %v", err)
	}

	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("reading tmux capture: %v", err)
	}
	got := string(args)
	if strings.Contains(got, "new-window") || strings.Contains(got, agentWorkersEnv) {
		t.Errorf("existing singleton must be selected without a workers relaunch:\n%s", got)
	}
	if !strings.Contains(got, "select-window\n-t\n@7") || !strings.Contains(got, "switch-client\n-t\n@7") {
		t.Errorf("existing singleton was not selected by exact window id:\n%s", got)
	}
	if out.String() != "Switched to existing operator tab.\n" {
		t.Errorf("stdout = %q", out.String())
	}
}

// TestRKOperatorPath_ProbeAgainstStubbedBinary runs the REAL probe (LookPath +
// `rk operator --help` + token check) against stub rk scripts on an isolated
// PATH: capable (help carries --workers), incapable (exit 0 without the token
// — an rk whose operator predates the flag), failing (non-zero help), and
// absent (empty PATH).
func TestRKOperatorPath_ProbeAgainstStubbedBinary(t *testing.T) {
	cases := []struct {
		name   string
		script string // empty = no rk on PATH
		want   bool
	}{
		{"capable", "#!/bin/sh\necho 'Usage: rk operator [--workers <provider>]'\n", true},
		{"token missing", "#!/bin/sh\necho 'Usage: rk operator'\n", false},
		{"help fails", "#!/bin/sh\nexit 1\n", false},
		{"absent", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			if tc.script != "" {
				if err := os.WriteFile(filepath.Join(bin, "rk"), []byte(tc.script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)
			path, ok := rkOperatorPath()
			if ok != tc.want {
				t.Errorf("rkOperatorPath() ok = %v, want %v", ok, tc.want)
			}
			if tc.want && path != filepath.Join(bin, "rk") {
				t.Errorf("rkOperatorPath() path = %q, want %q", path, filepath.Join(bin, "rk"))
			}
		})
	}
}

// TestRunOperator_DelegatesToRK verifies the delegation branch: with a capable
// rk, bare `fab operator` execs `rk operator` — argv carries --workers exactly
// when the flag was supplied, the environment passes through, and fab's own
// $TMUX precondition is skipped (rk owns its preconditions; TMUX is left unset
// here to prove the built-in guard never runs).
func TestRunOperator_DelegatesToRK(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantArgv []string
	}{
		{"bare", nil, []string{"rk", "operator"}},
		{"workers pass-through", []string{"--workers", "kimi"}, []string{"rk", "operator", "--workers", "kimi"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TMUX", "")
			prevProbe, prevExec := rkOperatorPath, execOperator
			t.Cleanup(func() { rkOperatorPath, execOperator = prevProbe, prevExec })
			rkOperatorPath = func() (string, bool) { return "/fake/bin/rk", true }
			var gotPath string
			var gotArgv, gotEnv []string
			execOperator = func(path string, argv []string, env []string) error {
				gotPath, gotArgv, gotEnv = path, argv, env
				return nil
			}

			cmd := operatorCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("operator delegation: %v", err)
			}
			if gotPath != "/fake/bin/rk" {
				t.Errorf("exec path = %q, want %q", gotPath, "/fake/bin/rk")
			}
			if strings.Join(gotArgv, " ") != strings.Join(tc.wantArgv, " ") {
				t.Errorf("exec argv = %v, want %v", gotArgv, tc.wantArgv)
			}
			if len(gotEnv) == 0 {
				t.Error("exec env is empty, want os.Environ() passed through")
			}
		})
	}
}

// TestRunOperator_DelegationExecErrorPropagates pins the no-fallback rule: a
// failing exec after a passing probe surfaces the error — the built-in
// launcher is never retried (rk may already have mutated tmux state).
func TestRunOperator_DelegationExecErrorPropagates(t *testing.T) {
	prevProbe, prevExec := rkOperatorPath, execOperator
	t.Cleanup(func() { rkOperatorPath, execOperator = prevProbe, prevExec })
	rkOperatorPath = func() (string, bool) { return "/fake/bin/rk", true }
	execCalls := 0
	execOperator = func(path string, argv []string, env []string) error {
		execCalls++
		return fmt.Errorf("exec rk: permission denied")
	}

	cmd := operatorCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("err = %v, want the exec error surfaced", err)
	}
	if execCalls != 1 {
		t.Errorf("exec called %d times, want exactly 1 (no retry)", execCalls)
	}
}

// TestFindWindowExact verifies the exact, server-wide window-name matcher
// that backs the operator singleton check: exact names only (no prefix/glob
// false positives), first match wins, names containing tabs survive the
// bounded split, and session names cannot shift columns (the format carries
// no session field at all).
func TestFindWindowExact(t *testing.T) {
	// `list-windows -a -F '#{window_id}\t#{window_name}'` — windows from
	// sessions _rk-ctl, alpha, and beta; the format emits no session field.
	out := "@2\tzsh\n@0\toperator-logs\n@1\tzsh\n@3\toperator\n"

	t.Run("exact match returns the window ID", func(t *testing.T) {
		id, found := findWindowExact(out, "operator")
		if !found || id != "@3" {
			t.Errorf("findWindowExact = (%q, %t), want (@3, true)", id, found)
		}
	})

	t.Run("prefix name is not a false positive", func(t *testing.T) {
		// Only operator-logs present — the old `select-window -t operator`
		// guard matched it by prefix; the exact matcher must not.
		noReal := "@0\toperator-logs\n@1\tzsh\n"
		if id, found := findWindowExact(noReal, "operator"); found {
			t.Errorf("prefix window operator-logs must not match, got (%q, true)", id)
		}
	})

	t.Run("cross-session window is found (server-wide)", func(t *testing.T) {
		// The operator window lives in another session; enumeration is -a so
		// it is visible regardless of the caller's session.
		id, found := findWindowExact(out, "operator")
		if !found || id != "@3" {
			t.Errorf("cross-session match = (%q, %t), want (@3, true)", id, found)
		}
	})

	t.Run("absent window returns not found", func(t *testing.T) {
		if id, found := findWindowExact(out, "missing"); found {
			t.Errorf("absent window must not match, got (%q, true)", id)
		}
	})

	t.Run("window name containing a tab survives the bounded split", func(t *testing.T) {
		tabbed := "@7\tweird\tname\n"
		id, found := findWindowExact(tabbed, "weird\tname")
		if !found || id != "@7" {
			t.Errorf("tabbed-name match = (%q, %t), want (@7, true)", id, found)
		}
	})

	t.Run("tab in session name cannot shift columns", func(t *testing.T) {
		// A session named "we\tird" holding the operator window: under the
		// old 3-field format ('#{session_name}\t#{window_id}\t#{window_name}')
		// tmux emitted "we\tird\t@3\toperator", which SplitN-3 parsed as
		// name "@3\toperator" — silently missing the match. The 2-field
		// format emits only "@3\toperator" for that window, so the session
		// name cannot influence the parse.
		oldStyleLine := "we\tird\t@3\toperator\n"
		if id, found := findWindowExact(oldStyleLine, "operator"); found && id == "@3" {
			t.Errorf("3-field parsing regression check is vacuous — fixture matched as if session were present (got %q)", id)
		}
		newStyleLine := "@3\toperator\n"
		id, found := findWindowExact(newStyleLine, "operator")
		if !found || id != "@3" {
			t.Errorf("2-field line for a tab-named session's window = (%q, %t), want (@3, true)", id, found)
		}
	})

	t.Run("empty output returns not found", func(t *testing.T) {
		if _, found := findWindowExact("", "operator"); found {
			t.Error("empty output must not match")
		}
	})
}

// TestOperatorRoleResolution verifies the operator-role resolution that backs the
// operator's coordinating-agent model selection, through the SHARED chain the tab
// launcher and `fab agent operator` both walk: a nil config resolves the built-in
// operator default, and a project override is honored (per-field merge over the
// built-in). The expected fill is DERIVED, so a defaults.yaml bump does not reach
// this suite.
//
// This used to test a private operatorProfile() helper. That helper was a second
// copy of the resolution chain and is gone; testing the shared one means the
// assertion now covers the code that actually runs.
func TestOperatorRoleResolution(t *testing.T) {
	model, effort := roleFill(t, agent.RoleOperator)

	// nil config → built-in operator default, composed into the built-in command.
	cmd, got, err := roleSessionCommand(nil, agent.RoleOperator)
	if err != nil {
		t.Fatalf("roleSessionCommand(nil, operator): %v", err)
	}
	if got.Provider != "claude" || got.Model != model || got.Effort != effort {
		t.Errorf("profile = %+v, want the built-in operator default", got)
	}
	if want := strings.TrimSuffix(builtinClaudeCommand(model, effort), "\n"); cmd != want {
		t.Errorf("command = %q, want the built-in claude command %q", cmd, want)
	}

	// A project override of the `operator` role is honored (only effort here;
	// provider+model inherit the built-in via per-field merge).
	cfg := &config.Config{Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{
		"operator": {Effort: "high"},
	}}}
	_, got, err = roleSessionCommand(cfg, agent.RoleOperator)
	if err != nil {
		t.Fatalf("roleSessionCommand(override, operator): %v", err)
	}
	if got.Provider != "claude" || got.Model != model || got.Effort != "high" {
		t.Errorf("profile = %+v, want effort=high with inherited provider+model", got)
	}
}

// TestRoleSessionCommand_EmptyWhenProviderHasNoInteractiveCommand pins the
// three-state contract the two callers branch on: an empty command with a nil
// error and the profile still populated, so the operator can fall back to
// spawn.DefaultSpawnCommand with the role's {model, effort} while `fab agent`
// turns the same signal into an actionable error.
func TestRoleSessionCommand_EmptyWhenProviderHasNoInteractiveCommand(t *testing.T) {
	cfg := &config.Config{
		Agent:     config.AgentConfig{Session: "bare"},
		Providers: map[string]config.ProviderConfig{"bare": {HeadlessCommand: "bare -p"}},
	}

	cmd, profile, err := roleSessionCommand(cfg, agent.RoleOperator)
	if err != nil {
		t.Fatalf("roleSessionCommand: %v — a missing interactive_command is a policy signal, not an error", err)
	}
	if cmd != "" {
		t.Errorf("command = %q, want empty when the provider carries no interactive_command", cmd)
	}
	if profile.Provider != "bare" {
		t.Errorf("profile = %+v, want the resolved profile returned alongside the empty command", profile)
	}
}

// TestOperatorSpawnMatchesAgentRolePath is the anti-drift assertion the shared
// helper exists to make possible: the operator tab and `fab agent operator`
// compose the SAME command. Two implementations of the resolution chain is how
// they silently diverged before.
func TestOperatorSpawnMatchesAgentRolePath(t *testing.T) {
	noProjectDir(t)

	viaAgent, err := runAgentPrint(t, "operator")
	if err != nil {
		t.Fatalf("agent operator --print: %v", err)
	}
	if viaOperator := operatorSpawnCommand(); viaOperator != strings.TrimSuffix(viaAgent, "\n") {
		t.Errorf("operator tab composes %q but `fab agent operator` composes %q — the two paths have drifted",
			viaOperator, strings.TrimSuffix(viaAgent, "\n"))
	}
}

// TestOperatorTickStart_IncrementsCount verifies that tick-start increments
// an existing tick_count, writes last_tick_at, preserves other fields, and
// outputs the correct stdout format.
func TestOperatorTickStart_IncrementsCount(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, ".fab-operator.yaml")

	initial := map[string]interface{}{
		"tick_count": 5,
		"monitored":  map[string]interface{}{},
	}
	raw, err := yaml.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial yaml: %v", err)
	}
	if err := os.WriteFile(yamlPath, raw, 0644); err != nil {
		t.Fatalf("write initial yaml: %v", err)
	}

	operatorStatePathOverride = yamlPath
	t.Cleanup(func() { operatorStatePathOverride = "" })

	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("tick-start failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tick: 6") {
		t.Errorf("stdout %q missing 'tick: 6'", out)
	}
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}

	// Read back and verify YAML
	updated, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read updated yaml: %v", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(updated, &result); err != nil {
		t.Fatalf("unmarshal updated yaml: %v", err)
	}
	if result["tick_count"] != 6 {
		t.Errorf("tick_count = %v, want 6", result["tick_count"])
	}
	lastTickAt, _ := result["last_tick_at"].(string)
	if lastTickAt == "" {
		t.Error("last_tick_at is empty or missing")
	}
	if _, ok := result["monitored"]; !ok {
		t.Error("monitored field was not preserved")
	}
}

// TestOperatorTickStart_MissingFile verifies that tick-start creates
// .fab-operator.yaml with tick_count=1 when the file does not exist.
func TestOperatorTickStart_MissingFile(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "operator-state.yaml")

	operatorStatePathOverride = yamlPath
	t.Cleanup(func() { operatorStatePathOverride = "" })

	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("tick-start failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tick: 1") {
		t.Errorf("stdout %q missing 'tick: 1'", out)
	}

	// Verify file was created
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read created yaml: %v", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal created yaml: %v", err)
	}
	if result["tick_count"] != 1 {
		t.Errorf("tick_count = %v, want 1", result["tick_count"])
	}
	lastTickAt, _ := result["last_tick_at"].(string)
	if lastTickAt == "" {
		t.Error("last_tick_at is empty or missing in created file")
	}
}

// TestOperatorTime_NoInterval verifies that 'fab operator time' with no flags
// outputs exactly one line matching 'now: HH:MM' and no 'next:' line.
func TestOperatorTime_NoInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("time failed: %v", err)
	}

	out := stdout.String()
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}
	if strings.Contains(out, "next:") {
		t.Errorf("stdout %q should not contain 'next:' when --interval not given", out)
	}
}

// TestOperatorTime_WithInterval verifies that --interval 3m produces both
// 'now: HH:MM' and 'next: HH:MM' in stdout.
func TestOperatorTime_WithInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--interval", "3m"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("time --interval 3m failed: %v", err)
	}

	out := stdout.String()
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	nextRe := regexp.MustCompile(`next: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}
	if !nextRe.MatchString(out) {
		t.Errorf("stdout %q missing 'next: HH:MM'", out)
	}
}

// TestOperatorTime_InvalidInterval verifies that an invalid --interval string
// causes the command to return an error (exit 1) and produce no stdout output.
func TestOperatorTime_InvalidInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	cmd.SetArgs([]string{"--interval", "notaduration"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid --interval, got nil")
	}
	if stdoutBuf.Len() != 0 {
		t.Errorf("expected no stdout on error, got %q", stdoutBuf.String())
	}
}

// TestStateDir verifies XDG state base resolution: XDG_STATE_HOME is honored
// only when set AND absolute; otherwise it falls back to $HOME/.local/state.
func TestStateDir(t *testing.T) {
	t.Run("XDG_STATE_HOME absolute is honored", func(t *testing.T) {
		abs := filepath.Join(t.TempDir(), "xdgstate")
		t.Setenv("XDG_STATE_HOME", abs)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		if got != abs {
			t.Errorf("stateDir() = %q, want %q", got, abs)
		}
	})

	t.Run("XDG_STATE_HOME unset falls back to HOME/.local/state", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", home)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		want := filepath.Join(home, ".local", "state")
		if got != want {
			t.Errorf("stateDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_STATE_HOME relative is ignored", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_STATE_HOME", "relative/path")
		t.Setenv("HOME", home)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		want := filepath.Join(home, ".local", "state")
		if got != want {
			t.Errorf("stateDir() = %q, want %q (relative XDG_STATE_HOME must be ignored)", got, want)
		}
	})
}

// TestSlugify verifies the socket-path slug is filesystem-safe, deterministic,
// and collision-free for distinct socket paths.
func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"typical socket path", "/tmp/tmux-1000/default", "tmp-tmux--1000-default"},
		{"custom label socket", "/private/tmp/tmux-501/work", "private-tmp-tmux--501-work"},
		{"no leading separator", "tmp/tmux-1000/default", "tmp-tmux--1000-default"},
		{"empty falls back to default", "", "default"},
		{"separator vs literal dash do not collide", "/tmp/tmux/1000/default", "tmp-tmux-1000-default"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.in)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// Filesystem-safe: no separators remain.
			if strings.ContainsRune(got, '/') || strings.ContainsRune(got, os.PathSeparator) {
				t.Errorf("slugify(%q) = %q contains a path separator", tc.in, got)
			}
			// Deterministic: same input → same output.
			if again := slugify(tc.in); again != got {
				t.Errorf("slugify(%q) not deterministic: %q vs %q", tc.in, got, again)
			}
		})
	}

	t.Run("distinct paths produce distinct slugs", func(t *testing.T) {
		paths := []string{
			"/tmp/tmux-1000/default",
			"/tmp/tmux-1000/work",
			"/tmp/tmux-1001/default",
			"/private/tmp/tmux-501/default",
			// Separator-vs-literal-dash pair that collided before "-" escaping:
			// without doubling "-", both slugified to "tmp-tmux-1000-default".
			"/tmp/tmux/1000/default",
		}
		seen := make(map[string]string)
		for _, p := range paths {
			s := slugify(p)
			if prev, ok := seen[s]; ok {
				t.Errorf("slug collision: %q and %q both → %q", prev, p, s)
			}
			seen[s] = p
		}
	})
}

// TestStatePath verifies the server-keyed state path layout and that the parent
// directory is created. serverSlug shells out to tmux; here we pin stateDir via
// HOME and accept whatever slug serverSlug derives (it falls back to "default"
// when tmux is unavailable in CI).
func TestStatePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)

	got, err := StatePath("")
	if err != nil {
		t.Fatalf("StatePath() error: %v", err)
	}

	dir := filepath.Join(home, ".local", "state", "fab", "operator")
	if filepath.Dir(got) != dir {
		t.Errorf("StatePath() dir = %q, want %q", filepath.Dir(got), dir)
	}
	if filepath.Ext(got) != ".yaml" {
		t.Errorf("StatePath() = %q, want a .yaml file", got)
	}
	// Parent directory must have been created.
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("StatePath() did not create parent dir %q: err=%v", dir, err)
	}
}
