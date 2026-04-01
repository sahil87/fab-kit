package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/change"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

func changeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change",
		Short: "Change lifecycle management",
	}

	cmd.AddCommand(
		changeNewCmd(),
		changeRenameCmd(),
		changeSwitchCmd(),
		changeListCmd(),
		changeResolveCmd(),
		changeArchiveCmd(),
		changeRestoreCmd(),
		changeArchiveListCmd(),
	)

	return cmd
}

func changeNewCmd() *cobra.Command {
	var slug, changeID, logArgs string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new change directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			folder, err := change.New(fabRoot, slug, changeID, logArgs)
			if err != nil {
				return err
			}
			fmt.Println(folder)
			return nil
		},
	}

	cmd.Flags().StringVar(&slug, "slug", "", "Folder name suffix (required)")
	cmd.Flags().StringVar(&changeID, "change-id", "", "Explicit 4-char ID (optional)")
	cmd.Flags().StringVar(&logArgs, "log-args", "", "Description for logman (optional)")

	return cmd
}

func changeRenameCmd() *cobra.Command {
	var folder, slug string

	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename a change folder's slug",
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			newName, err := change.Rename(fabRoot, folder, slug)
			if err != nil {
				return err
			}
			fmt.Println(newName)
			return nil
		},
	}

	cmd.Flags().StringVar(&folder, "folder", "", "Current folder name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "New slug (required)")

	return cmd
}

func changeSwitchCmd() *cobra.Command {
	var none bool

	cmd := &cobra.Command{
		Use:   "switch [name]",
		Short: "Switch the active change",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			if none {
				fmt.Println(change.SwitchNone(fabRoot))
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("switch requires <name> or --none")
			}

			output, err := change.Switch(fabRoot, args[0])
			if err != nil {
				return err
			}
			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().BoolVar(&none, "none", false, "Deactivate the current change")
	return cmd
}

func changeListCmd() *cobra.Command {
	var archive bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List changes with stage info",
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			results, err := change.List(fabRoot, archive)
			if err != nil {
				return err
			}
			for _, r := range results {
				fmt.Println(r)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&archive, "archive", false, "List archived changes")

	return cmd
}

func changeResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve [override]",
		Short: "Resolve a change name",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			override := ""
			if len(args) > 0 {
				override = args[0]
			}
			folder, err := change.Resolve(fabRoot, override)
			if err != nil {
				return err
			}
			fmt.Println(folder)
			return nil
		},
	}
}
