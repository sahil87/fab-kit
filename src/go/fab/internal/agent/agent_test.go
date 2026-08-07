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
// tier's built-in default profile. The expectation is DERIVED from the canonical
// maps (stageTiers → defaultTiers) rather than restating model literals, so a
// model bump touches only defaultTiers in agent.go and this test keeps asserting
// what it is actually about: that resolution routes each stage to its fixed tier
// and returns that tier's profile unmodified. The literals themselves are pinned
// once, in TestDefaultTierProfilesArePinned below.
func TestResolveDefaults(t *testing.T) {
	for _, stage := range StageNames() {
		t.Run(stage, func(t *testing.T) {
			tier, ok := TierForStage(stage)
			if !ok {
				t.Fatalf("stage %q has no tier mapping", stage)
			}
			want, ok := DefaultTier(tier)
			if !ok {
				t.Fatalf("tier %q has no default profile", tier)
			}
			got, err := Resolve(nil, stage)
			if err != nil {
				t.Fatalf("Resolve(%s): %v", stage, err)
			}
			if got != want {
				t.Errorf("Resolve(%s) = %+v, want %s-tier default %+v", stage, got, tier, want)
			}
		})
	}
}

// TestDefaultTierProfilesArePinned is the ONE place the built-in default values
// are asserted in Go. It exists so a model bump is a deliberate two-line edit
// (defaults.yaml + this table) instead of an unreviewed change that silently
// repoints every stage — and so the other tests in this package can derive their
// expectations from the maps without any of them pinning the values.
//
// When you bump a default: edit defaults.yaml, then update this table to match.
// Every doc mirror is guarded separately by TestMirrorDocsMatchDefaultTiers /
// TestCLIFabReferenceListsDefaultTiers.
func TestDefaultTierProfilesArePinned(t *testing.T) {
	pinned := map[string]Profile{
		TierDefault:  {Provider: "claude", Model: "claude-fable-5", Effort: "high"},
		TierOperator: {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
		TierDoing:    {Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"},
		TierReview:   {Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"},
		TierHydrate:  {Provider: "claude", Model: "claude-opus-5", Effort: "high"},
		TierFast:     {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	}
	if len(pinned) != len(TierNames()) {
		t.Fatalf("pinned table covers %d tiers, but %d tiers exist — add the new tier here", len(pinned), len(TierNames()))
	}
	for _, tier := range TierNames() {
		want, ok := pinned[tier]
		if !ok {
			t.Errorf("tier %q has no pinned profile — add it to this table", tier)
			continue
		}
		if got, _ := DefaultTier(tier); got != want {
			t.Errorf("defaultTiers[%s] = %+v, pinned %+v — intentional bump? update this table too", tier, got, want)
		}
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
	// Only effort overridden → default provider+model survive. Derived from the
	// hydrate tier's default so a model bump does not touch this test.
	hydrateDefault, _ := DefaultTier(TierHydrate)
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"hydrate": {Effort: "medium"},
	})
	got, err := Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Provider != hydrateDefault.Provider || got.Model != hydrateDefault.Model || got.Effort != "medium" {
		t.Errorf("Resolve(hydrate) = %+v, want default provider+model (%s/%s) + medium effort", got, hydrateDefault.Provider, hydrateDefault.Model)
	}

	// Only model overridden → default effort survives. ship ∈ fast tier
	// (sonnet/medium), so overriding only the model keeps medium effort.
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"fast": {Model: "claude-haiku-4-5"},
	})
	got, err = Resolve(cfg, "ship")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "claude-haiku-4-5" || got.Effort != "medium" {
		t.Errorf("Resolve(ship) = %+v, want overridden model + default medium effort", got)
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

// TestResolveCrossProviderCutoff (260805-j3cm): a tier that explicitly names a
// provider OTHER than its built-in's does NOT inherit the built-in's (i.e. another
// provider's) model/effort — the unset fields fill from the named provider's
// default fill, then empty. This is the footgun fix: `{provider: codex}` used to
// resolve a CLAUDE model.
func TestResolveCrossProviderCutoff(t *testing.T) {
	// No fill configured for codex → empty model and effort (NOT the doing tier's
	// claude-opus-5/xhigh).
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Provider: "codex"},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != (Profile{Provider: "codex"}) {
		t.Errorf("Resolve(apply) = %+v, want {codex, \"\", \"\"} — a claude model must not leak across the provider switch", got)
	}

	// With a provider fill configured, the unset fields take it.
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"doing": {Provider: "codex"},
	})
	cfg.Providers = map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex", Effort: "high"},
	}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{Provider: "codex", Model: "gpt-5.3-codex", Effort: "high"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (provider default fill)", got, want)
	}

	// An explicit tier field BEATS the provider fill (precedence rung 2 > rung 3).
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"doing": {Provider: "codex", Model: "gpt-5.2-codex"},
	})
	cfg.Providers = map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex", Effort: "high"},
	}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want = Profile{Provider: "codex", Model: "gpt-5.2-codex", Effort: "high"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (tier model beats provider fill; effort from fill)", got, want)
	}

	// The cutoff triggers from the project `default` tier too (the same footgun one
	// layer up).
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"default": {Provider: "codex"},
	})
	cfg.Providers = map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex"},
	}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want = Profile{Provider: "codex", Model: "gpt-5.3-codex"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (default-tier provider switch also cuts inheritance)", got, want)
	}
}

// TestResolveCrossProviderCutoffAcrossLayers (260805-j3cm, rework cycle 2): the
// cutoff is anchored to the provider that SUPPLIED each inherited value, not to a
// flattened "the config set this field" bit. A model/effort written on the project
// `default` tier under one provider must not survive a cross-provider switch made
// on the requested tier — the exact combination the flattened implementation got
// wrong (it read a `default`-tier model as "explicitly set" and let a claude ID
// reach the codex CLI).
func TestResolveCrossProviderCutoffAcrossLayers(t *testing.T) {
	codexFill := map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex", Effort: "high"},
	}

	cases := []struct {
		name      string
		tiers     map[string]config.TierProfile
		providers map[string]config.ProviderConfig
		want      Profile
	}{
		{
			// The must-fix case: the `default` tier supplies CLAUDE-shaped values
			// (no provider named there, so their owner is the built-in's claude),
			// and the requested tier switches to codex. Both values are foreign →
			// both refill from the codex fill.
			name: "default-tier model+effort lose to the requested tier's provider switch",
			tiers: map[string]config.TierProfile{
				"default": {Model: "claude-fable-5", Effort: "medium"},
				"doing":   {Provider: "codex"},
			},
			providers: codexFill,
			want:      Profile{Provider: "codex", Model: "gpt-5.3-codex", Effort: "high"},
		},
		{
			// Same shape with NO fill configured → empty, never the claude values.
			name: "default-tier model+effort with no provider fill resolve empty",
			tiers: map[string]config.TierProfile{
				"default": {Model: "claude-fable-5", Effort: "medium"},
				"doing":   {Provider: "codex"},
			},
			want: Profile{Provider: "codex"},
		},
		{
			// The mirror: the switch happens on the `default` tier and the model
			// comes from the built-in tier profile. The built-in's model is owned by
			// the built-in's claude, so it is foreign to codex → refills.
			name: "default-tier provider switch cuts the built-in tier's model+effort",
			tiers: map[string]config.TierProfile{
				"default": {Provider: "codex"},
			},
			providers: codexFill,
			want:      Profile{Provider: "codex", Model: "gpt-5.3-codex", Effort: "high"},
		},
		{
			// A value written at the SAME layer as the switch (or at any layer at or
			// above it) is owned by the new provider and survives.
			name: "default-tier switch keeps a default-tier model written under it",
			tiers: map[string]config.TierProfile{
				"default": {Provider: "codex", Model: "gpt-5.2-codex"},
			},
			providers: codexFill,
			want:      Profile{Provider: "codex", Model: "gpt-5.2-codex", Effort: "high"},
		},
		{
			// The requested tier's own fields are written above the `default`-tier
			// switch, so they inherit its codex context and survive.
			name: "requested-tier fields written above a default-tier switch survive",
			tiers: map[string]config.TierProfile{
				"default": {Provider: "codex"},
				"doing":   {Model: "gpt-5.1-codex", Effort: "xhigh"},
			},
			providers: codexFill,
			want:      Profile{Provider: "codex", Model: "gpt-5.1-codex", Effort: "xhigh"},
		},
		{
			// Net provider EQUALS the built-in's, so there is no switch at all
			// (plan Assumption 2) — even though an intermediate layer named another
			// provider. Nothing is cut.
			name: "a net no-op provider chain cuts nothing",
			tiers: map[string]config.TierProfile{
				"default": {Provider: "codex", Effort: "medium"},
				"doing":   {Provider: "claude", Model: "claude-opus-4-8"},
			},
			providers: codexFill,
			want:      Profile{Provider: "claude", Model: "claude-opus-4-8", Effort: "medium"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := cfgWithTiers(c.tiers)
			cfg.Providers = c.providers
			got, err := Resolve(cfg, "apply") // apply ∈ doing
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != c.want {
				t.Errorf("Resolve(apply) = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestResolveNoCutoffWithoutProviderSwitch: the cutoff is scoped to an EXPLICIT
// cross-provider switch. A tier with no `provider:` inherits exactly as before, and
// an explicit `provider: claude` (equal to the built-in's) is not a switch at all —
// so the all-claude default world is byte-unchanged.
func TestResolveNoCutoffWithoutProviderSwitch(t *testing.T) {
	doingDefault, _ := DefaultTier(TierDoing)

	// No provider set → unchanged inheritance.
	cfg := cfgWithTiers(map[string]config.TierProfile{
		"doing": {Effort: "high"},
	})
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Provider != doingDefault.Provider || got.Model != doingDefault.Model || got.Effort != "high" {
		t.Errorf("Resolve(apply) = %+v, want the built-in provider+model with effort=high", got)
	}

	// Explicit provider EQUAL to the built-in's → not a switch, model survives.
	cfg = cfgWithTiers(map[string]config.TierProfile{
		"doing": {Provider: doingDefault.Provider},
	})
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != doingDefault {
		t.Errorf("Resolve(apply) = %+v, want the unchanged built-in %+v", got, doingDefault)
	}

	// And with no config at all, every stage still resolves to its built-in tier
	// profile (the byte-unchanged default world — also covered by
	// TestResolveDefaults; asserted here as the cutoff's regression guard).
	for _, stage := range StageNames() {
		tier, _ := TierForStage(stage)
		want, _ := DefaultTier(tier)
		got, err := Resolve(nil, stage)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", stage, err)
		}
		if got != want {
			t.Errorf("Resolve(%s) = %+v, want %+v — the cutoff must not perturb the default world", stage, got, want)
		}
	}
}

// TestApplyOverrides (260805-j3cm): the top rung of the fill precedence. Each flag
// applies only when SUPPLIED (its Set companion), a provider SWAP refills the
// unoverridden model/effort from the new provider (never retaining the old
// provider's), and swapping to the provider already resolved is not a swap.
func TestApplyOverrides(t *testing.T) {
	base := Profile{Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"}
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex", Effort: "high"},
	}}

	cases := []struct {
		name string
		o    Overrides
		want Profile
	}{
		{
			name: "no overrides is identity",
			o:    Overrides{},
			want: base,
		},
		{
			name: "model only (no --provider) is a within-tier override",
			o:    Overrides{Model: "claude-sonnet-5", ModelSet: true},
			want: Profile{Provider: "claude", Model: "claude-sonnet-5", Effort: "xhigh"},
		},
		{
			name: "effort only (no --provider) is a within-tier override",
			o:    Overrides{Effort: "medium", EffortSet: true},
			want: Profile{Provider: "claude", Model: "claude-opus-5", Effort: "medium"},
		},
		{
			name: "provider swap refills from the new provider's fill",
			o:    Overrides{Provider: "codex", ProviderSet: true},
			want: Profile{Provider: "codex", Model: "gpt-5.3-codex", Effort: "high"},
		},
		{
			name: "provider swap with no fill configured resolves empty",
			o:    Overrides{Provider: "gemini", ProviderSet: true},
			want: Profile{Provider: "gemini"},
		},
		{
			name: "explicit flags beat the swapped provider's fill",
			o: Overrides{
				Provider: "codex", ProviderSet: true,
				Model: "gpt-5.4-codex", ModelSet: true,
				Effort: "xhigh", EffortSet: true,
			},
			want: Profile{Provider: "codex", Model: "gpt-5.4-codex", Effort: "xhigh"},
		},
		{
			name: "swapping to the already-resolved provider is not a swap",
			o:    Overrides{Provider: "claude", ProviderSet: true},
			want: base,
		},
		{
			name: "an unknown provider is not an error here (lookup is the caller's)",
			o:    Overrides{Provider: "bogus", ProviderSet: true},
			want: Profile{Provider: "bogus"},
		},
		{
			name: "an explicitly-empty --model clears the model (supplied-ness, not emptiness)",
			o:    Overrides{ModelSet: true},
			want: Profile{Provider: "claude", Model: "", Effort: "xhigh"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ApplyOverrides(cfg, base, c.o); got != c.want {
				t.Errorf("ApplyOverrides = %+v, want %+v", got, c.want)
			}
		})
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
	want, _ := DefaultTier(TierDoing) // apply ∈ doing
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

// TestIsTierName: the six role-tier names report true; non-tier names (stages that
// are NOT also tiers, plus unknowns) report false. The resolve-agent positional-arg
// contract: a name shared by a stage and a tier (review, hydrate) IS a tier, so
// those are not in the not-a-tier list. "ship" is a STAGE but not a tier — it maps
// to the fast tier — so it stays in the not-a-tier list.
func TestIsTierName(t *testing.T) {
	for _, tier := range TierNames() {
		if !IsTierName(tier) {
			t.Errorf("IsTierName(%q) = false, want true", tier)
		}
	}
	// "hydrate" is a tier (added this change); "ship" is a stage that maps to the
	// fast tier, so it is NOT a tier name.
	for _, notTier := range []string{"apply", "review-pr", "intake", "ship", "frobnicate", ""} {
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

	// A project override of a non-claude built-in per-field merges too.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {DispatchCommand: "codex exec --json"},
	}}
	prov, ok = ResolveProvider(cfg, "codex")
	if !ok || prov.DispatchCommand != "codex exec --json" {
		t.Errorf("codex dispatch_command = %q, ok=%v, want the project override", prov.DispatchCommand, ok)
	}
	if prov.SessionCommand != DefaultCodexSessionCommand {
		t.Errorf("codex session_command = %q, want the inherited built-in", prov.SessionCommand)
	}

	// A project-only provider (in neither the built-in table nor a tier) resolves
	// as known off the project entry alone.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"myagent": {SessionCommand: "myagent", DispatchCommand: "myagent run"},
	}}
	prov, ok = ResolveProvider(cfg, "myagent")
	if !ok || prov.DispatchCommand != "myagent run" {
		t.Errorf("myagent provider = %+v, ok=%v, want the project entry", prov, ok)
	}

	// An unknown provider reports ok=false.
	if _, ok := ResolveProvider(nil, "bogus"); ok {
		t.Error("unknown provider must report ok=false")
	}
}

// TestResolveProvider_BuiltInCodexAndGemini: codex and gemini are BUILT-IN
// providers (260805-j3cm) — resolvable with NO providers: config at all — and are
// GRAMMAR ONLY: both command fields present, neither model nor effort fill.
func TestResolveProvider_BuiltInCodexAndGemini(t *testing.T) {
	cases := []struct {
		name              string
		session, dispatch string
	}{
		{"codex", DefaultCodexSessionCommand, DefaultCodexDispatchCommand},
		{"gemini", DefaultGeminiSessionCommand, DefaultGeminiDispatchCommand},
	}
	for _, c := range cases {
		prov, ok := ResolveProvider(nil, c.name)
		if !ok {
			t.Fatalf("built-in %s provider must resolve with no config", c.name)
		}
		if prov.SessionCommand != c.session {
			t.Errorf("%s.SessionCommand = %q, want %q", c.name, prov.SessionCommand, c.session)
		}
		if prov.DispatchCommand != c.dispatch {
			t.Errorf("%s.DispatchCommand = %q, want %q (a non-claude built-in carries one, so naming it flips the stage to CLI dispatch)", c.name, prov.DispatchCommand, c.dispatch)
		}
		if prov.Model != "" || prov.Effort != "" {
			t.Errorf("%s built-in must carry NO fill values (model=%q effort=%q) — model IDs rot at CLI cadence", c.name, prov.Model, prov.Effort)
		}
	}

	// The gemini grammar deliberately omits {effort} (no reasoning-effort flag) and
	// -p (fab dispatch pipes the prompt to stdin; -p takes prompt TEXT).
	if strings.Contains(DefaultGeminiSessionCommand, "{effort}") || strings.Contains(DefaultGeminiDispatchCommand, "{effort}") {
		t.Error("the gemini built-in must carry no {effort} placeholder (the gemini CLI has no reasoning-effort flag)")
	}
	if strings.Contains(DefaultGeminiDispatchCommand, "-p") {
		t.Error("the gemini dispatch command must not carry -p (fab dispatch pipes the prompt to stdin)")
	}
}

// TestResolveProvider_FillFieldsMerge: the model/effort default-fill fields
// (260805-j3cm) per-field merge over the built-in exactly as the commands do — so a
// config supplying only fill inherits the built-in grammar.
func TestResolveProvider_FillFieldsMerge(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {Model: "gpt-5.3-codex", Effort: "high"},
	}}
	prov, ok := ResolveProvider(cfg, "codex")
	if !ok {
		t.Fatal("codex must resolve")
	}
	if prov.Model != "gpt-5.3-codex" || prov.Effort != "high" {
		t.Errorf("fill = {%q, %q}, want the configured {gpt-5.3-codex, high}", prov.Model, prov.Effort)
	}
	if prov.SessionCommand != DefaultCodexSessionCommand || prov.DispatchCommand != DefaultCodexDispatchCommand {
		t.Errorf("commands = {%q, %q}, want the inherited built-in grammar", prov.SessionCommand, prov.DispatchCommand)
	}

	// A partial fill leaves the other field empty (no cross-field invention).
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"gemini": {Model: "gemini-2.5-pro"},
	}}
	prov, _ = ResolveProvider(cfg, "gemini")
	if prov.Model != "gemini-2.5-pro" || prov.Effort != "" {
		t.Errorf("partial fill = {%q, %q}, want {gemini-2.5-pro, \"\"}", prov.Model, prov.Effort)
	}
}

// TestProviderNames: the resolvable provider set is the union of fab-kit's
// built-in table and the project's providers: block, sorted and de-duplicated —
// the set `fab agent --provider <unknown>` names in its lookup-failure error.
func TestProviderNames(t *testing.T) {
	// No project config → the built-in table alone: three built-in providers
	// (260805-j3cm), sorted.
	assertNames := func(t *testing.T, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("ProviderNames = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("ProviderNames = %v, want %v (sorted, de-duplicated)", got, want)
			}
		}
	}
	assertNames(t, ProviderNames(nil), []string{"claude", "codex", "gemini"})

	// Project providers union the built-in, de-duplicating shared keys and sorting
	// for stable error output.
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex":   {Model: "gpt-5.3-codex"},
		"claude":  {DispatchCommand: "claude -p"},
		"myagent": {SessionCommand: "myagent"},
	}}
	assertNames(t, ProviderNames(cfg), []string{"claude", "codex", "gemini", "myagent"})
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

	// The tier set is exactly the six role tiers.
	tiers := strings.Join(TierNames(), ",")
	wantTiers := "default,doing,fast,hydrate,operator,review"
	if tiers != wantTiers {
		t.Errorf("tier set = %q, want %q", tiers, wantTiers)
	}
}

// TestStageTierCollisionsAreFixedPoints: every name shared by the stage set and
// the tier set (review, hydrate) must be a FIXED POINT — the stage maps to the
// same-named tier (stageTiers[name] == name). This is what makes the tier-first
// resolution order in cmd/fab.resolveStageOrTier immaterial for those names: a
// shared name resolves identically whether read as a stage or a tier. It guards
// that order from ever silently changing a stage's resolution. (ship is a stage
// but NOT a tier — it maps to fast — so it is not a collision.)
func TestStageTierCollisionsAreFixedPoints(t *testing.T) {
	tierSet := make(map[string]bool)
	for _, tier := range TierNames() {
		tierSet[tier] = true
	}
	collisions := 0
	for _, stage := range StageNames() {
		if !tierSet[stage] {
			continue // not a shared name
		}
		collisions++
		tier, ok := TierForStage(stage)
		if !ok {
			t.Errorf("stage %q has no tier mapping", stage)
			continue
		}
		if tier != stage {
			t.Errorf("stage/tier name collision %q is NOT a fixed point: stageTiers[%q] = %q, want %q "+
				"(a name shared by a stage and a tier must map the stage to the same-named tier, "+
				"or the tier-first resolve order would change the stage's resolution)", stage, stage, tier, stage)
		}
	}
	// Guard the guard: the intended collisions (review, hydrate) must exist —
	// a zero-collision result would mean this test silently checks nothing.
	if collisions == 0 {
		t.Fatal("expected at least one stage/tier name collision (review, hydrate); found none")
	}
}
