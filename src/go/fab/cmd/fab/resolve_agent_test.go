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

// wantTierBytes builds the expected three-line resolve-agent output for a tier's
// BUILT-IN default profile. Derived from agent.DefaultTier so a model bump touches
// only defaultTiers in internal/agent — these tests assert the output CONTRACT
// (which lines, what order, exact bytes), not which model is current. The literals
// are pinned once, in agent.TestDefaultTierProfilesArePinned.
func wantTierBytes(t *testing.T, tier string) string {
	t.Helper()
	p, ok := agent.DefaultTier(tier)
	if !ok {
		t.Fatalf("unknown tier %q", tier)
	}
	return "model=" + p.Model + "\neffort=" + p.Effort + "\nprovider=" + p.Provider + "\n"
}

// wantTierModel returns just the built-in default model ID for a tier.
func wantTierModel(t *testing.T, tier string) string {
	t.Helper()
	p, ok := agent.DefaultTier(tier)
	if !ok {
		t.Fatalf("unknown tier %q", tier)
	}
	return p.Model
}

// pinnedTierLine renders an `agent.tiers` YAML line for `tier` that points at
// `provider` while PINNING the tier's own built-in model/effort. Pointing a tier at
// a non-claude provider is a cross-provider switch, and such a tier no longer
// inherits the built-in's (claude-shaped) model/effort — 260805-j3cm's
// cross-provider cutoff. Pinning the same values the built-in carries keeps each
// test's expectation ("the resolved tier profile rides the output") true while
// staying DERIVED from the canonical map, so a model bump touches no test.
func pinnedTierLine(t *testing.T, tier, provider string) string {
	t.Helper()
	p, ok := agent.DefaultTier(tier)
	if !ok {
		t.Fatalf("unknown tier %q", tier)
	}
	return "    " + tier + ": { provider: " + provider + ", model: " + p.Model + ", effort: " + p.Effort + " }\n"
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

	out, err := runResolveAgentCmd(t, "intake") // intake ∈ default tier
	if err != nil {
		t.Fatalf("resolve-agent intake: %v", err)
	}
	want := wantTierBytes(t, agent.TierDefault)
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
	want := wantTierBytes(t, agent.TierFast)
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
	want := "model=" + wantTierModel(t, agent.TierDoing) + "\neffort=medium\nprovider=claude\n"
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
	want := "model=some-model\neffort=high\nprovider=claude\n"
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
	want := "model=" + agent.ModelAlias(wantTierModel(t, agent.TierDoing)) + "\neffort=high\nprovider=claude\n"
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
	want := wantTierBytes(t, agent.TierDoing)
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
	want := wantTierBytes(t, agent.TierDoing)
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
`+pinnedTierLine(t, agent.TierDoing, "codex"))
	out, err := runResolveAgentCmd(t, "apply") // apply ∈ doing → provider codex
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantTierModel(t, agent.TierDoing)
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
  tiers:
`+pinnedTierLine(t, agent.TierDoing, "codex"))
	out, err := runResolveAgentCmd(t, "apply", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent apply --alias: %v", err)
	}
	want := "model=" + agent.ModelAlias(wantTierModel(t, agent.TierDoing)) + "\neffort=high\nprovider=codex\ndispatch=codex exec -m " + wantTierModel(t, agent.TierDoing) + " -c model_reasoning_effort=high\n"
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
`+pinnedTierLine(t, agent.TierFast, "codex"))
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
` + pinnedTierLine(t, agent.TierDoing, "codex")
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
	if !strings.Contains(first, "dispatch=codex exec -m "+wantTierModel(t, agent.TierDoing)+"\n") {
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

// TestResolveAgentOverrideProviderNoFill: `--provider codex` on a default config
// swaps the provider, re-derives dispatch= from the codex BUILT-IN's
// dispatch_command, and — because a swap does not retain the old provider's
// model/effort and no codex fill is configured — resolves an empty model with the
// effort= line omitted. Both placeholder tokens (and their preceding flags) drop out
// of the dispatch command, so the codex CLI's own default model applies.
func TestResolveAgentOverrideProviderNoFill(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex")
	if err != nil {
		t.Fatalf("resolve-agent apply --provider codex: %v", err)
	}
	want := "model=\nprovider=codex\ndispatch=codex exec\n"
	if out != want {
		t.Errorf("output = %q, want %q (no claude model may leak across the swap)", out, want)
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
		"dispatch=codex exec -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestResolveAgentOverrideProviderTakesFill: an unoverridden model/effort on a
// provider SWAP refills from that provider's default fill (providers.<name>.model /
// .effort) — precedence rung 3.
func TestResolveAgentOverrideProviderTakesFill(t *testing.T) {
	resolveAgentTestRepo(t, `providers:
  codex:
    model: gpt-5.3-codex
    effort: high
`)
	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex")
	if err != nil {
		t.Fatalf("resolve-agent apply --provider codex: %v", err)
	}
	want := "model=gpt-5.3-codex\neffort=high\nprovider=codex\n" +
		"dispatch=codex exec -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want the provider fill %q", out, want)
	}
}

// TestResolveAgentOverrideModelWithoutProvider: --model/--effort are valid WITHOUT
// --provider here (a within-tier override) — the documented asymmetry with
// `fab agent`, where they are a usage error without --provider.
func TestResolveAgentOverrideModelWithoutProvider(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--effort", "high")
	if err != nil {
		t.Fatalf("bare --effort must be valid on the pure query: %v", err)
	}
	want := "model=" + wantTierModel(t, agent.TierDoing) + "\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the tier's provider+model with the overridden effort %q", out, want)
	}

	out, err = runResolveAgentCmd(t, "apply", "--model", "claude-haiku-4-5")
	if err != nil {
		t.Fatalf("bare --model must be valid on the pure query: %v", err)
	}
	want = "model=claude-haiku-4-5\neffort=high\nprovider=claude\n"
	if out != want {
		t.Errorf("output = %q, want the overridden model with the tier's effort %q", out, want)
	}
}

// TestResolveAgentOverrideAliasKeepsNonClaudeVerbatim: --alias is a best-effort
// adapter — an overridden non-Claude model passes through verbatim on model=, and
// dispatch= always embeds the full ID.
func TestResolveAgentOverrideAliasKeepsNonClaudeVerbatim(t *testing.T) {
	resolveAgentTestRepo(t, "project:\n  name: test\n")

	out, err := runResolveAgentCmd(t, "apply", "--provider", "codex", "--model", "gpt-5.3-codex", "--alias")
	if err != nil {
		t.Fatalf("resolve-agent with overrides --alias: %v", err)
	}
	want := "model=gpt-5.3-codex\nprovider=codex\ndispatch=codex exec -m gpt-5.3-codex\n"
	if out != want {
		t.Errorf("output = %q, want %q (non-Claude model verbatim; full ID in dispatch=)", out, want)
	}
}

// TestResolveAgentOverrideDispatchDisappearsOnNativeSwap: swapping TO a provider
// with no dispatch_command drops the dispatch= line — the QUERY reports the named
// provider's dispatch_command (or its absence), which is all this assertion covers.
// It is NOT an adapter move: `fab dispatch start` takes no override flags and
// re-resolves the stage from config, so only a config/tier override relocates a
// stage between native Agent-tool dispatch and CLI dispatch.
func TestResolveAgentOverrideDispatchDisappearsOnNativeSwap(t *testing.T) {
	resolveAgentTestRepo(t, `agent:
  tiers:
`+pinnedTierLine(t, agent.TierDoing, "codex"))

	// Baseline: the codex tier emits a dispatch= line.
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

// TestResolveCrossScopeCascadeLimitation PINS THE R7 DOCUMENTED LIMITATION
// (260805-j3cm, rework cycle 3) — it asserts CURRENT behavior, not desired
// behavior. The cross-provider cutoff computes ownership over the MERGED config:
// internal/config.LoadPath deep-merges the system layer (~/.fab-kit/config.yaml)
// and the project layer per-key BEFORE internal/agent resolves, so agent.ResolveTier
// sees ONE `agent.tiers.doing` map and cannot tell which scope contributed which
// key. When both scopes name DIFFERENT providers for the same tier, the merged
// tier's model/effort are attributed to the merged layer's `provider:` and the
// cutoff does not fire across the scope boundary — here a codex model ID rides a
// gemini invocation.
//
// This test exists so the limitation is reproducible and cannot change silently:
// if a follow-up change makes ownership cascade-aware (folding the per-scope layers
// in ResolveTier), this test SHOULD fail and be rewritten to the new — correct —
// expectation of an empty model refilled from providers.gemini. It lives in cmd/fab
// because this is the layer that can compose both scopes end-to-end (TestMain
// already isolates HOME for the package; this test points it at its own tree so it
// can WRITE a system config). Documented in internal/agent's ResolveTier comment
// (§ SCOPE OF OWNERSHIP) and docs/specs/stage-models.md.
func TestResolveCrossScopeCascadeLimitation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// System scope: a fully-pinned codex tier plus the codex fill.
	systemConfig := `providers:
  codex:
    model: gpt-5.3-codex
    effort: high
agent:
  tiers:
    doing: { provider: codex, model: gpt-5.3-codex, effort: high }
`
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte(systemConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project scope: switch the SAME tier to gemini, supplying only the provider.
	// resolveAgentTestRepo chdirs into the new repo; it does not touch HOME, so the
	// t.Setenv above stands.
	resolveAgentTestRepo(t, `project:
  name: test
providers:
  gemini:
    model: gemini-2.5-pro
agent:
  tiers:
    doing: { provider: gemini }
`)

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}

	// CURRENT (limitation) behavior: the system scope's codex model/effort survive
	// the project scope's switch to gemini, because the two scopes were merged into
	// one tier before resolution. The CORRECT behavior would be
	// model=gemini-2.5-pro with no effort= line (refilled from providers.gemini).
	want := "model=gpt-5.3-codex\neffort=high\nprovider=gemini\ndispatch=gemini -m gpt-5.3-codex\n"
	if out != want {
		t.Errorf("output = %q, want %q\n(this test PINS the documented cross-scope cascade limitation — "+
			"if ownership became cascade-aware, update this expectation to the refilled gemini values "+
			"and drop the limitation note from internal/agent's ResolveTier comment and stage-models.md)", out, want)
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
// true` inside tmux makes the built-in claude tier (session_command only, NO
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
	doingModel := wantTierModel(t, agent.TierDoing)
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
	want := wantTierBytes(t, agent.TierDoing)
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
	want := wantTierBytes(t, agent.TierDoing)
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
  tiers:
`+pinnedTierLine(t, agent.TierDoing, "codex"))
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	out, err := runResolveAgentCmd(t, "apply")
	if err != nil {
		t.Fatalf("resolve-agent apply: %v", err)
	}
	doingModel := wantTierModel(t, agent.TierDoing)
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
	doingModel := wantTierModel(t, agent.TierDoing)
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
	doingModel := wantTierModel(t, agent.TierDoing)
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
	want := wantTierBytes(t, agent.TierDoing)
	if out != want {
		t.Errorf("output = %q, want the three-line contract %q (project `false` must beat system `true`)", out, want)
	}
}
