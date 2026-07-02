package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// resolveAgentCmd implements `fab resolve-agent <stage>` — a pure query (no side
// effects) in the same family as `fab resolve`. It maps the stage through the
// fixed stage→tier mapping, resolves the tier to a concrete {model, effort}
// (project agent.tiers override per-field-merged over the fab-kit default, else
// the default), and echoes the result VERBATIM — no validation against any
// provider's accepted set (provider neutrality).
//
// Output (byte-stable for the same config): two stdout lines, plus an optional
// third `spawn=` line
//
//	model=<id>
//	effort=<level>
//	spawn=<command>
//
// The effort line is omitted when the resolved tier has no effort. An empty
// model emits an empty `model=` line, signaling "inherit the session/orchestrator
// model". The spawn line is emitted ONLY when the resolved tier carries a
// spawn_command (the per-stage CLI-dispatch opt-in) — its absence signals native
// Agent-tool dispatch, and there is NO fallback to agent.spawn_command. Non-zero
// exit only on a real error: malformed/unreadable config, or an unknown stage
// name. A stage resolving to a default is success.
//
// The optional `--alias` flag is the Claude-Code Agent-tool adapter: when set,
// the resolved model is mapped to its short alias (opus/sonnet/haiku/fable) on the
// `model=` line via agent.ModelAlias, since the Agent tool's `model` enum rejects
// full IDs. Default (absent) is byte-identical to today (full ID). The `effort=`
// line is unaffected by `--alias`; empty/non-Claude models pass through verbatim.
// The `spawn=` line ALWAYS embeds the FULL model ID even under `--alias` — CLI
// dispatch never aliases (an external CLI's --model flag takes a full ID); the
// {model}/{effort} placeholders are substituted via internal/spawn.WithProfile
// (reused, not reimplemented) using the tier's own resolved model/effort.
func resolveAgentCmd() *cobra.Command {
	var alias bool
	cmd := &cobra.Command{
		Use:   "resolve-agent <stage>",
		Short: "Resolve a pipeline stage to its {model, effort} agent profile",
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

			profile, err := agent.Resolve(cfg, args[0])
			if err != nil {
				return err
			}

			// The spawn= command ALWAYS embeds the full resolved model ID (CLI
			// dispatch never aliases), so substitute placeholders from the full
			// model BEFORE --alias overwrites profile.Model with the short alias.
			var spawnLine string
			if profile.SpawnCommand != "" {
				spawnLine = spawn.WithProfile(profile.SpawnCommand, profile.Model, profile.Effort)
			}

			if alias {
				profile.Model = agent.ModelAlias(profile.Model)
			}

			fmt.Fprint(cmd.OutOrStdout(), formatAgentProfile(profile, spawnLine))
			return nil
		},
	}
	cmd.Flags().BoolVar(&alias, "alias", false, "emit the Claude-Code short model alias (opus/sonnet/haiku/fable) on the model= line instead of the full ID (Agent-tool adapter)")
	return cmd
}

// formatAgentProfile renders a resolved profile as the byte-stable stdout
// contract: a `model=<id>` line always, an `effort=<level>` line only when the
// effort is non-empty, and a `spawn=<command>` line only when spawnLine is
// non-empty. An empty model emits an empty `model=` line (the "inherit" signal).
// spawnLine is the ALREADY-substituted command (placeholders resolved via
// internal/spawn) — the caller passes "" to omit the line (native Agent-tool
// dispatch). Extracted so the omit-when-empty branches are unit-testable without
// needing a config whose RESOLVED effort/spawn_command is empty.
func formatAgentProfile(p agent.Profile, spawnLine string) string {
	out := fmt.Sprintf("model=%s\n", p.Model)
	if p.Effort != "" {
		out += fmt.Sprintf("effort=%s\n", p.Effort)
	}
	if spawnLine != "" {
		out += fmt.Sprintf("spawn=%s\n", spawnLine)
	}
	return out
}
