package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// agentTestRepo creates a temp repo with fab/project/config.yaml holding the
// given config body and chdirs into the repo root (cwd restored on cleanup).
func agentTestRepo(t *testing.T, configBody string) string {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configBody), 0o644); err != nil {
		t.Fatal(err)
	}
	hookTestEnv(t, root, map[string]string{"TMUX": ""})
	return root
}

// runAgentPrint executes `fab agent --print` with the given extra args.
func runAgentPrint(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := agentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"--print"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// TestAgentPrintDefaultTier: `fab agent --print` with no tier arg resolves the
// default tier (claude/claude-fable-5/xhigh) and appends the profile to the
// non-templated claude session command.
func TestAgentPrintDefaultTier(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude --dangerously-skip-permissions"
`)
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	want := "claude --dangerously-skip-permissions --model claude-fable-5 --effort xhigh\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintOperatorTier: `fab agent operator --print` resolves the operator
// tier (claude-sonnet-5/medium).
func TestAgentPrintOperatorTier(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude"
`)
	out, err := runAgentPrint(t, "operator")
	if err != nil {
		t.Fatalf("agent operator --print: %v", err)
	}
	want := "claude --model claude-sonnet-5 --effort medium\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintTemplatedSessionCommand: a templated session_command has the
// default profile substituted (not appended); no literal braces survive.
func TestAgentPrintTemplatedSessionCommand(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model} -c model_reasoning_effort={effort}"
agent:
  tiers:
    default: { provider: codex }
`)
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	want := "codex -m claude-fable-5 -c model_reasoning_effort=xhigh\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintBuiltinFallback: with no providers config, the built-in claude
// provider supplies the default session command, profile appended.
func TestAgentPrintBuiltinFallback(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	if !strings.Contains(out, "claude --dangerously-skip-permissions") {
		t.Errorf("output = %q, want the built-in claude session command", out)
	}
	if !strings.Contains(out, "--model claude-fable-5 --effort xhigh") {
		t.Errorf("output = %q, want the default-tier profile appended", out)
	}
}

// TestAgentPrintUnknownTierErrors: an unknown tier name exits non-zero.
func TestAgentPrintUnknownTierErrors(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentPrint(t, "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown tier")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the unknown tier, got: %v", err)
	}
}

// TestAgentPrintNoSessionCommandErrors: a resolved provider with no
// session_command (and not the built-in claude) errors with a config-key hint.
func TestAgentPrintNoSessionCommandErrors(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    dispatch_command: "codex exec"
agent:
  tiers:
    default: { provider: codex }
`)
	_, err := runAgentPrint(t)
	if err == nil {
		t.Fatal("expected an error when the resolved provider has no session_command")
	}
	if !strings.Contains(err.Error(), "providers.codex.session_command") {
		t.Errorf("error = %q, want the config-key hint", err.Error())
	}
}

// TestAgentPrintRepoFlag: --repo reads a different repo's config.
func TestAgentPrintRepoFlag(t *testing.T) {
	// The current repo has no providers; the target repo does.
	agentTestRepo(t, "project:\n  name: current\n")

	target := t.TempDir()
	projectDir := filepath.Join(target, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"),
		[]byte("providers:\n  claude:\n    session_command: \"target-claude\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runAgentPrint(t, "--repo", target)
	if err != nil {
		t.Fatalf("agent --print --repo: %v", err)
	}
	if !strings.Contains(out, "target-claude") {
		t.Errorf("output = %q, want the target repo's session command", out)
	}
}
