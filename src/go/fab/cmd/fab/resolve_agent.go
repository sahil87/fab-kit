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
// directly (the two name sets are disjoint) — the latter serves `fab agent` and
// the operator launcher's tier-level resolution. It resolves the tier to a
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
func resolveAgentCmd() *cobra.Command {
	var alias bool
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

			// The dispatch= command ALWAYS embeds the full resolved model ID (CLI
			// dispatch never aliases), so substitute placeholders from the full
			// model BEFORE --alias overwrites profile.Model with the short alias.
			var dispatchLine string
			if prov, ok := agent.ResolveProvider(cfg, profile.Provider); ok && prov.DispatchCommand != "" {
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
	return cmd
}

// resolveStageOrTier accepts either a pipeline stage name (mapped via the fixed
// stage→tier mapping) or a role-tier name (resolved directly). The two name sets
// are disjoint, so a tier name is dispatched to ResolveTier and everything else to
// Resolve (which surfaces the unknown-stage error for a genuinely unknown name).
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
