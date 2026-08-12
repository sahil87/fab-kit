package configref

import (
	"encoding/json"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// DefaultsMap projects the registry's canonical defaults into ONE YAML-shaped
// map — the materialized bottom tier of the config read model. Each row whose
// Default is non-nil is nested under its dotted key (agent.session → agent →
// session) and the resulting subtrees are merged, so a top-level key carrying
// several default-bearing rows contributes all of them.
//
// It is the projection `fab config show`, `--origin`, and the set/unset notices
// merge beneath the file and environment layers. The LOADER does not consume it:
// internal/config cannot import this package (configref → agent → config would
// close an import cycle), and the registry's agent.profiles default is a DERIVED
// value — merging it into the loader's tree would land a full {provider, model,
// effort} in Config.Agent.Profiles, which the resolver reads as a user override
// outranking the provider's own fills. Built-in values reach consumers through
// the retained point-of-use seams instead; this map is the read model's view of
// that same tier.
func DefaultsMap() (map[string]any, error) {
	fields, err := Fields()
	if err != nil {
		return nil, err
	}
	return defaultsMapFromFields(fields), nil
}

// DefaultsMapFor is DefaultsMap with the DERIVED per-role rows resolved against
// the LIVE config instead of against a nil one. The agent.profiles default is
// composed from agent.DefaultProfile — resolution against a nil *Config — so
// every role's provider row reads `claude` even when a depth knob names another
// provider; composing against cfg reports the provider (and its fills) honestly.
//
// The composition never echoes a tier above it — this map is the DEFAULTS tier,
// so a user's per-role model/effort overrides are stripped before resolving (an
// `agent.profiles.review.model` override would otherwise be reported as its own
// built-in, on the `default (shadowed)` line as well as the project one). A
// per-role PROVIDER override is different: it is KEPT for the model/effort
// derivation, because the built-in fill for a role is a function of the provider
// the role actually dispatches to — the same honesty argument as the depth knobs,
// and no echo, since the derived values come from that provider's own fill map,
// which no higher tier states. The provider leaf's own default stays knobs-only
// so keyed --origin still shows the override shadowing the knob's provider.
//
// Resolution stays provider-neutral: a knob naming a provider fab knows nothing
// about passes through verbatim (Constitution Principle I — fab validates no
// provider name), which is the honest report of what that role would dispatch to.
//
// A nil cfg yields the same map as DefaultsMap.
func DefaultsMapFor(cfg *config.Config) (map[string]any, error) {
	fields, err := Fields()
	if err != nil {
		return nil, err
	}
	live, err := liveRoleProfiles(cfg)
	if err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].Key == agentProfilesKey {
			fields[i].Default = live
		}
	}
	return defaultsMapFromFields(fields), nil
}

// agentProfilesKey is the one registry row whose Default is DERIVED from the
// resolved provider fills rather than being a literal built-in, and therefore the
// one row DefaultsMapFor recomposes.
const agentProfilesKey = "agent.profiles"

// liveRoleProfiles resolves every role through agent.ResolveRole twice — the two
// leaves have different honesty requirements. The PROVIDER leaf resolves against
// knobsOnly (per-role overrides fully stripped): it must report the built-in a
// user's provider override SHADOWS. The MODEL/EFFORT leaves resolve against
// fillsOnly (per-role provider overrides kept, model/effort cleared): the
// built-in fill is a function of the provider the role actually dispatches to,
// so deriving it from the knob's provider when an override redirects the role
// composes a chimera row no resolution path produces. Like roleRows it fails
// loud on role-map drift: a role agent.RoleNames reports must resolve (an
// unknown role name is ResolveRole's only error).
func liveRoleProfiles(cfg *config.Config) (map[string]roleProfileDefault, error) {
	names := agent.RoleNames()
	knobs := knobsOnly(cfg)
	fills := fillsOnly(cfg)
	out := make(map[string]roleProfileDefault, len(names))
	for _, name := range names {
		kp, err := agent.ResolveRole(knobs, name)
		if err != nil {
			return nil, err
		}
		fp, err := agent.ResolveRole(fills, name)
		if err != nil {
			return nil, err
		}
		out[name] = roleProfileDefault{Provider: kp.Provider, Model: fp.Model, Effort: fp.Effort}
	}
	return out, nil
}

// knobsOnly copies cfg with the per-role override maps (agent.profiles and its
// pre-2.17.0 spelling agent.tiers) CLEARED, leaving the depth knobs and the
// providers table — the inputs the derived PROVIDER leaf legitimately depends
// on. The copy is shallow, and neither the original nor its remaining maps are
// mutated.
func knobsOnly(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}
	stripped := *cfg
	stripped.Agent.Profiles = nil
	stripped.Agent.Tiers = nil
	return &stripped
}

// fillsOnly copies cfg with each per-role override reduced to its {provider}
// field (model/effort cleared, both the agent.profiles and legacy agent.tiers
// spellings) — the input set the derived MODEL/EFFORT leaves depend on: the
// provider selection (knob or per-role override) and the providers table, never
// the user's own pinned fills. The per-role maps are copied entry-by-entry;
// neither the original config nor its maps are mutated.
func fillsOnly(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}
	stripped := *cfg
	stripped.Agent.Profiles = providerOnlyRoles(cfg.Agent.Profiles)
	stripped.Agent.Tiers = providerOnlyRoles(cfg.Agent.Tiers)
	return &stripped
}

// providerOnlyRoles maps each role entry to {Provider: entry.Provider} — the
// reduction fillsOnly applies to one per-role override map.
func providerOnlyRoles(m map[string]config.RoleProfile) map[string]config.RoleProfile {
	if m == nil {
		return nil
	}
	out := make(map[string]config.RoleProfile, len(m))
	for role, p := range m {
		out[role] = config.RoleProfile{Provider: p.Provider}
	}
	return out
}

// defaultsMapFromFields nests each non-nil Default under its dotted key and
// merges the subtrees. Registry defaults are typed structs/maps; normalizing them
// through JSON gives the same map[string]any/[]any/scalar shape the decoded YAML
// layers use, so the merge and the provenance walk treat every tier uniformly.
func defaultsMapFromFields(fields []Field) map[string]any {
	out := make(map[string]any)
	for _, f := range fields {
		if f.Default == nil {
			continue
		}
		nested := normalizeToGeneric(f.Default)
		segs := strings.Split(f.Key, ".")
		for i := len(segs) - 1; i >= 1; i-- {
			nested = map[string]any{segs[i]: nested}
		}
		layer := map[string]any{segs[0]: nested}
		out = config.MergeLayers(out, layer)
	}
	return out
}

// normalizeToGeneric round-trips a typed registry default into the generic shape
// the layer maps use. It marshals via JSON, NOT YAML: the registry default structs
// (providerDefault, roleProfileDefault) carry `json:` tags whose names match the
// real config keys (interactive_command, provider, model, effort), whereas they carry
// no `yaml:` tags — a YAML marshal would emit lowercased Go field names
// (sessioncommand) that would not line up with the layer maps' snake_case keys,
// producing phantom leaves. Config values are strings/lists/maps, so JSON's
// numeric widening is not a concern; a marshal failure degrades to the typed value
// rather than dropping the row.
func normalizeToGeneric(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}
