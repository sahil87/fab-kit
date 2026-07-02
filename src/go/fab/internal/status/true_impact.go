package status

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// WriteTrueImpact computes the true_impact block for the given stage and
// writes it into .status.yaml. Best-effort: failures emit a one-line stderr
// warning and return nil so the caller's stage transition is unaffected.
//
// Stage MUST be one of: "apply", "hydrate", "ship". Other stages are silently
// ignored. In the standard pipeline nothing is committed until /git-pr (ship),
// so at apply-finish and hydrate-finish HEAD == merge-base and the three-dot
// diff is empty (0/0/0); the ship-finish recompute — run after /git-pr commits
// and pushes the branch — is the authoritative write, superseding the earlier
// zeros in place via computed_at_stage. The apply/hydrate writes are kept
// because they carry real values in non-standard flows where commits already
// exist before ship (adopted off-pipeline changes, manual mid-apply commits).
func WriteTrueImpact(statusFile *sf.StatusFile, statusPath, fabRoot, stage string) error {
	if stage != "apply" && stage != "hydrate" && stage != "ship" {
		return nil
	}

	repoDir := filepath.Dir(fabRoot)
	base, err := resolveMergeBase(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fab: skipping true_impact (%s)\n", err)
		return nil
	}

	res, err := impact.ComputeForRepo(fabRoot, base, "HEAD")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fab: skipping true_impact (%s)\n", err)
		return nil
	}

	ti := &sf.TrueImpact{
		Added:           res.Added,
		Deleted:         res.Deleted,
		Net:             res.Net,
		ComputedAt:      time.Now().UTC().Format(time.RFC3339),
		ComputedAtStage: stage,
	}
	if res.Excluding != nil {
		ti.Excluding = &sf.TrueImpactPair{
			Added:   res.Excluding.Added,
			Deleted: res.Excluding.Deleted,
			Net:     res.Excluding.Net,
		}
	}
	if res.Tests != nil {
		ti.Tests = &sf.TrueImpactPair{
			Added:   res.Tests.Added,
			Deleted: res.Tests.Deleted,
			Net:     res.Tests.Net,
		}
	}
	statusFile.TrueImpact = ti

	return statusFile.Save(statusPath)
}

// resolveMergeBase returns the merge-base of HEAD against origin/main, falling
// back to origin/master. The git invocation is pinned to repoDir (via
// `cmd.Dir`) so callers operating from nested git repos (e.g., submodules)
// resolve against the intended repository. Pass an empty repoDir to use the
// process cwd. Returns an actionable error when neither ref resolves.
func resolveMergeBase(repoDir string) (string, error) {
	for _, ref := range []string{"origin/main", "origin/master"} {
		cmd := exec.Command("git", "merge-base", ref, "HEAD")
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		out, err := cmd.Output()
		if err == nil {
			base := strings.TrimSpace(string(out))
			if base != "" {
				return base, nil
			}
		}
	}
	return "", fmt.Errorf("cannot resolve merge-base against origin/main or origin/master")
}
