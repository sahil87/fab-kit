package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// noneToken is the --or-none success output for "no change resolves". Exactly
// "(none)" — not "none" (a legal 4-char change ID, so it would collide with
// --id output) and not empty (illegible in transcripts, and hazardous in
// command substitution: `cd $(fab resolve --dir ...)` with empty output cds
// to $HOME). It replaces the mode-specific output for every output mode.
const noneToken = "(none)"

func resolveCmd() *cobra.Command {
	var outputMode string

	cmd := &cobra.Command{
		Use:   "resolve [change]",
		Short: "Resolve a change reference to a canonical output",
		Example: `  # 4-char ID of the active change (default output)
  fab resolve

  # Full folder name for a change reference
  fab resolve --folder b91h

  # Path to the change's .status.yaml
  fab resolve --status b91h

  # tmux pane ID, targeting a specific tmux socket
  fab resolve --pane -L work b91h

  # probe form: absence is data — prints "(none)", exit 0
  fab resolve --folder --or-none`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changeArg := ""
			if len(args) > 0 {
				changeArg = args[0]
			}
			server, _ := cmd.Flags().GetString("server")
			return runResolve(cmd, changeArg, outputMode, server, mustBool(cmd, "or-none"))
		},
	}

	// Register the --id, --folder, --dir, --status, --pane flags matching the
	// bash interface. The five booleans encode a single output enum, so they
	// are mutually exclusive — conflicting flags fail loudly instead of being
	// silently resolved by a priority chain.
	cmd.Flags().Bool("id", false, "Output 4-char change ID (default)")
	cmd.Flags().Bool("folder", false, "Output full folder name")
	cmd.Flags().Bool("dir", false, "Output directory path")
	cmd.Flags().Bool("status", false, "Output .status.yaml path")
	cmd.Flags().Bool("pane", false, "Output tmux pane ID")
	cmd.Flags().StringP("server", "L", "", "Target tmux socket label for --pane (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.")
	// --or-none is NOT part of the output-mode group — it composes with every
	// output mode (absence-as-data opt-in; the flagless default stays
	// absence-as-error for callers that want the hard stop).
	cmd.Flags().Bool("or-none", false, "Print \"(none)\" and exit 0 when no change resolves (not-found always; ambiguous only without <change>); real errors still fail")
	cmd.MarkFlagsMutuallyExclusive("id", "folder", "dir", "status", "pane")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		switch {
		case mustBool(cmd, "id"):
			outputMode = "id"
		case mustBool(cmd, "folder"):
			outputMode = "folder"
		case mustBool(cmd, "dir"):
			outputMode = "dir"
		case mustBool(cmd, "status"):
			outputMode = "status"
		case mustBool(cmd, "pane"):
			outputMode = "pane"
		default:
			outputMode = "id"
		}
		return nil
	}

	return cmd
}

// mustBool reads a bool flag, ignoring the lookup error (flags are registered
// statically above — a typo would fail every test immediately).
func mustBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

// runResolve is the single shared resolve implementation. Both `fab resolve`
// and the thin `fab change resolve` wrapper (folder mode, orNone always false
// — the wrapper is deliberately flag-free) execute it, so the two spellings
// of the same operation can never drift in behavior, help, or error strings.
//
// orNone maps STATE-SENTINEL resolution failures to a successful noneToken
// result: ErrNotFound always (bare and override — the reference names nothing),
// ErrAmbiguous only on bare resolution ("multiple changes exist, none active"
// IS the no-active-change state; a named-but-multi-matching override is a real
// user error). Infrastructure errors (FabRoot failure, I/O) stay errors, flag
// or no flag — the mapping applies to the change-resolution step only, so a
// pane-lookup failure after successful resolution is never mapped.
func runResolve(cmd *cobra.Command, changeArg, outputMode, server string, orNone bool) error {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		if orNone && (errors.Is(err, resolve.ErrNotFound) ||
			(changeArg == "" && errors.Is(err, resolve.ErrAmbiguous))) {
			fmt.Fprintln(cmd.OutOrStdout(), noneToken)
			return nil
		}
		return err
	}

	w := cmd.OutOrStdout()
	switch outputMode {
	case "id":
		fmt.Fprintln(w, resolve.ExtractID(folder))
	case "folder":
		fmt.Fprintln(w, folder)
	case "dir":
		fmt.Fprintf(w, "fab/changes/%s/\n", folder)
	case "status":
		fmt.Fprintf(w, "fab/changes/%s/.status.yaml\n", folder)
	case "pane":
		return resolvePaneOutput(cmd, folder, server)
	}
	return nil
}

// resolvePaneOutput resolves the tmux pane ID for a change's worktree. With
// --server set, discovery targets that socket SERVER-WIDE and the $TMUX guard
// is skipped — "current session" is undefined on a foreign socket, and the
// callers that need cross-socket lookup (daemons) are not inside that server.
// Without --server, behavior is unchanged: current-session discovery, $TMUX
// required.
func resolvePaneOutput(cmd *cobra.Command, folder, server string) error {
	mode := sessionDefault
	if server == "" {
		if os.Getenv("TMUX") == "" {
			return fmt.Errorf("not inside a tmux session")
		}
	} else {
		mode = sessionAll
	}

	panes, err := discoverPanes(mode, "", server)
	if err != nil {
		return err
	}

	matches, warning := matchPanesByFolder(panes, folder, resolvePaneChange)

	if len(matches) == 0 {
		return fmt.Errorf("no tmux pane found for change %q", folder)
	}

	if warning != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), warning)
	}

	fmt.Fprintln(cmd.OutOrStdout(), matches[0])
	return nil
}
