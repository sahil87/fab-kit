---
type: memory
description: "Workflow schema authority — the Go state machine (`internal/status` transitions + `internal/statusfile` stage order/progress; declarative `workflow.yaml` retired in c5tr): 6-stage pipeline, states, transitions, validation rules; `.status.yaml` `plan:` (`## Requirements`-aware), `confidence:` (indicative retired), and lazy `true_impact:` block schemas (incl. the `tests` sub-block + render-time `impl` residual, 7t5a); `fab impact` and `fab pr-meta` helper subcommands (rj31); allowed-states-enforced transition targets, `fab score --check-gate` non-zero gate-fail exit, iterations-preserving reset cascade (k4ge; cycle-count accuracy is a choreography property not a state-machine one, the fix lives in skill prose — qg64); `fab score` normal-mode hard-fail on load/persist/read errors (hv7t); per-stage allowed-states + transition-event tables enumerated, pinned by the exhaustive 216-cell matrix test (tb6f); `change_type_source: inferred|explicit` guard (set-change-type marks explicit, hook re-infers only when not explicit), read-time `acceptance_completed` via `status.LiveAcceptance` (counter demoted to cache), and `internal/resolve` `ErrNotFound`/`ErrAmbiguous` sentinels (jznd); optional `summary` field + `set-summary`/`get-summary` verbs (FKF C-lite log source, 5943); the `log.md` C-lite schema + registry-gated change-id join + extended (benign-only, no-new-category) `--check` loss tiers (bmzo); the `log.seed.md` seed-merge — `parseSeedLog`/`mergeSeedEntries` curated-sidecar input merged beneath git-projected entries, idempotent, seed-only folders still emit a log.md, loss tier stays benign (oovf)"
---
# Schemas

**Domain**: pipeline

## Overview

The single source of truth for the Fab workflow — stages, states, transitions, and validation rules — is the **Go state machine**: `src/go/fab/internal/status` (event-keyed transitions and their side-effects) and `src/go/fab/internal/statusfile` (stage order, progress schema, `.status.yaml` encode/decode). All scripts and skills query it via the `fab status` / `fab preflight` CLI surface rather than hardcoding workflow knowledge.

The former declarative schema artifact `src/kit/schemas/workflow.yaml` was **retired in 260612-c5tr** (file deleted; the `src/kit/schemas/` directory is gone): nothing consumed it — no script, skill, or binary parsed it — and it had silently drifted a full pipeline generation, still describing the pre-1.10.0 7-stage pipeline with a `spec` stage while `docs/specs/user-flow.md` called it the source of truth. That user-flow line now points at the Go state machine. It was retired rather than regenerated because a regenerated declarative artifact would re-create the same unenforced drift surface. (A frozen pre-retirement copy briefly survived as a benchmark fixture at `src/benchmark/fixtures/workflow.yaml`; that fixture tree was deleted in 260612-tb6f along with the rest of the benchmark implementations.)

## What the State Machine Defines

1. **States** — All valid progress values (`pending`, `active`, `ready`, `done`, `failed`, `skipped`)
   - Each state has an ID, a display symbol, and terminal semantics
   - `ready` means "stage work product exists, eligible for advancement or clarification" (non-terminal)
   - `skipped` means "stage intentionally bypassed" (terminal, symbol `⏭`). Allowed for all stages except intake
   - Terminal states (`done`, `skipped`) cannot transition without explicit reset

2. **Stages** — The workflow pipeline in execution order — 6 stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` stage was removed in qszh, and the `spec` stage in j6cs; plan generation is an apply-internal sub-step that produces a unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`). `allowedStates` contains neither a `tasks` nor a `spec` key, and `isValidStage("tasks")`/`isValidStage("spec")` both return false. `validateStage` returns a deprecation error for either removed stage (`spec` → `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`).
   - Each stage has: ID, name, artifact, description, requirements, initial state, allowed states, commands
   - Stages execute in sequence with dependency validation
   - Per-stage allowed states (`AllowedStates` in status.go — both the `.status.yaml` validation source and the transition target-validation source; enumerated here since tb6f so the documented spec pins the table the exhaustive matrix test asserts):

     | Stage | Allowed states |
     |-------|----------------|
     | `intake` | `active`, `ready`, `done` |
     | `apply` | `pending`, `active`, `ready`, `done`, `skipped` |
     | `review` | `pending`, `active`, `ready`, `done`, `failed`, `skipped` |
     | `hydrate` | `pending`, `active`, `ready`, `done`, `skipped` |
     | `ship` | `pending`, `active`, `done`, `skipped` |
     | `review-pr` | `pending`, `active`, `done`, `failed`, `skipped` |

3. **Transitions** — Valid state changes for each stage, event-keyed (event, from, to)
   - Default rules apply to all stages
   - Stage-specific overrides (e.g., `review` allows `fail` event)
   - Each transition is triggered by an event command (`start`, `advance`, `finish`, `reset`, `fail`, `skip`)
   - The complete event table (default rules + the review/review-pr overrides). Any (event, stage, from-state) cell outside it is rejected with a `no valid transition` error; a from-valid cell whose *target* the stage forbids is rejected by the target-state validation below:

     | Event | From | Target | Stages |
     |-------|------|--------|--------|
     | `start` | `pending` — review/review-pr also `failed` | `active` | all |
     | `advance` | `active` | `ready` | all except `ship`/`review-pr` (target `ready` forbidden) |
     | `finish` | `active`, `ready` | `done` | all |
     | `reset` | `done`, `ready`, `skipped` | `active` | all |
     | `skip` | `pending`, `active` | `skipped` | all except `intake` (target `skipped` forbidden) |
     | `fail` | `active` | `failed` | `review`/`review-pr` only |

   - `skip` event: `{pending,active} → skipped` with forward cascade (all downstream pending → skipped). No auto-activate
   - `reset` accepts `skipped` as a source state (`skipped → active` with downstream cascade to `pending`)
   - **Target-state validation (k4ge)**: `lookupTransition` validates the resolved target state against the stage's allowed states (a `validateTarget` helper applied to both the stage-override and the default resolution path). A schema-forbidden combination — `advance ship`/`advance review-pr` (target `ready`) or `skip intake` (target `skipped`) — exits non-zero with `Cannot {event} stage '{stage}' — target state '{state}' is not allowed for this stage` and writes nothing, instead of writing a state that permanently bricks `fab preflight` ("State 'ready' not allowed for stage ship"). The schema is the single constraint source, so any future forbidden combo is closed automatically. The now-unreachable `stageTransitions["review-pr"]["advance"]` override row in `status.go` is a recorded deletion candidate (k4ge plan) — removing it is byte-identical behavior since the default `advance` row produces the same rejection. tb6f's review widened the candidate: the `advance`/`finish`/`reset` rows in *both* the review and review-pr overrides duplicate `defaultTransitions` byte-for-byte (only `start`/`fail` genuinely differ; `lookupTransition` falls through to the default table for events absent from a stage override), and the exhaustive matrix test now proves any such removal behavior-neutral
   - **Cascade preserves `iterations` (k4ge)**: when the `reset`/`skip` cascade sets a stage to `pending`/`skipped`, a `stage_metrics` entry with `iterations > 0` is kept with only its `iterations` counter (timing fields `started_at`/`driver`/`completed_at` cleared; the next activation rewrites them); zero-iteration entries are still deleted, so no empty `{}` entries linger. This keeps `stage_metrics.review.iterations` truthful across the rework choreography's `fail review` + `reset apply`, making the cycle count `fab pr-meta` reports real. Preservation is uniform across all stages, not review-only — since tb6f it is regression-tested end-to-end through the public `Fail`/`Reset`/`Finish` functions (3-cycle rework choreography, iterations 1→2→3) and across the `skip` cascade; the test passes against the shipped implementation, confirming the PR #402 "1 cycle" anomaly is not a state-machine bug. **Cycle-count accuracy is an orchestrator-CHOREOGRAPHY property, not a state-machine one (qg64)**: the iterations-preserving cascade here is correct **as-is** and MUST NOT be touched — `iterations` is advanced by exactly one event, a review `→ active` transition (`status.go:627`), and the `reset` cascade deliberately neither increments nor zeroes it. So whether `fab pr-meta` renders the true cycle count depends entirely on the orchestrator skills driving exactly one counted review `→ active` re-entry per rework cycle (via the per-cycle `finish apply` auto-activation). The PR #402 "1 cycle" under-report was therefore fixed in **skill prose** (`_pipeline.md`'s Auto-Rework Loop now states the counting invariant + the baseline convention — iterations == initial entry + each re-entry, so initial fail + N cycles → N+1), **not** in `internal/status`. See [execution-skills.md](/pipeline/execution-skills.md) § Shared Pipeline Bracket (Per-cycle rework choreography → Cycle-count invariant) for the choreography half. See [change-lifecycle.md](/pipeline/change-lifecycle.md) for full `stage_metrics` semantics
   - **Exhaustively tested (tb6f)**: `internal/status/transitions_test.go` walks all 216 cells (6 stages × 6 events × 6 from-states) of the two tables above and asserts every cell's outcome — target state, forbidden-target rejection, or no-valid-transition rejection — against hand-written copies of the tables, deliberately NOT the implementation's own vars, so a table regression cannot silently rewrite the test's expectations. The enumerations in this doc and that test pin the same spec
   - **History shape is caller-identity-blind (fgxx)**: the `.status.yaml` / `.history.jsonl` transition history left by a conforming rework run depends only on the **call sequence**, never on caller identity. `driver` flows solely into `applyMetricsSideEffect` (it annotates `stage_metrics.driver` and the transition-log entry) — no state transition reads it — so the manual/foreground path (`driver=""`) and the dispatched-orchestrator path (`driver="fab-fff"`) issue the identical `Finish(intake) → Finish(apply) → [Fail(review) → Reset(apply) → Finish(apply)]×N` sequence and leave history of **identical shape** — agreeing on every caller-blind field (stage/action/from/reason), equal modulo the per-run `ts` timestamp and the optional `driver` field. `internal/status/mutators_test.go`'s `TestHistoryShape_IdenticalRegardlessOfDriver` pins this: it drives the rework choreography twice (`driver=""` and `driver="fab-fff"`) and asserts the `.history.jsonl` stage-transition entries match in count/stage/action/from/reason, differing only where `driver` is recorded. This is the structural invariant that made collapsing the post-intake dual execution mode safe (fgxx — both modes already issued the same CLI calls); the test guards against a future skills-layer regression that diverges the two sequences

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

The `Load()` function is tolerant of legacy `.status.yaml` files: it upgrades a `checklist:` block to a `plan:` raw mapping with field migration (`completed → acceptance_completed`, `total → acceptance_count`) and drops `checklist:` when both blocks coexist. The `1.8.0-to-1.9.0.md` migration rewrites in-flight `.status.yaml` files to the new schema (drops `progress.tasks`, replaces `checklist:` with `plan:`); the `1.9.7-to-1.10.0.md` migration drops `progress.spec`; see [migrations.md](/distribution/migrations.md).

As of j6cs the apply-stage `plan.md` carries a `## Requirements` section (RFC-2119 + GIVEN/WHEN/THEN, the requirement discipline absorbed from the removed `spec.md`) alongside `## Tasks` and `## Acceptance` — these three `##` headings are the stable parser contract.

**`acceptance_completed`/`acceptance_count` are read-time-derived; the `plan:` counter is a cache (jznd).** As of 260615-jznd acceptance progress is the truth on disk: `status.LiveAcceptance(changeDir) (done, total int, ok bool)` (`internal/status/acceptance.go`) counts the `## Acceptance` checkboxes in `{changeDir}/plan.md` at READ time via the existing `hooklib.HasSectionHeading` + `CountSectionItemsBounded`/`CountCompletedSectionItemsBounded`, and the read sites — `internal/preflight`, `internal/prmeta` (`Gather`, both the plan.md and legacy tasks.md branches), and `cmd/fab status plan` — prefer the live count over the persisted `plan.acceptance_*` counter. The hook-maintained `plan:` counter remains a write-time **cache/fallback**, used only when `LiveAcceptance` returns `ok=false` (no `plan.md`, or no `## Acceptance` heading — e.g. an intake-only or pre-plan change). This makes a hook-bypassing mutation — `sed`, a direct `.status.yaml` edit, or a checkbox toggled by a tool the PostToolUse hook doesn't observe — no longer able to make the readers report a stale number. `fab score` is out of scope (it reads `intake.md` only — see [_shared/configuration.md](/_shared/configuration.md) and the `score-binary-source-version-skew` note).

## `.status.yaml` Change-Type Fields (jznd)

`.status.yaml` carries a top-level `change_type` (the inferred/explicit taxonomy slot — `feat`/`fix`/`refactor`/`docs`/`test`/`ci`/`chore`) and, as of 260615-jznd, a companion `change_type_source` enum recording **how** that type was set:

```yaml
change_type: feat
change_type_source: explicit   # inferred | explicit ; absent/empty == inferred (back-compat)
```

- **`change_type_source`** is `inferred` (default) or `explicit`. The field is serialized only when non-empty (`yaml:"change_type_source,omitempty"`, inserted on a sparse document like `change_type` via `insertKey`/`syncToRaw`); an absent/empty value decodes as the `inferred`-equivalent, so every pre-jznd change behaves exactly as before. Exported constants `statusfile.SourceInferred`/`SourceExplicit`.
- **`fab status set-change-type` always marks the source `explicit`** (writes both `change_type` and `change_type_source: explicit`).
- **The PostToolUse intake-write hook (`artifactBookkeeping`) applies inference and overwrites `change_type` ONLY when `change_type_source` is absent or `inferred`.** When it is `explicit` the hook skips both `InferChangeType` and the type overwrite — a human-corrected type survives all subsequent intake re-writes (the pre-jznd silent-revert race is gone). Acceptance counting and the rest of the hook's bookkeeping still run regardless. See [planning-skills.md](/pipeline/planning-skills.md) § Change-Type Inference Is Hook-Owned, Explicit Set Is Sticky for the skill-side contract.

The `fix` keyword regex was also tightened in jznd (the old `\b(fix|bug|broken|regression)\b` matched `fix` inside hyphenated compounds because RE2 treats `-` as a word boundary): a passing `must-fix`/`must fix` in an otherwise-feature intake no longer classifies `fix`, while `bug-fix`/`hot-fix`/`bug-free` and standalone `fix`/`bug`/`broken`/`regression` still do. The old pattern was kept (renamed `fixCandidateRegex`, reused inside a `fixSignal` post-match guard) — not deleted.

## `.status.yaml` `summary` Field (5943)

`.status.yaml` MAY carry a top-level optional `summary` string — the per-change one-line log summary, the FKF C-lite source line `fab memory-index` joins with git history to generate `log.md` (see [fkf.md](../../specs/fkf.md) §6.3 and the **`log.md` C-lite Schema (bmzo)** section below). Added in 260615-5943 as the FOUNDATION of the FKF bundle; the generator that **consumes** this field shipped in 260615-bmzo (the KEYSTONE).

```yaml
summary: "added the .status.yaml summary field + migration"   # optional; absent/empty == no summary
```

- **`summary`** is modeled **exactly** on `change_type_source`: `yaml:"summary,omitempty"`, serialized only when non-empty, dropped on write when empty (`syncToRaw` `case "summary"` → `dropKeyAt`), and inserted before `last_updated` on a sparse document that lacks the key (`insertKey`, same as `change_type_source`/`true_impact`). An absent/empty `summary` decodes to `""` and round-trips to absent, so every pre-5943 change behaves exactly as before. The `StatusFile.Summary` field (`internal/statusfile`) is decoded by `Load()`'s explicit `switch key` (not pure struct-tag decode — `Load` walks the raw mapping).
- **`fab status set-summary <change> <text>`** sets the field and `Save`s via `status.SetSummary` (the conflict-free write path — each change touches only its own `.status.yaml`). Unlike `set-change-type` it has **no** `change_type_source: explicit`-style sticky side effect — `summary` has no inferring hook to guard against. An empty text clears the field.
- **`fab status get-summary <change>`** prints `st.Summary` via the lock-free `loadStatus` reader. An absent/empty summary prints an empty line (graceful absence — the generator falls back to the change slug).
- **No stage auto-populates `summary`.** 5943 creates the field + verbs only; authoring is "written once during the change (at hydrate, or carried from the intake)" per §6.3, but that wiring is deferred to a later FKF change. bmzo's `fab memory-index` only **reads** the field (joining it into `log.md`) — it never writes it.
- The template (`src/kit/templates/status.yaml`) seeds `summary: ""` between `prs: []` and `# true_impact`/`last_updated`. Migration `2.4.2-to-2.5.0` adds `summary: ""` to in-flight `fab/changes/*/.status.yaml` (before `last_updated`, idempotent — skips files already having the key, skips `archive/**`).

## `log.md` C-lite Schema (bmzo)

As of 260615-bmzo (FKF KEYSTONE, change 2/4) `fab memory-index` emits a generated **per-folder `log.md`** (FKF §6 — see [fkf.md](../../specs/fkf.md)) for every domain and sub-domain folder with attributable git history, alongside the `index.md` tiers. It is the C-lite change log; the generator code lives in `src/go/fab/internal/memoryindex/` (`log.go` `RenderLog` + `memoryindex.go` `GatherLogs`/`gatherChangeRegistry`/`attributeCommit`), and the CLI surface is documented in `_cli-fab.md` § fab memory-index. The memory-side doc lives in [memory-docs/templates.md](/memory-docs/templates.md) § Generated `log.md`; this section records the **schema/data-linkage** half.

**The two-source C-lite join (the schema-relevant part):**

1. **Git history** — the *when* / *which-file* / *per-commit name-status*, taken from the **same single batched pass** the index dates use: `git log -c core.quotepath=off --date=short --format=<NUL %ad US %s> --name-status -- docs/memory`. bmzo widened the former `--name-only` projection to `--name-status` so the one pass yields BOTH the existing newest-date-per-path map (`byPath`, index "Last Updated", behavior unchanged) AND a new ordered per-path commit list (`commitsByPath` — `(date, subject, status)` tuples). No per-file `git log` spawn is reintroduced (pw3k F34 invariant preserved).
2. **`.status.yaml` `summary:`** — the *what*, the **source-field linkage to Change 1 (5943)**. `GatherLogs` builds a change registry by enumerating `fab/changes/*` + `fab/changes/archive/**` (the canonical `(change-id, folder, slug, summary)` set), reading each `.status.yaml` `summary:` via `internal/statusfile.Load`. The entry's descriptive line is that change's `summary`, or the **change slug** when the summary is empty/absent (§6.3 graceful fallback).

**`LogData`/`LogEntry` render contract** (pure `RenderLog(LogData) string`, mirroring `RenderRoot`/`RenderDomain`): `LogData{Title, Entries []LogEntry}`; `LogEntry{Date, Verb, FileBase, BundleRelPath, Summary, ChangeID}`. Output is `# Log — {Title}` + the `Do not hand-edit` generated comment, then entries **date-grouped, newest date first** (`## YYYY-MM-DD`), each `- {**Verb** }[base](/{domain}[/{sub}]/base.md) — {summary-or-slug} ({change-id})`. Intra-date order is a stable sort (date desc, then file base, then change-id) so output is byte-stable / idempotent. Verb derivation maps git name-status `A`→`**Creation**`, `D`→`**Deprecation**`, `M`/`R`/`C`/`T`→`**Update**`, else omit (optional per §6.2). Links are **bundle-relative** (FKF §7 — `/`-rooted, resolved from `docs/memory/`). A folder with zero attributable commits is **skipped** (no empty file — Design Decision: target set = "folders with history").

**change-id join is registry-gated, with graceful degradation.** `attributeCommit` recovers a `{YYMMDD}-{XXXX}-{slug}` folder token (or a bare registered `{XXXX}`) from the commit **subject** and gates it against the registry — a token only counts when it maps to a known change, so the join is authoritative and false-positive-free. The merge-commit branch token (`Merge pull request #N from owner/<folder>`) is the only recoverable shape and works **only on legacy true-merge history**; against this repo's squash-merged history (subjects `feat: … (#NNN)` carry no branch token, transient branch refs gone) it recovers **≈0 ids in practice**, so most entries take the degraded path — the `(change-id)` token is **omitted** and the descriptive line falls back to the commit subject (still a conflict-free git projection), never erroring. The format physically exists now and self-heals as FKF-era changes land curated summaries on attributable commits. (This is the realizable form of intake assumption #12's "branch/merge graph" framing — a live `git branch --contains` walk is not realizable from `git log` alone in a fresh clone/CI, so the join uses registry-gated message-token recovery instead.)

**FKF frontmatter (generator-owned).** `RenderRoot` prepends `---\nfkf_version: "0.1"\n---` to the **root** `docs/memory/index.md` only (FKF §8); no other index tier carries it. The generator also **preserves** `type: memory` when round-tripping a file it owns (the §3.1 constant) but does NOT author or bulk-stamp it into topic files (authoring → FKF 3/4, bulk-stamp → FKF 4/4).

**Extended `--check` loss tiers (the classifier linkage).** `log.md` and the FKF frontmatter join the `Classify`/`CheckTarget` drift surface in `loss.go`, but only ever as **benign drift (TierBenignDrift, exit 1)** — a new `IsLog` flag on `CheckTarget` short-circuits the index-row destructive-loss detectors (description/tombstone/grouping are row-table-shaped and meaningless on a git-projected log). bmzo introduces **no new tier-2 `LossCategory`** (OQ4/assumption #9 decision): the `LossCategory` enum (`description`/`tombstone`/`grouping`) is unchanged, so the `--json` `losses[].category` enum is **unchanged** (the `{"tier", "drift", "losses":[{"category","path","detail"}]}` shape is additive-stable). A stale `log.md` or absent/changed `fkf_version` frontmatter reports tier 1; a **born-FKF tree is provably never tier 2** (native `log.md`/frontmatter is exactly what the generator produces). The exit-code/`--json` contract otherwise matches the existing `fab memory-index --check` surface (see [memory-docs/templates.md](/memory-docs/templates.md) § Memory Tree Shape for the `/docs-reorg-memory` consumer of the `--json` losses).

## `log.seed.md` Seed-Merge (oovf — FKF cutover crux)

The FKF cutover (260615-oovf, change 4/4) had to preserve **651 pre-FKF `## Changelog` rows** verbatim (DECISION b — faithful preservation over a slug-only git projection) while keeping `log.md` a generated single-writer output. Those rows carry their **own authored dates** and have **no live `.status.yaml` `summary:`** to project from (the changes are shipped/archived), so the bmzo two-source join (git history ⋈ live `summary:`) cannot regenerate them. The resolution: teach `fab memory-index` a **seed-merge** — it reads a per-folder curated sidecar and merges those entries beneath the git-projected ones. This section records the schema/data-linkage half; the CLI surface is in `_cli-fab.md` § fab memory-index and the memory-doc consumer view in [memory-docs/templates.md](/memory-docs/templates.md) § Generated `log.md`.

- **`log.seed.md` is an INPUT, not output (the single-writer invariant holds).** The sidecar `log.seed.md` (`seedFileName` const in `seed.go`) sits alongside the generated `log.md` in each domain/sub-domain folder. It is curated — like `description:` frontmatter — and `fab memory-index` only ever **reads** it, never writes it. So the FKF §5/§6 single-writer / byte-stable discipline is preserved: the generator stays the sole writer of `log.md`; the seed is just another gathered input. It is excluded from `gatherFiles` / `gatherLogEntries` exactly as `index.md` / `log.md` are — so it is **never a topic-index row** and never re-read as history.
- **Parse contract — `parseSeedLog` is `RenderLog`'s inverse.** `parseSeedLog(content string) []LogEntry` (pure, in `seed.go`) parses the FKF §6.2 rendered shape — `## YYYY-MM-DD` date headings and `- {**Verb** }[base](/bundle/rel.md) — summary{ (id)}` bullets — into the **same** `LogEntry{Date, Verb, FileBase, BundleRelPath, Summary, ChangeID}` shape the bmzo render contract uses (no parallel struct). A parse∘render round trip is the identity on well-formed entries. The seed's own date heading is preserved verbatim as `LogEntry.Date` (the pre-FKF changelog `Date` column — independent of git). The leading bold verb and trailing `(id)` token are both optional; `splitTrailingID` peels the id **only** when the last parenthesized group is space-free (so an in-prose `(#42)` or `(some aside)` is not mistaken for a change-id), and a missing-cell `—` summary round-trips to `""`. Malformed lines (no link cell, no ` — ` separator) are skipped, not errored.
- **Merge — concatenate-then-dedupe, ordering delegated to `RenderLog`.** `mergeSeedEntries(projected, seed []LogEntry) []LogEntry` unions the two slices, de-duplicating any seed entry **byte-equal** to a projected entry (full `LogEntry` struct equality — Date/Verb/FileBase/BundleRelPath/Summary/ChangeID) via a `map[LogEntry]bool`. Projected entries are appended first so that, within a date, git-projected lines render ahead of seed lines. The function does **no sorting** — it relies on `RenderLog`'s existing stable date-group sort (date desc, then file base, then change-id) for deterministic byte-stable output. The read-from-disk wiring (load `{folder}/log.seed.md` when present, `parseSeedLog`, `mergeSeedEntries` before `RenderLog`) lives in `memoryindex.go`'s `GatherLogs`/`buildLogTarget` alongside the other I/O, keeping `seed.go` pure.
- **Seed-only folders still emit a `log.md`.** A folder whose only history is a `log.seed.md` (no attributable git commits) still produces a `log.md` — the `GatherLogs` "skip folders with no history" short-circuit is relaxed so the target set is "folders with git history **or** a seed."
- **Idempotent (Constitution III).** A re-run on an unchanged tree is byte-stable: a seed entry that already matches a projected entry is dropped, so no duplication accumulates and `--check` exits 0.
- **Loss tier unchanged — seed-merge stays benign.** A `log.md` whose drift is driven by merged seed entries classifies as **benign drift (tier 1)**, never destructive loss. The existing bmzo `IsLog` guard already routes all `log.md` drift to benign and short-circuits the row-table detectors; the seed-merge adds **no new tier-2 `LossCategory`** (the enum and the `--json` `losses[].category` shape are unchanged — additive-stable, same as bmzo). The preserved seed is never reported as loss (`loss_test.go` pins this with a regression test).

## `internal/resolve` Typed Errors (jznd)

`internal/resolve` exposes two sentinels — `ErrNotFound` (`"no matching change"`) and `ErrAmbiguous` (`"ambiguous change reference"`) — so callers can `errors.Is` instead of string-matching. The package's "no change matches" / "no active change(s)" messages are built via `notFoundf(...)` and its "multiple changes match" / "multiple changes exist" messages via `ambiguousf(...)`; both return a `classifiedError` wrapper that **preserves the original user-facing message string** while carrying the sentinel kind (so `errors.Is` works and the surfaced text is unchanged). Precedent: `internal/archive` already declared `ErrAlreadyArchived`. The archive soft-skip now branches on these — see [execution-skills.md](/pipeline/execution-skills.md) § Idempotent Re-Archive.

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
- `tests` — same fields, attributing the test portion of the change (7t5a). Computed by a third `git diff --shortstat` pass whose pathspec combines the `test_paths` includes with the SAME `:(exclude)<pattern>` args as the `excluding` pass — so test lines are counted *within the scaffolding-excluded universe* (a test fixture under an excluded path is not double-counted). Each `test_paths` include is applied as a `:(glob)<pattern>` magic pathspec so `**` matches across directory boundaries. When `true_impact_exclude` is empty the test pass runs with the includes alone (tests attributed within the raw universe). Sub-block omitted entirely (lazily) when `test_paths` is absent/null/empty — behavior then collapses to today's single-number display. See [configuration.md](/_shared/configuration.md) for the `test_paths` config field.
- `computed_at` — RFC 3339 UTC timestamp.
- `computed_at_stage` — pipeline stage at which the snapshot was taken: `apply` or `hydrate`.

**No `impl` field is stored.** The implementation residual is `impl = max(0, total − tests)` *per component* (`added`/`deleted`/`net` each clamped independently, since the three-row display shows separate `+X / −Y` components and each must be non-negative on its own), where `total` is the scaffolding-excluded number (`excluding`, else raw when `true_impact_exclude` is empty). It is derived at RENDER TIME ONLY — the impact engine (`internal/impact/`), `.status.yaml`, and the `fab impact` YAML store only the *measured* passes (raw, `excluding`, `tests`), never a derived `impl`. This keeps the engine pure-measurement so no derived field can drift or go stale between the two diff passes; the cost is that the residual + clamp logic is implemented at both render sites (the `fab pr-meta` Impact line in `internal/prmeta/` — which renders the PR `## Meta` block for `/git-pr` as of rj31 — and `impactColumn()` in `internal/change/`). When the clamp triggers (a `test_paths` glob overlaps a `true_impact_exclude` path over-counting `tests`, OR a genuinely test-heavy diff where `total.Net − tests.Net` is negative), the displayed impl stays non-negative.

**The `prmeta` site annotates the clamp; `impactColumn` does not (jznd (e)).** As of 260615-jznd the `fab pr-meta` Impact line, when the clamp actually changes a value (the pre-clamp `added`/`deleted`/`net` was negative), keeps the clamped `+0` display value AND appends a trailing `(clamped from net -N[, added -M, deleted -K])` note naming only the fields that were clamped (a `clampAnnotation` helper; minus-signed, listed `net`→`added`→`deleted`). This stops PR-meta from silently hiding a net-deletion-in-production PR. The clamp itself is **kept** (downstream consumers may assume non-negative); only the truth is surfaced alongside it — the binary-review Refuted section did not adjudicate the clamp, so "annotate, don't remove/sign" was the minimal honest change. The other render site `impactColumn()` (the `fab change list` column) was out of scope and is unchanged — it still clamps without annotation.

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

`.fab-status.yaml` is a symlink at the repository root pointing to the active change's `.status.yaml`. It is the active change pointer — the replacement for the former `fab/current` text file. The symlink target is always a relative path: `fab/changes/{name}/.status.yaml`. See [change-lifecycle.md](/pipeline/change-lifecycle.md) for full lifecycle documentation.

Together with `.fab-runtime.yaml`, these two sibling files at the repo root form the complete ephemeral per-worktree state surface, scannable with a single glob.

## Ephemeral Runtime State

### Agent State — `.fab-runtime.yaml`

Agent runtime state lives in `.fab-runtime.yaml` at the repository root (gitignored). This file is NOT part of the workflow schema (distinct from the workflow state machine this doc describes), NOT initialized by templates, and NOT read by any workflow command. It is managed by Claude Code hook scripts via the `fab hook stop|session-start|user-prompt` subcommands.

**Schema and write pipeline**: See [runtime-agents.md](/runtime/runtime-agents.md) for the authoritative documentation. The file uses a top-level `_agents` map keyed by Claude's `session_id` (UUID from hook stdin) with `change`, `pid`, `tmux_server`, `tmux_pane`, and `transcript_path` as optional entry properties, plus a top-level `last_run_gc` timestamp that throttles an inline GC sweep. Entries populate regardless of active-change state, so agents running in discussion mode are tracked the same as change-associated agents.

Each worktree has its own repo root, so each gets its own `.fab-runtime.yaml` — no cross-worktree contention. External tools can read this file to detect agent idle state and correlate agents to panes without relying on timing heuristics.

## Future Enhancements

1. **Custom workflows** — Allow `fab/project/config.yaml` to override or extend the stage graph
2. **~~Conditional stages~~** — *(Partially addressed)* The `skipped` state and `skip` event now enable explicit stage bypassing via `fab status skip`. Skill-level orchestration (automatic skip based on change attributes) remains a future enhancement
3. **Parallel stages** — Multiple stages active simultaneously for different artifacts
4. **~~Stage hooks~~** — *(Shipped)* The `stage_hooks.{stage}.pre/post` config surface runs commands around `fab status start`/`finish` — live Go behavior, documented in `_cli-fab.md` § stage_hooks as of c5tr (pre blocks `start` on non-zero exit; post runs after `finish`'s save). See [change-lifecycle.md](/pipeline/change-lifecycle.md)
5. **State metadata** — Attach timestamps, user info, or exit codes to state transitions
