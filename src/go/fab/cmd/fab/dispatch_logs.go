package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

func dispatchLogsCmd() *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "logs <change> <stage>",
		Short: "Print the dispatch log (combined stdout+stderr)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchLogs(cmd, args[0], args[1], tail)
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 0, "Print only the last N lines (0 = all)")
	return cmd
}

// runDispatchLogs prints the headless wrapper's combined stdout+stderr log.
//
// A PANE dispatch has no log file: an interactive worker's output is tmux
// scrollback, not a redirected stream. Rather than surface the generic
// missing-log message (which reads as "the log is gone" and offers no next
// step), it reports the mode fact and names the pane-mode equivalent,
// `fab pane capture <pane>` — carrying the dispatch's recorded tmux socket as
// `-L <server>` when it has one, so the suggested command is copy-pasteable
// against the same server the pane actually lives on (a socket-scoped pane is
// invisible to a default-socket capture).
func runDispatchLogs(cmd *cobra.Command, changeArg, stage string, tail int) error {
	dir, _, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	if rec, lerr := dispatch.Load(dir, stage); lerr == nil && rec.IsPane() {
		return fmt.Errorf("%s/%s is a --pane dispatch and keeps no log file (an interactive worker's output is tmux scrollback); read it with `%s`",
			changeArg, stage, paneCaptureHint(rec.Server, rec.Pane))
	}

	data, err := os.ReadFile(dispatch.LogPath(dir, stage))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dispatch log for %s/%s", changeArg, stage)
		}
		return fmt.Errorf("read dispatch log: %w", err)
	}

	if tail > 0 {
		data = dispatch.Tail(data, tail)
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

// paneCaptureHint renders the copy-pasteable `fab pane capture` command that
// reads a pane dispatch's window.
//
// The socket is included as `-L <server>` when the dispatch recorded one (the
// `fab pane` family's persistent `--server`/`-L` flag), because a pane on a
// non-default socket is not reachable from a default-socket capture — a hint
// omitting it would send the reader to an empty or wrong server. An empty
// server means the dispatch used tmux's default-socket resolution, so the flag
// is omitted and the reader inherits the same resolution.
func paneCaptureHint(server, paneID string) string {
	if server != "" {
		return fmt.Sprintf("fab pane capture -L %s %s", server, paneID)
	}
	return fmt.Sprintf("fab pane capture %s", paneID)
}
