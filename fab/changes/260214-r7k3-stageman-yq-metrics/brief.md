# Brief: Stage Metrics, History Tracking & Stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Enhance .status.yaml with operational metrics (timing, drivers, iterations, confidence evolution, command logs) and migrate stageman from awk-based YAML parsing to `yq`. Analyzed from a live session — identified that status.yaml tracks *what* (states, scores) but not *how* or *when*.

## Why

`.status.yaml` captures stage states and confidence scores, but no operational data — when stages started/finished, which skill drove them, how many rework iterations occurred, or how confidence evolved over time. This data is needed to understand change dynamics and workflow efficiency. The current awk-based stageman write functions work but are fragile (30+ lines per mutation, regex YAML parsing) and can't easily handle the nested `stage_metrics` structure needed for 2D per-stage data. Migrating to `yq` enables flow-style compact representations and reduces each mutation to a one-liner.

## What Changes

**Schema additions:**

New `stage_metrics` block in `.status.yaml` (flow/inline style, one row per stage):

```yaml
# existing fields unchanged (name, created, progress, checklist, confidence, last_updated)
stage_metrics:
  brief: {started_at: "2026-02-14T07:11:18+05:30", completed_at: "2026-02-14T07:11:45+05:30", driver: fab-new, iterations: 1}
  spec:  {started_at: "2026-02-14T07:20:00+05:30", driver: fab-continue, iterations: 1}
  # absent stages = not yet started; max 6 rows
```

| Field | Type | Set when |
|-------|------|----------|
| `started_at` | ISO 8601 | state → `active` |
| `completed_at` | ISO 8601 | state → `done` |
| `driver` | freeform string | state → `active` (skill identity) |
| `iterations` | positive int | Incremented on each state → `active` (1 = first, 2+ = rework) |

New `.history.jsonl` — append-only JSONL event log (one JSON object per line):

```jsonl
{"ts":"2026-02-14T07:11:18+05:30","event":"command","cmd":"fab-new","args":"calc-score dev setup"}
{"ts":"2026-02-14T07:20:00+05:30","event":"command","cmd":"fab-continue"}
{"ts":"2026-02-14T07:25:00+05:30","event":"confidence","score":4.1,"delta":"-0.9","trigger":"fab-continue/spec"}
{"ts":"2026-02-14T08:00:00+05:30","event":"review","result":"failed","rework":"revise-tasks"}
{"ts":"2026-02-14T08:15:00+05:30","event":"review","result":"passed"}
```

Event types: `command` (all fab-* invocations), `confidence` (score recomputations), `review` (pass/fail outcomes). All share `ts` + `event` fields.

**Stageman yq migration (full):**
- Migrate all stageman accessor functions (`get_progress_map`, `get_checklist`, `get_confidence`) from awk/grep to yq
- Migrate all write functions (`set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`) from awk to yq
- Migrate `validate_status_file` to use yq for schema checks
- Add new functions: `get_stage_metrics`, `set_stage_metric`, `log_command`, `log_confidence`, `log_review`
- Add automatic stage_metrics side-effects to state transition functions

**Downstream changes:**
- `calc-score.sh` — call `log_confidence` after score computation; use `set_confidence_block` instead of inline awk
- Status.yaml template — add `stage_metrics: {}` block
- Skill prompts — pass driver identity on transitions, call `log_command` at invocation start

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) yq dependency, stageman function migration, new stage_metrics and history subsystems
- `fab-workflow/change-lifecycle`: (modify) Document stage_metrics fields and .history.jsonl event types
- `fab-workflow/planning-skills`: (modify) Skills now pass driver parameter and log commands
- `fab-workflow/execution-skills`: (modify) Same — driver and log_command integration

## Impact

- **`fab/.kit/scripts/lib/stageman.sh`** (or `_stageman.sh`) — primary target; full rewrite of read/write/validation functions
- **`fab/.kit/scripts/lib/calc-score.sh`** (or `_calc-score.sh`) — refactored to use stageman write API + log_confidence
- **`fab/.kit/templates/status.yaml`** — add stage_metrics block
- **`fab/.kit/schemas/workflow.yaml`** — document stage_metrics schema
- **`src/stageman/test.sh`** and **`test-simple.sh`** — extensive test additions
- **All skill prompts** (fab-new, fab-continue, fab-ff, fab-fff, fab-clarify, fab-switch) — pass driver, call log_command
- **New runtime dependency**: `yq` (Mike Farah Go v4) — single binary, no package manager

## Open Questions

None — all design decisions were resolved during planning:
- Hybrid storage (stage_metrics in status.yaml, logs in .history.jsonl)
- yq as parser (full migration, not incremental)
- Flow/inline style for stage_metrics
- JSONL for history (not YAML)
- Token tracking deferred (hooks don't expose data)
- Freeform driver strings (not enum)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | yq (Mike Farah Go v4) as the parser, not Python yq or dasel | Go version is the de facto standard, single static binary, most documented |
| 2 | Confident | Full migration of all stageman functions (not incremental) | Avoids mixed awk/yq maintenance; consistent codebase |
| 3 | Confident | JSONL for .history.jsonl instead of YAML | Append-only data is JSONL's sweet spot; YAML array appends with any tool are fragile |
| 4 | Confident | Freeform driver strings, not a fixed enum | Enum would need updates with every new skill; freeform is forward-compatible |
| 5 | Tentative | yq dependency acceptable under constitution's "no CLI binaries" rule | Constitution targets package managers and build steps; yq is a single binary like gh/git. May need constitution amendment or explicit exemption. |
<!-- assumed: yq dependency acceptable — constitution says "no CLI binaries" but project already uses gh, git, awk; yq is comparable -->

5 assumptions made (4 confident, 1 tentative). Run /fab-clarify to review.
