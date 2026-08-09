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

// TestDefaultsFileProviders: the providers block covers exactly the four
// built-ins, with the command fields each one is defined by. claude, codex and agy
// carry per-role fills (260806-ywkx — a knob pointed at one must resolve a real
// model per role, not an empty one), every fill key names a known role, and no
// built-in uses the DEPRECATED flat fill.
//
// kimi is the deliberate exception (260808-rpsr) and is asserted as such rather
// than merely exempted: its -m takes a USER-CONFIG model alias, not a catalog ID,
// so shipping any fill would break non-managed installs. Empty is the correct
// built-in there, and the empty-{model} token-drop is what makes it work.
//
// agy and kimi are the same kind of deliberate exception on the SESSION command:
// they ship none, so both absences are asserted rather than exempted.
func TestDefaultsFileProviders(t *testing.T) {
	cfg := parseDefaultsFile(t)

	wantProviders := []string{DefaultProviderName, providerCodex, providerAgy, providerKimi}
	assertSameKeys(t, "providers", providerKeys(cfg.Providers), wantProviders)

	for _, name := range wantProviders {
		prov, ok := cfg.Providers[name]
		if !ok {
			continue // already reported by assertSameKeys
		}
		if prov.Model != "" || prov.Effort != "" {
			t.Errorf("defaults.yaml providers.%s uses the DEPRECATED flat fill (%q/%q) — built-in fills belong under profiles.<role>", name, prov.Model, prov.Effort)
		}
		// A provider's `default` entry is its cross-role fallback, so a sparse map
		// is only well-defined when that entry exists and names a model. kimi ships
		// no map at all, which is asserted separately below.
		if name != providerKimi && prov.Profiles[RoleDefault].Model == "" {
			t.Errorf("defaults.yaml providers.%s has no profiles.default.model — a filled built-in must resolve a model for every role, and `default` is the cross-role fallback", name)
		}
		for role := range prov.Profiles {
			if !IsRoleName(role) {
				t.Errorf("defaults.yaml providers.%s.profiles.%s is not a known role (valid: %s)", name, role, strings.Join(RoleNames(), ", "))
			}
		}
	}

	// kimi ships NO fills — the deliberate no-fills built-in. Asserting the absence
	// (rather than skipping kimi) is what keeps a well-meaning pinned `k3` row from
	// landing unreviewed and breaking every custom-provider install.
	if got := cfg.Providers[providerKimi].Profiles; len(got) != 0 {
		t.Errorf("defaults.yaml providers.kimi.profiles = %v, want none — kimi's -m takes a USER-CONFIG model alias, so any pinned fill breaks non-managed installs", got)
	}

	// agy's fills carry NO effort: its model IDs EMBED the reasoning level
	// (gemini-3.1-pro-high), which is the same reason its command grammars carry no
	// {effort} placeholder — a separate effort flag would fight the suffix.
	for role, fill := range cfg.Providers[providerAgy].Profiles {
		if fill.Effort != "" {
			t.Errorf("defaults.yaml providers.agy.profiles.%s sets effort=%q — agy's model IDs embed the reasoning level instead", role, fill.Effort)
		}
	}

	// Capabilities are explicit data: claude supports native plus both command
	// forms; codex/agy/kimi support command forms but not native dispatch.
	claude := cfg.Providers[DefaultProviderName]
	if !claude.Native {
		t.Errorf("defaults.yaml providers.%s.native = false, want true", DefaultProviderName)
	}
	if claude.DispatchCommand == "" {
		t.Errorf("defaults.yaml providers.%s has no dispatch_command", DefaultProviderName)
	}
	for _, name := range []string{providerCodex, providerAgy, providerKimi} {
		if cfg.Providers[name].DispatchCommand == "" {
			t.Errorf("defaults.yaml providers.%s has no dispatch_command", name)
		}
		if cfg.Providers[name].Native {
			t.Errorf("defaults.yaml providers.%s.native = true, want false", name)
		}
	}

	// The mirror image: only claude and codex ship a session_command. agy and kimi
	// are DISPATCH-ONLY built-ins, and the absence is load-bearing rather than an
	// omission — a session_command makes a provider eligible for pane-mode dispatch,
	// which hands the worker its pointer prompt as a POSITIONAL argument, and neither
	// CLI can receive a prompt that way (kimi reads a bare positional as a subcommand
	// and exits non-zero; agy drops it silently and trust-prompts a fresh workspace).
	// Asserting the absence is what stops a well-meaning session_command from landing
	// and parking every tmux-dispatched stage at an empty prompt.
	for _, name := range []string{DefaultProviderName, providerCodex} {
		if cfg.Providers[name].SessionCommand == "" {
			t.Errorf("defaults.yaml providers.%s has no session_command", name)
		}
	}
	for _, name := range []string{providerAgy, providerKimi} {
		if got := cfg.Providers[name].SessionCommand; got != "" {
			t.Errorf("defaults.yaml providers.%s.session_command = %q, want absent — %s cannot receive a pane worker's pointer prompt as a positional argument, so shipping one would select pane dispatch and park the stage", name, got, name)
		}
	}

	// agy's and kimi's no-{effort} / nested-shell grammar is asserted once, over
	// these same embedded bytes, by TestResolveProvider_NonClaudeBuiltIns.
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
		// agy and kimi are dispatch-only, so they export no session-command var —
		// their session_command ABSENCE is asserted in TestDefaultsFileProviders.
		{"DefaultAgyDispatchCommand", DefaultAgyDispatchCommand, cfg.Providers[providerAgy].DispatchCommand},
		{"DefaultKimiDispatchCommand", DefaultKimiDispatchCommand, cfg.Providers[providerKimi].DispatchCommand},
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
