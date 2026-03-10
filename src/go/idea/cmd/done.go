package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/idea/internal/idea"
)

func doneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <query>",
		Short: "Mark an idea as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Done(path, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Done: %s\n", idea.FormatLine(i))
			return nil
		},
	}
}
