// Package agent owns fab-kit's per-stage model selection: the default
// tier→{provider, model, effort} table, the FIXED stage→tier mapping, the
// built-in provider table, and the resolution cascade consumed by
// `fab resolve-agent <stage>`, `fab agent`, and the operator launcher.
//
// The tables here are fab-kit's curated judgment. The stage→tier mapping is NOT
// user-overridable (there is no stage_tiers config and no per-stage escape
// hatch); the default tier→profile table is the single place to bump when a new
// top model lands (the "Fable upgrade path"). Users override only what each tier
// MEANS, via agent.tiers in config.yaml (per-field merge over the default), and
// which command grammars exist, via the top-level providers: table.
//
// The built-in provider table carries GRAMMAR for three providers (claude, codex,
// gemini) and fill values for NONE of them: a non-claude provider's model ID rots
// at CLI cadence, so it lives in user config (providers.<name>.model/.effort, a
// tier field, or an invocation flag). The fill precedence — invocation flag >
// explicit tier field > provider default fill > empty — is implemented once, in
// ResolveTier (config path) and ApplyOverrides (invocation path).
//
// Resolution applies NO validation — it echoes the resolved {provider, model,
// effort} verbatim, whatever they are (provider neutrality, Constitution
// Principle I). Compatibility is the runtime/harness's concern, not fab's.
//
// The two tables (defaultTiers, stageTiers) are mirrored in
// docs/specs/stage-models.md and guarded against drift by
// TestDocTablesMatchAgentMaps (stagemodels_doc_test.go), the same pattern
// internal/score uses for change-types.md.
package agent

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Role-tier names. Six tiers with concrete referents. A tier is stage-named only
// where it maps 1:1 to a single referent (review, hydrate); default, doing, and
// fast keep role names because each is multi-referent (fast governs the ship stage
// AND the /fab-proceed prefix-step dispatches).
const (
	TierDefault  = "default"  // spawned worker sessions, `fab agent` with no tier, intake (advisory), the /fab-proceed create-intake dispatch; per-field fallback for every other tier
	TierOperator = "operator" // the operator coordinator session (`fab operator`)
	TierDoing    = "doing"    // apply, review-pr — execution that must not err
	TierReview   = "review"   // review — author/critic separation
	TierHydrate  = "hydrate"  // hydrate — memory writing
	TierFast     = "fast"     // ship, the /fab-proceed prefix steps — speed on near-mechanical work
)

// DefaultProviderName is the built-in provider a fresh config resolves to when a
// tier declares no provider and the project sets no `default` tier provider.
const DefaultProviderName = "claude"

// DefaultSessionCommand is the built-in claude provider's session command — the
// relocated agent.spawn_command default. Kept here (not internal/spawn) because
// the provider table is agent-owned; internal/spawn re-exports the string for its
// own no-config fallback.
//
// It is a TEMPLATE: the trailing {model}/{effort} placeholders are substituted
// with the resolved tier profile by spawn.WithProfile's template mode (see
// internal/spawn). Placing them at the END makes the resolved command
// byte-identical to what WithProfile's append mode produced for the former plain
// form — zero behavior change, so the templated form is purely an explicitness
// upgrade (the substitution point is now visible in the command itself, matching
// the codex/gemini starter templates).
const DefaultSessionCommand = `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`

// The codex and gemini built-in provider commands (260805-j3cm). These are
// GRAMMAR ONLY — the invocation templates, carrying no model or effort fill
// values. Grammar changes at binary-release cadence and is safe to ship; model
// IDs rot in weeks, so a non-claude built-in NEVER carries one (a project or the
// system config supplies it via providers.<name>.model / .effort, a tier field,
// or an invocation flag — see the fill precedence in ResolveTier/ApplyOverrides).
//
// Both providers carry a dispatch_command (unlike claude, which carries none),
// so naming one in a tier flips that tier's stages from native Agent-tool
// dispatch to CLI dispatch — which is exactly what selecting a non-claude
// provider means.
//
// These are the canonical constants internal/configref interpolates into the
// rendered reference, so the reference text carries no literal copy (the same
// no-duplicate-literal rule DefaultSessionCommand follows).
const (
	// DefaultCodexSessionCommand opens an interactive codex TUI session.
	DefaultCodexSessionCommand = `codex -m {model} -c model_reasoning_effort={effort}`
	// DefaultCodexDispatchCommand runs one headless codex task. `codex exec` is a
	// SUBCOMMAND (not a flag) and reads the prompt from stdin, which is where
	// `fab dispatch` pipes it.
	DefaultCodexDispatchCommand = `codex exec -m {model} -c model_reasoning_effort={effort}`
	// DefaultGeminiSessionCommand opens an interactive gemini session. It carries
	// NO {effort} placeholder — the gemini CLI has no reasoning-effort flag, so a
	// resolved effort has nowhere to go and is simply not injected.
	DefaultGeminiSessionCommand = `gemini -m {model}`
	// DefaultGeminiDispatchCommand runs one headless gemini task. Deliberately has
	// no `-p`: gemini's -p takes prompt TEXT (appended after stdin), whereas
	// `fab dispatch` pipes the prompt to stdin, which gemini reads as the prompt in
	// non-TTY mode. Like the session command, it carries no {effort}.
	DefaultGeminiDispatchCommand = `gemini -m {model}`
)

// Profile is a concrete {provider, model, effort} triple. An empty Provider names
// no provider (resolution falls through to the built-in default provider at
// command-composition time); an empty Model signals "inherit the
// session/orchestrator model"; an empty Effort omits effort entirely.
type Profile struct {
	Provider string
	Model    string
	Effort   string
}

// defaultProviders is fab-kit's built-in provider table: three providers, all
// GRAMMAR ONLY (no model/effort fill values — see the command constants above).
//
//   - claude — the default: the default session command and NO dispatch_command
//     (native Agent-tool dispatch).
//   - codex, gemini — session AND dispatch commands (260805-j3cm). Naming either
//     in a tier or on an invocation flag resolves with ZERO providers: config, and
//     flips those stages to CLI dispatch (the dispatch_command is present).
//
// A built-in provider is INERT until a tier or a flag names it — adding a row
// changes no default behavior, which is why the presence=intent rule that keeps
// behavior-changing config commented does not force these rows out of Go
// (260805-j3cm reverses ho9y's "no new built-in providers in Go" narrowly, for
// grammar strings only).
//
// A project extends/overrides via its own providers: block, per-field merged over
// this (including the model/effort fill fields).
var defaultProviders = map[string]config.ProviderConfig{
	DefaultProviderName: {SessionCommand: DefaultSessionCommand},
	"codex": {
		SessionCommand:  DefaultCodexSessionCommand,
		DispatchCommand: DefaultCodexDispatchCommand,
	},
	"gemini": {
		SessionCommand:  DefaultGeminiSessionCommand,
		DispatchCommand: DefaultGeminiDispatchCommand,
	},
}

// defaultTiers is fab-kit's built-in tier→profile table (today). This is the ONE
// place bumped when a new top model lands. Provider is written explicitly on
// every line (documented style; inheritance is the safety net).
//
// TO BUMP A MODEL: edit the line here, then run the tests. Every other place that
// restates these values is drift-guarded and will name itself in the failure
// output — you do not have to go find them:
//
//	TestDefaultTierProfilesArePinned      the deliberate-change pin (agent_test.go)
//	TestDocTablesMatchAgentMaps           stage-models.md § default-tier TABLE
//	TestMirrorDocsMatchDefaultTiers       stage-models.md inline-YAML sample
//	TestCLIFabReferenceListsDefaultTiers  _cli-fab.md § resolve-agent enumeration
//
// No other test hardcodes these strings — the rest derive from DefaultTier(), so a
// bump does not touch them. Docs that used to restate the profiles (architecture.md,
// _shared/configuration.md, runtime/providers-and-tiers.md) now point at
// `fab config reference`, which renders this map live and cannot go stale.
var defaultTiers = map[string]Profile{
	TierDefault:  {Provider: "claude", Model: "claude-fable-5", Effort: "high"},
	TierOperator: {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	TierDoing:    {Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"},
	TierReview:   {Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"},
	TierHydrate:  {Provider: "claude", Model: "claude-opus-5", Effort: "high"},
	TierFast:     {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
}

// stageTiers is the FIXED, fab-owned stage→tier mapping. Exhaustive over the six
// pipeline stages (each stage belongs to exactly one tier). NOT user-overridable.
// Two stages share a name with their tier (review, hydrate) — each such collision
// is a FIXED POINT (stageTiers[name] == name), which is what makes the tier-first
// resolution order in cmd/fab.resolveStageOrTier immaterial for those names
// (guarded by TestStageTierCollisionsAreFixedPoints). ship maps to the fast tier
// (fast is multi-referent — it also governs the /fab-proceed prefix steps — so it
// keeps its role name, not the stage name). Note review (own tier — author/critic
// separation) and review-pr (responsive → doing) are in DIFFERENT tiers despite
// sharing the word "review". intake maps to default but is ADVISORY only — it runs
// foreground in the user's own session, which fab cannot re-model. Mirrored in
// docs/specs/stage-models.md § stage→tier table (drift-guarded).
var stageTiers = map[string]string{
	"intake":    TierDefault,
	"apply":     TierDoing,
	"review":    TierReview,
	"hydrate":   TierHydrate,
	"ship":      TierFast,
	"review-pr": TierDoing,
}

// DefaultTier returns the built-in default profile for a tier name and whether
// the tier is known. Exposed for the drift-guard test and the operator launcher.
func DefaultTier(tier string) (Profile, bool) {
	p, ok := defaultTiers[tier]
	return p, ok
}

// TierForStage returns the fixed tier a stage maps to and whether the stage is
// known. Exposed for the drift-guard test.
func TierForStage(stage string) (string, bool) {
	t, ok := stageTiers[stage]
	return t, ok
}

// IsTierName reports whether name is one of the known role-tier names. Used by
// `fab resolve-agent` to accept a tier name positionally alongside a stage name.
// The two sets overlap only at fixed points — a name shared by a stage and a tier
// (review, hydrate) is one the stage maps to that same-named tier
// (stageTiers[name] == name), so either interpretation resolves identically.
// ("ship" is a stage but NOT a tier — it maps to the fast tier.)
func IsTierName(name string) bool {
	_, ok := defaultTiers[name]
	return ok
}

// TierNames returns the known tier names, sorted (stable for the drift-guard
// test's set comparison).
func TierNames() []string {
	names := make([]string, 0, len(defaultTiers))
	for t := range defaultTiers {
		names = append(names, t)
	}
	sort.Strings(names)
	return names
}

// StageNames returns the known stage names, sorted (stable for the drift-guard
// test's set comparison).
func StageNames() []string {
	names := make([]string, 0, len(stageTiers))
	for s := range stageTiers {
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
// from becoming a Claude-only validator (provider neutrality): a tier overridden
// to another provider's model still gets its string through unchanged. Matched by
// family prefix so claude-haiku-4-5-20251001 → haiku.
func ModelAlias(model string) string {
	for _, m := range modelAliasPrefixes {
		if strings.HasPrefix(model, m.prefix) {
			return m.alias
		}
	}
	return model
}

// ResolveTier resolves a tier name → a concrete {provider, model, effort} profile
// via per-field inheritance:
//
//	built-in tier default  ←  project `default` tier  ←  project <tier> override
//
// (later wins per field). An unset field on the requested tier's override falls
// back to the project's `default` tier for that field, then to fab-kit's built-in
// default for the requested tier. This is why commands moved to providers:
// inheriting {provider, model, effort} is safe; the dangerous cross-semantics
// command inheritance can no longer happen.
//
// CROSS-PROVIDER CUTOFF (260805-j3cm). Field inheritance is safe WITHIN a
// provider but not ACROSS one: a tier that sets `provider: codex` and omits
// `model` used to inherit a CLAUDE model through the same per-field merge — the
// footgun fab documented rather than fixed, because there was no correct value to
// fill with. There is one now (the provider's default fill), so when the config
// explicitly names a provider DIFFERENT from the built-in tier profile's, every
// model/effort OWNED BY ANOTHER PROVIDER fills from:
//
//	that provider's default fill  →  empty
//
// never from the built-in tier's foreign values.
//
// A value's OWNER is the provider in effect at the layer that supplied it (see
// cutForeignFields): the layer's own `provider:` if it names one, else whatever
// the layers below it resolved. So a model written on the project `default` tier
// with no provider beside it is owned by the built-in's claude and does NOT
// survive a codex switch made on the requested tier, while a model written at (or
// above) that switch is owned by codex and does. Anchoring per field is what keeps
// the rule correct across the three layers — a single flattened "the config set
// this" bit cannot tell the two apart.
//
// SCOPE OF OWNERSHIP — a DOCUMENTED LIMITATION (260805-j3cm). The three layers
// above are the only ones ownership can see: `cfg` is the MERGED config, and
// config.LoadPath has already deep-merged the system layer (~/.fab-kit/config.yaml)
// and the project layer per-key BEFORE resolution runs. So per-SCOPE ownership is
// NOT tracked. When both scopes contribute to the SAME tier and name DIFFERENT
// providers, the merged tier reads as one layer and its values are attributed to
// the merged layer's `provider:` — the cutoff does not fire across that scope
// boundary. Concretely: a system-scope `agent.tiers.doing: {provider: codex, model:
// gpt-5.3-codex}` plus a project-scope `agent.tiers.doing: {provider: gemini}`
// resolves model=gpt-5.3-codex under provider=gemini (a codex model ID handed to
// the gemini CLI). This is pinned, not endorsed, by
// TestResolveCrossScopeCascadeLimitation (cmd/fab, which can compose both scopes)
// and stated in docs/specs/stage-models.md. Cascade-aware ownership (folding the
// per-scope layers here rather than consuming a pre-merged tree) is deferred to a
// follow-up change. Pin `model:`/`effort:` in the SAME scope as the `provider:`
// switch to stay unaffected.
//
// A tier that does not set `provider:` inherits exactly as before, and the
// all-claude default world is byte-unchanged (every built-in tier pins an explicit
// model, and an explicit `provider: claude` equals the built-in's provider, so it
// is not a switch at all — plan Assumption 2).
//
// NO validation: the resolved fields are returned verbatim. An unknown tier is
// the only tier-resolution error.
func ResolveTier(cfg *config.Config, tier string) (Profile, error) {
	builtin, ok := defaultTiers[tier]
	if !ok {
		return Profile{}, fmt.Errorf("unknown tier %q (valid: %s)", tier, strings.Join(TierNames(), ", "))
	}

	// Fold the config layers over the built-in profile in precedence order (the
	// project's `default` tier is the middle layer — below the requested tier's
	// own override, above the built-in), recording each value's OWNING provider
	// as it is set. Values that come from the built-in are owned by the built-in's
	// provider.
	resolved := builtin
	modelOwner, effortOwner := builtin.Provider, builtin.Provider
	var configuredProvider string

	applyLayer := func(layer config.TierProfile) {
		// Provider first, so a model/effort written in the SAME layer as the
		// switch is owned by the provider that layer names.
		if layer.Provider != "" {
			resolved.Provider = layer.Provider
			configuredProvider = layer.Provider
		}
		if layer.Model != "" {
			resolved.Model = layer.Model
			modelOwner = resolved.Provider
		}
		if layer.Effort != "" {
			resolved.Effort = layer.Effort
			effortOwner = resolved.Provider
		}
	}
	if def, ok := cfg.GetAgentTier(TierDefault); ok {
		applyLayer(def)
	}
	if override, ok := cfg.GetAgentTier(tier); ok {
		applyLayer(override)
	}

	// Cross-provider cutoff: the config explicitly named a provider other than
	// the built-in tier profile's, so every value still owned by a DIFFERENT
	// provider is foreign — refill it from the named provider (then empty). The
	// gate is the net configured provider vs the built-in's (Assumption 2): a
	// chain that ends back on the built-in's provider is not a switch at all.
	if configuredProvider != "" && configuredProvider != builtin.Provider {
		cutForeignFields(cfg, &resolved, modelOwner, effortOwner)
	}

	return resolved, nil
}

// cutForeignFields is the single implementation of the cross-provider cutoff,
// shared by ResolveTier (config layers) and ApplyOverrides (invocation flags). For
// each of model/effort whose OWNING provider differs from the one p now names, it
// replaces the value with the fill precedence's lower two rungs:
//
//	p.Provider's default fill (providers.<p.Provider>.model/.effort)  →  empty
//
// A field owned by p.Provider is kept verbatim — including an explicitly-empty one
// (ownership, not emptiness, is the test). Callers that swap a provider therefore
// need only record where each value came from; the rule itself lives here once.
//
// An unknown provider name is not an error here (resolution applies no
// validation): ResolveProvider reports ok=false with a zero config, so the fill is
// empty — the correct terminal state either way. Naming a provider is a lookup the
// CALLER may choose to error on (`fab agent` / `fab resolve-agent --provider` do);
// resolution does not.
func cutForeignFields(cfg *config.Config, p *Profile, modelOwner, effortOwner string) {
	prov, _ := ResolveProvider(cfg, p.Provider)
	if modelOwner != p.Provider {
		p.Model = prov.Model
	}
	if effortOwner != p.Provider {
		p.Effort = prov.Effort
	}
}

// Overrides is an invocation-time {provider, model, effort} override set — the
// top rung of the fill precedence, supplied by flags rather than config
// (`fab resolve-agent <stage> [--provider] [--model] [--effort]`). Each field is
// applied only when its Set companion is true, so an explicitly-empty flag value
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

// ApplyOverrides applies an invocation-time override set to an already-resolved
// profile, completing the fill precedence:
//
//	invocation flag  >  explicit tier field  >  provider default fill  >  empty
//
// The first two rungs are already baked into `p` by ResolveTier; this adds the
// flag rung and re-runs the lower rungs when the provider is SWAPPED. A provider
// swap does NOT retain the tier's model/effort — those belong to the old provider,
// so they are foreign and refill from the NEW provider's default fill, then empty.
// This is the SAME cutoff ResolveTier applies, through the same cutForeignFields
// helper: the flag set is just one more layer, whose incoming values are owned by
// p.Provider and whose own values are owned by the (possibly swapped) provider.
// Swapping to the provider the profile already names is not a swap and refills
// nothing (every value stays owned by it).
//
// NO validation: overridden strings pass through verbatim. An unknown provider
// name is a LOOKUP concern for the caller (which lists ProviderNames and exits
// non-zero), not a resolution error here.
func ApplyOverrides(cfg *config.Config, p Profile, o Overrides) Profile {
	modelOwner, effortOwner := p.Provider, p.Provider
	if o.ProviderSet {
		p.Provider = o.Provider
	}
	if o.ModelSet {
		p.Model = o.Model
		modelOwner = p.Provider
	}
	if o.EffortSet {
		p.Effort = o.Effort
		effortOwner = p.Provider
	}
	if o.ProviderSet {
		cutForeignFields(cfg, &p, modelOwner, effortOwner)
	}
	return p
}

// mergeTierField overwrites *dst with v only when v is non-empty (per-field merge:
// a set override field wins; an empty field inherits).
func mergeTierField(dst *string, v string) {
	if v != "" {
		*dst = v
	}
}

// Resolve maps a stage → its fixed tier → a concrete {provider, model, effort}
// profile (via ResolveTier). An unknown stage is the only resolution-side error.
func Resolve(cfg *config.Config, stage string) (Profile, error) {
	tier, ok := stageTiers[stage]
	if !ok {
		return Profile{}, fmt.Errorf("unknown stage %q (valid: %s)", stage, strings.Join(StageNames(), ", "))
	}
	// stageTiers only ever names tiers present in defaultTiers (guarded by the
	// drift-guard test), so ResolveTier cannot miss on a known stage.
	return ResolveTier(cfg, tier)
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

// ResolveProvider returns the {session_command, dispatch_command, model, effort}
// for a provider name: the project's providers.<name> override PER-FIELD merged
// over fab-kit's built-in provider table (an override field that is set wins; an
// omitted field inherits the built-in). A provider present in neither the project
// config nor the built-in table resolves to a zero ProviderConfig with ok=false —
// the caller decides whether that is an error (a session with no session_command,
// or a dispatch with no dispatch_command, are the two failure surfaces).
//
// The model/effort fields are the provider's DEFAULT FILL (260805-j3cm) and merge
// identically to the commands. fab-kit's built-ins supply fill for NO provider
// (grammar only — model IDs rot at CLI cadence), so a non-empty resolved fill
// always came from user config.
//
// NO validation: all four strings are returned verbatim.
func ResolveProvider(cfg *config.Config, name string) (config.ProviderConfig, bool) {
	resolved, known := defaultProviders[name]

	if override, ok := cfg.GetProvider(name); ok {
		known = true
		mergeTierField(&resolved.SessionCommand, override.SessionCommand)
		mergeTierField(&resolved.DispatchCommand, override.DispatchCommand)
		mergeTierField(&resolved.Model, override.Model)
		mergeTierField(&resolved.Effort, override.Effort)
	}

	return resolved, known
}
