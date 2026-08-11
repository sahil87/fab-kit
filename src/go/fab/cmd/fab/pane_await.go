package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func paneAwaitCmd() *cobra.Command {
	var (
		file        string
		timeoutSecs int
	)
	cmd := &cobra.Command{
		Use:   "await <pane> [--file <path>] [--timeout <secs>]",
		Short: "Block until a pane's agent reports idle or a contract file appears",
		Long: "Block until ANY waitable signal fires, print a one-word report, and exit 0 —\n" +
			"the record-free completion primitive (the `fab dispatch wait` observer\n" +
			"conventions, addressed by pane id with no dispatch record):\n\n" +
			"  idle     the pane's @rk_agent_state resolved to idle\n" +
			"  file     the --file path exists\n" +
			"  running  --timeout expired with neither signal fired (exit 0 — the timeout\n" +
			"           bounds the observer, never the pane)\n" +
			"  gone     the pane died mid-wait (exit 2 — the wait cannot complete)\n\n" +
			"Signals compose as OR: with both --file and an instrumented pane, whichever\n" +
			"fires first wins. An uninstrumented pane (no @rk_agent_state) with no --file\n" +
			"is an immediate error — there is nothing observable to wait on.\n\n" +
			"The report string is the sole discriminator; the first check runs before any\n" +
			"sleep, so an already-fired signal returns immediately.\n\n" +
			"Exit codes: 0 idle/file/running; 1 nothing to wait on, or a signal could\n" +
			"not be read; 2 pane missing or died mid-wait; 3 other tmux failure.",
		Example: `  fab pane await %12
  fab pane await %12 --file .fab-dispatch/b91h/apply-result.yaml
  fab pane await %3 --timeout 60 --server work`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneAwait(cmd, args[0], file, timeoutSecs)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "also fire when this contract file appears (composes as OR with the agent-state signal)")
	cmd.Flags().IntVar(&timeoutSecs, "timeout", 300,
		"Upper bound in seconds; on expiry print `running` and exit 0 (0 = wait indefinitely)")
	return cmd
}

// runPaneAwait validates the pane, decides what there is to watch, and blocks
// on the pure pane.Await loop. The observer re-reads the signals each tick;
// the record is nil because there is none — this verb is deliberately
// record-free, the generic twin of `fab dispatch wait`'s record-keyed block.
//
// The instrumented-pane decision is made ONCE at entry: an uninstrumented pane
// with no --file errors immediately rather than blocking to a `running` report
// that would mean "I was never watching anything". A fired signal wins over a
// mid-tick pane death (file/idle checked before liveness) — a worker that
// wrote its contract file and exited still reports `file`, not `gone`.
func runPaneAwait(cmd *cobra.Command, paneID, file string, timeoutSecs int) error {
	server, _ := cmd.Flags().GetString("server")

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	watchState := paneAgentInstrumented(paneID, server)
	if !watchState && file == "" {
		return fmt.Errorf("pane %s is not waitable: no %s option set and no --file given — there is nothing observable to wait on",
			paneID, pane.AgentStateOption)
	}

	observe := func() (pane.AwaitReport, error) {
		if file != "" {
			_, err := os.Stat(file)
			switch {
			case err == nil:
				return pane.AwaitFile, nil
			case !errors.Is(err, os.ErrNotExist):
				// "Absent" is a signal that has not fired; anything else means
				// the signal could not be READ. Swallowing it would re-poll a
				// permission/IO failure to the bound and then report `running`
				// — a claim about a signal this observer can never see.
				return "", fmt.Errorf("stat --file %s: %w", file, err)
			}
		}
		if watchState {
			if state, _ := pane.AgentDisplayFromOption(pane.ReadAgentStateOption(paneID, server)); state == pane.AgentStateIdle {
				return pane.AwaitIdle, nil
			}
		}
		// PaneAlive's conflation of "pane gone" with "tmux unreachable" is
		// deliberate HERE: a dispatch pane is routinely the last pane on its
		// socket, so its death takes the server down with it and the probe then
		// fails with `no server running` rather than a missing-pane error.
		// Mid-wait, either way the wait cannot complete, which is what `gone`
		// means. Exit 3 stays reserved for the ENTRY validation above — a
		// tmux failure before there was ever anything to observe.
		if !pane.PaneAlive(paneID, server) {
			return pane.AwaitGone, nil
		}
		return "", nil
	}

	report, err := pane.Await(cmd.Context(), observe,
		pane.AwaitTick, time.Duration(timeoutSecs)*time.Second)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), report)
	if report == pane.AwaitGone {
		os.Exit(2)
	}
	return nil
}

// paneAgentInstrumented reports whether the pane carries a parseable
// @rk_agent_state value — the entry gate for watching the idle signal. The
// read is the same one map/send/capture share; an absent or unparseable
// option is the unknown state, and unknown is not a signal.
func paneAgentInstrumented(paneID, server string) bool {
	state, _ := pane.AgentDisplayFromOption(pane.ReadAgentStateOption(paneID, server))
	return state != ""
}
