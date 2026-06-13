package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
)

// resolveAgentTestRepo creates a temp repo with fab/project/config.yaml holding
// the given config body and chdirs into the repo root (cwd restored on cleanup).
func resolveAgentTestRepo(t *testing.T, configBody string) {
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
}

// runResolveAgentCmd executes a fresh resolveAgentCmd with the given args.
func runResolveAgentCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := resolveAgentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestResolveAgentDefaultOutputExactBytes: on a config with no agent.tiers, a
// thinking stage emits exactly `model=claude-opus-4-8\neffort=xhigh\n` (the
// byte-stable contract the consuming skills rely on).
func TestResolveAgentDefaultOutputExactBytes(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "intake")
	if err != nil {
		t.Fatalf("resolve-agent intake: %v", err)
	}
	if out != "model=claude-opus-4-8\neffort=xhigh\n" {
		t.Errorf("output = %q, want %q", out, "model=claude-opus-4-8\neffort=xhigh\n")
	}

	// ship resolves to the one non-Opus default.
	out, err = runResolveAgentCmd(t, "ship")
	if err != nil {
		t.Fatalf("resolve-agent ship: %v", err)
	}
	if out != "model=claude-sonnet-4-6\neffort=low\n" {
		t.Errorf("ship output = %q, want %q", out, "model=claude-sonnet-4-6\neffort=low\n")
	}
}

// TestResolveAgentOverrideMerge: a per-field override (effort only) merges over
// the default model.
func TestResolveAgentOverrideMerge(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    doing: { effort: medium }
`)
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	if out != "model=claude-opus-4-8\neffort=medium\n" {
		t.Errorf("output = %q, want default model + medium effort", out)
	}
}

// TestResolveAgentEmptyOverrideEffortInheritsDefault: an empty override effort
// is a no-op merge — the DEFAULT effort survives (per-field merge). This is the
// observable behavior of an "effort: """ override; the effort= line is only
// truly omitted when the RESOLVED effort is empty (not reachable with today's
// defaults, all of which carry an effort — exercised at the print level by
// TestResolveAgentPrintsEmptyEffortOmitted).
func TestResolveAgentEmptyOverrideEffortInheritsDefault(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    doing: { model: some-model, effort: "" }
`)
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	if out != "model=some-model\neffort=high\n" {
		t.Errorf("output = %q, want overridden model + default effort", out)
	}
}

// TestResolveAgentVerbatimNoValidation: an incompatible override is emitted
// verbatim with exit 0 — fab does not validate.
func TestResolveAgentVerbatimNoValidation(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    fast: { model: claude-sonnet-4-6, effort: xhigh }
`)
	out, err := runResolveAgentCmd(t, "ship")
	if err != nil {
		t.Fatalf("resolve-agent ship must not error on an incompatible pair: %v", err)
	}
	if out != "model=claude-sonnet-4-6\neffort=xhigh\n" {
		t.Errorf("output = %q, want verbatim incompatible pair", out)
	}
}

// TestResolveAgentUnknownStageErrors: an unknown stage exits non-zero and names
// the stage.
func TestResolveAgentUnknownStageErrors(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	_, err := runResolveAgentCmd(t, "frobnicate")
	if err == nil {
		t.Fatal("expected an error for an unknown stage")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error should name the unknown stage, got: %v", err)
	}
}

// TestResolveAgentPrintsEmptyEffortOmitted: the print contract omits the effort=
// line when the resolved effort is empty, and emits an empty model= line when
// the model is empty (the "inherit" signal). Tested at the formatter level since
// today's defaults never resolve to an empty effort.
func TestResolveAgentPrintsEmptyEffortOmitted(t *testing.T) {
	if got := formatAgentProfile(agent.Profile{Model: "some-model", Effort: ""}); got != "model=some-model\n" {
		t.Errorf("empty effort = %q, want %q (effort line omitted)", got, "model=some-model\n")
	}
	if got := formatAgentProfile(agent.Profile{Model: "", Effort: ""}); got != "model=\n" {
		t.Errorf("empty model+effort = %q, want %q (inherit signal)", got, "model=\n")
	}
	if got := formatAgentProfile(agent.Profile{Model: "m", Effort: "high"}); got != "model=m\neffort=high\n" {
		t.Errorf("full profile = %q, want %q", got, "model=m\neffort=high\n")
	}
}

// TestResolveAgentAliasEmitsShortAlias: with --alias, a doing stage emits the
// short alias on the model= line while the effort= line is unaffected.
func TestResolveAgentAliasEmitsShortAlias(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	if out != "model=opus\neffort=high\n" {
		t.Errorf("output = %q, want %q", out, "model=opus\neffort=high\n")
	}
}

// TestResolveAgentNoAliasEmitsFullID: without --alias the default output is the
// full model ID, byte-identical to today (regression guard against the alias
// transform leaking into the default path).
func TestResolveAgentNoAliasEmitsFullID(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	if out != "model=claude-opus-4-8\neffort=high\n" {
		t.Errorf("output = %q, want %q", out, "model=claude-opus-4-8\neffort=high\n")
	}
}

// TestResolveAgentAliasEmptyModelInheritSignal: under --alias, an empty resolved
// model still yields an empty model= line (the inherit signal is preserved —
// ModelAlias("") is ""). Asserted at the alias+formatter level, because today's
// configs never RESOLVE to an empty model (an empty override is a no-op merge
// that keeps the default — see agent.TestResolveEmptyModelInherit), the same
// reason TestResolveAgentPrintsEmptyEffortOmitted tests the empty effort branch
// at the formatter level.
func TestResolveAgentAliasEmptyModelInheritSignal(t *testing.T) {
	if got := agent.ModelAlias(""); got != "" {
		t.Fatalf("ModelAlias(\"\") = %q, want empty (inherit signal preserved under --alias)", got)
	}
	if got := formatAgentProfile(agent.Profile{Model: agent.ModelAlias(""), Effort: "high"}); got != "model=\neffort=high\n" {
		t.Errorf("empty model under --alias = %q, want %q", got, "model=\neffort=high\n")
	}
}

// TestResolveAgentByteStable: repeated resolution of the same stage on the same
// config is byte-identical.
func TestResolveAgentByteStable(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	first, err := runResolveAgentCmd(t, "review")
	if err != nil {
		t.Fatalf("resolve-agent review: %v", err)
	}
	second, err := runResolveAgentCmd(t, "review")
	if err != nil {
		t.Fatalf("resolve-agent review (2nd): %v", err)
	}
	if first != second {
		t.Errorf("output not byte-stable: %q vs %q", first, second)
	}
}
