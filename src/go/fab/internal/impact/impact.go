// Package impact computes git-diff line counts (added/deleted/net) between
// two refs, optionally excluding pathspec patterns. It is the canonical
// shortstat math shared by the fab impact CLI subcommand and the
// status-finish hook for the .status.yaml `true_impact` block.
package impact

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Result is the canonical shortstat result. It holds only *measured* passes —
// the raw triple (Added/Deleted/Net), the optional Excluding pass, and the
// optional Tests pass. No derived `impl` field is stored here: the impl
// residual (total − tests) is computed at render time in the consumers so it
// cannot drift between the two diff passes.
type Result struct {
	Added     int
	Deleted   int
	Net       int
	Excluding *Pair
	Tests     *Pair
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
// (the raw pass), a second time with `:(exclude)<pattern>` pathspec args when
// excludes is non-empty (the excluding pass), and a third time with the test
// include pathspecs PLUS the excludes when testPaths is non-empty (the test
// pass). The test pass counts test lines WITHIN the scaffolding-excluded
// universe — its pathspec combines the test includes with the same
// `:(exclude)<pattern>` args as the excluding pass — so a test fixture under
// an excluded path is not double-counted. All git invocations run in repoDir
// (via `cmd.Dir`) so callers operating from nested git repos (e.g.,
// submodules) compute diffs against the intended repository. Pass an empty
// repoDir to use the process cwd. On merge-base/git-diff failure it returns
// an error; callers decide whether to abort or skip.
func Compute(repoDir, base, head string, excludes, testPaths []string) (Result, error) {
	if base == "" {
		return Result{}, fmt.Errorf("base ref is empty (merge-base resolution likely failed upstream)")
	}
	if head == "" {
		head = "HEAD"
	}

	rawAdded, rawDeleted, err := runShortstat(repoDir, base, head, nil, nil)
	if err != nil {
		return Result{}, fmt.Errorf("git diff (raw): %w", err)
	}

	res := Result{
		Added:   rawAdded,
		Deleted: rawDeleted,
		Net:     rawAdded - rawDeleted,
	}

	if len(excludes) > 0 {
		exAdded, exDeleted, err := runShortstat(repoDir, base, head, nil, excludes)
		if err != nil {
			return Result{}, fmt.Errorf("git diff (excluding): %w", err)
		}
		res.Excluding = &Pair{
			Added:   exAdded,
			Deleted: exDeleted,
			Net:     exAdded - exDeleted,
		}
	}

	if len(testPaths) > 0 {
		tAdded, tDeleted, err := runShortstat(repoDir, base, head, testPaths, excludes)
		if err != nil {
			return Result{}, fmt.Errorf("git diff (tests): %w", err)
		}
		res.Tests = &Pair{
			Added:   tAdded,
			Deleted: tDeleted,
			Net:     tAdded - tDeleted,
		}
	}

	return res, nil
}

// ComputeForRepo loads `true_impact_exclude` and `test_paths` from fabRoot's
// config and delegates to Compute, pinning all git invocations to the repo
// root (`filepath.Dir(fabRoot)`). fabRoot points to the `fab/` directory.
func ComputeForRepo(fabRoot, base, head string) (Result, error) {
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return Result{}, err
	}
	return Compute(filepath.Dir(fabRoot), base, head, cfg.TrueImpactExclude, cfg.TestPaths)
}

// runShortstat runs `git diff --shortstat <base>...<head>` with an optional
// pathspec built from includes and excludes. When includes is non-empty, each
// entry is passed as a literal pathspec; otherwise `.` is used as the base
// path so the `:(exclude)` magic pathspecs apply against the whole tree. Each
// exclude is appended as a `:(exclude)<pattern>` magic pathspec. When both
// slices are empty, no `--` pathspec separator is emitted (the raw pass).
func runShortstat(repoDir, base, head string, includes, excludes []string) (int, int, error) {
	args := []string{"diff", "--shortstat", base + "..." + head}
	if len(includes) > 0 || len(excludes) > 0 {
		args = append(args, "--")
		if len(includes) > 0 {
			// Use the :(glob) magic pathspec so wildcards behave like
			// .gitignore-style globs (notably `**` matches across directory
			// boundaries, and `*` does NOT match `/`). Without it, a pattern
			// like `**/*_test.go` is interpreted literally and silently misses
			// root-level test files (`*` matching `/`), under-counting tests.
			for _, p := range includes {
				args = append(args, ":(glob)"+p)
			}
		} else {
			args = append(args, ".")
		}
		for _, p := range excludes {
			args = append(args, ":(exclude)"+p)
		}
	}
	cmd := exec.Command("git", args...)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
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
