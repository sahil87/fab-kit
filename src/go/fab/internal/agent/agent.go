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

// Role-tier names. Five roles with concrete referents, replacing the old
// thinking/doing/fast cognitive-mode vocabulary.
const (
	TierDefault  = "default"  // spawned worker sessions, `fab agent` with no tier, intake (advisory); per-field fallback for every other tier
	TierOperator = "operator" // the operator coordinator session (`fab operator`)
	TierDoing    = "doing"    // apply, review-pr, hydrate — execution that must not err
	TierReview   = "review"   // review — author/critic separation
	TierFast     = "fast"     // ship — speed on near-mechanical work
)

// DefaultProviderName is the built-in provider a fresh config resolves to when a
// tier declares no provider and the project sets no `default` tier provider.
const DefaultProviderName = "claude"

// DefaultSessionCommand is the built-in claude provider's session command — the
// relocated agent.spawn_command default. Kept here (not internal/spawn) because
// the provider table is agent-owned; internal/spawn re-exports the string for its
// own no-config fallback.
const DefaultSessionCommand = `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"`

// Profile is a concrete {provider, model, effort} triple. An empty Provider names
// no provider (resolution falls through to the built-in default provider at
// command-composition time); an empty Model signals "inherit the
// session/orchestrator model"; an empty Effort omits effort entirely.
type Profile struct {
	Provider string
	Model    string
	Effort   string
}

// defaultProviders is fab-kit's built-in provider table: the claude provider,
// explicit and shipped as the default, with the default session command and NO
// dispatch_command (native Agent-tool dispatch). A project extends/overrides via
// its own providers: block, per-field merged over this.
var defaultProviders = map[string]config.ProviderConfig{
	DefaultProviderName: {SessionCommand: DefaultSessionCommand},
}

// defaultTiers is fab-kit's built-in tier→profile table (today). This is the ONE
// place bumped when a new top model lands. Provider is written explicitly on
// every line (documented style; inheritance is the safety net). Mirrored in
// docs/specs/stage-models.md § default-tier table (drift-guarded).
var defaultTiers = map[string]Profile{
	TierDefault:  {Provider: "claude", Model: "claude-fable-5", Effort: "xhigh"},
	TierOperator: {Provider: "claude", Model: "claude-sonnet-5", Effort: "medium"},
	TierDoing:    {Provider: "claude", Model: "claude-opus-4-8", Effort: "xhigh"},
	TierReview:   {Provider: "claude", Model: "claude-fable-5", Effort: "xhigh"},
	TierFast:     {Provider: "claude", Model: "claude-sonnet-5", Effort: "low"},
}

// stageTiers is the FIXED, fab-owned stage→tier mapping. Exhaustive over the six
// pipeline stages (each stage belongs to exactly one tier). NOT user-overridable.
// Note review (own tier — author/critic separation) and review-pr (responsive →
// doing) are in DIFFERENT tiers despite sharing the word "review". intake maps to
// default but is ADVISORY only — it runs foreground in the user's own session,
// which fab cannot re-model. Mirrored in docs/specs/stage-models.md
// § stage→tier table (drift-guarded).
var stageTiers = map[string]string{
	"intake":    TierDefault,
	"apply":     TierDoing,
	"review":    TierReview,
	"hydrate":   TierDoing,
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
// `fab resolve-agent` to accept a tier name positionally alongside a stage name
// (the two sets are disjoint).
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
// NO validation: the resolved fields are returned verbatim. An unknown tier is
// the only tier-resolution error.
func ResolveTier(cfg *config.Config, tier string) (Profile, error) {
	resolved, ok := defaultTiers[tier]
	if !ok {
		return Profile{}, fmt.Errorf("unknown tier %q (valid: %s)", tier, strings.Join(TierNames(), ", "))
	}

	// The project's `default` tier fills any field the built-in leaves unset AND
	// any field the requested tier's own override leaves unset. Apply it as the
	// middle layer (below the requested tier's override, above the built-in).
	if def, ok := cfg.GetAgentTier(TierDefault); ok {
		mergeTierField(&resolved.Provider, def.Provider)
		mergeTierField(&resolved.Model, def.Model)
		mergeTierField(&resolved.Effort, def.Effort)
	}

	if override, ok := cfg.GetAgentTier(tier); ok {
		mergeTierField(&resolved.Provider, override.Provider)
		mergeTierField(&resolved.Model, override.Model)
		mergeTierField(&resolved.Effort, override.Effort)
	}

	return resolved, nil
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

// ResolveProvider returns the {session_command, dispatch_command} for a provider
// name: the project's providers.<name> override PER-FIELD merged over fab-kit's
// built-in provider table (an override field that is set wins; an omitted field
// inherits the built-in). A provider present in neither the project config nor the
// built-in table resolves to a zero ProviderConfig with ok=false — the caller
// decides whether that is an error (a session with no session_command, or a
// dispatch with no dispatch_command, are the two failure surfaces).
//
// NO validation: command strings are returned verbatim.
func ResolveProvider(cfg *config.Config, name string) (config.ProviderConfig, bool) {
	resolved, known := defaultProviders[name]

	if override, ok := cfg.GetProvider(name); ok {
		known = true
		mergeTierField(&resolved.SessionCommand, override.SessionCommand)
		mergeTierField(&resolved.DispatchCommand, override.DispatchCommand)
	}

	return resolved, known
}
