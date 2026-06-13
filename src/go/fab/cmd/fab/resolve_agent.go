package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// resolveAgentCmd implements `fab resolve-agent <stage>` — a pure query (no side
// effects) in the same family as `fab resolve`. It maps the stage through the
// fixed stage→tier mapping, resolves the tier to a concrete {model, effort}
// (project agent.tiers override per-field-merged over the fab-kit default, else
// the default), and echoes the result VERBATIM — no validation against any
// provider's accepted set (provider neutrality).
//
// Output (byte-stable for the same config): two stdout lines
//
//	model=<id>
//	effort=<level>
//
// The effort line is omitted when the resolved tier has no effort. An empty
// model emits an empty `model=` line, signaling "inherit the session/orchestrator
// model". Non-zero exit only on a real error: malformed/unreadable config, or an
// unknown stage name. A stage resolving to a default is success.
func resolveAgentCmd() *cobra.Command {
	return &cobra.Command{
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

			fmt.Fprint(cmd.OutOrStdout(), formatAgentProfile(profile))
			return nil
		},
	}
}

// formatAgentProfile renders a resolved profile as the byte-stable stdout
// contract: a `model=<id>` line always, plus an `effort=<level>` line only when
// the effort is non-empty. An empty model emits an empty `model=` line (the
// "inherit" signal). Extracted so the omit-when-empty branches are unit-testable
// without needing a config whose RESOLVED effort is empty.
func formatAgentProfile(p agent.Profile) string {
	out := fmt.Sprintf("model=%s\n", p.Model)
	if p.Effort != "" {
		out += fmt.Sprintf("effort=%s\n", p.Effort)
	}
	return out
}
