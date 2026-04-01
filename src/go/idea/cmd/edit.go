package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/idea/internal/idea"
)

func editCmd() *cobra.Command {
	var newID, newDate string

	cmd := &cobra.Command{
		Use:   "edit <query> <new-text>",
		Short: "Modify an idea's text",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Edit(path, args[0], args[1], newID, newDate)
			if err != nil {
				return err
			}

			fmt.Printf("Updated: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().StringVar(&newID, "id", "", "Change the idea's ID")
	cmd.Flags().StringVar(&newDate, "date", "", "Change the idea's date")

	return cmd
}
