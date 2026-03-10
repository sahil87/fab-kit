package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/idea-go/internal/idea"
)

func showCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <query>",
		Short: "Show a single idea",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Show(path, args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				return enc.Encode(i)
			}

			fmt.Println(idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}
