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
	server, _ := cmd.Flags().GetString("server")

	// Step 1: Validate pane exists
	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(1)
	}

	// Step 2: Validate agent idle (unless --force)
	if !force {
		ctx, err := pane.ResolvePaneContext(paneID, "", server)
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

	// Step 3: Send keys — use -l for literal text to avoid tmux interpreting
	// key names like "Enter", "Space", "C-c" within the text itself.
	// The trailing Enter keystroke (if needed) is sent as a separate command.
	tmuxArgs := sendTextArgs(server, paneID, text)

	if err := exec.Command("tmux", tmuxArgs...).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}

	// Send Enter as a separate non-literal key press
	if !noEnter {
		if err := exec.Command("tmux", sendEnterArgs(server, paneID)...).Run(); err != nil {
			return fmt.Errorf("tmux send-keys (Enter): %w", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Sent to %s\n", paneID)
	return nil
}

// sendTextArgs builds the tmux argv for literal-text send-keys.
// When server is non-empty, the argv is prepended with `-L <server>`.
func sendTextArgs(server, paneID, text string) []string {
	return pane.WithServer(server, "send-keys", "-t", paneID, "-l", text)
}

// sendEnterArgs builds the tmux argv for the trailing Enter send-keys.
// When server is non-empty, the argv is prepended with `-L <server>`.
func sendEnterArgs(server, paneID string) []string {
	return pane.WithServer(server, "send-keys", "-t", paneID, "Enter")
}
