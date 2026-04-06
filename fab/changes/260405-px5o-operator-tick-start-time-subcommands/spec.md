# Spec: Operator tick-start and time subcommands for fab-go

**Change**: 260405-px5o-operator-tick-start-time-subcommands
**Created**: 2026-04-05
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- `fab operator tick-start` does NOT output a `next:` field — computing "next tick" at start-of-tick is not meaningful because the loop interval may change between tick start and idle display
- `fab operator time` does NOT write any files — it is a pure query command with no side effects
- No changes to the loop interval default (3m) or the `/loop` invocation in `fab-operator.md`
- No changes to the `.fab-operator.yaml` schema beyond the already-present `tick_count` and `last_tick_at` fields

---

## Go: `fab operator` command refactor

### Requirement: Operator command becomes a parent command

`operatorCmd()` in `src/go/fab/cmd/fab/operator.go` SHALL be refactored from a leaf command (cobra.NoArgs) to a parent command that accepts subcommands, while preserving the existing launch behavior as its default RunE.

The refactor MUST:
1. Remove the `Args: cobra.NoArgs` constraint from the returned `*cobra.Command`
2. Keep `RunE: runOperator` as the default RunE (cobra calls this when no subcommand is provided)
3. Register `tick-start` and `time` as subcommands via `cmd.AddCommand(...)`

The existing `runOperator` function body SHALL NOT be changed.

#### Scenario: Existing operator launch still works after refactor
- **GIVEN** the user is inside a tmux session
- **WHEN** `fab operator` is invoked with no subcommand
- **THEN** `runOperator` fires, a new tmux window named "operator" is created (or the existing one is selected)
- **AND** no behavior change compared to the pre-refactor implementation

#### Scenario: Subcommands are discoverable via help
- **GIVEN** the operator command has subcommands registered
- **WHEN** `fab operator --help` is invoked
- **THEN** `tick-start` and `time` appear as available subcommands in the help output

---

## Go: `fab operator tick-start` subcommand

### Requirement: tick-start atomically increments tick state and outputs current time

The `tick-start` subcommand SHALL:

1. Resolve the repo root via `gitRepoRoot()`
2. Read `.fab-operator.yaml` from the repo root. If the file is absent, treat it as an empty document (`tick_count` = 0, all other fields absent)
3. Increment `tick_count` by 1 (if the field is absent or zero, the result is 1)
4. Write `last_tick_at` as an RFC3339 UTC timestamp using `time.Now().UTC().Format(time.RFC3339)` (e.g., `2026-04-05T10:42:00Z`)
5. Write the updated document back to `.fab-operator.yaml`, preserving all other fields unchanged
6. Output to stdout exactly:
   ```
   tick: N
   now: HH:MM
   ```
   where `N` is the incremented `tick_count` value and `HH:MM` is `time.Now().Format("15:04")` (local time, 24-hour format)

The subcommand SHALL have no flags.

**File I/O**: Read the full file into a `map[string]interface{}` using `gopkg.in/yaml.v3`. Update only `tick_count` and `last_tick_at`. Marshal and write back. This preserves monitored set, autopilot queue, branch_map, and all other fields.

**Error handling**: Missing file → treat as empty (start from 0, do not error). Write failure → print error to stderr and exit 1.

#### Scenario: tick-start increments an existing tick_count
- **GIVEN** `.fab-operator.yaml` exists with `tick_count: 47`
- **WHEN** `fab operator tick-start` is invoked
- **THEN** stdout contains `tick: 48` on the first line
- **AND** stdout contains `now: HH:MM` on the second line (HH:MM matches local time)
- **AND** `.fab-operator.yaml` now has `tick_count: 48`
- **AND** `.fab-operator.yaml` now has `last_tick_at` set to a valid RFC3339 UTC timestamp

#### Scenario: tick-start creates file when .fab-operator.yaml is missing
- **GIVEN** `.fab-operator.yaml` does not exist in the repo root
- **WHEN** `fab operator tick-start` is invoked
- **THEN** stdout contains `tick: 1`
- **AND** `.fab-operator.yaml` is created with `tick_count: 1` and a valid `last_tick_at`

#### Scenario: tick-start preserves existing fields
- **GIVEN** `.fab-operator.yaml` exists with `tick_count: 5` and `monitored: {r3m7: {...}}`
- **WHEN** `fab operator tick-start` is invoked
- **THEN** the written `.fab-operator.yaml` still contains the `monitored` field with its original value unchanged

#### Scenario: tick-start fails on unwritable file
- **GIVEN** `.fab-operator.yaml` cannot be written (e.g., directory is read-only)
- **WHEN** `fab operator tick-start` is invoked
- **THEN** an error is printed to stderr
- **AND** the process exits with code 1

---

## Go: `fab operator time` subcommand

### Requirement: time outputs current time and optionally next-tick time

The `time` subcommand SHALL:

1. Always output `now: HH:MM` where `HH:MM` is `time.Now().Format("15:04")` (local time, 24-hour)
2. When `--interval <duration>` is provided:
   - Parse the duration string using `time.ParseDuration(interval)`
   - Compute `next = time.Now().Add(parsedDuration)`
   - Output `next: HH:MM` using `next.Format("15:04")` as a second line
3. Make no file reads and no file writes

**Flag**: `--interval` accepts a Go duration string (e.g., `3m`, `5m`, `2m`). Invalid duration string → print error to stderr and exit 1. No other failure modes.

#### Scenario: time without --interval outputs only now
- **GIVEN** no `--interval` flag is provided
- **WHEN** `fab operator time` is invoked
- **THEN** stdout contains exactly one line: `now: HH:MM`
- **AND** no `next:` line appears

#### Scenario: time with --interval outputs now and next
- **GIVEN** `--interval 3m` is provided
- **WHEN** `fab operator time --interval 3m` is invoked at 08:26 local time
- **THEN** stdout contains `now: 08:26` on the first line
- **AND** stdout contains `next: 08:29` on the second line

#### Scenario: time with invalid --interval exits 1
- **GIVEN** `--interval notaduration` is provided
- **WHEN** `fab operator time --interval notaduration` is invoked
- **THEN** an error is printed to stderr
- **AND** the process exits with code 1

---

## Go: Tests in `operator_test.go`

### Requirement: Tests added alongside existing tests

Five new test functions SHALL be added to `src/go/fab/cmd/fab/operator_test.go` (the existing file — not a new file). Tests follow the existing `test-alongside` pattern in the codebase.

Required tests:

| Function | Validates |
|---|---|
| `TestOperatorTickStart_IncrementsCount` | Creates temp `.fab-operator.yaml` with `tick_count: 5`, runs tick-start, verifies stdout contains `tick: 6` and `now: \d\d:\d\d`, verifies written YAML has `tick_count: 6` and `last_tick_at` set |
| `TestOperatorTickStart_MissingFile` | Runs tick-start with no `.fab-operator.yaml`, verifies file is created with `tick_count: 1` and stdout contains `tick: 1` |
| `TestOperatorTime_NoInterval` | Verifies stdout contains `now: ` matching `\d\d:\d\d` and no `next:` line |
| `TestOperatorTime_WithInterval` | Runs with `--interval 3m`, verifies stdout contains both `now: \d\d:\d\d` and `next: \d\d:\d\d` |
| `TestOperatorTime_InvalidInterval` | Runs with `--interval bogus`, verifies exit 1 |

Each test that exercises `.fab-operator.yaml` I/O SHALL use a `t.TempDir()` to isolate file system side effects.

#### Scenario: tick-start test uses temp dir
- **GIVEN** the test creates a temp dir and writes a `.fab-operator.yaml` there
- **WHEN** `TestOperatorTickStart_IncrementsCount` runs
- **THEN** the command reads from and writes to the temp dir, not the real repo root
- **AND** the test validates the updated YAML contents

#### Scenario: time subcommand tests require no file system
- **GIVEN** `TestOperatorTime_NoInterval` and `TestOperatorTime_WithInterval` run
- **WHEN** the subcommand executes
- **THEN** no files are read or written (pure stdout validation only)

---

## Skill: `src/kit/skills/fab-operator.md` updates

### Requirement: Tick Behavior Step 1 uses `fab operator tick-start`

The **Tick Behavior** step 1 (Snapshot) in Section 4 (The Loop) SHALL be updated to:

1. Replace any manual `tick_count` increment instruction with `fab operator tick-start`
2. Specify that the agent parses stdout for `tick: N` (tick number) and `now: HH:MM` (current time)
3. Retain the instruction to run `fab pane map` and read `.fab-operator.yaml` after the tick-start command

The updated step 1 SHALL read:

> **Snapshot** — run `fab operator tick-start` (increments `tick_count`, writes `last_tick_at`, outputs `tick: N` and `now: HH:MM`). Parse stdout for the tick number and current time. Then run `fab pane map` and read `.fab-operator.yaml`.

#### Scenario: Agent uses tick-start at step 1
- **GIVEN** the operator is executing a tick
- **WHEN** step 1 runs
- **THEN** the agent invokes `fab operator tick-start` and parses its stdout for the tick count and current time
- **AND** the agent does NOT use shell `date` or manually write `last_tick_at`

### Requirement: Idle Message uses `fab operator time --interval {interval}`

The **Idle Message** paragraph SHALL be updated to:

1. Specify `fab operator time --interval {interval}` as the command the operator runs to get `now:` and `next:` values
2. Show the output format (`now: HH:MM` and `next: HH:MM`) and how values map to the idle message

The updated idle message paragraph SHALL read (approximately):

> Between ticks, the operator displays an idle message with the current time and next-tick time:
>
> `Waiting for next tick. Time: 08:26 · next tick: 08:29`
>
> Run `fab operator time --interval {interval}` (where `{interval}` is the current loop interval, e.g. `3m`) to get the `now:` and `next:` values to fill in the message.

#### Scenario: Agent computes idle message time correctly
- **GIVEN** the loop interval is `3m` and the current time is 08:26 local
- **WHEN** the agent executes `fab operator time --interval 3m`
- **THEN** stdout is `now: 08:26\nnext: 08:29`
- **AND** the agent constructs the idle message as `Waiting for next tick. Time: 08:26 · next tick: 08:29`

---

## Skill: `src/kit/skills/_cli-fab.md` updates

### Requirement: fab operator section documents tick-start and time subcommands

The `## fab operator` section in `_cli-fab.md` SHALL be updated to document both new subcommands. The existing section prose and behavior description for the parent `fab operator` command SHALL be preserved.

The following documentation SHALL be added:

```
### fab operator tick-start

Called at the start of each operator tick. Increments `tick_count` by 1, writes `last_tick_at` (ISO 8601 UTC) to `.fab-operator.yaml`, and outputs current time.

\`\`\`
fab operator tick-start
\`\`\`

**Output** (stdout):
\`\`\`
tick: N
now: HH:MM
\`\`\`

No flags. Side effects: updates `.fab-operator.yaml` (`tick_count`, `last_tick_at`).

### fab operator time

Pure time query — no side effects, no file writes.

\`\`\`
fab operator time [--interval <duration>]
\`\`\`

| Flag | Type | Description |
|------|------|-------------|
| `--interval` | duration string | If given, also outputs `next: HH:MM` = now + interval |

**Output** without `--interval`:
\`\`\`
now: HH:MM
\`\`\`

**Output** with `--interval 3m`:
\`\`\`
now: HH:MM
next: HH:MM
\`\`\`

Duration format: Go duration strings (`3m`, `5m`, `2m`). Invalid duration → exit 1.
```

#### Scenario: _cli-fab.md documents the new subcommands
- **GIVEN** `_cli-fab.md` has been updated
- **WHEN** an agent reads the `fab operator` section
- **THEN** the agent can determine the correct invocation for `fab operator tick-start` and `fab operator time --interval 3m`

---

## Memory: `docs/memory/fab-workflow/execution-skills.md`

### Requirement: operator section updated to document tick-start and time

The `execution-skills.md` memory file SHALL be updated (during the hydrate stage) to document:

1. `fab operator tick-start` — its purpose (start-of-tick atomic state update), output format, and side effects on `.fab-operator.yaml`
2. `fab operator time` — its purpose (pure clock query), `--interval` flag, and output format
3. How these commands are used in the tick behavior (step 1 and idle message)
4. The reason for two separate commands (separation of concerns: tick-start has side effects, time is pure)

#### Scenario: Memory accurately reflects new command usage
- **GIVEN** the hydrate stage runs
- **WHEN** `execution-skills.md` is updated
- **THEN** the operator section includes both new subcommands with their purpose and output format
- **AND** the tick behavior step 1 and idle message are documented as using these commands

---

## Memory: `docs/memory/fab-workflow/kit-architecture.md`

### Requirement: Command Reference updated to reflect fab operator subcommands

The Command Reference entry for `fab operator` in `kit-architecture.md` SHALL be updated to reflect that it now has subcommands (`tick-start`, `time`) in addition to its default launch behavior.

#### Scenario: kit-architecture reflects new subcommands
- **GIVEN** the hydrate stage runs
- **WHEN** `kit-architecture.md` is updated
- **THEN** the `fab operator` entry notes that it has two subcommands: `tick-start` (tick state + time) and `time` (pure clock query)

---

## Design Decisions

### 1. Two commands instead of one

**Decision**: Two separate subcommands (`tick-start` and `time`) rather than a single multipurpose command.

**Why**: `tick-start` has a side effect (writes `.fab-operator.yaml`) that is only appropriate at one specific point in the tick lifecycle (step 1). `fab operator time` is a pure query called at a different point (idle message display). Merging them would mean either always writing the YAML (wrong for idle queries) or adding a `--no-write` flag (confusing inversion). Separating them makes the side-effect contract explicit at the call site: seeing `tick-start` in the skill tells readers "this writes state"; seeing `time` tells readers "this is read-only." Constitution §III (Idempotent Operations) further reinforces keeping pure queries free of side effects.

**Rejected**: A single `fab operator tick [--write]` command — the flag inversion is awkward and makes the pure-query case harder to discover. A single `fab operator tick` that always writes — breaks the idle message use case where no write is wanted.

### 2. tick-start does NOT output `next:`

**Decision**: `fab operator tick-start` outputs only `tick: N` and `now: HH:MM`. It does not accept `--interval` and does not output `next:`.

**Why**: At the time `tick-start` runs (step 1 of the tick), the loop continuation decision has not yet been made (step 7 determines whether the loop continues). The next-tick time is only meaningful after step 7 confirms the loop continues. Even if the interval were known, computing `next` before the loop-continue decision would produce a value that might be displayed relative to a tick that does not happen. Using `fab operator time --interval {interval}` at idle-message time (after step 7) produces the correct `next` value aligned with the actual next-tick schedule.

**Rejected**: Adding `--interval` to `tick-start` — the field is semantically incorrect at start-of-tick; see above. Computing `next` internally in `tick-start` without a flag — would hard-code a loop interval that belongs to the skill's session scope, not the binary.

### 3. Cobra parent-with-default-RunE pattern

**Decision**: Refactor `operatorCmd()` to return a parent command that keeps `RunE: runOperator` as the default, and registers subcommands via `cmd.AddCommand(...)`.

**Why**: Cobra supports parent commands with a `RunE` that fires when no subcommand is given. This preserves the existing `fab operator` behavior (open tmux tab) with zero behavioral change for current users, while enabling subcommand dispatch for `tick-start` and `time`. The alternative — a separate top-level command — would break the logical grouping and require separate entries in the command reference. The parent-with-default pattern is the standard cobra idiom for this pattern and is already used elsewhere in fab-go (e.g., `fab batch`).

**Rejected**: Making `fab operator` purely a parent command with no default RunE — would break existing `fab operator` invocations that expect the tmux launch behavior. Adding a `fab operator launch` subcommand to house the tmux behavior — unnecessary churn; all callers of `fab operator` (including `_cli-fab.md` and the constitution's "singleton tmux tab" description) would need updating.

---

## Deprecated Requirements

### `fab operator` Args constraint

**Reason**: `Args: cobra.NoArgs` is removed as part of the leaf → parent refactor. The parent command now accepts subcommands, so cobra.NoArgs would prevent subcommand dispatch.

**Migration**: The default RunE (`runOperator`) continues to fire when `fab operator` is called with no arguments, producing identical behavior. Users invoking `fab operator` directly see no change.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use Go `time` package for all clock operations (not shell `date`) | Confirmed from intake #1. Explicitly stated; constitution §V (Portability). | S:95 R:95 A:95 D:95 |
| 2 | Certain | `fab operator tick-start` outputs `tick: N\nnow: HH:MM` format (exact two-line format) | Confirmed from intake #2. Exact format given in intake and reproduced verbatim here. | S:95 R:90 A:95 D:95 |
| 3 | Certain | `fab operator time --interval` outputs `now: HH:MM\nnext: HH:MM` | Confirmed from intake #3. Exact format given in intake. | S:95 R:90 A:95 D:95 |
| 4 | Certain | `--interval` accepts Go duration strings (`3m`, `5m`, etc.) | Confirmed from intake #4. Explicitly stated. | S:95 R:90 A:95 D:95 |
| 5 | Certain | Operator command refactored from leaf to parent, keeping `runOperator` as default RunE | Confirmed from intake #5. Intake explicitly describes the pattern; cobra parent-with-default-RunE is standard. | S:95 R:90 A:95 D:95 |
| 6 | Certain | Use `gopkg.in/yaml.v3` for `.fab-operator.yaml` read/write in tick-start | Upgraded from intake Confident #6. Spec analysis confirms `gopkg.in/yaml.v3` is already used by `internal/spawn/` in fab-go (kit-architecture.md line 120 confirms this). Direct code evidence eliminates uncertainty. | S:90 R:85 A:90 D:90 |
| 7 | Certain | `tick_count` starts at 0 if `.fab-operator.yaml` is missing or field absent | Confirmed from intake #7. Intake says "treat tick_count as 0" — unambiguous. | S:90 R:90 A:90 D:90 |
| 8 | Certain | Tests added to `operator_test.go` alongside existing tests (not a new file) | Confirmed from intake #8. Existing `operator_test.go` confirms this pattern; constitution: "MUST include corresponding test updates"; code-quality: test-alongside strategy. | S:90 R:90 A:90 D:90 |
| 9 | Certain | `last_tick_at` written as RFC3339 UTC string (`time.RFC3339`) | Confirmed from intake #9. Intake says "ISO 8601 UTC timestamp; e.g. `2026-04-05T10:42:00Z`" — Go's `time.RFC3339` produces this format exactly. | S:90 R:85 A:90 D:90 |
| 10 | Confident | Memory updates scoped to `execution-skills.md` and `kit-architecture.md` (not a new memory file) | Confirmed from intake #10. Both files confirmed to exist and already document operator behavior. Kit-architecture.md has `fab operator` entry at line 144, 312. Execution-skills.md documents loop/tick behavior. | S:80 R:80 A:85 D:85 |
| 11 | Confident | Tests use `t.TempDir()` to isolate `.fab-operator.yaml` I/O | Not in intake. Spec-level addition. Standard Go test isolation practice; `tick-start` reads from repo root by default, so tests must redirect to a temp dir to be hermetic. Pattern-consistent with Go stdlib test conventions. | S:75 R:80 A:85 D:80 |
| 12 | Confident | Existing `TestOperatorCmd_Structure` test updated (or a new parallel test added) to verify subcommands are registered | Not in intake. Spec-level addition. The existing test checks `cmd.Use == "operator"` — after refactor it should also verify that `tick-start` and `time` are registered. Low risk: test-only change. | S:70 R:85 A:80 D:75 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).
