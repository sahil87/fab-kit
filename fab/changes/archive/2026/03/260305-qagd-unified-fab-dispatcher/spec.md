# Spec: Unified Fab Dispatcher

**Change**: 260305-qagd-unified-fab-dispatcher
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing CLI signatures — all existing `fab <command> [subcommand] [args]` invocations remain identical
- Implementing a Rust backend — only the dispatcher slot for `fab-rust` is added
- Modifying skill files — skills already call `fab/.kit/bin/fab`, which becomes the dispatcher

## Dispatcher: Entry Point

### Requirement: Single Entry Point

The file `fab/.kit/bin/fab` SHALL be a POSIX-compatible shell script that serves as the sole entry point for all fab CLI operations. It SHALL replace the current Go binary at that path.

#### Scenario: Rust backend available
- **GIVEN** `fab/.kit/bin/fab-rust` exists and is executable
- **WHEN** any fab command is invoked (e.g., `fab/.kit/bin/fab status finish qagd intake`)
- **THEN** the dispatcher SHALL `exec` the command to `fab-rust` with all arguments forwarded

#### Scenario: Go backend available (no Rust)
- **GIVEN** `fab/.kit/bin/fab-rust` does not exist or is not executable
- **AND** `fab/.kit/bin/fab-go` exists and is executable
- **WHEN** any fab command is invoked
- **THEN** the dispatcher SHALL `exec` the command to `fab-go` with all arguments forwarded

#### Scenario: Shell fallback (no compiled backends)
- **GIVEN** neither `fab-rust` nor `fab-go` is available
- **WHEN** any fab command is invoked
- **THEN** the dispatcher SHALL print `[fab] using shell backend` to stderr
- **AND** route the command to the appropriate shell script in `fab/.kit/scripts/lib/` via a `case` statement

#### Scenario: Unknown command
- **GIVEN** no compiled backend is available
- **WHEN** an unrecognized command is invoked (e.g., `fab/.kit/bin/fab frobnicate`)
- **THEN** the dispatcher SHALL print `Unknown command: frobnicate` to stderr and exit 1

### Requirement: Backend Priority Chain

The dispatcher SHALL check backends in this order: `fab-rust` → `fab-go` → shell scripts. The first available backend SHALL handle the command. The priority chain is fixed — not configurable.

#### Scenario: Both compiled backends present
- **GIVEN** both `fab-rust` and `fab-go` exist and are executable in `fab/.kit/bin/`
- **WHEN** any fab command is invoked
- **THEN** `fab-rust` SHALL be used (first match wins)
- **AND** `fab-go` SHALL NOT be invoked

### Requirement: Shell Routing Table

When falling back to shell scripts, the dispatcher SHALL route commands using this mapping:

| Command | Script |
|---------|--------|
| `resolve` | `resolve.sh` |
| `status` | `statusman.sh` |
| `log` | `logman.sh` |
| `preflight` | `preflight.sh` |
| `change` | `changeman.sh` |
| `score` | `calc-score.sh` |
| `archive` | `archiveman.sh` |

The `archive` command SHALL inject `"archive"` as the first positional argument before forwarding, because `archiveman.sh` expects `archiveman.sh archive <change>` while the CLI signature is `fab archive <change>`.

#### Scenario: Archive command routing
- **GIVEN** shell fallback is active
- **WHEN** `fab archive mychange --description "done"` is invoked
- **THEN** the dispatcher SHALL execute `bash "$LIB_DIR/archiveman.sh" "archive" "mychange" "--description" "done"`

### Requirement: Version Handling

The dispatcher SHALL handle `--version` directly, without delegating to any backend.

It SHALL read the version from `fab/.kit/VERSION` and determine which backend would be used (by running the same priority check), then output: `fab {version} ({backend} backend)` where `{backend}` is `rust`, `go`, or `shell`.

#### Scenario: Version with Go backend
- **GIVEN** `fab-go` exists and is executable, `fab-rust` does not exist
- **AND** `fab/.kit/VERSION` contains `0.32.1`
- **WHEN** `fab --version` is invoked
- **THEN** stdout SHALL print `fab 0.32.1 (go backend)`

#### Scenario: Version with shell fallback
- **GIVEN** neither `fab-rust` nor `fab-go` exists
- **AND** `fab/.kit/VERSION` contains `0.32.1`
- **WHEN** `fab --version` is invoked
- **THEN** stdout SHALL print `fab 0.32.1 (shell backend)`
- **AND** the shell fallback diagnostic SHALL NOT be printed (version is handled before routing)

### Requirement: Shell Fallback Diagnostic

When the dispatcher falls through to shell scripts (neither compiled backend available), it SHALL print `[fab] using shell backend` to stderr before routing the command. This diagnostic SHALL NOT be printed for `--version` handling.

#### Scenario: Diagnostic on fallback
- **GIVEN** no compiled backend is available
- **WHEN** `fab status finish qagd intake` is invoked
- **THEN** stderr SHALL contain `[fab] using shell backend`
- **AND** the command SHALL execute normally via `statusman.sh`

#### Scenario: No diagnostic for compiled backends
- **GIVEN** `fab-go` is available
- **WHEN** any fab command is invoked
- **THEN** stderr SHALL NOT contain `[fab] using shell backend`

## Binary Rename

### Requirement: Go Binary Path

The Go binary SHALL be built to `fab/.kit/bin/fab-go` instead of `fab/.kit/bin/fab`.

#### Scenario: Build output path
- **GIVEN** the `justfile` `build-go` target is invoked
- **WHEN** compilation completes
- **THEN** the binary SHALL be written to `fab/.kit/bin/fab-go`

### Requirement: Parity Test Binary Path

The parity test helper `fabBinary()` in `src/go/fab/test/parity/parity_test.go` SHALL reference `fab/.kit/bin/fab-go` instead of `fab/.kit/bin/fab`.

#### Scenario: Parity tests find binary
- **GIVEN** `fab-go` has been built at `fab/.kit/bin/fab-go`
- **WHEN** parity tests run
- **THEN** `fabBinary()` SHALL return the path to `fab-go`

### Requirement: Gitignore Update

`.gitignore` SHALL replace `fab/.kit/bin/fab` with entries for the compiled backends:
- `fab/.kit/bin/fab-go`
- `fab/.kit/bin/fab-rust`

The dispatcher script `fab/.kit/bin/fab` SHALL NOT be gitignored (it is a tracked shell script).

#### Scenario: Dispatcher is tracked
- **GIVEN** the new `.gitignore` entries are in place
- **WHEN** `git status` is run
- **THEN** `fab/.kit/bin/fab` (the dispatcher script) SHALL appear as a tracked file
- **AND** `fab/.kit/bin/fab-go` and `fab/.kit/bin/fab-rust` SHALL be ignored

## Shim Removal

### Requirement: Remove Delegation Shims

The `_fab_bin` shim block SHALL be removed from all 7 shell scripts in `fab/.kit/scripts/lib/`:

- `resolve.sh` — remove lines 13-17 (shim + blank line)
- `statusman.sh` — remove lines 16-20
- `logman.sh` — remove lines 16-20
- `preflight.sh` — remove lines 4-8
- `changeman.sh` — remove lines 17-21
- `calc-score.sh` — remove lines 4-8
- `archiveman.sh` — remove lines 16-24

After removal, these scripts SHALL be pure shell implementations with no binary delegation logic. They SHALL remain independently executable (shebang, `set -euo pipefail`, own path resolution).

#### Scenario: Script works standalone after shim removal
- **GIVEN** shims have been removed from `resolve.sh`
- **AND** no compiled binary exists
- **WHEN** `bash fab/.kit/scripts/lib/resolve.sh --folder qagd` is invoked directly
- **THEN** it SHALL produce the same output as before shim removal

#### Scenario: archiveman.sh loses reverse-translation
- **GIVEN** `archiveman.sh` currently strips leading `"archive"` before forwarding to the Go binary (lines 19-22)
- **WHEN** shims are removed
- **THEN** `archiveman.sh` SHALL retain its original interface (`archiveman.sh archive <change>`)
- **AND** the dispatcher handles the `archive` → `archiveman.sh archive` translation

## Batch Script Updates

### Requirement: Use Dispatcher for Change Resolution

`batch-fab-switch-change.sh` and `batch-fab-archive-change.sh` SHALL use `fab/.kit/bin/fab change resolve` instead of calling `changeman.sh` directly via the `$CHANGEMAN` variable.

#### Scenario: batch-fab-switch-change resolves via dispatcher
- **GIVEN** the batch script needs to resolve a change name
- **WHEN** it calls the resolution function
- **THEN** it SHALL use `"$KIT_DIR/bin/fab" change resolve "$change"` instead of `"$CHANGEMAN" resolve "$change"`
- **AND** the `CHANGEMAN` variable SHALL be removed

#### Scenario: batch-fab-archive-change resolves via dispatcher
- **GIVEN** the archive batch script needs to resolve a change name
- **WHEN** it calls the resolution function
- **THEN** it SHALL use `"$KIT_DIR/bin/fab" change resolve "$change"` instead of `"$CHANGEMAN" resolve "$change"`
- **AND** the `CHANGEMAN` variable SHALL be removed

## Documentation Updates

### Requirement: Update `_scripts.md`

`fab/.kit/skills/_scripts.md` SHALL be updated to reflect the new architecture:

- **Primary invocation**: `fab/.kit/bin/fab` is a shell dispatcher (always shipped, always tracked)
- **Backend resolution**: dispatcher checks `fab-rust` → `fab-go` → shell scripts
- **Shell scripts**: pure implementations in `scripts/lib/` (no shims, no delegation)
- **Remove**: "Legacy (shell scripts — still works via shim delegation)" framing and the "Shell script" calling convention example
- **Retain**: command mapping table, `<change>` argument convention, all subcommand documentation

#### Scenario: Skills calling convention unchanged
- **GIVEN** a skill calls `fab/.kit/bin/fab status finish qagd intake`
- **WHEN** the dispatcher is in place
- **THEN** the command SHALL work identically to before
- **AND** no skill files need modification

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | CLI signature unchanged | Confirmed from intake #1 — same `fab <command> [subcommand] [args]` interface | S:95 R:90 A:95 D:95 |
| 2 | Certain | Priority: rust > go > shell | Confirmed from intake #2 — user explicitly chose this order | S:95 R:85 A:90 D:95 |
| 3 | Certain | Shell fallback must be complete | Confirmed from intake #3 — user explicitly said "don't assume Go binary always available" | S:95 R:80 A:90 D:95 |
| 4 | Certain | Rename Go binary to fab-go | Confirmed from intake #4 — user confirmed this naming | S:90 R:85 A:90 D:90 |
| 5 | Certain | Archive command needs arg injection | Confirmed from intake #5 — archiveman.sh expects `archive` as first positional | S:95 R:90 A:85 D:80 |
| 6 | Certain | Diagnostic output on shell fallback | Confirmed from intake #6 — user changed to "print diagnostic when falling back" | S:95 R:90 A:70 D:65 |
| 7 | Certain | Batch scripts use dispatcher, not direct changeman.sh | Confirmed from intake #7 — user confirmed | S:95 R:85 A:80 D:75 |
| 8 | Certain | Parity tests need path update | Confirmed from intake #8 — `fabBinary()` references `fab/.kit/bin/fab` line 41 | S:95 R:90 A:80 D:80 |
| 9 | Certain | Version handled by dispatcher, not backend | Clarified in intake — user chose dispatcher handles `--version` | S:95 R:90 A:90 D:90 |
| 10 | Certain | No diagnostic printed for --version | Inferred — version handling occurs before backend resolution, consistent with UX | S:85 R:95 A:90 D:85 |
| 11 | Certain | POSIX-compatible dispatcher script | Constitution requires "shell scripts (Bash)" but dispatcher should be minimal — POSIX sh compatible for portability | S:80 R:90 A:85 D:80 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
