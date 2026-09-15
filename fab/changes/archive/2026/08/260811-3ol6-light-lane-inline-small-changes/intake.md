# Intake: Light Lane — Plan-Time Fork to Inline Execution for Small Changes

**Change**: 260811-3ol6-light-lane-inline-small-changes
**Created**: 2026-08-11

## Origin

Created via promptless dispatch (`{questioning-mode} = promptless-defer`) from a synthesized design-discussion description (discussion dated 2026-08-11). All design decisions below were settled in that discussion; a design reference with state-machine diagrams is already in-tree at `docs/findings/light-mode-state-machines.md` + `.html` (validated against loom's archive) and rides along with this change.

> Light lane — plan-time fork to inline execution for small changes in /fab-ff and /fab-fff. For a small change (a few-task docs/fix/chore), the expensive part of the ff/fff pipeline is not the six-stage state machine (cheap CLI transitions + small artifacts) but the execution locus: ~4-5 subagent cold starts (apply, review, hydrate, git-pr, git-pr-review), each re-reading the always-load layer, intake, and affected memory the orchestrator already holds in context. The dispatch overhead dwarfs the work several times over for one-line changes.

## Why

1. **The pain point.** For a small change, `/fab-ff` and `/fab-fff` pay ~4-5 subagent cold starts (apply, review, hydrate, ship via /git-pr, review-pr via /git-pr-review). Each dispatched worker re-reads the always-load layer, `intake.md`, and the affected memory that the orchestrator *already holds in context*. For a one-line change the dispatch overhead dwarfs the actual work several times over. The state machine itself is cheap (CLI transitions + small artifacts) — the cost is entirely the execution locus.

2. **The consequence of not fixing it.** Small changes (21% of loom's 972-change archive would qualify at the ≤5-task cut) keep paying full-pipeline latency and token cost, which discourages routing small work through the pipeline at all — eroding the review/hydrate discipline the pipeline exists to enforce.

3. **Why this approach.** A plan-time fork on task count rides an existing seam (the apply contract's "co-generate plan.md ... unless plan.md exists") and changes ZERO state machinery. Empirical evidence from loom's archive (972 completed changes, 927 with recoverable task counts, analyzed 2026-08-11) shows task count is the right signal: rework rate is a clean gradient over plan size — 8% (1-3 tasks), 7% (4-5), 11% (6-8), 16% (9-12), 29% (13+). At the ≤5 cut: 210 changes light-eligible at plan time; 195 (21% of all) finish with zero rework; 137 of those shipped recorded PRs; only ~7% of light runs would see any rework at all. The rejected alternative (change-type entry heuristic) was ruled out on the same data: 68% of type-eligible changes outgrow the size threshold while 154 small feat/fix changes escape it (type recall ~26%, precision ~32%). Task count is the signal; type is noise.

## What Changes

All seven design points below were settled in the 2026-08-11 discussion. **v1 is skill-prose only: zero Go changes, no `.status.yaml` schema change, no config registry change** (Constitution I, Pure Prompt Play). The lane lives in the orchestrator's context for the run.

### 1. Both state machines unchanged — a hard constraint

The stage graph (intake → apply → review → hydrate → ship → review-pr, rework back-edge, exhaustion parking) and the per-stage state machine (pending/active/ready/done/failed/skipped + start/advance/finish/reset/skip/fail in `internal/status/status.go`) gain ZERO new states and ZERO new transitions. Everything that reads `.status.yaml` (resumability, pr-meta, display, preflight routing, archive) is untouched **by construction**. The same `fab status finish/fail/reset` choreography fires in the same order in both lanes; the orchestrator stays the pure sequencer; the review cycle-count invariant (finish apply is the only counted review re-entry) holds verbatim.

### 2. plan.md co-generation moves inline into the orchestrator at apply entry — ALWAYS, both lanes

The orchestrator (the `_pipeline.md` bracket, at apply entry) co-generates `plan.md` inline in its own context — in BOTH lanes. The full lane's dispatched apply worker then receives the finished plan through the plan-exists seam already in the apply contract ("co-generate plan.md ... unless plan.md exists" — `src/kit/skills/_pipeline.md` Step 1): its cold start does task execution only, and the dispatch prompt can carry the task list.

**Guard for the worst case**: when the intake's affected scope is obviously large, the orchestrator MAY skip inline co-gen and dispatch apply-with-co-gen exactly as today — graceful degradation to the shipped path, bounding inline-planning context cost.

### 3. The lane decision is a one-time fork, not a machine

Immediately after inline co-gen: task count ≤ 5 → LIGHT lane; > 5 → FULL lane. Per-invocation `--light` / `--full` flags on `/fab-ff` and `/fab-fff` skip the check. No intake-time heuristic (see Rejected Alternatives). Threshold hardcoded at 5 in v1 — a config knob (`light_max_tasks`) is a recorded follow-up, NOT in scope.

### 4. LIGHT lane execution loci

Task execution (apply), hydrate, ship (/git-pr behavior), and review-pr (/git-pr-review behavior) run INLINE in the orchestrator's own context, following the same behavior sections the dispatched workers follow today.

- Inline stages skip `fab resolve-agent` and run on the session model — consistent with the existing rule that an undispatched stage MAY report the configured profile but MUST NOT switch the session model (`_preamble.md` § Per-Stage Model Resolution).
- Inline ship/review-pr is essentially today's standalone path (a user running /git-pr directly IS inline execution) — those skills keep managing their own stage transitions exactly as they do standalone.
- Inline review-pr also eliminates the subagent yield-seam hazard the synchronous-poll directive in fab-fff Step 5 exists to fight (the 10-minute Copilot poll runs in the main context with no yield risk).
- For `/fab-ff` the light lane covers task execution + hydrate (its terminal); ship/review-pr apply to `/fab-fff` only.

### 5. REVIEW stays a fresh dispatched worker in BOTH lanes — never inline

Reviewer independence is the pipeline's highest-value dispatch: author self-review shares the author's blind spots, and sweep misses are this repo's top rework cause (code-quality.md § Sibling & Mirror Sweeps). A fresh reviewer over a tiny diff is cheap anyway.

### 6. NO promotion valve — deliberately removed

Light rework stays inline under the same max_cycles budget (code-review.md Rework Budget, default 3) with the same fail+reset per-cycle choreography; exhaustion parks `review: failed` exactly as today. Safe because every diff — light or full — passes the same independent review: a misclassified small change wastes bounded inline rework cycles but can never ship unreviewed. A parked light run re-enters however the user chooses, including `--full`. Scope discovered mid-rework (plan revision adding tasks) rides the same backstop rather than any automatic promotion.

Light rework also gets worker-continuation for free: the orchestrator IS the apply author and remembers what the reviewer rejected — the `_preamble.md` § Worker Continuation apparatus (named handles, pane delivery, fallbacks) becomes a full-lane-only concern.

### 7. FULL lane = today's bracket verbatim

Except that the plan pre-exists (point 2). All three dispatch adapters (native/pane/headless), worker continuation, and recovery budgets are unchanged.

### Rejected Alternatives

- **Change-type entry heuristic** (infer light from docs/chore/test/ci at intake): rejected on empirical evidence — loom archive analysis (972 completed changes, 927 with recoverable task counts) shows 68% of type-eligible changes outgrow the size threshold while 154 small feat/fix changes escape it (type recall ~26%, precision ~32%). Task count is the signal; type is noise.
- **Promotion valve** (light→full mid-run on 2nd rework cycle): removed for simplicity; bounded rework + exhaustion parking is a sufficient backstop; zero mid-run mode machinery.
- **New stage states** (e.g. a merged light-done): would fork every `.status.yaml` consumer.
- **Promoting ship/review-pr to full CLI-adapter workers for mechanism symmetry**: rejected — dual-contract drift (block variant + standalone variant of the same skill), thin payoff (ship is gh plumbing, no model-quality dimension), and review-pr owns a deliberately non-terminal outcome (Copilot timeout leaves the stage active) that the five-state dispatch machine cannot express.

### Recorded follow-ups (NOT in scope)

- `light_max_tasks` config knob (threshold stays hardcoded at 5 in v1).
- A `weight` event in `.history.jsonl` for measurement/resume-consistency.
- Requesting the Copilot review at ship time to overlap the poll.

### Sweep classes and mirror obligations

- ~~Constitution 1.5.0 SPEC-mirror obligation~~ **OBSOLETED mid-pipeline by rebase onto v2.19.5**: PR #582 (constitution 1.6.0) deleted the `docs/specs/skills/SPEC-*.md` tree and retired the mirror rule, absorbing structural content into `docs/specs/skills.md`. The mirror updates this change made were dropped at rebase; their reviewed content was transplanted into `docs/specs/skills.md`'s absorbed `_pipeline` flow skeleton and the `/fab-fff` ship/review-pr flow lines.
- Known sweep classes (code-quality.md § Sibling & Mirror Sweeps): twin skills `fab-ff` ↔ `fab-fff`; aggregate specs restating per-skill facts (`docs/specs/skills.md`, `glossary.md`, `architecture.md`); the memory files documenting pipeline behavior (`docs/memory/pipeline/execution-skills.md`, `change-lifecycle.md` § Pipeline shortcuts).
- Owner-or-pointer rule: state a rule once, point everywhere else.

## Affected Memory

- `pipeline/execution-skills.md`: (modify) light-lane/full-lane execution loci for /fab-ff and /fab-fff, inline plan co-gen at apply entry, the one-time task-count fork, review-always-dispatched invariant, rework-loop locus
- `_shared/context-loading.md`: (modify) the "every post-intake stage dispatches a sub-agent ... no post-intake foreground execution path" claim gains the light-lane inline exception (added during review rework — sweep-class member the original list missed)
- `pipeline/change-lifecycle.md`: (modify) § Pipeline shortcuts paragraph — the two-lane bracket, unchanged state machines, `--light`/`--full` overrides

## Impact

- `src/kit/skills/_pipeline.md` — the bracket: inline co-gen step at apply entry (both lanes, with the large-scope MAY-skip guard), the one-time fork (≤ 5 tasks → light), light-lane execution rules (inline apply/hydrate; inline ship/review-pr for fff), rework-loop locus notes (inline rework; Worker Continuation becomes full-lane-only)
- `src/kit/skills/fab-fff.md` — `--light`/`--full` arguments; Steps 4-5 (ship, review-pr) run inline in the light lane; the Step 5 synchronous-poll directive becomes moot in the light lane
- `src/kit/skills/fab-ff.md` — twin sweep: same arguments and fork; light lane covers task execution + hydrate (its terminal)
- ~~`docs/specs/skills/SPEC-*.md` mirrors~~ — tree deleted upstream by PR #582 mid-pipeline; the structural updates live in `docs/specs/skills.md`'s absorbed flow skeletons instead (constitution 1.6.0 retired the mirror rule)
- `docs/specs/skills.md` — aggregate spec restating per-skill facts
- `docs/specs/glossary.md` — possibly a "light lane" term entry
- `docs/findings/light-mode-state-machines.md` + `.html` — design reference, already in-tree, included in this change
- Zero Go changes; zero `.status.yaml` schema changes; zero config registry changes (`true_impact_exclude` covers `fab/` and `docs/` — true impact is the three `src/kit/skills/*.md` files)

## Open Questions

None — the 2026-08-11 design discussion settled every decision point (see `## Assumptions`; no Unresolved rows emerged that would have been asked interactively).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both state machines gain zero states and zero transitions; same finish/fail/reset choreography in both lanes; orchestrator stays the pure sequencer; review cycle-count invariant holds verbatim | Discussed — hard constraint settled 2026-08-11; design doc in-tree | S:95 R:90 A:95 D:95 |
| 2 | Certain | plan.md co-generation moves inline into the orchestrator at apply entry, ALWAYS, both lanes; full-lane apply worker receives the finished plan via the existing plan-exists seam | Discussed — rides the "unless plan.md exists" seam already in the apply contract | S:95 R:85 A:90 D:90 |
| 3 | Certain | One-time fork after inline co-gen: task count ≤ 5 → LIGHT, > 5 → FULL; threshold hardcoded at 5 in v1 (`light_max_tasks` knob is a recorded follow-up) | Discussed — loom-archive evidence chose task count over type; v1 is prose-only per Constitution I | S:95 R:90 A:90 D:95 |
| 4 | Certain | Per-invocation `--light` / `--full` flags on /fab-ff and /fab-fff skip the count check | Discussed — explicit override decided | S:90 R:90 A:90 D:90 |
| 5 | Certain | LIGHT lane runs apply task execution, hydrate, ship, and review-pr INLINE in the orchestrator's context, following the same behavior sections workers follow today; /fab-ff light lane = task execution + hydrate only | Discussed — inline ship/review-pr is today's standalone path; those skills keep managing their own stage transitions | S:90 R:85 A:90 D:90 |
| 6 | Certain | REVIEW stays a fresh dispatched worker in BOTH lanes, never inline | Discussed — reviewer independence is the highest-value dispatch; author self-review shares blind spots | S:95 R:85 A:95 D:95 |
| 7 | Certain | NO promotion valve; light rework stays inline under the same max_cycles (3) with same fail+reset choreography; exhaustion parks review: failed; parked run re-enters per user choice incl. --full; mid-rework scope growth rides the same backstop | Discussed — deliberately removed; independent review makes misclassification safe (bounded waste, never ships unreviewed) | S:90 R:85 A:90 D:90 |
| 8 | Certain | FULL lane = today's bracket verbatim except plan pre-exists; all three adapters, worker continuation, recovery budgets unchanged; § Worker Continuation becomes a full-lane-only concern | Discussed — graceful degradation target | S:90 R:85 A:90 D:90 |
| 9 | Certain | v1 is skill-prose only: zero Go, no .status.yaml schema change, no config registry change; the lane lives in the orchestrator's context for the run (no persisted lane marker; `weight` history event is a follow-up) | Discussed — Constitution I Pure Prompt Play; follow-ups explicitly recorded out of scope | S:95 R:85 A:95 D:90 |
| 10 | Certain | Inline stages skip `fab resolve-agent` and run on the session model | Discussed — consistent with existing undispatched-stage rule (MAY report profile, MUST NOT switch session model) | S:85 R:90 A:90 D:90 |
| 11 | Certain | SPEC mirrors for _pipeline, fab-ff, fab-fff update in the same change (flow + sub-agent structure change per Constitution 1.5.0); sweep classes: twin skills, aggregate specs (skills.md, glossary.md, architecture.md), pipeline memory files | Discussed — constitution trigger fires; known sweep classes named in discussion | S:95 R:80 A:95 D:95 |
| 12 | Certain | Rejected alternatives recorded in intake: change-type entry heuristic (empirical loom evidence), promotion valve, new stage states, promoting ship/review-pr to CLI-adapter workers | Discussed — all four rejections settled with rationale | S:95 R:90 A:95 D:95 |
| 13 | Confident | Large-scope guard criterion: the orchestrator MAY skip inline co-gen and dispatch apply-with-co-gen as today when the intake's affected scope is "obviously large" — exact criterion left to orchestrator judgment in prose | Discussed as a MAY guard; threshold wording unspecified, but degradation path is the shipped behavior so any reasonable criterion is safe | S:45 R:85 A:65 D:55 |
| 14 | Certain | Task count = number of task entries in the co-generated plan.md `## Tasks` (all phases) | Natural reading of "task count" against the plan template's T{NNN} entries; not spelled out verbatim in discussion | S:65 R:90 A:85 D:80 |
| 15 | Confident | `--light` and `--full` are mutually exclusive; passing both is rejected as a usage error in prose | Not discussed explicitly; obvious default for contradictory flags, trivially reversible prose | S:35 R:90 A:75 D:65 |
| 16 | Confident | docs/specs/glossary.md gains a "light lane" term entry | Description says "possibly"; glossary is the terminology owner and the term is new, low-cost to add | S:40 R:95 A:70 D:60 |
| 17 | Confident | docs/findings/light-mode-state-machines.md's "Status: design proposal" header is updated to reflect adoption when the files ride along in this change | Not discussed; keeping a shipped mechanism labeled "proposal" would violate present-truth; trivially reversible | S:30 R:90 A:60 D:50 |

17 assumptions (13 certain, 4 confident, 0 tentative, 0 unresolved).
