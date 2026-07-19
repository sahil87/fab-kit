package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// homeDir resolves the current user's home directory. It is a package var (not a
// direct os.UserHomeDir call) so tests can pin the system-config path with
// t.Setenv("HOME", …) — os.UserHomeDir honors $HOME on unix, so the seam is the
// env var, and this indirection also lets a test stub it if needed.
var homeDir = os.UserHomeDir

// warnw is where the loader writes fail-open scope/parse warnings. os.Stderr in
// production; tests redirect it to capture the `fab: warning:` lines. Warnings
// never affect the return value or exit code (fail-open — a broken personal
// system file must not brick every repo on the machine).
var warnw io.Writer = os.Stderr

// systemConfigPath returns ~/.fab-kit/config.yaml, the system (user-global) config
// layer. Co-located with the fab-kit version cache (decision 5; XDG rejected).
// An error resolving the home dir yields ("", err) — the caller treats that as
// "no system layer" (fail-open).
func systemConfigPath() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fab-kit", "config.yaml"), nil
}

// StageHook holds pre/post shell commands for a pipeline stage.
type StageHook struct {
	Pre  string `yaml:"pre"`
	Post string `yaml:"post"`
}

// ProviderConfig models one entry of the top-level `providers:` table: a named
// invocation grammar for an agent harness. Provider names are opaque, user-chosen
// strings (fab never infers a provider from a model string).
//
//   - SessionCommand opens an interactive agent SESSION (the relocated
//     agent.spawn_command semantics — consumed by fab operator / fab batch /
//     fab agent).
//   - DispatchCommand runs ONE headless stage task via fab dispatch (the relocated
//     per-tier spawn_command semantics). ABSENT DispatchCommand = native
//     Agent-tool dispatch (the default). There is NO fallback between the two
//     fields: absence of DispatchCommand signals native dispatch, never "use
//     SessionCommand".
//
// The two fields are deliberately NOT merged into one command: session and
// dispatch are different invocations of the same binary (claude interactive `-n`
// vs headless `-p`; codex TUI vs `codex exec`), and no single template expresses
// both. Both strings pass through verbatim — fab applies NO validation against any
// provider's accepted set (provider neutrality, Constitution Principle I). The
// {model}/{effort} placeholders are substituted at resolve time via internal/spawn.
type ProviderConfig struct {
	SessionCommand  string `yaml:"session_command"`
	DispatchCommand string `yaml:"dispatch_command"`
}

// TierProfile is a named `{provider, model, effort}` agent profile. Every field
// MAY be empty: an empty Provider/Model/Effort inherits from the project's
// `default` tier, then from fab-kit's built-in (per-field merge, performed by
// internal/agent). An empty Model additionally signals "inherit the
// session/orchestrator model" once resolution bottoms out.
//
// Provider names the entry in the top-level `providers:` table whose command
// grammar this tier's stages use. The command itself lives on the provider, NOT
// the tier — inheriting {provider, model, effort} is safe precisely because the
// dangerous cross-semantics command inheritance can no longer happen.
//
// All strings are pass-through — fab applies NO validation (provider neutrality,
// Constitution Principle I). See internal/agent for resolution.
type TierProfile struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
}

// AgentConfig models the `agent:` section of config.yaml.
//
// Tiers is the sole per-stage-model override surface: a map of role-tier name
// (default/operator/doing/review/fast) → profile. The stage→tier mapping itself
// is fab-owned and NOT user-overridable (no stage_tiers, no per-stage escape
// hatch); users override only what each tier *means*. An omitted tier — or an
// omitted field within a tier — falls back to the project's `default` tier and
// then fab-kit's built-in default (per-field merge, performed by internal/agent).
// yaml.v3 ignores unknown keys, so adding Tiers is free for existing configs (the
// same property that made stage_hooks free).
type AgentConfig struct {
	Tiers map[string]TierProfile `yaml:"tiers"`
}

// ProjectConfig models the `project:` section of config.yaml.
type ProjectConfig struct {
	LinearWorkspace string `yaml:"linear_workspace"`
}

// Config holds the parsed project config relevant to the fab binary. It is
// the single owner of fab/project/config.yaml parsing — every key the fab
// module consumes is modeled here and read through a nil-safe accessor, so no
// satellite one-off parser re-reads the file (260612-ye8r). yaml.v3 ignores
// unknown keys, so widening this struct is free for existing configs.
//
// Known coupled-failure caveat: a yaml type error on ANY modeled key fails
// the single Unmarshal, sending every accessor to its documented fallback
// (default spawn command, empty branch prefix, empty workspace, silent
// staleness skip). The documented per-caller fallbacks make this safe for
// malformed configs — a deliberate, recorded semantic for the consolidation.
type Config struct {
	StageHooks        map[string]StageHook `yaml:"stage_hooks"`
	TrueImpactExclude []string             `yaml:"true_impact_exclude"`
	TestPaths         []string             `yaml:"test_paths"`
	BranchPrefix      string               `yaml:"branch_prefix"`
	// FabVersion is NOT parsed from config.yaml — the version pin lives in the
	// plain-text sibling fab/.fab-version (260708-j0qm). The explicit `yaml:"-"`
	// (not a bare untagged field) stops yaml.v3 from matching the lowercased field
	// name, so a stale `fab_version:` key in config.yaml is an inert unknown key.
	// The field is populated only by Load's readDotFabVersion overlay and consumed
	// by GetFabVersion → preflight's staleness check.
	FabVersion string                    `yaml:"-"`
	Providers  map[string]ProviderConfig `yaml:"providers"`
	Agent      AgentConfig               `yaml:"agent"`
	Project    ProjectConfig             `yaml:"project"`
}

// Load reads fab/project/config.yaml from fabRoot and returns the parsed config.
// Returns an empty config if the file doesn't exist.
//
// fab_version resolution (260708-j0qm): the version lives in the plain-text
// sibling file fab/.fab-version, written by `fab init`/`fab upgrade-repo` and
// stamped there instead of into config.yaml. It is the SOLE source — Load reads
// it via the readDotFabVersion overlay and overwrites Config.FabVersion when
// present. config.yaml is never consulted for the version (Config.FabVersion is
// tagged `yaml:"-"`). LoadPath itself is version-agnostic — it takes a bare path
// with no repo-root context — so the .fab-version overlay lives here in Load, the
// only seam that knows fabRoot.
func Load(fabRoot string) (*Config, error) {
	cfg, err := LoadPath(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		return nil, err
	}
	if v := readDotFabVersion(fabRoot); v != "" {
		cfg.FabVersion = v
	}
	return cfg, nil
}

// readDotFabVersion reads the bare-semver value from fab/.fab-version, or "" when
// the file is absent/empty/unreadable (fail-open — a missing .fab-version simply
// leaves Config.FabVersion empty, which preflight's staleness check silently
// skips). The file is a one-line plain-text sibling to fab/.kit-migration-version.
func readDotFabVersion(fabRoot string) string {
	data, err := os.ReadFile(filepath.Join(fabRoot, ".fab-version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// LoadPath reads a config.yaml at an explicit path and returns the EFFECTIVE
// config after resolving the three-layer cascade at this single seam:
//
//	project (the path given)  >  system (~/.fab-kit/config.yaml)  >  built-in defaults
//
// The two FILES merge here at the YAML map level (per-field deep merge: maps
// merge per-key recursively, lists replace, scalars replace, project wins); the
// built-in-defaults layer stays where it lives today — the point-of-use fallbacks
// (internal/agent's tier/provider merge, the nil-safe accessors) — which composes
// to identical three-layer semantics with zero per-caller change.
//
// Fail-open contract (config must never brick):
//   - Absent system file ⇒ byte-identical to the pre-cascade single-file behavior
//     (empty overlay, no error, no warning).
//   - Malformed/unreadable system file ⇒ a `fab: warning:` on stderr and the
//     system layer is SKIPPED (a broken personal file must not break every repo).
//   - A project-scoped field appearing in the system file is PRUNED from the
//     system layer with a `fab: warning:` (scope enforcement — decision 6).
//   - A malformed PROJECT file keeps today's behavior: the parse error is returned.
//
// Callers that build the path themselves (e.g. `fab agent --repo <path>`) use
// this directly; everyone else goes through Load. Returns an empty config (no
// error) when neither file exists.
func LoadPath(path string) (*Config, error) {
	projectMap, _, err := readYAMLMap(path)
	if err != nil {
		// A malformed PROJECT file keeps today's error behavior.
		return nil, err
	}

	systemMap := loadSystemLayer()

	// Merge project OVER system (project wins per field). A nil project map (file
	// absent) still lets the system layer through — the system layer is
	// user-global and must apply even in a repo with no project config.
	merged := deepMerge(systemMap, projectMap)

	var cfg Config
	if len(merged) > 0 {
		data, err := yaml.Marshal(merged)
		if err != nil {
			// Re-marshalling a map we just decoded should never fail; treat a
			// failure as a project-side error (the merged tree is dominated by
			// project content).
			return nil, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}

	if cfg.StageHooks == nil {
		cfg.StageHooks = make(map[string]StageHook)
	}

	// Absent system layer + absent project file, or a project file that decoded
	// to nothing, both leave `merged` empty and yield the zero Config — the
	// byte-identical empty-config result the old missing-file path returned.
	return &cfg, nil
}

// Layers holds the raw decoded config layers behind the effective config,
// exposed for provenance queries (fab config show --origin). It is produced by
// LoadLayers, which runs the SAME cascade LoadPath runs — same system-file
// resolution, same scope pruning, same deep merge — so `show` cannot drift from
// what consumers actually see. The maps are the decoded YAML trees (map-valued
// fields are nested maps), enabling per-key provenance drill-down.
type Layers struct {
	// ProjectPath / SystemPath are the resolved file paths (SystemPath is "" only
	// when the home dir could not be resolved). They are the origin labels
	// `show --origin` prints.
	ProjectPath string
	SystemPath  string
	// Project is the raw project-file map (nil when the file is absent/empty).
	Project map[string]any
	// System is the system-file map AFTER scope pruning (nil when absent/empty or
	// skipped fail-open). Project-scoped keys are already removed, so a key present
	// here is genuinely a system-layer contributor.
	System map[string]any
	// Effective is deepMerge(System, Project) — the merged tree LoadPath unmarshals.
	Effective map[string]any
}

// LoadLayers resolves the cascade and returns the raw layers for provenance
// display, without unmarshalling into Config. It shares the loader's fail-open
// contract: a malformed system file is warned + skipped, project-scoped system
// fields are pruned with a warning, and a malformed PROJECT file returns the
// parse error (mirroring LoadPath). Used by `fab config show [--origin]`.
func LoadLayers(projectPath string) (*Layers, error) {
	projectMap, _, err := readYAMLMap(projectPath)
	if err != nil {
		return nil, err
	}
	systemMap := loadSystemLayer()
	sysPath, _ := systemConfigPath() // "" only if HOME is unresolvable (fail-open)
	return &Layers{
		ProjectPath: projectPath,
		SystemPath:  sysPath,
		Project:     projectMap,
		System:      systemMap,
		Effective:   deepMerge(systemMap, projectMap),
	}, nil
}

// readYAMLMap reads a config.yaml at path into a generic map for merging. Returns
// (nil, false, nil) when the file does not exist (an absent layer, not an error),
// (map, true, nil) on success, and (nil, false, err) on a read error other than
// not-exist or a YAML decode error. An empty file decodes to a nil map with
// exists=true (a present-but-empty layer).
func readYAMLMap(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, false, err
	}
	return m, true, nil
}

// loadSystemLayer reads ~/.fab-kit/config.yaml, prunes project-scoped fields (with
// a warning), and returns the resulting overlay map. It NEVER returns an error —
// every failure mode is fail-open (config must never brick):
//   - home-dir unresolvable, or file absent ⇒ nil (no system layer, silent).
//   - unreadable or malformed ⇒ a `fab: warning:` on stderr, then nil (skip layer).
func loadSystemLayer() map[string]any {
	path, err := systemConfigPath()
	if err != nil {
		return nil // cannot resolve HOME — no system layer, silently (not the user's fault)
	}
	m, exists, err := readYAMLMap(path)
	if err != nil {
		// Unreadable or malformed system file — fail-open: warn and skip.
		fmt.Fprintf(warnw, "fab: warning: ignoring malformed system config %s (%v)\n", path, err)
		return nil
	}
	if !exists || m == nil {
		return nil // absent or empty ⇒ byte-identical current behavior
	}
	pruneProjectScoped(m, path)
	return m
}

// pruneProjectScoped removes project-scoped top-level keys from a system-layer map
// in place, emitting a `fab: warning:` for each pruned key. A key whose scope is
// `both` or `system` is honored (kept); an UNKNOWN top-level key is left in place
// silently (matching project-file behavior — typo surfacing is `show --origin`'s
// job, and yaml.v3 ignores unknown keys at unmarshal anyway). path names the
// system file in the warning.
//
// fab_version is not a config key — it lives in the plain-text sibling
// fab/.fab-version (260708-j0qm) and Config.FabVersion is tagged `yaml:"-"`, so a
// stale `fab_version:` here is an inert unknown key (nothing unmarshals it) and is
// left in place silently like any other unknown key. It can never reach a repo's
// resolved version.
func pruneProjectScoped(m map[string]any, path string) {
	for key := range m {
		scope, known := configscope.ScopeFor(key)
		if !known {
			continue // unknown key — ignored silently, like the project file
		}
		if scope == configscope.ScopeProject {
			delete(m, key)
			fmt.Fprintf(warnw, "fab: warning: ignoring project-scoped field %q in %s (project-scoped fields belong in fab/project/config.yaml)\n", key, path)
		}
	}
}

// deepMerge returns the per-field deep merge of two decoded YAML maps with
// `over` winning: MAPS merge per-key recursively, LISTS replace (never
// concatenate), SCALARS replace. It does not mutate `base` or `over` at the top
// level (it builds a fresh result), so callers may reuse the inputs. A nil `over`
// yields a shallow copy of `base`; a nil `base` yields a shallow copy of `over`.
func deepMerge(base, over map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, ov := range over {
		if bv, ok := out[k]; ok {
			if bm, bok := asStringMap(bv); bok {
				if om, ook := asStringMap(ov); ook {
					// Both sides are maps — merge per-key recursively.
					out[k] = deepMerge(bm, om)
					continue
				}
			}
		}
		// Lists replace, scalars replace, and a map-vs-non-map mismatch replaces:
		// the `over` value wins wholesale.
		out[k] = ov
	}
	return out
}

// asStringMap coerces a decoded YAML value to a map[string]any when it is one.
// yaml.v3 decodes mappings into map[string]interface{} when the target is `any`,
// so this is the only map shape encountered; it also tolerates map[any]any for
// robustness against alternate decoders.
func asStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	default:
		return nil, false
	}
}

// GetStageHook returns the hook config for a stage, or an empty hook if none configured.
func (c *Config) GetStageHook(stage string) StageHook {
	if c == nil || c.StageHooks == nil {
		return StageHook{}
	}
	return c.StageHooks[stage]
}

// GetBranchPrefix returns branch_prefix, or "" when unset (nil-safe).
func (c *Config) GetBranchPrefix() string {
	if c == nil {
		return ""
	}
	return c.BranchPrefix
}

// GetFabVersion returns fab_version, or "" when unset (nil-safe).
func (c *Config) GetFabVersion() string {
	if c == nil {
		return ""
	}
	return c.FabVersion
}

// GetProvider returns the configured ProviderConfig for a provider name and
// whether one was set. Nil-safe: a nil *Config, an absent providers block, or an
// unconfigured name all report (zero, false). The bool lets a caller distinguish
// "no provider entry" from "entry present but with empty fields" — the distinction
// internal/agent relies on for per-field merge over fab-kit's built-in provider
// table.
func (c *Config) GetProvider(name string) (ProviderConfig, bool) {
	if c == nil || c.Providers == nil {
		return ProviderConfig{}, false
	}
	p, ok := c.Providers[name]
	return p, ok
}

// GetAgentTier returns the configured override profile for a tier name and
// whether one was set. Nil-safe: a nil *Config, an absent agent.tiers block, or
// an unconfigured tier all report (zero, false). The bool lets a caller
// distinguish "no override" from "override present but with empty fields" — the
// distinction internal/agent.Resolve relies on for per-field merge over the
// fab-kit default.
func (c *Config) GetAgentTier(tier string) (TierProfile, bool) {
	if c == nil || c.Agent.Tiers == nil {
		return TierProfile{}, false
	}
	p, ok := c.Agent.Tiers[tier]
	return p, ok
}

// GetLinearWorkspace returns project.linear_workspace, or "" when unset (nil-safe).
func (c *Config) GetLinearWorkspace() string {
	if c == nil {
		return ""
	}
	return c.Project.LinearWorkspace
}
