package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func paneSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <pane> <text>",
		Short: "Send keystrokes to a tmux pane with validation",
		Args:  cobra.ExactArgs(2),
		RunE:  runPaneSend,
	}
	cmd.Flags().Bool("no-enter", false, "Don't append Enter keystroke")
	cmd.Flags().Bool("force", false, "Skip idle validation (still validates pane existence)")
	return cmd
}

func runPaneSend(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	text := args[1]
	noEnter, _ := cmd.Flags().GetBool("no-enter")
	force, _ := cmd.Flags().GetBool("force")

	// Step 1: Validate pane exists
	if err := pane.ValidatePane(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(1)
	}

	// Step 2: Validate agent idle (unless --force)
	if !force {
		ctx, err := pane.ResolvePaneContext(paneID)
		if err != nil {
			return fmt.Errorf("resolve context: %w", err)
		}

		state := "unknown"
		if ctx.AgentState != nil {
			state = *ctx.AgentState
		}

		if state != "idle" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: agent in pane %s is not idle (state: %s)\n", paneID, state)
			os.Exit(1)
		}
	}

	// Step 3: Send keys
	tmuxArgs := []string{"send-keys", "-t", paneID, text}
	if !noEnter {
		tmuxArgs = append(tmuxArgs, "Enter")
	}

	if err := exec.Command("tmux", tmuxArgs...).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Sent to %s\n", paneID)
	return nil
}
