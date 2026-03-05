# Spec: Ship fab Go Binary

**Change**: 260305-g0uq-2-ship-fab-go-binary
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Windows support — only darwin and linux targets
- CI-based cross-compilation — local `go build` only
- Removal of bash script implementations — shims preserve them as fallback
- Go toolchain requirement for end users — pre-built binaries only

## Parity Testing

### Requirement: Parity Test Harness

A Go test suite at `src/fab-go/test/parity/` SHALL compare the output of each bash script against its Go binary equivalent. For each operation, the test MUST run both:
1. `bash fab/.kit/scripts/lib/{script}.sh <subcommand> <args>` (using the repo's actual scripts)
2. The Go binary equivalent (`fab {command} <subcommand> <args>`)

And diff all four dimensions: stdout, stderr, exit code, and file mutations (before/after diff of any modified files like `.status.yaml` or `.history.jsonl`).

The test harness SHALL use isolated temporary directories per test case, copying fixtures into them so tests don't interfere with each other or with the repo state.

#### Scenario: Parity passes for all operations
- **GIVEN** fixture files for a change with `.status.yaml`, `.history.jsonl`, and change directory structure
- **WHEN** each script operation is run via both bash and Go binary
- **THEN** stdout, stderr, and exit code are identical for both implementations
- **AND** any file mutations (`.status.yaml` changes, `.history.jsonl` appends) produce identical results

#### Scenario: Parity test catches a divergence
- **GIVEN** a Go binary that produces different output than the bash script for some operation
- **WHEN** the parity test suite runs
- **THEN** the test fails with a clear diff showing the divergence (expected from bash vs actual from Go)

### Requirement: Test Fixtures

The parity test suite SHALL maintain fixtures covering all 7 ported scripts and their subcommands:

| Script | Go command | Key operations to cover |
|--------|-----------|------------------------|
| `resolve.sh` | `fab resolve` | `--id`, `--folder`, `--dir`, `--status` with change ID, substring, full name |
| `logman.sh` | `fab log` | `command`, `confidence`, `review`, `transition` |
| `statusman.sh` | `fab status` | `progress-map`, `progress-line`, `current-stage`, `start`, `advance`, `finish`, `reset`, `skip`, `fail`, `set-change-type`, `set-checklist`, `set-confidence` |
| `preflight.sh` | `fab preflight` | Valid change, missing config, missing change, override resolution |
| `changeman.sh` | `fab change` | `new`, `rename`, `switch`, `list`, `resolve` |
| `calc-score.sh` | `fab score` | Normal scoring, `--check-gate`, `--stage intake`, gate pass/fail |
| `archiveman.sh` | `fab archive` | `archive`, `restore`, `list` |

Fixtures SHALL include: `.status.yaml` variants (different stages, states, confidence values), `config.yaml`, `constitution.md`, change directories with intake/spec artifacts, `.history.jsonl`, `workflow.yaml` schema, and archive directory structures.

The existing benchmark fixtures at `src/benchmark/fixtures/` MAY be referenced or extended but the parity suite SHALL maintain its own independent fixture set under `src/fab-go/test/parity/fixtures/`.

#### Scenario: Fixtures cover all scripts
- **GIVEN** the parity test suite
- **WHEN** `go test ./test/parity/...` is run from `src/fab-go/`
- **THEN** at least one test case executes for each of the 7 scripts listed above

### Requirement: Test Execution

The parity tests SHALL be runnable via `go test ./test/parity/...` from `src/fab-go/`. Tests MAY also be wrapped in a shell script for convenience. Tests MUST skip gracefully (not fail) if required tools (`yq`, `jq`) are not installed, since the bash scripts depend on these.

#### Scenario: Run parity tests
- **GIVEN** Go toolchain, `yq`, and `jq` are installed
- **WHEN** `cd src/fab-go && go test ./test/parity/...` is executed
- **THEN** all parity tests pass

#### Scenario: Missing prerequisites
- **GIVEN** `yq` is not installed
- **WHEN** parity tests are run
- **THEN** tests that require bash script execution are skipped with a message, not failed

## Release Pipeline

### Requirement: Go Cross-Compilation

`src/scripts/fab-release.sh` SHALL build the Go binary for 4 platform targets before packaging archives:

| GOOS | GOARCH | Output |
|------|--------|--------|
| `darwin` | `arm64` | `fab` binary for macOS Apple Silicon |
| `darwin` | `amd64` | `fab` binary for macOS Intel |
| `linux` | `arm64` | `fab` binary for Linux ARM64 |
| `linux` | `amd64` | `fab` binary for Linux x86_64 |

Cross-compilation SHALL use `GOOS=<os> GOARCH=<arch> go build -o <output> ./cmd/fab` from the `src/fab-go/` directory. CGo MUST be disabled (`CGO_ENABLED=0`) to ensure static binaries.

The build step SHALL occur after the version bump and before archive packaging.

#### Scenario: Cross-compile for all platforms
- **GIVEN** Go toolchain is installed and `src/fab-go/` contains the binary source
- **WHEN** `fab-release.sh patch` is run
- **THEN** 4 platform binaries are built successfully
- **AND** each binary is a statically-linked executable for its target platform

#### Scenario: Go toolchain not installed
- **GIVEN** `go` is not in PATH
- **WHEN** `fab-release.sh patch` is run
- **THEN** the script exits with an error: "ERROR: Go toolchain not found. Install from https://go.dev/"

### Requirement: Per-Platform Archives

After cross-compilation, `fab-release.sh` SHALL produce 5 release archives:

| Archive | Contents |
|---------|----------|
| `kit-darwin-arm64.tar.gz` | `.kit/` contents + `.kit/bin/fab` (darwin/arm64 binary) |
| `kit-darwin-amd64.tar.gz` | `.kit/` contents + `.kit/bin/fab` (darwin/amd64 binary) |
| `kit-linux-arm64.tar.gz` | `.kit/` contents + `.kit/bin/fab` (linux/arm64 binary) |
| `kit-linux-amd64.tar.gz` | `.kit/` contents + `.kit/bin/fab` (linux/amd64 binary) |
| `kit.tar.gz` | `.kit/` contents only (no binary — fallback for unsupported platforms) |

Each platform archive SHALL contain the same `.kit/` tree as the generic archive, plus the platform-specific binary placed at `.kit/bin/fab`. The binary MUST be executable (`chmod +x`).

All 5 archives SHALL be uploaded as GitHub Release assets via `gh release create`.

#### Scenario: Release produces all archives
- **GIVEN** cross-compilation succeeded for all 4 platforms
- **WHEN** the release is created
- **THEN** 5 `.tar.gz` files are uploaded to the GitHub Release
- **AND** the release summary lists all 5 assets

#### Scenario: Archive structure is correct
- **GIVEN** `kit-linux-arm64.tar.gz` is downloaded and extracted into `fab/`
- **WHEN** the archive contents are listed
- **THEN** `.kit/VERSION`, `.kit/skills/`, `.kit/scripts/`, `.kit/templates/` exist (same as generic)
- **AND** `.kit/bin/fab` exists and is an executable Linux ARM64 binary

### Requirement: Release Script Cleanup

After uploading, `fab-release.sh` SHALL clean up all temporary archives and binaries from the repo root. No build artifacts SHALL remain after a successful release.

#### Scenario: Cleanup after release
- **GIVEN** a successful release
- **WHEN** the script completes
- **THEN** no `kit*.tar.gz` files or `fab` binaries remain in the repo root or `src/fab-go/`

## Upgrade Pipeline

### Requirement: Platform Detection

`fab/.kit/scripts/fab-upgrade.sh` SHALL detect the current platform using:
- OS: `uname -s` normalized to lowercase (e.g., `Darwin` → `darwin`, `Linux` → `linux`)
- Architecture: `uname -m` normalized (`x86_64` → `amd64`, `aarch64` → `arm64`, `arm64` → `arm64`)

#### Scenario: Detect macOS Apple Silicon
- **GIVEN** running on macOS with Apple Silicon
- **WHEN** platform is detected
- **THEN** os=`darwin`, arch=`arm64`

#### Scenario: Detect Linux x86_64
- **GIVEN** running on Linux x86_64
- **WHEN** platform is detected
- **THEN** os=`linux`, arch=`amd64`

### Requirement: Platform-Specific Download

After platform detection, `fab-upgrade.sh` SHALL attempt to download the platform-specific archive `kit-${os}-${arch}.tar.gz` first. If the platform-specific archive is not available (download fails), it SHALL fall back to `kit.tar.gz`.

The download pattern for `gh release download` SHALL change from `--pattern 'kit.tar.gz'` to `--pattern "kit-${os}-${arch}.tar.gz"` with fallback.

#### Scenario: Platform archive available
- **GIVEN** running on linux/amd64 and the release has `kit-linux-amd64.tar.gz`
- **WHEN** `fab-upgrade.sh` runs
- **THEN** `kit-linux-amd64.tar.gz` is downloaded and extracted
- **AND** `fab/.kit/bin/fab` exists and is executable after upgrade

#### Scenario: Platform archive not available (older release)
- **GIVEN** running on linux/amd64 but the release only has `kit.tar.gz` (pre-binary release)
- **WHEN** `fab-upgrade.sh` runs
- **THEN** `kit.tar.gz` is downloaded as fallback
- **AND** `fab/.kit/bin/fab` does NOT exist after upgrade (no binary in generic archive)
- **AND** shell scripts work normally (shims fall through to bash implementation)

#### Scenario: Unsupported platform
- **GIVEN** running on an unrecognized OS/arch combination
- **WHEN** `fab-upgrade.sh` runs
- **THEN** platform-specific download fails, falls back to `kit.tar.gz`
- **AND** a message is printed: "Platform ${os}/${arch} not available, using generic archive"

### Requirement: Upgrade Preserves Binary Location

After the atomic swap (`rm -rf` + `mv`), if the downloaded archive contained `.kit/bin/fab`, the binary SHALL be in place at `fab/.kit/bin/fab` and be executable. No additional steps needed.

#### Scenario: Binary preserved after atomic swap
- **GIVEN** platform archive was downloaded containing `.kit/bin/fab`
- **WHEN** the atomic swap completes
- **THEN** `fab/.kit/bin/fab` exists, is executable, and runs correctly

## Shim Layer

### Requirement: Script Delegation

Each of the 7 ported shell scripts in `fab/.kit/scripts/lib/` SHALL have a shim block inserted at the top (after `set -euo pipefail`, before any other logic) that delegates to the Go binary when available.

The shim SHALL resolve the binary path relative to the script's own location: `${script_dir}/../../bin/fab` where `script_dir` is the directory containing the script. This avoids depending on PATH.

| Script | Go subcommand | Shim invocation |
|--------|--------------|-----------------|
| `resolve.sh` | `resolve` | `exec "$_fab_bin" resolve "$@"` |
| `logman.sh` | `log` | `exec "$_fab_bin" log "$@"` |
| `statusman.sh` | `status` | `exec "$_fab_bin" status "$@"` |
| `preflight.sh` | `preflight` | `exec "$_fab_bin" preflight "$@"` |
| `changeman.sh` | `change` | `exec "$_fab_bin" change "$@"` |
| `calc-score.sh` | `score` | `exec "$_fab_bin" score "$@"` |
| `archiveman.sh` | `archive` | `exec "$_fab_bin" archive "$@"` |

Scripts NOT ported to Go (`frontmatter.sh`, `env-packages.sh`) SHALL NOT receive shims.

#### Scenario: Binary available — shim delegates
- **GIVEN** `fab/.kit/bin/fab` exists and is executable
- **WHEN** `bash fab/.kit/scripts/lib/statusman.sh progress-map <change>` is called
- **THEN** the shim detects the binary and `exec`s `fab status progress-map <change>`
- **AND** the bash implementation below the shim is never reached

#### Scenario: Binary not available — fallback to bash
- **GIVEN** `fab/.kit/bin/fab` does NOT exist (generic archive, or binary deleted)
- **WHEN** `bash fab/.kit/scripts/lib/statusman.sh progress-map <change>` is called
- **THEN** the shim falls through and the original bash implementation executes
- **AND** output is identical to the pre-shim behavior

#### Scenario: Binary exists but is wrong platform
- **GIVEN** `fab/.kit/bin/fab` exists but is for a different OS/arch (not executable on current platform)
- **WHEN** a shell script with shim is called
- **THEN** the shim's `[ -x "$_fab_bin" ]` check fails (or exec fails)
- **AND** the bash implementation executes as fallback

### Requirement: Shim Pattern

The shim block SHALL follow this exact pattern for consistency across all 7 scripts:

```bash
# Shim: delegate to Go binary if available
_fab_bin="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)/../../bin/fab"
if [ -x "$_fab_bin" ]; then
  exec "$_fab_bin" {subcommand} "$@"
fi
```

Where `{subcommand}` is the Go command name from the mapping table above. The shim uses `exec` to replace the shell process entirely (no subprocess overhead). The `readlink -f` resolves symlinks so the shim works regardless of how the script is invoked.

#### Scenario: Shim is consistent across scripts
- **GIVEN** all 7 ported scripts have shims added
- **WHEN** the shim blocks are compared
- **THEN** the pattern is identical except for the `{subcommand}` value

## Skill Caller Switchover

### Requirement: Update _scripts.md Documentation

`fab/.kit/skills/_scripts.md` SHALL be updated to document both calling conventions:
1. **Legacy** (via shell scripts): `bash fab/.kit/scripts/lib/statusman.sh <subcommand> <args>`
2. **Direct** (via Go binary): `fab/.kit/bin/fab status <subcommand> <args>`

The documentation SHOULD recommend the direct convention for new skill development while noting that the legacy convention continues to work via shims.

#### Scenario: Documentation shows both conventions
- **GIVEN** `_scripts.md` has been updated
- **WHEN** an agent reads it for invocation guidance
- **THEN** both legacy and direct calling conventions are documented
- **AND** the mapping between script names and Go subcommands is clear

### Requirement: Update Skill Callers

All skill files in `fab/.kit/skills/` and the shared preamble `_preamble.md` SHOULD be updated to invoke the Go binary directly using the relative path `fab/.kit/bin/fab`:

- `bash fab/.kit/scripts/lib/statusman.sh <subcommand> <args>` → `fab/.kit/bin/fab status <subcommand> <args>`
- `bash fab/.kit/scripts/lib/resolve.sh <args>` → `fab/.kit/bin/fab resolve <args>`
- `bash fab/.kit/scripts/lib/logman.sh <args>` → `fab/.kit/bin/fab log <args>`
- `bash fab/.kit/scripts/lib/preflight.sh <args>` → `fab/.kit/bin/fab preflight <args>`
- `bash fab/.kit/scripts/lib/changeman.sh <args>` → `fab/.kit/bin/fab change <args>`
- `bash fab/.kit/scripts/lib/calc-score.sh <args>` → `fab/.kit/bin/fab score <args>`
- `bash fab/.kit/scripts/lib/archiveman.sh <args>` → `fab/.kit/bin/fab archive <args>`

The relative path `fab/.kit/bin/fab` is used instead of bare `fab` to avoid PATH dependency. Skills are run from repo root, so this relative path is always valid.

Affected files: `_preamble.md`, `_scripts.md`, `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-switch.md`, `fab-status.md`, `fab-setup.md`, `fab-help.md`, `fab-discuss.md`, `fab-archive.md`, `git-branch.md`, `git-pr.md`, `git-pr-review.md`.

#### Scenario: Skill uses direct binary invocation
- **GIVEN** `_preamble.md` has been updated
- **WHEN** the agent reads it and runs preflight
- **THEN** it invokes `fab/.kit/bin/fab preflight <change>` instead of `bash fab/.kit/scripts/lib/preflight.sh <change>`
- **AND** the output is identical

#### Scenario: Binary not available after skill switchover
- **GIVEN** skills invoke `fab/.kit/bin/fab` directly and the binary doesn't exist
- **WHEN** the agent runs a skill command
- **THEN** the Bash tool reports "No such file or directory" or "Permission denied"
- **AND** the agent can fall back to the legacy `bash fab/.kit/scripts/lib/` convention
<!-- clarified: Agents receive Bash error output and can retry with legacy shell convention; shim layer provides primary fallback -->

## Bootstrap

### Requirement: Platform-Aware Bootstrap One-Liner

The README and documentation SHALL provide a platform-aware bootstrap one-liner that downloads the platform-specific archive:

```bash
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac; mkdir -p fab; curl -sL "https://github.com/wvrdz/fab-kit/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

The old generic bootstrap (`kit.tar.gz`) SHALL remain documented as a fallback option.

#### Scenario: Bootstrap on macOS ARM64
- **GIVEN** a new project on macOS Apple Silicon
- **WHEN** the platform-aware bootstrap one-liner is run
- **THEN** `kit-darwin-arm64.tar.gz` is downloaded and extracted
- **AND** `fab/.kit/bin/fab` exists and is executable

#### Scenario: Bootstrap on unsupported platform
- **GIVEN** a new project on an unsupported platform
- **WHEN** the platform-aware bootstrap one-liner is run
- **THEN** curl returns 404 for the platform-specific archive
- **AND** the user can use the generic `kit.tar.gz` bootstrap instead

## Design Decisions

1. **Relative binary path in shims (not PATH-based)**:
   - *Why*: `${script_dir}/../../bin/fab` is deterministic and doesn't depend on `.envrc` being sourced, PATH being configured, or the user's shell setup. Works in any execution context (agent, cron, CI).
   - *Rejected*: `command -v fab` — requires PATH setup, fragile in non-interactive shells.

2. **Relative path in skill callers (`fab/.kit/bin/fab`) instead of bare `fab`**:
   - *Why*: Skills are always executed from repo root via the Bash tool. The relative path works without any PATH configuration. Avoids introducing a dependency on `.envrc`/`direnv` for skill execution.
   - *Rejected*: Adding `fab/.kit/bin` to PATH via `env-packages.sh` — adds coupling between binary location and environment setup.

3. **5 archives (4 platform + 1 generic) instead of binary-only downloads**:
   - *Why*: Single archive per platform contains everything needed — no multi-step download. Generic archive serves as fallback for unsupported platforms and pre-binary compatibility.
   - *Rejected*: Separate binary downloads alongside generic kit.tar.gz — requires two downloads for platform users, more complex upgrade logic.

4. **Parity tests as Go tests (not shell-based)**:
   - *Why*: Go's `os/exec` and `testing` package provide reliable process execution, temp directory management, and assertion capabilities. Co-located with the Go binary source.
   - *Rejected*: BATS (shell-based tests) — would work but adds another test framework; Go tests keep everything in one toolchain.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Parity test diffs stdout/stderr/exit codes/file mutations | Confirmed from intake #1 — this is the safety gate before switchover | S:90 R:85 A:90 D:95 |
| 2 | Certain | Local cross-compilation via `GOOS/GOARCH go build`, not CI | Confirmed from intake #2 — Go cross-compiles cleanly without CGo | S:85 R:85 A:85 D:90 |
| 3 | Certain | Per-platform archives named `kit-{os}-{arch}.tar.gz` | Confirmed from intake #3 — transparent platform selection for users | S:90 R:80 A:90 D:90 |
| 4 | Certain | Generic `kit.tar.gz` preserved as fallback (no binary) | Confirmed from intake #4 — graceful degradation for unsupported platforms | S:85 R:90 A:85 D:90 |
| 5 | Certain | Shim layer in shell scripts before direct caller switchover | Confirmed from intake #5 — transparent delegation, easy rollback | S:90 R:90 A:85 D:90 |
| 6 | Certain | Binary placed at `fab/.kit/bin/fab` | Upgraded from intake Confident #6 — consistent with project structure, `bin/` subdirectory is standard | S:80 R:90 A:85 D:80 |
| 7 | Certain | Shim uses relative path resolution, not PATH-based lookup | Codebase convention — scripts already resolve paths via `readlink -f` and `dirname`; no PATH dependency | S:85 R:90 A:90 D:85 |
| 8 | Certain | `CGO_ENABLED=0` for static binaries | Required for cross-compilation without platform-specific C toolchains | S:90 R:95 A:90 D:95 |
| 9 | Confident | Skill callers updated to use `fab/.kit/bin/fab` relative path | Eliminates shell startup overhead; direct binary is ~100x faster. Shim works as interim fallback | S:75 R:85 A:80 D:70 |
| 10 | Confident | Parity fixtures independent from benchmark fixtures | Different purpose (correctness vs performance); shared fixtures would create coupling | S:70 R:90 A:75 D:75 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
