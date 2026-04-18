# Spec: Pane Skip Config Check

**Change**: 260417-y0sw-pane-skip-config-check
**Created**: 2026-04-17
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Moving any `pane` subcommand implementation into the `fab-kit` binary — deferred (would require extracting a shared Go module from `src/go/fab/internal/{pane,resolve,status,statusfile}` or duplicating ~500 lines; not warranted for one command group)
- Changing the CWD-independence story of `pane` subcommands themselves — they are already CWD-independent; this change only unblocks the router
- Broadening the exemption to any other fab-go command — `runtime`, `status`, `preflight`, `resolve`, `change`, `score`, `log`, `hook`, `operator`, `batch`, `kit-path`, `fab-help` all remain config-required
- Modifying `pane send`'s idle guard behavior — exemption is router-level only; the downstream safety check on target pane agent state is untouched
- Introducing any version-selection concept other than "project-pinned when inside a fab repo, router-bundled when outside and exempt"

## fab-workflow: Router Exemption for `pane`

### Requirement: fabGoNoConfigArgs Allowlist
The `fab` router (`src/go/fab-kit/cmd/fab/main.go`) SHALL declare a new static allowlist `fabGoNoConfigArgs map[string]bool` alongside the existing `fabKitArgs` allowlist. The allowlist SHALL contain exactly one entry: `"pane": true`. No other fab-go subcommand SHALL be added to this allowlist as part of this change.

#### Scenario: Allowlist contents
- **GIVEN** the `fab` router source at `src/go/fab-kit/cmd/fab/main.go`
- **WHEN** the file is compiled
- **THEN** a package-level variable named `fabGoNoConfigArgs` of type `map[string]bool` SHALL exist
- **AND** `fabGoNoConfigArgs["pane"]` SHALL evaluate to `true`
- **AND** `fabGoNoConfigArgs["runtime"]` SHALL evaluate to `false` (zero value)
- **AND** `fabGoNoConfigArgs["status"]` SHALL evaluate to `false`
- **AND** `fabGoNoConfigArgs["preflight"]` SHALL evaluate to `false`

### Requirement: Exemption Check Placement in `execFabGo`
The `execFabGo` function in `src/go/fab-kit/cmd/fab/main.go` SHALL determine whether the invoked command is exempt (by looking up `args[0]` in `fabGoNoConfigArgs`) and select the `fab_version` to dispatch based on (a) whether `cfg` is non-nil and (b) whether the command is exempt. The exemption check SHALL be evaluated **after** `internal.ResolveConfig` returns without error and **before** any decision to exit the process with "Not in a fab-managed repo."

#### Scenario: Version selection inside a fab repo (any command)
- **GIVEN** `cfg, _ := internal.ResolveConfig()` returns a non-nil `cfg`
- **WHEN** `execFabGo` is invoked with any argument vector (including `"pane"`, `"status"`, `"preflight"`, etc.)
- **THEN** the selected `fab_version` SHALL be `cfg.FabVersion`
- **AND** `internal.EnsureCached(cfg.FabVersion)` SHALL be called to resolve the binary
- **AND** `syscall.Exec` SHALL replace the process with that binary

#### Scenario: Version selection outside a fab repo, command exempt
- **GIVEN** `cfg, _ := internal.ResolveConfig()` returns a nil `cfg` (no `fab/project/config.yaml` in any ancestor of CWD)
- **AND** `args[0]` is `"pane"`
- **WHEN** `execFabGo` is invoked
- **THEN** the selected `fab_version` SHALL be the router's build-time `version` constant (the `var version` declared near the top of `main.go`, set to the release tag via `-ldflags -X` at build time; `"dev"` for local builds)
- **AND** `internal.EnsureCached(version)` SHALL be called to resolve the binary
- **AND** `syscall.Exec` SHALL replace the process with that binary
- **AND** no "Not in a fab-managed repo" error SHALL be emitted

#### Scenario: Version selection outside a fab repo, command NOT exempt
- **GIVEN** `cfg, _ := internal.ResolveConfig()` returns a nil `cfg`
- **AND** `args[0]` is any non-exempt command (e.g., `"status"`, `"runtime"`, `"preflight"`, `"change"`)
- **WHEN** `execFabGo` is invoked
- **THEN** the process SHALL write `"Not in a fab-managed repo. Run 'fab init' to set one up.\n"` to stderr
- **AND** SHALL exit with status code 1
- **AND** no binary resolution or exec SHALL be attempted

#### Scenario: ResolveConfig returns an error (unchanged)
- **GIVEN** `internal.ResolveConfig` returns a non-nil error (e.g., `config.yaml` exists but is malformed or missing `fab_version`)
- **WHEN** `execFabGo` is invoked with any argument vector, exempt or not
- **THEN** the process SHALL write `"ERROR: {err}\n"` to stderr
- **AND** SHALL exit with status code 1
- **AND** no binary resolution or exec SHALL be attempted

### Requirement: Pure Helper for Version Resolution
To enable unit testing without invoking `syscall.Exec`, the decision of which `fab_version` to use SHALL be extracted into a pure helper function with the signature `resolveFabVersion(cfg *internal.ConfigResult, arg0 string, routerVersion string) (fabVersion string, shouldExit bool)`. The helper SHALL have no side effects (no I/O, no `os.Exit`, no `syscall.Exec`). `execFabGo` SHALL call this helper and then handle either path (exit vs. exec) based on `shouldExit`.

#### Scenario: Helper returns project version when cfg non-nil
- **GIVEN** `cfg` is a non-nil `*internal.ConfigResult` with `FabVersion = "1.3.7"`
- **WHEN** `resolveFabVersion(cfg, "pane", "dev")` is called
- **THEN** the return value SHALL be `("1.3.7", false)`

#### Scenario: Helper returns router version when cfg nil and arg exempt
- **GIVEN** `cfg` is nil
- **AND** `arg0` is `"pane"`
- **WHEN** `resolveFabVersion(nil, "pane", "1.3.7")` is called
- **THEN** the return value SHALL be `("1.3.7", false)`

#### Scenario: Helper signals exit when cfg nil and arg not exempt
- **GIVEN** `cfg` is nil
- **AND** `arg0` is `"status"`
- **WHEN** `resolveFabVersion(nil, "status", "1.3.7")` is called
- **THEN** the return value SHALL be `("", true)`

#### Scenario: Helper signals exit when cfg nil and arg empty
- **GIVEN** `cfg` is nil
- **AND** `arg0` is `""` (no subcommand passed)
- **WHEN** `resolveFabVersion(nil, "", "1.3.7")` is called
- **THEN** the return value SHALL be `("", true)`

### Requirement: `printHelp` Runs fab-go Help Even Without Config
The `printHelp` function SHALL invoke the fab-go `--help` subprocess regardless of whether `internal.ResolveConfig` returned a non-nil `cfg`, so that the workflow command list (including `pane`) is discoverable from directories outside any fab repo. When `cfg != nil`, the project-pinned `FabVersion` SHALL be used. When `cfg == nil`, the router's build-time `version` constant SHALL be used. Errors during version resolution, cache, or subprocess execution SHALL be silently swallowed (matching existing best-effort behavior).

#### Scenario: Help inside a fab repo (unchanged)
- **GIVEN** CWD is inside a fab repo with `fab_version: "1.3.7"` in `config.yaml`
- **WHEN** `fab --help` is invoked
- **THEN** stdout SHALL contain the workspace-commands section
- **AND** stdout SHALL contain a `"Workflow commands (fab-go):"` section followed by fab-go's help output
- **AND** the fab-go binary invoked for help resolution SHALL be the one for version `1.3.7`

#### Scenario: Help outside a fab repo
- **GIVEN** CWD is outside any fab repo (no `fab/project/config.yaml` in any ancestor)
- **WHEN** `fab --help` is invoked
- **THEN** stdout SHALL contain the workspace-commands section
- **AND** stdout SHALL contain a `"Workflow commands (fab-go):"` section followed by fab-go's help output
- **AND** the fab-go binary invoked for help resolution SHALL be the one matching the router's build-time `version` constant

#### Scenario: Help best-effort under failure
- **GIVEN** CWD is outside any fab repo
- **AND** `internal.EnsureCached(version)` fails (e.g., network down on a pristine machine)
- **WHEN** `fab --help` is invoked
- **THEN** stdout SHALL still contain the workspace-commands section
- **AND** the `"Workflow commands"` section SHALL be silently omitted
- **AND** the process SHALL exit with status code 0

### Requirement: Safe-Default Preservation
All fab-go commands NOT present in `fabGoNoConfigArgs` SHALL continue to emit the existing `"Not in a fab-managed repo. Run 'fab init' to set one up."` error and exit with status code 1 when invoked outside a fab repo. In particular, `runtime set-idle`, `runtime clear-idle`, `runtime is-idle`, `status`, `preflight`, `change`, `score`, `log`, `hook`, `operator`, `batch`, `kit-path`, and `fab-help` SHALL NOT be exempted in this change.

#### Scenario: runtime stays config-required
- **GIVEN** CWD is outside any fab repo
- **WHEN** `fab runtime is-idle some-change` is invoked
- **THEN** the process SHALL write `"Not in a fab-managed repo. Run 'fab init' to set one up.\n"` to stderr
- **AND** SHALL exit with status code 1

#### Scenario: Other commands stay config-required
- **GIVEN** CWD is outside any fab repo
- **WHEN** `fab status` (or any other non-exempt fab-go command) is invoked
- **THEN** the process SHALL write `"Not in a fab-managed repo. Run 'fab init' to set one up.\n"` to stderr
- **AND** SHALL exit with status code 1

### Requirement: `pane send` Idle Guard Unchanged
This change SHALL NOT modify any file under `src/go/fab/`. The existing idle-guard behavior in `pane send` (reject when target pane's agent state is not `idle`, unless `--force`) SHALL remain intact. A user invoking `fab pane send <paneID> <text>` from outside a fab repo SHALL receive the same downstream behavior they would have received from inside a fab repo targeting the same pane.

#### Scenario: pane send rejects busy pane from scratch tab
- **GIVEN** CWD is outside any fab repo
- **AND** the target pane `%7` is running a busy (non-idle) agent
- **WHEN** `fab pane send %7 "hello"` is invoked
- **THEN** the router SHALL successfully route to fab-go (no "Not in a fab-managed repo" error)
- **AND** fab-go's `pane send` implementation SHALL emit its existing non-idle error and exit 1
- **AND** no keystrokes SHALL be sent

### Requirement: Test Coverage
Unit tests SHALL be added to `src/go/fab-kit/cmd/fab/main_test.go` covering:

- Allowlist membership (`pane` in, `runtime`/`status`/`preflight` out)
- `resolveFabVersion` all four branches: (cfg non-nil, exempt), (cfg non-nil, non-exempt), (cfg nil, exempt), (cfg nil, non-exempt), plus (cfg nil, empty arg)
- `printHelp` behavior with `cfg == nil` asserting the fab-go help section appears

Tests MUST NOT depend on network access, the local `~/.fab-kit/versions/` cache being populated, or a fab-go binary existing on disk. Tests that exercise `printHelp` end-to-end SHOULD use the existing test patterns in `main_test.go` (mocking `EnsureCached` via a test seam if needed) or, if mocking is disproportionate, MAY restrict coverage to the pure helper and skip the `printHelp` integration test.

#### Scenario: Allowlist test
- **GIVEN** the test file `main_test.go`
- **WHEN** `go test ./src/go/fab-kit/cmd/fab/...` is run
- **THEN** a test named `TestFabGoNoConfigArgs` (or equivalent) SHALL pass
- **AND** it SHALL assert `fabGoNoConfigArgs["pane"] == true`
- **AND** it SHALL assert `fabGoNoConfigArgs["runtime"] == false`
- **AND** it SHALL assert `fabGoNoConfigArgs["status"] == false`
- **AND** it SHALL assert `fabGoNoConfigArgs["preflight"] == false`

#### Scenario: resolveFabVersion table test
- **GIVEN** the test file `main_test.go`
- **WHEN** `go test ./src/go/fab-kit/cmd/fab/...` is run
- **THEN** a test named `TestResolveFabVersion` (or equivalent) SHALL pass
- **AND** it SHALL cover at minimum these cases:
  - `cfg = &{FabVersion:"1.3.7"}, arg0 = "pane"` → expect `("1.3.7", false)`
  - `cfg = &{FabVersion:"1.3.7"}, arg0 = "status"` → expect `("1.3.7", false)`
  - `cfg = nil, arg0 = "pane"` → expect `("<router version>", false)`
  - `cfg = nil, arg0 = "status"` → expect `("", true)`
  - `cfg = nil, arg0 = ""` → expect `("", true)`

### Requirement: No Changes Outside Router
The implementation SHALL confine its changes to the two files `src/go/fab-kit/cmd/fab/main.go` and `src/go/fab-kit/cmd/fab/main_test.go`. No source file under `src/go/fab/` (the fab-go module) SHALL be modified. No source file under `src/go/fab-kit/internal/` SHALL be modified.

#### Scenario: Diff scope check
- **GIVEN** the working tree after implementation completes
- **WHEN** `git diff --name-only main` is run
- **THEN** the output SHALL list at most these files under `src/`:
  - `src/go/fab-kit/cmd/fab/main.go`
  - `src/go/fab-kit/cmd/fab/main_test.go`
- **AND** no file with path prefix `src/go/fab/` SHALL appear
- **AND** no file with path prefix `src/go/fab-kit/internal/` SHALL appear

## Design Decisions

1. **Router-level exemption via a sibling allowlist (`fabGoNoConfigArgs`)**: Chosen over per-command config checks in fab-go.
   - *Why*: Preserves the fail-closed default. All future fab-go commands inherit the config gate automatically — a new command added without thinking about config requirements is protected, not accidentally exempt. The allowlist is tiny, explicit, and greppable. Extends the existing `fabKitArgs` precedent.
   - *Rejected*: Pushing the config check into each fab-go command. Would require retrofitting ~12 commands, reverses the safe default (fail-open), adds regression risk (new command silently running without config), and fragments the error message across many call sites.

2. **Bundled version fallback only when `cfg == nil`**: Inside a fab repo, the project-pinned `cfg.FabVersion` SHALL be used for exempt commands too (unchanged behavior).
   - *Why*: Users who have pinned a specific fab version expect every `fab <cmd>` invocation inside the repo to use that version for reproducibility. The bundled-version fallback exists specifically to unblock the "scratch tab outside any repo" scenario. Using the project-pinned version inside the repo means no behavior change for the common case.
   - *Rejected*: Always using the bundled version for exempt commands. Would break reproducibility inside pinned repos — a user running `fab pane map` inside a repo pinned to an older fab-go would get the newer brew-installed pane behavior, which could produce a stale/new divergence.

3. **Router-level version exemption over moving `pane` into `fab-kit`**: Rejected for this change (out of scope).
   - *Why*: `fab pane` depends on `~500 lines of fab-go helpers (src/go/fab/internal/{pane, resolve, status, statusfile})`. Moving pane into fab-kit means either duplicating those helpers or extracting a new shared Go module. Not worth the refactor for one command group. Revisit only if more cross-cutting commands accumulate.
   - *Rejected*: Moving `pane` entirely into the `fab-kit` binary.

4. **Extract version-selection into a pure helper** (`resolveFabVersion`):
   - *Why*: `execFabGo` ends in `syscall.Exec`, which replaces the process and cannot be unit-tested directly. Extracting the decision logic into a pure function is standard Go testability. The same helper is reused by `printHelp`, giving a single source of truth.
   - *Rejected*: Testing via a process-level integration harness that invokes the real binary. Too heavy for the decision logic being validated; would require network access and real cache state.

5. **Silent swallowing of help-block errors remains the default**:
   - *Why*: `fab --help` must always succeed end-to-end; workflow-command discovery is best-effort. Any failure mode (cache miss, network down, binary missing) falls back to showing only the workspace-commands section. This matches existing behavior for in-repo invocations at `main.go:108-118`.
   - *Rejected*: Surfacing errors from help resolution. Would break `fab --help` on pristine installs (no cache yet) and under network failure.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exemption implemented at the router, not in each fab-go command | Confirmed from intake #1; reading `main.go:17-23` verifies the existing `fabKitArgs` precedent and confirms the extension is one-line and additive | S:95 R:85 A:90 D:95 |
| 2 | Certain | Only `pane` is exempt; `runtime` and all other commands stay config-required | Confirmed from intake #2; `runtime.go:34,55,76` in fab-go calls `resolve.FabRoot()` which is CWD-dependent and must stay gated | S:95 R:85 A:95 D:95 |
| 3 | Certain | Outside-repo exempt commands use router's build-time `version` constant via `EnsureCached` | Confirmed from intake #3; `main.go:14` declares `var version = "dev"`, set via `-ldflags -X` at brew install time | S:95 R:80 A:90 D:90 |
| 4 | Certain | Inside-repo: continue to use `cfg.FabVersion` for all commands (including exempt) | Confirmed from intake #4; preserves reproducibility for users who have pinned a specific fab version | S:95 R:85 A:95 D:95 |
| 5 | Certain | `printHelp` runs fab-go `--help` block with `cfg == nil` using bundled version | Confirmed from intake #5; required for `pane` discoverability from scratch tabs | S:95 R:90 A:90 D:95 |
| 6 | Certain | `pane send` idle guard unchanged — no changes to `src/go/fab/` | Confirmed from intake #6; idle guard is downstream of router, untouched by router-level exemption | S:95 R:85 A:95 D:95 |
| 7 | Certain | Change type is `fix` (removes usability blocker, no new capability) | Confirmed from intake #7; `.status.yaml` already set to `change_type: fix` | S:95 R:95 A:90 D:95 |
| 8 | Certain | Tests live in `src/go/fab-kit/cmd/fab/main_test.go` alongside existing tests | Confirmed from intake #8; `code-quality.md` declares test-alongside strategy; existing file already covers allowlist/version logic | S:90 R:95 A:95 D:95 |
| 9 | Certain | Version-selection logic extracted into pure helper `resolveFabVersion(cfg, arg0, routerVersion) (string, bool)` for testability | Confirmed from intake #9; helper signature chosen: `(fabVersion string, shouldExit bool)` — avoids returning an error for the "must exit" case which is a policy signal, not an error | S:90 R:95 A:90 D:90 |
| 10 | Certain | First-run download cost on pristine machines is acceptable | Confirmed from intake #10; same cost users already pay for the first in-repo workflow command | S:95 R:90 A:90 D:95 |
| 11 | Certain | Primary memory update: `docs/memory/fab-workflow/kit-architecture.md` | Confirmed from intake #11; verified the Router section (line 254-267) documents fab-go dispatch rules and the "Not in a fab-managed repo" error at line 265 | S:95 R:95 A:95 D:95 |
| 12 | Certain | Secondary memory update: `docs/memory/fab-workflow/distribution.md` | Upgraded from intake Tentative #12 — verified at spec stage: distribution.md lines 23-25 document router dispatch steps (fabKitArgs, fab-go dispatch, config walk-up), line 43 documents "Not in a fab-managed repo" scenario. Material enough to warrant a one-paragraph update | S:90 R:90 A:95 D:90 |
| 13 | Certain | Helper returns `(string, bool)` rather than `(string, error)` | The "must exit" condition is a policy decision, not an I/O or programmer error. Using `bool` avoids the antipattern of wrapping sentinel "error" values for non-error control flow | S:85 R:90 A:90 D:85 |
| 14 | Certain | `printHelp` reuses `resolveFabVersion` via a small adapter OR duplicates the 3-line `if cfg != nil / else version` branch inline | Either is acceptable; the 3-line version-pick is simple enough that DRY is not urgent. Implementation is free to choose; both options are covered by the Requirement text | S:80 R:95 A:90 D:75 |
| 15 | Certain | Non-goal: moving `pane` into `fab-kit` is deferred, not blocked | Confirmed from intake "Non-Goals" and "Why router-level exemption over the alternatives"; explicitly revisit only when more cross-cutting commands accumulate | S:95 R:90 A:90 D:95 |

15 assumptions (15 certain, 0 confident, 0 tentative, 0 unresolved).
