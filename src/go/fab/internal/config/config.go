package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configvalue"
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

// ProviderProfile is the {model, effort} a provider supplies when it plays ONE
// role — the `providers.<name>.profiles.<role>` entry. It carries no provider
// field (the provider is the map it hangs off) and no command (that lives on the
// ProviderConfig). Either field MAY be empty; an empty value falls through to the
// provider's `default`-role profile and then to empty, whose established meaning
// is spawn.WithProfile's token-drop (so the CLI's own default applies) and, on a
// resolve-agent `model=` line, "inherit the session model".
type ProviderProfile struct {
	Model  string `yaml:"model"`
	Effort string `yaml:"effort"`
}

// ProviderConfig models one entry of the top-level `providers:` table: a named
// invocation grammar for an agent harness, plus its per-role fills. Provider names
// are opaque, user-chosen strings (fab never infers a provider from a model
// string).
//
//   - SessionCommand opens an interactive agent SESSION (the relocated
//     agent.spawn_command semantics — consumed by fab operator / fab batch /
//     fab agent).
//   - DispatchCommand runs ONE headless stage task via fab dispatch.
//   - Native declares that the provider can run through the native Agent-tool
//     adapter. Provider names are opaque, so this capability is shipped data.
//
// The two command fields are deliberately NOT merged into one: session and
// dispatch are different invocations of the same binary (claude interactive `-n`
// vs headless `-p`; codex TUI vs `codex exec`), and no single template expresses
// both. Both strings pass through verbatim — fab applies NO validation against any
// provider's accepted set (provider neutrality, Constitution Principle I). The
// {model}/{effort} placeholders are substituted at resolve time via internal/spawn.
//
// Profiles is the provider's PER-ROLE FILL: "when this provider plays this role,
// use this model/effort". It is keyed by role name, and the `default` role doubles
// as the provider's cross-role fallback. It is the only cross-role fallback chain
// in the resolver — the agent side has none — and sits below an explicit
// agent.profiles.<role> field in the fill precedence:
//
//	invocation flag  >  agent.profiles.<role>.<field>
//	                 >  providers.<p>.profiles.<role>.<field>
//	                 >  providers.<p>.profiles.default.<field>  >  empty
//
// The deprecated flat providers.<p>.<field> is NOT a rung of its own — it is folded
// into that override's own profiles.default (see Model and Effort below).
//
// Model and Effort are the DEPRECATED FLAT FILL (pre-2.17.0
// providers.<name>.model/.effort). They are still read, as an ALIAS for the
// override's own Profiles["default"]: internal/agent.ResolveProvider folds them into
// that entry per field before merging fab-kit's built-in table, so a config that has
// not yet run the 2.16.19-to-2.17.0 migration keeps resolving AND keeps outranking
// the built-in fill it is trying to replace. The override's own profiles.default wins
// where it sets a field (the modern spelling beats its alias), and a built-in ROLE
// fill still outranks the folded default exactly as it outranks a hand-written
// profiles.default. New configs write providers.<name>.profiles.default instead. One
// fill per provider resolved the same model for every role, which is exactly the role
// differentiation the nested map restores.
//
// The whole table is scope `both`, so a machine-wide fill is settable once in
// ~/.fab-kit/config.yaml.
type ProviderConfig struct {
	SessionCommand  string `yaml:"session_command"`
	DispatchCommand string `yaml:"dispatch_command"`
	Native          bool   `yaml:"native"`
	// NativeSet preserves YAML presence so an explicit `native: false` can
	// override a built-in true value during the provider-table merge.
	NativeSet bool                       `yaml:"-"`
	Profiles  map[string]ProviderProfile `yaml:"profiles"`

	// Deprecated: the flat fill. Read as an ALIAS for Profiles["default"] — folded
	// into it per field by internal/agent.ResolveProvider; see the type doc.
	// Removed from the documented surface in 2.17.0.
	Model  string `yaml:"model"`
	Effort string `yaml:"effort"`
}

// UnmarshalYAML preserves whether native was explicitly present. A plain bool
// cannot distinguish absent from false, but provider overrides need that
// distinction to disable a built-in native capability deliberately.
func (p *ProviderConfig) UnmarshalYAML(value *yaml.Node) error {
	type plain ProviderConfig
	var decoded plain
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*p = ProviderConfig(decoded)
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			if value.Content[i].Value == "native" {
				p.NativeSet = true
				break
			}
		}
	}
	return nil
}

// RoleProfile is a named `{provider, model, effort}` agent profile — one entry of
// `agent.profiles`. Every field MAY be empty, and the map is deliberately SPARSE:
// an unset field is simply not an override, and resolution continues down the fill
// precedence (see ProviderConfig) rather than inheriting from another role. There
// is NO agent-side `default`-role inheritance — agent.profiles.default is the
// `default` role's own override, not a fallback source for the other five.
//
// Provider names the entry in the top-level `providers:` table whose command
// grammar this role's agents use; when unset, the role's depth knob
// (agent.session or agent.workers) supplies it. The command itself lives on the
// provider, NOT the role.
//
// All strings are pass-through — fab applies NO validation (provider neutrality,
// Constitution Principle I). See internal/agent for resolution.
type RoleProfile struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
}

// AgentConfig models the `agent:` section of config.yaml.
//
// Session and Workers are the two ADVERTISED knobs, selecting a provider by agent
// DEPTH: Session covers the Tier-1 roles (the agents a user talks to — `default`,
// `operator`), Workers the Tier-2 roles (the agents pipeline stages dispatch to —
// `doing`, `review`, `hydrate`, `fast`). Both default to `claude`. The role→depth
// partition is fab-owned and NOT user-overridable (internal/agent), as is the
// stage→role mapping; users say "claude for what I talk to, gemini for the
// workers" and stop there.
//
// Profiles is the sparse per-role escape hatch beneath them: a map of role name
// (default/operator/doing/review/hydrate/fast) → {provider, model, effort}, each
// field optional and each set field beating the knob (provider) or the provider's
// own fill (model/effort).
//
// Tiers is the DEPRECATED spelling of Profiles (pre-2.17.0 `agent.tiers`), read
// per role as a fallback so a config that has not yet run the 2.16.19-to-2.17.0
// migration keeps resolving. Note the semantics differ in one respect that the
// migration does not paper over: the old `tiers` map re-based every unset field
// from its `default` tier, and that inheritance is gone.
//
// yaml.v3 ignores unknown keys, so widening this struct is free for existing
// configs (the same property that made stage_hooks free).
type AgentConfig struct {
	Session  string                 `yaml:"session"`
	Workers  string                 `yaml:"workers"`
	Profiles map[string]RoleProfile `yaml:"profiles"`

	// Deprecated: the pre-2.17.0 spelling of Profiles. See the type doc.
	Tiers map[string]RoleProfile `yaml:"tiers"`
}

// ProjectConfig models the `project:` section of config.yaml.
type ProjectConfig struct {
	LinearWorkspace string `yaml:"linear_workspace"`
}

// DispatchConfig models the `dispatch:` section of config.yaml — machine-level
// dispatch preferences (scope `both`, so a single ~/.fab-kit/config.yaml setting
// covers every repo on the machine).
//
// Mode is the PREFERRED dispatch rung. Automatic resolution starts there and
// descends pane → native → headless, never ascending, until provider capability
// and environment make a rung possible. The default is native, which preserves
// the shipped built-in behavior: claude runs natively while codex/gemini descend
// to their headless dispatch commands.
//
// ColumnWidth is the WORKER-COLUMN WIDTH, in percent of the window, used by the
// column-carving `-h` split that opens a pane-mode worker beside its dispatching
// agent (`tmux split-window -h -l <n>%`). It exists because an even halving leaves
// the session agent — the pane the user actually watches — with only half the
// window from the moment the column is carved. Only the CARVING split is sized;
// later workers stack inside the column with unsized `-v` splits, and the
// Left/Right separator is never touched again.
//
// ReapDone is the DONE-WORKER REAPING policy read by `fab dispatch reap`: a
// pane-mode worker never exits on completion (it writes its result file and sits at
// its prompt), so without reaping every finished stage holds its slice of the carved
// column for the rest of the run. Default TRUE — space-reclaimed; setting it false
// preserves the leave-the-pane-alone behavior for anyone who wants a done worker's
// scrollback.
//
// It is a *bool, unlike its two siblings, because its default is TRUE: the Go zero
// value would then mean the OPPOSITE of the default, making an absent key
// indistinguishable from an explicit `reap_done: false` and silently disabling
// reaping for every project that never sets the key. nil = unset = the default; a
// non-nil pointer is the user's explicit choice either way.
type DispatchConfig struct {
	Mode        string `yaml:"mode"`
	ColumnWidth int    `yaml:"column_width"`
	ReapDone    *bool  `yaml:"reap_done"`
}

// DefaultDispatchMode is the built-in dispatch.mode preference. It is the
// canonical symbol both GetDispatchMode and internal/configref consume.
const DefaultDispatchMode = "native"

// DefaultDispatchColumnWidth is the built-in dispatch.column_width — the percent of
// the window a pane-mode worker column takes when it is carved, leaving the
// dispatching agent the rest. It is the canonical symbol both the accessor below
// and internal/configref's registry row read, so the default exists once.
const DefaultDispatchColumnWidth = 35

// DefaultDispatchReapDone is the built-in dispatch.reap_done — whether a done
// pane-mode worker's tmux pane is reclaimed by `fab dispatch reap`. Like
// DefaultDispatchColumnWidth it is the canonical symbol both the accessor below and
// internal/configref's registry row read, so the default exists once.
const DefaultDispatchReapDone = true

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
	Dispatch   DispatchConfig            `yaml:"dispatch"`
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
// config after resolving the four-layer cascade at this single seam:
//
//	env  >  project (the path given)  >  system (~/.fab-kit/config.yaml)  >  built-in defaults
//
// The env overlay and two FILES merge here at the YAML map level (per-field deep
// merge: maps merge per-key recursively, lists replace, scalars replace, env
// wins); the built-in-defaults layer stays where it lives today — the
// point-of-use fallbacks (internal/agent's role/provider resolution, the nil-safe
// accessors) — which composes to four-layer semantics with zero per-caller change.
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
	envMap, _ := loadEnvLayer()

	// Merge project OVER system, then env OVER both. A nil project map (file
	// absent) still lets the system layer through; a nil env map preserves the
	// pre-env file merge byte-for-byte.
	merged := deepMerge(deepMerge(systemMap, projectMap), envMap)

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
	// Env is the recognized, scope-eligible environment overlay. EnvOrigins maps
	// each dotted registry key present in Env to the environment variable that
	// supplied it (without the display-only '$' prefix).
	Env        map[string]any
	EnvOrigins map[string]string
	// Effective is deepMerge(deepMerge(System, Project), Env) — the merged tree
	// LoadPath unmarshals.
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
	envMap, envOrigins := loadEnvLayer()
	sysPath, _ := systemConfigPath() // "" only if HOME is unresolvable (fail-open)
	return &Layers{
		ProjectPath: projectPath,
		SystemPath:  sysPath,
		Project:     projectMap,
		System:      systemMap,
		Env:         envMap,
		EnvOrigins:  envOrigins,
		Effective:   deepMerge(deepMerge(systemMap, projectMap), envMap),
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

// ParseYAMLValue parses one environment override through the shared value
// parser, including its collection support in either YAML style, then decodes it to the generic
// shape the layer merge uses. Mutation callers apply a narrower scalar-only
// contract on top of that parser.
func ParseYAMLValue(raw string) (any, error) {
	parsed, err := configvalue.Parse(raw)
	if err != nil {
		return nil, err
	}
	var value any
	if err := parsed.Node.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// loadEnvLayer builds the highest-precedence config overlay by walking the
// cycle-free dotted registry enumeration FORWARD. It deliberately never scans
// FAB_* variables or reverse-parses their names: underscores in a key segment
// would otherwise be ambiguous with the dots replaced by underscores.
//
// Only system/both-scoped rows are eligible. Project-scoped variables warn and
// are ignored to preserve repository reproducibility. Empty values behave as
// unset; malformed or Config-incompatible YAML warns and is skipped. Every path
// is fail-open — an environment preference must never brick config loading.
func loadEnvLayer() (map[string]any, map[string]string) {
	var overlay map[string]any
	var origins map[string]string
	for _, key := range configscope.DottedKeys() {
		envName := envNameForKey(key)
		raw, set := os.LookupEnv(envName)
		if !set || raw == "" {
			continue
		}

		scope, known := configscope.ScopeFor(topLevel(key))
		if !known {
			continue // parity/lint tests make this unreachable; stay fail-open
		}
		if scope == configscope.ScopeProject {
			fmt.Fprintf(warnw, "fab: warning: ignoring project-scoped environment override $%s for %q (project-scoped fields belong in fab/project/config.yaml)\n", envName, key)
			continue
		}

		value, err := ParseYAMLValue(raw)
		if err != nil {
			fmt.Fprintf(warnw, "fab: warning: ignoring malformed environment override $%s for %q (%v)\n", envName, key, err)
			continue
		}
		fragment := make(map[string]any)
		setDotted(fragment, key, value)
		if err := validateConfigFragment(fragment); err != nil {
			fmt.Fprintf(warnw, "fab: warning: ignoring malformed environment override $%s for %q (%v)\n", envName, key, err)
			continue
		}
		if overlay == nil {
			overlay = make(map[string]any)
			origins = make(map[string]string)
		}
		overlay = deepMerge(overlay, fragment)
		origins[key] = envName
	}
	return overlay, origins
}

// validateConfigFragment checks one environment variable independently against
// Config's YAML shape. Keeping the probe per-variable preserves fail-open
// behavior: one incompatible override is skipped without discarding valid env
// values or allowing it to poison the final merged-tree unmarshal.
func validateConfigFragment(fragment map[string]any) error {
	data, err := yaml.Marshal(fragment)
	if err != nil {
		return err
	}
	var probe Config
	return yaml.Unmarshal(data, &probe)
}

func envNameForKey(key string) string {
	return "FAB_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
}

func topLevel(key string) string {
	if i := strings.IndexByte(key, '.'); i >= 0 {
		return key[:i]
	}
	return key
}

// setDotted nests value beneath key's dotted path in root. Every intermediate
// mapping is created by this helper from the registry enumeration, so replacing a
// non-map intermediate is safe and deterministic.
func setDotted(root map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	cur := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			cur[part] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = value
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

// ProviderNames returns the provider names configured in the project's
// `providers:` block, in unspecified order (callers that need stability sort).
// Nil-safe: a nil *Config or an absent providers block yields nil. Callers that
// want the RESOLVABLE set (project ∪ fab-kit's built-in table) use
// internal/agent.ProviderNames, which owns the built-in table.
func (c *Config) ProviderNames() []string {
	if c == nil || c.Providers == nil {
		return nil
	}
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

// GetAgentProfile returns the configured override profile for a role name and
// whether one was set. Nil-safe: a nil *Config, an absent agent.profiles block, or
// an unconfigured role all report (zero, false). The bool lets a caller
// distinguish "no override" from "override present but with empty fields" — the
// distinction internal/agent's resolution relies on.
//
// The lookup is PER ROLE, not per block: a role absent from agent.profiles falls
// back to the deprecated agent.tiers spelling for that role, so a half-migrated
// config (some roles moved, some not) resolves every role. agent.profiles wins
// whenever it carries the role.
//
// LIMITATION — the alias resolves AFTER the scope cascade, so for a role written in
// the NEW spelling in one scope and the LEGACY spelling in another, the spelling
// decides, not the scope: LoadPath merges the system (~/.fab-kit/config.yaml) and
// project layers per key first, leaving `profiles` and `tiers` as two separate maps,
// and this accessor then prefers `profiles` wherever it carries the role. So a
// SYSTEM-layer agent.profiles.<role> beats a PROJECT-layer agent.tiers.<role>,
// inverting the documented project > system precedence. It only bites a
// hand-half-migrated pair of scopes; running the 2.16.19-to-2.17.0 migration (which
// sweeps BOTH files) removes the legacy spelling from both and restores the normal
// precedence, as does moving the role to `profiles` in the losing scope. Making the
// alias cascade-aware would mean tracking per-scope, per-key provenance through
// LoadPath for a spelling that exists only for the pre-migration window — a cost
// out of proportion to a transitional key. Pinned by
// TestResolveCrossScopeLegacyAliasPrecedence in cmd/fab.
func (c *Config) GetAgentProfile(role string) (RoleProfile, bool) {
	if c == nil {
		return RoleProfile{}, false
	}
	if p, ok := c.Agent.Profiles[role]; ok {
		return p, true
	}
	p, ok := c.Agent.Tiers[role]
	return p, ok
}

// GetAgentSession returns agent.session — the provider knob for the Tier-1
// (session) roles — or "" when unset (nil-safe). internal/agent supplies the
// built-in fallback; an empty string here means "the knob is not configured", not
// "no provider".
func (c *Config) GetAgentSession() string {
	if c == nil {
		return ""
	}
	return c.Agent.Session
}

// GetAgentWorkers returns agent.workers — the provider knob for the Tier-2
// (dispatched worker) roles — or "" when unset (nil-safe). Same empty-means-unset
// contract as GetAgentSession.
func (c *Config) GetAgentWorkers() string {
	if c == nil {
		return ""
	}
	return c.Agent.Workers
}

// GetDispatchMode returns the validated dispatch.mode preference (nil-safe).
// Absent resolves to DefaultDispatchMode. Invalid values warn and fail open to
// the same default: a malformed personal preference must not brick every repo.
func (c *Config) GetDispatchMode() string {
	if c == nil || c.Dispatch.Mode == "" {
		return DefaultDispatchMode
	}
	switch c.Dispatch.Mode {
	case "pane", "native", "headless":
		return c.Dispatch.Mode
	default:
		fmt.Fprintf(warnw, "fab: warning: invalid dispatch.mode %q; using %q\n", c.Dispatch.Mode, DefaultDispatchMode)
		return DefaultDispatchMode
	}
}

// GetDispatchColumnWidth returns dispatch.column_width — the pane-worker column's
// width in percent — or DefaultDispatchColumnWidth when unset or out of range
// (nil-safe).
//
// An ABSENT yaml int is indistinguishable from an explicit 0, so 0 cannot mean
// "carve an unsized column"; it reads as unset and resolves to the default. Values
// outside 1..99 resolve to the default for the same reason they are nonsense as a
// percentage: 0 would give the worker nothing and 100 would leave the dispatching
// agent nothing — the exact outcome the knob exists to prevent.
func (c *Config) GetDispatchColumnWidth() int {
	if c == nil {
		return DefaultDispatchColumnWidth
	}
	if w := c.Dispatch.ColumnWidth; w > 0 && w < 100 {
		return w
	}
	return DefaultDispatchColumnWidth
}

// GetDispatchReapDone returns dispatch.reap_done — whether `fab dispatch reap`
// reclaims a done pane worker's tmux pane — or DefaultDispatchReapDone (true) when
// unset (nil-safe in both senses: a nil *Config and a nil field both read as unset).
//
// The POINTER is what makes this accessor honest, and it is the one place the
// difference shows: a default-TRUE bool cannot ride the Go zero value, so
// `reap_done: false` must be storable as a real value rather than collapsing into
// "absent". See DispatchConfig.
func (c *Config) GetDispatchReapDone() bool {
	if c == nil || c.Dispatch.ReapDone == nil {
		return DefaultDispatchReapDone
	}
	return *c.Dispatch.ReapDone
}

// GetLinearWorkspace returns project.linear_workspace, or "" when unset (nil-safe).
func (c *Config) GetLinearWorkspace() string {
	if c == nil {
		return ""
	}
	return c.Project.LinearWorkspace
}
