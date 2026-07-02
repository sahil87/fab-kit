package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
	"github.com/spf13/cobra"
)

// configCmd is the `fab config` command group. Today it holds a single
// subcommand, `reference`; the group naming deliberately leaves room for a
// future `fab config validate` (unknown-key/typo linting — a non-goal here).
// Running the group with no subcommand shows its help (cobra default).
func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect fab project configuration",
		Long: "Config-related queries. `fab config reference` prints the fully " +
			"commented reference config.yaml (all available options). The group " +
			"leaves room for future config subcommands (e.g. validate).",
	}
	cmd.AddCommand(configReferenceCmd())
	return cmd
}

// configReferenceCmd implements `fab config reference` — a pure query (no side
// effects, no file writes) in the same family as `fab resolve` / `fab
// resolve-agent`. It prints the fully-commented reference config.yaml to
// stdout and exits 0. The output is GENERATED from Go constants (see
// internal/configref) and byte-stable for a given binary version. No flags.
func configReferenceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reference",
		Short: "Print the fully-commented reference config.yaml (all available options)",
		Long: "Prints a fully-commented reference fab/project/config.yaml to " +
			"stdout, documenting every available option (both binary-consumed " +
			"and skill-consumed keys). Baseline keys appear live with example " +
			"values; optional override blocks (agent.tiers, stage_hooks, " +
			"branch_prefix) appear commented-out with fab-kit's built-in " +
			"defaults. The output is generated from the binary's own constants " +
			"(never hand-written) and is byte-stable for a given version. Pure " +
			"query — writes no file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprint(cmd.OutOrStdout(), configref.Render())
			return nil
		},
	}
}
