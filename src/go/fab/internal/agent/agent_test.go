package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"gopkg.in/yaml.v3"
)

// cfgWithProfiles builds a *config.Config with the given agent.profiles overrides.
func cfgWithProfiles(profiles map[string]config.RoleProfile) *config.Config {
	return &config.Config{Agent: config.AgentConfig{Profiles: profiles}}
}

// TestResolveDefaults: with no overrides, every stage resolves to its fixed
// role's built-in default profile. The expectation is DERIVED from the canonical
// maps (stageRoles → DefaultProfile) rather than restating model literals, so a
// model bump touches only defaults.yaml and this test keeps asserting what it is
// actually about: that resolution routes each stage to its fixed role and returns
// that role's profile unmodified. The literals themselves are pinned once, in
// TestDefaultRoleProfilesArePinned below.
func TestResolveDefaults(t *testing.T) {
	for _, stage := range StageNames() {
		t.Run(stage, func(t *testing.T) {
			role, ok := RoleForStage(stage)
			if !ok {
				t.Fatalf("stage %q has no role mapping", stage)
			}
			want, ok := DefaultProfile(role)
			if !ok {
				t.Fatalf("role %q has no default profile", role)
			}
			got, err := Resolve(nil, stage)
			if err != nil {
				t.Fatalf("Resolve(%s): %v", stage, err)
			}
			if got != want {
				t.Errorf("Resolve(%s) = %+v, want %s-role default %+v", stage, got, role, want)
			}
		})
	}
}

// TestDefaultRoleProfilesArePinned is the ONE place the built-in default values
// are asserted in Go. It exists so a model bump is a deliberate two-line edit
// (defaults.yaml + this table) instead of an unreviewed change that silently
// repoints every stage — and so the other tests in this package can derive their
// expectations from the maps without any of them pinning the values.
//
// These are ALSO the byte-identical-defaults guard for the 260806-j9nh reshape:
// the six values moved from agent.tiers onto providers.claude.profiles, and every
// resolved stage profile must be unchanged by that move.
//
// When you bump a default: edit defaults.yaml, then update this table to match.
// Every doc mirror is guarded separately by TestMirrorDocsMatchDefaultProfiles /
// TestCLIFabReferenceListsDefaultRoles.
func TestDefaultRoleProfilesArePinned(t *testing.T) {
	pinned := map[string]Profile{
		RoleDefault:  {Provider: "claude", Model: "claude-fable-5", Effort: "high"},
		RoleOperator: {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
		RoleDoing:    {Provider: "claude", Model: "claude-opus-5", Effort: "high"},
		RoleReview:   {Provider: "claude", Model: "claude-opus-5", Effort: "high"},
		RoleHydrate:  {Provider: "claude", Model: "claude-opus-5", Effort: "high"},
		RoleFast:     {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	}
	if len(pinned) != len(RoleNames()) {
		t.Fatalf("pinned table covers %d roles, but %d roles exist — add the new role here", len(pinned), len(RoleNames()))
	}
	for _, role := range RoleNames() {
		want, ok := pinned[role]
		if !ok {
			t.Errorf("role %q has no pinned profile — add it to this table", role)
			continue
		}
		if got, _ := DefaultProfile(role); got != want {
			t.Errorf("DefaultProfile(%s) = %+v, pinned %+v — intentional bump? update this table too", role, got, want)
		}
	}
}

// TestNonClaudeProviderFillsArePinned is the deliberate-change pin for the codex
// and gemini fills fab-kit ships (260806-ywkx). It is the non-claude sibling of
// TestDefaultRoleProfilesArePinned: those fills never reach DefaultProfile (which
// resolves through the claude-defaulted depth knobs), so without this table a
// catalog bump would land unreviewed.
//
// The tables are SPARSE, exactly as defaults.yaml is — a role absent here must be
// absent there, so the cross-role fallback through the provider's `default` entry
// stays the shipped shape rather than an accident of enumeration.
//
// When you bump a fill: edit defaults.yaml, then update this table to match. The
// doc mirrors are guarded separately by TestMirrorDocsMatchDefaultProfiles.
func TestNonClaudeProviderFillsArePinned(t *testing.T) {
	pinned := map[string]map[string]config.ProviderProfile{
		providerCodex: {
			RoleDefault: {Model: "gpt-5.6-sol", Effort: "high"},
			RoleDoing:   {Effort: "xhigh"},
			RoleReview:  {Effort: "xhigh"},
			RoleFast:    {Model: "gpt-5.6-luna", Effort: "low"},
		},
		// Model-only: the gemini CLI has no reasoning-effort flag. The values are
		// that CLI's own stable ALIASES rather than versioned IDs, so they track
		// its current best pro/flash model without a bump.
		providerGemini: {
			RoleDefault: {Model: "pro"},
			RoleFast:    {Model: "flash"},
		},
	}
	for name, want := range pinned {
		prov, ok := ResolveProvider(nil, name)
		if !ok {
			t.Errorf("built-in provider %q must resolve", name)
			continue
		}
		if !reflect.DeepEqual(prov.Profiles, want) {
			t.Errorf("built-in %s fills = %+v, pinned %+v — intentional bump? update this table too", name, prov.Profiles, want)
		}
	}
}

// TestReviewVsReviewPrSplit: review (its own role) and review-pr (doing) must NOT
// be grouped — the author/critic distinction is load-bearing.
func TestReviewVsReviewPrSplit(t *testing.T) {
	if role, _ := RoleForStage("review"); role != RoleReview {
		t.Errorf("review role = %q, want %q", role, RoleReview)
	}
	if role, _ := RoleForStage("review-pr"); role != RoleDoing {
		t.Errorf("review-pr role = %q, want %q", role, RoleDoing)
	}
}

// TestRoleDepthPartition pins the fab-owned role→depth partition: default and
// operator are Tier-1 (session) roles, the other four are Tier-2 (workers). The
// partition is what the two advertised knobs select on, so a silent reshuffle here
// would silently move a role onto the other knob.
func TestRoleDepthPartition(t *testing.T) {
	want := map[string]depth{
		RoleDefault:  depthSession,
		RoleOperator: depthSession,
		RoleDoing:    depthWorkers,
		RoleReview:   depthWorkers,
		RoleHydrate:  depthWorkers,
		RoleFast:     depthWorkers,
	}
	if len(roleDepth) != len(want) {
		t.Fatalf("roleDepth covers %d roles, want %d", len(roleDepth), len(want))
	}
	for role, wantDepth := range want {
		if got, ok := roleDepth[role]; !ok || got != wantDepth {
			t.Errorf("roleDepth[%s] = %v (ok=%v), want %v", role, got, ok, wantDepth)
		}
	}
}

// TestDepthKnobSelectsProvider is the headline behavior: `agent.workers: <name>`
// moves exactly the four Tier-2 roles and leaves the two session roles alone, and
// vice versa. Neither knob touches the other's partition.
func TestDepthKnobSelectsProvider(t *testing.T) {
	workers := &config.Config{Agent: config.AgentConfig{Workers: "gemini"}}
	for _, role := range RoleNames() {
		want := "claude"
		if roleDepth[role] == depthWorkers {
			want = "gemini"
		}
		got, err := ResolveRole(workers, role)
		if err != nil {
			t.Fatalf("ResolveRole(%s): %v", role, err)
		}
		if got.Provider != want {
			t.Errorf("with agent.workers=gemini, role %q resolved provider %q, want %q", role, got.Provider, want)
		}
	}

	session := &config.Config{Agent: config.AgentConfig{Session: "codex"}}
	for _, role := range RoleNames() {
		want := "claude"
		if roleDepth[role] == depthSession {
			want = "codex"
		}
		got, err := ResolveRole(session, role)
		if err != nil {
			t.Fatalf("ResolveRole(%s): %v", role, err)
		}
		if got.Provider != want {
			t.Errorf("with agent.session=codex, role %q resolved provider %q, want %q", role, got.Provider, want)
		}
	}
}

// TestWorkersKnobResolvesBuiltInFills is the flagship UX (260806-ywkx): ONE knob
// line moves every Tier-2 stage onto another provider AND gives each role that
// provider's own per-role model — role differentiation survives the swap instead of
// collapsing to an empty model for all four. The Tier-1 roles stay on claude.
//
// Expectations are DERIVED from ResolveProvider (with its cross-role fallback
// through the provider's `default` entry), never restated, so a fill bump touches
// defaults.yaml and TestNonClaudeProviderFillsArePinned only.
func TestWorkersKnobResolvesBuiltInFills(t *testing.T) {
	for _, provider := range []string{providerCodex, providerGemini} {
		t.Run(provider, func(t *testing.T) {
			cfg := &config.Config{Agent: config.AgentConfig{Workers: provider}}
			prov, ok := ResolveProvider(nil, provider)
			if !ok {
				t.Fatalf("built-in %s must resolve", provider)
			}

			for _, stage := range StageNames() {
				role, _ := RoleForStage(stage)
				got, err := Resolve(cfg, stage)
				if err != nil {
					t.Fatalf("Resolve(%s): %v", stage, err)
				}

				if roleDepth[role] == depthSession {
					want, _ := DefaultProfile(role)
					if got != want {
						t.Errorf("Resolve(%s) = %+v, want the untouched Tier-1 claude default %+v — agent.workers must not move a session role", stage, got, want)
					}
					continue
				}

				fill := prov.Profiles[role]
				want := Profile{
					Provider: provider,
					Model:    firstNonEmpty(fill.Model, prov.Profiles[RoleDefault].Model),
					Effort:   firstNonEmpty(fill.Effort, prov.Profiles[RoleDefault].Effort),
				}
				if got != want {
					t.Errorf("Resolve(%s) = %+v, want %s's %s fill %+v", stage, got, provider, role, want)
				}
				if got.Model == "" {
					t.Errorf("Resolve(%s) resolved an EMPTY model — a knob pointed at %s must resolve a real model for every role", stage, provider)
				}
				if provider == providerGemini && got.Effort != "" {
					t.Errorf("Resolve(%s) = %+v, want no effort — the gemini CLI has no reasoning-effort flag", stage, got)
				}
			}
		})
	}

	// The headline differentiation: the two heavyweight roles and the cheap one do
	// NOT resolve the same profile.
	cfg := &config.Config{Agent: config.AgentConfig{Workers: providerCodex}}
	apply, _ := Resolve(cfg, "apply") // doing
	ship, _ := Resolve(cfg, "ship")   // fast
	if apply == ship {
		t.Errorf("apply and ship both resolved %+v — shipping fills exists precisely so the roles differ", apply)
	}
}

// TestResolveFullOverride: an agent.profiles entry sets provider, model, and effort.
func TestResolveFullOverride(t *testing.T) {
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
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

// TestResolvePerFieldMerge: an override that sets only effort keeps the provider's
// own per-role model fill (per-field), and vice versa.
func TestResolvePerFieldMerge(t *testing.T) {
	// Only effort overridden → the provider's hydrate fill supplies provider+model.
	hydrateDefault, _ := DefaultProfile(RoleHydrate)
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
		"hydrate": {Effort: "medium"},
	})
	got, err := Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Provider != hydrateDefault.Provider || got.Model != hydrateDefault.Model || got.Effort != "medium" {
		t.Errorf("Resolve(hydrate) = %+v, want default provider+model (%s/%s) + medium effort", got, hydrateDefault.Provider, hydrateDefault.Model)
	}

	// Only model overridden → the fast fill's effort survives. ship ∈ fast
	// (sonnet/medium), so overriding only the model keeps medium effort.
	cfg = cfgWithProfiles(map[string]config.RoleProfile{
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

// TestNoAgentSideDefaultRoleInheritance (260806-j9nh): agent.profiles.default is
// the `default` ROLE's own override, NOT a fallback source for the other five. The
// old agent.tiers map re-based every unset field from its `default` tier; that
// second fallback chain is deleted, leaving only the provider-side one.
func TestNoAgentSideDefaultRoleInheritance(t *testing.T) {
	doingDefault, _ := DefaultProfile(RoleDoing)

	cfg := cfgWithProfiles(map[string]config.RoleProfile{
		"default": {Model: "claude-fable-5", Effort: "low"},
		"doing":   {},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != doingDefault {
		t.Errorf("Resolve(apply) = %+v, want the doing role's own %+v — agent.profiles.default must not re-base another role", got, doingDefault)
	}

	// It DOES still govern the default role itself.
	got, err = ResolveRole(cfg, RoleDefault)
	if err != nil {
		t.Fatalf("ResolveRole(default): %v", err)
	}
	if got.Model != "claude-fable-5" || got.Effort != "low" {
		t.Errorf("ResolveRole(default) = %+v, want the configured fable-5/low", got)
	}

	// And a provider named on the default role does not leak onto another role
	// either — the other role falls to its depth knob.
	cfg = cfgWithProfiles(map[string]config.RoleProfile{
		"default": {Provider: "codex"},
	})
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != doingDefault {
		t.Errorf("Resolve(apply) = %+v, want %+v — a default-role provider must not re-base doing", got, doingDefault)
	}
}

// TestProviderPerRoleFills: a provider's profiles map supplies model/effort per
// role, with its `default` entry acting as the cross-role fallback and an explicit
// agent.profiles field beating both.
//
// It uses a provider name fab-kit does NOT ship, so the per-role/per-field merge
// with a built-in table is out of the picture and the precedence under test is the
// only thing the expectations depend on (TestPartialProviderFillMergesOverBuiltIn
// covers the merge itself).
func TestProviderPerRoleFills(t *testing.T) {
	myagent := map[string]config.ProviderConfig{
		"myagent": {Profiles: map[string]config.ProviderProfile{
			"default": {Model: "gpt-5.3-codex", Effort: "medium"},
			"doing":   {Model: "gpt-5.3-codex-max", Effort: "high"},
		}},
	}

	cases := []struct {
		name      string
		agentCfg  config.AgentConfig
		providers map[string]config.ProviderConfig
		stage     string
		want      Profile
	}{
		{
			name:      "the role's own fill wins",
			agentCfg:  config.AgentConfig{Workers: "myagent"},
			providers: myagent,
			stage:     "apply", // doing
			want:      Profile{Provider: "myagent", Model: "gpt-5.3-codex-max", Effort: "high"},
		},
		{
			name:      "a role with no fill falls to the provider's default entry",
			agentCfg:  config.AgentConfig{Workers: "myagent"},
			providers: myagent,
			stage:     "review",
			want:      Profile{Provider: "myagent", Model: "gpt-5.3-codex", Effort: "medium"},
		},
		{
			name: "an explicit agent.profiles field beats the provider fill",
			agentCfg: config.AgentConfig{
				Workers:  "myagent",
				Profiles: map[string]config.RoleProfile{"doing": {Model: "gpt-5.2-codex"}},
			},
			providers: myagent,
			stage:     "apply",
			want:      Profile{Provider: "myagent", Model: "gpt-5.2-codex", Effort: "high"},
		},
		{
			name: "a per-role provider: override picks that provider's fills",
			agentCfg: config.AgentConfig{
				Profiles: map[string]config.RoleProfile{"review": {Provider: "myagent"}},
			},
			providers: myagent,
			stage:     "review",
			want:      Profile{Provider: "myagent", Model: "gpt-5.3-codex", Effort: "medium"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &config.Config{Agent: c.agentCfg, Providers: c.providers}
			got, err := Resolve(cfg, c.stage)
			if err != nil {
				t.Fatalf("Resolve(%s): %v", c.stage, err)
			}
			if got != c.want {
				t.Errorf("Resolve(%s) = %+v, want %+v", c.stage, got, c.want)
			}
		})
	}
}

// TestPartialProviderFillMergesOverBuiltIn: overriding ONE role's fill on the
// built-in claude provider leaves the other five (and the overridden role's other
// field) on the shipped values — the per-role, per-field merge.
func TestPartialProviderFillMergesOverBuiltIn(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"claude": {Profiles: map[string]config.ProviderProfile{
			"review": {Model: "claude-fable-5"},
		}},
	}}

	got, err := Resolve(cfg, "review")
	if err != nil {
		t.Fatalf("Resolve(review): %v", err)
	}
	reviewDefault, _ := DefaultProfile(RoleReview)
	if got.Model != "claude-fable-5" || got.Effort != reviewDefault.Effort {
		t.Errorf("Resolve(review) = %+v, want the overridden model with the built-in effort %q", got, reviewDefault.Effort)
	}

	// The untouched roles keep the built-in fills.
	for _, stage := range []string{"apply", "hydrate", "ship"} {
		role, _ := RoleForStage(stage)
		want, _ := DefaultProfile(role)
		got, err := Resolve(cfg, stage)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", stage, err)
		}
		if got != want {
			t.Errorf("Resolve(%s) = %+v, want the untouched built-in %+v", stage, got, want)
		}
	}
}

// TestResolveProviderDoesNotMutateBuiltIns: the resolved Profiles map is a fresh
// copy, so a caller writing through it cannot poison the shipped defaults for the
// rest of the process.
func TestResolveProviderDoesNotMutateBuiltIns(t *testing.T) {
	prov, ok := ResolveProvider(nil, "claude")
	if !ok {
		t.Fatal("built-in claude must resolve")
	}
	prov.Profiles[RoleDoing] = config.ProviderProfile{Model: "poisoned"}

	want, _ := DefaultProfile(RoleDoing)
	if want.Model == "poisoned" {
		t.Fatal("mutating a resolved provider's Profiles map corrupted the built-in table")
	}
}

// TestLegacyAgentTiersAlias (260806-j9nh): a config still carrying the pre-2.17.0
// `agent.tiers:` spelling keeps resolving, PER ROLE — so a half-migrated config
// (some roles moved to profiles, some not) resolves every role — and agent.profiles
// wins whenever it carries the role.
func TestLegacyAgentTiersAlias(t *testing.T) {
	cfg := &config.Config{Agent: config.AgentConfig{
		Tiers: map[string]config.RoleProfile{
			"doing":   {Effort: "medium"},
			"hydrate": {Model: "claude-fable-5"},
		},
	}}
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Effort != "medium" {
		t.Errorf("Resolve(apply) = %+v, want the legacy agent.tiers effort to still resolve", got)
	}

	// profiles wins over tiers for a role present in both; tiers still covers the
	// roles profiles does not name.
	cfg.Agent.Profiles = map[string]config.RoleProfile{"doing": {Effort: "low"}}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Effort != "low" {
		t.Errorf("Resolve(apply) effort = %q, want the agent.profiles value to win over agent.tiers", got.Effort)
	}
	got, err = Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "claude-fable-5" {
		t.Errorf("Resolve(hydrate) model = %q, want the legacy agent.tiers value (profiles does not name hydrate)", got.Model)
	}
}

// TestLegacyFlatProviderFill (260806-j9nh): the pre-2.17.0 flat
// providers.<name>.model/.effort is still read, as an alias for
// profiles.default — so a config that has not yet run the migration keeps
// resolving. profiles.default wins when both are present.
func TestLegacyFlatProviderFill(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{Workers: "myagent"},
		Providers: map[string]config.ProviderConfig{
			"myagent": {Model: "gpt-5.3-codex", Effort: "high"},
		},
	}
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{Provider: "myagent", Model: "gpt-5.3-codex", Effort: "high"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (legacy flat fill still read)", got, want)
	}

	// profiles.default is the modern spelling and outranks it.
	cfg.Providers["myagent"] = config.ProviderConfig{
		Model:    "gpt-5.3-codex",
		Profiles: map[string]config.ProviderProfile{"default": {Model: "gpt-5.4-codex"}},
	}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "gpt-5.4-codex" {
		t.Errorf("Resolve(apply) model = %q, want profiles.default to beat the legacy flat fill", got.Model)
	}
}

// TestFlatProviderFillBeatsBuiltInDefault (260806-ywkx) is the regression guard for
// shipping non-claude fills: the flat spelling is an ALIAS for the user's own
// profiles.default, so it must outrank the BUILT-IN profiles.default it is trying to
// replace. Were it read as a rung below profiles.default instead — its shape before
// fab-kit shipped a non-claude default — a pre-migration config's pinned model would
// be silently shadowed by fab-kit's shipped one.
func TestFlatProviderFillBeatsBuiltInDefault(t *testing.T) {
	builtin, _ := ResolveProvider(nil, providerCodex)
	if builtin.Profiles[RoleDefault].Model == "" {
		t.Fatal("this guard is meaningless unless the codex built-in ships a profiles.default")
	}

	// A pre-migration config carrying ONLY the flat spelling.
	cfg := &config.Config{
		Agent:     config.AgentConfig{Workers: providerCodex},
		Providers: map[string]config.ProviderConfig{providerCodex: {Model: "my-pinned-model"}},
	}
	got, err := Resolve(cfg, "hydrate") // hydrate has no codex fill of its own → the default entry
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "my-pinned-model" {
		t.Errorf("Resolve(hydrate) model = %q, want the user's flat pin — a built-in profiles.default must not shadow it", got.Model)
	}
	// The effort the user did NOT pin still comes from the built-in fill (per-field).
	if got.Effort != builtin.Profiles[RoleDefault].Effort {
		t.Errorf("Resolve(hydrate) effort = %q, want the built-in %q (the flat fill pinned only the model)", got.Effort, builtin.Profiles[RoleDefault].Effort)
	}

	// The user's OWN profiles.default still beats their flat fill — the modern
	// spelling wins over its alias, exactly as TestLegacyFlatProviderFill asserts
	// for a provider fab-kit ships nothing for.
	cfg.Providers[providerCodex] = config.ProviderConfig{
		Model:    "my-pinned-model",
		Profiles: map[string]config.ProviderProfile{RoleDefault: {Model: "my-modern-model"}},
	}
	got, err = Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "my-modern-model" {
		t.Errorf("Resolve(hydrate) model = %q, want profiles.default to beat the flat alias", got.Model)
	}
}

// TestResolveRoleWithOverrides: the top rung of the fill precedence. Each flag
// applies only when SUPPLIED (its Set companion); a provider SWAP re-derives the
// unoverridden model/effort from the NEW provider's own per-role fills (never
// retaining the old provider's); and an explicit agent.profiles pin survives a swap,
// because a value the user wrote is not inheritance.
func TestResolveRoleWithOverrides(t *testing.T) {
	base, _ := DefaultProfile(RoleDoing)
	gemini, _ := ResolveProvider(nil, providerGemini)
	geminiDoing := Profile{
		Provider: providerGemini,
		// gemini ships no `doing` fill, so the role falls through to its `default`
		// entry — derived, not restated, so a fill bump does not touch this table.
		Model: gemini.Profiles[RoleDefault].Model,
	}
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {Profiles: map[string]config.ProviderProfile{
			"doing": {Model: "gpt-5.3-codex", Effort: "high"},
		}},
	}}

	cases := []struct {
		name string
		cfg  *config.Config
		o    Overrides
		want Profile
	}{
		{
			name: "no overrides is identity",
			cfg:  cfg,
			o:    Overrides{},
			want: base,
		},
		{
			name: "model only (no --provider) is a within-role override",
			cfg:  cfg,
			o:    Overrides{Model: "claude-sonnet-5", ModelSet: true},
			want: Profile{Provider: "claude", Model: "claude-sonnet-5", Effort: base.Effort},
		},
		{
			name: "effort only (no --provider) is a within-role override",
			cfg:  cfg,
			o:    Overrides{Effort: "medium", EffortSet: true},
			want: Profile{Provider: "claude", Model: base.Model, Effort: "medium"},
		},
		{
			name: "provider swap re-derives from the new provider's per-role fill",
			cfg:  cfg,
			o:    Overrides{Provider: "codex", ProviderSet: true},
			want: Profile{Provider: "codex", Model: "gpt-5.3-codex", Effort: "high"},
		},
		{
			// gemini carries no `doing` fill of its own, so the swap lands on that
			// provider's `default` entry — never on claude's model, and (since
			// 260806-ywkx ships the fills) never on an empty one.
			name: "provider swap with no per-role fill falls to that provider's default entry",
			cfg:  cfg,
			o:    Overrides{Provider: "gemini", ProviderSet: true},
			want: geminiDoing,
		},
		{
			name: "explicit flags beat the swapped provider's fill",
			cfg:  cfg,
			o: Overrides{
				Provider: "codex", ProviderSet: true,
				Model: "gpt-5.4-codex", ModelSet: true,
				Effort: "xhigh", EffortSet: true,
			},
			want: Profile{Provider: "codex", Model: "gpt-5.4-codex", Effort: "xhigh"},
		},
		{
			name: "swapping to the already-resolved provider changes nothing",
			cfg:  cfg,
			o:    Overrides{Provider: "claude", ProviderSet: true},
			want: base,
		},
		{
			name: "an unknown provider is not an error here (lookup is the caller's)",
			cfg:  cfg,
			o:    Overrides{Provider: "bogus", ProviderSet: true},
			want: Profile{Provider: "bogus"},
		},
		{
			name: "an explicitly-empty --provider resolves empty, never the depth knob",
			cfg:  cfg,
			o:    Overrides{ProviderSet: true},
			want: Profile{},
		},
		{
			name: "an explicitly-empty --model clears the model (supplied-ness, not emptiness)",
			cfg:  cfg,
			o:    Overrides{ModelSet: true},
			want: Profile{Provider: "claude", Model: "", Effort: base.Effort},
		},
		{
			name: "an explicit agent.profiles model survives a provider swap",
			cfg: &config.Config{
				Agent:     config.AgentConfig{Profiles: map[string]config.RoleProfile{"doing": {Model: "pinned-by-hand"}}},
				Providers: cfg.Providers,
			},
			o:    Overrides{Provider: "codex", ProviderSet: true},
			want: Profile{Provider: "codex", Model: "pinned-by-hand", Effort: "high"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveRoleWith(c.cfg, RoleDoing, c.o)
			if err != nil {
				t.Fatalf("ResolveRoleWith: %v", err)
			}
			if got != c.want {
				t.Errorf("ResolveRoleWith = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestResolveVerbatimNoValidation: a deliberately-incompatible override (Sonnet +
// xhigh, which Sonnet rejects at dispatch) is echoed verbatim with no error — fab
// does NOT validate or correct. The harness is the safety net.
func TestResolveVerbatimNoValidation(t *testing.T) {
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
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
	cfg = cfgWithProfiles(map[string]config.RoleProfile{
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
// fields empty is a no-op that keeps the built-in default.
func TestResolveEmptyOverrideKeepsDefault(t *testing.T) {
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
		"doing": {},
	})
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, _ := DefaultProfile(RoleDoing) // apply ∈ doing
	if got != want {
		t.Errorf("Resolve(apply) with empty override = %+v, want built-in default %+v", got, want)
	}
}

// TestResolveRole: a role name resolves directly (the path fab agent / operator
// use), independent of any stage.
func TestResolveRole(t *testing.T) {
	got, err := ResolveRole(nil, RoleOperator)
	if err != nil {
		t.Fatalf("ResolveRole(operator): %v", err)
	}
	want := Profile{Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"}
	if got != want {
		t.Errorf("ResolveRole(operator) = %+v, want %+v", got, want)
	}

	if _, err := ResolveRole(nil, "bogus"); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}

// TestRoleForName: the positional `<stage|role>` argument maps to a role — role
// names first, then stages, then the unknown-stage error.
func TestRoleForName(t *testing.T) {
	cases := map[string]string{
		"operator":  RoleOperator, // a role, not a stage
		"apply":     RoleDoing,    // a stage
		"review":    RoleReview,   // a fixed point (both)
		"hydrate":   RoleHydrate,  // a fixed point (both)
		"ship":      RoleFast,     // a stage that is not a role
		"review-pr": RoleDoing,
	}
	for name, want := range cases {
		got, err := RoleForName(name)
		if err != nil {
			t.Fatalf("RoleForName(%q): %v", name, err)
		}
		if got != want {
			t.Errorf("RoleForName(%q) = %q, want %q", name, got, want)
		}
	}
	if _, err := RoleForName("frobnicate"); err == nil {
		t.Error("expected an error for a name that is neither a stage nor a role")
	}
}

// TestIsRoleName: the six role names report true; non-role names (stages that are
// NOT also roles, plus unknowns) report false. The resolve-agent positional-arg
// contract: a name shared by a stage and a role (review, hydrate) IS a role, so
// those are not in the not-a-role list. "ship" is a STAGE but not a role — it maps
// to the fast role — so it stays in the not-a-role list.
func TestIsRoleName(t *testing.T) {
	for _, role := range RoleNames() {
		if !IsRoleName(role) {
			t.Errorf("IsRoleName(%q) = false, want true", role)
		}
	}
	for _, notRole := range []string{"apply", "review-pr", "intake", "ship", "frobnicate", ""} {
		if IsRoleName(notRole) {
			t.Errorf("IsRoleName(%q) = true, want false", notRole)
		}
	}
}

// TestResolveProvider: the built-in claude provider resolves with all three
// capabilities; a project override per-field merges;
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
	if prov.DispatchCommand != DefaultDispatchCommand || !prov.Native {
		t.Errorf("claude capabilities = %+v, want dispatch command %q and native=true", prov, DefaultDispatchCommand)
	}

	// Project override replaces dispatch_command; the session/native capability inherits the
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
	if !prov.Native {
		t.Error("native capability must survive a command-only override")
	}

	// YAML presence distinguishes an explicit false from absent, so users can
	// disable a built-in capability rather than only enabling one.
	var disableNative config.Config
	if err := yaml.Unmarshal([]byte("providers:\n  claude:\n    native: false\n"), &disableNative); err != nil {
		t.Fatal(err)
	}
	prov, _ = ResolveProvider(&disableNative, "claude")
	if prov.Native {
		t.Error("an explicit native:false override must disable the built-in capability")
	}
	// The built-in per-role fills survive a command-only override.
	if prov.Profiles[RoleDoing].Model == "" {
		t.Error("a command-only override must not drop the built-in per-role fills")
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

	// A project-only provider (in neither the built-in table nor a knob) resolves
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
// providers — resolvable with NO providers: config at all — carrying both command
// fields plus their own per-role fills, and never the DEPRECATED flat pair (which
// exists only as a read-time alias for a user's profiles.default).
func TestResolveProvider_BuiltInCodexAndGemini(t *testing.T) {
	cases := []struct {
		name                  string
		session, dispatch     string
		approvalBypassGrammar string
	}{
		{"codex", DefaultCodexSessionCommand, DefaultCodexDispatchCommand, "--dangerously-bypass-approvals-and-sandbox"},
		{"gemini", DefaultGeminiSessionCommand, DefaultGeminiDispatchCommand, "--approval-mode=yolo"},
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
			t.Errorf("%s.DispatchCommand = %q, want %q", c.name, prov.DispatchCommand, c.dispatch)
		}
		if prov.Native {
			t.Errorf("%s.Native = true, want false", c.name)
		}
		if !strings.Contains(prov.SessionCommand, c.approvalBypassGrammar) {
			t.Errorf("%s.SessionCommand = %q, want approval-bypass grammar %q (stage workers cannot answer approval prompts)", c.name, prov.SessionCommand, c.approvalBypassGrammar)
		}
		if !strings.Contains(prov.DispatchCommand, c.approvalBypassGrammar) {
			t.Errorf("%s.DispatchCommand = %q, want approval-bypass grammar %q (stage workers cannot answer approval prompts)", c.name, prov.DispatchCommand, c.approvalBypassGrammar)
		}
		if len(prov.Profiles) == 0 {
			t.Errorf("%s built-in carries no per-role fills — naming it on a depth knob must resolve a real model for every role", c.name)
		}
		if prov.Model != "" || prov.Effort != "" {
			t.Errorf("%s built-in uses the DEPRECATED flat fill (%q/%q) — built-in fills belong under profiles.<role>", c.name, prov.Model, prov.Effort)
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

// TestResolveProvider_ProfilesMerge: the per-role fill map merges over the built-in
// per ROLE and per FIELD — so a config supplying only fills inherits the built-in
// grammar, and a partial fill leaves the other field empty (no cross-field
// invention).
func TestResolveProvider_ProfilesMerge(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex": {Profiles: map[string]config.ProviderProfile{
			"default": {Model: "gpt-5.3-codex", Effort: "high"},
		}},
	}}
	prov, ok := ResolveProvider(cfg, "codex")
	if !ok {
		t.Fatal("codex must resolve")
	}
	if prov.Profiles["default"].Model != "gpt-5.3-codex" || prov.Profiles["default"].Effort != "high" {
		t.Errorf("fill = %+v, want the configured {gpt-5.3-codex, high}", prov.Profiles["default"])
	}
	if prov.SessionCommand != DefaultCodexSessionCommand || prov.DispatchCommand != DefaultCodexDispatchCommand {
		t.Errorf("commands = {%q, %q}, want the inherited built-in grammar", prov.SessionCommand, prov.DispatchCommand)
	}

	// A partial fill leaves the other field empty.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"gemini": {Profiles: map[string]config.ProviderProfile{
			"default": {Model: "gemini-2.5-pro"},
		}},
	}}
	prov, _ = ResolveProvider(cfg, "gemini")
	if got := prov.Profiles["default"]; got.Model != "gemini-2.5-pro" || got.Effort != "" {
		t.Errorf("partial fill = %+v, want {gemini-2.5-pro, \"\"}", got)
	}

	// Overriding one FIELD of one ROLE on claude leaves the rest of that role's
	// built-in fill intact.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"claude": {Profiles: map[string]config.ProviderProfile{
			"doing": {Effort: "medium"},
		}},
	}}
	prov, _ = ResolveProvider(cfg, "claude")
	builtinDoing, _ := DefaultProfile(RoleDoing)
	if got := prov.Profiles[RoleDoing]; got.Effort != "medium" || got.Model != builtinDoing.Model {
		t.Errorf("claude doing fill = %+v, want the overridden effort with the built-in model %q", got, builtinDoing.Model)
	}
}

// TestProviderNames: the resolvable provider set is the union of fab-kit's
// built-in table and the project's providers: block, sorted and de-duplicated —
// the set `fab agent --provider <unknown>` names in its lookup-failure error.
func TestProviderNames(t *testing.T) {
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
		"codex":   {Profiles: map[string]config.ProviderProfile{"default": {Model: "gpt-5.3-codex"}}},
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

// TestTablesExhaustive: every stage's role has a default profile, and the stage
// set is exactly the six pipeline stages.
func TestTablesExhaustive(t *testing.T) {
	for _, stage := range StageNames() {
		role, _ := RoleForStage(stage)
		if _, ok := DefaultProfile(role); !ok {
			t.Errorf("stage %q maps to role %q which has no default profile", stage, role)
		}
	}
	stages := strings.Join(StageNames(), ",")
	want := "apply,hydrate,intake,review,review-pr,ship"
	if stages != want {
		t.Errorf("stage set = %q, want %q", stages, want)
	}

	// The role set is exactly the six roles.
	roles := strings.Join(RoleNames(), ",")
	wantRoles := "default,doing,fast,hydrate,operator,review"
	if roles != wantRoles {
		t.Errorf("role set = %q, want %q", roles, wantRoles)
	}
}

// TestStageRoleCollisionsAreFixedPoints: every name shared by the stage set and
// the role set (review, hydrate) must be a FIXED POINT — the stage maps to the
// same-named role (stageRoles[name] == name). This is what makes the role-first
// resolution order in RoleForName immaterial for those names: a shared name
// resolves identically whether read as a stage or a role. It guards that order from
// ever silently changing a stage's resolution. (ship is a stage but NOT a role — it
// maps to fast — so it is not a collision.)
func TestStageRoleCollisionsAreFixedPoints(t *testing.T) {
	roleSet := make(map[string]bool)
	for _, role := range RoleNames() {
		roleSet[role] = true
	}
	collisions := 0
	for _, stage := range StageNames() {
		if !roleSet[stage] {
			continue // not a shared name
		}
		collisions++
		role, ok := RoleForStage(stage)
		if !ok {
			t.Errorf("stage %q has no role mapping", stage)
			continue
		}
		if role != stage {
			t.Errorf("stage/role name collision %q is NOT a fixed point: stageRoles[%q] = %q, want %q "+
				"(a name shared by a stage and a role must map the stage to the same-named role, "+
				"or the role-first resolve order would change the stage's resolution)", stage, stage, role, stage)
		}
	}
	// Guard the guard: the intended collisions (review, hydrate) must exist —
	// a zero-collision result would mean this test silently checks nothing.
	if collisions == 0 {
		t.Fatal("expected at least one stage/role name collision (review, hydrate); found none")
	}
}
