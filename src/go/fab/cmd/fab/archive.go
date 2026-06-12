package main

import (
	"errors"
	"fmt"

	archivePkg "github.com/sahil87/fab-kit/src/go/fab/internal/archive"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func changeArchiveCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "archive <change>",
		Short: "Archive a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			result, err := archivePkg.ArchiveWithBacklog(fabRoot, args[0], description)
			if err != nil {
				if errors.Is(err, archivePkg.ErrAlreadyArchived) {
					fmt.Printf("already archived: %s\n", args[0])
					return nil
				}
				// A non-nil result means the archive move succeeded but the
				// backlog mark failed. Emit the YAML report so the success is
				// visible, then surface the backlog failure as the command error.
				if result != nil {
					fmt.Println(archivePkg.FormatArchiveYAML(result))
				}
				return err
			}
			fmt.Println(archivePkg.FormatArchiveYAML(result))
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Description for archive index (optional; defaults to intake title)")

	return cmd
}

func changeRestoreCmd() *cobra.Command {
	var doSwitch bool

	cmd := &cobra.Command{
		Use:   "restore <change>",
		Short: "Restore an archived change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			result, err := archivePkg.Restore(fabRoot, args[0], doSwitch)
			if err != nil {
				return err
			}
			fmt.Println(archivePkg.FormatRestoreYAML(result))
			return nil
		},
	}

	cmd.Flags().BoolVar(&doSwitch, "switch", false, "Activate the restored change")

	return cmd
}

func changeArchiveListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive-list",
		Short: "List archived changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			results, err := archivePkg.List(fabRoot)
			if err != nil {
				return err
			}
			for _, r := range results {
				fmt.Println(r)
			}
			return nil
		},
	}
}
