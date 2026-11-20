package configref

import (
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// TestDefaultsMap_ProjectsEveryDefaultBearingRow: the projection nests each
// non-nil row Default under its dotted key and merges the subtrees, so a
// top-level key documented by SEVERAL rows (agent: session + workers + profiles)
// contributes all of them rather than stopping at the first.
func TestDefaultsMap_ProjectsEveryDefaultBearingRow(t *testing.T) {
	defaults, err := DefaultsMap()
	if err != nil {
		t.Fatalf("DefaultsMap: %v", err)
	}
	agentMap, ok := defaults["agent"].(map[string]any)
	if !ok {
		t.Fatalf("agent subtree missing: %#v", defaults["agent"])
	}
	for _, knob := range []string{"session", "workers"} {
		if agentMap[knob] != "claude" {
			t.Errorf("agent.%s default = %#v, want claude", knob, agentMap[knob])
		}
	}
	if _, ok := agentMap["profiles"].(map[string]any); !ok {
		t.Errorf("agent.profiles default subtree missing: %#v", agentMap["profiles"])
	}
	if _, ok := defaults["clis"].(map[string]any); !ok {
		t.Errorf("CLIs default subtree missing: %#v", defaults["clis"])
	}
	// A row with no built-in default contributes nothing (the empty-default
	// convention — its rendered example is not a default).
	if _, present := defaults["source_paths"]; present {
		t.Errorf("source_paths has no built-in default and must not appear: %#v", defaults["source_paths"])
	}
}

// TestDefaultsMapFor_IsKnobAware: the agent.profiles rows are DERIVED from the
// resolved CLI, so composing them against a nil config reports `claude` for
// every role even when a depth knob names another CLI. DefaultsMapFor
// composes against the live config instead — the knob-blind drill-down fix.
func TestDefaultsMapFor_IsKnobAware(t *testing.T) {
	nilDefaults, err := DefaultsMapFor(nil)
	if err != nil {
		t.Fatalf("DefaultsMapFor(nil): %v", err)
	}
	if got := roleCLI(t, nilDefaults, "doing"); got != "claude" {
		t.Fatalf("nil-config doing CLI = %q, want claude", got)
	}

	cfg := &config.Config{Agent: config.AgentConfig{Workers: "codex"}}
	live, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatalf("DefaultsMapFor(cfg): %v", err)
	}
	// `doing` is a Tier-2 (worker) role, so agent.workers governs it.
	if got := roleCLI(t, live, "doing"); got != "codex" {
		t.Errorf("live doing CLI = %q, want codex (the workers knob)", got)
	}
	// `operator` is Tier-1, governed by agent.session, which the config leaves unset.
	if got := roleCLI(t, live, "operator"); got != "claude" {
		t.Errorf("live operator CLI = %q, want claude (session knob unset)", got)
	}
}

// TestDefaultsMapFor_UnknownCLIPassesThrough: resolution is CLI-neutral
// (Constitution Principle I — fab validates no CLI name), so a knob naming a
// CLI fab ships nothing for is reported verbatim rather than being hidden
// behind the built-in. `fab config show --origin` stays usable either way.
func TestDefaultsMapFor_UnknownCLIPassesThrough(t *testing.T) {
	cfg := &config.Config{Agent: config.AgentConfig{Workers: "nope-not-a-cli"}}
	defaults, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatalf("DefaultsMapFor must not fail on an unknown CLI name: %v", err)
	}
	if got := roleCLI(t, defaults, "doing"); got != "nope-not-a-cli" {
		t.Errorf("doing CLI = %q, want the configured name reported verbatim", got)
	}
}

// TestDefaultsMapFor_IgnoresUserRoleOverrides: this projection IS the defaults
// tier, so it must never echo a user's own agent.profiles entry back. Resolving
// the live config through agent.ResolveRole unfiltered folds the override in,
// which made `fab config show agent.profiles.review.model --origin` print the
// user's pinned model on the `default  (shadowed)` line as well as the project
// one. The pinned CLI is kept for the model/effort derivation (see
// TestDefaultsMapFor_RoleCLIOverrideDrivesFills) — here that CLI has no
// fills, so the derived model/effort are empty, and in particular never the
// pinned values. The CLI leaf itself stays knobs-only-resolved.
func TestDefaultsMapFor_IgnoresUserRoleOverrides(t *testing.T) {
	builtin, err := DefaultsMapFor(nil)
	if err != nil {
		t.Fatalf("DefaultsMapFor(nil): %v", err)
	}
	pinned := config.RoleProfile{CLI: "pinned-cli", Model: "pinned-model", Effort: "pinned-effort"}
	for _, tc := range []struct {
		name string
		cfg  *config.Config
	}{
		{"agent.profiles", &config.Config{Agent: config.AgentConfig{
			Profiles: map[string]config.RoleProfile{"review": pinned}}}},
		{"agent.tiers (the pre-2.17.0 spelling)", &config.Config{Agent: config.AgentConfig{
			Tiers: map[string]config.RoleProfile{"review": pinned}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DefaultsMapFor(tc.cfg)
			if err != nil {
				t.Fatalf("DefaultsMapFor: %v", err)
			}
			if want := roleField(t, builtin, "review", "cli"); roleField(t, got, "review", "cli") != want {
				t.Errorf("default tier agent.profiles.review.cli = %q, want the built-in %q "+
					"(the CLI leaf stays knobs-only-resolved)", roleField(t, got, "review", "cli"), want)
			}
			for field, pinnedValue := range map[string]string{"model": pinned.Model, "effort": pinned.Effort} {
				have := roleField(t, got, "review", field)
				if have == pinnedValue {
					t.Errorf("default tier agent.profiles.review.%s = %q — echoes the user's pinned override "+
						"(that value belongs to its own tier, not this one)", field, have)
				}
				if have != "" {
					t.Errorf("default tier agent.profiles.review.%s = %q, want empty "+
						"(pinned-cli ships no fills to derive from)", field, have)
				}
			}
		})
	}

	// The knobs still compose: an override on ONE role leaves the depth knob
	// governing the derived rows (this is R5, and it must survive the stripping).
	cfg := &config.Config{Agent: config.AgentConfig{
		Workers:  "codex",
		Profiles: map[string]config.RoleProfile{"review": pinned},
	}}
	got, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatalf("DefaultsMapFor: %v", err)
	}
	if have := roleField(t, got, "review", "cli"); have != "codex" {
		t.Errorf("review CLI = %q, want codex (the workers knob still governs the derived row)", have)
	}
}

// TestDefaultsMapFor_RoleCLIOverrideDrivesFills: the chimera regression. A
// per-role CLI override (agent.profiles.operator.cli: codex under a
// claude session knob) used to be stripped with the rest of the entry, so the
// derived model/effort came from CLAUDE's operator fill while the CLI leaf
// merged to codex — a composed row no resolution path produces. The override must
// drive the fills derivation (same values the knob naming that CLI would
// derive — the oracle below), while the CLI leaf itself stays knobs-only.
func TestDefaultsMapFor_RoleCLIOverrideDrivesFills(t *testing.T) {
	knobbed, err := DefaultsMapFor(&config.Config{Agent: config.AgentConfig{Session: "codex"}})
	if err != nil {
		t.Fatalf("DefaultsMapFor(session knob): %v", err)
	}
	wantModel := roleField(t, knobbed, "operator", "model")
	wantEffort := roleField(t, knobbed, "operator", "effort")
	if wantModel == "" || wantEffort == "" {
		t.Fatalf("oracle broke: codex ships no operator fill (model %q, effort %q)", wantModel, wantEffort)
	}

	override := map[string]config.RoleProfile{"operator": {CLI: "codex"}}
	for _, tc := range []struct {
		name string
		cfg  *config.Config
	}{
		{"agent.profiles", &config.Config{Agent: config.AgentConfig{Session: "claude", Profiles: override}}},
		{"agent.tiers (the pre-2.17.0 spelling)", &config.Config{Agent: config.AgentConfig{Session: "claude", Tiers: override}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DefaultsMapFor(tc.cfg)
			if err != nil {
				t.Fatalf("DefaultsMapFor: %v", err)
			}
			if have := roleCLI(t, got, "operator"); have != "claude" {
				t.Errorf("CLI leaf = %q, want claude (knobs-only — the override's codex belongs to its own tier)", have)
			}
			if have := roleField(t, got, "operator", "model"); have != wantModel {
				t.Errorf("model = %q, want %q (the overridden CLI's fill, not the knob CLI's)", have, wantModel)
			}
			if have := roleField(t, got, "operator", "effort"); have != wantEffort {
				t.Errorf("effort = %q, want %q (the overridden CLI's fill, not the knob CLI's)", have, wantEffort)
			}
		})
	}
}

func roleCLI(t *testing.T, defaults map[string]any, role string) string {
	t.Helper()
	return roleField(t, defaults, role, "cli")
}

func roleField(t *testing.T, defaults map[string]any, role, field string) string {
	t.Helper()
	agentMap, ok := defaults["agent"].(map[string]any)
	if !ok {
		t.Fatalf("agent subtree missing: %#v", defaults["agent"])
	}
	profiles, ok := agentMap["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("agent.profiles subtree missing: %#v", agentMap["profiles"])
	}
	entry, ok := profiles[role].(map[string]any)
	if !ok {
		t.Fatalf("agent.profiles.%s missing: %#v", role, profiles[role])
	}
	value, _ := entry[field].(string)
	return value
}

// TestDerivedRoleSources: each derived default-tier leaf reports the config key it
// was derived from — the depth knob for the CLI, the CLI fill for
// model/effort — and "" when only built-ins were involved.
func TestDerivedRoleSources(t *testing.T) {
	t.Run("nil config is all built-in CLI, fills still named", func(t *testing.T) {
		sources, err := DerivedRoleSources(nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := sources["doing"].CLI; got != "" {
			t.Errorf("built-in CLI must have no source key, got %q", got)
		}
	})

	t.Run("knob and fills", func(t *testing.T) {
		cfg := &config.Config{Agent: config.AgentConfig{Workers: "agy", Session: "codex"}}
		sources, err := DerivedRoleSources(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if got := sources["doing"].CLI; got != "agent.workers" {
			t.Errorf("doing CLI source = %q, want agent.workers", got)
		}
		if got := sources["operator"].CLI; got != "agent.session" {
			t.Errorf("operator CLI source = %q, want agent.session", got)
		}
		if got := sources["doing"].Model; got != "clis.agy.profiles.doing.model" && got != "clis.agy.profiles.default.model" {
			t.Errorf("doing model source = %q, want a clis.agy.profiles.*.model path", got)
		}
	})

	t.Run("per-role CLI override is stripped from the CLI leaf", func(t *testing.T) {
		cfg := &config.Config{Agent: config.AgentConfig{
			Workers:  "agy",
			Profiles: map[string]config.RoleProfile{"review": {CLI: "codex"}},
		}}
		sources, err := DerivedRoleSources(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if got := sources["review"].CLI; got != "agent.workers" {
			t.Errorf("CLI leaf derives from the knob, not the override, got %q", got)
		}
	})
}

// TestDefaultsMapFor_ToolDefaultFillKeepsToken: a CLI fill holding the
// reserved tool-default value resolves to an empty model, but the derived
// default tier still shows the token and names the key that holds it — an empty
// derived model would read as "no tier defines this key".
func TestDefaultsMapFor_ToolDefaultFillKeepsToken(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{"review": {CLI: "codex"}}},
		CLIs: map[string]config.CLIConfig{"codex": {Profiles: map[string]config.CLIProfile{
			"review": {Model: agent.ToolDefaultModel},
		}}},
	}
	defaults, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := roleField(t, defaults, "review", "model"); got != agent.ToolDefaultModel {
		t.Errorf("derived review model = %q, want %q", got, agent.ToolDefaultModel)
	}
	sources, err := DerivedRoleSources(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := sources["review"].Model; got != "clis.codex.profiles.review.model" {
		t.Errorf("review model source = %q, want clis.codex.profiles.review.model", got)
	}
}
