# Spec: Build fab Go Binary

**Change**: 260305-bhd6-1-build-fab-go-binary
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md` (modify), `docs/memory/fab-workflow/distribution.md` (modify)

## Non-Goals

- Shell script removal or shim switchover — scripts remain unchanged, switchover is a separate change
- Release integration — binary inclusion in `kit.tar.gz` is a separate change
- Modifying existing bats tests — they test shell scripts, not the Go binary
- Cross-compilation CI setup — build pipeline is a separate change

## Go Module: Scaffold and Structure

### Requirement: Go Module at `src/go/fab/`

The Go module SHALL be located at `src/go/fab/` with module path `github.com/wvrdz/fab-kit/src/go/fab`. The module SHALL use Go 1.22+ and depend only on `github.com/spf13/cobra` (CLI framework) and `gopkg.in/yaml.v3` (YAML parsing). No CGo dependencies SHALL be used.

#### Scenario: Module initialization
- **GIVEN** no `src/go/fab/` directory exists
- **WHEN** the module is scaffolded
- **THEN** `src/go/fab/go.mod` declares the module path and Go version
- **AND** `src/go/fab/go.sum` contains checksums for cobra and yaml.v3

### Requirement: Binary Entry Point

The binary entry point SHALL be at `src/go/fab/cmd/fab/main.go`. It SHALL register all subcommands via cobra and exit with the command's exit code.

#### Scenario: Root command with no args
- **GIVEN** the `fab` binary is built
- **WHEN** invoked with no arguments
- **THEN** it prints usage help listing all subcommands
- **AND** exits with code 0

### Requirement: Internal Package Structure

The binary SHALL use Go `internal/` packages to enforce encapsulation:

| Package | Responsibility |
|---------|---------------|
| `internal/statusfile` | Parse, modify, write `.status.yaml` — single `StatusFile` struct |
| `internal/resolve` | Change folder resolution (4-char ID, substring, full name) |
| `internal/log` | JSON-line append to `.history.jsonl` |
| `internal/status` | Stage state machine (all statusman subcommands) |
| `internal/preflight` | Validation + structured YAML output |
| `internal/change` | Create, rename, list, switch changes |
| `internal/score` | Confidence scoring from Assumptions tables |
| `internal/archive` | Archive/restore lifecycle |

#### Scenario: Package isolation
- **GIVEN** the `internal/` directory structure
- **WHEN** an external package attempts to import `internal/statusfile`
- **THEN** the Go compiler rejects the import

## Shared Foundation: `internal/statusfile`

### Requirement: StatusFile Struct

The `internal/statusfile` package SHALL define a `StatusFile` struct that maps 1:1 to the `.status.yaml` schema. The struct SHALL be parsed once via `Load(path string) (*StatusFile, error)` and written atomically via `Save(path string) error`.

#### Scenario: Load valid .status.yaml
- **GIVEN** a valid `.status.yaml` file exists
- **WHEN** `Load()` is called
- **THEN** all fields are populated: `name`, `created`, `created_by`, `change_type`, `issues`, `progress` (map of stage→state), `checklist`, `confidence`, `stage_metrics`, `prs`, `last_updated`

#### Scenario: Atomic save
- **GIVEN** a modified `StatusFile` in memory
- **WHEN** `Save()` is called
- **THEN** the file is written via temp file + rename (atomic)
- **AND** `last_updated` is set to the current ISO 8601 timestamp

#### Scenario: Round-trip fidelity
- **GIVEN** a `.status.yaml` loaded into `StatusFile`
- **WHEN** saved without modification
- **THEN** the output YAML is semantically equivalent to the input (field order may differ, but values are identical)

### Requirement: Progress Map

The `StatusFile.Progress` field SHALL be an ordered map of stage name → state string. The stage order SHALL be fixed: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`, `ship`, `review-pr`.

#### Scenario: Stage ordering preserved
- **GIVEN** a StatusFile with all stages
- **WHEN** serialized to YAML
- **THEN** stages appear in pipeline order, not alphabetical

## Subcommand: `fab resolve`

### Requirement: Change Resolution

`fab resolve` SHALL accept `[--id|--folder|--dir|--status] [<change>]` and produce identical output to `resolve.sh`.

Resolution logic:
1. If `<change>` provided: match against `fab/changes/` folder names (exact → substring → 4-char ID extraction)
2. If `<change>` omitted: read `fab/current` line 2
3. Case-insensitive substring matching
4. If multiple matches: exit 1 with "Multiple changes match: {list}" on stderr
5. If no match and exactly one change exists: use it as fallback

#### Scenario: Resolve by 4-char ID
- **GIVEN** a change folder `260305-bhd6-1-build-fab-go-binary` exists
- **WHEN** `fab resolve bhd6` is called
- **THEN** stdout is `bhd6` (default --id mode)

#### Scenario: Resolve by substring
- **GIVEN** a change folder `260305-bhd6-1-build-fab-go-binary` exists
- **WHEN** `fab resolve build-fab` is called with `--folder`
- **THEN** stdout is `260305-bhd6-1-build-fab-go-binary`

#### Scenario: Ambiguous match
- **GIVEN** two change folders matching "fix"
- **WHEN** `fab resolve fix` is called
- **THEN** exit code 1
- **AND** stderr contains "Multiple changes match"

#### Scenario: No fab/current
- **GIVEN** `fab/current` does not exist
- **WHEN** `fab resolve` is called with no argument
- **THEN** exit code 1
- **AND** stderr contains an error about missing active change

## Subcommand: `fab log`

### Requirement: Append-Only JSON Logging

`fab log` SHALL append JSON lines to `<change_dir>/.history.jsonl` with identical format to `logman.sh`.

Subcommands:
- `fab log command <cmd> [change] [args]` — log skill invocation
- `fab log confidence <change> <score> <delta> <trigger>` — log confidence change
- `fab log review <change> <result> [rework]` — log review outcome
- `fab log transition <change> <stage> <action> [from] [reason] [driver]` — log stage transition

Each JSON line SHALL contain a `ts` field (ISO 8601 UTC) and an `event` field matching the subcommand name.

#### Scenario: Log command without explicit change
- **GIVEN** `fab/current` points to a valid change
- **WHEN** `fab log command "fab-continue"` is called (no change arg)
- **THEN** a JSON line is appended to the active change's `.history.jsonl`
- **AND** the line contains `{"ts":"...","event":"command","cmd":"fab-continue"}`

#### Scenario: Log command with invalid change (no fab/current)
- **GIVEN** `fab/current` does not exist
- **WHEN** `fab log command "fab-status"` is called (no change arg)
- **THEN** exit code 0 (silent — graceful degradation)

#### Scenario: Log review result
- **GIVEN** a valid change "bhd6"
- **WHEN** `fab log review bhd6 "passed"` is called
- **THEN** a JSON line is appended with `{"ts":"...","event":"review","result":"passed"}`

## Subcommand: `fab status`

### Requirement: Stage State Machine

`fab status` SHALL implement the complete statusman.sh state machine with identical subcommands, arguments, and stdout/stderr output.

**Event subcommands** (modify .status.yaml):
- `fab status start <change> <stage> [driver] [from] [reason]` — {pending,failed} → active
- `fab status advance <change> <stage> [driver]` — active → ready
- `fab status finish <change> <stage> [driver]` — {active,ready} → done, auto-activate next pending stage
- `fab status reset <change> <stage> [driver] [from] [reason]` — {done,ready,skipped} → active, cascade downstream to pending
- `fab status skip <change> <stage> [driver]` — {pending,active} → skipped, cascade downstream pending to skipped
- `fab status fail <change> <stage> [driver] [rework]` — active → failed (review/review-pr only)

**Write subcommands** (modify .status.yaml metadata):
- `fab status set-change-type <change> <type>` — set change_type
- `fab status set-checklist <change> <field> <value>` — update checklist (generated/completed/total)
- `fab status set-confidence <change> <certain> <confident> <tentative> <unresolved> <score> [--indicative]`
- `fab status set-confidence-fuzzy <change> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d> [--indicative]`
- `fab status add-issue <change> <id>` — append issue ID (idempotent)
- `fab status add-pr <change> <url>` — append PR URL (idempotent)

**Query subcommands** (read-only):
- `fab status progress-map <change>` — output `stage:state` pairs, one per line
- `fab status progress-line <change>` — single-line visual progress with Unicode symbols
- `fab status current-stage <change>` — detect active/next stage
- `fab status display-stage <change>` — output `stage:state` for display
- `fab status checklist <change>` — output checklist key:value pairs
- `fab status confidence <change>` — output confidence key:value pairs
- `fab status all-stages` — list all stage IDs
- `fab status validate-status-file <change>` — validate against schema
- `fab status get-issues <change>` — list issue IDs
- `fab status get-prs <change>` — list PR URLs

#### Scenario: Finish intake stage
- **GIVEN** a change with `intake: active`
- **WHEN** `fab status finish bhd6 intake fab-ff` is called
- **THEN** `intake` becomes `done`
- **AND** `spec` becomes `active`
- **AND** `stage_metrics.intake` gets `completed_at` timestamp
- **AND** logman transition is recorded (best-effort)

#### Scenario: Finish review stage (auto-log)
- **GIVEN** a change with `review: active`
- **WHEN** `fab status finish bhd6 review fab-ff` is called
- **THEN** `review` becomes `done`
- **AND** `hydrate` becomes `active`
- **AND** a review "passed" log entry is appended to `.history.jsonl`

#### Scenario: Fail review stage (auto-log)
- **GIVEN** a change with `review: active`
- **WHEN** `fab status fail bhd6 review fab-ff "fix-tests"` is called
- **THEN** `review` becomes `failed`
- **AND** a review "failed" log entry with rework info is appended

#### Scenario: Reset with cascade
- **GIVEN** a change with `apply: done`, `review: done`, `hydrate: active`
- **WHEN** `fab status reset bhd6 apply fab-ff` is called
- **THEN** `apply` becomes `active`
- **AND** `review` becomes `pending`
- **AND** `hydrate` becomes `pending`

#### Scenario: Progress line output
- **GIVEN** a change with intake done, spec done, tasks active
- **WHEN** `fab status progress-line bhd6` is called
- **THEN** stdout shows a single line with Unicode symbols: `✓` for done, `⏳` for active, `→` for pending

### Requirement: Stage Metrics

Event subcommands SHALL maintain `stage_metrics` in `.status.yaml`. When a stage becomes `active`, `started_at` and `driver` are recorded. When `finish` is called, `completed_at` is added. The `iterations` counter increments on each `start` or `reset` into a stage.

#### Scenario: First activation sets metrics
- **GIVEN** a stage has never been active
- **WHEN** it transitions to `active`
- **THEN** `stage_metrics.{stage}.started_at` is set to current timestamp
- **AND** `stage_metrics.{stage}.driver` is set to the driver argument
- **AND** `stage_metrics.{stage}.iterations` is set to 1

#### Scenario: Re-entry increments iterations
- **GIVEN** a stage has `iterations: 1`
- **WHEN** it is reset and re-activated
- **THEN** `iterations` becomes 2
- **AND** `started_at` is updated to current timestamp

## Subcommand: `fab preflight`

### Requirement: Validation and Structured Output

`fab preflight [<change-name>]` SHALL produce identical YAML output to `preflight.sh`:

```yaml
id: {4-char}
name: {folder-name}
change_dir: {path}
stage: {current-stage}
display_stage: {display-stage}
display_state: {display-state}
progress:
  intake: {state}
  spec: {state}
  ...
checklist:
  generated: {bool}
  completed: {int}
  total: {int}
confidence:
  certain: {int}
  confident: {int}
  tentative: {int}
  unresolved: {int}
  score: {float}
  indicative: {bool}
```

#### Scenario: Successful preflight
- **GIVEN** a valid project (config.yaml, constitution.md exist) and active change
- **WHEN** `fab preflight` is called
- **THEN** stdout contains structured YAML with all fields
- **AND** exit code 0

#### Scenario: Missing config.yaml
- **GIVEN** `fab/project/config.yaml` does not exist
- **WHEN** `fab preflight` is called
- **THEN** exit code 1
- **AND** stderr contains "Project not initialized"

#### Scenario: Stale sync version
- **GIVEN** `fab/.kit-sync-version` contains an older version than `fab/.kit/VERSION`
- **WHEN** `fab preflight` is called
- **THEN** a non-blocking warning is emitted to stderr
- **AND** the YAML output is still produced (exit code 0)

#### Scenario: Change override
- **GIVEN** `fab/current` points to change A
- **WHEN** `fab preflight change-B-slug` is called
- **THEN** output describes change B, not change A
- **AND** `fab/current` is NOT modified

## Subcommand: `fab change`

### Requirement: Change Lifecycle

`fab change` SHALL implement the complete changeman.sh lifecycle:

**`fab change new --slug <slug> [--change-id <4char>] [--log-args <desc>]`**
- Generate folder name: `YYMMDD-{ID}-{slug}`
- Random 4-char ID from lowercase alphanumeric (retry up to 10 on collision)
- Validate slug: alphanumeric start/end, hyphens allowed in middle
- Initialize `.status.yaml` from template with `name`, `created` (ISO 8601), `created_by` (from `gh api user` → `git config user.name` → "unknown")
- Set intake to `active` via internal status start
- Output: folder name on stdout (one line)

**`fab change rename --folder <current-folder> --slug <new-slug>`**
- Rename change folder preserving YYMMDD-XXXX prefix
- Update `.status.yaml` name field
- Update `fab/current` if it points to old folder
- Output: new folder name on stdout

**`fab change switch <name> | --blank`**
- `--blank`: delete `fab/current`, output "No active change."
- Normal: write `fab/current` (line 1: ID, line 2: folder name), output structured summary

**`fab change list [--archive]`**
- Output: `folder:display_stage:display_state:score:indicative` per line

**`fab change resolve [<override>]`**
- Passthrough to resolve logic with `--folder` mode

#### Scenario: Create new change
- **GIVEN** no existing change with the same ID
- **WHEN** `fab change new --slug build-fab-go --log-args "Build fab Go binary"` is called
- **THEN** a folder `YYMMDD-{ID}-build-fab-go` is created under `fab/changes/`
- **AND** `.status.yaml` is initialized with intake active
- **AND** stdout contains the folder name

#### Scenario: Slug validation failure
- **GIVEN** an invalid slug like "-bad-slug"
- **WHEN** `fab change new --slug "-bad-slug"` is called
- **THEN** exit code 1
- **AND** stderr contains slug validation error

#### Scenario: Switch output format
- **GIVEN** a valid change
- **WHEN** `fab change switch <name>` is called
- **THEN** stdout contains: `fab/current → {name}`, blank line, `Stage:`, `Confidence:`, `Next:` lines

#### Scenario: List changes
- **GIVEN** three changes exist with various stages
- **WHEN** `fab change list` is called
- **THEN** each line has format `folder:display_stage:display_state:score:indicative`

## Subcommand: `fab score`

### Requirement: Confidence Scoring

`fab score [--check-gate] [--stage <stage>] <change>` SHALL implement the complete calc-score.sh algorithm.

**Normal mode**: Parse Assumptions table from spec.md (or intake.md with `--stage intake`), compute confidence score, write to `.status.yaml`, log via internal log package.

**Gate mode** (`--check-gate`): Read-only. Compare score against per-type threshold. Output YAML:
```yaml
gate: pass|fail
score: {float}
threshold: {float}
change_type: {type}
certain: {int}
confident: {int}
tentative: {int}
unresolved: {int}
```

**Scoring algorithm**:
1. Parse `## Assumptions` table from markdown artifact
2. Count grades: certain, confident, tentative, unresolved
3. If unresolved > 0: score = 0.0
4. Else: `base = max(0, 5.0 - 0.3*confident - 1.0*tentative)`, `cover = min(1.0, total/expected_min)`, `score = base * cover`
5. Round to 1 decimal
6. If Scores column contains `S:nn R:nn A:nn D:nn`: compute dimension means, use `set-confidence-fuzzy`

**Expected minimums**: embedded lookup by stage + change_type (same values as calc-score.sh).

**Gate thresholds**: fix=2.0, feat=3.0, refactor=3.0, docs/test/ci/chore=2.0.

#### Scenario: Score from spec
- **GIVEN** spec.md has 8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved)
- **WHEN** `fab score bhd6` is called
- **THEN** score = max(0, 5.0 - 0.3*2) * min(1.0, 8/6) = 4.4 * 1.0 = 4.4
- **AND** `.status.yaml` confidence block is updated

#### Scenario: Gate check pass
- **GIVEN** a feat-type change with score 4.4
- **WHEN** `fab score --check-gate bhd6` is called
- **THEN** stdout contains `gate: pass` and `threshold: 3.0`

#### Scenario: Forced zero on unresolved
- **GIVEN** spec.md has 1 unresolved assumption
- **WHEN** `fab score bhd6` is called
- **THEN** score = 0.0 regardless of other grades

## Subcommand: `fab archive`

### Requirement: Archive Lifecycle

`fab archive` SHALL implement the complete archiveman.sh operations:

**`fab archive <change> --description "..."`**
1. Clean: delete `.pr-done` if present
2. Move: `fab/changes/{folder}` → `fab/changes/archive/{folder}`
3. Index: create or update `fab/changes/archive/index.md` (prepend entry after header)
4. Backfill: add entries for unindexed archived folders
5. Pointer: clear `fab/current` if it points to archived change
- Output: YAML (action, name, clean, move, index, pointer)

**`fab archive restore <change> [--switch]`**
1. Move: `fab/changes/archive/{folder}` → `fab/changes/{folder}`
2. Index: remove entry from `archive/index.md`
3. Pointer: optionally activate via switch (if `--switch`)
- Output: YAML (action, name, move, index, pointer)

**`fab archive list`**
- Output: one folder name per line

#### Scenario: Archive active change
- **GIVEN** `fab/current` points to the change being archived
- **WHEN** `fab archive bhd6 --description "Completed"` is called
- **THEN** folder is moved to archive/
- **AND** `fab/current` is deleted
- **AND** index.md is updated

#### Scenario: Restore with switch
- **GIVEN** an archived change
- **WHEN** `fab archive restore bhd6 --switch` is called
- **THEN** folder is moved back to fab/changes/
- **AND** `fab/current` is written to point to the restored change

## CLI Output Parity

### Requirement: Byte-Compatible Output

All subcommands SHALL produce stdout/stderr output that is byte-compatible with the corresponding shell script, with these exceptions:
- Trailing whitespace differences are tolerated
- Timestamp values will differ (but format must match)
- YAML field ordering may differ within blocks (but pipeline-ordered stages must remain ordered)

This enables parity testing by running both the shell script and Go binary against the same inputs and diffing the outputs (modulo timestamps).

#### Scenario: Parity test approach
- **GIVEN** a test fixture `.status.yaml`
- **WHEN** the same command is run via both `statusman.sh` and `fab status`
- **THEN** stdout output matches (after normalizing timestamps)

## Design Decisions

1. **Single binary, no subpackage binaries**: All subcommands live under one `fab` binary rather than separate binaries per script.
   - *Why*: Eliminates inter-process spawning overhead — the primary performance goal. One process, one YAML parse, in-memory function calls.
   - *Rejected*: Separate binaries per script — preserves bash-like isolation but loses the compound call chain win.

2. **`internal/statusfile` as the shared foundation**: All packages that need `.status.yaml` access import `internal/statusfile` rather than parsing YAML themselves.
   - *Why*: Single parse, single struct, passed by pointer. This is the architectural analog of eliminating 43 yq invocations.
   - *Rejected*: Each package parsing independently — duplicated code, no shared struct.

3. **Atomic file writes via temp+rename**: All `.status.yaml` modifications use write-to-temp-then-rename.
   - *Why*: Prevents corruption on interruption. Consistent with bash version's behavior.
   - *Rejected*: In-place writes — risk of partial writes on crash.

4. **cobra for CLI framework**: Using `github.com/spf13/cobra` rather than stdlib `flag` or alternatives.
   - *Why*: First-class nested subcommand support (`fab status finish`), automatic help generation, widely adopted in Go CLI tooling.
   - *Rejected*: stdlib `flag` — no subcommand nesting support. `urfave/cli` — viable but cobra is more widely used.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go as the implementation language | Confirmed from intake #1 — benchmark data validated, constitution-compliant | S:95 R:85 A:95 D:95 |
| 2 | Certain | Single `fab` binary with subcommands | Confirmed from intake #2 — eliminates inter-script subprocess overhead | S:90 R:80 A:90 D:90 |
| 3 | Certain | Module location at `src/go/fab/` | Confirmed from intake #3 — follows src/ convention | S:85 R:90 A:90 D:90 |
| 4 | Certain | Cobra for CLI framework | Confirmed from intake #4 — standard Go CLI library | S:80 R:90 A:90 D:95 |
| 5 | Certain | `internal/statusfile/` as shared YAML package | Confirmed from intake #5 — single struct, single parse | S:90 R:85 A:90 D:90 |
| 6 | Certain | Identical CLI interface to bash scripts | Confirmed from intake #6 — required for parity testing | S:85 R:70 A:90 D:95 |
| 7 | Certain | Byte-compatible stdout/stderr output | Follows from #6 — parity testing requires output matching | S:85 R:75 A:90 D:90 |
| 8 | Confident | `gopkg.in/yaml.v3` for YAML parsing | Confirmed from intake #7 — standard library, used by yq itself | S:75 R:90 A:85 D:70 |
| 9 | Confident | No CGo dependencies | Confirmed from intake #8 — enables trivial cross-compilation | S:80 R:75 A:85 D:85 |
| 10 | Certain | Atomic file writes via temp+rename | Standard Go pattern for safe file mutation, matches bash behavior | S:80 R:90 A:95 D:95 |
| 11 | Certain | Go 1.22+ minimum version | Current stable Go, no reason to support older versions | S:85 R:90 A:90 D:95 |

11 assumptions (9 certain, 2 confident, 0 tentative, 0 unresolved).
