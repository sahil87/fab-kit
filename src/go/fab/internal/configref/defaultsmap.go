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
// The composition reads the DEPTH KNOBS and the provider fills only: the user's
// own agent.profiles/agent.tiers entries are stripped first, because this map is
// the DEFAULTS tier and a tier must never echo back the value of a tier above it
// (an `agent.profiles.review.model` override would otherwise be reported as its
// own built-in, on the `default (shadowed)` line as well as the project one).
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

// liveRoleProfiles resolves every role through agent.ResolveRole against the live
// config MINUS its per-role overrides (see knobsOnly), so the row it produces is
// the built-in the user's own agent.profiles entry SHADOWS rather than a copy of
// that entry. Like roleRows it fails loud on role-map drift: a role
// agent.RoleNames reports must resolve (an unknown role name is ResolveRole's
// only error).
func liveRoleProfiles(cfg *config.Config) (map[string]roleProfileDefault, error) {
	names := agent.RoleNames()
	base := knobsOnly(cfg)
	out := make(map[string]roleProfileDefault, len(names))
	for _, name := range names {
		p, err := agent.ResolveRole(base, name)
		if err != nil {
			return nil, err
		}
		out[name] = roleProfileDefault{Provider: p.Provider, Model: p.Model, Effort: p.Effort}
	}
	return out, nil
}

// knobsOnly copies cfg with the per-role override maps (agent.profiles and its
// pre-2.17.0 spelling agent.tiers) CLEARED, leaving the depth knobs and the
// providers table — the two inputs the derived default legitimately depends on.
// The copy is shallow, and neither the original nor its remaining maps are
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
