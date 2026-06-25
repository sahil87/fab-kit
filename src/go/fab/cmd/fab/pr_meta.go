package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/prmeta"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func prMetaCmd() *cobra.Command {
	var prType string
	var issues string

	cmd := &cobra.Command{
		Use:   "pr-meta <change>",
		Short: "Render the `## Meta` block of a fab-generated PR as final markdown",
		Long: "Mechanically renders the complete `## Meta` block (table, " +
			"Pipeline, optional Issues, optional Impact) for a change, " +
			"reading .status.yaml, plan.md, fab/project/config.yaml, the " +
			"impact math, and git/gh context itself. The skill passes only " +
			"the change reference, the resolved PR --type, and optional " +
			"--issues. Exits non-zero (emitting nothing) when there is no " +
			"fab context, so /git-pr omits the Meta block exactly as before.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			data, ok, err := prmeta.Gather(fabRoot, args[0], prType, issues)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("no fab context for %q (change unresolved or .status.yaml absent)", args[0])
			}

			// Stamp the running binary's version into the Meta block's
			// provenance caption. Sourced here (not in Gather) so Render stays a
			// pure function of Data — `version` is private to package main (pnao).
			data.Version = version

			fmt.Print(prmeta.Render(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&prType, "type", "", "Resolved PR type (feat|fix|refactor|docs|test|ci|chore) — required")
	cmd.Flags().StringVar(&issues, "issues", "", "Space-joined issue IDs (e.g. \"DEV-1 DEV-2\") — optional")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}
