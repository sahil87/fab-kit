package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/idea-go/internal/idea"
)

func rmCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <query>",
		Short: "Delete an idea from the backlog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Rm(path, args[0], force)
			if err != nil {
				return err
			}

			fmt.Printf("Removed: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion")

	return cmd
}
