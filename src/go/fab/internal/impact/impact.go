// Package impact computes git-diff line counts (added/deleted/net) between
// two refs, optionally excluding pathspec patterns. It is the canonical
// shortstat math shared by the fab impact CLI subcommand and the
// status-finish hook for the .status.yaml `true_impact` block.
package impact

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Result is the canonical shortstat result.
type Result struct {
	Added     int
	Deleted   int
	Net       int
	Excluding *Pair
}

// Pair holds a single shortstat triple.
type Pair struct {
	Added   int
	Deleted int
	Net     int
}

var (
	insertionsRe = regexp.MustCompile(`(\d+) insertion`)
	deletionsRe  = regexp.MustCompile(`(\d+) deletion`)
)

// Compute runs `git diff --shortstat <base>...<head>` once unconditionally
// (the raw pass) and a second time with `:(exclude)<pattern>` pathspec args
// when excludes is non-empty (the excluding pass). On merge-base/git-diff
// failure it returns an error; callers decide whether to abort or skip.
func Compute(base, head string, excludes []string) (Result, error) {
	if base == "" {
		return Result{}, fmt.Errorf("base ref is empty (merge-base resolution likely failed upstream)")
	}
	if head == "" {
		head = "HEAD"
	}

	rawAdded, rawDeleted, err := runShortstat(base, head, nil)
	if err != nil {
		return Result{}, fmt.Errorf("git diff (raw): %w", err)
	}

	res := Result{
		Added:   rawAdded,
		Deleted: rawDeleted,
		Net:     rawAdded - rawDeleted,
	}

	if len(excludes) == 0 {
		return res, nil
	}

	exAdded, exDeleted, err := runShortstat(base, head, excludes)
	if err != nil {
		return Result{}, fmt.Errorf("git diff (excluding): %w", err)
	}

	res.Excluding = &Pair{
		Added:   exAdded,
		Deleted: exDeleted,
		Net:     exAdded - exDeleted,
	}
	return res, nil
}

// ComputeForRepo loads `true_impact_exclude` from fabRoot's config and
// delegates to Compute. fabRoot points to the `fab/` directory.
func ComputeForRepo(fabRoot, base, head string) (Result, error) {
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return Result{}, err
	}
	return Compute(base, head, cfg.TrueImpactExclude)
}

func runShortstat(base, head string, excludes []string) (int, int, error) {
	args := []string{"diff", "--shortstat", base + "..." + head}
	if len(excludes) > 0 {
		args = append(args, "--", ".")
		for _, p := range excludes {
			args = append(args, ":(exclude)"+p)
		}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	added, deleted := parseShortstat(string(out))
	return added, deleted, nil
}

// parseShortstat extracts insertions/deletions from a git --shortstat line.
// Missing clauses default to zero (e.g., "1 file changed, 5 insertions(+)").
func parseShortstat(line string) (int, int) {
	added := 0
	deleted := 0
	if m := insertionsRe.FindStringSubmatch(line); m != nil {
		added, _ = strconv.Atoi(m[1])
	}
	if m := deletionsRe.FindStringSubmatch(line); m != nil {
		deleted, _ = strconv.Atoi(m[1])
	}
	return added, deleted
}
