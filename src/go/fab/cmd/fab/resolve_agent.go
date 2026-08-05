package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// resolveAgentCmd implements `fab resolve-agent <stage|tier>` — a pure query (no
// side effects) in the same family as `fab resolve`. Its argument is either a
// pipeline STAGE (mapped through the fixed stage→tier mapping) or a role-TIER name
// directly — the latter serves `fab agent` and the operator launcher's tier-level
// resolution. A name shared by a stage and a tier (review, hydrate) is a fixed
// point (that stage maps to that same-named tier), so tier-first dispatch resolves
// it identically either way. It resolves the tier to a
// concrete {provider, model, effort} (project agent.tiers override per-field
// merged over the fab-kit default, with default-tier inheritance), and echoes the
// result VERBATIM — no validation against any provider's accepted set (provider
// neutrality).
//
// Output (byte-stable for the same config): a `model=` line always, then optional
// `effort=`, `provider=`, and `dispatch=` lines:
//
//	model=<id>
//	effort=<level>
//	provider=<name>
//	dispatch=<command>
//
// The effort line is omitted when the resolved tier has no effort; the provider
// line is omitted when the resolved tier has no provider. An empty model emits an
// empty `model=` line, signaling "inherit the session/orchestrator model". The
// dispatch line is emitted ONLY when the resolved tier's provider carries a
// dispatch_command (the CLI-dispatch opt-in) — its absence signals native
// Agent-tool dispatch, and there is NO fallback to a session command. Non-zero
// exit only on a real error: malformed/unreadable config, or an unknown
// stage/tier name.
//
// The optional `--alias` flag is the Claude-Code Agent-tool adapter: when set,
// the resolved model is mapped to its short alias (opus/sonnet/haiku/fable) on the
// `model=` line via agent.ModelAlias, since the Agent tool's `model` enum rejects
// full IDs. Default (absent) is the full ID. The `effort=`/`provider=` lines are
// unaffected by `--alias`; empty/non-Claude models pass through verbatim. The
// `dispatch=` line ALWAYS embeds the FULL model ID even under `--alias` — CLI
// dispatch never aliases (an external CLI's --model flag takes a full ID); the
// {model}/{effort} placeholders are substituted via internal/spawn.WithProfile
// (reused, not reimplemented) using the tier's own resolved model/effort.
//
// INVOCATION-TIME OVERRIDES (260805-j3cm): `--provider <name>`, `--model <id>`,
// and `--effort <level>` are the top rung of the fill precedence
//
//	invocation flag  >  explicit tier field  >  provider default fill  >  empty
//
// applied to the resolved profile by agent.ApplyOverrides (the precedence lives in
// internal/agent, not here). `--provider` swaps the provider and re-derives the
// `dispatch=` line from the NAMED provider's dispatch_command — so the emitted
// `dispatch=` presence can differ from the stage's unoverridden one. That is a
// QUERY RESULT, not an adapter move: this command only reports what the named
// provider's dispatch_command is. `fab dispatch start` takes no override flags and
// re-resolves the stage from config itself, so relocating a stage between native
// Agent-tool dispatch and CLI dispatch takes a config/tier override, never an
// invocation flag. A swap does not retain the tier's model/effort (they belong to
// the old provider); an unoverridden field refills from the new provider's default
// fill, then empty.
//
// `--model`/`--effort` are valid WITHOUT `--provider` here — a within-tier
// override of the profile this pure query would otherwise print. This is a
// deliberate, documented asymmetry with `fab agent`, where they remain usage
// errors without `--provider`: `fab agent` is a session launcher with two mutually
// exclusive addressing modes, where a bare `--model` would invent an undocumented
// tier-override surface.
//
// An unknown `--provider` name is a LOOKUP failure (non-zero exit naming the
// resolvable providers) — not validation of any command's content
// (document-don't-validate stands: resolved strings still pass through verbatim).
// The error is byte-identical to `fab agent`'s because both call the shared
// unknownProviderError helper (agent.go).
func resolveAgentCmd() *cobra.Command {
	var alias bool
	var provider string
	var model string
	var effort string
	cmd := &cobra.Command{
		Use:   "resolve-agent <stage|tier>",
		Short: "Resolve a pipeline stage (or role tier) to its {provider, model, effort} agent profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			cfg, err := config.Load(fabRoot)
			if err != nil {
				return err
			}

			profile, err := resolveStageOrTier(cfg, args[0])
			if err != nil {
				return err
			}

			// Overrides key on whether the flag was SUPPLIED (cobra's
			// Flag.Changed), not on value emptiness — so `--model=` explicitly
			// clears the tier's model (emitting the inherit signal) instead of
			// being silently ignored.
			providerSet := cmd.Flags().Changed("provider")
			profile = agent.ApplyOverrides(cfg, profile, agent.Overrides{
				Provider:    provider,
				ProviderSet: providerSet,
				Model:       model,
				ModelSet:    cmd.Flags().Changed("model"),
				Effort:      effort,
				EffortSet:   cmd.Flags().Changed("effort"),
			})

			prov, known := agent.ResolveProvider(cfg, profile.Provider)
			if providerSet && !known {
				// A supplied --provider that resolves to nothing is a lookup
				// failure naming the resolvable set — shared verbatim with
				// `fab agent` via unknownProviderError.
				return unknownProviderError(cfg, profile.Provider)
			}
			// The dispatch= command ALWAYS embeds the full resolved model ID (CLI
			// dispatch never aliases), so substitute placeholders from the full
			// model BEFORE --alias overwrites profile.Model with the short alias.
			var dispatchLine string
			if known && prov.DispatchCommand != "" {
				dispatchLine = spawn.WithProfile(prov.DispatchCommand, profile.Model, profile.Effort)
			}

			if alias {
				profile.Model = agent.ModelAlias(profile.Model)
			}

			fmt.Fprint(cmd.OutOrStdout(), formatAgentProfile(profile, dispatchLine))
			return nil
		},
	}
	cmd.Flags().BoolVar(&alias, "alias", false, "emit the Claude-Code short model alias (opus/sonnet/haiku/fable) on the model= line instead of the full ID (Agent-tool adapter)")
	cmd.Flags().StringVar(&provider, "provider", "", "override the resolved provider (re-derives dispatch= from that provider; unoverridden model/effort refill from its default fill, then empty)")
	cmd.Flags().StringVar(&model, "model", "", "override the resolved model (valid without --provider — a within-tier override)")
	cmd.Flags().StringVar(&effort, "effort", "", "override the resolved effort (valid without --provider — a within-tier override)")
	return cmd
}

// resolveStageOrTier accepts either a pipeline stage name (mapped via the fixed
// stage→tier mapping) or a role-tier name (resolved directly). Tier names are
// checked FIRST: a tier name is dispatched to ResolveTier and everything else to
// Resolve (which surfaces the unknown-stage error for a genuinely unknown name).
// A name shared by a stage and a tier (review, hydrate) is a fixed point
// (stageTiers[name] == name), so this tier-first order resolves such a name to the
// same profile either interpretation would — the order is immaterial for results.
func resolveStageOrTier(cfg *config.Config, name string) (agent.Profile, error) {
	if agent.IsTierName(name) {
		return agent.ResolveTier(cfg, name)
	}
	return agent.Resolve(cfg, name)
}

// formatAgentProfile renders a resolved profile as the byte-stable stdout
// contract: a `model=<id>` line always, an `effort=<level>` line only when the
// effort is non-empty, a `provider=<name>` line only when the provider is
// non-empty, and a `dispatch=<command>` line only when dispatchLine is non-empty.
// An empty model emits an empty `model=` line (the "inherit" signal). dispatchLine
// is the ALREADY-substituted command (placeholders resolved via internal/spawn) —
// the caller passes "" to omit the line (native Agent-tool dispatch). Extracted so
// the omit-when-empty branches are unit-testable without needing a config whose
// RESOLVED effort/provider/dispatch_command is empty.
func formatAgentProfile(p agent.Profile, dispatchLine string) string {
	out := fmt.Sprintf("model=%s\n", p.Model)
	if p.Effort != "" {
		out += fmt.Sprintf("effort=%s\n", p.Effort)
	}
	if p.Provider != "" {
		out += fmt.Sprintf("provider=%s\n", p.Provider)
	}
	if dispatchLine != "" {
		out += fmt.Sprintf("dispatch=%s\n", dispatchLine)
	}
	return out
}
