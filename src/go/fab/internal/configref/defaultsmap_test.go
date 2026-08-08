package configref

import (
	"testing"

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
	if _, ok := defaults["providers"].(map[string]any); !ok {
		t.Errorf("providers default subtree missing: %#v", defaults["providers"])
	}
	// A row with no built-in default contributes nothing (the empty-default
	// convention — its rendered example is not a default).
	if _, present := defaults["source_paths"]; present {
		t.Errorf("source_paths has no built-in default and must not appear: %#v", defaults["source_paths"])
	}
}

// TestDefaultsMapFor_IsKnobAware: the agent.profiles rows are DERIVED from the
// resolved provider, so composing them against a nil config reports `claude` for
// every role even when a depth knob names another provider. DefaultsMapFor
// composes against the live config instead — the knob-blind drill-down fix.
func TestDefaultsMapFor_IsKnobAware(t *testing.T) {
	nilDefaults, err := DefaultsMapFor(nil)
	if err != nil {
		t.Fatalf("DefaultsMapFor(nil): %v", err)
	}
	if got := roleProvider(t, nilDefaults, "doing"); got != "claude" {
		t.Fatalf("nil-config doing provider = %q, want claude", got)
	}

	cfg := &config.Config{Agent: config.AgentConfig{Workers: "codex"}}
	live, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatalf("DefaultsMapFor(cfg): %v", err)
	}
	// `doing` is a Tier-2 (worker) role, so agent.workers governs it.
	if got := roleProvider(t, live, "doing"); got != "codex" {
		t.Errorf("live doing provider = %q, want codex (the workers knob)", got)
	}
	// `operator` is Tier-1, governed by agent.session, which the config leaves unset.
	if got := roleProvider(t, live, "operator"); got != "claude" {
		t.Errorf("live operator provider = %q, want claude (session knob unset)", got)
	}
}

// TestDefaultsMapFor_UnknownProviderPassesThrough: resolution is provider-neutral
// (Constitution Principle I — fab validates no provider name), so a knob naming a
// provider fab ships nothing for is reported verbatim rather than being hidden
// behind the built-in. `fab config show --origin` stays usable either way.
func TestDefaultsMapFor_UnknownProviderPassesThrough(t *testing.T) {
	cfg := &config.Config{Agent: config.AgentConfig{Workers: "nope-not-a-provider"}}
	defaults, err := DefaultsMapFor(cfg)
	if err != nil {
		t.Fatalf("DefaultsMapFor must not fail on an unknown provider name: %v", err)
	}
	if got := roleProvider(t, defaults, "doing"); got != "nope-not-a-provider" {
		t.Errorf("doing provider = %q, want the configured name reported verbatim", got)
	}
}

// TestDefaultsMapFor_IgnoresUserRoleOverrides: this projection IS the defaults
// tier, so it must report the built-in a user's own agent.profiles entry SHADOWS —
// never echo that entry back. Resolving the live config through agent.ResolveRole
// unfiltered folds the override in, which made `fab config show
// agent.profiles.review.model --origin` print the user's pinned model on the
// `default  (shadowed)` line as well as the project one.
func TestDefaultsMapFor_IgnoresUserRoleOverrides(t *testing.T) {
	builtin, err := DefaultsMapFor(nil)
	if err != nil {
		t.Fatalf("DefaultsMapFor(nil): %v", err)
	}
	pinned := config.RoleProfile{Provider: "pinned-provider", Model: "pinned-model", Effort: "pinned-effort"}
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
			for _, field := range []string{"provider", "model", "effort"} {
				want := roleField(t, builtin, "review", field)
				if have := roleField(t, got, "review", field); have != want {
					t.Errorf("default tier agent.profiles.review.%s = %q, want the built-in %q "+
						"(the user's override belongs to its own tier, not this one)", field, have, want)
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
	if have := roleField(t, got, "review", "provider"); have != "codex" {
		t.Errorf("review provider = %q, want codex (the workers knob still governs the derived row)", have)
	}
}

func roleProvider(t *testing.T, defaults map[string]any, role string) string {
	t.Helper()
	return roleField(t, defaults, role, "provider")
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
