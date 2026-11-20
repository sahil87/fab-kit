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
// the six values moved from agent.tiers onto clis.claude.profiles, and every
// resolved stage profile must be unchanged by that move.
//
// When you bump a default: edit defaults.yaml, then update this table to match.
// The one doc mirror that spells these IDs (docs/specs/stage-models.md § Default
// role profiles) is guarded separately by TestDocTablesMatchAgentMaps; the rendered
// config reference derives its fill lines from ResolveCLI and is guarded by
// TestConfigReferenceDocumentsCLIFill in cmd/fab.
func TestDefaultRoleProfilesArePinned(t *testing.T) {
	pinned := map[string]Profile{
		RoleDefault:  {CLI: "claude", Model: "opus", Effort: "high"},
		RoleOperator: {CLI: "claude", Model: "sonnet", Effort: "medium"},
		RoleDoing:    {CLI: "claude", Model: "sonnet", Effort: "high"},
		RoleReview:   {CLI: "claude", Model: "opus", Effort: "high"},
		RoleHydrate:  {CLI: "claude", Model: "sonnet", Effort: "high"},
		RoleFast:     {CLI: "claude", Model: "sonnet", Effort: "medium"},
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

// TestNonClaudeCLIFillsArePinned is the deliberate-change pin for the
// non-claude fills fab-kit ships (260806-ywkx). It is the non-claude sibling of
// TestDefaultRoleProfilesArePinned: those fills never reach DefaultProfile (which
// resolves through the claude-defaulted depth knobs), so without this table a
// catalog bump would land unreviewed.
//
// Each table has exactly the SHAPE defaults.yaml ships — a role absent here must be
// absent there — so the shape itself is what a bump reviews, not an accident of
// enumeration. codex's map is DENSE by policy (260908-wcib): every role pins its own
// model, so a `default` bump never silently repoints doing/review/hydrate the way
// the earlier effort-only rows did (TestDefaultsFileCLIs enforces the
// density). agy's map stays sparse (its non-`fast` roles fall back to `default`),
// and kimi's EMPTY table is pinned for the same reason (260808-rpsr): shipping it a
// fill is the change that must be deliberate, not the change that slips through.
//
// When you bump a fill: edit defaults.yaml, then update this table to match. The
// spec's inline CLIs sample is shape-only (IDs elided) and unguarded by
// design; the rendered config reference is guarded by
// TestConfigReferenceDocumentsCLIFill in cmd/fab.
func TestNonClaudeCLIFillsArePinned(t *testing.T) {
	pinned := map[string]map[string]config.CLIProfile{
		// codex: concrete catalog slugs read from the installed CLI's cache
		// (~/.codex/models_cache.json). Author (doing) and critic (review) run
		// DIFFERENT models on purpose; no row ships `ultra`.
		cliCodex: {
			RoleDefault:  {Model: "gpt-6-astra", Effort: "high"},
			RoleDoing:    {Model: "gpt-6-astra", Effort: "high"},
			RoleReview:   {Model: "gpt-5.6-sol", Effort: "xhigh"},
			RoleHydrate:  {Model: "gpt-5.6-sol", Effort: "high"},
			RoleOperator: {Model: "gpt-5.6-luna", Effort: "medium"},
			RoleFast:     {Model: "gpt-5.6-luna", Effort: "low"},
		},
		// Model-only: agy's model IDs embed the reasoning level as an ID SUFFIX
		// (…-high / …-low), so there is no separate effort to fill. These are
		// concrete IDs from `agy models` and are the rows to bump when that
		// catalog moves.
		cliAgy: {
			RoleDefault: {Model: "gemini-3.1-pro-high"},
			RoleFast:    {Model: "gemini-3.6-flash-low"},
		},
		// kimi ships NOTHING, deliberately: its -m takes a user-config model alias
		// rather than a catalog ID, so a pinned value breaks non-managed installs.
		cliKimi: nil,
	}
	for name, want := range pinned {
		prov, ok := ResolveCLI(nil, name)
		if !ok {
			t.Errorf("built-in CLI %q must resolve", name)
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

// TestDepthKnobSelectsCLI is the headline behavior: `agent.workers: <name>`
// moves exactly the four Tier-2 roles and leaves the two session roles alone, and
// vice versa. Neither knob touches the other's partition.
func TestDepthKnobSelectsCLI(t *testing.T) {
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
		if got.CLI != want {
			t.Errorf("with agent.workers=agy, role %q resolved CLI %q, want %q", role, got.CLI, want)
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
		if got.CLI != want {
			t.Errorf("with agent.session=codex, role %q resolved CLI %q, want %q", role, got.CLI, want)
		}
	}
}

// TestWorkersKnobResolvesBuiltInFills is the flagship UX (260806-ywkx): ONE knob
// line moves every Tier-2 stage onto another CLI AND gives each role that
// CLI's own per-role model — role differentiation survives the swap instead of
// collapsing to an empty model for all four. The Tier-1 roles stay on claude.
//
// Expectations are DERIVED from ResolveCLI (with its cross-role fallback
// through the CLI's `default` entry), never restated, so a fill bump touches
// defaults.yaml and TestNonClaudeCLIFillsArePinned only.
//
// kimi is deliberately NOT in this loop — it ships no fills, so it has no role
// differentiation to preserve. Its opposite property (an empty model resolves for
// every workers role) is asserted by TestWorkersKnobOnNoFillsBuiltIn below.
func TestWorkersKnobResolvesBuiltInFills(t *testing.T) {
	for _, cli := range []string{cliCodex, cliAgy} {
		t.Run(cli, func(t *testing.T) {
			cfg := &config.Config{Agent: config.AgentConfig{Workers: cli}}
			prov, ok := ResolveCLI(nil, cli)
			if !ok {
				t.Fatalf("built-in %s must resolve", cli)
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
					CLI:    cli,
					Model:  firstNonEmpty(fill.Model, prov.Profiles[RoleDefault].Model),
					Effort: firstNonEmpty(fill.Effort, prov.Profiles[RoleDefault].Effort),
				}
				if got != want {
					t.Errorf("Resolve(%s) = %+v, want %s's %s fill %+v", stage, got, cli, role, want)
				}
				if got.Model == "" {
					t.Errorf("Resolve(%s) resolved an EMPTY model — a knob pointed at %s must resolve a real model for every role", stage, cli)
				}
				if cli == cliAgy && got.Effort != "" {
					t.Errorf("Resolve(%s) = %+v, want no effort — agy's model IDs embed the reasoning level instead", stage, got)
				}
			}
		})
	}

	// The headline differentiation: the two heavyweight roles and the cheap one do
	// NOT resolve the same profile.
	cfg := &config.Config{Agent: config.AgentConfig{Workers: cliCodex}}
	apply, _ := Resolve(cfg, "apply") // doing
	ship, _ := Resolve(cfg, "ship")   // fast
	if apply == ship {
		t.Errorf("apply and ship both resolved %+v — shipping fills exists precisely so the roles differ", apply)
	}
}

// TestWorkersKnobOnNoFillsBuiltIn is the counterpart for kimi, the one built-in
// that deliberately ships NO fills (260808-rpsr). `agent.workers: kimi` must move
// every Tier-2 stage onto kimi with an EMPTY model — the inherit-the-CLI's-own-
// default_model signal — rather than silently borrowing another CLI's ID.
// kimi's -m takes a user-config alias, so an empty model is the only value that is
// correct for every install shape; the token-drop in internal/spawn is what turns
// it into a command with no -m flag.
func TestWorkersKnobOnNoFillsBuiltIn(t *testing.T) {
	cfg := &config.Config{Agent: config.AgentConfig{Workers: cliKimi}}

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

		if want := (Profile{CLI: cliKimi}); got != want {
			t.Errorf("Resolve(%s) = %+v, want %+v — kimi ships no fills, so model and effort must resolve EMPTY", stage, got, want)
		}
	}
}

// TestResolveFullOverride: an agent.profiles entry sets CLI, model, and effort.
func TestResolveFullOverride(t *testing.T) {
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
		"doing": {CLI: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	})
	got, err := Resolve(cfg, "apply") // apply ∈ doing
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{CLI: "claude", Model: "claude-sonnet-5", Effort: "medium"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v", got, want)
	}
}

// TestResolvePerFieldMerge: an override that sets only effort keeps the CLI's
// own per-role model fill (per-field), and vice versa.
func TestResolvePerFieldMerge(t *testing.T) {
	// Only effort overridden → the CLI's hydrate fill supplies CLI+model.
	hydrateDefault, _ := DefaultProfile(RoleHydrate)
	cfg := cfgWithProfiles(map[string]config.RoleProfile{
		"hydrate": {Effort: "medium"},
	})
	got, err := Resolve(cfg, "hydrate")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.CLI != hydrateDefault.CLI || got.Model != hydrateDefault.Model || got.Effort != "medium" {
		t.Errorf("Resolve(hydrate) = %+v, want default CLI+model (%s/%s) + medium effort", got, hydrateDefault.CLI, hydrateDefault.Model)
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
// second fallback chain is deleted, leaving only the CLI-side one.
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

	// And a CLI named on the default role does not leak onto another role
	// either — the other role falls to its depth knob.
	cfg = cfgWithProfiles(map[string]config.RoleProfile{
		"default": {CLI: "codex"},
	})
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != doingDefault {
		t.Errorf("Resolve(apply) = %+v, want %+v — a default-role CLI must not re-base doing", got, doingDefault)
	}
}

// TestCLIPerRoleFills: a CLI's profiles map supplies model/effort per
// role, with its `default` entry acting as the cross-role fallback and an explicit
// agent.profiles field beating both.
//
// It uses a CLI name fab-kit does NOT ship, so the per-role/per-field merge
// with a built-in table is out of the picture and the precedence under test is the
// only thing the expectations depend on (TestPartialCLIFillMergesOverBuiltIn
// covers the merge itself).
func TestCLIPerRoleFills(t *testing.T) {
	myagent := map[string]config.CLIConfig{
		"myagent": {Profiles: map[string]config.CLIProfile{
			"default": {Model: "gpt-5.3-codex", Effort: "medium"},
			"doing":   {Model: "gpt-5.3-codex-max", Effort: "high"},
		}},
	}

	cases := []struct {
		name     string
		agentCfg config.AgentConfig
		clis     map[string]config.CLIConfig
		stage    string
		want     Profile
	}{
		{
			name:     "the role's own fill wins",
			agentCfg: config.AgentConfig{Workers: "myagent"},
			clis:     myagent,
			stage:    "apply", // doing
			want:     Profile{CLI: "myagent", Model: "gpt-5.3-codex-max", Effort: "high"},
		},
		{
			name:     "a role with no fill falls to the CLI's default entry",
			agentCfg: config.AgentConfig{Workers: "myagent"},
			clis:     myagent,
			stage:    "review",
			want:     Profile{CLI: "myagent", Model: "gpt-5.3-codex", Effort: "medium"},
		},
		{
			name: "an explicit agent.profiles field beats the CLI fill",
			agentCfg: config.AgentConfig{
				Workers:  "myagent",
				Profiles: map[string]config.RoleProfile{"doing": {Model: "gpt-5.2-codex"}},
			},
			clis:  myagent,
			stage: "apply",
			want:  Profile{CLI: "myagent", Model: "gpt-5.2-codex", Effort: "high"},
		},
		{
			name: "a per-role cli: override picks that CLI's fills",
			agentCfg: config.AgentConfig{
				Profiles: map[string]config.RoleProfile{"review": {CLI: "myagent"}},
			},
			clis:  myagent,
			stage: "review",
			want:  Profile{CLI: "myagent", Model: "gpt-5.3-codex", Effort: "medium"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &config.Config{Agent: c.agentCfg, CLIs: c.clis}
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

// TestPartialCLIFillMergesOverBuiltIn: overriding ONE role's fill on the
// built-in claude CLI leaves the other five (and the overridden role's other
// field) on the shipped values — the per-role, per-field merge.
func TestPartialCLIFillMergesOverBuiltIn(t *testing.T) {
	cfg := &config.Config{CLIs: map[string]config.CLIConfig{
		"claude": {Profiles: map[string]config.CLIProfile{
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

// TestResolveCLIDoesNotMutateBuiltIns: the resolved Profiles map is a fresh
// copy, so a caller writing through it cannot poison the shipped defaults for the
// rest of the process.
func TestResolveCLIDoesNotMutateBuiltIns(t *testing.T) {
	prov, ok := ResolveCLI(nil, "claude")
	if !ok {
		t.Fatal("built-in claude must resolve")
	}
	prov.Profiles[RoleDoing] = config.CLIProfile{Model: "poisoned"}

	want, _ := DefaultProfile(RoleDoing)
	if want.Model == "poisoned" {
		t.Fatal("mutating a resolved CLI's Profiles map corrupted the built-in table")
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

// TestLegacyFlatCLIFill (260806-j9nh): the pre-2.17.0 flat
// clis.<name>.model/.effort is still read, as an alias for
// profiles.default — so a config that has not yet run the migration keeps
// resolving. profiles.default wins when both are present.
func TestLegacyFlatCLIFill(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{Workers: "myagent"},
		CLIs: map[string]config.CLIConfig{
			"myagent": {Model: "gpt-5.3-codex", Effort: "high"},
		},
	}
	got, err := Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := Profile{CLI: "myagent", Model: "gpt-5.3-codex", Effort: "high"}
	if got != want {
		t.Errorf("Resolve(apply) = %+v, want %+v (legacy flat fill still read)", got, want)
	}

	// profiles.default is the modern spelling and outranks it.
	cfg.CLIs["myagent"] = config.CLIConfig{
		Model:    "gpt-5.3-codex",
		Profiles: map[string]config.CLIProfile{"default": {Model: "gpt-5.4-codex"}},
	}
	got, err = Resolve(cfg, "apply")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "gpt-5.4-codex" {
		t.Errorf("Resolve(apply) model = %q, want profiles.default to beat the legacy flat fill", got.Model)
	}
}

// TestResolveCLI_CommandFieldsAlias (260809-n1he): a config still carrying the
// pre-2.19.0 `session_command`/`dispatch_command` spellings keeps resolving, PER
// FIELD — so a half-migrated CLI (one field renamed, one not) resolves both
// commands — and the new spelling wins wherever both are present, independently per
// field. Driven through yaml.Unmarshal rather than struct literals because the
// deprecated `yaml:` tags on config.CLIConfig are half of what makes the alias
// work; a struct-literal test would keep passing with the tags dropped.
func TestResolveCLI_CommandFieldsAlias(t *testing.T) {
	cases := []struct {
		name            string
		cliYAML         string
		wantInteractive string
		wantHeadless    string
	}{
		{
			name: "only the old spellings — both fields resolve",
			cliYAML: "    session_command: myagent\n" +
				"    dispatch_command: myagent -p\n",
			wantInteractive: "myagent",
			wantHeadless:    "myagent -p",
		},
		{
			name: "both spellings on both fields — the new one wins on each",
			cliYAML: "    interactive_command: myagent --new\n" +
				"    session_command: myagent --old\n" +
				"    headless_command: myagent -p --new\n" +
				"    dispatch_command: myagent -p --old\n",
			wantInteractive: "myagent --new",
			wantHeadless:    "myagent -p --new",
		},
		{
			name: "half-migrated — new interactive_command beside an old dispatch_command",
			cliYAML: "    interactive_command: myagent --new\n" +
				"    dispatch_command: myagent -p --old\n",
			wantInteractive: "myagent --new",
			wantHeadless:    "myagent -p --old",
		},
		{
			name: "per-field independence — new wins the field that carries both, old still resolves the field it alone carries",
			cliYAML: "    interactive_command: myagent --new\n" +
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
			if err := yaml.Unmarshal([]byte("clis:\n  myagent:\n"+tc.cliYAML), &cfg); err != nil {
				t.Fatalf("yaml.Unmarshal: %v", err)
			}
			prov, ok := ResolveCLI(&cfg, "myagent")
			if !ok {
				t.Fatal("a CLI carrying deprecated command spellings must still resolve as known")
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

// TestFlatCLIFillBeatsBuiltInDefault (260806-ywkx) is the regression guard for
// shipping non-claude fills: the flat spelling is an ALIAS for the user's own
// profiles.default, so it must outrank the BUILT-IN profiles.default it is trying to
// replace. Were it read as a rung below profiles.default instead — its shape before
// fab-kit shipped a non-claude default — a pre-migration config's pinned model would
// be silently shadowed by fab-kit's shipped one.
//
// Since 260908-wcib codex's map is DENSE, so the only role the flat alias still
// reaches is `default` itself (intake maps there); every other role's built-in ROLE
// fill outranks the folded value, exactly as a hand-written profiles.default would
// be outranked. Both halves are asserted: the alias wins on `default`, and a role
// fill wins over it on apply/hydrate (the upgrade note in stage-models.md).
func TestFlatCLIFillBeatsBuiltInDefault(t *testing.T) {
	builtin, _ := ResolveCLI(nil, cliCodex)
	if builtin.Profiles[RoleDefault].Model == "" {
		t.Fatal("this guard is meaningless unless the codex built-in ships a profiles.default")
	}

	// A pre-migration config carrying ONLY the flat spelling.
	cfg := &config.Config{
		Agent: config.AgentConfig{Session: cliCodex, Workers: cliCodex},
		CLIs:  map[string]config.CLIConfig{cliCodex: {Model: "my-pinned-model"}},
	}
	got, err := Resolve(cfg, "intake") // intake → the `default` role, the alias's target
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "my-pinned-model" {
		t.Errorf("Resolve(intake) model = %q, want the user's flat pin — a built-in profiles.default must not shadow it", got.Model)
	}
	// The effort the user did NOT pin still comes from the built-in fill (per-field).
	if got.Effort != builtin.Profiles[RoleDefault].Effort {
		t.Errorf("Resolve(intake) effort = %q, want the built-in %q (the flat fill pinned only the model)", got.Effort, builtin.Profiles[RoleDefault].Effort)
	}

	// A built-in ROLE fill outranks the folded alias: with codex's dense map, the
	// flat spelling no longer reaches apply (doing) or hydrate.
	for _, stage := range []string{"apply", "hydrate"} {
		role, _ := RoleForStage(stage)
		got, err := Resolve(cfg, stage)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", stage, err)
		}
		if want := builtin.Profiles[role].Model; got.Model != want {
			t.Errorf("Resolve(%s) model = %q, want the built-in %s fill %q — a shipped role fill outranks the flat alias", stage, got.Model, role, want)
		}
	}

	// The user's OWN profiles.default still beats their flat fill — the modern
	// spelling wins over its alias, exactly as TestLegacyFlatCLIFill asserts
	// for a CLI fab-kit ships nothing for.
	cfg.CLIs[cliCodex] = config.CLIConfig{
		Model:    "my-pinned-model",
		Profiles: map[string]config.CLIProfile{RoleDefault: {Model: "my-modern-model"}},
	}
	got, err = Resolve(cfg, "intake")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "my-modern-model" {
		t.Errorf("Resolve(intake) model = %q, want profiles.default to beat the flat alias", got.Model)
	}
}

func TestEffectiveModel(t *testing.T) {
	if ToolDefaultModel != "tool-default" {
		t.Fatalf("ToolDefaultModel = %q, want %q", ToolDefaultModel, "tool-default")
	}
	cases := []struct {
		model string
		want  string
	}{
		{model: ToolDefaultModel, want: ""},
		{model: "Tool-Default", want: "Tool-Default"},
		{model: " tool-default", want: " tool-default"},
		{model: "gpt-5.6-sol", want: "gpt-5.6-sol"},
		{model: "opus", want: "opus"},
		{model: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			if got := EffectiveModel(tc.model); got != tc.want {
				t.Errorf("EffectiveModel(%q) = %q, want %q", tc.model, got, tc.want)
			}
		})
	}
}

func TestResolveRoleWithSourceCollapsesToolDefaultAfterPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		cfg        *config.Config
		overrides  Overrides
		wantSource string
	}{
		{
			name: "agent profile beats CLI fill",
			cfg: &config.Config{
				Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{
					RoleReview: {CLI: cliCodex, Model: ToolDefaultModel},
				}},
				CLIs: map[string]config.CLIConfig{
					cliCodex: {Profiles: map[string]config.CLIProfile{
						RoleReview: {Model: "CLI-fill"},
					}},
				},
			},
			wantSource: "agent.profiles.review",
		},
		{
			name: "CLI role fill",
			cfg: &config.Config{CLIs: map[string]config.CLIConfig{
				cliCodex: {Profiles: map[string]config.CLIProfile{
					RoleReview: {Model: ToolDefaultModel},
				}},
			}},
			overrides:  Overrides{CLI: cliCodex, CLISet: true},
			wantSource: "clis.codex.profiles.review",
		},
		{
			name: "CLI default fill",
			cfg: &config.Config{CLIs: map[string]config.CLIConfig{
				"custom": {Profiles: map[string]config.CLIProfile{
					RoleDefault: {Model: ToolDefaultModel},
				}},
			}},
			overrides:  Overrides{CLI: "custom", CLISet: true},
			wantSource: "clis.custom.profiles.default",
		},
		{
			name:       "model flag",
			overrides:  Overrides{Model: ToolDefaultModel, ModelSet: true},
			wantSource: "flag",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, source, err := ResolveRoleWithSource(tc.cfg, RoleReview, tc.overrides)
			if err != nil {
				t.Fatalf("ResolveRoleWithSource: %v", err)
			}
			if got.Model != "" {
				t.Errorf("Model = %q, want empty", got.Model)
			}
			if source.Model != tc.wantSource {
				t.Errorf("source.Model = %q, want %q", source.Model, tc.wantSource)
			}
		})
	}
}

// TestResolveRoleWithOverrides: the top rung of the fill precedence. Each flag
// applies only when SUPPLIED (its Set companion); a CLI SWAP re-derives the
// unoverridden model/effort from the NEW CLI's own per-role fills (never
// retaining the old CLI's); and an explicit agent.profiles pin survives a swap,
// because a value the user wrote is not inheritance.
func TestResolveRoleWithOverrides(t *testing.T) {
	base, _ := DefaultProfile(RoleDoing)
	agy, _ := ResolveCLI(nil, cliAgy)
	agyDoing := Profile{
		CLI: cliAgy,
		// agy ships no `doing` fill, so the role falls through to its `default`
		// entry — derived, not restated, so a fill bump does not touch this table.
		Model: agy.Profiles[RoleDefault].Model,
	}
	cfg := &config.Config{CLIs: map[string]config.CLIConfig{
		"codex": {Profiles: map[string]config.CLIProfile{
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
			name: "model only (no --cli) is a within-role override",
			cfg:  cfg,
			o:    Overrides{Model: "claude-sonnet-5", ModelSet: true},
			want: Profile{CLI: "claude", Model: "claude-sonnet-5", Effort: base.Effort},
		},
		{
			name: "effort only (no --cli) is a within-role override",
			cfg:  cfg,
			o:    Overrides{Effort: "medium", EffortSet: true},
			want: Profile{CLI: "claude", Model: base.Model, Effort: "medium"},
		},
		{
			name: "CLI swap re-derives from the new CLI's per-role fill",
			cfg:  cfg,
			o:    Overrides{CLI: "codex", CLISet: true},
			want: Profile{CLI: "codex", Model: "gpt-5.3-codex", Effort: "high"},
		},
		{
			// agy carries no `doing` fill of its own, so the swap lands on that
			// CLI's `default` entry — never on claude's model, and (since
			// 260806-ywkx ships the fills) never on an empty one.
			name: "CLI swap with no per-role fill falls to that CLI's default entry",
			cfg:  cfg,
			o:    Overrides{CLI: "agy", CLISet: true},
			want: agyDoing,
		},
		{
			// kimi ships no fills at all, so a swap onto it resolves an EMPTY model
			// — the deliberate inherit-the-CLI's-own-default_model signal, which the
			// empty-value token-drop turns into a command with no -m flag.
			name: "CLI swap onto the no-fills built-in resolves an empty model",
			cfg:  cfg,
			o:    Overrides{CLI: "kimi", CLISet: true},
			want: Profile{CLI: "kimi"},
		},
		{
			name: "explicit flags beat the swapped CLI's fill",
			cfg:  cfg,
			o: Overrides{
				CLI: "codex", CLISet: true,
				Model: "gpt-5.4-codex", ModelSet: true,
				Effort: "xhigh", EffortSet: true,
			},
			want: Profile{CLI: "codex", Model: "gpt-5.4-codex", Effort: "xhigh"},
		},
		{
			name: "swapping to the already-resolved CLI changes nothing",
			cfg:  cfg,
			o:    Overrides{CLI: "claude", CLISet: true},
			want: base,
		},
		{
			name: "an unknown CLI is not an error here (lookup is the caller's)",
			cfg:  cfg,
			o:    Overrides{CLI: "bogus", CLISet: true},
			want: Profile{CLI: "bogus"},
		},
		{
			name: "an explicitly-empty --cli resolves empty, never the depth knob",
			cfg:  cfg,
			o:    Overrides{CLISet: true},
			want: Profile{},
		},
		{
			name: "an explicitly-empty --model clears the model (supplied-ness, not emptiness)",
			cfg:  cfg,
			o:    Overrides{ModelSet: true},
			want: Profile{CLI: "claude", Model: "", Effort: base.Effort},
		},
		{
			name: "an explicit agent.profiles model survives a CLI swap",
			cfg: &config.Config{
				Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{"doing": {Model: "pinned-by-hand"}}},
				CLIs:  cfg.CLIs,
			},
			o:    Overrides{CLI: "codex", CLISet: true},
			want: Profile{CLI: "codex", Model: "pinned-by-hand", Effort: "high"},
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

	// A non-Claude CLI's vocabulary passes through untouched too.
	cfg = cfgWithProfiles(map[string]config.RoleProfile{
		"review": {CLI: "codex", Model: "gpt-5", Effort: "reasoning_effort:high"},
	})
	got, err = Resolve(cfg, "review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.CLI != "codex" || got.Model != "gpt-5" || got.Effort != "reasoning_effort:high" {
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
	// Derived, not restated: defaults.yaml is the one place a fill is spelled,
	// and TestDefaultRoleProfilesArePinned is the one place its value is pinned.
	want, _ := DefaultProfile(RoleOperator)
	if got != want {
		t.Errorf("ResolveRole(operator) = %+v, want built-in default %+v", got, want)
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

// TestResolveCLI: the built-in claude CLI resolves with all three
// capabilities; a project override per-field merges;
// an unknown CLI reports ok=false.
func TestResolveCLI(t *testing.T) {
	// Built-in claude, no project config.
	prov, ok := ResolveCLI(nil, "claude")
	if !ok {
		t.Fatal("built-in claude CLI must resolve")
	}
	if prov.InteractiveCommand != DefaultInteractiveCommand {
		t.Errorf("claude.InteractiveCommand = %q, want the built-in default", prov.InteractiveCommand)
	}
	if prov.HeadlessCommand != DefaultHeadlessCommand || !prov.Native {
		t.Errorf("claude capabilities = %+v, want dispatch command %q and native=true", prov, DefaultHeadlessCommand)
	}

	// Project override replaces headless_command; the session/native capability inherits the
	// built-in (per-field merge).
	cfg := &config.Config{CLIs: map[string]config.CLIConfig{
		"claude": {HeadlessCommand: "claude -p", ModelsCommand: "claude models --json"},
	}}
	prov, ok = ResolveCLI(cfg, "claude")
	if !ok {
		t.Fatal("claude CLI must resolve with a project override")
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
	if prov.ModelsCommand != "claude models --json" {
		t.Errorf("models_command = %q, want the override", prov.ModelsCommand)
	}
	for _, name := range CLINames(nil) {
		builtin, _ := ResolveCLI(nil, name)
		if builtin.ModelsCommand != "" {
			t.Errorf("built-in %s models_command = %q, want empty", name, builtin.ModelsCommand)
		}
	}

	// YAML presence distinguishes an explicit false from absent, so users can
	// disable a built-in capability rather than only enabling one.
	var disableNative config.Config
	if err := yaml.Unmarshal([]byte("clis:\n  claude:\n    native: false\n"), &disableNative); err != nil {
		t.Fatal(err)
	}
	prov, _ = ResolveCLI(&disableNative, "claude")
	if prov.Native {
		t.Error("an explicit native:false override must disable the built-in capability")
	}
	// The built-in per-role fills survive a command-only override.
	if prov.Profiles[RoleDoing].Model == "" {
		t.Error("a command-only override must not drop the built-in per-role fills")
	}

	// A project override of a non-claude built-in per-field merges too.
	cfg = &config.Config{CLIs: map[string]config.CLIConfig{
		"codex": {HeadlessCommand: "codex exec --json"},
	}}
	prov, ok = ResolveCLI(cfg, "codex")
	if !ok || prov.HeadlessCommand != "codex exec --json" {
		t.Errorf("codex headless_command = %q, ok=%v, want the project override", prov.HeadlessCommand, ok)
	}
	if prov.InteractiveCommand != DefaultCodexInteractiveCommand {
		t.Errorf("codex interactive_command = %q, want the inherited built-in", prov.InteractiveCommand)
	}

	// A project-only CLI (in neither the built-in table nor a knob) resolves
	// as known off the project entry alone.
	cfg = &config.Config{CLIs: map[string]config.CLIConfig{
		"myagent": {InteractiveCommand: "myagent", HeadlessCommand: "myagent run"},
	}}
	prov, ok = ResolveCLI(cfg, "myagent")
	if !ok || prov.HeadlessCommand != "myagent run" {
		t.Errorf("myagent CLI = %+v, ok=%v, want the project entry", prov, ok)
	}

	// An unknown CLI reports ok=false.
	if _, ok := ResolveCLI(nil, "bogus"); ok {
		t.Error("unknown CLI must report ok=false")
	}
}

// TestResolveCLI_NonClaudeBuiltIns: every non-claude built-in is resolvable
// with NO clis: config at all, carries the command fields that define it,
// ships a full-auto posture on the form that needs one, and never the DEPRECATED
// flat pair (which exists only as a read-time alias for a user's profiles.default).
//
// `wantFills` distinguishes the two fill shapes rather than asserting one of them:
// codex and agy ship per-role fills so a knob pointed at them resolves a real model
// per role, while kimi ships none on purpose (its -m takes a user-config alias).
//
// Every built-in carries interactive launch grammar. CLI first-run walls do
// not remove pane capability: the generic readiness gate classifies them and gives
// the dispatcher bounded judgment rounds before prompt delivery.
func TestResolveCLI_NonClaudeBuiltIns(t *testing.T) {
	cases := []struct {
		name              string
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
		prov, ok := ResolveCLI(nil, c.name)
		if !ok {
			t.Fatalf("built-in %s CLI must resolve with no config", c.name)
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

	// Both agy grammars deliberately omit {effort}: its model IDs embed the reasoning
	// level as an ID suffix, so a separate effort flag would fight the suffix.
	for _, command := range []string{DefaultAgyInteractiveCommand, DefaultAgyHeadlessCommand} {
		if strings.Contains(command, "{effort}") {
			t.Errorf("the agy built-in command %q must carry no {effort} placeholder (its model IDs embed the reasoning level)", command)
		}
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

	// agy is pane-eligible at the same resolved-cli seam the dispatcher reads.
	// Its trust prompt is a readiness-gate concern, not a reason to remove launch
	// grammar from the CLI.
	agy, ok := ResolveCLI(nil, "agy")
	if !ok {
		t.Fatalf("built-in agy CLI must resolve with no config")
	}
	if agy.InteractiveCommand != DefaultAgyInteractiveCommand {
		t.Errorf("agy.InteractiveCommand = %q, want pane-capable grammar %q", agy.InteractiveCommand, DefaultAgyInteractiveCommand)
	}

	// kimi is the same seam read the other way: shipping an interactive_command is
	// exactly what makes it pane-ELIGIBLE, which is the whole point of the probe
	// that closed. Its {model} must stay droppable — kimi ships no fills, so a
	// pinned model here would be a value no install is guaranteed to accept.
	kimi, ok := ResolveCLI(nil, "kimi")
	if !ok {
		t.Fatalf("built-in kimi CLI must resolve with no config")
	}
	if kimi.InteractiveCommand == "" {
		t.Error("kimi must be eligible for pane dispatch — its interactive first run and input echo were probed on 2026-08-10")
	}
	if !strings.Contains(kimi.InteractiveCommand, "-m {model}") {
		t.Errorf("kimi.InteractiveCommand = %q, want the droppable `-m {model}` pair (it ships no fills, so the empty model must remove the flag)", kimi.InteractiveCommand)
	}
}

// TestResolveCLI_ProfilesMerge: the per-role fill map merges over the built-in
// per ROLE and per FIELD — so a config supplying only fills inherits the built-in
// grammar, and a partial fill leaves the other field empty (no cross-field
// invention).
func TestResolveCLI_ProfilesMerge(t *testing.T) {
	cfg := &config.Config{CLIs: map[string]config.CLIConfig{
		"codex": {Profiles: map[string]config.CLIProfile{
			"default": {Model: "gpt-5.3-codex", Effort: "high"},
		}},
	}}
	prov, ok := ResolveCLI(cfg, "codex")
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
	cfg = &config.Config{CLIs: map[string]config.CLIConfig{
		"agy": {Profiles: map[string]config.CLIProfile{
			"default": {Model: "gemini-3.1-pro-low"},
		}},
	}}
	prov, _ = ResolveCLI(cfg, "agy")
	if got := prov.Profiles["default"]; got.Model != "gemini-3.1-pro-low" || got.Effort != "" {
		t.Errorf("partial fill = %+v, want {gemini-3.1-pro-low, \"\"}", got)
	}

	// Overriding one FIELD of one ROLE on claude leaves the rest of that role's
	// built-in fill intact.
	cfg = &config.Config{CLIs: map[string]config.CLIConfig{
		"claude": {Profiles: map[string]config.CLIProfile{
			"doing": {Effort: "medium"},
		}},
	}}
	prov, _ = ResolveCLI(cfg, "claude")
	builtinDoing, _ := DefaultProfile(RoleDoing)
	if got := prov.Profiles[RoleDoing]; got.Effort != "medium" || got.Model != builtinDoing.Model {
		t.Errorf("claude doing fill = %+v, want the overridden effort with the built-in model %q", got, builtinDoing.Model)
	}
}

// TestCLINames: the resolvable CLI set is the union of fab-kit's
// built-in table and the project's clis: block, sorted and de-duplicated —
// the set `fab agent --cli <unknown>` names in its lookup-failure error.
func TestCLINames(t *testing.T) {
	assertNames := func(t *testing.T, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("CLINames = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("CLINames = %v, want %v (sorted, de-duplicated)", got, want)
			}
		}
	}
	assertNames(t, CLINames(nil), []string{"agy", "claude", "codex", "kimi"})

	// Project CLIs union the built-in, de-duplicating shared keys and sorting
	// for stable error output.
	cfg := &config.Config{CLIs: map[string]config.CLIConfig{
		"codex":   {Profiles: map[string]config.CLIProfile{"default": {Model: "gpt-5.3-codex"}}},
		"claude":  {HeadlessCommand: "claude -p"},
		"myagent": {InteractiveCommand: "myagent"},
	}}
	assertNames(t, CLINames(cfg), []string{"agy", "claude", "codex", "kimi", "myagent"})
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
		"opus":                       "opus",  // a bare alias (the built-in fills) passes through
		"sonnet":                     "sonnet",
		"haiku":                      "haiku",
		"fable":                      "fable",
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

// TestIsModelAlias: exactly the four Agent-tool aliases are aliases; versioned
// IDs, bare family IDs, non-Claude IDs and empty are not.
func TestIsModelAlias(t *testing.T) {
	cases := map[string]bool{
		"opus":            true,
		"sonnet":          true,
		"haiku":           true,
		"fable":           true,
		"claude-opus-5-5": false, // versioned ID — ModelAlias maps it first
		"claude-sonnet":   false, // bare family ID is not an alias
		"gpt-5":           false,
		"glm-latest[1m]":  false,
		"":                false,
	}
	for in, want := range cases {
		if got := IsModelAlias(in); got != want {
			t.Errorf("IsModelAlias(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestHasNativeModel(t *testing.T) {
	cases := map[string]bool{
		"":                          true,
		"opus":                      true,
		"sonnet":                    true,
		"haiku":                     true,
		"fable":                     true,
		"claude-opus-5-5":           true,
		"claude-haiku-4-5-20251001": true,
		"glm-latest[1m]":            false,
		"gpt-5.6-sol":               false,
		"auto[1m]":                  false,
	}
	for in, want := range cases {
		if got := HasNativeModel(in); got != want {
			t.Errorf("HasNativeModel(%q) = %v, want %v", in, got, want)
		}
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

// TestBuiltinCLI pins the unmerged-table contract of BuiltinCLI: it
// returns the embedded defaults.yaml entry BEFORE user-override merge (so an
// override present in cfg must NOT leak in), and reports ok=false for names the
// embedded table does not define. This is the lookup the `fab setup check`
// override-masking skew check compares user tiers against — returning merged
// data instead would make a load-bearing override invisible (the #573 shape).
func TestBuiltinCLI(t *testing.T) {
	cfg := &config.Config{
		CLIs: map[string]config.CLIConfig{
			// A user override that MUST NOT appear in BuiltinCLI's answer.
			"claude": {InteractiveCommand: "user-claude-override"},
		},
	}
	_ = cfg // BuiltinCLI takes no config — the override cannot leak by construction.

	builtIn, ok := BuiltinCLI("claude")
	if !ok {
		t.Fatal("BuiltinCLI(claude) ok = false, want true (claude is a built-in)")
	}
	if builtIn.InteractiveCommand != DefaultInteractiveCommand {
		t.Errorf("BuiltinCLI(claude).InteractiveCommand = %q, want the embedded default %q (unmerged)",
			builtIn.InteractiveCommand, DefaultInteractiveCommand)
	}
	if !builtIn.Native {
		t.Error("BuiltinCLI(claude).Native = false, want true (embedded defaults declare it)")
	}

	if _, ok := BuiltinCLI("not-a-cli"); ok {
		t.Error("BuiltinCLI(not-a-cli) ok = true, want false")
	}
}
