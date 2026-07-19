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
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})
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

// TestResolveAgentDefaultOutputExactBytes: on a config with no agent.tiers, the
// default output includes model=/effort=/provider= (the byte-stable contract the
// consuming skills rely on). intake ∈ default tier; ship ∈ fast tier.
func TestResolveAgentDefaultOutputExactBytes(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "intake") // default tier: claude/claude-fable-5/high
	if err != nil {
		t.Fatalf("resolve-agent intake: %v", err)
	}
	want := "model=claude-fable-5\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	// ship resolves to the fast tier default.
	out, err = runResolveAgentCmd(t, "ship")
	if err != nil {
		t.Fatalf("resolve-agent ship: %v", err)
	}
	want = "model=claude-sonnet-5\neffort=medium\nprovider=claude\n"
	if out != want {
		t.Errorf("ship output = %q, want %q", out, want)
	}
}

// TestResolveAgentAcceptsTierName: a role-tier name resolves directly (the
// stage/tier positional-arg contract that serves fab agent / operator; shared
// names are fixed points, so tier-first dispatch resolves them identically).
func TestResolveAgentAcceptsTierName(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "operator") // tier name, not a stage
	if err != nil {
		t.Fatalf("resolve-agent operator: %v", err)
	}
	want := "model=claude-sonnet-5\neffort=medium\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the operator tier profile %q", out, want)
	}
}

// TestResolveAgentOverrideMerge: a per-field override (effort only) merges over
// the default model/provider.
func TestResolveAgentOverrideMerge(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    doing: { effort: medium }
`)
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=claude-fable-5\neffort=medium\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want default model/provider + medium effort", out)
	}
}

// TestResolveAgentEmptyOverrideEffortInheritsDefault: an empty override effort is
// a no-op merge — the DEFAULT effort survives (per-field merge).
func TestResolveAgentEmptyOverrideEffortInheritsDefault(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    doing: { model: some-model, effort: "" }
`)
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=some-model\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want overridden model + default effort", out)
	}
}

// TestResolveAgentVerbatimNoValidation: an incompatible override is emitted
// verbatim with exit 0 — fab does not validate.
func TestResolveAgentVerbatimNoValidation(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
    fast: { model: claude-sonnet-5, effort: xhigh }
`)
	out, err := runResolveAgentCmd(t, "ship") // ship ∈ fast
	if err != nil {
		t.Fatalf("resolve-agent ship must not error on an incompatible pair: %v", err)
	}
	want := "model=claude-sonnet-5\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want verbatim incompatible pair", out)
	}
}

// TestResolveAgentUnknownStageErrors: an unknown stage/tier exits non-zero and
// names the argument.
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

// TestResolveAgentPrintsEmptyLinesOmitted: the print contract omits the effort=
// and provider= lines when those fields are empty, and emits an empty model= line
// when the model is empty (the "inherit" signal). Tested at the formatter level
// since today's defaults never resolve to an empty effort/provider.
func TestResolveAgentPrintsEmptyLinesOmitted(t *testing.T) {
	if got := formatAgentProfile(agent.Profile{Model: "some-model"}, ""); got != "model=some-model\n" {
		t.Errorf("empty effort+provider = %q, want %q (both lines omitted)", got, "model=some-model\n")
	}
	if got := formatAgentProfile(agent.Profile{}, ""); got != "model=\n" {
		t.Errorf("all-empty = %q, want %q (inherit signal)", got, "model=\n")
	}
	if got := formatAgentProfile(agent.Profile{Provider: "claude", Model: "m", Effort: "high"}, ""); got != "model=m\neffort=high\nprovider=claude\n" {
		t.Errorf("full profile = %q, want %q", got, "model=m\neffort=high\nprovider=claude\n")
	}
}

// TestResolveAgentPrintsDispatchLine: the print contract appends a dispatch= line
// only when a non-empty dispatch command is passed (native dispatch omits it).
// dispatchLine is the already-substituted command — the formatter emits it
// verbatim.
func TestResolveAgentPrintsDispatchLine(t *testing.T) {
	got := formatAgentProfile(agent.Profile{Provider: "codex", Model: "claude-opus-4-8", Effort: "high"}, "codex exec -m claude-opus-4-8")
	want := "model=claude-opus-4-8\neffort=high\nprovider=codex\ndispatch=codex exec -m claude-opus-4-8\n"
	if got != want {
		t.Errorf("with dispatch line = %q, want %q", got, want)
	}
	// Empty dispatchLine omits the dispatch= line (native Agent-tool dispatch).
	if got := formatAgentProfile(agent.Profile{Provider: "claude", Model: "m", Effort: "high"}, ""); got != "model=m\neffort=high\nprovider=claude\n" {
		t.Errorf("empty dispatch = %q, want the three-line contract", got)
	}
}

// TestResolveAgentAliasEmitsShortAlias: with --alias, a doing stage emits the
// short alias on the model= line while effort=/provider= are unaffected.
func TestResolveAgentAliasEmitsShortAlias(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--alias") // apply ∈ doing → fable
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	want := "model=fable\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestResolveAgentNoAliasEmitsFullID: without --alias the default output is the
// full model ID (regression guard against the alias transform leaking into the
// default path).
func TestResolveAgentNoAliasEmitsFullID(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=claude-fable-5\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestResolveAgentAliasEmptyModelInheritSignal: under --alias, an empty resolved
// model still yields an empty model= line (ModelAlias("") is ""). Asserted at the
// alias+formatter level.
func TestResolveAgentAliasEmptyModelInheritSignal(t *testing.T) {
	if got := agent.ModelAlias(""); got != "" {
		t.Fatalf("ModelAlias(\"\") = %q, want empty (inherit signal preserved under --alias)", got)
	}
	if got := formatAgentProfile(agent.Profile{Model: agent.ModelAlias(""), Effort: "high", Provider: "claude"}, ""); got != "model=\neffort=high\nprovider=claude\n" {
		t.Errorf("empty model under --alias = %q, want %q", got, "model=\neffort=high\nprovider=claude\n")
	}
}

// TestResolveAgentNoDispatchThreeLines: a config whose resolved provider has no
// dispatch_command emits exactly model=/effort=/provider= (no dispatch= line) —
// the "absence signals native dispatch" guard.
func TestResolveAgentNoDispatchThreeLines(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  claude:
    session_command: claude --dangerously-skip-permissions
`)
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=claude-fable-5\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the three-line contract (no dispatch= — session_command is NOT a fallback)", out)
	}
}

// TestResolveAgentDispatchFourLines: a provider with a dispatch_command emits the
// fourth dispatch= line with {model}/{effort} substituted from the resolved
// profile. The tier must point its provider at that dispatch-carrying provider.
func TestResolveAgentDispatchFourLines(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model}"
    dispatch_command: "codex exec -m {model} -c model_reasoning_effort={effort}"
agent:
  tiers:
    doing: { provider: codex }
`)
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing → provider codex
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=claude-fable-5\neffort=xhigh\nprovider=codex\ndispatch=codex exec -m claude-fable-5 -c model_reasoning_effort=xhigh\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestResolveAgentAliasDispatchUsesFullModelID: under --alias the model= line is
// aliased while the dispatch= line embeds the FULL model ID (CLI dispatch never
// aliases) — the load-bearing --alias interaction.
func TestResolveAgentAliasDispatchUsesFullModelID(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    dispatch_command: "codex exec -m {model} -c model_reasoning_effort={effort}"
agent:
  tiers:
    doing: { provider: codex }
`)
	out, err := runResolveAgentCmd(t, "apply", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	want := "model=fable\neffort=xhigh\nprovider=codex\ndispatch=codex exec -m claude-fable-5 -c model_reasoning_effort=xhigh\n"
	if out != want {
		t.Errorf("output = %q, want aliased model= and full-ID dispatch=, got %q", out, want)
	}
}

// TestResolveAgentDispatchSubstitutionReusesSpawnPackage: the dispatch= line's
// {model}/{effort} substitution is delegated to internal/spawn.WithProfile
// (reused, not reimplemented) — non-empty values substitute in place, preserving
// the author's whitespace runs (spawn's whitespace-preserving fast path).
func TestResolveAgentDispatchSubstitutionReusesSpawnPackage(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    dispatch_command: "codex  exec  -m {model}  -c reasoning={effort}"
agent:
  tiers:
    fast: { provider: codex }
`)
	out, err := runResolveAgentCmd(t, "ship") // ship ∈ fast (sonnet/medium), provider codex
	if err != nil {
		t.Fatalf("resolve-agent ship: %v", err)
	}
	want := "model=claude-sonnet-5\neffort=medium\nprovider=codex\ndispatch=codex  exec  -m claude-sonnet-5  -c reasoning=medium\n"
	if out != want {
		t.Errorf("output = %q, want %q (whitespace preserved via spawn.WithProfile)", out, want)
	}
}

// TestResolveAgentDispatchByteStable: repeated resolution with a dispatch_command
// is byte-identical (the dispatch= line participates in the byte-stable contract).
func TestResolveAgentDispatchByteStable(t *testing.T) {
	body := `providers:
  codex:
    dispatch_command: "codex exec -m {model}"
agent:
  tiers:
    doing: { provider: codex }
`
	resolveAgentTestRepo(t, body)
	first, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	second, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply (2nd): %v", err)
	}
	if first != second {
		t.Errorf("dispatch output not byte-stable: %q vs %q", first, second)
	}
	if !strings.Contains(first, "dispatch=codex exec -m claude-fable-5\n") {
		t.Errorf("output = %q, want a substituted dispatch= line", first)
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
