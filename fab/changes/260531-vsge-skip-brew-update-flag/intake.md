# Intake: Add --skip-brew-update flag to update command

**Change**: 260531-vsge-skip-brew-update-flag
**Created**: 2026-05-31
**Status**: Draft

## Origin

> Add a boolean `--skip-brew-update` flag to the `update` command. CONTRACT (cross-toolkit,
> identical in 6 tools): flag name must be EXACTLY `--skip-brew-update`. When set, skip ONLY
> the internal `brew update --quiet` tap-metadata refresh. Everything else runs unchanged:
> the `brew info` version check, the up-to-date short-circuit, and `brew upgrade`. Default
> (absent) = current behavior exactly preserved.
>
> THIS REPO (fab-kit): update logic in `src/go/fab-kit/internal/update.go` (func `Update`,
> the `brew update` call ~L28); wire a real cobra BoolVar flag in
> `src/go/fab-kit/cmd/fab-kit/main.go` `updateCmd()` (~L99). Thread `skipBrewUpdate bool`
> through `Update()`. Match existing subprocess convention (do NOT refactor exec style). Add
> a test asserting `--skip-brew-update` omits `brew update` but still runs `brew upgrade`,
> following the repo's existing test pattern. Build + run the update package tests before
> opening the PR.

One-shot invocation via `/fab-new`. No prior `/fab-discuss` session — the contract above is
the complete and authoritative source. The contract is a **cross-toolkit standard**: the same
flag name (`--skip-brew-update`) and the same semantics (skip only the tap-metadata refresh)
are being added identically across 6 sibling tools. fab-kit's implementation MUST conform to
that shared contract exactly — no local naming or behavioral variation.

## Why

1. **Problem**: `fab update` always runs `brew update --quiet` first to refresh Homebrew's tap
   metadata before checking for a newer `fab-kit`. In CI/automation, batch-update scripts, or
   environments where the tap was *just* refreshed by an outer process, that refresh is
   redundant and can be slow (network round-trip to every tap, governed by a 30s timeout in
   `runWithTimeout`). It is also a common source of friction when `brew update` itself emits
   noise or transiently fails, aborting an upgrade that would otherwise have succeeded against
   already-fresh metadata.

2. **Consequence if not fixed**: Callers who have already refreshed brew metadata (or who
   accept slightly stale metadata to save time) have no way to skip the redundant step. They
   must either tolerate the extra latency/failure surface or bypass `fab update` entirely and
   invoke `brew upgrade fab-kit` by hand — losing the version check, the friendly up-to-date
   short-circuit, and the consistent UX.

3. **Why this approach**: A single, explicitly-named, opt-in boolean flag is the minimal,
   least-surprising surface. The flag is **additive and default-off**, so existing behavior is
   byte-for-byte preserved when the flag is absent. Threading a plain `bool` parameter through
   `Update()` matches the existing function-signature style (`Update(currentVersion string)`)
   and keeps the subprocess/exec convention untouched, as the contract requires. The flag name
   is fixed by the cross-toolkit contract — it is NOT a local design choice.

## What Changes

### 1. `Update()` signature gains a `skipBrewUpdate bool` parameter

**File**: `src/go/fab-kit/internal/update.go`

Change the function signature from:

```go
func Update(currentVersion string) error {
```

to:

```go
func Update(currentVersion string, skipBrewUpdate bool) error {
```

Inside `Update()`, the `brew update --quiet` block (currently ~L26–33) is gated on the flag.
The current block:

```go
// Refresh Homebrew index
fmt.Println("Checking for updates...")
cmd := exec.Command("brew", "update", "--quiet")
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
if err := runWithTimeout(cmd, 30*time.Second); err != nil {
    return fmt.Errorf("could not check for updates (brew update failed): %w", err)
}
```

becomes (illustrative — final form follows the existing exec/subprocess convention exactly,
NO refactor of how commands are constructed or run):

```go
// Refresh Homebrew index (unless skipped)
if !skipBrewUpdate {
    fmt.Println("Checking for updates...")
    cmd := exec.Command("brew", "update", "--quiet")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := runWithTimeout(cmd, 30*time.Second); err != nil {
        return fmt.Errorf("could not check for updates (brew update failed): %w", err)
    }
}
```

**Everything else in `Update()` is untouched**:
- The `isBrewInstalled()` guard (~L17–22) runs as before.
- `brewLatestVersion()` — the `brew info --json=v2 fab-kit` version check (~L36) — runs
  unchanged regardless of the flag.
- The up-to-date short-circuit `if latest == currentVersion { ... }` (~L41–44) runs unchanged.
- The `brew upgrade fab-kit` invocation (~L48–53) runs unchanged.

The `cmd :=` / `cmd =` reuse pattern in the original is preserved: with the `brew update`
block now inside an `if`, the variable for `brew upgrade` will be declared with `:=` rather
than reassigned with `=`. (Trivial, mechanical consequence of the guard — not an exec-style
refactor.)

### 2. Wire a real cobra `BoolVar` flag in `updateCmd()`

**File**: `src/go/fab-kit/cmd/fab-kit/main.go`

The current `updateCmd()` (~L99) returns a bare command:

```go
func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update fab-kit itself via Homebrew",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Update(version)
		},
	}
}
```

becomes (following the `syncCmd()` flag-wiring pattern already present in the same file —
declare a local `bool`, build the command, register the flag with `cmd.Flags().BoolVar`,
return `cmd`):

```go
func updateCmd() *cobra.Command {
	var skipBrewUpdate bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update fab-kit itself via Homebrew",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Update(version, skipBrewUpdate)
		},
	}
	cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false,
		"Skip the brew update tap-metadata refresh (still runs brew info + brew upgrade)")
	return cmd
}
```

Flag name MUST be EXACTLY `--skip-brew-update`. Default value MUST be `false`
(absent = current behavior).

### 3. Add a test for the update package

**File**: `src/go/fab-kit/internal/update_test.go` (new)

No `update_test.go` exists today. Add one following the repo's existing test pattern. The repo
convention (see `download_test.go`, `sync_test.go`) is to use `t.TempDir()` and `t.Setenv()`,
and to exercise real subprocess invocation by placing **fake executables on `PATH`** rather
than mocking the exec layer (which would require refactoring the exec style — explicitly
prohibited).

Test strategy — a fake `brew` shell script on `PATH` that appends each invocation's
subcommand to a log file:

- Create a `t.TempDir()`, write an executable `brew` script there that records `"$1"` (the
  brew subcommand: `update`, `info`, or `upgrade`) to a log file, and emits valid
  `--json=v2` output for `brew info` so `brewLatestVersion()` parses successfully and returns
  a version DIFFERENT from `currentVersion` (so the up-to-date short-circuit does NOT fire and
  `brew upgrade` is reached).
- Prepend the temp dir to `PATH` via `t.Setenv("PATH", tmp+":"+os.Getenv("PATH"))`.
- The `isBrewInstalled()` guard resolves the *running test binary's* path and looks for
  `/Cellar/` — under `go test` it will not contain `/Cellar/`, so `Update()` would
  short-circuit at the guard and never reach the brew calls. The test MUST drive the brew-call
  logic directly. Resolve at apply time which is cleanest and conforms to the contract's "do
  NOT refactor exec style":
  - **Option A**: assert on the gating logic by calling `Update()` and accepting that the
    guard returns early — NOT viable, defeats the test's purpose.
  - **Option B (preferred)**: the fake-`brew`-on-`PATH` test exercises the brew sequence. To
    reach it, the test needs `isBrewInstalled()` to return true. Since refactoring exec is
    prohibited but adding test seams is allowed, the apply stage SHALL choose the lowest-touch
    seam that does not alter production exec behavior — e.g. extract the brew-sequence body
    into a small internal helper the test calls directly, OR make `isBrewInstalled` a package
    var func overridable in tests. The decision between these is deferred to spec/apply (see
    Open Questions); both preserve the exec convention and the default runtime behavior. The
    flag-gating assertion itself is the invariant.

  Core assertions (the contract's acceptance criteria), regardless of seam chosen:
  - With `skipBrewUpdate == true`: the brew log MUST NOT contain `update`, MUST contain
    `info` (version check still runs), and MUST contain `upgrade` (upgrade still runs).
  - With `skipBrewUpdate == false`: the brew log MUST contain `update`, `info`, AND `upgrade`
    (current behavior preserved).

### 4. Build + run the update package tests before opening the PR

Run `go build ./...` and `go test ./internal/...` (or scoped `go test ./internal/ -run Update`)
from `src/go/fab-kit/` and confirm both pass before the PR is opened.

## Affected Memory

No memory updates required. This is an **implementation-only, behavior-preserving** change:
it adds an opt-in CLI flag whose default exactly preserves current `fab update` behavior. The
`update` command appears in `docs/memory/fab-workflow/distribution.md` only as a member of the
workspace-lifecycle command list (`init, upgrade-repo, sync, update, doctor`); the internal
`brew update` step is not documented as spec-level behavior, so no memory file's documented
contract changes.

- `(none)`: No spec-level system behavior changes — additive opt-in flag, default preserves current behavior.

## Impact

- **`src/go/fab-kit/internal/update.go`** — `Update()` signature changes (one new `bool`
  param); one `if !skipBrewUpdate { ... }` guard added around the existing `brew update` block.
- **`src/go/fab-kit/cmd/fab-kit/main.go`** — `updateCmd()` gains a `BoolVar` flag and passes
  the bool to `Update()`. This is the only caller of `Update()` (confirm during apply via grep).
- **`src/go/fab-kit/internal/update_test.go`** (new) — adds update-package test coverage where
  none existed.
- **No API/dependency changes**, no new third-party deps. cobra is already in use.
- **Cross-toolkit contract**: flag name and semantics are fixed by an external standard shared
  across 6 tools — fab-kit MUST NOT deviate.
- **Constitution alignment**: Principle VII (Test Integrity) — the new test verifies
  conformance to the contract, not the reverse. The contract requires CLI test updates for Go
  changes (Additional Constraints) — satisfied by `update_test.go`. `_cli-fab.md` documents the
  `fab` *workflow* CLI surface, not the `fab-kit` workspace binary's `update` subcommand, so it
  does not require an update for this flag (confirm during apply).

## Open Questions

- Test seam for `isBrewInstalled()`: under `go test`, the running binary's path lacks
  `/Cellar/`, so `Update()` short-circuits before reaching the brew calls. To exercise the
  flag-gating logic, the apply stage must pick the lowest-touch test seam that does NOT
  refactor the exec style — extract the brew-sequence into a small internal helper, or make
  `isBrewInstalled` an overridable package-level func var. Which seam is cleanest is deferred
  to spec/apply; both preserve runtime exec behavior. (Tentative — resolved at apply.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name is EXACTLY `--skip-brew-update` | Fixed by cross-toolkit contract, stated verbatim; zero discretion | S:100 R:90 A:100 D:100 |
| 2 | Certain | Flag default is `false`; absent = current behavior preserved byte-for-byte | Explicit in contract; cobra `BoolVar` default `false` is the idiomatic encoding | S:100 R:85 A:100 D:100 |
| 3 | Certain | Skip gates ONLY the `brew update --quiet` block; `brew info` check, up-to-date short-circuit, and `brew upgrade` all run unchanged | Contract enumerates exactly what is skipped vs. preserved | S:100 R:80 A:95 D:100 |
| 4 | Certain | Thread a plain `skipBrewUpdate bool` param through `Update()` | Contract says "thread skipBrewUpdate bool through Update()"; matches existing `Update(currentVersion string)` style | S:95 R:80 A:95 D:95 |
| 5 | Confident | Wire flag via `cmd.Flags().BoolVar` following the existing `syncCmd()` pattern in main.go | `syncCmd()` in the same file already uses this exact pattern (`var x bool` → build cmd → `BoolVar` → return); contract says "wire a real cobra BoolVar" | S:90 R:80 A:90 D:85 |
| 6 | Confident | No memory/spec updates required — additive opt-in flag, behavior-preserving | `update` internals not documented as spec-level behavior; template rule excludes implementation-only changes | S:80 R:75 A:85 D:80 |
| 7 | Confident | Test uses a fake `brew` on `PATH` logging subcommands, asserting `update` omitted-but-`upgrade`-present (skip) and all-three-present (default) | Repo convention (download_test.go, sync_test.go) uses real exec + fake binaries on PATH via `t.Setenv`; honors "do NOT refactor exec style" | S:75 R:70 A:80 D:70 |
| 8 | Tentative | Lowest-touch test seam for `isBrewInstalled()` (helper extraction vs. overridable func var) chosen at apply | `go test` binary path lacks `/Cellar/`, so guard short-circuits; both seams preserve exec convention — pick cleanest at apply, easily reversed | S:55 R:65 A:60 D:45 |

8 assumptions (4 certain, 3 confident, 1 tentative, 0 unresolved).
