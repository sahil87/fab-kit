// Package agent owns fab-kit's per-stage model selection: the six agent ROLES and
// their fixed depth partition, the FIXED stage→role mapping, the built-in provider
// table with its per-role fills, and the resolution cascade consumed by
// `fab resolve-agent <stage|role>`, `fab agent`, and the operator launcher.
//
// VOCABULARY. A ROLE is one of six fixed slot names (default, operator, doing,
// review, hydrate, fast). A PROFILE is a concrete {provider, model, effort} value.
// A PROVIDER is an invocation grammar plus its per-role fills. TIER 1 / TIER 2 name
// agent DEPTH — the agents a user talks to versus the agents pipeline stages
// dispatch to — and that axis is what the two advertised knobs (agent.session,
// agent.workers) select a provider by.
//
// The tables here are fab-kit's curated judgment, and they are split across two
// files by whether a user can override them. The knobs and the provider table
// (grammars plus claude's per-role fills) are DATA: they live in defaults.yaml
// (embedded below), shaped as a config-file fragment, and defaults.yaml is the
// single place to bump when a new top model lands (the "Fable upgrade path"). The
// stage→role mapping and the role→depth partition are POLICY: they stay Go maps
// here and are NOT user-overridable (there is no stage_roles config and no
// per-stage escape hatch). Users override only which provider each depth uses, via
// agent.session/agent.workers; what a single role means, via the sparse
// agent.profiles map; and which command grammars exist, via providers:.
//
// THE FILL PRECEDENCE, implemented once here (ResolveRoleWith):
//
//	provider:  invocation --provider  >  agent.profiles.<role>.provider
//	           >  the role's depth knob  >  the built-in claude
//	model /    invocation flag  >  agent.profiles.<role>.<field>
//	effort:    >  providers.<p>.profiles.<role>.<field>
//	           >  providers.<p>.profiles.default.<field>  >  empty
//
// There is exactly ONE cross-role fallback chain and it lives on the PROVIDER side
// (a provider's `default` role fill). The agent side has none: agent.profiles is
// sparse, and agent.profiles.default is the `default` role's own override, never a
// fallback source for another role. That single-chain shape is what retired the
// cross-provider cutoff rule — with model/effort always sourced from the resolved
// provider's own fills, no value can be foreign to the provider that will run it.
//
// Resolution applies NO validation — it echoes the resolved {provider, model,
// effort} verbatim, whatever they are (provider neutrality, Constitution
// Principle I). Compatibility is the runtime/harness's concern, not fab's.
//
// The role profiles and the stage→role map are mirrored in
// docs/specs/stage-models.md and guarded against drift by TestDocTablesMatchAgentMaps
// (stagemodels_doc_test.go), the same pattern internal/score uses for
// change-types.md.
package agent

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// defaultsYAML is fab-kit's built-in agent defaults — the depth knobs and the
// provider table (grammars plus claude's per-role fills) — compiled into the
// binary. It is deliberately EMBEDDED rather than read from the kit cache at
// runtime: kit and binary release atomically, so an on-disk read would gain
// nothing and add a binary↔kit version-skew failure mode to a resolution path that
// cannot fail today.
//
//go:embed defaults.yaml
var defaultsYAML []byte

// builtinDefaults is defaultsYAML parsed once, at package initialization, into
// the SAME struct config.LoadPath fills from a user's config.yaml. The file is
// shaped as a config-file fragment, so parsing it through the config schema is
// what keeps the two shapes from diverging — and is what will let this become
// layer 0 of the config cascade without a parser change.
var builtinDefaults = mustParseDefaults(defaultsYAML)

// mustParseDefaults parses the embedded defaults, panicking on a malformed file.
// The bytes are compiled into the binary, so a parse failure is a defective build
// artifact rather than a runtime condition a user can produce or recover from —
// returning an error would force an error path onto every DefaultProfile/ResolveRole
// caller for a state a released binary cannot reach. defaults_test.go is the
// safety net a YAML typo used to get from the compiler.
func mustParseDefaults(data []byte) *config.Config {
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprintf("internal/agent: embedded defaults.yaml is malformed: %v", err))
	}
	return &cfg
}

// Role names. Six roles with concrete referents. A role is stage-named only where
// it maps 1:1 to a single referent (review, hydrate); default, doing, and fast keep
// role names because each is multi-referent (fast governs the ship stage AND the
// /fab-proceed prefix-step dispatches).
const (
	RoleDefault  = "default"  // spawned worker sessions, `fab agent` with no role, intake (advisory), the /fab-proceed create-intake dispatch
	RoleOperator = "operator" // the operator coordinator session (`fab operator`)
	RoleDoing    = "doing"    // apply, review-pr — execution that must not err
	RoleReview   = "review"   // review — author/critic separation
	RoleHydrate  = "hydrate"  // hydrate — memory writing
	RoleFast     = "fast"     // ship, the /fab-proceed prefix steps — speed on near-mechanical work
)

// Built-in provider names — the three keys defaults.yaml defines under
// `providers:`. DefaultProviderName is the provider a role resolves to when
// neither an agent.profiles override nor the role's depth knob names one; the
// other two are named here so the lookups below (and the validation test) carry no
// bare string.
const (
	DefaultProviderName = "claude"

	providerCodex  = "codex"
	providerGemini = "gemini"
)

// DefaultSessionCommand is the built-in claude provider's session command — the
// relocated agent.spawn_command default. Kept here (not internal/spawn) because
// the provider table is agent-owned; internal/spawn re-exports the string for its
// own no-config fallback.
//
// It is a TEMPLATE: the trailing {model}/{effort} placeholders are substituted
// with the resolved role profile by spawn.WithProfile's template mode (see
// internal/spawn). Placing them at the END makes the resolved command
// byte-identical to what WithProfile's append mode produced for the former plain
// form — zero behavior change, so the templated form is purely an explicitness
// upgrade (the substitution point is now visible in the command itself, matching
// the codex/gemini starter templates).
//
// It is a var rather than a const because defaults.yaml owns the string: a const
// here would mean the same command text lived in two places, which is the drift
// the data file exists to make impossible.
var DefaultSessionCommand = defaultProviders[DefaultProviderName].SessionCommand

// The codex and gemini built-in provider commands (260805-j3cm). These are
// GRAMMAR ONLY — the invocation templates, carrying no fills. Grammar changes at
// binary-release cadence and is safe to ship; model IDs rot in weeks, so a
// non-claude built-in NEVER carries one (a project or the system config supplies it
// via providers.<name>.profiles, an agent.profiles field, or an invocation flag —
// see the fill precedence in the package doc).
//
// Both providers carry a dispatch_command (unlike claude, which carries none),
// so pointing a role at one flips that role's stages from native Agent-tool
// dispatch to CLI dispatch — which is exactly what selecting a non-claude
// provider means.
//
// These are the canonical names internal/configref interpolates into the rendered
// reference, so the reference text carries no literal copy (the same
// no-duplicate-literal rule DefaultSessionCommand follows). Their values, like
// every other string here, come from defaults.yaml.
var (
	// DefaultCodexSessionCommand opens an interactive codex TUI session.
	DefaultCodexSessionCommand = defaultProviders[providerCodex].SessionCommand
	// DefaultCodexDispatchCommand runs one headless codex task. `codex exec` is a
	// SUBCOMMAND (not a flag) and reads the prompt from stdin, which is where
	// `fab dispatch` pipes it.
	DefaultCodexDispatchCommand = defaultProviders[providerCodex].DispatchCommand
	// DefaultGeminiSessionCommand opens an interactive gemini session. It carries
	// NO {effort} placeholder — the gemini CLI has no reasoning-effort flag, so a
	// resolved effort has nowhere to go and is simply not injected.
	DefaultGeminiSessionCommand = defaultProviders[providerGemini].SessionCommand
	// DefaultGeminiDispatchCommand runs one headless gemini task. Deliberately has
	// no `-p`: gemini's -p takes prompt TEXT (appended after stdin), whereas
	// `fab dispatch` pipes the prompt to stdin, which gemini reads as the prompt in
	// non-TTY mode. Like the session command, it carries no {effort}.
	DefaultGeminiDispatchCommand = defaultProviders[providerGemini].DispatchCommand
)

// Profile is a concrete {provider, model, effort} triple. An empty Provider names
// no provider (the caller decides whether that is a lookup failure); an empty Model
// signals "inherit the session/orchestrator model"; an empty Effort omits effort
// entirely.
type Profile struct {
	Provider string
	Model    string
	Effort   string
}

// defaultProviders is fab-kit's built-in provider table: three providers, parsed
// from the `providers:` block of defaults.yaml verbatim.
//
//   - claude — the default: the default session command, NO dispatch_command
//     (native Agent-tool dispatch), and the six per-role fills.
//   - codex, gemini — session AND dispatch commands, GRAMMAR ONLY (no fills, since
//     a non-claude model ID rots at CLI cadence). Naming either resolves with ZERO
//     providers: config and flips those stages to CLI dispatch.
//
// A built-in provider is INERT until a knob, an agent.profiles entry, or a flag
// names it — adding a row changes no default behavior, which is why the
// presence=intent rule that keeps behavior-changing config commented does not force
// these rows out of the shipped binary.
//
// A project extends/overrides via its own providers: block, per-field merged over
// this (including a per-role, per-field merge of the profiles map).
var defaultProviders = builtinDefaults.Providers

// depth is the TIER of agent a role belongs to — the axis the two advertised knobs
// select a provider by.
type depth int

const (
	// depthSession is Tier 1: agents the user talks to (`fab agent`, `fab operator`,
	// `fab batch` worker sessions). Its provider comes from agent.session, and it
	// applies at LAUNCH time — fab cannot switch a running session's provider.
	depthSession depth = iota
	// depthWorkers is Tier 2: agents pipeline stages dispatch to. Its provider comes
	// from agent.workers, and it applies at every stage dispatch (`fab resolve-agent`).
	depthWorkers
)

// roleDepth is the FIXED, fab-owned role→depth partition, and doubles as the
// canonical set of role names (RoleNames/IsRoleName read it). It stays in Go —
// deliberately NOT in defaults.yaml — because the YAML/Go split doubles as the
// overridable/fixed signal: everything in that file is user-overridable by writing
// the same key in config.yaml, everything here is fab-owned policy.
//
// The split is mechanically real rather than cosmetic: a session role's provider is
// fixed when the session launches, while a workers role's provider is re-resolved at
// every dispatch. intake rides `default` and is a session role for exactly that
// reason — it runs foreground in the user's own session.
var roleDepth = map[string]depth{
	RoleDefault:  depthSession,
	RoleOperator: depthSession,
	RoleDoing:    depthWorkers,
	RoleReview:   depthWorkers,
	RoleHydrate:  depthWorkers,
	RoleFast:     depthWorkers,
}

// stageRoles is the FIXED, fab-owned stage→role mapping. Like roleDepth it stays in
// Go, with tested invariants (the review/hydrate fixed-point property below).
// Exhaustive over the six pipeline stages (each stage belongs to exactly one role).
// NOT user-overridable. Two stages share a name with their role (review, hydrate) —
// each such collision is a FIXED POINT (stageRoles[name] == name), which is what
// makes the role-first resolution order in RoleForName immaterial for those names
// (guarded by TestStageRoleCollisionsAreFixedPoints). ship maps to the fast role
// (fast is multi-referent — it also governs the /fab-proceed prefix steps — so it
// keeps its role name, not the stage name). Note review (own role — author/critic
// separation) and review-pr (responsive → doing) are in DIFFERENT roles despite
// sharing the word "review". intake maps to default but is ADVISORY only — it runs
// foreground in the user's own session, which fab cannot re-model. Mirrored in
// docs/specs/stage-models.md § stage→role table (drift-guarded).
var stageRoles = map[string]string{
	"intake":    RoleDefault,
	"apply":     RoleDoing,
	"review":    RoleReview,
	"hydrate":   RoleHydrate,
	"ship":      RoleFast,
	"review-pr": RoleDoing,
}

// DefaultProfile returns fab-kit's built-in profile for a role name and whether the
// role is known. It is defined as resolution against a NIL config — the built-in
// answer is exactly "what resolves when the user has configured nothing" — so there
// is no second built-in table to keep in sync with the resolver. Exposed for the
// drift-guard tests, `fab config reference`, and the launcher fallbacks.
func DefaultProfile(role string) (Profile, bool) {
	p, err := ResolveRole(nil, role)
	if err != nil {
		return Profile{}, false
	}
	return p, true
}

// RoleForStage returns the fixed role a stage maps to and whether the stage is
// known. Exposed for the drift-guard test and for error messages that name a
// stage's role.
func RoleForStage(stage string) (string, bool) {
	r, ok := stageRoles[stage]
	return r, ok
}

// IsRoleName reports whether name is one of the six known role names. Used by
// `fab resolve-agent` to accept a role name positionally alongside a stage name.
// The two sets overlap only at fixed points — a name shared by a stage and a role
// (review, hydrate) is one the stage maps to that same-named role
// (stageRoles[name] == name), so either interpretation resolves identically.
// ("ship" is a stage but NOT a role — it maps to the fast role.)
func IsRoleName(name string) bool {
	_, ok := roleDepth[name]
	return ok
}

// IsSessionRole reports whether a role sits at the SESSION depth (Tier 1 — the
// agents you talk to, provider from agent.session) rather than the WORKERS depth
// (Tier 2 — provider from agent.workers). It is the exported read of the fixed
// roleDepth partition, so no caller has to re-encode which roles are which — the
// rendered `agent:` reference segment in internal/configref derives its
// session/workers role lists from this. An unknown role reports false (it is not a
// session role); use IsRoleName to distinguish unknown from workers.
func IsSessionRole(role string) bool {
	return roleDepth[role] == depthSession && IsRoleName(role)
}

// RoleForName maps a positional `<stage|role>` argument to a role. ROLE NAMES ARE
// CHECKED FIRST; everything else is looked up as a stage, and a name that is
// neither yields the unknown-stage error naming the valid stage set. The role-first
// order is immaterial for results (see IsRoleName's fixed-point note).
func RoleForName(name string) (string, error) {
	if IsRoleName(name) {
		return name, nil
	}
	role, ok := stageRoles[name]
	if !ok {
		return "", fmt.Errorf("unknown stage %q (valid: %s)", name, strings.Join(StageNames(), ", "))
	}
	return role, nil
}

// RoleNames returns the known role names, sorted (stable for the drift-guard
// test's set comparison and for error messages).
func RoleNames() []string {
	names := make([]string, 0, len(roleDepth))
	for r := range roleDepth {
		names = append(names, r)
	}
	sort.Strings(names)
	return names
}

// StageNames returns the known stage names, sorted (stable for the drift-guard
// test's set comparison).
func StageNames() []string {
	names := make([]string, 0, len(stageRoles))
	for s := range stageRoles {
		names = append(names, s)
	}
	sort.Strings(names)
	return names
}

// modelAliasPrefixes maps a Claude full-ID family prefix to the Claude-Code
// short alias the Agent tool's `model` enum accepts (opus/sonnet/haiku/fable).
// Prefix-matched so dated/versioned variants (claude-haiku-4-5-20251001) resolve
// to their family alias.
var modelAliasPrefixes = []struct{ prefix, alias string }{
	{"claude-opus-", "opus"},
	{"claude-sonnet-", "sonnet"},
	{"claude-haiku-", "haiku"},
	{"claude-fable-", "fable"},
}

// ModelAlias maps a full Claude model ID to its Claude-Code short alias (the
// Agent tool's `model` enum: opus/sonnet/haiku/fable). Returns the input VERBATIM
// when no mapping applies — an empty string (preserving the "inherit the session
// model" signal) or an unrecognized/non-Claude ID. This keeps the alias adapter
// from becoming a Claude-only validator (provider neutrality): a role pointed at
// another provider's model still gets its string through unchanged. Matched by
// family prefix so claude-haiku-4-5-20251001 → haiku.
func ModelAlias(model string) string {
	for _, m := range modelAliasPrefixes {
		if strings.HasPrefix(model, m.prefix) {
			return m.alias
		}
	}
	return model
}

// Overrides is an invocation-time {provider, model, effort} override set — the
// top rung of the fill precedence, supplied by flags rather than config
// (`fab resolve-agent <stage|role> [--provider] [--model] [--effort]`). Each field
// is applied only when its Set companion is true, so an explicitly-empty flag value
// is distinguishable from an absent flag (the same Flag.Changed discipline
// `fab agent` uses).
type Overrides struct {
	Provider    string
	ProviderSet bool
	Model       string
	ModelSet    bool
	Effort      string
	EffortSet   bool
}

// ResolveRole resolves a role name → a concrete {provider, model, effort} profile
// from config alone (no invocation overrides). See ResolveRoleWith.
func ResolveRole(cfg *config.Config, role string) (Profile, error) {
	return ResolveRoleWith(cfg, role, Overrides{})
}

// ResolveRoleWith resolves a role name → a concrete {provider, model, effort}
// profile, applying an invocation-time override set as the top rung. It is the
// SINGLE implementation of the fill precedence documented in the package doc:
//
//	provider:  --provider  >  agent.profiles.<role>.provider  >  depth knob  >  claude
//	model /    flag  >  agent.profiles.<role>.<field>
//	effort:    >  providers.<p>.profiles.<role>.<field>
//	           >  providers.<p>.profiles.default.<field>  >  empty
//
// where <p> is the RESOLVED provider — so a --provider swap re-derives model and
// effort from the new provider's own fills rather than carrying the old provider's
// values. That is what makes a swap safe WITHOUT the retired cross-provider cutoff:
// nothing is inherited across providers in the first place. An explicit
// agent.profiles.<role>.model still wins over the swap — a pin the user wrote is the
// user's own escape hatch, not inheritance.
//
// The depth knob is applied only when the provider was NOT supplied on the command
// line, so an explicitly-empty `--provider=` resolves an empty provider (a LOOKUP
// failure for the caller to report) rather than falling through to the knob.
//
// NO validation: the resolved fields are returned verbatim. An unknown role is the
// only role-resolution error; an unknown PROVIDER name is a lookup concern for the
// caller (which lists ProviderNames and exits non-zero), not an error here.
func ResolveRoleWith(cfg *config.Config, role string, o Overrides) (Profile, error) {
	if !IsRoleName(role) {
		return Profile{}, fmt.Errorf("unknown role %q (valid: %s)", role, strings.Join(RoleNames(), ", "))
	}

	override, _ := cfg.GetAgentProfile(role)

	provider := override.Provider
	if provider == "" {
		provider = depthProvider(cfg, role)
	}
	if o.ProviderSet {
		provider = o.Provider
	}

	prov, _ := ResolveProvider(cfg, provider)
	fillModel, fillEffort := providerFill(prov, role)

	p := Profile{
		Provider: provider,
		Model:    firstNonEmpty(override.Model, fillModel),
		Effort:   firstNonEmpty(override.Effort, fillEffort),
	}
	if o.ModelSet {
		p.Model = o.Model
	}
	if o.EffortSet {
		p.Effort = o.Effort
	}
	return p, nil
}

// Resolve maps a stage → its fixed role → a concrete {provider, model, effort}
// profile (via ResolveRole). An unknown stage is the only resolution-side error.
func Resolve(cfg *config.Config, stage string) (Profile, error) {
	role, ok := stageRoles[stage]
	if !ok {
		return Profile{}, fmt.Errorf("unknown stage %q (valid: %s)", stage, strings.Join(StageNames(), ", "))
	}
	// stageRoles only ever names roles present in roleDepth (guarded by the
	// drift-guard test), so ResolveRole cannot miss on a known stage.
	return ResolveRole(cfg, role)
}

// depthProvider returns the provider the role's DEPTH KNOB names: agent.session for
// a Tier-1 role, agent.workers for a Tier-2 one, then defaults.yaml's own knob
// value, then the built-in claude. The last rung is belt-and-braces — defaults.yaml
// ships both knobs — so a hand-trimmed defaults file can never resolve a role to no
// provider at all.
func depthProvider(cfg *config.Config, role string) string {
	configured, builtin := cfg.GetAgentWorkers(), builtinDefaults.Agent.Workers
	if roleDepth[role] == depthSession {
		configured, builtin = cfg.GetAgentSession(), builtinDefaults.Agent.Session
	}
	return firstNonEmpty(configured, builtin, DefaultProviderName)
}

// providerFill returns the {model, effort} a resolved provider supplies for a role:
// its own `profiles.<role>` entry, then its `profiles.default` entry (the provider's
// cross-role fallback), then the DEPRECATED flat providers.<name>.model/.effort
// fill, per field. The flat rung exists only so a config that has not yet run the
// 2.16.19-to-2.17.0 migration keeps resolving; it sits below profiles.default
// because that is the modern spelling of the same value.
func providerFill(prov config.ProviderConfig, role string) (model, effort string) {
	forRole := prov.Profiles[role]
	forDefault := prov.Profiles[RoleDefault]
	return firstNonEmpty(forRole.Model, forDefault.Model, prov.Model),
		firstNonEmpty(forRole.Effort, forDefault.Effort, prov.Effort)
}

// firstNonEmpty returns the first non-empty string, or "" when all are empty. It is
// the whole of the per-field fill precedence: each rung is one argument, in order.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// ProviderNames returns the provider names ResolveProvider can resolve for cfg:
// the union of fab-kit's built-in provider table and the project's own providers:
// block, sorted (stable for error messages and tests). Exposed so a lookup failure
// (`fab agent --provider <unknown>`) can name the available set rather than leaving
// the caller to guess it. Listing the resolvable NAMES is not validation of any
// command's CONTENT — the document-don't-validate contract is untouched.
func ProviderNames(cfg *config.Config) []string {
	seen := make(map[string]struct{}, len(defaultProviders))
	for name := range defaultProviders {
		seen[name] = struct{}{}
	}
	for _, name := range cfg.ProviderNames() {
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveProvider returns the {session_command, dispatch_command, profiles} for a
// provider name: the project's providers.<name> override PER-FIELD merged over
// fab-kit's built-in provider table (an override field that is set wins; an omitted
// field inherits the built-in). The profiles map merges per ROLE and then per FIELD,
// so overriding `providers.claude.profiles.review.model` leaves the other five roles
// (and review's effort) on the built-in values. A provider present in neither the
// project config nor the built-in table resolves to a zero ProviderConfig with
// ok=false — the caller decides whether that is an error (a session with no
// session_command, or a dispatch with no dispatch_command, are the two failure
// surfaces).
//
// The returned Profiles map is always a fresh map, never the built-in table's own,
// so a caller cannot mutate the shipped defaults through it.
//
// NO validation: every string is returned verbatim.
func ResolveProvider(cfg *config.Config, name string) (config.ProviderConfig, bool) {
	resolved, known := defaultProviders[name]
	resolved.Profiles = mergeProviderProfiles(resolved.Profiles, nil)

	if override, ok := cfg.GetProvider(name); ok {
		known = true
		mergeField(&resolved.SessionCommand, override.SessionCommand)
		mergeField(&resolved.DispatchCommand, override.DispatchCommand)
		mergeField(&resolved.Model, override.Model)
		mergeField(&resolved.Effort, override.Effort)
		resolved.Profiles = mergeProviderProfiles(resolved.Profiles, override.Profiles)
	}

	return resolved, known
}

// mergeProviderProfiles returns a FRESH per-role fill map: base copied, then over
// merged in per role and per field (a set field wins; an empty one inherits). A nil
// `over` therefore yields a defensive copy of base — which is how ResolveProvider
// detaches the built-in table's map before handing it out.
func mergeProviderProfiles(base, over map[string]config.ProviderProfile) map[string]config.ProviderProfile {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	merged := make(map[string]config.ProviderProfile, len(base)+len(over))
	for role, p := range base {
		merged[role] = p
	}
	for role, p := range over {
		existing := merged[role]
		mergeField(&existing.Model, p.Model)
		mergeField(&existing.Effort, p.Effort)
		merged[role] = existing
	}
	return merged
}

// mergeField overwrites *dst with v only when v is non-empty (per-field merge:
// a set override field wins; an empty field inherits).
func mergeField(dst *string, v string) {
	if v != "" {
		*dst = v
	}
}
