package agent

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// The built-in defaults moved from Go literals into the embedded defaults.yaml
// (260806-2j2i), so a typo in them is no longer a compile error. This file is the
// replacement safety net: it re-parses the embedded bytes INDEPENDENTLY of the
// package's own initialization and asserts both the file's shape (every knob,
// provider, and per-role fill present and populated, nothing outside the surface it
// is meant to define) and the wiring (defaultProviders and the exported command
// vars read the keys they claim to).
//
// It deliberately pins NO model IDs or command strings — those are pinned once,
// in TestDefaultRoleProfilesArePinned, so a model bump stays a one-line edit to
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

// TestDefaultsFileIsWellFormed: the embedded file parses, both depth knobs are set,
// and claude's per-role fill map covers exactly the six known roles with every
// field populated. (260806-j9nh: the role models live on the PROVIDER now, so the
// file defines no agent.profiles block at all — the knobs plus the provider fills
// are the whole built-in surface.)
func TestDefaultsFileIsWellFormed(t *testing.T) {
	cfg := parseDefaultsFile(t)

	if cfg.Agent.Session == "" || cfg.Agent.Workers == "" {
		t.Errorf("defaults.yaml must set both depth knobs, got session=%q workers=%q", cfg.Agent.Session, cfg.Agent.Workers)
	}
	if len(cfg.Agent.Profiles) != 0 || len(cfg.Agent.Tiers) != 0 {
		t.Errorf("defaults.yaml must define no agent.profiles/agent.tiers block — the built-in role models live on providers.<name>.profiles; got %v / %v", cfg.Agent.Profiles, cfg.Agent.Tiers)
	}

	wantRoles := []string{RoleDefault, RoleOperator, RoleDoing, RoleReview, RoleHydrate, RoleFast}
	claude := cfg.Providers[cfg.Agent.Session]
	assertSameKeys(t, "providers."+cfg.Agent.Session+".profiles", roleKeys(claude.Profiles), wantRoles)

	for _, role := range wantRoles {
		fill, ok := claude.Profiles[role]
		if !ok {
			continue // already reported by assertSameKeys
		}
		if fill.Model == "" {
			t.Errorf("defaults.yaml providers.claude.profiles.%s has no model — an empty model is the inherit-the-session-model signal, never a built-in default", role)
		}
		if fill.Effort == "" {
			t.Errorf("defaults.yaml providers.claude.profiles.%s has no effort", role)
		}
	}
}

// TestDefaultsFileProviders: the providers block covers exactly the three
// built-ins, with the command fields each one is defined by. ALL THREE carry
// per-role fills (260806-ywkx — a knob pointed at codex/gemini must resolve a real
// model per role, not an empty one), every fill key names a known role, and no
// built-in uses the DEPRECATED flat fill.
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
			t.Errorf("defaults.yaml providers.%s uses the DEPRECATED flat fill (%q/%q) — built-in fills belong under profiles.<role>", name, prov.Model, prov.Effort)
		}
		// A provider's `default` entry is its cross-role fallback, so a sparse map
		// is only well-defined when that entry exists and names a model.
		if prov.Profiles[RoleDefault].Model == "" {
			t.Errorf("defaults.yaml providers.%s has no profiles.default.model — every built-in must resolve a model for every role, and `default` is the cross-role fallback", name)
		}
		for role := range prov.Profiles {
			if !IsRoleName(role) {
				t.Errorf("defaults.yaml providers.%s.profiles.%s is not a known role (valid: %s)", name, role, strings.Join(RoleNames(), ", "))
			}
		}
	}

	// gemini's fills carry NO effort: the gemini CLI has no reasoning-effort flag
	// (the same reason its command grammars carry no {effort} placeholder), so a
	// resolved effort would have nowhere to go.
	for role, fill := range cfg.Providers[providerGemini].Profiles {
		if fill.Effort != "" {
			t.Errorf("defaults.yaml providers.gemini.profiles.%s sets effort=%q — the gemini CLI has no reasoning-effort flag", role, fill.Effort)
		}
	}

	// Capabilities are explicit data: claude supports native plus both command
	// forms; codex/gemini support the command forms but not native dispatch.
	claude := cfg.Providers[DefaultProviderName]
	if !claude.Native {
		t.Errorf("defaults.yaml providers.%s.native = false, want true", DefaultProviderName)
	}
	if claude.DispatchCommand == "" {
		t.Errorf("defaults.yaml providers.%s has no dispatch_command", DefaultProviderName)
	}
	for _, name := range []string{providerCodex, providerGemini} {
		if cfg.Providers[name].DispatchCommand == "" {
			t.Errorf("defaults.yaml providers.%s has no dispatch_command", name)
		}
		if cfg.Providers[name].Native {
			t.Errorf("defaults.yaml providers.%s.native = true, want false", name)
		}
	}

	// gemini's no-{effort} / no-`-p` grammar is asserted once, over these same
	// embedded bytes, by TestResolveProvider_BuiltInCodexAndGemini.
}

// TestDefaultsFileDefinesOnlyItsSurface: defaults.yaml is layer 0 of the config
// cascade in shape, but it defines only the two blocks internal/agent owns — and
// within `agent:`, only the two depth knobs (the role→depth partition and the
// stage→role mapping are fab-owned POLICY and stay in Go). yaml.v3 ignores unknown
// keys, so a key written at the wrong nesting level (or a block this package does
// not read) would otherwise be silently inert.
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
	assertSameKeys(t, "defaults.yaml agent", agentKeys, []string{"session", "workers"})
}

// TestPackageTablesMatchDefaultsFile: the package's tables and exported command
// vars are wired to the file's keys. Without this, a lookup pointed at the wrong
// provider name would resolve to the zero value and every other test — all of
// which derive their expectations from these same tables — would stay green.
func TestPackageTablesMatchDefaultsFile(t *testing.T) {
	cfg := parseDefaultsFile(t)

	// Each role's built-in profile is the knob's provider plus that provider's own
	// per-role fill — the file's two blocks composed exactly as ResolveRole does.
	for _, role := range RoleNames() {
		wantProvider := cfg.Agent.Workers
		if roleDepth[role] == depthSession {
			wantProvider = cfg.Agent.Session
		}
		fill := cfg.Providers[wantProvider].Profiles[role]
		want := Profile{Provider: wantProvider, Model: fill.Model, Effort: fill.Effort}
		if got, ok := DefaultProfile(role); !ok || got != want {
			t.Errorf("DefaultProfile(%q) = %+v (ok=%v), defaults.yaml composes %+v", role, got, ok, want)
		}
	}

	for name, want := range cfg.Providers {
		got, ok := ResolveProvider(nil, name)
		if !ok {
			t.Errorf("ResolveProvider(nil, %q) reports unknown, but defaults.yaml defines it", name)
			continue
		}
		// The resolved Profiles map is a defensive copy, so compare by value.
		if got.SessionCommand != want.SessionCommand || got.DispatchCommand != want.DispatchCommand || got.Native != want.Native ||
			got.Model != want.Model || got.Effort != want.Effort ||
			!reflect.DeepEqual(got.Profiles, want.Profiles) {
			t.Errorf("ResolveProvider(nil, %q) = %+v, defaults.yaml says %+v", name, got, want)
		}
	}

	commands := []struct {
		name string
		got  string
		want string
	}{
		{"DefaultSessionCommand", DefaultSessionCommand, cfg.Providers[DefaultProviderName].SessionCommand},
		{"DefaultDispatchCommand", DefaultDispatchCommand, cfg.Providers[DefaultProviderName].DispatchCommand},
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

func roleKeys(profiles map[string]config.ProviderProfile) []string {
	keys := make([]string, 0, len(profiles))
	for name := range profiles {
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
