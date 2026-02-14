# Spec: Stage Metrics, History Tracking, and stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Created**: 2026-02-14
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Capture per-token model usage metrics in `.history.jsonl` — current hooks do not expose reliable token counts.
- Redesign the workflow stage graph or SRAD scoring formula — this change extends observability and status plumbing only.

## Fab Workflow: Status Observability

### Requirement: Status file SHALL track per-stage operational metrics

`.status.yaml` MUST include a top-level `stage_metrics` map keyed by stage id (`brief`, `spec`, `tasks`, `apply`, `review`, `hydrate`). Each present stage entry SHALL support `started_at`, `completed_at`, `driver`, and `iterations` fields.

#### Scenario: Newly started stage records start metadata

- **GIVEN** `spec` is transitioned to `active`
- **WHEN** `lib/stageman.sh set-state` or `lib/stageman.sh transition` performs the mutation
- **THEN** `.status.yaml` contains `stage_metrics.spec.started_at` with an ISO 8601 timestamp
- **AND** `.status.yaml` contains `stage_metrics.spec.driver`
- **AND** `.status.yaml` contains `stage_metrics.spec.iterations` incremented to at least `1`

### Requirement: Stage completion SHALL record completion metadata

When a stage state is set to `done`, stageman MUST set `stage_metrics.<stage>.completed_at` to the transition timestamp without removing previously recorded `started_at`, `driver`, or `iterations` fields.

#### Scenario: Completing a stage preserves prior metadata

- **GIVEN** `stage_metrics.brief` already includes `started_at`, `driver`, and `iterations: 1`
- **WHEN** `brief` is transitioned from `active` to `done`
- **THEN** `stage_metrics.brief.completed_at` is written
- **AND** `stage_metrics.brief.started_at` remains unchanged
- **AND** `stage_metrics.brief.iterations` remains `1`

### Requirement: Iterations SHALL represent re-entry count

Each transition of a stage into `active` MUST increment `stage_metrics.<stage>.iterations` by exactly one. First activation SHALL set `iterations: 1`.

#### Scenario: Rework increments iteration count

- **GIVEN** `review` previously failed and `apply` is set back to `active`
- **WHEN** apply is re-entered
- **THEN** `stage_metrics.apply.iterations` increments from `1` to `2`

### Requirement: Workflow command history SHALL be append-only JSONL

Each change directory MUST maintain `.history.jsonl` as append-only JSON objects, one event per line, with required fields `ts`, `event`, and `outcome` (`success` or `error`).

#### Scenario: Command invocation is appended

- **GIVEN** a change has an existing `.history.jsonl`
- **WHEN** `/fab-continue` is invoked for that change
- **THEN** exactly one new line is appended
- **AND** the new JSON object includes `event: "command"`, `cmd`, optional `args`, and `outcome: "success"`
- **AND** previous lines are unchanged

#### Scenario: Failed command invocation is still recorded

- **GIVEN** a fab command invocation fails before mutating stage state
- **WHEN** the invocation terminates with an error
- **THEN** `.history.jsonl` appends a `command` event for that attempt
- **AND** the event includes `outcome: "error"`

<!-- clarified: command history will log both successful and failed invocations with explicit outcome field -->

### Requirement: Confidence and review outcomes SHALL be logged

Confidence recomputation and review decisions MUST be recorded as explicit history events.

#### Scenario: Confidence event captured after score recomputation

- **GIVEN** `lib/calc-score.sh` computes a new score
- **WHEN** scoring completes successfully
- **THEN** `.history.jsonl` includes an event with `event: "confidence"`, `score`, `delta`, and `trigger`

#### Scenario: Review result event captured

- **GIVEN** review behavior reaches a pass or fail verdict
- **WHEN** verdict is persisted in `.status.yaml`
- **THEN** `.history.jsonl` includes an event with `event: "review"` and `result` (`passed` or `failed`)

## Fab Workflow: stageman yq Migration

### Requirement: stageman SHALL use yq for status read operations

Status accessors in `fab/.kit/scripts/lib/stageman.sh` (`get_progress_map`, `get_checklist`, `get_confidence`, and future nested accessors) MUST use `yq` queries rather than regex-based YAML parsing.

#### Scenario: Nested metrics are readable without custom regex

- **GIVEN** `.status.yaml` contains nested `stage_metrics` data
- **WHEN** stageman reads progress, checklist, confidence, and stage metrics
- **THEN** values are returned accurately from YAML structure
- **AND** no accessor requires hardcoded line-order assumptions

### Requirement: stageman SHALL use yq for status writes

All write functions (`set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`) MUST use `yq` mutations, preserve unrelated YAML fields, and refresh `last_updated`.

#### Scenario: Write operation preserves unrelated blocks

- **GIVEN** `.status.yaml` contains `progress`, `checklist`, `confidence`, and `stage_metrics`
- **WHEN** `set-checklist completed 3` is executed
- **THEN** only `checklist.completed` and `last_updated` change
- **AND** `progress`, `confidence`, and `stage_metrics` remain unchanged

### Requirement: stageman SHALL expose stage metrics and history helpers

`stageman.sh` MUST provide helper functions for stage metrics and history logging, including `get_stage_metrics`, `set_stage_metric`, `log_command`, `log_confidence`, and `log_review`.

#### Scenario: Command logging helper appends valid JSON

- **GIVEN** `log_command` is called with command metadata
- **WHEN** the function writes to `.history.jsonl`
- **THEN** the appended line is valid JSON
- **AND** includes `ts`, `event`, and command fields

### Requirement: Status validation SHALL be yq-backed

`validate_status_file` MUST validate required fields and allowed stage states using `yq`-based extraction instead of line-based parsing.

#### Scenario: Invalid state is rejected

- **GIVEN** `.status.yaml` contains `spec: broken_state`
- **WHEN** `validate_status_file` runs
- **THEN** validation fails with non-zero exit
- **AND** emits a clear error describing the invalid state

### Requirement: Missing yq SHALL fail fast with guidance

When `yq` is unavailable in the runtime environment, stageman and dependent scripts MUST fail immediately with a clear error and installation guidance, and MUST NOT fall back to awk/grep parsing.

#### Scenario: Runtime without yq exits clearly

- **GIVEN** `yq` is not installed or not on `PATH`
- **WHEN** `lib/stageman.sh` or `lib/calc-score.sh` is invoked
- **THEN** execution exits non-zero before reading or writing `.status.yaml`
- **AND** stderr includes a clear message that `yq` is required and how to install it

<!-- clarified: yq availability will be enforced with fail-fast behavior and explicit install guidance; no awk fallback -->

## Fab Workflow: Integration Updates

### Requirement: calc-score SHALL delegate writes and emit confidence history

`fab/.kit/scripts/lib/calc-score.sh` MUST write confidence via stageman write APIs and append a confidence history event after successful recomputation.

#### Scenario: Score update and history entry happen together

- **GIVEN** `calc-score.sh` computes confidence score `3.8` with delta `-0.6`
- **WHEN** script execution succeeds
- **THEN** `.status.yaml` confidence block reflects `3.8`
- **AND** `.history.jsonl` includes a `confidence` event with `score: 3.8` and `delta: "-0.6"`

### Requirement: Template and schema SHALL include metrics-compatible structure

`fab/.kit/templates/status.yaml` and `fab/.kit/schemas/workflow.yaml` MUST document the new `stage_metrics` block and validation expectations.

#### Scenario: New changes start with metrics-ready status structure

- **GIVEN** `/fab-new` creates a new change from template
- **WHEN** `.status.yaml` is generated
- **THEN** it contains an initialized `stage_metrics` map
- **AND** schema validation accepts the file without manual edits

### Requirement: Planning and execution skills SHALL pass driver identity and log command events

Skills that mutate stage state (`/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`, `/fab-switch`) MUST pass a driver identifier to stageman transitions and append command history at invocation start.

#### Scenario: Driver identity is visible in stage metrics

- **GIVEN** `/fab-switch` activates a change
- **WHEN** the corresponding stage is set `active`
- **THEN** `stage_metrics.<stage>.driver` stores `fab-switch`
- **AND** `.history.jsonl` has a matching `command` event for the invocation

### Requirement: Constitution SHALL explicitly allow runtime YAML tooling

The constitution MUST be amended to explicitly permit single-binary runtime tooling used by kit scripts (including `yq`) while preserving the prohibition on package-manager dependencies and build steps.

#### Scenario: Governance text aligns with yq dependency

- **GIVEN** this change introduces `yq` as a runtime dependency for stageman
- **WHEN** `fab/constitution.md` is updated
- **THEN** the governance text explicitly permits this dependency class
- **AND** no contradiction remains between constitution principles and implementation requirements

<!-- clarified: constitution will be amended to explicitly allow single-binary runtime tooling like yq for kit scripts -->

## Design Decisions

1. **YAML parser implementation**: Use Mike Farah `yq` v4 for all `.status.yaml` reads/writes.
   - *Why*: Nested map support, predictable mutation semantics, concise write expressions.
   - *Rejected*: Keep awk parsing for reads or writes (fragile for nested structures), Python wrappers (adds runtime/tooling variance).
2. **History format**: Use append-only `.history.jsonl`.
   - *Why*: Atomic append behavior and simple line-by-line tooling for operational event logs.
   - *Rejected*: YAML arrays (append/update complexity and merge noise).
3. **Driver field typing**: Keep `driver` as freeform string.
   - *Why*: Forward-compatible with new skills/scripts without enum churn.
   - *Rejected*: Strict enum (requires frequent schema/script updates).
4. **Metrics placement**: Keep `stage_metrics` in `.status.yaml` and event stream in `.history.jsonl`.
   - *Why*: Fast stage snapshot in one file, richer temporal trace in append-only log.
   - *Rejected*: Put everything in one file (either too verbose status YAML or too little snapshot fidelity in log-only model).
5. **Governance alignment**: Amend constitution to explicitly allow single-binary runtime tools used by kit scripts.
   - *Why*: Removes ambiguity while keeping original intent against package-manager and build-step coupling.
   - *Rejected*: Per-change exceptions (policy drift risk), abandoning yq migration (blocks nested YAML improvements).

## Clarifications

### Session 2026-02-14

- Q: Should yq governance be handled by constitution amendment, per-change exception, or dropping yq migration?
- A: Amend constitution to explicitly allow single-binary runtime tooling used by kit scripts (including yq).
- Q: Should command history include failed invocations?
- A: Yes. Log both successful and failed invocations with `outcome` set to `success` or `error`.
- Q: What should happen when yq is missing at runtime?
- A: Fail fast with a clear error and install guidance; do not fall back to awk/grep parsing.
