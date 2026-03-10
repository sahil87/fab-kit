# Intake: Ship fab Go Binary

**Change**: 260305-g0uq-2-ship-fab-go-binary
**Created**: 2026-03-05
**Status**: Draft

## Origin

> 2-ship-fab-go-binary: Ship the fab Go binary — parity test harness to verify Go binary produces identical output to bash scripts, local cross-compilation in fab-release.sh for 4 platform targets, per-platform kit archives, update fab-upgrade.sh with platform detection, shim layer in old shell scripts, update skill callers.

Follows the build change (260305-bhd6-1-build-fab-go-binary). Once the Go binary exists and passes parity tests, this change integrates it into the release pipeline and switches all callers.

## Why

1. **Parity verification**: The Go binary must produce identical output to the bash scripts before any caller is switched. A test harness running both implementations against the same fixtures and diffing all outputs (stdout, stderr, exit codes, file mutations) is the safety gate.

2. **Distribution**: Users receive pre-built binaries — they should never need a Go toolchain. The release script must cross-compile for darwin-arm64, darwin-amd64, linux-arm64, linux-amd64 and produce per-platform archives.

3. **Graceful switchover**: Shell scripts become shims that delegate to the Go binary when available, with bash fallback. This allows progressive rollout and easy rollback.

## What Changes

### Parity Test Harness

A test suite at `src/go/fab/test/parity/` that:

1. Maintains fixture files (`.status.yaml` variants, change directories, backlog files) covering all operations
2. For each operation, runs both:
   - `bash fab/.kit/scripts/lib/statusman.sh <subcommand> <args>`
   - `fab status <subcommand> <args>`
3. Diffs stdout, stderr, exit code, and any file mutations (before/after diff of `.status.yaml`)
4. Covers all subcommands across all 8 ported scripts
5. Can be run via `go test ./test/parity/...` or a wrapper script

The existing benchmark fixtures at `src/benchmark/fixtures/` provide a starting point but only cover statusman. Fixtures needed for resolve, logman, preflight, changeman, calc-score, and archiveman.

### Local Cross-Compilation in fab-release.sh

Modify `src/scripts/fab-release.sh` to:

1. Build Go binary for 4 targets: `GOOS/GOARCH go build -o fab/.kit/bin/fab ./cmd/fab` for each platform
2. Produce per-platform archives: `kit-darwin-arm64.tar.gz`, `kit-darwin-amd64.tar.gz`, `kit-linux-arm64.tar.gz`, `kit-linux-amd64.tar.gz`
3. Also produce generic `kit.tar.gz` (no binary, fallback for unsupported platforms)
4. Upload all 5 assets to GitHub Release

Build happens locally (~15 seconds for all 4 targets). No CI needed — Go cross-compiles cleanly without CGo.

### Platform Detection in fab-upgrade.sh

Modify `fab/.kit/scripts/fab-upgrade.sh` to:

1. Detect platform: `uname -s` → os, `uname -m` → arch (with normalization: x86_64→amd64, aarch64→arm64)
2. Try platform-specific archive first: `kit-${os}-${arch}.tar.gz`
3. Fall back to generic `kit.tar.gz` if platform-specific not available
4. The shim in shell scripts handles the case where no binary exists gracefully

### Bootstrap One-Liner Update

Update README and docs with platform-aware bootstrap:

```bash
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac; mkdir -p fab; curl -sL "https://github.com/wvrdz/fab-kit/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

### Shim Layer in Shell Scripts

Each lib/ shell script gets a shim at the top:

```bash
if command -v fab >/dev/null 2>&1; then
  fab <subcommand> "$@"
  exit $?
fi
# ... original bash implementation continues below ...
```

This means skill callers don't need to change their invocations immediately — `bash statusman.sh progress-map <change>` transparently delegates to the Go binary.

### Skill Caller Switchover

After shim validation, update all skill scripts and bash callers to invoke the Go binary directly:

- `bash fab/.kit/scripts/lib/statusman.sh <subcommand> <args>` → `fab status <subcommand> <args>`
- `bash fab/.kit/scripts/lib/resolve.sh <args>` → `fab resolve <args>`
- `bash fab/.kit/scripts/lib/logman.sh <args>` → `fab log <args>`
- `bash fab/.kit/scripts/lib/preflight.sh <args>` → `fab preflight <args>`
- `bash fab/.kit/scripts/lib/changeman.sh <args>` → `fab change <args>`
- `bash fab/.kit/scripts/lib/calc-score.sh <args>` → `fab score <args>`
- `bash fab/.kit/scripts/lib/archiveman.sh <args>` → `fab archive <args>`

Affected callers: all skill files in `fab/.kit/skills/`, batch scripts in `fab/.kit/scripts/`, and `_scripts.md` documentation.

## Affected Memory

- `fab-workflow/distribution`: (modify) Document per-platform archives, platform detection in upgrade/bootstrap, Go binary in kit structure
- `fab-workflow/kit-architecture`: (modify) Document shim layer, binary location at `fab/.kit/bin/fab`

## Impact

- **Release pipeline**: `src/scripts/fab-release.sh` — adds Go build step (~15s), produces 5 archives instead of 1
- **Upgrade pipeline**: `fab/.kit/scripts/fab-upgrade.sh` — adds platform detection, tries platform-specific archive first
- **All lib/ scripts**: Shim added at top of each (non-destructive, preserves original implementation)
- **All skill files**: Caller invocations updated from bash scripts to `fab` binary
- **Documentation**: README bootstrap one-liner, distribution memory file

## Open Questions

- None — all design decisions resolved in preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Parity test diffs stdout/stderr/exit codes/file mutations | Discussed — this is the safety gate before switchover | S:90 R:85 A:90 D:95 |
| 2 | Certain | Local cross-compilation (not CI) | Discussed — Go cross-compiles cleanly without CGo, ~15s for 4 targets | S:85 R:85 A:85 D:90 |
| 3 | Certain | Per-platform archives: kit-{os}-{arch}.tar.gz | Discussed — user never thinks about platform, curl/upgrade detect automatically | S:90 R:80 A:90 D:90 |
| 4 | Certain | Generic kit.tar.gz as fallback (no binary) | Discussed — graceful degradation for unsupported platforms | S:85 R:90 A:85 D:90 |
| 5 | Certain | Shim layer in shell scripts before direct switchover | Discussed — transparent delegation, easy rollback, callers don't change initially | S:90 R:90 A:85 D:90 |
| 6 | Confident | Binary placed at `fab/.kit/bin/fab` | Follows convention of binaries in bin/ directory. Alternative: `fab/.kit/fab` at root. bin/ is cleaner | S:70 R:90 A:80 D:70 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
