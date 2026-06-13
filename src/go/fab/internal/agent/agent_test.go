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
		"intake":    {Model: "claude-opus-4-8", Effort: "xhigh"}, // thinking
		"review":    {Model: "claude-opus-4-8", Effort: "xhigh"}, // thinking
		"apply":     {Model: "claude-opus-4-8", Effort: "high"},  // doing
		"review-pr": {Model: "claude-opus-4-8", Effort: "high"},  // doing
		"hydrate":   {Model: "claude-opus-4-8", Effort: "high"},  // doing
		"ship":      {Model: "claude-sonnet-4-6", Effort: "low"}, // fast
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

// TestReviewVsReviewPrSplit: review (thinking) and review-pr (doing) must NOT be
// grouped — the cognitive-mode distinction is load-bearing.
func TestReviewVsReviewPrSplit(t *testing.T) {
	if tier, _ := TierForStage("review"); tier != TierThinking {
		t.Errorf("review tier = %q, want %q", tier, TierThinking)
	}
	if tier, _ := TierForStage("review-pr"); tier != TierDoing {
		t.Errorf("review-pr tier = %q, want %q", tier, TierDoing)
	}
}

// TestResolveFullOverride: an override sets both model and effort.
func TestResolveFullOverride(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Model: "claude-sonnet-4-6", Effort: "medium"},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{Model: "claude-sonnet-4-6", Effort: "medium"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v", got, want)
	}
}

// TestResolvePerFieldMerge: an override that sets only effort keeps the default
// model (per-field merge), and vice versa.
func TestResolvePerFieldMerge(t *testing.T) {
	// Only effort overridden → default model survives.
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Effort: "medium"},
	})
	got, err := Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "claude-opus-4-8" || got.Effort != "medium" {
		t.Errorf("Resolve(hydrate) = %+v, want default model + medium effort", got)
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

// TestResolveVerbatimNoValidation: a deliberately-incompatible override (Sonnet
// + xhigh, which Sonnet rejects at dispatch) is echoed verbatim with no error —
// fab does NOT validate or correct. The harness is the safety net.
func TestResolveVerbatimNoValidation(t *testing.T) {
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"fast": {Model: "claude-sonnet-4-6", Effort: "xhigh"},
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
		"thinking": {Model: "gpt-5", Effort: "reasoning_effort:high"},
	})
	got, err = Resolve(cfg, "intake")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "gpt-5" || got.Effort != "reasoning_effort:high" {
		t.Errorf("Resolve(intake) = %+v, want verbatim non-Claude profile", got)
	}
}

// TestResolveEmptyModelInherit: a tier overridden to an empty model is allowed —
// it signals "inherit". (An override entry present but with both fields empty is
// a no-op merge that keeps the default; an explicit empty model is only reachable
// when the user wants inherit, which is the documented signal.)
func TestResolveEmptyModelInherit(t *testing.T) {
	// Override present but empty → no-op merge, keeps default (the override has
	// nothing to contribute).
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {},
	})
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "claude-opus-4-8" || got.Effort != "high" {
		t.Errorf("Resolve(apply) with empty override = %+v, want default", got)
	}
}

// TestResolveUnknownStage: an unknown stage is the only resolution-side error.
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
		"claude-sonnet-4-6":          "sonnet",
		"claude-haiku-4-5":           "haiku",
		"claude-fable-1":             "fable",
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
}
