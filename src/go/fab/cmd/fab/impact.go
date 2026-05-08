package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func impactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "impact <base> <head>",
		Short: "Compute git diff line counts (added/deleted/net) between two refs",
		Long: "Computes the canonical true-impact shortstat math: " +
			"git diff --shortstat <base>...<head>, plus an optional " +
			"`excluding` pass when fab/project/config.yaml's " +
			"true_impact_exclude is non-empty. Outputs YAML to stdout.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, head := args[0], args[1]

			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			res, err := impact.ComputeForRepo(fabRoot, base, head)
			if err != nil {
				return err
			}

			fmt.Print(renderYAML(res))
			return nil
		},
	}
}

func renderYAML(r impact.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "added: %d\n", r.Added)
	fmt.Fprintf(&b, "deleted: %d\n", r.Deleted)
	fmt.Fprintf(&b, "net: %d\n", r.Net)
	if r.Excluding != nil {
		fmt.Fprintln(&b, "excluding:")
		fmt.Fprintf(&b, "    added: %d\n", r.Excluding.Added)
		fmt.Fprintf(&b, "    deleted: %d\n", r.Excluding.Deleted)
		fmt.Fprintf(&b, "    net: %d\n", r.Excluding.Net)
	}
	fmt.Fprintf(&b, "computed_at: %s\n", time.Now().UTC().Format(time.RFC3339))
	return b.String()
}
