package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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

// wantRoleBytes builds the expected three-line resolve-agent output for a role's
// BUILT-IN default profile. Derived from agent.DefaultProfile so a model bump
// touches only defaults.yaml's providers.claude.profiles — these tests assert the
// output CONTRACT (which lines, what order, exact bytes), not which model is
// current. The literals are pinned once, in agent.TestDefaultRoleProfilesArePinned.
func wantRoleBytes(t *testing.T, role string) string {
	t.Helper()
	p, ok := agent.DefaultProfile(role)
	if !ok {
		t.Fatalf("unknown role %q", role)
	}
	return "model=" + p.Model + "\neffort=" + p.Effort + "\nprovider=" + p.Provider + "\n"
}

// wantRoleModel returns just the built-in default model ID for a role.
func wantRoleModel(t *testing.T, role string) string {
	t.Helper()
	p, ok := agent.DefaultProfile(role)
	if !ok {
		t.Fatalf("unknown role %q", role)
	}
	return p.Model
}

// wantRoleEffort returns just the built-in default effort for a role.
func wantRoleEffort(t *testing.T, role string) string {
	t.Helper()
	p, ok := agent.DefaultProfile(role)
	if !ok {
		t.Fatalf("unknown role %q", role)
	}
	return p.Effort
}

// pinnedRoleLine renders an `agent.profiles` YAML line for `role` that points at
// `provider` while PINNING the role's own built-in claude model/effort. Model and
// effort otherwise come from the RESOLVED provider's own per-role fills, so a role
// pointed at another provider would resolve THAT provider's values. Pinning claude's
// keeps each test's expectation ("the resolved role profile rides the output") true
// and independent of the other providers' fills, while staying DERIVED from the
// canonical defaults — so a model bump touches no test.
func pinnedRoleLine(t *testing.T, role, provider string) string {
	t.Helper()
	p, ok := agent.DefaultProfile(role)
	if !ok {
		t.Fatalf("unknown role %q", role)
	}
	return "    " + role + ": { provider: " + provider + ", model: " + p.Model + ", effort: " + p.Effort + " }\n"
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

// TestResolveAgentDefaultOutputExactBytes: on a config with no agent.profiles, the
// default output includes model=/effort=/provider= (the byte-stable contract the
// consuming skills rely on). intake ∈ `default` role; ship ∈ `fast` role.
func TestResolveAgentDefaultOutputExactBytes(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "intake") // intake ∈ the `default` role
	if err != nil {
		t.Fatalf("resolve-agent intake: %v", err)
	}
	want := wantRoleBytes(t, agent.RoleDefault)
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	// ship resolves to the `fast` role's default.
	out, err = runResolveAgentCmd(t, "ship")
	if err != nil {
		t.Fatalf("resolve-agent ship: %v", err)
	}
	want = "model=claude-sonnet-5\neffort=medium\nprovider=claude\n"
	if out != want {
		t.Errorf("ship output = %q, want %q", out, want)
	}
}

// TestResolveAgentAcceptsRoleName: a role name resolves directly (the stage-or-role
// positional-arg contract that serves fab agent / operator; shared names are fixed
// points, so role-first dispatch resolves them identically).
func TestResolveAgentAcceptsRoleName(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "operator") // role name, not a stage
	if err != nil {
		t.Fatalf("resolve-agent operator: %v", err)
	}
	want := wantRoleBytes(t, agent.RoleFast)
	if out != want {
		t.Errorf("output = %q, want the `operator` role profile %q", out, want)
	}
}

// TestResolveAgentOverrideMerge: a per-field override (effort only) merges over
// the default model/provider.
func TestResolveAgentOverrideMerge(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  profiles:
    doing: { effort: medium }
`)
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=" + wantRoleModel(t, agent.RoleDoing) + "\neffort=medium\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want default model/provider + medium effort", out)
	}
}

// TestResolveAgentEmptyOverrideEffortInheritsDefault: an empty override effort is
// a no-op merge — the DEFAULT effort survives (per-field merge).
func TestResolveAgentEmptyOverrideEffortInheritsDefault(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  profiles:
    doing: { model: some-model, effort: "" }
`)
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=some-model\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want overridden model + default effort", out)
	}
}

// TestResolveAgentVerbatimNoValidation: an incompatible override is emitted
// verbatim with exit 0 — fab does not validate.
func TestResolveAgentVerbatimNoValidation(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  profiles:
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

// TestResolveAgentUnknownStageErrors: an unknown stage/role exits non-zero and
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
	want := "model=" + agent.ModelAlias(wantRoleModel(t, agent.RoleDoing)) + "\neffort=high\nprovider=claude\n"
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
	want := wantRoleBytes(t, agent.RoleDoing)
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
	want := wantRoleBytes(t, agent.RoleDoing)
	if out != want {
		t.Errorf("output = %q, want the three-line contract (no dispatch= — session_command is NOT a fallback)", out)
	}
}

// TestResolveAgentDispatchFourLines: a provider with a dispatch_command emits the
// fourth dispatch= line with {model}/{effort} substituted from the resolved
// profile. The role profile must point its provider at that dispatch-carrying
// provider.
func TestResolveAgentDispatchFourLines(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model}"
    dispatch_command: "codex exec -m {model} -c model_reasoning_effort={effort}"
agent:
  profiles:
`+pinnedRoleLine(t, agent.RoleDoing, "codex"))
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing → provider codex
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantRoleModel(t, agent.RoleDoing)
	want := "model=" + doingModel + "\neffort=high\nprovider=codex\ndispatch=codex exec -m " + doingModel + " -c model_reasoning_effort=high\n"
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
  profiles:
`+pinnedRoleLine(t, agent.RoleDoing, "codex"))
	out, err := runResolveAgentCmd(t, "apply", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	want := "model=" + agent.ModelAlias(wantRoleModel(t, agent.RoleDoing)) + "\neffort=high\nprovider=codex\ndispatch=codex exec -m " + wantRoleModel(t, agent.RoleDoing) + " -c model_reasoning_effort=high\n"
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
  profiles:
`+pinnedRoleLine(t, agent.RoleFast, "codex"))
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
  profiles:
` + pinnedRoleLine(t, agent.RoleDoing, "codex")
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
	if !strings.Contains(first, "dispatch=codex exec -m "+wantRoleModel(t, agent.RoleDoing)+"\n") {
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

// --- Invocation-time overrides (260805-j3cm) ---

// TestResolveAgentOverrideProviderBuiltInFill: `--provider codex` on a default
// config swaps the provider, re-derives dispatch= from the codex BUILT-IN's
// dispatch_command, and — because a swap re-derives model/effort from the NEW
// provider's own fills — resolves codex's shipped `doing` profile rather than
// retaining any claude value. Since 260806-ywkx that profile is a real one, so both
// placeholder tokens are substituted rather than dropped.
//
// Expectations are DERIVED from ResolveProvider, so a fill bump touches
// defaults.yaml and the pinned table in internal/agent only.
func TestResolveAgentOverrideProviderBuiltInFill(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	codex, _ := agent.ResolveProvider(nil, "codex")
	// doing carries an effort of its own and inherits the model from `default`.
	model := codex.Profiles[agent.RoleDefault].Model
	effort := codex.Profiles[agent.RoleDoing].Effort

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex")
	if err != nil {
		t.Fatalf("resolve-agent apply --provider codex: %v", err)
	}
	want := "model=" + model + "\neffort=" + effort + "\nprovider=codex\n" +
		"dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m " + model + " -c model_reasoning_effort=" + effort + "\n"
	if out != want {
		t.Errorf("output = %q, want %q (the swap re-derives from codex's own fills, never claude's)", out, want)
	}
	if strings.Contains(out, "claude") {
		t.Errorf("output = %q — no claude model may leak across the swap", out)
	}
}

// TestResolveAgentOverrideFullTriple: --provider + --model + --effort is the
// top precedence rung — each value lands verbatim on its line and in the
// substituted dispatch= command.
func TestResolveAgentOverrideFullTriple(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex", "--model", "gpt-5.3-codex", "--effort", "high")
	if err != nil {
		t.Fatalf("resolve-agent with overrides: %v", err)
	}
	want := "model=gpt-5.3-codex\neffort=high\nprovider=codex\n" +
		"dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestResolveAgentOverrideProviderTakesFill: an unoverridden model/effort on a
// provider SWAP refills from that provider's fills — and a config carrying the
// pre-2.17.0 FLAT spelling still wins there, because the flat fill is folded into
// the user's own `profiles.default` (260806-ywkx) rather than read as a rung below
// the BUILT-IN one. Without that fold, shipping codex fills would silently shadow
// this pinned model with fab-kit's.
//
// The effort still comes from codex's built-in `doing` fill: the flat spelling is a
// `default`-ROLE value, and a role-specific entry outranks the default entry — the
// same precedence a user writing `profiles.default.effort` by hand would get.
func TestResolveAgentOverrideProviderTakesFill(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    model: gpt-5.3-codex
    effort: high
`)
	codex, _ := agent.ResolveProvider(nil, "codex")
	effort := codex.Profiles[agent.RoleDoing].Effort

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex")
	if err != nil {
		t.Fatalf("resolve-agent apply --provider codex: %v", err)
	}
	want := "model=gpt-5.3-codex\neffort=" + effort + "\nprovider=codex\n" +
		"dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m gpt-5.3-codex -c model_reasoning_effort=" + effort + "\n"
	if out != want {
		t.Errorf("output = %q, want the user's flat-fill model to beat the built-in %q", out, want)
	}
}

// TestResolveAgentOverrideModelWithoutProvider: --model/--effort are valid WITHOUT
// --provider here (a within-role override) — the documented asymmetry with
// `fab agent`, where they are a usage error without --provider.
func TestResolveAgentOverrideModelWithoutProvider(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--effort", "high")
	if err != nil {
		t.Fatalf("bare --effort must be valid on the pure query: %v", err)
	}
	want := "model=" + wantRoleModel(t, agent.RoleDoing) + "\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the role's provider+model with the overridden effort %q", out, want)
	}

	out, err = runResolveAgentCmd(t, "apply", "--model", "claude-haiku-4-5")
	if err != nil {
		t.Fatalf("bare --model must be valid on the pure query: %v", err)
	}
	want = "model=claude-haiku-4-5\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the overridden model with the role's effort %q", out, want)
	}
}

// TestResolveAgentOverrideAliasKeepsNonClaudeVerbatim: --alias is a best-effort
// adapter — an overridden non-Claude model passes through verbatim on model=, and
// dispatch= always embeds the full ID. The unoverridden effort comes from codex's
// own built-in `doing` fill (derived, so a bump does not touch this test).
func TestResolveAgentOverrideAliasKeepsNonClaudeVerbatim(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	codex, _ := agent.ResolveProvider(nil, "codex")
	effort := codex.Profiles[agent.RoleDoing].Effort

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex", "--model", "gpt-5.3-codex", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent with overrides --alias: %v", err)
	}
	want := "model=gpt-5.3-codex\neffort=" + effort + "\nprovider=codex\n" +
		"dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m gpt-5.3-codex -c model_reasoning_effort=" + effort + "\n"
	if out != want {
		t.Errorf("output = %q, want %q (non-Claude model verbatim; full ID in dispatch=)", out, want)
	}
}

// TestResolveAgentOverrideDispatchDisappearsOnNativeSwap: swapping TO a provider
// with no dispatch_command drops the dispatch= line — the QUERY reports the named
// provider's dispatch_command (or its absence), which is all this assertion covers.
// It is NOT an adapter move: `fab dispatch start` takes no override flags and
// re-resolves the stage from config, so only a config/role override relocates a
// stage between native Agent-tool dispatch and CLI dispatch.
func TestResolveAgentOverrideDispatchDisappearsOnNativeSwap(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  profiles:
`+pinnedRoleLine(t, agent.RoleDoing, "codex"))

	// Baseline: the codex-pointed role emits a dispatch= line.
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	if !strings.Contains(out, "dispatch=") {
		t.Fatalf("baseline output = %q, want a dispatch= line", out)
	}

	// Swapping to claude (no built-in dispatch_command) drops it.
	out, err = runResolveAgentCmd(t, "apply", "--provider", "claude", "--model", "claude-opus-5", "--effort", "xhigh")
	if err != nil {
		t.Fatalf("resolve-agent apply --provider claude: %v", err)
	}
	want := "model=claude-opus-5\neffort=xhigh\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want %q (no dispatch= — native Agent-tool dispatch)", out, want)
	}
}

// TestResolveAgentOverrideUnknownProviderErrors: a supplied --provider that
// resolves to nothing is a LOOKUP failure naming the resolvable set — mirroring
// `fab agent`'s error. An explicitly-empty --provider= is the same failure (the
// guard keys on supplied-ness), with a placeholder in the config-key hint so the
// suggested path is never malformed.
func TestResolveAgentOverrideUnknownProviderErrors(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--provider", "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown provider")
	}
	// No profile is emitted on the lookup failure. cobra prints its usage block to
	// the same buffer (and that block's own prose mentions `model=`), so assert that
	// no LINE is a contract line rather than that the buffer is empty.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "model=") || strings.HasPrefix(line, "dispatch=") {
			t.Errorf("output = %q, want no resolved profile printed on the lookup failure", out)
			break
		}
	}
	msg := err.Error()
	if !strings.Contains(msg, "bogus") {
		t.Errorf("error should name the unknown provider, got: %v", err)
	}
	for _, want := range []string{"claude", "codex", "gemini"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should list the resolvable provider %q, got: %v", want, err)
		}
	}

	_, err = runResolveAgentCmd(t, "apply", "--provider=")
	if err == nil {
		t.Fatal("expected an error for an explicitly-empty --provider=")
	}
	if !strings.Contains(err.Error(), "providers.<name>") {
		t.Errorf("error = %q, want the placeholder config-key hint for an empty name", err.Error())
	}
}

// TestResolveCrossScopeRoleProfileMerge pins how the two config SCOPES compose a
// single role profile. internal/config.LoadPath deep-merges the system layer
// (~/.fab-kit/config.yaml) and the project layer PER KEY before internal/agent
// resolves anything, so a role named in both scopes reaches resolution as ONE
// merged `agent.profiles.<role>` map — here the system's model/effort beside the
// project's provider.
//
// The resolved bytes are what the fill precedence says they should be: an explicit
// `agent.profiles.<role>.model` is a pin the USER wrote, so it outranks the
// resolved provider's own fills and survives the project scope's provider switch
// (260806-j9nh — there is no cross-provider cutoff rule any more, because with
// model/effort otherwise sourced from the resolved provider's own per-role fills
// nothing is inherited across providers in the first place).
//
// The sharp edge is worth pinning rather than merely documenting: the project
// author sees only `{provider: gemini}` in their own file, yet gets the machine-wide
// layer's codex model ID on a gemini invocation. The escape is the documented one —
// do not pin a role's model in one scope and swap its provider in another; set the
// model in the SAME scope as the switch, or leave it to the provider's fills. This
// test lives in cmd/fab because that is the layer that can compose both scopes
// end-to-end (TestMain already isolates HOME for the package; this test points it at
// its own tree so it can WRITE a system config).
func TestResolveCrossScopeRoleProfileMerge(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// System scope: a fully-pinned codex doing role plus codex's own fills.
	systemConfig := `providers:
  codex:
    profiles:
      default: { model: gpt-5.3-codex, effort: high }
agent:
  profiles:
    doing: { provider: codex, model: gpt-5.3-codex, effort: high }
`
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte(systemConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project scope: switch the SAME role to gemini, supplying only the provider.
	// resolveAgentTestRepo chdirs into the new repo; it does not touch HOME, so the
	// t.Setenv above stands.
	resolveAgentTestRepo(t, `project:
  name: test
providers:
  gemini:
    profiles:
      default: { model: gemini-2.5-pro }
agent:
  profiles:
    doing: { provider: gemini }
`)

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}

	// The merged role profile carries the system scope's explicit model/effort and
	// the project scope's provider, and an explicit role-level pin outranks the
	// resolved provider's fills — so gemini is invoked with the codex model ID
	// rather than with providers.gemini.profiles.default.model.
	want := "model=gpt-5.3-codex\neffort=high\nprovider=gemini\ndispatch=gemini --approval-mode=yolo -m gpt-5.3-codex\n"
	if out != want {
		t.Errorf("output = %q, want %q\n(the two scopes deep-merge into one agent.profiles.doing map "+
			"before resolution, and an explicit role-level model/effort outranks the resolved provider's "+
			"own fills — see docs/specs/stage-models.md § Fill precedence)", out, want)
	}
}

// TestResolveCrossScopeLegacyAliasPrecedence pins the ONE cross-scope case where the
// documented `project > system` precedence inverts: the deprecated `agent.tiers`
// alias resolves AFTER the scope cascade, so for a role written in the NEW spelling
// in one scope and the LEGACY spelling in the other, the SPELLING decides rather
// than the scope. internal/config.LoadPath merges the two layers per key, leaving
// `profiles` and `tiers` as two separate maps; GetAgentProfile then prefers
// `profiles` wherever it carries the role — including when that entry came from the
// system layer and the project layer's is the legacy spelling.
//
// It is pinned rather than fixed: making the alias cascade-aware would mean
// threading per-scope, per-key provenance through LoadPath for a spelling that
// exists only for the pre-migration window. The escape is the migration itself
// (2.16.19-to-2.17.0 sweeps BOTH scopes, so no half-migrated pair survives it) or
// moving the losing scope's role to `profiles`. Sibling of
// TestResolveCrossScopeRoleProfileMerge, and lives here for the same reason: cmd/fab
// is the layer that can compose both scopes end-to-end. See
// config.GetAgentProfile's LIMITATION note.
func TestResolveCrossScopeLegacyAliasPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// System scope: the NEW spelling for the doing role.
	systemConfig := "agent:\n  profiles:\n    doing: { model: system-new-spelling }\n"
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte(systemConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project scope: the SAME role in the LEGACY spelling. Normally the project
	// layer wins; here the spelling does.
	resolveAgentTestRepo(t, `project:
  name: test
agent:
  tiers:
    doing: { model: project-legacy-spelling }
`)

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := "model=system-new-spelling\neffort=" + wantRoleEffort(t, agent.RoleDoing) + "\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want %q\n(the agent.tiers alias resolves AFTER the cascade, so a system-layer "+
			"agent.profiles.<role> beats a project-layer agent.tiers.<role> — run the 2.16.19-to-2.17.0 "+
			"migration, which sweeps both scopes, or move the project role to agent.profiles)", out, want)
	}

	// Control: with the project scope on the NEW spelling too, the normal
	// project > system precedence holds — the inversion is the alias's, not the
	// cascade's.
	resolveAgentTestRepo(t, `project:
  name: test
agent:
  profiles:
    doing: { model: project-new-spelling }
`)
	out, err = runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply (control): %v", err)
	}
	want = "model=project-new-spelling\neffort=" + wantRoleEffort(t, agent.RoleDoing) + "\nprovider=claude\n"
	if out != want {
		t.Errorf("control output = %q, want %q (same spelling in both scopes ⇒ project wins)", out, want)
	}
}

// ---------------------------------------------------------------------------
// dispatch.watchable — the watchable-pane opt-in (tmux presence decides pane vs
// native for a session_command-only provider).
// ---------------------------------------------------------------------------

// TestDispatchLineFor_Matrix is the full emission matrix over the PURE helper: the
// provider's two command fields × the opt-in × $TMUX. It pins the precedence (a
// dispatch_command always wins), the three-way AND the watchable trigger requires,
// and the unknown-provider short-circuit.
func TestDispatchLineFor_Matrix(t *testing.T) {
	const sess = "claude -n {model}"
	const disp = "codex exec -m {model}"

	tests := []struct {
		name      string
		prov      config.ProviderConfig
		known     bool
		watchable bool
		tmux      string
		want      string
	}{
		// Trigger 1 — a dispatch_command wins in every combination.
		{"dispatch_command, watchable off, no tmux", config.ProviderConfig{SessionCommand: sess, DispatchCommand: disp}, true, false, "", disp},
		{"dispatch_command, watchable off, in tmux", config.ProviderConfig{SessionCommand: sess, DispatchCommand: disp}, true, false, "/tmp/tmux-1000/default,1,0", disp},
		{"dispatch_command, watchable on, in tmux", config.ProviderConfig{SessionCommand: sess, DispatchCommand: disp}, true, true, "/tmp/tmux-1000/default,1,0", disp},
		{"dispatch_command only (no session), watchable on, in tmux", config.ProviderConfig{DispatchCommand: disp}, true, true, "/tmp/tmux-1000/default,1,0", disp},

		// Trigger 2 — session_command-only provider needs ALL of watchable + $TMUX.
		{"session only, watchable off, no tmux", config.ProviderConfig{SessionCommand: sess}, true, false, "", ""},
		{"session only, watchable off, in tmux", config.ProviderConfig{SessionCommand: sess}, true, false, "/tmp/tmux-1000/default,1,0", ""},
		// An EMPTY $TMUX reads as unset — Go cannot distinguish the two and tmux
		// never exports an empty value (the SelectMode reading).
		{"session only, watchable on, no tmux", config.ProviderConfig{SessionCommand: sess}, true, true, "", ""},
		{"session only, watchable on, in tmux ⇒ session_command", config.ProviderConfig{SessionCommand: sess}, true, true, "/tmp/tmux-1000/default,1,0", sess},

		// Neither field: nothing to emit, opt-in or not.
		{"no commands, watchable on, in tmux", config.ProviderConfig{}, true, true, "/tmp/tmux-1000/default,1,0", ""},

		// An unknown provider short-circuits before either trigger.
		{"unknown provider, watchable on, in tmux", config.ProviderConfig{SessionCommand: sess}, false, true, "/tmp/tmux-1000/default,1,0", ""},
		{"unknown provider with dispatch_command", config.ProviderConfig{DispatchCommand: disp}, false, false, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := dispatchLineFor(tc.prov, tc.known, tc.watchable, tc.tmux); got != tc.want {
				t.Errorf("dispatchLineFor = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestResolveAgentWatchableEmitsSessionCommandInTmux: end-to-end, `dispatch.watchable:
// true` inside tmux makes the built-in claude provider (session_command only, NO
// dispatch_command) emit a dispatch= line carrying the PROFILE-SUBSTITUTED
// session_command — the watchable pane opt-in.
func TestResolveAgentWatchableEmitsSessionCommandInTmux(t *testing.T) {
	resolveAgentTestRepo(t, `dispatch:
  watchable: true
providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing → claude
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantRoleModel(t, agent.RoleDoing)
	want := "model=" + doingModel + "\neffort=high\nprovider=claude\n" +
		"dispatch=claude -n " + doingModel + " --effort high\n"
	if out != want {
		t.Errorf("output = %q, want %q (watchable + $TMUX ⇒ dispatch= from session_command)", out, want)
	}
}

// TestResolveAgentWatchableOmitsLineOutsideTmux: with the SAME config but $TMUX
// unset, the line is omitted — the stage stays on NATIVE Agent-tool dispatch (not
// headless CLI). tmux presence is what decides pane-vs-native.
func TestResolveAgentWatchableOmitsLineOutsideTmux(t *testing.T) {
	resolveAgentTestRepo(t, `dispatch:
  watchable: true
providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	// resolveAgentTestRepo already unsets TMUX; assert the no-tmux arm explicitly.
	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := wantRoleBytes(t, agent.RoleDoing)
	if out != want {
		t.Errorf("output = %q, want the three-line contract %q (no $TMUX ⇒ native dispatch, never headless CLI)", out, want)
	}
}

// TestResolveAgentWatchableOffInTmuxOmitsLine: the DEFAULT (opt-in absent) is
// byte-stable even inside tmux — the whole point of defaulting to false.
func TestResolveAgentWatchableOffInTmuxOmitsLine(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := wantRoleBytes(t, agent.RoleDoing)
	if out != want {
		t.Errorf("output = %q, want the three-line contract %q (watchable defaults to false)", out, want)
	}
}

// TestResolveAgentWatchableDispatchCommandPrecedence: a provider that DOES carry a
// dispatch_command emits that command, not its session_command, even with the
// opt-in on inside tmux. Watchable only ADDS eligibility; it never rewrites an
// existing CLI-dispatch opt-in.
func TestResolveAgentWatchableDispatchCommandPrecedence(t *testing.T) {
	resolveAgentTestRepo(t, `dispatch:
  watchable: true
providers:
  codex:
    session_command: "codex -m {model}"
    dispatch_command: "codex exec -m {model} -c model_reasoning_effort={effort}"
agent:
  profiles:
`+pinnedRoleLine(t, agent.RoleDoing, "codex"))
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantRoleModel(t, agent.RoleDoing)
	want := "model=" + doingModel + "\neffort=high\nprovider=codex\n" +
		"dispatch=codex exec -m " + doingModel + " -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want the dispatch_command %q (dispatch_command wins over watchable)", out, want)
	}
}

// TestResolveAgentWatchableAliasKeepsFullModelIDInDispatch: --alias aliases the
// model= line while the watchable dispatch= line still embeds the FULL model ID —
// the same rule the dispatch_command path follows (the flag's behavior is
// unaffected by the new trigger).
func TestResolveAgentWatchableAliasKeepsFullModelIDInDispatch(t *testing.T) {
	resolveAgentTestRepo(t, `dispatch:
  watchable: true
providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	doingModel := wantRoleModel(t, agent.RoleDoing)
	want := "model=" + agent.ModelAlias(doingModel) + "\neffort=high\nprovider=claude\n" +
		"dispatch=claude -n " + doingModel + " --effort high\n"
	if out != want {
		t.Errorf("output = %q, want aliased model= with a full-ID dispatch= %q", out, want)
	}
}

// TestResolveAgentWatchableFromSystemConfig: dispatch.watchable is scope `both`, so
// setting it ONCE in ~/.fab-kit/config.yaml applies to a repo whose project config
// never mentions it — the machine-wide-opt-in requirement. (Cascade pruning would
// silently drop a project-scoped key here; this is the guard that it is not.)
func TestResolveAgentWatchableFromSystemConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte("dispatch:\n  watchable: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// resolveAgentTestRepo chdirs but does not touch HOME, so the t.Setenv stands.
	resolveAgentTestRepo(t, `project:
  name: test
providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantRoleModel(t, agent.RoleDoing)
	want := "model=" + doingModel + "\neffort=high\nprovider=claude\n" +
		"dispatch=claude -n " + doingModel + " --effort high\n"
	if out != want {
		t.Errorf("output = %q, want %q (dispatch.watchable is scope `both` — honored from the system layer)", out, want)
	}
}

// TestResolveAgentWatchableProjectOverridesSystem: the project layer wins over the
// system layer for a `both`-scoped field — a machine-wide `true` is switchable off
// per repo.
func TestResolveAgentWatchableProjectOverridesSystem(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte("dispatch:\n  watchable: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolveAgentTestRepo(t, `dispatch:
  watchable: false
providers:
  claude:
    session_command: "claude -n {model} --effort {effort}"
`)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	want := wantRoleBytes(t, agent.RoleDoing)
	if out != want {
		t.Errorf("output = %q, want the three-line contract %q (project `false` must beat system `true`)", out, want)
	}
}
