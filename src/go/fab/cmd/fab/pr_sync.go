package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/prmeta"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// gh seams (the runner-variable convention used across cmd/fab — operator.go's
// rkOperatorPath, pane_map.go's rkPanesRunner) so tests stub the two network
// calls without a live gh. prBodyReader errors when the branch has no open PR
// (gh pr view exits non-zero); prBodyWriter applies a new body via stdin.
var prBodyReader = func(repoDir string) (string, error) {
	cmd := exec.Command("gh", "pr", "view", "--json", "body", "-q", ".body")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var prBodyWriter = func(repoDir, body string) error {
	cmd := exec.Command("gh", "pr", "edit", "--body-file", "-")
	cmd.Dir = repoDir
	cmd.Stdin = strings.NewReader(body)
	return cmd.Run()
}

func prSyncCmd() *cobra.Command {
	var prType string
	var issues string

	cmd := &cobra.Command{
		Use:   "pr-sync <change>",
		Short: "Refresh the marker-delimited `## Meta` block of the branch's open PR",
		Long: "Renders the `## Meta` block for a change (the same inputs as " +
			"pr-meta), reads the current PR body via gh, splices the fresh " +
			"block between the `<!-- fab pr-meta:start/end -->` markers " +
			"(adopting a pre-marker bare `## Meta` section in place), and " +
			"applies the result via `gh pr edit --body-file -` ONLY when the " +
			"body changed — a second run with unchanged inputs is a no-op. " +
			"Exits non-zero with no side effects when there is no fab context " +
			"or no open PR on the branch, mirroring pr-meta's " +
			"graceful-degradation contract. Orchestrators call this at stage " +
			"boundaries; stage workers never do.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			data, ok, err := prmeta.Gather(fabRoot, args[0], prType, issues)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("no fab context for %q (change unresolved or .status.yaml absent)", args[0])
			}
			// Same version-provenance seam as prMetaCmd: package main's
			// `version` is threaded here so Render stays a pure function of Data.
			data.Version = version

			repoDir := filepath.Dir(fabRoot)
			applied, err := syncPRBody(prmeta.Render(data),
				func() (string, error) { return prBodyReader(repoDir) },
				func(body string) error { return prBodyWriter(repoDir, body) })
			if err != nil {
				return err
			}
			if !applied {
				fmt.Fprintln(cmd.OutOrStdout(), "PR meta already current — no edit")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PR meta refreshed")
			return nil
		},
	}

	cmd.Flags().StringVar(&prType, "type", "", "Resolved PR type (feat|fix|refactor|docs|test|ci|chore) — required")
	cmd.Flags().StringVar(&issues, "issues", "", "Space-joined issue IDs (e.g. \"DEV-1 DEV-2\") — optional")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

// syncPRBody is the network-free core of pr-sync: read the current body,
// splice, compare, and apply via write only when the splice changed something.
// Returns whether an edit was applied. A read failure (no open PR on the
// branch) surfaces as an error before any write — no side effects.
func syncPRBody(rendered string, read func() (string, error), write func(string) error) (bool, error) {
	body, err := read()
	if err != nil {
		return false, fmt.Errorf("no open PR on the current branch (gh pr view failed): %w", err)
	}
	spliced := prmeta.Splice(body, rendered)
	if spliced == body {
		return false, nil
	}
	if err := write(spliced); err != nil {
		return false, fmt.Errorf("gh pr edit: %w", err)
	}
	return true, nil
}
