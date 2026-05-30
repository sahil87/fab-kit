// Package impact computes git-diff line counts (added/deleted/net) between
// two refs, optionally excluding pathspec patterns and optionally attributing
// a test-only subset via test pathspecs. It is the canonical shortstat math
// shared by the fab impact CLI subcommand and the status-finish hook for the
// .status.yaml `true_impact` block. The engine is pure measurement: it never
// derives the impl residual (total − tests) — consumers compute and clamp that
// at render time.
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

// Result is the canonical shortstat result. It stores only measured passes —
// raw (Added/Deleted/Net), Excluding, and Tests. It does NOT compute or store
// the derived impl residual (total − tests); that is the consumers'
// render-time responsibility.
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
// excludes is non-empty (the excluding pass), and a third time scoped to
// testPaths (the test pass) when testPaths is non-empty. The test pass applies
// BOTH the testPaths includes AND the excludes — so the test count lives
// strictly inside the scaffolding-excluded universe (a test fixture under an
// excluded path is not double-counted). All git invocations run in repoDir
// (via `cmd.Dir`) so callers operating from nested git repos (e.g.,
// submodules) compute diffs against the intended repository. Pass an empty
// repoDir to use the process cwd. On merge-base/git-diff failure it returns an
// error; callers decide whether to abort or skip.
//
// Compute stores only the measured passes (raw, Excluding, Tests). It never
// derives or clamps the impl residual — consumers do that at render time.
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
// pathspec. When includes is non-empty each entry is passed as a positive
// pathspec; when excludes is non-empty each is passed as `:(exclude)<pattern>`.
// A `--` separator plus a leading `.` (matching everything) is added when only
// excludes are present, so the exclude pathspecs have something to subtract
// from. When includes are present, the `.` is omitted — the includes ARE the
// scope, and the excludes carve test fixtures back out of it.
//
// Include patterns carry the `:(glob)` pathspec magic so wildcards behave
// gitignore-style: `*` does not cross `/`, and `**` matches zero or more path
// segments (so `**/*_test.go` matches both root-level and nested test files).
// Without `:(glob)`, git's default fnmatch treats `**/` as requiring a leading
// directory segment and silently misses root-level matches. Excludes keep the
// default (literal) matching to preserve the existing directory-prefix
// behavior of `true_impact_exclude` (e.g. `fab/`, `docs/`).
func runShortstat(repoDir, base, head string, includes, excludes []string) (int, int, error) {
	args := []string{"diff", "--shortstat", base + "..." + head}
	if len(includes) > 0 || len(excludes) > 0 {
		args = append(args, "--")
		if len(includes) > 0 {
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
