// Package configscope is the leaf source of truth for a config field's override
// visibility (its "scope") across the cascade layers. It is deliberately
// dependency-free (imports nothing internal) so that BOTH the config loader
// (internal/config) and the reference generator (internal/configref) can consume
// it without an import cycle.
//
// WHY A LEAF PACKAGE. The cascade resolver in internal/config must prune
// project-scoped fields out of the system layer (~/.fab-kit/config.yaml), so it
// needs each top-level field's scope. But internal/config CANNOT import
// internal/configref — configref imports internal/agent, which imports
// internal/config, so a configref import would close a cycle. Extracting the
// scope enum and the key→scope table into this leaf package breaks the cycle
// while keeping the scope values SINGLE-SOURCED: configref derives its per-row
// Field.Scope from ScopeFor here (no second copy of the taxonomy), and the loader
// reads ScopeFor directly. See docs/specs/config.md § Scope taxonomy and
// § Override cascade [Change 2].
package configscope

// Scope is a config field's override visibility across the cascade layers
// (project file / system file / built-in defaults). It is the single scope enum
// for the whole binary — internal/configref aliases this type so there is one
// definition, not two.
type Scope string

const (
	// ScopeProject: overridable only in the project file (fab/project/config.yaml).
	// Semantics-class fields stay repo-reproducible for teammates/CI. The cascade
	// PRUNES a project-scoped field found in the system file (with a warning).
	ScopeProject Scope = "project"
	// ScopeSystem: overridable only in the system file (~/.fab-kit/config.yaml).
	// (No field is system-only today; the value exists for completeness.)
	ScopeSystem Scope = "system"
	// ScopeBoth: overridable in either layer (preference-class fields). Honored in
	// the system file by the cascade.
	ScopeBoth Scope = "both"
)

// Valid reports whether s is one of the three known scope values. Used by the
// configref registry lint (a scope outside this set is a construction error).
func Valid(s Scope) bool {
	switch s {
	case ScopeProject, ScopeSystem, ScopeBoth:
		return true
	default:
		return false
	}
}

// keyScopes maps a TOP-LEVEL config key (the YAML key at the root of
// config.yaml) to its scope. The cascade resolver keys on top-level YAML keys —
// the granularity a system-file override unit lands at — so the table is keyed by
// top-level key, not by the finer dotted registry key (e.g. `project.name`
// collapses to the top-level `project`). This is the decision-6 taxonomy
// (config-upgrade.md): agent/providers are preference-class (`both`); everything
// else is semantics-class (`project`). System-only (`system`) has no member today.
//
// It is the SINGLE source for the taxonomy: configref derives each Field.Scope
// from ScopeFor(topLevel(field.Key)), and the loader prunes on it directly.
//
// fab_version is intentionally ABSENT (260708-j0qm): it moved out of config.yaml
// to the plain-text sibling fab/.fab-version and is no longer a scoped/registry
// key, so it carries no scope. It is NOT, however, ignored: a stale `fab_version:`
// in a not-yet-migrated PROJECT file is still parsed by internal/config (the
// Config.FabVersion field survives one compat window) and read as a legacy
// fallback when fab/.fab-version is absent, until the 2.14.0-to-2.15.0 migration
// removes it. In the SYSTEM file it is handled differently: fab_version is repo-scoped state, so the
// loader (internal/config.pruneProjectScoped) STRIPS a system-file fab_version as a
// named compat-window exception — it must never bleed a machine-global value into a
// repo's resolved version. That strip is not driven by this table (fab_version has
// no scope row); it is a one-off in the loader, removed with the compat window.
var keyScopes = map[string]Scope{
	"project":             ScopeProject,
	"source_paths":        ScopeProject,
	"test_paths":          ScopeProject,
	"true_impact_exclude": ScopeProject,
	"checklist":           ScopeProject,
	"providers":           ScopeBoth,
	"agent":               ScopeBoth,
	"stage_hooks":         ScopeProject,
	"branch_prefix":       ScopeProject,
}

// ScopeFor returns the scope of a top-level config key and whether the key is
// known. An unknown top-level key reports (ScopeProject, false); the loader
// treats unknown keys as ignored-silently (matching project-file behavior), so
// the bool — not a default scope — is what a caller keys on.
func ScopeFor(topLevelKey string) (Scope, bool) {
	s, ok := keyScopes[topLevelKey]
	return s, ok
}
