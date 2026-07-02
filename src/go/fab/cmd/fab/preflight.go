package main

import (
	"fmt"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/preflight"
	"github.com/sahil87/fab-kit/src/go/fab/internal/refresh"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/spf13/cobra"
)

func preflightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight [change-name]",
		Short: "Validate project state and output structured YAML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			changeOverride := ""
			if len(args) > 0 {
				changeOverride = args[0]
			}

			// Self-heal artifact-derived .status.yaml fields (change_type +
			// confidence from intake.md; plan counts from plan.md) before the
			// read-only derivation, so preflight — the orient seam — reflects a
			// hook-bypassing artifact edit (the pull-based successor to the
			// removed artifact-write hook). This runs under the status flock as
			// a locked load-mutate-save, distinct from preflight.Run's pure
			// read. It is BEST-EFFORT: any failure (unresolvable change,
			// unreadable status, scoring hiccup) is swallowed so preflight still
			// surfaces state. LiveAcceptance already makes acceptance counts
			// correct-on-read, so the load-bearing gain here is change_type +
			// confidence self-healing.
			refreshPreflightState(fabRoot, changeOverride)

			result, err := preflight.Run(fabRoot, changeOverride)
			if err != nil {
				return err
			}

			fmt.Print(preflight.FormatYAML(result))
			return nil
		},
	}
}

// refreshPreflightState runs the artifact-derived recompute for the resolved
// change under the status flock, persisting a single Save when dirty. Every
// failure is swallowed — preflight must orient even when a recompute cannot
// run. Kept in cmd/fab (not internal/preflight) so preflight.Run stays a pure
// reader and the write path routes through the same lockfile.WithLock discipline
// as fab status.
func refreshPreflightState(fabRoot, changeOverride string) {
	statusPath, err := resolve.ToAbsStatus(fabRoot, changeOverride)
	if err != nil {
		return
	}
	_ = lockfile.WithLock(statusPath, func() error {
		st, err := sf.Load(statusPath)
		if err != nil {
			return nil // swallow — preflight.Run will surface the real load error
		}
		changeDir := filepath.Dir(statusPath)
		dirty, err := refresh.Refresh(fabRoot, changeDir, st)
		if err != nil {
			return nil // swallow — best-effort self-heal
		}
		if dirty {
			_ = st.Save(statusPath)
		}
		return nil
	})
}
