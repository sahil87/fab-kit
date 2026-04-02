package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
	"github.com/spf13/cobra"
)

func kitPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kit-path",
		Short: "Print the resolved kit directory path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := kitpath.KitDir()
			if err != nil {
				return fmt.Errorf("cannot resolve kit path: %w\nRun 'fab sync' or 'fab upgrade' to populate the cache", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), dir)
			return nil
		},
	}
}
