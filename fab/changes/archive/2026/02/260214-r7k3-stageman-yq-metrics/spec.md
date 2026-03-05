# Spec: Stage Metrics, History Tracking & Stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Created**: 2026-02-14
**Affected memory**:
- `docs/memory/fab-workflow/kit-architecture.md` (modify)
- `docs/memory/fab-workflow/change-lifecycle.md` (modify)
- `docs/memory/fab-workflow/planning-skills.md` (modify)
- `docs/memory/fab-workflow/execution-skills.md` (modify)

## Non-Goals

- Token/cost tracking — hooks don't expose token data; deferred to a future change
- Interactive stage_metrics visualization or `/fab-status` display — this change adds the data; display can come later
- Incremental/partial yq migration — all awk-based `.status.yaml` functions migrate at once
- Schema query migration — `workflow.yaml` functions remain awk-based (static controlled file)

## Kit Architecture: yq Dependency & Stageman Migration

### Requirement: yq Runtime Dependency

`lib/stageman.sh` SHALL depend on `yq` (Mike Farah's Go v4 binary) for all `.status.yaml` read and write operations. All `yq` invocations MUST use the Go v4 syntax (e.g., `yq '.path.to.key' file.yaml`).
<!-- clarified: yq dependency resolved — constitution I amended (v1.1.0) to allow single-binary utilities -->

#### Scenario: yq available
- **GIVEN** `yq` (v4) is installed and on PATH
- **WHEN** any stageman `.status.yaml` function is invoked
- **THEN** the function SHALL execute successfully using yq for YAML parsing

#### Scenario: yq not available
- **GIVEN** `yq` is not installed or not on PATH
- **WHEN** `lib/stageman.sh` is sourced or executed
- **THEN** it SHALL emit an error to stderr: `"ERROR: yq (v4) is required but not found. Install: https://github.com/mikefarah/yq"`
- **AND** return/exit with code 1

### Requirement: Accessor Migration to yq

All `.status.yaml` accessor functions (`get_progress_map`, `get_checklist`, `get_confidence`) SHALL be migrated from awk/grep/sed to yq. The output format (`key:value` per line) SHALL remain unchanged for backward compatibility with existing consumers (preflight, calc-score, skill prompts).

#### Scenario: get_progress_map output unchanged
- **GIVEN** a valid `.status.yaml` with `brief: done`, `spec: active`, and remaining stages `pending`
- **WHEN** `get_progress_map <file>` is called
- **THEN** it SHALL output one `stage:state` pair per line in stage order, identical to the current awk-based output

#### Scenario: get_checklist output unchanged
- **GIVEN** a valid `.status.yaml` with `generated: true`, `completed: 3`, `total: 10`
- **WHEN** `get_checklist <file>` is called
- **THEN** it SHALL output `generated:true`, `completed:3`, `total:10` on separate lines

#### Scenario: get_confidence output unchanged
- **GIVEN** a valid `.status.yaml` with confidence fields
- **WHEN** `get_confidence <file>` is called
- **THEN** it SHALL output `certain:N`, `confident:N`, `tentative:N`, `unresolved:N`, `score:N.N` on separate lines

### Requirement: Write Function Migration to yq

All stageman write functions (`set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`) SHALL be migrated from awk to yq. Each function SHALL maintain its current interface for non-metrics parameters (arguments, return codes, error messages). Temp-file-then-mv atomicity SHALL be preserved. `last_updated` auto-refresh SHALL be preserved.

#### Scenario: set_stage_state with yq
- **GIVEN** a valid `.status.yaml`
- **WHEN** `set_stage_state <file> spec active fab-continue` is called
- **THEN** `progress.spec` SHALL be `active`
- **AND** `last_updated` SHALL be refreshed to current ISO 8601 timestamp
- **AND** all other fields SHALL remain unchanged

#### Scenario: transition_stages with yq
- **GIVEN** a `.status.yaml` where `brief` is `active`
- **WHEN** `transition_stages <file> brief spec fab-continue` is called
- **THEN** `progress.brief` SHALL be `done` and `progress.spec` SHALL be `active`
- **AND** the write SHALL be atomic (temp-file-then-mv)

#### Scenario: Validation errors preserved
- **GIVEN** `set_stage_state` is called with an invalid stage
- **WHEN** validation fails
- **THEN** the error message and return code SHALL match current behavior

### Requirement: Validation Migration to yq

`validate_status_file` SHALL be migrated from grep/awk to yq for schema checks. The validation logic SHALL remain unchanged: valid states per stage, active count <= 1. `validate_status_file` SHALL NOT validate the `stage_metrics` block — it is internal bookkeeping written exclusively by stageman's own functions.
<!-- clarified: validate_status_file skips stage_metrics validation — internal bookkeeping, trust the writer -->

#### Scenario: valid status file
- **GIVEN** a `.status.yaml` with valid states for all stages and at most one `active`
- **WHEN** `validate_status_file <file>` is called
- **THEN** it SHALL return 0

#### Scenario: invalid state detected
- **GIVEN** a `.status.yaml` where `spec` has state `invalid_state`
- **WHEN** `validate_status_file <file>` is called
- **THEN** it SHALL print an error to stderr and return 1

#### Scenario: stage_metrics block ignored during validation
- **GIVEN** a `.status.yaml` with a `stage_metrics` block containing any content
- **WHEN** `validate_status_file <file>` is called
- **THEN** it SHALL NOT inspect or validate the `stage_metrics` block

### Requirement: Schema Query Functions Unchanged

Schema query functions (`get_all_stages`, `get_all_states`, `validate_stage`, `validate_state`, `get_stage_number`, `get_stage_name`, `get_stage_artifact`, `get_allowed_states`, `get_initial_state`, `has_auto_checklist`, `get_state_symbol`, `is_terminal_state`, `get_next_stage`) SHALL remain awk-based. These read `workflow.yaml`, a controlled schema file that does not require yq's capabilities.

#### Scenario: schema queries unaffected
- **GIVEN** any schema query function
- **WHEN** invoked with a valid argument
- **THEN** it SHALL produce identical output to the pre-migration implementation

## Change Lifecycle: Stage Metrics

### Requirement: stage_metrics Block in .status.yaml

`.status.yaml` SHALL include a `stage_metrics` block that tracks per-stage operational data. The block SHALL use flow/inline YAML style (one line per stage) for compact representation. Absent stage entries mean "not yet started." Maximum 6 entries (one per pipeline stage).

```yaml
stage_metrics:
  brief: {started_at: "2026-02-14T07:11:18+05:30", completed_at: "2026-02-14T07:11:45+05:30", driver: fab-new, iterations: 1}
  spec:  {started_at: "2026-02-14T07:20:00+05:30", driver: fab-continue, iterations: 1}
```

| Field | Type | Set when |
|-------|------|----------|
| `started_at` | ISO 8601 timestamp | state -> `active` |
| `completed_at` | ISO 8601 timestamp | state -> `done` |
| `driver` | freeform string | state -> `active` (skill identity) |
| `iterations` | positive integer | Incremented on each state -> `active` (1 = first, 2+ = rework) |

#### Scenario: First activation of a stage
- **GIVEN** stage `spec` has no entry in `stage_metrics`
- **WHEN** `set_stage_state <file> spec active fab-continue` is called
- **THEN** `stage_metrics.spec` SHALL be created: `{started_at: "<now>", driver: fab-continue, iterations: 1}`

#### Scenario: Stage completion
- **GIVEN** stage `spec` has `started_at`, `driver`, and `iterations: 1` in stage_metrics
- **WHEN** `set_stage_state <file> spec done` is called
- **THEN** `stage_metrics.spec.completed_at` SHALL be set to current timestamp
- **AND** `started_at`, `driver`, and `iterations` SHALL be preserved

#### Scenario: Rework re-activation (review failure path)
- **GIVEN** stage `apply` has `completed_at` and `iterations: 1` in stage_metrics
- **WHEN** `set_stage_state <file> apply active fab-continue` is called (after review failure)
- **THEN** `stage_metrics.apply.started_at` SHALL be updated to current timestamp
- **AND** `stage_metrics.apply.iterations` SHALL be incremented to `2`
- **AND** `completed_at` SHALL be removed (stage is in-progress again)
- **AND** `driver` SHALL be updated to `fab-continue`

#### Scenario: Reset to pending clears metrics
- **GIVEN** stage `tasks` has metrics: `{started_at: ..., completed_at: ..., driver: fab-continue, iterations: 1}`
- **WHEN** `set_stage_state <file> tasks pending` is called (downstream reset)
- **THEN** `stage_metrics.tasks` entry SHALL be removed entirely
- **AND** future activation of `tasks` SHALL start fresh at `iterations: 1`

#### Scenario: Transition triggers metrics side-effects
- **GIVEN** `brief` is `active` with metrics `{started_at: ..., driver: fab-new, iterations: 1}`
- **WHEN** `transition_stages <file> brief spec fab-continue` is called
- **THEN** `stage_metrics.brief.completed_at` SHALL be set (from->done side-effect)
- **AND** `stage_metrics.spec` SHALL be created with `started_at`, `driver: fab-continue`, `iterations: 1` (to->active side-effect)

### Requirement: stage_metrics API Functions

`lib/stageman.sh` SHALL provide new functions for stage_metrics:

- **`get_stage_metrics <status_file> [stage]`** — Without stage: output all stage metrics as `stage:{flow-yaml}` per line. With stage: output the single stage's fields as `field:value` per line.
- **`set_stage_metric <status_file> <stage> <field> <value>`** — Set an individual metric field. Creates the `stage_metrics` map and stage entry if absent. Used internally by state transition side-effects.

#### Scenario: get_stage_metrics for all stages
- **GIVEN** a `.status.yaml` with metrics for `brief` and `spec`
- **WHEN** `get_stage_metrics <file>` is called
- **THEN** it SHALL output two lines with `stage:{flow-yaml}` pairs

#### Scenario: get_stage_metrics with missing block
- **GIVEN** a `.status.yaml` without a `stage_metrics` key (or with `stage_metrics: {}`)
- **WHEN** `get_stage_metrics <file>` is called
- **THEN** it SHALL return empty output (no lines) and exit 0
<!-- clarified: get_stage_metrics returns empty for missing/empty stage_metrics — backward-compatible with old status files -->

#### Scenario: get_stage_metrics for single stage
- **GIVEN** a `.status.yaml` with `spec: {started_at: "...", driver: fab-continue, iterations: 1}`
- **WHEN** `get_stage_metrics <file> spec` is called
- **THEN** it SHALL output `started_at:...`, `driver:fab-continue`, `iterations:1` on separate lines

#### Scenario: set_stage_metric creates new entry
- **GIVEN** `stage_metrics` has no entry for `spec`
- **WHEN** `set_stage_metric <file> spec started_at "2026-02-14T07:20:00+05:30"` is called
- **THEN** `stage_metrics.spec.started_at` SHALL be set
- **AND** `last_updated` SHALL be refreshed

### Requirement: Automatic Metrics Side-Effects

`set_stage_state` and `transition_stages` SHALL automatically update `stage_metrics` as a side-effect of state transitions.

**Updated function signatures:**
- `set_stage_state <status_file> <stage> <state> [driver]` — `driver` required when state is `active`, ignored otherwise
- `transition_stages <status_file> <from_stage> <to_stage> [driver]` — `driver` required (applied to `to_stage` which transitions to `active`)

**Updated CLI signatures:**
- `stageman.sh set-state <file> <stage> <state> [driver]`
- `stageman.sh transition <file> <from-stage> <to-stage> [driver]`

**Side-effect rules:**

| New state | Metrics action |
|-----------|---------------|
| `active` | Set `started_at` to now, set `driver`, increment `iterations` (or init to 1), remove `completed_at` |
| `done` | Set `completed_at` to now |
| `pending` | Remove stage entry from `stage_metrics` |
| `failed` | No metrics change (preserves timing data) |

#### Scenario: driver required for active transition
- **GIVEN** a valid `.status.yaml`
- **WHEN** `set_stage_state <file> spec active` is called without a driver argument
- **THEN** it SHALL emit an error: `"ERROR: driver required when setting state to 'active'"`
- **AND** return 1

#### Scenario: driver ignored for non-active states
- **GIVEN** a valid `.status.yaml`
- **WHEN** `set_stage_state <file> spec done` is called without driver
- **THEN** it SHALL succeed without error

### Requirement: status.yaml Template Update

`fab/.kit/templates/status.yaml` SHALL include `stage_metrics: {}` as an empty map, placed between the `confidence` block and `last_updated`.

#### Scenario: New change initialization
- **GIVEN** `/fab-new` creates a new change from the template
- **WHEN** `.status.yaml` is initialized
- **THEN** it SHALL include `stage_metrics: {}` as an empty map

## Change Lifecycle: History Log

### Requirement: .history.jsonl Event Log

Each change folder SHALL support an append-only `.history.jsonl` file for event logging. Events are one JSON object per line, sharing `ts` (ISO 8601) and `event` (string) fields.

**Event types:**

| Event | Fields | Triggered by |
|-------|--------|-------------|
| `command` | `cmd` (string), `args` (string, optional) | Skill invocation start |
| `confidence` | `score` (float), `delta` (string), `trigger` (string) | `calc-score.sh` |
| `review` | `result` ("passed"\|"failed"), `rework` (string, optional) | Review verdict |

#### Scenario: Log a command event
- **GIVEN** a change directory
- **WHEN** `log_command <change_dir> "fab-continue" ""` is called
- **THEN** a JSON line SHALL be appended to `<change_dir>/.history.jsonl`: `{"ts":"...","event":"command","cmd":"fab-continue"}`
- **AND** if `args` is non-empty, an `"args"` field SHALL be included

#### Scenario: Log a confidence event
- **GIVEN** a change directory
- **WHEN** `log_confidence <change_dir> 4.1 "+4.1" "fab-continue/spec"` is called
- **THEN** a JSON line SHALL be appended: `{"ts":"...","event":"confidence","score":4.1,"delta":"+4.1","trigger":"fab-continue/spec"}`

#### Scenario: Log a review event (pass)
- **GIVEN** a change directory
- **WHEN** `log_review <change_dir> "passed"` is called
- **THEN** a JSON line SHALL be appended: `{"ts":"...","event":"review","result":"passed"}`

#### Scenario: Log a review event (failure)
- **GIVEN** a change directory
- **WHEN** `log_review <change_dir> "failed" "revise-tasks"` is called
- **THEN** a JSON line SHALL be appended: `{"ts":"...","event":"review","result":"failed","rework":"revise-tasks"}`

#### Scenario: First event creates file
- **GIVEN** no `.history.jsonl` exists in the change directory
- **WHEN** any log function is called
- **THEN** `.history.jsonl` SHALL be created with the event as its first line

### Requirement: History Log API Functions

`lib/stageman.sh` SHALL provide new logging functions:

- **`log_command <change_dir> <cmd> [args]`** — Append a `command` event
- **`log_confidence <change_dir> <score> <delta> <trigger>`** — Append a `confidence` event
- **`log_review <change_dir> <result> [rework]`** — Append a `review` event

These SHALL also be available as CLI commands:
- `stageman.sh log-command <change_dir> <cmd> [args]`
- `stageman.sh log-confidence <change_dir> <score> <delta> <trigger>`
- `stageman.sh log-review <change_dir> <result> [rework]`

#### Scenario: CLI log-command
- **GIVEN** a valid change directory
- **WHEN** `stageman.sh log-command <dir> fab-new "add oauth"` is called
- **THEN** a command event SHALL be appended to `<dir>/.history.jsonl`

## Downstream: calc-score.sh Integration

### Requirement: calc-score.sh History Logging

`lib/calc-score.sh` SHALL call `log_confidence` after computing the score, passing the computed score, delta, and trigger string `"calc-score"`.

#### Scenario: Score computation logs confidence event
- **GIVEN** `calc-score.sh <change_dir>` is invoked and computes score=4.1, delta="+4.1"
- **WHEN** the score write completes
- **THEN** `log_confidence <change_dir> 4.1 "+4.1" "calc-score"` SHALL be called
- **AND** a `confidence` event SHALL be appended to `.history.jsonl`

### Requirement: calc-score.sh Accessor Migration

`lib/calc-score.sh` SHALL migrate its direct `.status.yaml` reads (currently `grep`/`sed` for `certain:` and `score:`) to use `get_confidence` from stageman. The Assumptions table parsing (awk-based markdown parsing) MAY remain awk-based since it parses markdown, not YAML.

#### Scenario: Previous values read via stageman accessor
- **GIVEN** `calc-score.sh` needs previous `certain` and `score` values for carry-forward and delta
- **WHEN** it reads from `.status.yaml`
- **THEN** it SHALL use `get_confidence <file>` and parse the output
- **AND** NOT use direct grep/sed on the status file

## Downstream: Skill Prompt Integration

### Requirement: Driver Parameter in Skill Transitions

All skill prompts that invoke `lib/stageman.sh` write commands for state transitions SHALL pass a `driver` parameter identifying the skill.

| Skill | Driver string |
|-------|--------------|
| `/fab-new` | `fab-new` |
| `/fab-continue` | `fab-continue` |
| `/fab-ff` | `fab-ff` |
| `/fab-fff` | `fab-fff` |
| `/fab-clarify` | `fab-clarify` |

#### Scenario: fab-continue spec transition
- **GIVEN** `/fab-continue` completes spec generation
- **WHEN** it calls `stageman.sh transition <file> brief spec`
- **THEN** it SHALL pass `fab-continue` as the driver: `stageman.sh transition <file> brief spec fab-continue`

#### Scenario: fab-new brief activation
- **GIVEN** `/fab-new` initializes a new change
- **WHEN** it sets `brief: active` in `.status.yaml`
- **THEN** it SHALL pass `fab-new` as the driver: `stageman.sh set-state <file> brief active fab-new`

### Requirement: Command Logging at Skill Invocation

All `/fab-*` skill prompts SHALL call `stageman.sh log-command` at the start of invocation (after preflight succeeds), passing the change directory, skill name, and any arguments.

#### Scenario: fab-continue invoked without arguments
- **GIVEN** user runs `/fab-continue`
- **WHEN** the skill starts execution after preflight
- **THEN** it SHALL call `stageman.sh log-command <change_dir> "fab-continue"`

#### Scenario: fab-continue invoked with reset argument
- **GIVEN** user runs `/fab-continue spec`
- **WHEN** the skill starts execution after preflight
- **THEN** it SHALL call `stageman.sh log-command <change_dir> "fab-continue" "spec"`

### Requirement: Review Verdict Logging

`/fab-continue` review behavior SHALL call `stageman.sh log-review` after determining the review verdict.

#### Scenario: Review passes
- **GIVEN** review behavior completes with all checks passing
- **WHEN** the verdict is determined
- **THEN** it SHALL call `stageman.sh log-review <change_dir> "passed"`

#### Scenario: Review fails with rework
- **GIVEN** review behavior identifies failures
- **WHEN** the user selects a rework option (e.g., "revise-tasks")
- **THEN** it SHALL call `stageman.sh log-review <change_dir> "failed" "revise-tasks"`

## Design Decisions

1. **yq for .status.yaml Only, awk for workflow.yaml**
   - *Why*: `workflow.yaml` is a controlled schema file with static structure — existing awk parsing is reliable. `.status.yaml` is increasingly complex (nested `stage_metrics`, flow-style values) and needs yq's capabilities. Splitting avoids unnecessary migration work.
   - *Rejected*: Full yq migration including schema queries — unnecessary complexity for a static file.

2. **Automatic Metrics Side-Effects on State Transitions**
   - *Why*: Embedding metrics updates inside `set_stage_state` and `transition_stages` ensures every state change is tracked without requiring callers to make separate calls. Prevents data gaps when skills forget to log.
   - *Rejected*: Manual metrics calls from skill prompts — error-prone, requires updating every skill, guaranteed to have gaps.

3. **Hybrid Storage: stage_metrics in .status.yaml, Events in .history.jsonl**
   - *Why*: stage_metrics are structured (keyed by stage, fixed fields) and queried alongside other status data. Events are append-only, variable-length, and never queried mid-pipeline. YAML array appends are fragile with any tool; JSONL appends are trivially `echo >> file`.
   - *Rejected*: All in .status.yaml (YAML array appends are error-prone). All in JSONL (would need to reconstruct stage_metrics from events).

4. **Driver as Freeform String**
   - *Why*: A fixed enum needs updates every time a skill is added or renamed. Freeform strings are forward-compatible and self-documenting.
   - *Rejected*: Enum validation — maintenance burden without meaningful safety benefit since driver values are set by trusted skill prompts.

5. **Flow-Style YAML for stage_metrics**
   - *Why*: Each stage fits on one line: `{started_at: "...", completed_at: "...", driver: fab-continue, iterations: 1}`. Max 6 lines for 6 stages. Block style would add 4+ lines per stage.
   - *Rejected*: Block style — inflates the file, reduces scannability.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | yq (Mike Farah Go v4) as the parser | De facto standard, single static binary, most documented. Brief confirmed. |
| 2 | Confident | Full migration of all stageman `.status.yaml` functions at once | Avoids mixed awk/yq maintenance. Brief explicitly specified full migration. |
| 3 | Confident | Schema query functions remain awk-based | Static controlled file, reliable existing parsing, no benefit from yq migration. |
| 4 | Confident | JSONL for `.history.jsonl` | Append-only data is JSONL's natural fit; YAML array appends are fragile. Brief confirmed. |
| 5 | Confident | Freeform driver strings, not an enum | Forward-compatible, no enum maintenance. Brief confirmed. |
| 6 | Confident | Flow/inline YAML style for `stage_metrics` | Compact representation, max 6 lines in status file. Brief confirmed. |
| 7 | Certain | yq dependency acceptable under constitution | Constitution I amended (v1.1.0) to allow single-binary utilities. No longer a concern. |
<!-- clarified: yq dependency resolved — constitution amended to explicitly allow single-binary utilities like yq and gh -->

7 assumptions made (6 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

Q: Should `validate_status_file` also validate the `stage_metrics` block?
A: No — stage_metrics is internal bookkeeping written only by stageman itself. Trust the writer; keeps validation simple.

Q: Should `get_stage_metrics` handle missing `stage_metrics` block gracefully (old status files)?
A: Return empty output (no lines, exit 0). Backward-compatible with pre-migration status files.

Q: Does yq dependency clash with constitution's "no CLI binaries" rule?
A: Yes — constitution I amended to v1.1.0. Now allows single-binary utilities (yq, gh) that require no runtime or library installation. Assumption #7 reclassified from Tentative to Certain.
