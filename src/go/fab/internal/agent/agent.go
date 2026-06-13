// Package agent owns fab-kit's per-stage model selection: the default
// tier→{model, effort} table, the FIXED stage→tier mapping, and the resolution
// cascade consumed by `fab resolve-agent <stage>`.
//
// The two tables here are fab-kit's curated judgment. The stage→tier mapping is
// NOT user-overridable (there is no stage_tiers config and no per-stage escape
// hatch); the default tier→profile table is the single place to bump when a new
// top model lands (the "Fable upgrade path"). Users override only what each
// tier MEANS, via agent.tiers in config.yaml (per-field merge over the default).
//
// Resolution applies NO validation — it echoes the resolved {model, effort}
// verbatim, whatever they are (provider neutrality, Constitution Principle I).
// Compatibility is the runtime/harness's concern, not fab's.
//
// These tables are mirrored in docs/specs/stage-models.md and guarded against
// drift by TestDocTablesMatchAgentMaps (stagemodels_doc_test.go), the same
// pattern internal/score uses for change-types.md.
package agent

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Tier names. Three tiers grouped by cognitive mode.
const (
	TierThinking = "thinking" // generative judgment (intake discovers requirements; review discovers bugs)
	TierDoing    = "doing"    // execution that must not err (apply writes the diff; review-pr fixes feedback; hydrate writes memory)
	TierFast     = "fast"     // speed on near-mechanical work (commit/push/PR mechanics + PR-description summary)
)

// Profile is a concrete {model, effort} pair. An empty Model signals "inherit
// the session/orchestrator model"; an empty Effort omits effort entirely.
type Profile struct {
	Model  string
	Effort string
}

// defaultTiers is fab-kit's built-in tier→profile table (today). This is the
// ONE place bumped when a new top model lands. Mirrored in
// docs/specs/stage-models.md § default-tier table (drift-guarded).
var defaultTiers = map[string]Profile{
	TierThinking: {Model: "claude-opus-4-8", Effort: "xhigh"},
	TierDoing:    {Model: "claude-opus-4-8", Effort: "high"},
	TierFast:     {Model: "claude-sonnet-4-6", Effort: "low"},
}

// stageTiers is the FIXED, fab-owned stage→tier mapping. Exhaustive over the six
// pipeline stages (each stage belongs to exactly one tier). NOT user-overridable.
// Note review (generative → thinking) and review-pr (responsive → doing) are in
// DIFFERENT tiers despite sharing the word "review". Mirrored in
// docs/specs/stage-models.md § stage→tier table (drift-guarded).
var stageTiers = map[string]string{
	"intake":    TierThinking,
	"review":    TierThinking,
	"apply":     TierDoing,
	"review-pr": TierDoing,
	"hydrate":   TierDoing,
	"ship":      TierFast,
}

// DefaultTier returns the built-in default profile for a tier name and whether
// the tier is known. Exposed for the drift-guard test.
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

// Resolve maps a stage → its fixed tier → a concrete {model, effort} profile.
//
// The tier profile is the project's agent.tiers.<tier> override PER-FIELD merged
// over the fab-kit default: an override field that is set wins; an omitted
// override field inherits the default for that field. A tier with no override
// resolves to the default unchanged.
//
// NO validation: the resolved model and effort are returned verbatim, whatever
// they are. An unknown stage is the only resolution-side error.
func Resolve(cfg *config.Config, stage string) (Profile, error) {
	tier, ok := stageTiers[stage]
	if !ok {
		return Profile{}, fmt.Errorf("unknown stage %q (valid: %s)", stage, strings.Join(StageNames(), ", "))
	}

	// defaultTiers always has an entry for every tier in stageTiers (guarded by
	// the drift-guard test), so this lookup cannot miss.
	resolved := defaultTiers[tier]

	if override, ok := cfg.GetAgentTier(tier); ok {
		if override.Model != "" {
			resolved.Model = override.Model
		}
		if override.Effort != "" {
			resolved.Effort = override.Effort
		}
	}

	return resolved, nil
}
