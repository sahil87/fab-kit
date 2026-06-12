---
description: "Workflow schema authority — the Go state machine (`internal/status` transitions + `internal/statusfile` stage order/progress; declarative `workflow.yaml` retired in c5tr): 6-stage pipeline, states, transitions, validation rules; `.status.yaml` `plan:` (`## Requirements`-aware), `confidence:` (indicative retired), and lazy `true_impact:` block schemas (incl. the `tests` sub-block + render-time `impl` residual, 7t5a); `fab impact` and `fab pr-meta` helper subcommands (rj31); allowed-states-enforced transition targets, `fab score --check-gate` non-zero gate-fail exit, iterations-preserving reset cascade (k4ge); `fab score` normal-mode hard-fail on load/persist/read errors (hv7t)"
---
# Schemas

**Domain**: pipeline

## Overview

The single source of truth for the Fab workflow — stages, states, transitions, and validation rules — is the **Go state machine**: `src/go/fab/internal/status` (event-keyed transitions and their side-effects) and `src/go/fab/internal/statusfile` (stage order, progress schema, `.status.yaml` encode/decode). All scripts and skills query it via the `fab status` / `fab preflight` CLI surface rather than hardcoding workflow knowledge.

The former declarative schema artifact `src/kit/schemas/workflow.yaml` was **retired in 260612-c5tr** (file deleted; the `src/kit/schemas/` directory is gone): nothing consumed it — no script, skill, or binary parsed it — and it had silently drifted a full pipeline generation, still describing the pre-1.10.0 7-stage pipeline with a `spec` stage while `docs/specs/user-flow.md` called it the source of truth. That user-flow line now points at the Go state machine. It was retired rather than regenerated because a regenerated declarative artifact would re-create the same unenforced drift surface. (A frozen pre-retirement copy survives as a benchmark fixture at `src/benchmark/fixtures/workflow.yaml`, itself flagged as a deletion candidate — zero consumers.)

## What the State Machine Defines

1. **States** — All valid progress values (`pending`, `active`, `ready`, `done`, `failed`, `skipped`)
   - Each state has an ID, a display symbol, and terminal semantics
   - `ready` means "stage work product exists, eligible for advancement or clarification" (non-terminal)
   - `skipped` means "stage intentionally bypassed" (terminal, symbol `⏭`). Allowed for all stages except intake
   - Terminal states (`done`, `skipped`) cannot transition without explicit reset

2. **Stages** — The workflow pipeline in execution order — 6 stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` stage was removed in qszh, and the `spec` stage in j6cs; plan generation is an apply-internal sub-step that produces a unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`). `allowedStates` contains neither a `tasks` nor a `spec` key, and `isValidStage("tasks")`/`isValidStage("spec")` both return false. `validateStage` returns a deprecation error for either removed stage (`spec` → `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`).
   - Each stage has: ID, name, artifact, description, requirements, initial state, allowed states, commands
   - Stages execute in sequence with dependency validation

3. **Transitions** — Valid state changes for each stage, event-keyed (event, from, to)
   - Default rules apply to all stages
   - Stage-specific overrides (e.g., `review` allows `fail` event)
   - Each transition is triggered by an event command (`start`, `advance`, `finish`, `reset`, `fail`, `skip`)
   - `skip` event: `{pending,active} → skipped` with forward cascade (all downstream pending → skipped). No auto-activate
   - `reset` accepts `skipped` as a source state (`skipped → active` with downstream cascade to `pending`)
   - **Target-state validation (k4ge)**: `lookupTransition` validates the resolved target state against the stage's allowed states (a `validateTarget` helper applied to both the stage-override and the default resolution path). A schema-forbidden combination — `advance ship`/`advance review-pr` (target `ready`) or `skip intake` (target `skipped`) — exits non-zero with `Cannot {event} stage '{stage}' — target state '{state}' is not allowed for this stage` and writes nothing, instead of writing a state that permanently bricks `fab preflight` ("State 'ready' not allowed for stage ship"). The schema is the single constraint source, so any future forbidden combo is closed automatically. The now-unreachable `stageTransitions["review-pr"]["advance"]` override row in `status.go` is a recorded deletion candidate (k4ge plan) — removing it is byte-identical behavior since the default `advance` row produces the same rejection
   - **Cascade preserves `iterations` (k4ge)**: when the `reset`/`skip` cascade sets a stage to `pending`/`skipped`, a `stage_metrics` entry with `iterations > 0` is kept with only its `iterations` counter (timing fields `started_at`/`driver`/`completed_at` cleared; the next activation rewrites them); zero-iteration entries are still deleted, so no empty `{}` entries linger. This keeps `stage_metrics.review.iterations` truthful across the rework choreography's `fail review` + `reset apply`, making the cycle count `fab pr-meta` reports real. Preservation is uniform across all stages, not review-only. See [change-lifecycle.md](change-lifecycle.md) for full `stage_metrics` semantics

4. **Progression** — How to navigate the workflow
   - Current stage detection: first `active` or `ready` stage, or first `pending` after last `done`/`skipped`, or `review-pr` if all done/skipped (`CurrentStage`'s all-done fallback — this doc previously mis-stated it as `hydrate`; corrected in k4ge)
   - Next stage calculation: first `pending` stage with satisfied dependencies (prerequisites `done` or `skipped`)
   - Completion check: `hydrate` is `done` or `skipped`

5. **Validation** — Rules for `.status.yaml` correctness
   - Exactly 0-1 active stages
   - States must be in `allowed_states` for that stage
   - Prerequisites must be satisfied before activation
   - Terminal states require explicit reset

6. **Stage numbers** — Display numbering for status output (1-indexed positions)

## Querying the State Machine

Neither scripts nor skills parse a schema file — all workflow queries go through the CLI surface:

- `fab status <event> <change> <stage>` — the event commands (`start`, `advance`, `finish`, `reset`, `fail`, `skip`) validate transitions inside `internal/status` and reject invalid ones with actionable errors
- `fab preflight [<change>]` — emits validated `stage` / `display_stage` / `display_state` / `progress` fields derived by the state machine

For the full CLI reference, see `$(fab kit-path)/skills/_cli-fab.md` (headline command families inlined in `_preamble.md` § Common fab Commands).

## Design Principles

1. **Single Source of Truth** — one canonical definition in code, queried by all consumers via the CLI
2. **Validated** — transitions are enforced at runtime by the event commands; invalid transitions are rejected, never silently coerced
3. **Tested over declared** (c5tr) — the schema lives where it cannot drift: `internal/status`/`internal/statusfile` plus their Go test suite. The declarative-artifact approach was retired after `workflow.yaml` proved unenforceable — nothing consumed it, so nothing noticed it describing a retired pipeline

## `.status.yaml` Plan Block (qszh)

Every `.status.yaml` SHALL contain a `plan:` block describing the apply-stage artifact (`plan.md`):

```yaml
plan:
  generated: false      # bool — true after first plan.md write
  task_count: 0         # int — count of - [ ] + - [x] items in ## Tasks section
  acceptance_count: 0   # int — count of - [ ] + - [x] items in ## Acceptance section
  acceptance_completed: 0  # int — count of - [x] items in ## Acceptance section
```

This block replaces the prior `checklist:` block. Field rename: `total → acceptance_count`, `completed → acceptance_completed`. New field: `task_count`. Removed field: `path` (location is fixed at change root).

The `progress` block contains exactly 6 keys (no `tasks` or `spec` key):

```yaml
progress:
  intake: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
```

`StageOrder` is `["intake", "apply", "review", "hydrate", "ship", "review-pr"]` (length 6). `StageNumber("apply") == 2`; `NextStage("intake")` returns `"apply"`. An orphan `progress.spec` key on an un-migrated `.status.yaml` is tolerated on load (`Validate()` skips it; `GetProgressMap()` omits it; a subsequent `Save` may preserve it via raw-node passthrough) — only the `1.9.7-to-1.10.0` migration removes it. The `set-acceptance` CLI command (`fab status set-acceptance <change> <field> <value>`) updates `plan:` block fields; the legacy `set-checklist` errors immediately with a pointer to `set-acceptance`.

The `Load()` function is tolerant of legacy `.status.yaml` files: it upgrades a `checklist:` block to a `plan:` raw mapping with field migration (`completed → acceptance_completed`, `total → acceptance_count`) and drops `checklist:` when both blocks coexist. The `1.8.0-to-1.9.0.md` migration rewrites in-flight `.status.yaml` files to the new schema (drops `progress.tasks`, replaces `checklist:` with `plan:`); the `1.9.7-to-1.10.0.md` migration drops `progress.spec`; see [migrations.md](../distribution/migrations.md).

As of j6cs the apply-stage `plan.md` carries a `## Requirements` section (RFC-2119 + GIVEN/WHEN/THEN, the requirement discipline absorbed from the removed `spec.md`) alongside `## Tasks` and `## Acceptance` — these three `##` headings are the stable parser contract.

## `.status.yaml` Confidence Block (`indicative` retired in j6cs)

The `confidence` block holds SRAD scoring: `certain`, `confident`, `tentative`, `unresolved` counts and a derived `score` (0.0–5.0). The `confidence.indicative` flag was **retired in j6cs** — `encodeConfidence` no longer writes it, and `SetConfidence`/`SetConfidenceFuzzy` dropped their `indicative` parameter. The struct keeps a decode-tolerant `Indicative *bool` field so a legacy `indicative: true` key on an un-migrated/archived file round-trips harmlessly (load succeeds, the rest of the block decodes normally, and no write re-emits the key). The `--indicative` CLI flag on `set-confidence`/`set-confidence-fuzzy` is retained for one release as an accepted-but-ignored no-op. `fab score` reads `intake.md` only (the sole scoring source); the migration leaves any `confidence.indicative` key on disk untouched.

**`--check-gate` exit contract (k4ge)**: `fab score --check-gate` exits non-zero when the gate result is `fail` — the gate YAML (`gate: fail`, score, threshold, counts) stays on stdout for parsing, and the error (`intake gate failed: score {x} below threshold {y}`) reaches stderr via main's handler as `ERROR: ...`. Exit 0 on `gate: pass`. Previously the command always exited 0 regardless of gate result, so `/fab-ff`/`/fab-fff` could not detect a failed intake gate via the documented exit-code contract — the pipeline's only safety gate was silently bypassable. The Go fix made the long-standing doc rows (`_preamble.md` § Common fab Commands, `_cli-fab.md` § fab score, `_pipeline.md` Pre-flight) true without editing them.

**Normal-mode failure surfacing (hv7t)**: `fab score <change>` (normal mode) hard-errors instead of printing a score while silently persisting nothing. `score.Compute` returns — and `cmd/fab/score.go`'s `RunE` surfaces via main's handler, the same stderr `ERROR: ...` + non-zero routing as the k4ge gate-fail exit — failures of: the `.status.yaml` load (previously the entire write-back block was skipped silently and `change_type` defaulted to `feat`), the confidence write-back (`SetConfidence`/`SetConfidenceFuzzy`, previously `_ =`-discarded), the `.history.jsonl` confidence-log append, and the `intake.md` read. The YAML report appears on stdout only when scoring *and* persistence succeed. The intake read is honest end-to-end: `CheckGate` and `Compute` read `intake.md` themselves via `os.ReadFile` (whole-file, IsNotExist-classified — mz4q F02/F06) and `countGrades(content)` parses the already-read content via `lines.Split` instead of a `bufio.Scanner` — no 64KB truncation is possible at any point, so a truncated Assumptions table can no longer flip the gate from fail to pass by dropping graded rows (hv7t F09), and a read failure is distinguishable from an intake with no Assumptions table (zero counts, nil error). Within `Compute`, the load-mutate-save cycle runs under the mz4q cross-process status lock with `ComputeWithStatus` as the shared single-load entry point; hv7t makes that path truthful (load failure, `persist confidence:`, `log confidence:` all hard errors). The PostToolUse hook caller (`cmd/fab/hook.go`) keeps its `if err == nil` guard unchanged — the hook path stays best-effort with zero hook changes.

## `.status.yaml` `true_impact` Block (ogf2)

`.status.yaml` MAY contain a top-level optional `true_impact` block that records the merge-base-to-HEAD line-count impact of the change at apply-finish and hydrate-finish. The block is created lazily on first computation — there is no template placeholder, and existing `.status.yaml` files without the block remain valid.

```yaml
true_impact:
    added: 142
    deleted: 38
    net: 104
    excluding:               # only present when true_impact_exclude is non-empty
        added: 87
        deleted: 38
        net: 49
    tests:                   # only present when test_paths is non-empty (7t5a)
        added: 60
        deleted: 0
        net: 60
    computed_at: 2026-05-07T14:32:00Z
    computed_at_stage: apply
```

Field semantics:
- `added`, `deleted`, `net` — raw `git diff --shortstat <merge-base>...HEAD` results. `net = added - deleted` (signed).
- `excluding` — same fields with `:(exclude)<pattern>` pathspec applied for each entry in `fab/project/config.yaml` `true_impact_exclude` (sister change asvz; default scaffold `[fab/, docs/]`). Sub-block omitted entirely when `true_impact_exclude` is absent/null/empty (consumer treats "no excludes" identically to "excluding == raw"; emitting a duplicate sub-block adds no signal).
- `tests` — same fields, attributing the test portion of the change (7t5a). Computed by a third `git diff --shortstat` pass whose pathspec combines the `test_paths` includes with the SAME `:(exclude)<pattern>` args as the `excluding` pass — so test lines are counted *within the scaffolding-excluded universe* (a test fixture under an excluded path is not double-counted). Each `test_paths` include is applied as a `:(glob)<pattern>` magic pathspec so `**` matches across directory boundaries. When `true_impact_exclude` is empty the test pass runs with the includes alone (tests attributed within the raw universe). Sub-block omitted entirely (lazily) when `test_paths` is absent/null/empty — behavior then collapses to today's single-number display. See [configuration.md](../_shared/configuration.md) for the `test_paths` config field.
- `computed_at` — RFC 3339 UTC timestamp.
- `computed_at_stage` — pipeline stage at which the snapshot was taken: `apply` or `hydrate`.

**No `impl` field is stored.** The implementation residual is `impl = max(0, total − tests)` *per component* (`added`/`deleted`/`net` each clamped independently, since the three-row display shows separate `+X / −Y` components and each must be non-negative on its own), where `total` is the scaffolding-excluded number (`excluding`, else raw when `true_impact_exclude` is empty). It is derived at RENDER TIME ONLY — the impact engine (`internal/impact/`), `.status.yaml`, and the `fab impact` YAML store only the *measured* passes (raw, `excluding`, `tests`), never a derived `impl`. This keeps the engine pure-measurement so no derived field can drift or go stale between the two diff passes; the cost is that the residual + clamp logic is implemented at both render sites (the `fab pr-meta` Impact line in `internal/prmeta/` — which renders the PR `## Meta` block for `/git-pr` as of rj31 — and `impactColumn()` in `internal/change/`). When the clamp triggers (a `test_paths` glob overlaps a `true_impact_exclude` path, over-counting `tests` relative to `total`), the render site emits a one-line stderr warning and never renders a negative impl.

**Engine surface** (7t5a): `impact.Result` gains a `Tests *Pair` field (alongside `Excluding *Pair`), nil when `test_paths` is empty. `Compute(repoDir, base, head, excludes, testPaths)` takes a trailing `testPaths`; `ComputeForRepo` reads both `cfg.TrueImpactExclude` and `cfg.TestPaths`. `statusfile.TrueImpact` gains `Tests *TrueImpactPair` (`yaml:"tests,omitempty"`); `encodeTrueImpact` emits the `tests` mapping after `excluding`, before `computed_at`; `WriteTrueImpact` copies `res.Tests` when non-nil. The `fab impact <base> <head>` CLI's `renderYAML` emits the `tests` sub-block only when present.

**Write path**: `WriteTrueImpact(statusPath, base, head, stage)` in `internal/status/true_impact.go` calls `impact.ComputeForRepo` (canonical math in `internal/impact/`) and writes the block via the existing `Save` flow. `status.Finish` invokes the helper for stages `apply` and `hydrate` only — invoked AFTER `applyMetricsSideEffect` and the file save, BEFORE post-hooks. **Best-effort**: on computation failure (e.g., no merge-base resolvable), the helper logs a one-line warning to stderr and returns nil — the stage transition never fails because of a `true_impact` write error. This matches the `fab log command` posture (telemetry hooks never become new failure modes) — a posture `fab log command` itself fully owns since 260612-ye8r: the CLI always exits 0 given valid usage (cobra arg-count errors exit non-zero before RunE), printing `Warning: fab log command: …` to stderr on any internal failure (no fab root, unresolvable explicit change arg, unwritable `.history.jsonl`), so the per-call-site `2>/dev/null || true` guard boilerplate is retired from `_preamble.md` and every skill file. `log review`/`log confidence`/`log transition` keep fail-loud non-zero exits (auto-logged by `fab status`/`fab score`, never called by skills directly).

**Helper subcommand**: `fab impact <base> <head>` is the canonical CLI for computing the block (consumed by `WriteTrueImpact`). It emits the same YAML schema (minus `computed_at_stage` — that is the caller's responsibility) on stdout, exits non-zero with an actionable stderr message on merge-base or `git diff` failure, and reads `true_impact_exclude` from `fab/project/config.yaml` to apply the same `excluding` rule. See `_cli-fab.md` for the full CLI reference.

**`fab pr-meta` subcommand** (rj31): `fab pr-meta <change> --type <type> [--issues "<space-joined IDs>"]` renders the complete `## Meta` block of a fab-generated PR as final markdown (table, `**Pipeline**:`, optional `**Issues**:`, multi-form `**Impact**:`), replacing the inlined `/git-pr` Step 3c formatting prose. It reuses `internal/impact` (`ComputeForRepo`) for the Impact line against an internally-resolved merge-base (HEAD vs `origin/main`, falling back to `origin/master`) rather than shelling to `fab impact`, and derives the `impl` residual at render time per the rule above. It is self-contained otherwise — reading `.status.yaml`, `plan.md` task checkboxes, and config (`true_impact_exclude`, `test_paths`, `project.linear_workspace`) directly. Non-zero exit (no fab context) or empty stdout signals `/git-pr` to omit the Meta block; `gh` failure degrades to plain-text Pipeline labels and a missing merge-base drops only the Impact line. Render logic lives in `internal/prmeta/`; see `_cli-fab.md` for the full CLI reference.

## `.status.yaml` Identity Fields

### `id` Field

The `id` field is a top-level field in `.status.yaml` containing the 4-character change ID (the `XXXX` component of the folder name). It is derived from the `name` at creation time and is immutable.

```yaml
id: x2tx
name: 260307-x2tx-status-symlink-pointer
created: 2026-03-07T16:54:29+05:30
```

The `id` field makes the change ID directly available from reading `.status.yaml` without needing to parse the folder name. This is especially useful when reading status via the `.fab-status.yaml` symlink — the consumer gets the ID from the file content rather than having to parse the symlink target path.

### `.fab-status.yaml` Symlink

`.fab-status.yaml` is a symlink at the repository root pointing to the active change's `.status.yaml`. It is the active change pointer — the replacement for the former `fab/current` text file. The symlink target is always a relative path: `fab/changes/{name}/.status.yaml`. See [change-lifecycle.md](change-lifecycle.md) for full lifecycle documentation.

Together with `.fab-runtime.yaml`, these two sibling files at the repo root form the complete ephemeral per-worktree state surface, scannable with a single glob.

## Ephemeral Runtime State

### Agent State — `.fab-runtime.yaml`

Agent runtime state lives in `.fab-runtime.yaml` at the repository root (gitignored). This file is NOT part of the workflow schema (distinct from the workflow state machine this doc describes), NOT initialized by templates, and NOT read by any workflow command. It is managed by Claude Code hook scripts via the `fab hook stop|session-start|user-prompt` subcommands.

**Schema and write pipeline**: See [runtime-agents.md](../runtime/runtime-agents.md) for the authoritative documentation. The file uses a top-level `_agents` map keyed by Claude's `session_id` (UUID from hook stdin) with `change`, `pid`, `tmux_server`, `tmux_pane`, and `transcript_path` as optional entry properties, plus a top-level `last_run_gc` timestamp that throttles an inline GC sweep. Entries populate regardless of active-change state, so agents running in discussion mode are tracked the same as change-associated agents.

Each worktree has its own repo root, so each gets its own `.fab-runtime.yaml` — no cross-worktree contention. External tools can read this file to detect agent idle state and correlate agents to panes without relying on timing heuristics.

## Future Enhancements

1. **Custom workflows** — Allow `fab/project/config.yaml` to override or extend the stage graph
2. **~~Conditional stages~~** — *(Partially addressed)* The `skipped` state and `skip` event now enable explicit stage bypassing via `fab status skip`. Skill-level orchestration (automatic skip based on change attributes) remains a future enhancement
3. **Parallel stages** — Multiple stages active simultaneously for different artifacts
4. **~~Stage hooks~~** — *(Shipped)* The `stage_hooks.{stage}.pre/post` config surface runs commands around `fab status start`/`finish` — live Go behavior, documented in `_cli-fab.md` § stage_hooks as of c5tr (pre blocks `start` on non-zero exit; post runs after `finish`'s save). See [change-lifecycle.md](change-lifecycle.md)
5. **State metadata** — Attach timestamps, user info, or exit codes to state transitions

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260612-ye8r-cli-single-sourcing-doc-conformance | 2026-06-12 | `fab log command` now fully owns the best-effort telemetry contract (binary-review B4, F28): the CLI subcommand always exits 0 given valid usage (cobra arg-count errors exit non-zero before RunE), wrapping `runLogCommand` and printing a one-line `Warning: fab log command: …` to stderr on any internal failure — the paths that previously exited 1 (FabRoot failure, an explicit change arg failing resolve, unwritable `.history.jsonl`) can no longer STOP a pipeline via a forgotten shell guard. The ":telemetry hooks never become new failure modes" posture note in this file is now literally true for `fab log command`. Call-site guard boilerplate (`2>/dev/null \|\| true`) retired from `_preamble.md` (Common-fab-Commands row, key-behaviors bullet, §2 step-4 template, failure rule) and the 5 skill files with explicit calls (`fab-help`, `fab-switch`, `fab-setup`, `fab-operator`, `fab-discuss`), plus `_cli-fab.md`'s fab-log section and the SPEC mirrors. `log review`/`confidence`/`transition` unchanged (fail-loud). `internal/log` untouched — the contract lives at the CLI layer (`cmd/fab/log.go`). |
| 260612-hv7t-scanner-truncation-sweep-score-truth-telling | 2026-06-12 | `fab score` truth-telling (binary-review batch B2/6). **Normal-mode failure surfacing**: `score.Compute` now returns — and the CLI exits non-zero with stderr detail, per the k4ge RunE routing — `.status.yaml` load failures (previously: write-back block silently skipped, `change_type` defaulted to `feat`), confidence write-back failures (`SetConfidence`/`SetConfidenceFuzzy`, previously `_ =`-discarded), `.history.jsonl` confidence-log append failures, and `intake.md` read failures; YAML on stdout only on full success. **Gate integrity**: `countGrades(content)` parses caller-read content via the new `internal/lines` helper's `Split` (composing with mz4q's F02 content-based signature and F06 classified `os.ReadFile` errors in `CheckGate`/`Compute`) — read failure distinguishable from a missing Assumptions table, truncation-driven fail→pass gate flips impossible. Hook caller's `if err == nil` guard untouched (hook path stays best-effort). New "Normal-mode failure surfacing" paragraph extending the k4ge `--check-gate` exit-contract section. |
| 260612-k4ge-cli-exit-contract-conformance | 2026-06-12 | CLI exit-code contract conformance (skills-audit batch 1/5). **Transition target-state validation**: `lookupTransition` now validates the resolved target against the stage's allowed states via a `validateTarget` helper (both override and default paths) — `advance ship`, `advance review-pr`, and `skip intake` are rejected non-zero with no write, instead of writing a forbidden state that permanently bricks `fab preflight`; the dead `stageTransitions["review-pr"]["advance"]` override row is a recorded deletion candidate. **`fab score --check-gate` exit contract**: exits non-zero on `gate: fail` (YAML intact on stdout, `intake gate failed: score {x} below threshold {y}` on stderr) — the single intake gate is now observable by `/fab-ff`/`/fab-fff`; exit 0 on pass. **Iterations-preserving cascade**: the `reset`/`skip` cascade keeps `stage_metrics` entries with `iterations > 0` (only timing fields cleared; zero-iteration entries still deleted), so review cycle counts survive the fail+reset rework choreography and `fab pr-meta` reports real cycles. Also corrected pre-existing drift in § Progression: the all-done routing fallback is `review-pr` (`CurrentStage`, status.go), not `hydrate`. |
| 260612-c5tr-scaffold-config-truth-srad-coherence | 2026-06-12 | **`workflow.yaml` retired — schema authority is the Go state machine** (skills-audit batch 4/5, Theme 5). Deleted `src/kit/schemas/workflow.yaml` (and with it the `src/kit/schemas/` directory): zero consumers, and the file had drifted a full pipeline generation — it still defined the pre-1.10.0 7-stage pipeline with `spec` while `docs/specs/user-flow.md:201` called it the source of truth. That line now points at `src/go/fab/internal/status` (transitions + side-effects) and `src/go/fab/internal/statusfile` (stage order + progress schema). Retired, not regenerated — a fresh declarative artifact would re-create the unenforced drift surface; code + tests are the schema. This doc's Overview / definition / query / design-principles sections rewritten accordingly (stale `statusman.sh` references swept to the `fab status`/`fab preflight` CLI surface). Future-enhancement "Stage hooks" marked shipped — the live `stage_hooks` config behavior is documented in `_cli-fab.md` § stage_hooks (see [change-lifecycle.md](change-lifecycle.md)). |
| 260604-rj31-mechanical-pr-meta | 2026-06-04 | Added the `fab pr-meta <change> --type <type> [--issues "..."]` subcommand documentation: renders the complete PR `## Meta` block (table, Pipeline, optional Issues, multi-form Impact) deterministically, reusing the `internal/impact` engine (`ComputeForRepo`) for the Impact line and deriving the render-time `impl` residual per the existing rule. The `impl` render-site note updated — the PR-side render site moved from `/git-pr` PR-body prose assembly to the `fab pr-meta` Impact line in `internal/prmeta/`; the `fab impact` "Helper subcommand" note no longer lists `/git-pr` Step 3c-impact as a consumer (`/git-pr` now delegates the whole Meta block to `fab pr-meta`, which uses the engine directly). No `.status.yaml` / `true_impact` / `workflow.yaml` schema changes — `fab pr-meta` is read-only over status/plan/config. |
| 260602-7t5a-true-impact-test-split | 2026-06-02 | Added an optional `tests` sub-block to the `.status.yaml` `true_impact` block — `added`/`deleted`/`net` attributing the test portion of the change, computed by a third `git diff --shortstat` pass whose pathspec combines the `test_paths` includes (as `:(glob)<pattern>` magic pathspecs) with the same `:(exclude)` args as the `excluding` pass (counted within the scaffolding-excluded universe). Lazily omitted when `test_paths` is empty/absent. Emitted after `excluding`, before `computed_at`. No `impl` field is stored anywhere: the residual `impl = max(0, total − tests)` is per-component and derived at RENDER TIME only (engine/`.status.yaml`/`fab impact` YAML stay pure-measurement). Engine: `impact.Result.Tests *Pair`, `Compute` gains a trailing `testPaths`, `ComputeForRepo` reads `cfg.TestPaths`; `statusfile.TrueImpact.Tests *TrueImpactPair` (`yaml:"tests,omitempty"`). |
| 260601-j6cs-merge-spec-into-apply | 2026-06-01 | `.status.yaml` schema: `progress.spec` key dropped (6-key progress block, no `tasks`/`spec`). `StageOrder` → `["intake", "apply", "review", "hydrate", "ship", "review-pr"]` (length 6); `StageNumber("apply") == 2`; `NextStage("intake") == "apply"`. Orphan `progress.spec` tolerated on load (`Validate()` skips, `GetProgressMap()` omits, raw-node passthrough on Save) — removed only by the `1.9.7-to-1.10.0` migration. `validateStage` returns a deprecation error for the removed `spec` stage (mirroring `tasks`). Added "`.status.yaml` Confidence Block" section: `confidence.indicative` retired (no longer written; decode-tolerant `Indicative *bool` kept; `SetConfidence`/`SetConfidenceFuzzy` dropped the param; `--indicative` flag is a one-release no-op). Noted `plan.md` now carries a `## Requirements` section. Updated the Stages bullet (6 stages, neither `tasks` nor `spec` in `allowedStates`). |
| 260507-ogf2-restrain-ai-code-bloat | 2026-05-07 | Added `.status.yaml` `true_impact` block: optional top-level mapping with `added`/`deleted`/`net`/`computed_at`/`computed_at_stage` (values `apply` or `hydrate`) plus optional `excluding` sub-block (omitted when `true_impact_exclude` is empty). Block is lazily created — no template placeholder; existing files without the block remain valid. Written by `status.Finish` for stages `apply` and `hydrate` via `WriteTrueImpact` (best-effort: stderr warning on failure, never propagates). Canonical math lives in `internal/impact/`; CLI surface is `fab impact <base> <head>`. |
| 260423-qszh-merge-tasks-checklist | 2026-05-06 | `.status.yaml` schema: `progress.tasks` key dropped entirely (no rename). `checklist:` block replaced by `plan:` block with fields `generated`, `task_count`, `acceptance_count`, `acceptance_completed` (rename: `total → acceptance_count`, `completed → acceptance_completed`; new: `task_count`; removed: `path`). Added "`.status.yaml` Plan Block" section documenting the new schema, 7-key progress block, `StageOrder`/`NextStage` updates, `set-acceptance` CLI replacing `set-checklist`, and `Load()` tolerance of legacy files. Updated Stages bullet to note 7-stage pipeline and removal of `tasks` from `allowedStates`/`isValidStage`. |
| 260419-o5ej-agents-runtime-unified | 2026-04-19 | Replaced the in-file `.fab-runtime.yaml` schema description with a cross-reference to the new [runtime-agents.md](../runtime/runtime-agents.md) (authoritative doc for the `_agents[session_id]` + `last_run_gc` schema, hook write pipeline, GC, grandparent PID walker, and pane-map matching rule). Clarified that `.fab-runtime.yaml` is a distinct schema from `workflow.yaml` — this file documents the latter. |
| 260307-x2tx-status-symlink-pointer | 2026-03-07 | Replaced `fab/current` pointer file with `.fab-status.yaml` symlink at repo root. Added `id` field to `.status.yaml`. Updated resolution, switch, rename, pane-map, hooks, and dispatch. Migration `0.32.0-to-0.34.0` covers conversion. |
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Updated Ephemeral Runtime State: `.fab-runtime.yaml` operations now use `fab runtime set-idle` and `fab runtime clear-idle` Go subcommands instead of direct yq manipulation in hooks. |
| 260306-1lwf-extract-agent-runtime-file | 2026-03-06 | Moved agent runtime state from `.status.yaml` to `.fab-runtime.yaml` (repo root, gitignored, keyed by change folder name). Updated Ephemeral Runtime State section accordingly. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added Ephemeral Runtime State section documenting the optional `agent` block (`agent.idle_since` timestamp) managed by Claude Code hooks, not part of workflow schema or templates |
| 260215-lqm5-statusman-cli-only | 2026-02-15 | Updated script example from `source statusman.sh` to CLI subprocess pattern (`$STATUSMAN <subcommand>`) |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` in skill example; updated `src/statusman/README.md` → `src/lib/statusman/README.md` |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Updated progression references: terminal stage from `archive` to `hydrate` |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Transitions are now event-keyed (event, from, to) instead of from→to with conditions. Five event commands: `start`, `advance`, `finish`, `reset`, `fail`. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | Added `ready` state (artifact exists, eligible for advancement). Removed unused `skipped` state. Updated transitions (`active→ready`, `ready→done`), progression (current stage includes `ready`), and validation (terminal states: `done` only). |
| 260228-wyhd-add-skipped-stage-state | 2026-02-28 | Added `skipped` state (`⏭`, terminal) and `skip` event (`{pending,active} → skipped` with forward cascade). Updated `reset` to accept `skipped → active`. Updated progression rules to treat `skipped` alongside `done`. Allowed for all stages except intake. Six event commands: `start`, `advance`, `finish`, `reset`, `fail`, `skip`. |
| 260212-4tw0-migrate-scripts-statusman | 2026-02-12 | Moved from `$(fab kit-path)/schemas/README.md`, trimmed statusman API duplication |
