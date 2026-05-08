package status

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// WriteTrueImpact computes the true_impact block for the given stage and
// writes it into .status.yaml. Best-effort: failures emit a one-line stderr
// warning and return nil so the caller's stage transition is unaffected.
//
// Stage MUST be one of: "apply", "hydrate". Other stages are silently
// ignored — only the apply-finish and hydrate-finish hooks compute the
// block per spec assumption #16.
func WriteTrueImpact(statusFile *sf.StatusFile, statusPath, fabRoot, stage string) error {
	if stage != "apply" && stage != "hydrate" {
		return nil
	}

	base, err := resolveMergeBase()
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
	statusFile.TrueImpact = ti

	return statusFile.Save(statusPath)
}

// resolveMergeBase returns the merge-base of HEAD against origin/main, falling
// back to origin/master. Returns an actionable error when neither resolves.
func resolveMergeBase() (string, error) {
	for _, ref := range []string{"origin/main", "origin/master"} {
		out, err := exec.Command("git", "merge-base", ref, "HEAD").Output()
		if err == nil {
			base := strings.TrimSpace(string(out))
			if base != "" {
				return base, nil
			}
		}
	}
	return "", fmt.Errorf("cannot resolve merge-base against origin/main or origin/master")
}
