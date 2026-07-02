package agent

import (
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// cfgWithTiers builds a *config.Config with the given agent.tiers overrides.
func cfgWithTiers(tiers map[string]config.TierProfile) *config.Config {
	return &config.Config{Agent: config.AgentConfig{Tiers: tiers}}
}

// TestResolveDefaults: with no overrides, every stage resolves to its fixed
// tier's built-in default profile.
func TestResolveDefaults(t *testing.T) {
	cases := map[string]Profile{
		"intake":    {Provider: "claude", Model: "claude-fable-5", Effort: "xhigh"},  // default (advisory)
		"apply":     {Provider: "claude", Model: "claude-opus-4-8", Effort: "xhigh"}, // doing
		"review":    {Provider: "claude", Model: "claude-fable-5", Effort: "xhigh"},  // review
		"hydrate":   {Provider: "claude", Model: "claude-opus-4-8", Effort: "xhigh"}, // doing
		"ship":      {Provider: "claude", Model: "claude-sonnet-5", Effort: "low"},   // fast
		"review-pr": {Provider: "claude", Model: "claude-opus-4-8", Effort: "xhigh"}, // doing
	}
	for stage, want := range cases {
		t.Run(stage, func(t *testing.T) {
			got, err := Resolve(nil, stage)
			if err != nil {
				t.Fatalf("Resolve(%s): %v", stage, err)
			}
			if got != want {
				t.Errorf("Resolve(%s) = %+v, want %+v", stage, got, want)
			}
		})
	}
}

// TestReviewVsReviewPrSplit: review (its own tier) and review-pr (doing) must NOT
// be grouped — the author/critic distinction is load-bearing.
func TestReviewVsReviewPrSplit(t *testing.T) {
	if tier, _ := TierForStage("review"); tier != TierReview {
		t.Errorf("review tier = %q, want %q", tier, TierReview)
	}
	if tier, _ := TierForStage("review-pr"); tier != TierDoing {
		t.Errorf("review-pr tier = %q, want %q", tier, TierDoing)
	}
}

// TestResolveFullOverride: an override sets provider, model, and effort.
func TestResolveFullOverride(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v", got, want)
	}
}

// TestResolvePerFieldMerge: an override that sets only effort keeps the default
// provider+model (per-field merge), and vice versa.
func TestResolvePerFieldMerge(t *testing.T) {
	// Only effort overridden → default provider+model survive.
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Effort: "medium"},
	})
	got, err := Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Provider != "claude" || got.Model != "claude-opus-4-8" || got.Effort != "medium" {
		t.Errorf("Resolve(hydrate) = %+v, want default provider+model + medium effort", got)
	}

	// Only model overridden → default effort survives.
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"fast": {Model: "claude-haiku-4-5"},
	})
	got, err = Resolve(cfg, "ship")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "claude-haiku-4-5" || got.Effort != "low" {
		t.Errorf("Resolve(ship) = %+v, want overridden model + default low effort", got)
	}
}

// TestResolveDefaultTierInheritance: a field unset on both the requested tier's
// override AND its built-in inherits from the project's `default` tier. Here the
// project default tier sets a provider, and the doing override sets only effort;
// the resolved provider comes from the project default tier (which sits between
// the requested-tier override and the built-in in the merge cascade).
func TestResolveDefaultTierInheritance(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"default": {Provider: "codex"},
		"doing":   {Model: "gpt-5", Effort: "high"},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// provider inherits from the project `default` tier; model/effort from the
	// doing override.
	want := Profile{Provider: "codex", Model: "gpt-5", Effort: "high"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (provider inherited from default tier)", got, want)
	}
}

// TestResolveOverrideBeatsDefaultTier: a field set on the requested tier's
// override wins over the project `default` tier for that field.
func TestResolveOverrideBeatsDefaultTier(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"default": {Provider: "codex", Effort: "medium"},
		"doing":   {Provider: "claude", Model: "claude-opus-4-8"},
	})
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// provider from doing override (beats default tier's codex); model from doing
	// override; effort inherits from the default tier (doing did not set it).
	want := Profile{Provider: "claude", Model: "claude-opus-4-8", Effort: "medium"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v", got, want)
	}
}

// TestResolveVerbatimNoValidation: a deliberately-incompatible override (Sonnet +
// xhigh, which Sonnet rejects at dispatch) is echoed verbatim with no error — fab
// does NOT validate or correct. The harness is the safety net.
func TestResolveVerbatimNoValidation(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"fast": {Model: "claude-sonnet-5", Effort: "xhigh"},
	})
	got, err := Resolve(cfg, "ship")
	if err != nil {
		t.Fatalf("Resolve must not error on an incompatible pair: %v", err)
	}
	if got.Effort != "xhigh" {
		t.Errorf("effort = %q, want verbatim %q", got.Effort, "xhigh")
	}

	// A non-Claude provider's vocabulary passes through untouched too.
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"review": {Provider: "codex", Model: "gpt-5", Effort: "reasoning_effort:high"},
	})
	got, err = Resolve(cfg, "review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Provider != "codex" || got.Model != "gpt-5" || got.Effort != "reasoning_effort:high" {
		t.Errorf("Resolve(review) = %+v, want verbatim non-Claude profile", got)
	}
}

// TestResolveEmptyOverrideKeepsDefault: an override entry present but with all
// fields empty is a no-op merge that keeps the built-in default.
func TestResolveEmptyOverrideKeepsDefault(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {},
	})
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{Provider: "claude", Model: "claude-opus-4-8", Effort: "xhigh"}
	if got != want {
		t.Errorf("Resolve(apply) with empty override = %+v, want built-in default %+v", got, want)
	}
}

// TestResolveTier: a tier name resolves directly (the path fab agent / operator
// use), independent of any stage.
func TestResolveTier(t *testing.T) {
	got, err := ResolveTier(nil, TierOperator)
	if err != nil {
		t.Fatalf("ResolveTier(operator): %v", err)
	}
	want := Profile{Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"}
	if got != want {
		t.Errorf("ResolveTier(operator) = %+v, want %+v", got, want)
	}

	if _, err := ResolveTier(nil, "bogus"); err == nil {
		t.Fatal("expected an error for an unknown tier")
	}
}

// TestIsTierName: the five role-tier names report true; stage names and unknowns
// report false (disjoint sets — the resolve-agent positional-arg contract).
func TestIsTierName(t *testing.T) {
	for _, tier := range TierNames() {
		if !IsTierName(tier) {
			t.Errorf("IsTierName(%q) = false, want true", tier)
		}
	}
	for _, notTier := range []string{"apply", "review-pr", "ship", "frobnicate", ""} {
		if IsTierName(notTier) {
			t.Errorf("IsTierName(%q) = true, want false", notTier)
		}
	}
}

// TestResolveProvider: the built-in claude provider resolves with its default
// session command and no dispatch command; a project override per-field merges;
// an unknown provider reports ok=false.
func TestResolveProvider(t *testing.T) {
	// Built-in claude, no project config.
	prov, ok := ResolveProvider(nil, "claude")
	if !ok {
		t.Fatal("built-in claude provider must resolve")
	}
	if prov.SessionCommand != DefaultSessionCommand {
		t.Errorf("claude.SessionCommand = %q, want the built-in default", prov.SessionCommand)
	}
	if prov.DispatchCommand != "" {
		t.Errorf("claude.DispatchCommand = %q, want empty (native dispatch)", prov.DispatchCommand)
	}

	// Project override adds a dispatch_command; the session_command inherits the
	// built-in (per-field merge).
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"claude": {DispatchCommand: "claude -p"},
	}}
	prov, ok = ResolveProvider(cfg, "claude")
	if !ok {
		t.Fatal("claude provider must resolve with a project override")
	}
	if prov.SessionCommand != DefaultSessionCommand {
		t.Errorf("session_command = %q, want the inherited built-in", prov.SessionCommand)
	}
	if prov.DispatchCommand != "claude -p" {
		t.Errorf("dispatch_command = %q, want the override", prov.DispatchCommand)
	}

	// A project-only provider (not in the built-in table) resolves as known.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {SessionCommand: "codex", DispatchCommand: "codex exec"},
	}}
	prov, ok = ResolveProvider(cfg, "codex")
	if !ok || prov.DispatchCommand != "codex exec" {
		t.Errorf("codex provider = %+v, ok=%v, want the project entry", prov, ok)
	}

	// An unknown provider reports ok=false.
	if _, ok := ResolveProvider(nil, "gemini"); ok {
		t.Error("unknown provider must report ok=false")
	}
}

// TestResolveUnknownStage: an unknown stage is the only Resolve-side error.
func TestResolveUnknownStage(t *testing.T) {
	_, err := Resolve(nil, "frobnicate")
	if err == nil {
		t.Fatal("expected an error for an unknown stage")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error should name the unknown stage, got: %v", err)
	}
}

// TestModelAlias: full Claude IDs (incl. dated variants) map to their family
// alias by prefix; empty and unmapped/non-Claude inputs pass through verbatim.
func TestModelAlias(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-8":            "opus",
		"claude-sonnet-5":            "sonnet",
		"claude-haiku-4-5":           "haiku",
		"claude-fable-5":             "fable",
		"claude-haiku-4-5-20251001":  "haiku", // dated variant resolves by prefix
		"":                           "",      // empty in, empty out (inherit signal)
		"gpt-5":                      "gpt-5", // non-Claude passes through verbatim
		"some-unrecognized-model-id": "some-unrecognized-model-id",
	}
	for in, want := range cases {
		name := in
		if name == "" {
			name = "empty" // avoid an empty subtest name (TestModelAlias/)
		}
		t.Run(name, func(t *testing.T) {
			if got := ModelAlias(in); got != want {
				t.Errorf("ModelAlias(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

// TestTablesExhaustive: every stage's tier has a default profile, and the stage
// set is exactly the six pipeline stages.
func TestTablesExhaustive(t *testing.T) {
	for _, stage := range StageNames() {
		tier, _ := TierForStage(stage)
		if _, ok := DefaultTier(tier); !ok {
			t.Errorf("stage %q maps to tier %q which has no default profile", stage, tier)
		}
	}
	stages := strings.Join(StageNames(), ",")
	want := "apply,hydrate,intake,review,review-pr,ship"
	if stages != want {
		t.Errorf("stage set = %q, want %q", stages, want)
	}

	// The tier set is exactly the five role tiers.
	tiers := strings.Join(TierNames(), ",")
	wantTiers := "default,doing,fast,operator,review"
	if tiers != wantTiers {
		t.Errorf("tier set = %q, want %q", tiers, wantTiers)
	}
}
