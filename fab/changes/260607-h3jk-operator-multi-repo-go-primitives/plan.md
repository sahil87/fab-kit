# Plan: Operator Multi-Repo Go Primitives

**Change**: 260607-h3jk-operator-multi-repo-go-primitives
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Operator State: Server-Keyed XDG State File (A1)

#### R1: XDG state base directory resolution
The binary SHALL resolve the XDG state base directory uniformly on Linux and macOS via a `stateDir()` helper in `src/go/fab/cmd/fab/operator.go`. It SHALL honor `$XDG_STATE_HOME` only when set AND absolute; otherwise it SHALL fall back to `$HOME/.local/state`. It MUST NOT use `~/Library/...` on macOS.

- **GIVEN** `XDG_STATE_HOME` is set to an absolute path
- **WHEN** `stateDir()` is called
- **THEN** it returns that path
- **AND GIVEN** `XDG_STATE_HOME` is unset or set to a relative path
- **WHEN** `stateDir()` is called
- **THEN** it returns `$HOME/.local/state`

#### R2: Socket-keyed state file path
The binary SHALL derive a filesystem-safe, deterministic, collision-free slug from the tmux socket path via `serverSlug(server string)` (querying `#{socket_path}` through `pane.WithServer`), falling back to `"default"` when tmux cannot be queried. `StatePath(server string)` SHALL return `<stateDir>/fab/operator/<server-slug>.yaml`, creating the parent directory with `MkdirAll` (0o755).

- **GIVEN** tmux reports socket path `/tmp/tmux-1000/default`
- **WHEN** `StatePath("")` is called
- **THEN** it returns `<stateDir>/fab/operator/<slug>.yaml` where `<slug>` is a deterministic filesystem-safe slug of `/tmp/tmux-1000/default` and the parent dir exists
- **AND GIVEN** distinct socket paths
- **WHEN** slugified
- **THEN** the slugs are distinct (collision-free)
- **AND GIVEN** tmux cannot be queried
- **WHEN** `serverSlug` runs
- **THEN** the slug is `"default"`

#### R3: tick-start writes to the server-keyed state path
`runOperatorTickStart` (`src/go/fab/cmd/fab/operator_tick_start.go`) SHALL write tick state to `StatePath(server)` instead of `gitRepoRoot()/.fab-operator.yaml`. The test seam `operatorRepoRootOverride` SHALL be renamed to `operatorStatePathOverride` and SHALL hold a full file path (not a directory). The existing tick-increment / last-tick-at / field-preservation behavior SHALL be unchanged. No migration of old repo-rooted files SHALL be performed.

- **GIVEN** a server-keyed state file with `tick_count: 5` and a `monitored` field
- **WHEN** `fab operator tick-start` runs
- **THEN** `tick_count` becomes 6, `last_tick_at` is written (RFC3339 UTC), `monitored` is preserved, and stdout is `tick: 6\nnow: HH:MM`
- **AND GIVEN** the state file does not exist
- **WHEN** `fab operator tick-start` runs
- **THEN** it is created with `tick_count: 1`

#### R4: runOperator launch CWD unchanged
`runOperator` (`src/go/fab/cmd/fab/operator.go`) SHALL continue to use `gitRepoRoot()` as the new-window launch CWD. Only the *state path* decouples from the repo.

- **GIVEN** the operator is launched
- **WHEN** `runOperator` creates the new window
- **THEN** the working directory is still `gitRepoRoot()`

### Pane Map: Per-Repo mainRoot + `repo` Field (A2)

#### R5: Per-repo mainRoot and `repo` JSON field
`runPaneMap` (`src/go/fab/cmd/fab/panemap.go`) SHALL compute `mainRoot` **per distinct repo**, cached by the pane's `GitWorktreeRoot`, so each pane's worktree display path is computed relative to its OWN repo's main root. A `repo` field (absolute main-worktree root) SHALL be added to `paneRow` and exposed as `repo` in `fab pane map --json` (`paneJSON`) ONLY — no human-table column SHALL be added.

- **GIVEN** panes from two distinct repos
- **WHEN** `fab pane map` resolves display paths
- **THEN** each pane's worktree display path is relative to its own repo's main root (not the first repo's)
- **AND GIVEN** `fab pane map --json`
- **WHEN** the JSON is emitted
- **THEN** each element carries a `repo` field (absolute main-worktree root, nullable when unresolved) and the human table has no `Repo` column

### Spawn Command Helper (A4)

#### R6: `fab spawn-command [--repo <path>]`
A new `spawnCommandCmd()` cobra command SHALL be wired into the root command. `fab spawn-command --repo <path>` SHALL read `agent.spawn_command` from `<path>/fab/project/config.yaml` via the existing `spawn.Command(configPath)` and print it to stdout. When `--repo` is omitted, it SHALL default to the current repo, resolving the config via `resolve.FabRoot()` (consistent with `runOperator`).

- **GIVEN** `--repo <path>` pointing at a repo whose config sets `agent.spawn_command`
- **WHEN** `fab spawn-command --repo <path>` runs
- **THEN** the configured spawn command is printed to stdout
- **AND GIVEN** `--repo` is omitted inside a fab repo
- **WHEN** `fab spawn-command` runs
- **THEN** the current repo's spawn command (via `resolve.FabRoot()`) is printed
- **AND GIVEN** a repo with no `agent.spawn_command`
- **WHEN** queried
- **THEN** `spawn.DefaultSpawnCommand` is printed

### Documentation (Constitution-mandated)

#### R7: `_cli-fab.md` reflects new/changed signatures
`src/kit/skills/_cli-fab.md` SHALL document (a) the new `fab spawn-command [--repo <path>]` command and (b) the `fab operator tick-start` state-path change (now server-keyed XDG `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`, not repo-rooted), and add `repo` to the `fab pane map --json` field list.

- **GIVEN** the constitution requires CLI changes to update `_cli-fab.md`
- **WHEN** this change ships
- **THEN** `_cli-fab.md` documents `fab spawn-command`, the new tick-start state path, and the `repo` JSON field

### Non-Goals

- No skill behavior changes (that is change 2, `260607-oy0k-operator-multi-repo-skill`).
- No migration of existing repo-rooted `.fab-operator.yaml` files — abandoned in place.
- No human-table `Repo` column in `fab pane map`.
- No new `internal/operator` package — helpers stay in `cmd/fab/operator.go`.

### Design Decisions

1. **Helper placement**: A1 helpers (`stateDir`/`serverSlug`/`StatePath`/`slugify`) live in `cmd/fab/operator.go` — *Why*: minimal diff, alongside `gitRepoRoot` — *Rejected*: a new `internal/operator` package (intake clarification #10).
2. **`repo` visibility**: exposed in `fab pane map --json` only — *Why*: the change-2 skill consumes JSON; the per-repo `mainRoot` fix already corrects the human Worktree column — *Rejected*: a human-table `Repo` column (intake clarification #11).
3. **Socket-path key**: state file keyed by tmux socket path, not server PID — *Why*: socket path survives a server restart (same `-L` → same path); PID would orphan the file — *Rejected*: PID key, fixed global path (machine-wide singleton), repo-rooting (loses cross-repo state).
4. **Slugify rule**: replace path separators (`/`) with `-` and strip the leading separator, producing a deterministic, collision-free, filesystem-safe slug — *Why*: `filepath.Rel`-free, trivially reversible-distinct for distinct inputs — *Rejected*: hashing (opaque, harder to debug).

## Tasks

### Phase 1: A1 — Server-Keyed State File (foundational)

- [x] T001 Add `stateDir()`, `slugify(string) string`, `serverSlug(server string) string`, and `StatePath(server string) (string, error)` helpers to `src/go/fab/cmd/fab/operator.go` (import `pane` for `WithServer`). <!-- R1 R2 -->
- [x] T002 Rewire `runOperatorTickStart` in `src/go/fab/cmd/fab/operator_tick_start.go` to resolve the state file via `StatePath(server)`; rename the seam `operatorRepoRootOverride` → `operatorStatePathOverride` (now a full file path); update the command's `Short` text referencing `.fab-operator.yaml` to the new server-keyed path. <!-- R3 -->
- [x] T003 Add unit tests in `src/go/fab/cmd/fab/operator_test.go`: `stateDir` table test (env set-absolute / unset / relative-ignored), `serverSlug`/`slugify` determinism + collision-free + filesystem-safe test, and update the two existing tick-start tests to use `operatorStatePathOverride` (full file path). <!-- R1 R2 R3 -->

### Phase 2: A2 — Per-Repo mainRoot + `repo` Field (bug fix)

- [x] T004 In `src/go/fab/cmd/fab/panemap.go`: add a `repo` field to `paneRow`; compute `mainRoot` per distinct repo (cached by the pane's `GitWorktreeRoot`) inside `runPaneMap` and pass each pane its own repo's main root to `resolvePane`; set `paneRow.repo` to that absolute main-worktree root. <!-- R5 -->
- [x] T005 In `src/go/fab/cmd/fab/panemap.go`: add a nullable `Repo *string` `json:"repo"` field to `paneJSON` and populate it in `printPaneJSON` (via `toNullable`); leave `printPaneTable` unchanged (no `Repo` column). <!-- R5 -->
- [x] T006 [P] Add/extend tests in `src/go/fab/cmd/fab/panemap_test.go`: a per-repo mainRoot resolution test (two distinct repos → display paths relative to their own roots) and a `printPaneJSON` test asserting the `repo` field is present + snake_case and null when unresolved. <!-- R5 -->

### Phase 3: A4 — `fab spawn-command` Command (new command)

- [x] T007 Add `src/go/fab/cmd/fab/spawn_command.go` with `spawnCommandCmd()` — a `--repo` string flag; resolve config path from `<repo>/fab/project/config.yaml` when set, else `resolve.FabRoot()`; print `spawn.Command(configPath)` to stdout. <!-- R6 -->
- [x] T008 Wire `spawnCommandCmd()` into the root command in `src/go/fab/cmd/fab/main.go`. <!-- R6 -->
- [x] T009 [P] Add `src/go/fab/cmd/fab/spawn_command_test.go`: `--repo` with configured command, `--repo` with no command (DefaultSpawnCommand), and command structure/flag registration. <!-- R6 -->

### Phase 4: Documentation

- [x] T010 Update `src/kit/skills/_cli-fab.md`: add a `## fab spawn-command` section (`fab spawn-command [--repo <path>]`), update the `fab operator tick-start` section to describe the server-keyed XDG state path, and add `repo` to the `fab pane map --json` field list. <!-- R7 -->

## Execution Order

- T001 blocks T002 and T003 (helpers must exist first).
- T004 blocks T005 (both edit `panemap.go`; field added before JSON wiring) and T006.
- T007 blocks T008 and T009.
- Phases 1, 2, 3 are independent of each other; Phase 4 (T010) follows once the signatures are final.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `stateDir()` returns `$XDG_STATE_HOME` when absolute, else `$HOME/.local/state`; never `~/Library`.
- [x] A-002 R2: `serverSlug` produces a deterministic, collision-free, filesystem-safe slug and falls back to `"default"`; `StatePath` returns `<stateDir>/fab/operator/<slug>.yaml` with the parent dir created.
- [x] A-003 R3: `fab operator tick-start` reads/writes the server-keyed state path via `operatorStatePathOverride`; tick increment, `last_tick_at`, field preservation, and stdout format are unchanged; no migration runs.
- [x] A-004 R4: `runOperator` still launches the new window at `gitRepoRoot()`.
- [x] A-005 R5: `fab pane map` computes per-repo `mainRoot` cached by worktree root; each pane's display path is relative to its own repo.
- [x] A-006 R5: `fab pane map --json` emits a `repo` field per element; the human table has no `Repo` column.
- [x] A-007 R6: `fab spawn-command --repo <path>` prints `<path>`'s configured spawn command; omitting `--repo` uses `resolve.FabRoot()`; absent config key prints `DefaultSpawnCommand`.
- [x] A-008 R7: `_cli-fab.md` documents `fab spawn-command`, the server-keyed tick-start state path, and the `repo` JSON field.

### Behavioral Correctness

- [x] A-009 R3: The renamed `operatorStatePathOverride` seam is a full file path (not a directory) and all tick-start tests use it.
- [x] A-010 R5: With panes from two distinct repos, display paths are NOT computed against a single shared `mainRoot` (the prior bug).

### Scenario Coverage

- [x] A-011 R1: `stateDir` table test covers env set-absolute, unset, and relative-ignored cases.
- [x] A-012 R6: `spawn-command` tests cover configured, default-fallback, and structure cases.

### Edge Cases & Error Handling

- [x] A-013 R2: `serverSlug` returns `"default"` when the `#{socket_path}` query fails. <!-- verified by inspection (operator.go:112-114); covered indirectly via StatePath/slugify("") tests, not a dedicated isolated unit test -->
- [x] A-014 R5: `repo` JSON field is `null` (omitted-as-null) when the main root cannot be resolved (non-git pane).

### Code Quality

- [x] A-015 Pattern consistency: New code follows the cobra command structure, `WithServer` plumbing, and table/JSON output patterns already in `operator.go`/`panemap.go`.
- [x] A-016 No unnecessary duplication: `spawn.Command`, `pane.WithServer`, `resolve.FabRoot`, and `runtime.SaveFile` are reused rather than reimplemented.
- [x] A-017 Readability: No god functions; helpers are focused and named (no magic strings for the slug rule).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change is additive (new `stateDir`/`slugify`/`serverSlug`/`StatePath` helpers, new `spawn-command` command, new `repo` field). The one behavioral relocation — `runOperatorTickStart` moving off `gitRepoRoot()/.fab-operator.yaml` — already removed the old repo-rooted path inline; the intake deliberately mandates "no migration, abandon old files in place", so no in-repo code or config became redundant.

## Assumptions

<!-- SRAD record for under-specified points resolved inline at apply. Both open
     intake items (helper placement; repo --json-only) were already clarified in
     the intake (#10, #11) and are honored as Certain here, not re-decided. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | A1 helpers (`stateDir`/`serverSlug`/`StatePath`/`slugify`) live in `cmd/fab/operator.go`, not a new `internal/operator` package | Intake clarification #10 — user confirmed; honored, not re-decided | S:95 R:78 A:90 D:90 |
| 2 | Certain | `repo` exposed in `fab pane map --json` only; no human-table `Repo` column | Intake clarification #11 — user confirmed; honored, not re-decided | S:95 R:80 A:90 D:90 |
| 3 | Certain | Slugify = replace `/` with `-` and strip leading separator (deterministic, collision-free, fs-safe) | Intake left exact rule as apply-time detail but fixed the constraints; this is the simplest rule meeting them | S:88 R:75 A:90 D:80 |
| 4 | Confident | `paneJSON.Repo` is `*string` (nullable via `toNullable`), matching the existing nullable-field pattern (`Change`/`Stage`) | Consistent with how unresolved fields are emitted as JSON null in `panemap.go` | S:82 R:78 A:88 D:80 |
| 5 | Confident | `spawn-command --repo <path>` reads `<path>/fab/project/config.yaml` directly (no upward search from `<path>`); upward search via `resolve.FabRoot()` only for the omitted-`--repo` default | Intake says `--repo` names the repo root; `resolve.FabRoot()` searches from CWD, so the explicit path joins the known config location directly | S:80 R:70 A:85 D:78 |

5 assumptions (3 certain, 2 confident, 0 tentative).
