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
	cmd.Flags().Bool("answer", false, "Answer mode: permit sending to a waiting agent (still refuses active)")
	cmd.Flags().Bool("force", false, "Skip idle validation (still validates pane existence)")
	return cmd
}

func runPaneSend(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	text := args[1]
	noEnter, _ := cmd.Flags().GetBool("no-enter")
	answer, _ := cmd.Flags().GetBool("answer")
	force, _ := cmd.Flags().GetBool("force")
	server, _ := cmd.Flags().GetString("server")

	// Step 1: Validate pane exists. Exit codes follow the pane-family scheme
	// shared with window-name: 2 = pane missing, 3 = other tmux failure. The
	// in-handler os.Exit stays because non-1 codes are genuinely needed here.
	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	// Step 2: Validate agent state (unless --force — the skip-everything
	// override, which wins over --answer when both are given). Reads
	// @rk_agent_state via the shared reader. Three known states plus unknown:
	//   idle           → send.
	//   waiting        → refuse (plain); send under --answer — this send IS
	//                    the answer the blocked agent is waiting for.
	//   active         → refuse in both modes (never interrupt a working
	//                    agent unattended).
	//   unknown        → warn and send anyway, both modes (a foreign-agent
	//                    pane — an absent option / unparseable value / a pane
	//                    with no instrumented agent — carries no state to
	//                    gate on).
	if !force {
		ctx, err := pane.ResolvePaneContext(paneID, "", server)
		if err != nil {
			return fmt.Errorf("resolve context: %w", err)
		}
		warning, err := idleGate(paneID, ctx.AgentState, answer)
		if err != nil {
			return err
		}
		if warning != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
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
// resolved agent state (nil = unknown) and the send mode, it reports whether
// a send is allowed — a non-empty warning for the warn-and-proceed case, an
// error carrying the exact refusal contract when refused. Extracted from
// runPaneSend so the gate matrix is unit-testable without the cobra/tmux
// plumbing. (--force never reaches this function — runPaneSend skips the
// state check entirely, which is also why --force wins over --answer.)
//
//	                     plain                --answer
//	nil (unknown)        warn + send          warn + send (same posture)
//	idle                 send                 send
//	waiting              refuse               send (the answer it waits for)
//	active               refuse               refuse
//
// Refusals are three-state aware (state name in message) and exit 1 via the
// returned error; pane existence is validated before the gate either way.
func idleGate(paneID string, agentState *string, answer bool) (warning string, err error) {
	switch {
	case agentState == nil:
		return "agent state unknown — sending anyway", nil
	case *agentState == pane.AgentStateIdle:
		return "", nil
	case answer && *agentState == pane.AgentStateWaiting:
		return "", nil
	case answer:
		return "", fmt.Errorf("agent in pane %s is %s (--answer permits idle and waiting only)", paneID, *agentState)
	default:
		return "", fmt.Errorf("agent in pane %s is not idle (state: %s)", paneID, *agentState)
	}
}

// sendTextArgs builds the tmux argv for literal-text send-keys.
// When server is non-empty, the argv is prepended with `-L <server>`.
//
// Both builders live in internal/pane, shared with the dispatch delivery
// choreography; these are the cobra-layer names this file's tests use.
func sendTextArgs(server, paneID, text string) []string {
	return pane.SendLiteralArgs(server, paneID, text)
}

// sendEnterArgs builds the tmux argv for the trailing Enter send-keys.
// When server is non-empty, the argv is prepended with `-L <server>`.
func sendEnterArgs(server, paneID string) []string {
	return pane.SendKeyArgs(server, paneID, pane.KeyEnter)
}
