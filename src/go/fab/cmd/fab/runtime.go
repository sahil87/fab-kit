package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/runtime"
)

func runtimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Manage runtime state (.fab-runtime.yaml)",
	}

	cmd.AddCommand(
		runtimeSetIdleCmd(),
		runtimeClearIdleCmd(),
		runtimeIsIdleCmd(),
	)

	return cmd
}

func runtimeSetIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-idle <change>",
		Short: "Record agent idle timestamp for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				return err
			}

			return runtime.SetIdle(fabRoot, folder)
		},
	}
}

func runtimeClearIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-idle <change>",
		Short: "Clear agent idle state for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				return err
			}

			return runtime.ClearIdle(fabRoot, folder)
		},
	}
}

func runtimeIsIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-idle <change>",
		Short: "Check if an agent is idle for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				return nil
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				return nil
			}

			rtPath := runtime.FilePath(fabRoot)
			m, err := runtime.LoadFile(rtPath)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				return nil
			}

			folderEntry, ok := m[folder].(map[string]interface{})
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			agentBlock, ok := folderEntry["agent"].(map[string]interface{})
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			idleSince, ok := agentBlock["idle_since"]
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			// Parse idle_since as Unix timestamp
			var ts int64
			switch v := idleSince.(type) {
			case int:
				ts = int64(v)
			case int64:
				ts = v
			case float64:
				ts = int64(v)
			default:
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			elapsed := time.Now().Unix() - ts
			if elapsed < 0 {
				elapsed = 0
			}

			fmt.Fprintf(cmd.OutOrStdout(), "idle %s\n", formatIdleDuration(elapsed))
			return nil
		},
	}
}
