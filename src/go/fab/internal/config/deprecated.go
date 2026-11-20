package config

import (
	"sort"
	"strings"
)

// Deprecated spellings of the CLI-tool vocabulary (261120-f2ly, kit 2.31.0).
//
// The agent CLIs fab can launch (claude, codex, agy, kimi, user-defined) were
// called "providers" until 2.31.0, which collided with the tools' own use of the
// word for the MODEL VENDOR. The canonical spellings are now the top-level
// `clis:` table and `agent.profiles.<role>.cli`; the old `providers:` table and
// `.provider` leaf stay readable, and settable through `fab config set`/`unset`,
// for ONE minor release after the one that ships the rename, then are removed in a
// separate change. Reads are silent; only set/unset, `fab config validate`, the
// hidden `fab agent --provider` flag and the migration warn.
//
// The compatibility lives at two seams so no other package carries dual-spelling
// logic: NormalizeDeprecated folds the old spellings into the new ones on every raw
// layer map before the cascade merge (the resolver, `show`, `explain`, setup checks
// and `fab agent models` therefore only ever see the new shape), and CanonicalKey
// maps a dotted key for the write-side surfaces.

const (
	deprecatedCLIsKey       = "providers" // top-level table, now `clis`
	deprecatedRoleCLILeaf   = "provider"  // agent.profiles.<role>.provider, now `.cli`
	canonicalCLIsKey        = "clis"
	canonicalRoleCLILeaf    = "cli"
	agentProfilesKey        = "profiles"
	deprecatedAgentTiersKey = "tiers"
	agentKey                = "agent"
)

// DeprecatedUse is one live deprecated spelling found in a raw config layer:
// Old is the dotted key as written, New the canonical key that replaces it.
type DeprecatedUse struct {
	Old string
	New string
}

// NormalizeDeprecated folds the deprecated CLI spellings of one raw config layer
// into the canonical ones, per field, the new spelling winning:
//
//   - a top-level `providers:` map merges into `clis:` per entry and per leaf
//     (recursing into `profiles.<role>`); a non-empty `clis` leaf beats the
//     `providers` leaf, an empty one falls through (the cascade's empty-skip rule),
//     and an explicit `native: false` survives either way;
//   - `agent.profiles.<role>.provider` (and the legacy `agent.tiers.<role>.provider`)
//     folds into `.cli` the same way.
//
// After the call no `providers` / `.provider` key remains. The input is never
// mutated — a layer with no deprecated key is returned as-is, otherwise the touched
// path is copied — so callers can reuse the raw map for provenance display.
func NormalizeDeprecated(layer map[string]any) map[string]any {
	if layer == nil {
		return nil
	}
	_, hasProviders := layer[deprecatedCLIsKey]
	agent, agentIsMap := asStringMap(layer[agentKey])
	foldAgent := agentIsMap && (hasRoleProvider(agent[agentProfilesKey]) || hasRoleProvider(agent[deprecatedAgentTiersKey]))
	if !hasProviders && !foldAgent {
		return layer
	}

	out := make(map[string]any, len(layer))
	for k, v := range layer {
		out[k] = v
	}
	if hasProviders {
		foldProvidersTable(out)
	}
	if foldAgent {
		agentCopy := make(map[string]any, len(agent))
		for k, v := range agent {
			agentCopy[k] = v
		}
		for _, key := range []string{agentProfilesKey, deprecatedAgentTiersKey} {
			if roles, ok := asStringMap(agentCopy[key]); ok && hasRoleProvider(roles) {
				agentCopy[key] = foldRoleProviders(roles)
			}
		}
		out[agentKey] = agentCopy
	}
	return out
}

// foldProvidersTable moves out[providers] into out[clis] in place (out is already
// a private copy). A non-map `providers` value (a malformed file) is carried to
// `clis` verbatim when `clis` is absent so the type error still surfaces at decode
// time exactly as it did under the old key; when `clis` exists it is dropped.
func foldProvidersTable(out map[string]any) {
	old := out[deprecatedCLIsKey]
	delete(out, deprecatedCLIsKey)
	oldMap, oldIsMap := asStringMap(old)
	if !oldIsMap {
		if _, has := out[canonicalCLIsKey]; !has && !IsEmptyValue(old) {
			out[canonicalCLIsKey] = old
		}
		return
	}
	newMap, newIsMap := asStringMap(out[canonicalCLIsKey])
	if !newIsMap && out[canonicalCLIsKey] != nil {
		return // a malformed `clis` value keeps surfacing as the type error
	}
	merged := mergeOver(mergeOver(nil, oldMap), newMap)
	if len(merged) > 0 {
		out[canonicalCLIsKey] = merged
	}
}

// hasRoleProvider reports whether a `profiles`/`tiers` value is a map with at
// least one role entry carrying the deprecated `provider` leaf.
func hasRoleProvider(v any) bool {
	roles, ok := asStringMap(v)
	if !ok {
		return false
	}
	for _, entry := range roles {
		if m, ok := asStringMap(entry); ok {
			if _, has := m[deprecatedRoleCLILeaf]; has {
				return true
			}
		}
	}
	return false
}

// foldRoleProviders returns a copy of a role map with every entry's `provider`
// leaf folded into `cli` (a non-empty `cli` wins).
func foldRoleProviders(roles map[string]any) map[string]any {
	out := make(map[string]any, len(roles))
	for role, entry := range roles {
		m, ok := asStringMap(entry)
		if !ok {
			out[role] = entry
			continue
		}
		old, has := m[deprecatedRoleCLILeaf]
		if !has {
			out[role] = entry
			continue
		}
		folded := make(map[string]any, len(m))
		for k, v := range m {
			if k != deprecatedRoleCLILeaf {
				folded[k] = v
			}
		}
		if IsEmptyValue(folded[canonicalRoleCLILeaf]) && !IsEmptyValue(old) {
			folded[canonicalRoleCLILeaf] = old
		}
		out[role] = folded
	}
	return out
}

// CanonicalKey maps a dotted config key written with a deprecated spelling to its
// canonical form: `providers…` → `clis…` and `agent.profiles.<role>.provider` (also
// `agent.tiers.<role>.provider`) → `agent.profiles.<role>.cli`. It reports whether
// the key was deprecated; a canonical or unrelated key comes back unchanged with
// false. It is the ONE key-mapping helper shared by `fab config set`/`unset`,
// `fab config validate` and `fab config explain`.
func CanonicalKey(key string) (string, bool) {
	parts := strings.Split(key, ".")
	switch {
	case parts[0] == deprecatedCLIsKey:
		parts[0] = canonicalCLIsKey
		return strings.Join(parts, "."), true
	case len(parts) == 4 && parts[0] == agentKey &&
		(parts[1] == agentProfilesKey || parts[1] == deprecatedAgentTiersKey) &&
		parts[3] == deprecatedRoleCLILeaf:
		parts[1], parts[3] = agentProfilesKey, canonicalRoleCLILeaf
		return strings.Join(parts, "."), true
	}
	return key, false
}

// TwinKey returns the other spelling of a canonical or deprecated key — the old
// spelling for a canonical `clis…`/`agent.profiles.<role>.cli` key, and the new one
// for a deprecated key — or "" when the key has no twin. `set` and `unset` use it
// to keep one spelling per file.
func TwinKey(key string) string {
	if canonical, deprecated := CanonicalKey(key); deprecated {
		return canonical
	}
	parts := strings.Split(key, ".")
	switch {
	case parts[0] == canonicalCLIsKey:
		parts[0] = deprecatedCLIsKey
		return strings.Join(parts, ".")
	case len(parts) == 4 && parts[0] == agentKey && parts[1] == agentProfilesKey && parts[3] == canonicalRoleCLILeaf:
		parts[3] = deprecatedRoleCLILeaf
		return strings.Join(parts, ".")
	}
	return ""
}

// DeprecatedUses lists the live deprecated spellings in one RAW (un-normalized)
// config layer, sorted by old key. `fab config validate` warns on each. The legacy
// `agent.tiers` spelling is a separate, older deprecation and is not reported here.
func DeprecatedUses(layer map[string]any) []DeprecatedUse {
	var uses []DeprecatedUse
	if v, has := layer[deprecatedCLIsKey]; has && !IsEmptyValue(v) {
		uses = append(uses, DeprecatedUse{Old: deprecatedCLIsKey, New: canonicalCLIsKey})
	}
	if agent, ok := asStringMap(layer[agentKey]); ok {
		if roles, ok := asStringMap(agent[agentProfilesKey]); ok {
			for role, entry := range roles {
				m, ok := asStringMap(entry)
				if !ok {
					continue
				}
				if v, has := m[deprecatedRoleCLILeaf]; has && !IsEmptyValue(v) {
					old := strings.Join([]string{agentKey, agentProfilesKey, role, deprecatedRoleCLILeaf}, ".")
					uses = append(uses, DeprecatedUse{Old: old, New: strings.Join([]string{agentKey, agentProfilesKey, role, canonicalRoleCLILeaf}, ".")})
				}
			}
		}
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Old < uses[j].Old })
	return uses
}

// RawDeprecatedUses reports the live deprecated spellings in the project file at
// projectPath ("" skips it) and in the system file (whose path it also returns), read RAW — the cascade loaders
// normalize them away, so a diagnostic that wants to name them (`fab setup`'s
// check) needs its own un-normalized read. Fail-open and silent: an unreadable or
// absent file simply reports nothing, since the loaders already warned about it.
func RawDeprecatedUses(projectPath string) (project, system []DeprecatedUse, systemPath string) {
	if projectPath != "" {
		if m, _, err := readYAMLMap(projectPath); err == nil {
			project = DeprecatedUses(m)
		}
	}
	if sysPath, err := systemConfigPath(); err == nil {
		systemPath = sysPath
		if m, _, err := readYAMLMap(sysPath); err == nil {
			system = DeprecatedUses(m)
		}
	}
	return project, system, systemPath
}
