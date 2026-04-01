package main

import (
	"fmt"

	"github.com/spf13/cobra"
	archivePkg "github.com/sahil87/fab-kit/src/go/fab/internal/archive"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

func changeArchiveCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "archive <change>",
		Short: "Archive a change",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			result, err := archivePkg.Archive(fabRoot, args[0], description)
			if err != nil {
				return err
			}
			fmt.Println(archivePkg.FormatArchiveYAML(result))
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Description for archive index (required)")

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
