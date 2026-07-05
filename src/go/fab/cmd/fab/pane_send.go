package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
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

	// Step 1: Validate pane exists. Exit codes follow the pane-family scheme
	// shared with window-name: 2 = pane missing, 3 = other tmux failure. The
	// in-handler os.Exit stays because non-1 codes are genuinely needed here.
	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	// Step 2: Validate agent state (unless --force). Reads @rk_agent_state
	// via the shared reader. Three known states plus unknown:
	//   idle           → send.
	//   active/waiting → refuse, three-state-aware (state name in message).
	//   unknown        → refuse with a DISTINCT message pointing at --force
	//                    (absent option / unparseable / non-Claude pane with
	//                    no instrumented agent).
	if !force {
		ctx, err := pane.ResolvePaneContext(paneID, "", server)
		if err != nil {
			return fmt.Errorf("resolve context: %w", err)
		}
		if err := idleGate(paneID, ctx.AgentState); err != nil {
			return err
		}
	}

	// Step 3: Send keys — use -l for literal text to avoid tmux interpreting
	// key names like "Enter", "Space", "C-c" within the text itself.
	// The trailing Enter keystroke (if needed) is sent as a separate command.
	tmuxArgs := sendTextArgs(server, paneID, text)

	if _, stderr, err := pane.RunCmd("tmux", tmuxArgs...); err != nil {
		return pane.StderrError(fmt.Errorf("tmux send-keys to %s: %w", paneID, err), stderr)
	}

	// Send Enter as a separate non-literal key press
	if !noEnter {
		if _, stderr, err := pane.RunCmd("tmux", sendEnterArgs(server, paneID)...); err != nil {
			return pane.StderrError(fmt.Errorf("tmux send-keys (Enter) to %s: %w", paneID, err), stderr)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Sent to %s\n", paneID)
	return nil
}

// idleGate is the pure decision half of the pane-send state gate: given the
// resolved agent state (nil = unknown), it reports whether a send is allowed
// and, when refused, carries the exact error contract. Extracted from
// runPaneSend so the three-state gate is unit-testable without the cobra/tmux
// plumbing — behavior is identical to the inline switch it replaced.
//
//	nil (unknown)        → distinct "unknown" refusal naming --force
//	active / waiting     → "not idle (state: <state>)" refusal (three-state aware)
//	idle                 → nil (send permitted)
func idleGate(paneID string, agentState *string) error {
	switch {
	case agentState == nil:
		return fmt.Errorf("agent state for pane %s is unknown (missing or unparseable %s) — use --force to send anyway", paneID, pane.AgentStateOption)
	case *agentState != pane.AgentStateIdle:
		return fmt.Errorf("agent in pane %s is not idle (state: %s)", paneID, *agentState)
	}
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
