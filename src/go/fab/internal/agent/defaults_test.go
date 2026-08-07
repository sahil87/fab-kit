package agent

import (
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// The built-in defaults moved from Go literals into the embedded defaults.yaml
// (260806-2j2i), so a typo in them is no longer a compile error. This file is the
// replacement safety net: it re-parses the embedded bytes INDEPENDENTLY of the
// package's own initialization and asserts both the file's shape (every tier and
// provider present and populated, nothing outside the surface it is meant to
// define) and the wiring (defaultTiers/defaultProviders and the exported command
// vars read the keys they claim to).
//
// It deliberately pins NO model IDs or command strings — those are pinned once,
// in TestDefaultTierProfilesArePinned, so a model bump stays a one-line edit to
// defaults.yaml plus that one table.

// parseDefaultsFile unmarshals the embedded bytes into the config schema, the
// same way the package initializer does — but as a fresh parse the assertions can
// compare against, so a mis-wired var cannot make its own check pass.
func parseDefaultsFile(t *testing.T) config.Config {
	t.Helper()
	var cfg config.Config
	if err := yaml.Unmarshal(defaultsYAML, &cfg); err != nil {
		t.Fatalf("embedded defaults.yaml does not parse: %v", err)
	}
	return cfg
}

// TestDefaultsFileIsWellFormed: the embedded file parses, and its agent.tiers
// block covers exactly the six known tiers with every field populated.
func TestDefaultsFileIsWellFormed(t *testing.T) {
	cfg := parseDefaultsFile(t)

	wantTiers := []string{TierDefault, TierOperator, TierDoing, TierReview, TierHydrate, TierFast}
	assertSameKeys(t, "agent.tiers", tierKeys(cfg.Agent.Tiers), wantTiers)

	for _, tier := range wantTiers {
		profile, ok := cfg.Agent.Tiers[tier]
		if !ok {
			continue // already reported by assertSameKeys
		}
		if profile.Provider == "" {
			t.Errorf("defaults.yaml agent.tiers.%s has no provider — every built-in tier writes it explicitly", tier)
		}
		if profile.Model == "" {
			t.Errorf("defaults.yaml agent.tiers.%s has no model — an empty model is the inherit-the-session-model signal, never a built-in default", tier)
		}
		if profile.Effort == "" {
			t.Errorf("defaults.yaml agent.tiers.%s has no effort", tier)
		}
	}
}

// TestDefaultsFileProviders: the providers block covers exactly the three
// built-ins, with the command fields each one is defined by — and with NO
// model/effort fill on any of them (grammar only; model IDs rot at CLI cadence
// and belong in user config).
func TestDefaultsFileProviders(t *testing.T) {
	cfg := parseDefaultsFile(t)

	wantProviders := []string{DefaultProviderName, providerCodex, providerGemini}
	assertSameKeys(t, "providers", providerKeys(cfg.Providers), wantProviders)

	for _, name := range wantProviders {
		prov, ok := cfg.Providers[name]
		if !ok {
			continue // already reported by assertSameKeys
		}
		if prov.SessionCommand == "" {
			t.Errorf("defaults.yaml providers.%s has no session_command", name)
		}
		if prov.Model != "" || prov.Effort != "" {
			t.Errorf("defaults.yaml providers.%s carries a model/effort fill (%q/%q) — built-ins ship GRAMMAR ONLY; fill belongs in user config", name, prov.Model, prov.Effort)
		}
	}

	// claude's ABSENT dispatch_command is what selects native Agent-tool dispatch;
	// codex/gemini carry one, which is what flips their tiers to CLI dispatch.
	if got := cfg.Providers[DefaultProviderName].DispatchCommand; got != "" {
		t.Errorf("defaults.yaml providers.%s.dispatch_command = %q, want absent — its absence is the native-dispatch signal", DefaultProviderName, got)
	}
	for _, name := range []string{providerCodex, providerGemini} {
		if cfg.Providers[name].DispatchCommand == "" {
			t.Errorf("defaults.yaml providers.%s has no dispatch_command — naming it in a tier must select CLI dispatch", name)
		}
	}

	// gemini's no-{effort} / no-`-p` grammar is asserted once, over these same
	// embedded bytes, by TestResolveProvider_BuiltInCodexAndGemini.
}

// TestDefaultsFileDefinesOnlyItsSurface: defaults.yaml is layer 0 of the config
// cascade in shape, but it defines only the two blocks internal/agent owns. yaml.v3
// ignores unknown keys, so a key written at the wrong nesting level (or a block
// this package does not read) would otherwise be silently inert.
func TestDefaultsFileDefinesOnlyItsSurface(t *testing.T) {
	var raw map[string]yaml.Node
	if err := yaml.Unmarshal(defaultsYAML, &raw); err != nil {
		t.Fatalf("embedded defaults.yaml does not parse: %v", err)
	}

	top := make([]string, 0, len(raw))
	for key := range raw {
		top = append(top, key)
	}
	assertSameKeys(t, "defaults.yaml top-level", top, []string{"providers", "agent"})

	var agentBlock map[string]yaml.Node
	node, ok := raw["agent"]
	if !ok {
		return // already reported by assertSameKeys
	}
	if err := node.Decode(&agentBlock); err != nil {
		t.Fatalf("defaults.yaml agent block does not decode: %v", err)
	}
	agentKeys := make([]string, 0, len(agentBlock))
	for key := range agentBlock {
		agentKeys = append(agentKeys, key)
	}
	assertSameKeys(t, "defaults.yaml agent", agentKeys, []string{"tiers"})
}

// TestPackageTablesMatchDefaultsFile: the package's tables and exported command
// vars are wired to the file's keys. Without this, a lookup pointed at the wrong
// provider name would resolve to the zero value and every other test — all of
// which derive their expectations from these same tables — would stay green.
func TestPackageTablesMatchDefaultsFile(t *testing.T) {
	cfg := parseDefaultsFile(t)

	for tier, want := range tierProfiles(cfg.Agent.Tiers) {
		if got, ok := DefaultTier(tier); !ok || got != want {
			t.Errorf("DefaultTier(%q) = %+v (ok=%v), defaults.yaml says %+v", tier, got, ok, want)
		}
	}

	for name, want := range cfg.Providers {
		got, ok := ResolveProvider(nil, name)
		if !ok || got != want {
			t.Errorf("ResolveProvider(nil, %q) = %+v (ok=%v), defaults.yaml says %+v", name, got, ok, want)
		}
	}

	commands := []struct {
		name string
		got  string
		want string
	}{
		{"DefaultSessionCommand", DefaultSessionCommand, cfg.Providers[DefaultProviderName].SessionCommand},
		{"DefaultCodexSessionCommand", DefaultCodexSessionCommand, cfg.Providers[providerCodex].SessionCommand},
		{"DefaultCodexDispatchCommand", DefaultCodexDispatchCommand, cfg.Providers[providerCodex].DispatchCommand},
		{"DefaultGeminiSessionCommand", DefaultGeminiSessionCommand, cfg.Providers[providerGemini].SessionCommand},
		{"DefaultGeminiDispatchCommand", DefaultGeminiDispatchCommand, cfg.Providers[providerGemini].DispatchCommand},
	}
	for _, c := range commands {
		if c.got != c.want {
			t.Errorf("%s = %q, defaults.yaml says %q", c.name, c.got, c.want)
		}
	}
}

func tierKeys(tiers map[string]config.TierProfile) []string {
	keys := make([]string, 0, len(tiers))
	for name := range tiers {
		keys = append(keys, name)
	}
	return keys
}

func providerKeys(providers map[string]config.ProviderConfig) []string {
	keys := make([]string, 0, len(providers))
	for name := range providers {
		keys = append(keys, name)
	}
	return keys
}

// assertSameKeys fails if got and want are not the same set (order-independent),
// naming both sides so a dropped or misspelled key is readable from the failure.
func assertSameKeys(t *testing.T, what string, got, want []string) {
	t.Helper()
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	if strings.Join(g, ",") != strings.Join(w, ",") {
		t.Errorf("%s covers %v, want exactly %v", what, g, w)
	}
}
