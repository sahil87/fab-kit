# Spec: Rust Binary Port

**Change**: 260307-bmp3-3-rust-binary-port
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- CI/release pipeline changes (cross-compilation, archive packaging) — deferred to a follow-up change
- Removing or replacing the Go binary — both binaries coexist
- Changing any existing CLI behavior or output format — strict parity with Go

## Rust Binary: Project Structure

### Requirement: Crate Layout

The Rust binary SHALL be a single binary crate at `src/fab-rust/` with `clap` derive for CLI parsing, `serde` + `serde_yaml` for YAML handling, and `anyhow` for error handling. The project SHALL use a flat module structure: one file per subcommand plus shared utility modules.

#### Scenario: Fresh build from source

- **GIVEN** the `src/fab-rust/` directory exists with `Cargo.toml` and `src/`
- **WHEN** `cargo build --manifest-path src/fab-rust/Cargo.toml --release` is run
- **THEN** a binary is produced at `target/release/fab-rust`
- **AND** the binary size with `strip` and `lto` is significantly smaller than the Go equivalent

#### Scenario: Module structure

- **GIVEN** the Rust source at `src/fab-rust/src/`
- **WHEN** inspecting the file layout
- **THEN** there is one module file per subcommand (`resolve.rs`, `log.rs`, `status.rs`, `preflight.rs`, `change.rs`, `score.rs`, `runtime.rs`, `panemap.rs`, `sendkeys.rs`)
- **AND** shared utilities in `common/` or top-level modules (`types.rs` for shared types like `StatusFile`, `resolve_util.rs` for change resolution)
- **AND** `main.rs` defines the clap CLI structure and dispatches to subcommand modules

### Requirement: Cargo.toml Configuration

The `Cargo.toml` SHALL specify `lto = true` and `strip = true` in the release profile. `Cargo.lock` SHALL be committed (binary crate convention).

#### Scenario: Release profile optimization

- **GIVEN** the `Cargo.toml` at `src/fab-rust/Cargo.toml`
- **WHEN** building in release mode
- **THEN** the binary uses link-time optimization and is stripped of debug symbols

## Rust Binary: CLI Parity

### Requirement: Subcommand Parity

The Rust binary SHALL implement all 9 top-level subcommands with identical CLI signatures (flags, positional arguments, subcommands) to the Go binary.

| Subcommand | Subcommands | Key Flags |
|------------|-------------|-----------|
| `resolve` | — | `--id`, `--folder`, `--dir`, `--status` |
| `log` | `command`, `confidence`, `review`, `transition` | — |
| `status` | `show`, `finish`, `start`, `advance`, `reset`, `skip`, `fail`, `set-change-type`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`, `add-issue`, `get-issues`, `add-pr`, `get-prs`, `progress-map`, `progress-line`, `current-stage`, `display-stage`, `all-stages`, `checklist`, `confidence`, `validate-status-file` | `show`: `--all`, `--json`; `set-confidence`: `--indicative`; `set-confidence-fuzzy`: `--indicative` |
<!-- clarified: validate subcommand is actually validate-status-file in Go source; set-confidence and set-confidence-fuzzy both have --indicative flag -->
| `preflight` | — | positional: `[change-name]` |
| `change` | `new`, `rename`, `switch`, `list`, `resolve`, `archive`, `restore`, `archive-list` | `new`: `--slug`, `--change-id`, `--log-args`; `rename`: `--folder`, `--slug`; `list`: `--archive`; `archive`: `--description`; `restore`: `--switch`; `switch`: `--blank` |
| `score` | — | `--check-gate`, `--stage` |
| `runtime` | `set-idle`, `clear-idle` | — |
| `pane-map` | — | — |
| `send-keys` | — | positional: `<change>`, `<text>` |

#### Scenario: Help output equivalence

- **GIVEN** the Rust binary is built
- **WHEN** running `fab-rust --help` or `fab-rust <subcommand> --help`
- **THEN** the help output lists the same subcommands and flags as the Go binary
- **AND** clap's auto-generated help is acceptable (exact text parity not required)

#### Scenario: Unknown subcommand

- **GIVEN** the Rust binary is invoked with an unknown subcommand
- **WHEN** running `fab-rust nonexistent`
- **THEN** it exits with code 2 (clap default) and prints an error to stderr

### Requirement: Output Parity

For every subcommand, the Rust binary SHALL produce identical stdout, stderr, and exit codes as the Go binary when given the same inputs and file system state. Exceptions:

- Timestamp fields (`last_updated`, `started_at`, `completed_at`, `ts`) MAY differ (they are execution-time dependent)
- Help text format MAY differ (clap vs cobra style)
- The `indicative: false` boolean field MAY be omitted when false (Go behavior)

#### Scenario: Resolve output parity

- **GIVEN** a temp repo with an active change via `.fab-status.yaml` symlink
- **WHEN** running both `fab-go resolve --folder` and `fab-rust resolve --folder`
- **THEN** stdout is identical

#### Scenario: Preflight output parity

- **GIVEN** a temp repo with valid config, constitution, and active change
- **WHEN** running both `fab-go preflight` and `fab-rust preflight`
- **THEN** the YAML output is semantically identical (after normalizing timestamps and optional false booleans)

#### Scenario: Status transition parity

- **GIVEN** a temp repo with a change at intake stage (active)
- **WHEN** running `fab-rust status finish <change> intake` then `fab-rust status finish <change> spec`
- **THEN** the `.status.yaml` file matches what `fab-go` would produce (semantically identical YAML)
- **AND** auto-activation of next stages works identically

#### Scenario: Score computation parity

- **GIVEN** a change with an `intake.md` containing an Assumptions table
- **WHEN** running `fab-rust score --stage intake <change>`
- **THEN** the computed confidence score, grade counts, and dimension means match Go output

### Requirement: Change Resolution Parity

Change resolution SHALL follow the same algorithm: (1) if override provided — exact match, then case-insensitive substring match against `fab/changes/` folders; (2) if no override — read `.fab-status.yaml` symlink, fallback to single-change guess. Errors for 0 or >1 matches SHALL use identical message text.

#### Scenario: Substring resolution

- **GIVEN** a temp repo with change folder `260305-t3st-parity-test-change`
- **WHEN** running `fab-rust resolve --folder parity`
- **THEN** stdout is `260305-t3st-parity-test-change`

#### Scenario: Ambiguous resolution

- **GIVEN** a temp repo with two change folders both containing "test"
- **WHEN** running `fab-rust resolve --folder test`
- **THEN** exit code is 1 and stderr contains "Multiple changes match"

### Requirement: State Machine Parity

The stage state machine SHALL implement identical transitions, cascading rules, and side effects:

- **Transitions**: start (pending/failed→active), advance (active→ready), finish (active/ready→done + auto-activate next), reset (done/ready/skipped→active + cascade downstream to pending), skip (pending/active→skipped + cascade downstream), fail (active→failed, review/review-pr only)
- **Allowed states per stage**: Same whitelist as Go (e.g., intake has no `pending`, ship has no `ready`)
- **Metrics**: iterations, started_at, completed_at, driver tracking per stage
- **Hooks**: Pre-hook on start (blocks if fails), post-hook on finish (fails stage if fails)
- **Auto-logging**: Transitions logged to `.history.jsonl`; review finish/fail auto-logged

#### Scenario: Cascading reset

- **GIVEN** a change with stages intake=done, spec=done, tasks=done, apply=active
- **WHEN** running `fab-rust status reset <change> spec`
- **THEN** spec becomes active, tasks/apply/review/hydrate/ship/review-pr become pending

#### Scenario: Stage hooks

- **GIVEN** a `config.yaml` with `stage_hooks: { apply: { pre: "echo pre", post: "echo post" } }`
- **WHEN** running `fab-rust status start <change> apply` then `fab-rust status finish <change> apply`
- **THEN** the pre-hook runs before start (blocking on failure) and post-hook runs after finish

### Requirement: YAML Round-Trip Preservation

When reading and writing `.status.yaml`, the Rust binary SHALL preserve YAML formatting, comments, and field order to the extent possible. Atomic writes SHALL use temp file + rename pattern.

#### Scenario: Atomic write safety

- **GIVEN** a valid `.status.yaml` file
- **WHEN** a status transition is applied
- **THEN** the write uses a temp file in the same directory followed by rename
- **AND** if the process is interrupted mid-write, the original file remains intact

### Requirement: JSONL Logging Parity

The log subcommand SHALL produce identical JSONL entries to the Go binary: same field names, same field ordering within entries, same ISO 8601 timestamp format. Optional fields SHALL be omitted (not null) when empty.

#### Scenario: Command log entry

- **GIVEN** an active change
- **WHEN** running `fab-rust log command "fab-continue" <change>`
- **THEN** a JSONL line is appended with fields: `ts`, `event`, `cmd`, `change`, and optionally `args`

### Requirement: Archive/Restore Parity

The change archive and restore subcommands SHALL produce identical directory structures, index.md updates, and YAML output to the Go binary. Archive uses nested `YYYY/MM/` directory structure. Index uses `- **{folder}** — {description}` format. Backfill of unindexed entries SHALL work identically.

#### Scenario: Archive with index update

- **GIVEN** a change with description "Test change"
- **WHEN** running `fab-rust change archive <change> --description "Test change"`
- **THEN** the change directory is moved to `fab/changes/archive/YYYY/MM/{name}/`
- **AND** `fab/changes/archive/index.md` contains `- **{name}** — Test change`
- **AND** `.fab-status.yaml` symlink is removed if it pointed to this change

### Requirement: Tmux Integration Parity

The `pane-map` and `send-keys` subcommands SHALL produce identical output and behavior to the Go binary. `pane-map` discovers panes via `tmux list-panes -a`, resolves each pane's git worktree root via `git rev-parse --show-toplevel`, uses `git worktree list --porcelain` to find the main worktree root for relative path display, resolves fab state, and outputs an aligned table. `send-keys` resolves a change to its tmux pane and sends text via `tmux send-keys`.
<!-- clarified: pane-map discovery uses tmux list-panes first, then git rev-parse, not git worktree list --porcelain for primary discovery -->

#### Scenario: Pane map outside tmux

- **GIVEN** the `$TMUX` environment variable is not set
- **WHEN** running `fab-rust pane-map`
- **THEN** stderr contains "not inside a tmux session" and exit code is 1

## Dispatcher: Backend Override

### Requirement: Override Mechanism

The `fab/.kit/bin/fab` dispatcher SHALL support a backend override mechanism. Priority order: (1) `FAB_BACKEND` environment variable, (2) `.fab-backend` file at repo root, (3) default priority (rust > go).

#### Scenario: Environment variable override to Go

- **GIVEN** both `fab-rust` and `fab-go` exist in `fab/.kit/bin/`
- **WHEN** running `FAB_BACKEND=go fab resolve`
- **THEN** the Go binary is invoked (not Rust)

#### Scenario: File-based override to Go

- **GIVEN** both binaries exist and `.fab-backend` at repo root contains `go`
- **WHEN** running `fab resolve` (no env var)
- **THEN** the Go binary is invoked

#### Scenario: Env var takes precedence over file

- **GIVEN** `.fab-backend` contains `go` and `FAB_BACKEND=rust` is set
- **WHEN** running `fab resolve`
- **THEN** the Rust binary is invoked (env var wins)

#### Scenario: Invalid override value

- **GIVEN** `FAB_BACKEND=python` is set
- **WHEN** running `fab resolve`
- **THEN** the default priority chain is used (override is ignored for unrecognized values)

#### Scenario: Override to unavailable backend

- **GIVEN** `FAB_BACKEND=rust` but `fab-rust` does not exist
- **WHEN** running `fab resolve`
- **THEN** the default priority chain is used (falls through to `fab-go` if available)

### Requirement: .fab-backend File Location

The `.fab-backend` file SHALL live at the repo root (two levels up from `fab/.kit/bin/`). It contains a single word: `go` or `rust`, with optional whitespace trimmed. The file SHALL be gitignored.

#### Scenario: File with trailing newline

- **GIVEN** `.fab-backend` contains `go\n`
- **WHEN** the dispatcher reads it
- **THEN** whitespace is trimmed and `go` is used as the override value

## Build System

### Requirement: Justfile Rust Recipe

The `justfile` SHALL include a `build-rust` recipe that compiles the Rust binary for the current platform and copies it to `fab/.kit/bin/fab-rust`.

#### Scenario: Local Rust build

- **GIVEN** Rust toolchain is installed and `src/fab-rust/Cargo.toml` exists
- **WHEN** running `just build-rust`
- **THEN** `fab/.kit/bin/fab-rust` exists and is executable
- **AND** the binary was built with release profile (lto + strip)

### Requirement: Test Recipe

The `justfile` SHALL include a `test-rust` recipe that runs the Rust integration tests via `cargo test`.

#### Scenario: Running Rust tests

- **GIVEN** the Rust project at `src/fab-rust/`
- **WHEN** running `just test-rust`
- **THEN** Rust integration tests execute and report results

## Testing

### Requirement: Rust Integration Tests

The Rust binary SHALL have integration tests at `src/fab-rust/tests/` that validate all subcommand behaviors against expected outputs. Tests SHALL use the same fixture-based approach as Go parity tests: temp directory with copied fixtures simulating a repo root.

#### Scenario: Test setup

- **GIVEN** test fixtures exist (config.yaml, constitution.md, .status.yaml, etc.)
- **WHEN** a test runs
- **THEN** it creates a temp directory, copies fixtures, and runs the Rust binary against it
- **AND** validates stdout, stderr, exit code, and file mutations

#### Scenario: Comprehensive coverage

- **GIVEN** the Rust integration test suite
- **WHEN** all tests pass
- **THEN** every subcommand has at least basic happy-path coverage
- **AND** critical behaviors (state transitions, cascading, resolution) have edge-case coverage

### Requirement: Go Tests Unmodified

The existing Go parity tests at `src/fab-go/test/parity/` SHALL NOT be modified by this change. They continue to test the Go binary independently.

#### Scenario: Go tests still pass

- **GIVEN** no Go source files are modified
- **WHEN** running `just test-go`
- **THEN** all existing Go parity tests pass

## Git Configuration

### Requirement: Gitignore Update

`.fab-backend` SHALL be added to `.gitignore` at the repo root — it is a local developer preference file, not committed.

#### Scenario: Backend file not tracked

- **GIVEN** `.fab-backend` is listed in `.gitignore`
- **WHEN** running `echo "go" > .fab-backend && git status`
- **THEN** `.fab-backend` does not appear as an untracked file

## Design Decisions

1. **Big bang port over incremental**: All 9 subcommands ported at once because cobra/clap each want to own the full command tree — incremental per-subcommand delegation is more complex than porting everything.
   - *Why*: Go codebase is small (~38 source files, ~2000 lines), making a complete port feasible
   - *Rejected*: Incremental port with per-subcommand delegation — adds dispatcher complexity that exceeds the port complexity

2. **Backend override via env var + file**: Provides a way to switch back to Go for comparison during the transition period without modifying the dispatcher or rebuilding.
   - *Why*: Low-friction comparison; env var for per-command, file for persistent
   - *Rejected*: CLI flag (would require dispatcher to parse args before delegating)

3. **Both test suites run independently**: Go parity tests and Rust integration tests coexist rather than sharing a harness.
   - *Why*: Simpler — each test suite is self-contained, no cross-language test infrastructure
   - *Rejected*: Shared test harness — adds complexity for questionable benefit; symlink trick is fragile

4. **CI/release deferred**: This change focuses on the port and local dev build. CI cross-compilation and archive packaging for Rust is a separate follow-up.
   - *Why*: Keeps change scope manageable; Rust cross-compilation is significantly more complex than Go's `GOOS/GOARCH`
   - *Rejected*: Including CI in this change — too much scope, different concerns

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Big bang port — all 9 subcommands at once | Confirmed from intake #1 — incremental doesn't work with cobra/clap command tree ownership | S:95 R:70 A:90 D:95 |
| 2 | Certain | Use `clap` derive for CLI parsing | Confirmed from intake #2 — auto-help, completions, man pages | S:90 R:85 A:90 D:90 |
| 3 | Certain | Backend override via FAB_BACKEND env var + .fab-backend file | Confirmed from intake #3 | S:85 R:90 A:85 D:80 |
| 4 | Certain | Dispatcher priority remains rust > go by default | Confirmed from intake #4 — already in production | S:95 R:85 A:95 D:95 |
| 5 | Certain | Use `serde_yaml` for YAML handling | Confirmed from intake #5 | S:95 R:80 A:85 D:75 |
| 6 | Certain | Use `anyhow` for error handling | Confirmed from intake #6 | S:95 R:85 A:80 D:80 |
| 7 | Certain | Flat module structure (one file per subcommand + shared modules) | Confirmed from intake #7 | S:95 R:90 A:80 D:75 |
| 8 | Certain | `.fab-backend` file at repo root, gitignored | Confirmed from intake #8 | S:95 R:90 A:80 D:75 |
| 9 | Certain | Both test suites run — Go parity tests stay, Rust gets own integration tests | Confirmed from intake #9 | S:95 R:75 A:60 D:50 |
| 10 | Certain | CI/release deferred — this change is port + local dev build only | Confirmed from intake #10 | S:95 R:90 A:85 D:90 |
| 11 | Certain | Strict output parity with Go binary (same stdout/stderr/exit codes) | Codebase shows consumers parse stdout directly; any deviation breaks skills | S:90 R:40 A:90 D:90 |
| 12 | Certain | Atomic file writes via temp+rename pattern | Matches Go implementation pattern; prevents corruption on interruption | S:90 R:60 A:95 D:95 |
| 13 | Certain | YAML round-trip preservation for .status.yaml | Go uses yaml.Node for this; Rust needs equivalent (serde_yaml Value or similar) | S:85 R:50 A:80 D:80 |
| 14 | Certain | Invalid/unavailable backend override falls through to default priority | Most forgiving behavior — prevents lockout if .fab-backend has a typo | S:80 R:90 A:85 D:85 |
| 15 | Certain | Cargo.lock committed, release profile with lto+strip | Binary crate convention; produces smallest possible binary | S:90 R:95 A:90 D:90 |

15 assumptions (15 certain, 0 confident, 0 tentative, 0 unresolved).
