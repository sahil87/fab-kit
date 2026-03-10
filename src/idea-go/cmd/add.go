package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/idea-go/internal/idea"
)

func addCmd() *cobra.Command {
	var customID, customDate string

	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a new idea to the backlog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}
			i, err := idea.Add(path, args[0], customID, customDate)
			if err != nil {
				return err
			}
			fmt.Printf("Added: [%s] %s: %s\n", i.ID, i.Date, i.Text)
			return nil
		},
	}

	cmd.Flags().StringVar(&customID, "id", "", "Custom 4-char ID")
	cmd.Flags().StringVar(&customDate, "date", "", "Custom date (YYYY-MM-DD)")

	return cmd
}
