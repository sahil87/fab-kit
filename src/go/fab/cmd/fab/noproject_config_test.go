package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// The CONFIG-ONLY commands — `fab agent`, `fab resolve-agent`, `fab config show`
// — resolve a profile or read the cascade; none of them touches change state. Run
// outside any fab/ project they therefore degrade to the project-free cascade
// (env > system > built-in defaults) instead of failing closed with
// "fab/ directory not found".
//
// This is what makes `fab agent` usable as a plain session launcher from any
// directory, and it keeps ~/.fab-kit/config.yaml authoritative outside a project.
// The PROJECT-STATE commands must keep failing closed — a missing fab/ is a real
// error for anything reading or writing change state — which is pinned at the
// bottom of this file. cmd/fab/skill.md states the rule these tests enforce.

// noProjectDir chdirs into a directory with NO fab/ anywhere up the tree, with an
// isolated HOME (so the developer's own ~/.fab-kit/config.yaml cannot leak in) and
// every FAB_* override cleared. Returns the isolated HOME for writeSystemConfig.
//
// t.TempDir() on macOS lives under /var/folders/... and on Linux under /tmp — no
// fab/ on either path — so the upward search genuinely finds nothing.
func noProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	home := t.TempDir()
	env := map[string]string{"HOME": home, "TMUX": ""}
	for _, key := range configscope.DottedKeys() {
		env["FAB_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_"))] = ""
	}
	chdirTestEnv(t, dir, env)
	return home
}

// TestAgentPrint_NoProjectUsesBuiltins: with no fab/ and no system file, `fab
// agent --print` composes the built-in claude command rather than erroring.
func TestAgentPrint_NoProjectUsesBuiltins(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print with no fab/ project: %v — a config-only command must not fail closed", err)
	}
	model, effort := roleFill(t, agent.RoleDefault)
	if want := builtinClaudeCommand(model, effort); out != want {
		t.Errorf("output = %q, want the built-in default %q", out, want)
	}
}

// TestAgentPrint_NoProjectHonorsSystemLayer: the system file still wins outside a
// project — the regression this whole change exists to prevent. A nil-config
// fallback would silently discard it and quietly hand back built-in defaults.
func TestAgentPrint_NoProjectHonorsSystemLayer(t *testing.T) {
	home := noProjectDir(t)
	writeSystemConfig(t, home, "agent:\n  session: codex\n")

	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	if !strings.HasPrefix(out, "codex ") {
		t.Errorf("output = %q, want the system layer's codex session command outside a project", out)
	}
}

// TestAgentPrint_NoProjectEnvOutranksSystem: tier ORDER survives the degradation,
// not just tier presence.
func TestAgentPrint_NoProjectEnvOutranksSystem(t *testing.T) {
	home := noProjectDir(t)
	writeSystemConfig(t, home, "agent:\n  session: codex\n")
	t.Setenv("FAB_AGENT_SESSION", "claude")

	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	if !strings.HasPrefix(out, "claude ") {
		t.Errorf("output = %q, want env's claude to outrank the system layer's codex", out)
	}
}

// TestResolveAgent_NoProjectResolves: `fab resolve-agent` is config-only too.
func TestResolveAgent_NoProjectResolves(t *testing.T) {
	home := noProjectDir(t)
	writeSystemConfig(t, home, "agent:\n  workers: codex\n")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply with no fab/ project: %v", err)
	}
	if !strings.Contains(out, "provider=codex") {
		t.Errorf("output = %q, want provider=codex from the system layer outside a project", out)
	}
}

// TestConfigShow_NoProjectReadsCascade: `fab config show` is read-only, so it
// degrades; its WRITING siblings do not (pinned below).
func TestConfigShow_NoProjectReadsCascade(t *testing.T) {
	home := noProjectDir(t)
	writeSystemConfig(t, home, "dispatch:\n  mode: pane\n")

	cmd := configCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"show", "dispatch.mode"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config show with no fab/ project: %v — a read-only command must not fail closed", err)
	}
	if got := strings.TrimSpace(out.String()); got != "pane" {
		t.Errorf("config show dispatch.mode = %q, want the system layer's %q", got, "pane")
	}
}

func TestConfigUpgradeSystem_NoProject(t *testing.T) {
	home := noProjectDir(t)

	cmd := configCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"upgrade", "--system"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config upgrade --system with no fab/ project: %v", err)
	}
	path := filepath.Join(home, ".fab-kit", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("system upgrade did not create %s: %v", path, err)
	}
	if !strings.Contains(string(data), "# >>> fab reference") {
		t.Fatalf("created system config has no managed fence:\n%s", data)
	}
	if strings.Contains(errOut.String(), "fab/ directory not found") {
		t.Fatalf("system-only upgrade attempted project resolution: %s", errOut.String())
	}
}

// TestProjectStateCommandsStillFailClosed: the other half of the contract. These
// read or write change state, so a missing fab/ stays a hard error — degrading
// them would silently operate on the wrong (or no) project.
func TestProjectStateCommandsStillFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"config set", []string{"set", "dispatch.mode", "pane"}},
		{"config unset", []string{"unset", "dispatch.mode"}},
		{"config init", []string{"init"}},
		{"config upgrade", []string{"upgrade"}},
		{"config upgrade --all", []string{"upgrade", "--all"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			noProjectDir(t)

			cmd := configCmd()
			var out, errOut bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("%s succeeded with no fab/ project — a project-state command must fail closed", tc.name)
			}
			if !strings.Contains(err.Error(), "fab/ directory not found") {
				t.Errorf("%s error = %v, want the fab/-not-found gate", tc.name, err)
			}
		})
	}
}

// guard: t.TempDir() must not itself sit under a fab/ project, or every test above
// would silently exercise the project path instead of the degradation path.
func TestNoProjectDirHasNoFabRootAbove(t *testing.T) {
	noProjectDir(t)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "fab")); err == nil && info.IsDir() {
			t.Fatalf("found a fab/ at %s — the no-project fixture is not project-free, so the tests above are vacuous", dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
