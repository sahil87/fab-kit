package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func paneReadyCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "ready <pane>",
		Short: "Probe whether a tmux pane can accept typed input (ready / booting / parked)",
		Long: "Answer one question about a tmux pane: can it accept typed input right now?\n\n" +
			"Before any keystroke the probe checks who owns the pane: while the foreground\n" +
			"command is still a shell (the provider binary has not taken the tty yet), it\n" +
			"reports `booting` and types NOTHING — a shell in cooked mode echoes typed\n" +
			"characters by itself, so the sentinel would echo for a reason that has nothing\n" +
			"to do with an agent being ready. Only once a non-shell process owns the pane\n" +
			"does the echo-and-stability probe run.\n\n" +
			"That probe is purely MECHANICAL — it types a sentinel literally, checks whether\n" +
			"the sentinel echoed, clears it with C-u, and looks at whether the screen is\n" +
			"still moving. It carries no table of known dialogs, presses no other key, and\n" +
			"answers nothing: dialog text is a version treadmill, and a half-matched pattern\n" +
			"pressing Enter into an unknown screen is worse than stalling.\n\n" +
			"  ready    the sentinel echoed — hand the pane its prompt with `fab pane deliver`\n" +
			"  booting  the pane is still a shell, or no echo on a blank/changing screen —\n" +
			"           wait and re-probe\n" +
			"  parked   no echo on a stable screen — a dialog, survey, login wall, or wedged\n" +
			"           process is holding the input; the snippet below shows what\n\n" +
			"SIDE EFFECT: once an agent owns the pane the probe TYPES into it (the sentinel,\n" +
			"cleared with C-u before return), so run it only against panes you own — never\n" +
			"against a pane an agent or a human is actively working in. A pane whose\n" +
			"foreground is still a shell is never typed into at all.\n\n" +
			"Deciding what a parked screen wants is the caller's judgment, which is why every\n" +
			"non-ready report carries the pane, its socket, and a capture snippet. All three\n" +
			"answers exit 0 — the report string is the sole discriminator.\n\n" +
			"--json emits a single {\"state\",\"pane\",\"server\",\"snippet\"} object (snippet is\n" +
			"\"\" when the screen is blank; server is null for the default socket).\n\n" +
			"Exit codes: 0 classified (any of the three); 2 pane missing; 3 other tmux failure.",
		Example: `  fab pane ready %12
  fab pane ready %3 --server work --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneReady(cmd, args[0], jsonFlag)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Emit structured JSON output")
	return cmd
}

// paneReadyJSON is the --json output shape for `fab pane ready` — always an
// object, for every classification (the window-name precedent). Snippet is
// the same trailing-blank-trimmed capture the text report carries ("" when
// the screen is blank); Server is null for the default socket (toNullable).
type paneReadyJSON struct {
	State   string  `json:"state"`
	Pane    string  `json:"pane"`
	Server  *string `json:"server"`
	Snippet string  `json:"snippet"`
}

// runPaneReady validates the pane and reports the gate's classification — the
// same report form `fab dispatch ready` prints, addressed by pane id with no
// dispatch record.
//
// Non-zero exit is reserved for REAL errors — a missing pane (2), any other
// tmux failure (3) — never for a classification: an observed answer is a
// success however inconvenient the answer is.
func runPaneReady(cmd *cobra.Command, paneID string, jsonFlag bool) error {
	server, _ := cmd.Flags().GetString("server")

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	state, snippet, err := pane.NewGate(server).Probe(paneID)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	out := cmd.OutOrStdout()
	if jsonFlag {
		return json.NewEncoder(out).Encode(paneReadyJSON{
			State:   string(state),
			Pane:    paneID,
			Server:  toNullable(server),
			Snippet: snippet,
		})
	}
	fmt.Fprintln(out, state)
	if state == pane.ReadyReady {
		return nil
	}
	// The pane and socket are printed for the judgment rounds that follow a
	// non-ready answer: answering a wall means `tmux [-L <server>] send-keys -t
	// <pane> …` (dispatch-ready report parity).
	fmt.Fprintf(out, "pane: %s\n", paneID)
	if server != "" {
		fmt.Fprintf(out, "server: %s\n", server)
	}
	// No header over an empty snippet: a pane that has drawn nothing yet is the
	// ordinary `booting` case, and a header with nothing under it reads as a
	// truncated report rather than as "there is nothing on the screen".
	if snippet != "" {
		fmt.Fprintf(out, "--- last %d lines ---\n", pane.SnippetLines)
		fmt.Fprint(out, snippet)
	}
	return nil
}
