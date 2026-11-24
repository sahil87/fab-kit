package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent presets (261124-e9nz). A PRESET is a NAME for a sparse bundle of fields fab
// already has — `session`, `workers`, `profiles.<role>.{cli,model,effort}` — chosen
// by the scalar `agent.preset` (FAB_AGENT_PRESET in the environment). It adds one
// rung to the fill precedence and nothing else: the resolver and every consumer keep
// their existing semantics.
//
// WHY THE SPLICE LIVES HERE. Layer identity (env / system / project) exists only in
// this package's loader; internal/agent receives an already-merged *Config. The
// rule is positional — the preset enters AT THE LAYER THAT CHOSE IT, directly below
// that layer's own explicit agent.* fields — so it cannot be expressed anywhere
// downstream. applyPreset is the one implementation, called from both fromProjectMap
// (typed Config) and layersFrom (Layers for `fab config show`), so what `show` prints
// is what consumers resolve.

// NoPreset is the reserved agent.preset value meaning "no preset here, even if a
// lower layer chose one". It wins the selection like any name but activates nothing,
// and a definition by this name is refused (warned and ignored).
const NoPreset = "none"

// AgentPreset is one definition in the `agent_presets:` table. Every field is
// optional (the definition is sparse): an unset field simply contributes nothing.
// Description is display-only (`fab agent presets`) and never reaches the agent.*
// tree. Profiles reuses RoleProfile with the same meaning as agent.profiles.
type AgentPreset struct {
	Description string                 `yaml:"description,omitempty"`
	Session     string                 `yaml:"session,omitempty"`
	Workers     string                 `yaml:"workers,omitempty"`
	Profiles    map[string]RoleProfile `yaml:"profiles,omitempty"`
}

// BuiltinAgentPresets is the built-in tier of the definitions merge — the
// `agent_presets:` block of the embedded defaults.yaml. Like the DefaultDispatch*
// vars it carries no literal: internal/agent's init() pushes the parsed table here
// (config cannot import agent — import cycle). A binary that never links
// internal/agent sees nil, i.e. no built-in presets.
var BuiltinAgentPresets map[string]AgentPreset

// AppliedPreset records the preset applyPreset spliced into the effective tree.
// Layer is the layer that CHOSE it ("env", "system" or "project"); Supplied is the
// set of agent.* leaves (dotted, relative to `agent`: "session", "workers",
// "profiles.review.cli", …) whose effective value came FROM the preset — i.e. no
// layer at or above the choosing layer set that leaf explicitly. internal/agent
// reads it to label `source.*`.
type AppliedPreset struct {
	Name     string
	Layer    string
	Supplied map[string]bool
	// Fields is the preset's full agent.* contribution (session, workers,
	// profiles.<role>.<leaf>), including leaves an explicit field at or above the
	// choosing layer overrides — the provenance walk needs them to show the preset
	// as a (possibly shadowed) tier.
	Fields map[string]any
}

// Supplies reports whether the preset supplied the agent.* leaf at path
// (nil-safe: no active preset supplies nothing).
func (p *AppliedPreset) Supplies(path string) bool {
	return p != nil && p.Supplied[path]
}

// Layer names of the three file/env layers a preset can be chosen at, and the label
// for the invocation flag.
const (
	presetLayerEnv     = "env"
	presetLayerSystem  = "system"
	presetLayerProject = "project"
	presetFlagLabel    = "--preset flag"
)

// presetFields are the fields a definition may set beyond description.
var presetRoleLeaves = []string{"cli", "model", "effort"}

// splicedLayers is applyPreset's result: the three layers (the choosing one
// replaced, the others returned as given) plus the applied-preset record.
type splicedLayers struct {
	project, system, env map[string]any
	applied              *AppliedPreset
}

// applyPreset selects the active preset and splices it into the layer that chose
// it. envPreset, when non-empty, is an invocation-time choice (`fab agent --preset`)
// applied at the ENV layer, replacing env's own agent.preset for this resolution.
//
// Selection: the highest layer (env > system > project) with a non-empty
// agent.preset. No such layer, or the winner is `none` → layers returned untouched
// (byte-identical to the pre-preset cascade). An undefined name warns and is treated
// as unset — never an error, so a stale name can never fail a spawn.
func applyPreset(project, system, env map[string]any, envPreset string) splicedLayers {
	out := splicedLayers{project: project, system: system, env: env}

	defs, defsWarn := presetDefinitions(project, system, env)
	if defsWarn {
		warnf("fab: warning: ignoring agent preset definition %q (reserved name)\n", NoPreset)
	}

	layer, name, fromFlag := choosePresetLayer(project, system, env, envPreset)
	if name == "" || name == NoPreset {
		return out
	}
	rawDef, defined := defs[name]
	if !defined {
		from := layer
		if fromFlag {
			from = presetFlagLabel
		}
		warnf("fab: warning: unknown agent preset %q (agent.preset from %s); ignoring it\n", name, from)
		return out
	}

	fields := presetAgentFields(rawDef)
	chosen := map[string]map[string]any{presetLayerEnv: env, presetLayerSystem: system, presetLayerProject: project}[layer]
	spliced := make(map[string]any, len(chosen)+1)
	for k, v := range chosen {
		spliced[k] = v
	}
	own, _ := asStringMap(chosen[agentKey])
	spliced[agentKey] = mergeOver(fields, own)

	switch layer {
	case presetLayerEnv:
		out.env = spliced
	case presetLayerSystem:
		out.system = spliced
	default:
		out.project = spliced
	}

	// The preset SUPPLIES a leaf only when nothing at or above the choosing layer
	// sets it explicitly (those win the merge and keep their own label).
	above := []map[string]any{env}
	switch layer {
	case presetLayerSystem:
		above = []map[string]any{env, system}
	case presetLayerProject:
		above = []map[string]any{env, system, project}
	}
	supplied := make(map[string]bool)
	for _, path := range leafPaths(fields) {
		explicit := false
		for _, l := range above {
			if v, ok := valueAt(l, append([]string{agentKey}, strings.Split(path, ".")...)); ok && !IsEmptyValue(v) {
				explicit = true
				break
			}
		}
		if !explicit {
			supplied[path] = true
		}
	}
	out.applied = &AppliedPreset{Name: name, Layer: layer, Supplied: supplied, Fields: fields}
	return out
}

// choosePresetLayer returns the layer holding the winning agent.preset value and
// that value, or ("", "") when no layer sets one. A non-empty envPreset overrides
// the env layer's own value (fromFlag reports it).
func choosePresetLayer(project, system, env map[string]any, envPreset string) (layer, name string, fromFlag bool) {
	if envPreset != "" {
		return presetLayerEnv, envPreset, true
	}
	for _, c := range []struct {
		layer string
		m     map[string]any
	}{{presetLayerEnv, env}, {presetLayerSystem, system}, {presetLayerProject, project}} {
		if v := presetScalar(c.m); v != "" {
			return c.layer, v, false
		}
	}
	return "", "", false
}

// presetScalar reads agent.preset from one raw layer as a string ("" when unset or
// not a scalar). A scalar that YAML decoded as a non-string (a bare number) is
// stringified, matching the typed Config's string decode.
func presetScalar(layer map[string]any) string {
	v, ok := valueAt(layer, []string{agentKey, "preset"})
	if !ok || IsEmptyValue(v) {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case map[string]any, map[any]any, []any:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// presetDefinitions composes the definitions table across the tiers — built-in <
// project < system < env, merged per leaf through MergeLayers — as raw maps keyed by
// name. A definition named `none` is dropped; the bool reports that it was present
// so the caller can warn once.
func presetDefinitions(project, system, env map[string]any) (map[string]any, bool) {
	builtin := map[string]any{}
	if len(BuiltinAgentPresets) > 0 {
		builtin[agentPresetsKey] = presetsToMap(BuiltinAgentPresets)
	}
	merged := MergeLayers(builtin, subtree(project, agentPresetsKey), subtree(system, agentPresetsKey), subtree(env, agentPresetsKey))
	defs, _ := asStringMap(merged[agentPresetsKey])
	_, hadNone := defs[NoPreset]
	delete(defs, NoPreset)
	return defs, hadNone
}

const agentPresetsKey = "agent_presets"

func subtree(layer map[string]any, key string) map[string]any {
	if v, ok := layer[key]; ok {
		return map[string]any{key: v}
	}
	return nil
}

// presetsToMap converts a typed definitions table to its raw-map form (the shape
// MergeLayers composes). Empty fields vanish via the emptiness rule at merge time.
func presetsToMap(p map[string]AgentPreset) map[string]any {
	data, err := yaml.Marshal(p)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// presetAgentFields projects one raw definition onto the agent.* fields it may
// supply: session, workers and profiles.<role>.{cli,model,effort}, non-empty strings
// only. description (and any unknown key) never reaches the agent tree.
func presetAgentFields(def any) map[string]any {
	fields := make(map[string]any)
	d, ok := asStringMap(def)
	if !ok {
		return fields
	}
	for _, k := range []string{"session", "workers"} {
		if s, ok := d[k].(string); ok && s != "" {
			fields[k] = s
		}
	}
	profiles, _ := asStringMap(d["profiles"])
	outProfiles := make(map[string]any)
	for role, rv := range profiles {
		rm, ok := asStringMap(rv)
		if !ok {
			continue
		}
		leaves := make(map[string]any)
		for _, leaf := range presetRoleLeaves {
			if s, ok := rm[leaf].(string); ok && s != "" {
				leaves[leaf] = s
			}
		}
		if len(leaves) > 0 {
			outProfiles[role] = leaves
		}
	}
	if len(outProfiles) > 0 {
		fields["profiles"] = outProfiles
	}
	return fields
}

// leafPaths lists the dotted leaf paths of an agent-field map, sorted.
func leafPaths(fields map[string]any) []string {
	var out []string
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			if sub, ok := asStringMap(v); ok {
				walk(path, sub)
				continue
			}
			out = append(out, path)
		}
	}
	walk("", fields)
	sort.Strings(out)
	return out
}

// valueAt walks a raw layer map down parts, reporting the value and whether the
// whole path exists.
func valueAt(layer map[string]any, parts []string) (any, bool) {
	var cur any = layer
	for _, p := range parts {
		m, ok := asStringMap(cur)
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// DefinedAgentPresets returns the composed definitions (built-in tier under the
// config's own file+env definitions, merged per field), keyed by name. `none` is
// never present. Nil-safe.
func DefinedAgentPresets(c *Config) map[string]AgentPreset {
	var user map[string]AgentPreset
	if c != nil {
		user = c.AgentPresets
	}
	merged := MergeLayers(
		map[string]any{agentPresetsKey: presetsToMap(BuiltinAgentPresets)},
		map[string]any{agentPresetsKey: presetsToMap(user)},
	)
	return presetsFromMap(merged[agentPresetsKey])
}

// presetsFromMap decodes a raw definitions map into the typed table, dropping the
// reserved name.
func presetsFromMap(raw any) map[string]AgentPreset {
	out := map[string]AgentPreset{}
	m, ok := asStringMap(raw)
	if !ok {
		return out
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return out
	}
	_ = yaml.Unmarshal(data, &out)
	delete(out, NoPreset)
	return out
}

// PresetSource names the highest tier that contributed to one preset definition:
// "built-in", "project", "system" or "env" — the `source` of `fab agent presets
// --json`. A system tweak of a built-in reports "system"; an env-supplied or
// env-tweaked definition reports "env".
type PresetSource string

const (
	PresetSourceBuiltIn PresetSource = "built-in"
	PresetSourceProject PresetSource = PresetSource(presetLayerProject)
	PresetSourceSystem  PresetSource = PresetSource(presetLayerSystem)
	PresetSourceEnv     PresetSource = PresetSource(presetLayerEnv)
)

// AgentPresetDefinitions returns every defined preset with its highest contributing
// tier, computed over the RAW layers (so a tier that merely tweaks a built-in is
// still credited). It is the read model behind `fab agent presets`.
func AgentPresetDefinitions(l *Layers) (map[string]AgentPreset, map[string]PresetSource) {
	defs, _ := presetDefinitions(l.Project, l.System, l.Env)
	typed := presetsFromMap(defs)
	sources := make(map[string]PresetSource, len(typed))
	for name := range typed {
		sources[name] = PresetSourceBuiltIn
		for _, t := range []struct {
			src   PresetSource
			layer map[string]any
		}{{PresetSourceProject, l.Project}, {PresetSourceSystem, l.System}, {PresetSourceEnv, l.Env}} {
			if v, ok := valueAt(t.layer, []string{agentPresetsKey, name}); ok && !IsEmptyValue(pruneEmptyValue(v)) {
				sources[name] = t.src
			}
		}
	}
	return typed, sources
}

// pruneEmptyValue drops a value's empty leaves recursively (the merge rule), so a
// layer that only carries `{profiles: {}}` is not credited as a contributor.
func pruneEmptyValue(v any) any {
	if IsEmptyValue(v) {
		return nil
	}
	if _, isMap := asStringMap(v); !isMap {
		return v
	}
	return MergeLayers(map[string]any{"v": v})["v"]
}

// DefinedAgentPresetNames returns the sorted names of the defined presets visible
// from layers (built-in ∪ project ∪ system ∪ env), without the reserved `none`.
func DefinedAgentPresetNames(l *Layers) []string {
	defs, _ := AgentPresetDefinitions(l)
	names := make([]string, 0, len(defs))
	for n := range defs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
