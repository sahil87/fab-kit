package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
)

func resolveCmd() *cobra.Command {
	var outputMode string

	cmd := &cobra.Command{
		Use:   "resolve [change]",
		Short: "Resolve a change reference to a canonical output",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			changeArg := ""
			if len(args) > 0 {
				changeArg = args[0]
			}

			folder, err := resolve.ToFolder(fabRoot, changeArg)
			if err != nil {
				return err
			}

			switch outputMode {
			case "id":
				fmt.Println(resolve.ExtractID(folder))
			case "folder":
				fmt.Println(folder)
			case "dir":
				fmt.Printf("fab/changes/%s/\n", folder)
			case "status":
				fmt.Printf("fab/changes/%s/.status.yaml\n", folder)
			}
			return nil
		},
	}

	// Register the --id, --folder, --dir, --status flags matching the bash interface
	cmd.Flags().Bool("id", false, "Output 4-char change ID (default)")
	cmd.Flags().Bool("folder", false, "Output full folder name")
	cmd.Flags().Bool("dir", false, "Output directory path")
	cmd.Flags().Bool("status", false, "Output .status.yaml path")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if f, _ := cmd.Flags().GetBool("folder"); f {
			outputMode = "folder"
		} else if f, _ := cmd.Flags().GetBool("dir"); f {
			outputMode = "dir"
		} else if f, _ := cmd.Flags().GetBool("status"); f {
			outputMode = "status"
		} else {
			outputMode = "id"
		}
		return nil
	}

	return cmd
}
