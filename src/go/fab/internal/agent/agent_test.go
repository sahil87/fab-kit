package agent

import (
	"io"
	"os"
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

// TestNonClaudeProviderFillsArePinned is the deliberate-change pin for the
// non-claude fills fab-kit ships (260806-ywkx). It is the non-claude sibling of
// TestDefaultRoleProfilesArePinned: those fills never reach DefaultProfile (which
// resolves through the claude-defaulted depth knobs), so without this table a
// catalog bump would land unreviewed.
//
// The tables are SPARSE, exactly as defaults.yaml is — a role absent here must be
// absent there, so the cross-role fallback through the provider's `default` entry
// stays the shipped shape rather than an accident of enumeration. kimi's EMPTY
// table is pinned for the same reason (260808-rpsr): shipping it a fill is the
// change that must be deliberate, not the change that slips through.
//
// When you bump a fill: edit defaults.yaml, then update this table to match. The
// doc mirrors are guarded separately by TestMirrorDocsMatchDefaultProfiles.
func TestNonClaudeProviderFillsArePinned(t *testing.T) {
	pinned := map[string]map[string]config.ProviderProfile{
		providerCodex: {
			RoleDefault:  {Model: "gpt-5.6-sol", Effort: "high"},
			RoleOperator: {Model: "gpt-5.6-luna", Effort: "medium"},
			RoleDoing:    {Effort: "xhigh"},
			RoleReview:   {Effort: "xhigh"},
			RoleFast:     {Model: "gpt-5.6-luna", Effort: "low"},
		},
		// Model-only: agy's model IDs embed the reasoning level as an ID SUFFIX
		// (…-high / …-low), so there is no separate effort to fill. These are
		// concrete IDs from `agy models` and are the rows to bump when that
		// catalog moves.
		providerAgy: {
			RoleDefault: {Model: "gemini-3.1-pro-high"},
			RoleFast:    {Model: "gemini-3.6-flash-low"},
		},
		// kimi ships NOTHING, deliberately: its -m takes a user-config model alias
		// rather than a catalog ID, so a pinned value breaks non-managed installs.
		providerKimi: nil,
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
	workers := &config.Config{Agent: config.AgentConfig{Workers: "agy"}}
	for _, role := range RoleNames() {
		want := "claude"
		if roleDepth[role] == depthWorkers {
			want = "agy"
		}
		got, err := ResolveRole(workers, role)
		if err != nil {
			t.Fatalf("ResolveRole(%s): %v", role, err)
		}
		if got.Provider != want {
			t.Errorf("with agent.workers=agy, role %q resolved provider %q, want %q", role, got.Provider, want)
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
//
// kimi is deliberately NOT in this loop — it ships no fills, so it has no role
// differentiation to preserve. Its opposite property (an empty model resolves for
// every workers role) is asserted by TestWorkersKnobOnNoFillsBuiltIn below.
func TestWorkersKnobResolvesBuiltInFills(t *testing.T) {
	for _, provider := range []string{providerCodex, providerAgy} {
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
				if provider == providerAgy && got.Effort != "" {
					t.Errorf("Resolve(%s) = %+v, want no effort — agy's model IDs embed the reasoning level instead", stage, got)
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

// TestWorkersKnobOnNoFillsBuiltIn is the counterpart for kimi, the one built-in
// that deliberately ships NO fills (260808-rpsr). `agent.workers: kimi` must move
// every Tier-2 stage onto kimi with an EMPTY model — the inherit-the-CLI's-own-
// default_model signal — rather than silently borrowing another provider's ID.
// kimi's -m takes a user-config alias, so an empty model is the only value that is
// correct for every install shape; the token-drop in internal/spawn is what turns
// it into a command with no -m flag.
func TestWorkersKnobOnNoFillsBuiltIn(t *testing.T) {
	cfg := &config.Config{Agent: config.AgentConfig{Workers: providerKimi}}

	for _, stage := range StageNames() {
		role, _ := RoleForStage(stage)
		got, err := Resolve(cfg, stage)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", stage, err)
		}

		if roleDepth[role] == depthSession {
			want, _ := DefaultProfile(role)
			if got != want {
				t.Errorf("Resolve(%s) = %+v, want the untouched Tier-1 claude default %+v", stage, got, want)
			}
			continue
		}

		if want := (Profile{Provider: providerKimi}); got != want {
			t.Errorf("Resolve(%s) = %+v, want %+v — kimi ships no fills, so model and effort must resolve EMPTY", stage, got, want)
		}
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

// TestResolveProvider_CommandFieldsAlias (260809-n1he): a config still carrying the
// pre-2.19.0 `session_command`/`dispatch_command` spellings keeps resolving, PER
// FIELD — so a half-migrated provider (one field renamed, one not) resolves both
// commands — and the new spelling wins wherever both are present, independently per
// field. Driven through yaml.Unmarshal rather than struct literals because the
// deprecated `yaml:` tags on config.ProviderConfig are half of what makes the alias
// work; a struct-literal test would keep passing with the tags dropped.
func TestResolveProvider_CommandFieldsAlias(t *testing.T) {
	cases := []struct {
		name            string
		providerYAML    string
		wantInteractive string
		wantHeadless    string
	}{
		{
			name: "only the old spellings — both fields resolve",
			providerYAML: "    session_command: myagent\n" +
				"    dispatch_command: myagent -p\n",
			wantInteractive: "myagent",
			wantHeadless:    "myagent -p",
		},
		{
			name: "both spellings on both fields — the new one wins on each",
			providerYAML: "    interactive_command: myagent --new\n" +
				"    session_command: myagent --old\n" +
				"    headless_command: myagent -p --new\n" +
				"    dispatch_command: myagent -p --old\n",
			wantInteractive: "myagent --new",
			wantHeadless:    "myagent -p --new",
		},
		{
			name: "half-migrated — new interactive_command beside an old dispatch_command",
			providerYAML: "    interactive_command: myagent --new\n" +
				"    dispatch_command: myagent -p --old\n",
			wantInteractive: "myagent --new",
			wantHeadless:    "myagent -p --old",
		},
		{
			name: "per-field independence — new wins the field that carries both, old still resolves the field it alone carries",
			providerYAML: "    interactive_command: myagent --new\n" +
				"    session_command: myagent --old\n" +
				"    dispatch_command: myagent -p --old\n",
			wantInteractive: "myagent --new",
			wantHeadless:    "myagent -p --old",
		},
	}

	stderr := captureStderr(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg config.Config
			if err := yaml.Unmarshal([]byte("providers:\n  myagent:\n"+tc.providerYAML), &cfg); err != nil {
				t.Fatalf("yaml.Unmarshal: %v", err)
			}
			prov, ok := ResolveProvider(&cfg, "myagent")
			if !ok {
				t.Fatal("a provider carrying deprecated command spellings must still resolve as known")
			}
			if prov.InteractiveCommand != tc.wantInteractive {
				t.Errorf("interactive_command = %q, want %q", prov.InteractiveCommand, tc.wantInteractive)
			}
			if prov.HeadlessCommand != tc.wantHeadless {
				t.Errorf("headless_command = %q, want %q", prov.HeadlessCommand, tc.wantHeadless)
			}
		})
	}

	// R1: the alias is a silent read affordance — an old spelling resolves without a
	// deprecation notice (the on-disk rewrite is the 2.18.1-to-2.19.0 migration's job).
	if out := stderr(); out != "" {
		t.Errorf("resolving deprecated command spellings wrote %q, want silence", out)
	}
}

// captureStderr redirects os.Stderr for the rest of the test and returns a func that
// restores it and yields everything written meanwhile (calling it more than once
// yields "" after the first). internal/config resolves its warning stream at CALL
// time — os.Stderr unless a test in that package overrides it — so swapping the file
// here is how a test in this package observes, or asserts the absence of, a warning.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	restored := false
	restore := func() {
		if !restored {
			restored = true
			os.Stderr = orig
			w.Close()
		}
	}
	t.Cleanup(func() {
		restore()
		r.Close()
	})
	return func() string {
		restore()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("read captured stderr: %v", err)
		}
		return string(out)
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
	agy, _ := ResolveProvider(nil, providerAgy)
	agyDoing := Profile{
		Provider: providerAgy,
		// agy ships no `doing` fill, so the role falls through to its `default`
		// entry — derived, not restated, so a fill bump does not touch this table.
		Model: agy.Profiles[RoleDefault].Model,
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
			// agy carries no `doing` fill of its own, so the swap lands on that
			// provider's `default` entry — never on claude's model, and (since
			// 260806-ywkx ships the fills) never on an empty one.
			name: "provider swap with no per-role fill falls to that provider's default entry",
			cfg:  cfg,
			o:    Overrides{Provider: "agy", ProviderSet: true},
			want: agyDoing,
		},
		{
			// kimi ships no fills at all, so a swap onto it resolves an EMPTY model
			// — the deliberate inherit-the-CLI's-own-default_model signal, which the
			// empty-value token-drop turns into a command with no -m flag.
			name: "provider swap onto the no-fills built-in resolves an empty model",
			cfg:  cfg,
			o:    Overrides{Provider: "kimi", ProviderSet: true},
			want: Profile{Provider: "kimi"},
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
	if prov.InteractiveCommand != DefaultInteractiveCommand {
		t.Errorf("claude.InteractiveCommand = %q, want the built-in default", prov.InteractiveCommand)
	}
	if prov.HeadlessCommand != DefaultHeadlessCommand || !prov.Native {
		t.Errorf("claude capabilities = %+v, want dispatch command %q and native=true", prov, DefaultHeadlessCommand)
	}

	// Project override replaces headless_command; the session/native capability inherits the
	// built-in (per-field merge).
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"claude": {HeadlessCommand: "claude -p"},
	}}
	prov, ok = ResolveProvider(cfg, "claude")
	if !ok {
		t.Fatal("claude provider must resolve with a project override")
	}
	if prov.InteractiveCommand != DefaultInteractiveCommand {
		t.Errorf("interactive_command = %q, want the inherited built-in", prov.InteractiveCommand)
	}
	if prov.HeadlessCommand != "claude -p" {
		t.Errorf("headless_command = %q, want the override", prov.HeadlessCommand)
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
		"codex": {HeadlessCommand: "codex exec --json"},
	}}
	prov, ok = ResolveProvider(cfg, "codex")
	if !ok || prov.HeadlessCommand != "codex exec --json" {
		t.Errorf("codex headless_command = %q, ok=%v, want the project override", prov.HeadlessCommand, ok)
	}
	if prov.InteractiveCommand != DefaultCodexInteractiveCommand {
		t.Errorf("codex interactive_command = %q, want the inherited built-in", prov.InteractiveCommand)
	}

	// A project-only provider (in neither the built-in table nor a knob) resolves
	// as known off the project entry alone.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"myagent": {InteractiveCommand: "myagent", HeadlessCommand: "myagent run"},
	}}
	prov, ok = ResolveProvider(cfg, "myagent")
	if !ok || prov.HeadlessCommand != "myagent run" {
		t.Errorf("myagent provider = %+v, ok=%v, want the project entry", prov, ok)
	}

	// An unknown provider reports ok=false.
	if _, ok := ResolveProvider(nil, "bogus"); ok {
		t.Error("unknown provider must report ok=false")
	}
}

// TestResolveProvider_NonClaudeBuiltIns: every non-claude built-in is resolvable
// with NO providers: config at all, carries the command fields that define it,
// ships a full-auto posture on the form that needs one, and never the DEPRECATED
// flat pair (which exists only as a read-time alias for a user's profiles.default).
//
// `wantFills` distinguishes the two fill shapes rather than asserting one of them:
// codex and agy ship per-role fills so a knob pointed at them resolves a real model
// per role, while kimi ships none on purpose (its -m takes a user-config alias).
//
// An empty `session` likewise ASSERTS the absence of an interactive_command. agy
// ships `agy --dangerously-skip-permissions --model {model}`; the first-run trust
// wall is an ordinary readiness-gate judgment round (kimi precedent) and the trust
// store is user-seedable. If a built-in ships none, mode resolution descends to
// headless instead of parking a pane worker.
//
// kimi's probe is done (2026-08-10, kimi 0.34.0): its trust wall is an ordinary
// readiness-gate judgment round and its side-bordered input box verifies under the
// gate's box-drawing-tolerant squeeze, so it ships a session command and is
// pane-capable.
func TestResolveProvider_NonClaudeBuiltIns(t *testing.T) {
	cases := []struct {
		name string
		// An empty `session` (if any built-in lacked one) would assert the absence
		// of an interactive_command. All current built-ins ship one.
		session, dispatch string
		// sessionBypass / dispatchBypass are the full-auto grammar each FORM must
		// carry, checked only when that form exists. kimi's dispatch form is
		// deliberately empty: `kimi -p` already auto-approves tools and ERRORS on
		// --yolo/--auto, so a bypass flag there would break the invocation rather
		// than harden it.
		sessionBypass, dispatchBypass string
		wantFills                     bool
	}{
		{
			name: "codex", session: DefaultCodexInteractiveCommand, dispatch: DefaultCodexHeadlessCommand,
			sessionBypass:  "--dangerously-bypass-approvals-and-sandbox",
			dispatchBypass: "--dangerously-bypass-approvals-and-sandbox",
			wantFills:      true,
		},
		{
			name: "agy", session: DefaultAgyInteractiveCommand, dispatch: DefaultAgyHeadlessCommand,
			sessionBypass:  "--dangerously-skip-permissions",
			dispatchBypass: "--dangerously-skip-permissions",
			wantFills:      true,
		},
		{
			name: "kimi", session: DefaultKimiInteractiveCommand, dispatch: DefaultKimiHeadlessCommand,
			// The full-auto flag rides the SESSION form only: `kimi -p` already
			// auto-approves tools and errors when combined with it.
			sessionBypass:  "--auto",
			dispatchBypass: "",
			wantFills:      false,
		},
	}
	for _, c := range cases {
		prov, ok := ResolveProvider(nil, c.name)
		if !ok {
			t.Fatalf("built-in %s provider must resolve with no config", c.name)
		}
		if prov.InteractiveCommand != c.session {
			t.Errorf("%s.InteractiveCommand = %q, want %q", c.name, prov.InteractiveCommand, c.session)
		}
		if prov.HeadlessCommand != c.dispatch {
			t.Errorf("%s.HeadlessCommand = %q, want %q", c.name, prov.HeadlessCommand, c.dispatch)
		}
		if prov.Native {
			t.Errorf("%s.Native = true, want false", c.name)
		}
		if c.sessionBypass != "" && !strings.Contains(prov.InteractiveCommand, c.sessionBypass) {
			t.Errorf("%s.InteractiveCommand = %q, want approval-bypass grammar %q (stage workers cannot answer approval prompts)", c.name, prov.InteractiveCommand, c.sessionBypass)
		}
		if c.dispatchBypass != "" && !strings.Contains(prov.HeadlessCommand, c.dispatchBypass) {
			t.Errorf("%s.HeadlessCommand = %q, want approval-bypass grammar %q (stage workers cannot answer approval prompts)", c.name, prov.HeadlessCommand, c.dispatchBypass)
		}
		if gotFills := len(prov.Profiles) > 0; gotFills != c.wantFills {
			if c.wantFills {
				t.Errorf("%s built-in carries no per-role fills — naming it on a depth knob must resolve a real model for every role", c.name)
			} else {
				t.Errorf("%s built-in carries fills %+v, want none — its -m takes a USER-CONFIG model alias, so a pinned value breaks non-managed installs", c.name, prov.Profiles)
			}
		}
		if prov.Model != "" || prov.Effort != "" {
			t.Errorf("%s built-in uses the DEPRECATED flat fill (%q/%q) — built-in fills belong under profiles.<role>", c.name, prov.Model, prov.Effort)
		}
	}

	// The agy grammar deliberately omits {effort}: its model IDs embed the reasoning
	// level as an ID suffix, so a separate effort flag would fight the suffix.
	if strings.Contains(DefaultAgyHeadlessCommand, "{effort}") {
		t.Error("the agy built-in must carry no {effort} placeholder (its model IDs embed the reasoning level)")
	}

	// Both new dispatch commands NEST a shell around `-p "$(cat)"`. Neither CLI
	// reads stdin as the prompt, and POSIX expands $(cat) BEFORE fab dispatch's
	// stdin redirect applies — so the un-nested form would read the OUTER stdin and
	// hand the worker an empty prompt. This is the load-bearing shape.
	for _, c := range []struct{ name, dispatch string }{
		{"agy", DefaultAgyHeadlessCommand},
		{"kimi", DefaultKimiHeadlessCommand},
	} {
		if !strings.HasPrefix(c.dispatch, "sh -c ") {
			t.Errorf("the %s dispatch command %q must nest a shell — $(cat) expands before the stdin redirect applies", c.name, c.dispatch)
		}
		if !strings.Contains(c.dispatch, `-p "$(cat)"`) {
			t.Errorf("the %s dispatch command %q must pass the piped prompt as the -p ARGUMENT (%s ignores stdin)", c.name, c.dispatch, c.name)
		}
	}

	// kimi's dispatch form must carry NO approval flag: `kimi -p` auto-approves
	// tools already and errors with "Cannot combine --prompt with --yolo".
	for _, bad := range []string{"--yolo", "--auto"} {
		if strings.Contains(DefaultKimiHeadlessCommand, bad) {
			t.Errorf("the kimi dispatch command must not carry %s — kimi -p rejects it and already auto-approves tools", bad)
		}
	}

	// kimi is the same seam read the other way: shipping an interactive_command is
	// exactly what makes it pane-ELIGIBLE, which is the whole point of the probe
	// that closed. Its {model} must stay droppable — kimi ships no fills, so a
	// pinned model here would be a value no install is guaranteed to accept.
	kimi, ok := ResolveProvider(nil, "kimi")
	if !ok {
		t.Fatalf("built-in kimi provider must resolve with no config")
	}
	if kimi.InteractiveCommand == "" {
		t.Error("kimi must be eligible for pane dispatch — its interactive first run and input echo were probed on 2026-08-10")
	}
	if !strings.Contains(kimi.InteractiveCommand, "-m {model}") {
		t.Errorf("kimi.InteractiveCommand = %q, want the droppable `-m {model}` pair (it ships no fills, so the empty model must remove the flag)", kimi.InteractiveCommand)
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
	if prov.InteractiveCommand != DefaultCodexInteractiveCommand || prov.HeadlessCommand != DefaultCodexHeadlessCommand {
		t.Errorf("commands = {%q, %q}, want the inherited built-in grammar", prov.InteractiveCommand, prov.HeadlessCommand)
	}

	// A partial fill leaves the other field empty. agy is the natural case: its
	// built-in fills are model-only, so a user pinning a newer ID must not acquire
	// an invented effort.
	cfg = &config.Config{Providers: map[string]config.ProviderConfig{
		"agy": {Profiles: map[string]config.ProviderProfile{
			"default": {Model: "gemini-3.1-pro-low"},
		}},
	}}
	prov, _ = ResolveProvider(cfg, "agy")
	if got := prov.Profiles["default"]; got.Model != "gemini-3.1-pro-low" || got.Effort != "" {
		t.Errorf("partial fill = %+v, want {gemini-3.1-pro-low, \"\"}", got)
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
	assertNames(t, ProviderNames(nil), []string{"agy", "claude", "codex", "kimi"})

	// Project providers union the built-in, de-duplicating shared keys and sorting
	// for stable error output.
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"codex":   {Profiles: map[string]config.ProviderProfile{"default": {Model: "gpt-5.3-codex"}}},
		"claude":  {HeadlessCommand: "claude -p"},
		"myagent": {InteractiveCommand: "myagent"},
	}}
	assertNames(t, ProviderNames(cfg), []string{"agy", "claude", "codex", "kimi", "myagent"})
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
