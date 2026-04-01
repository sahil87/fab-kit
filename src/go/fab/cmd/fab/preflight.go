package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/preflight"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

func preflightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight [change-name]",
		Short: "Validate project state and output structured YAML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			changeOverride := ""
			if len(args) > 0 {
				changeOverride = args[0]
			}

			result, err := preflight.Run(fabRoot, changeOverride)
			if err != nil {
				return err
			}

			fmt.Print(preflight.FormatYAML(result))
			return nil
		},
	}
}
