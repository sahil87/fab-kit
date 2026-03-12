package main

import (
	"fmt"
	"os"

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
			case "pane":
				if os.Getenv("TMUX") == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Error: not inside a tmux session")
					os.Exit(1)
				}

				panes, err := discoverPanes()
				if err != nil {
					return err
				}

				matches, warning := matchPanesByFolder(panes, folder, resolvePaneChange)

				if len(matches) == 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "no tmux pane found for change %q\n", folder)
					os.Exit(1)
				}

				if warning != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), warning)
				}

				fmt.Println(matches[0])
			}
			return nil
		},
	}

	// Register the --id, --folder, --dir, --status, --pane flags matching the bash interface
	cmd.Flags().Bool("id", false, "Output 4-char change ID (default)")
	cmd.Flags().Bool("folder", false, "Output full folder name")
	cmd.Flags().Bool("dir", false, "Output directory path")
	cmd.Flags().Bool("status", false, "Output .status.yaml path")
	cmd.Flags().Bool("pane", false, "Output tmux pane ID")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if f, _ := cmd.Flags().GetBool("folder"); f {
			outputMode = "folder"
		} else if f, _ := cmd.Flags().GetBool("dir"); f {
			outputMode = "dir"
		} else if f, _ := cmd.Flags().GetBool("status"); f {
			outputMode = "status"
		} else if f, _ := cmd.Flags().GetBool("pane"); f {
			outputMode = "pane"
		} else {
			outputMode = "id"
		}
		return nil
	}

	return cmd
}
