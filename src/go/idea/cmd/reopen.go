package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/idea/internal/idea"
)

func reopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <query>",
		Short: "Reopen a completed idea",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Reopen(path, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Reopened: %s\n", idea.FormatLine(i))
			return nil
		},
	}
}
