# Intake: Collapse post-intake dual execution mode — one execution mode, orchestrators as sequencers

**Change**: 260613-fgxx-collapse-post-intake-dual-mode
**Created**: 2026-06-13

## Origin

This is **Change A** — the highest-risk of three coordinated refactors that fall out of an
architecture discussion about fab-kit's skill structure. The full analysis lives in two findings
docs in this repo (both already written; read them in full before planning):

- `docs/findings/intake-is-the-context-boundary.md` — the primary source for this change
- `docs/findings/per-stage-model-tier-application.md` — its companion (this change closes that
  finding's Gap 1a)

The verbatim source description that initiated this change:

> refactor: Collapse post-intake dual execution mode — one execution mode, orchestrators as
> sequencers. The guiding principle (newly named): INTAKE IS THE SOLE CONTEXT BOUNDARY. Main
> session context runs up to and including intake creation; after intake, the intake artifact IS
> the context, and every stage (apply → review → hydrate → ship → review-pr) runs as a dispatched
> subagent fed by artifacts (intake.md + plan.md + docs/memory/), never by conversation history.
> Today this principle is violated by a dual execution mode baked into each post-intake block.

**Interaction mode**: one-shot draft (`/fab-draft`), created without activation. Natural-language
input — no Linear/backlog ID.

**Dependency / sequencing context** (from the execution plan in the discussion):

- This change is **INDEPENDENT** of Change B (extract `_intake.md` — the pre-boundary
  de-duplication described in the same finding under "Pre-boundary de-duplication: extract
  `_intake.md`"). B is NOT in scope here.
- Change C (uniform per-stage model tier) **DEPENDS ON this change** — "uniform tier application"
  only becomes true once there is a single post-intake execution mode. C must follow A. C is NOT
  in scope here, but this change should leave the door open for it (see § Why, point on Gap 1a).

## Why

### The guiding principle (newly named): INTAKE IS THE SOLE CONTEXT BOUNDARY

There is exactly **one context-bearing boundary** in the whole pipeline: **intake**.

- **Up to and including intake creation** → runs in the **main session context**, because it needs
  the live conversation. This is `/fab-new`, `/fab-draft`, `/fab-proceed`'s intake-creation prefix,
  and `/fab-clarify` (interactive intake refinement — clarify is **pre-boundary**, runs in main
  context). These have zero context breakage — the conversation is right there.
- **After intake** → the intake artifact **IS the context**. Every subsequent stage
  (apply → review → hydrate → ship → review-pr) runs as a **dispatched subagent** that reconstructs
  its context from artifacts (`intake.md` + `plan.md` + `docs/memory/`), never from conversation
  history.

Stated as a rule:

> **Main context ≤ intake. Dispatched, artifact-fed blocks > intake. The intake is the handoff
> payload across the boundary.**

### How the principle is violated today

The principle is violated by a **dual execution mode** baked into each post-intake block.
`src/kit/skills/fab-continue.md` carries **three** caller-aware conditionals (currently at lines
**100, 156, 176** in both the source and the deployed copy) of the form:

> *"When invoked as a subagent (dispatched by `/fab-ff`/`/fab-fff`): do NOT run any `fab status`
> command — the orchestrator owns the transitions."*

This means a stage block behaves **differently** depending on whether it runs:

- **Foreground (Path A)**: plain `/fab-continue` runs the stage in-session and owns its own
  `fab status` transitions.
- **Dispatched (Paths B/C/D)**: `/fab-ff`, `/fab-fff`, `/fab-proceed` dispatch the block as a
  subagent and the orchestrator owns the transitions itself.

A Lego brick that behaves differently depending on which baseplate it's snapped onto is not a Lego
brick. The work layer is *already* Lego (`_generation.md`, `_review.md`, hydrate behavior are clean
caller-agnostic procedures). What's tangled is the **bookkeeping + context skin** around each block.

### The duplicated rework state machine

The dual mode ALSO duplicates the **rework state machine**. The fail→reset→re-apply→re-review
choreography exists **twice**:

1. once as `fab-continue.md`'s interactive **Verdict** rework menu (Path A — human chooses
   fix-code / revise-plan / revise-requirements), and
2. once as `_pipeline.md`'s autonomous **Auto-Rework Loop** (Paths B/C/D — bounded autonomous
   triage, default 3 cycles).

### What fixing it buys

1. **The dual-mode conditional collapses, not by deletion but by elimination of the fork.** With
   exactly one post-intake execution mode (dispatched), the conditional never has a second branch
   to express.
2. **It closes Gap 1a of the model-tier finding** (`per-stage-model-tier-application.md`). Gap 1a:
   foreground stages can't be tiered because `fab` can't switch the session model mid-run, so
   `fab resolve-agent` is "advisory only" for manual `/fab-continue` apply/hydrate/ship. If **every**
   post-intake stage dispatches a subagent regardless of A/B/C/D, then `fab resolve-agent` applies
   uniformly on every post-intake stage — there is no non-dispatch path left to be the exception.
   This is what makes Change C ("uniform per-stage model tier") become true, hence C depends on A.
3. **It makes "the block is a Lego" a testable invariant**: a block is correct iff it produces the
   same result given the same `intake.md` + `plan.md` + memory, regardless of what conversation
   preceded it.

## What Changes

> **Scope discipline**: this is a **behavior reorganization**, not a feature addition. No new `fab`
> CLI commands are introduced (so `src/kit/skills/_cli-fab.md` is NOT expected to change). The Go
> state machine is **not modified** — only a Go *test* may be added (see § 5). The change is
> overwhelmingly a skills-prose + specs-prose refactor with an optional Go-test component.

### 1. Make post-intake `/fab-continue` ALWAYS dispatch a subagent for its stage

Post-intake `/fab-continue` (apply / review / hydrate stages) MUST **always dispatch a subagent**
for its stage — the **SAME dispatch** the orchestrators (`_pipeline.md`) already do per
`_preamble.md` § Subagent Dispatch. There is no longer an in-session foreground execution path for
apply/review/hydrate. Plain `/fab-continue apply` dispatches a subagent just like `/fab-fff` does.

- Affected file: `src/kit/skills/fab-continue.md` — its **Normal Flow** (Step 1 dispatch table,
  Step 3 SRAD + Generation table at current lines ~74–76, Step 4 status update, Step 5 output) and
  the **Apply / Review / Hydrate Behavior** sections.
- The intake stage and the ship/review-pr stages are **out of this change's behavioral rewrite**:
  - **intake** is pre-boundary (main-context) — it is the boundary itself, not a post-intake stage.
  - **ship** (`/git-pr`) and **review-pr** (`/git-pr-review`) already self-manage their own stage
    transitions internally (documented in `fab-fff.md` as the dispatch exception). They are NOT part
    of the three dual-mode conditionals. Confirm during planning that nothing in this change
    regresses their self-managed transition behavior; do not change it.

### 2. Remove the three dual-mode `do NOT run fab status` conditionals from `fab-continue.md`

Remove these three blockquote conditionals (verbatim current text):

- **Apply Behavior, line ~100**:
  > **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): do NOT run any `fab status`
  > command — skip the finish steps below and return results only. The orchestrator owns all status
  > transitions.

- **Review Behavior, line ~156**:
  > **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): skip §Verdict entirely — do
  > NOT run any `fab status` command; return the merged findings with pass/fail status only. The
  > orchestrator owns the finish/fail/reset transitions (and the rework loop).

- **Hydrate Behavior, line ~176**:
  > **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): skip step 5 (the finish) —
  > do NOT run any `fab status` command; return completion status only. The orchestrator owns the
  > transition.

With one execution mode there is no second branch to express. Whether the block or the orchestrator
owns the `fab status` call becomes a free implementation choice (**recommended: orchestrator-as-
sequencer**, see § 3), not a forked behavior baked into the block.

### 3. Demote orchestrators to PURE SEQUENCERS

Orchestrators (`_pipeline.md`, and by extension the `/fab-ff` / `/fab-fff` / `/fab-proceed` wrappers)
become pure sequencers:

> **dispatch block → read returned status/findings → decide proceed / loop / stop.**

They own the `fab status` transitions (`finish` / `fail` / `reset`) and the `fab resolve-agent
<stage>` per-stage model resolution; they **never reach into block internals**. The block behaviors
(**Apply Behavior**, **Review Behavior**, **Hydrate Behavior** in `fab-continue.md`) become the
**SINGLE shared definitions** that both `/fab-continue` (manual) and `_pipeline` (auto) dispatch
**identically**.

Recommended design for `resolve-agent` ownership (per the finding's open seam #3): the
`fab resolve-agent <stage>` call lives in the **orchestrator/sequencer**, consistent with
"invocations control order / grouping / model-tier; blocks control work." This keeps the tier
decision out of the block, matching the Lego model. `/fab-continue` in its new always-dispatch form
*is* a one-stage sequencer, so it too runs `fab resolve-agent <stage>` immediately before dispatching
its stage's subagent (mirroring `_pipeline.md`'s existing pre-dispatch resolution at its Steps 1/2/3
and the rework loop's items 3/4).

`_pipeline.md` already implements the orchestrator-as-sequencer shape (its § Behavior Dispatch note
on line 48 already states the orchestrator runs the transitions; Steps 1–3 already dispatch
`/fab-continue` subagents with "do NOT run `fab status`; return results only"). The change here is to
make `/fab-continue`'s **own** post-intake path do the same dispatch, so the prompt-level "do NOT run
`fab status`" instruction the orchestrator passes becomes the **universal** contract for the block
(no longer a per-caller conditional living inside the block).

### 4. The autonomous rework loop is the HIGHEST-RISK seam — handle deliberately

The autonomous rework loop (`_pipeline.md` § Auto-Rework Loop) is the **one place** post-intake work
makes **DECISIONS**: it triages the review block's findings → picks one of fix-code / revise-plan /
revise-requirements → edits `plan.md`. Under the principle it MUST run dispatched, fed by the review
block's **STRUCTURED returned findings + `plan.md`** — **never by conversation**.

Constraints (all MUST hold):

- **Keep the failure-policy fork in the orchestrator**, not the block. *Who decides on failure* is
  legitimately invocation-level:
  - **Path A** (`/fab-continue` review, foreground/manual): the **interactive human rework menu**
    (`fab-continue.md` § Verdict — the **Fail** options table: Fix code / Revise plan / Revise
    requirements). The human now reads the *dispatched* review subagent's returned findings rather
    than findings produced in-session, but the menu and the choice stay human. **This menu stays.**
  - **Paths B/C/D** (`/fab-ff` / `/fab-fff` / `/fab-proceed`): the **bounded autonomous triage** in
    `_pipeline.md` § Auto-Rework Loop (default 3 cycles, escalation after 2 consecutive fix-code).
    **This loop stays.**
- **Keep findings as the block's RETURN VALUE.** Findings are a return value, not conversation — that
  is what makes the dispatched rework loop achievable. The Review Behavior block's job is identical
  in both modes: review the diff, return pass/fail + prioritized must-fix / should-fix /
  nice-to-have findings.
- **NEVER give the block a "skip §Verdict when subagent" flag.** The very conditional being removed
  (the line-156 "skip §Verdict entirely" instruction) must NOT be reintroduced in a different shape.
  The block always returns findings; whether a §Verdict-style decision runs, and who runs it, is the
  orchestrator's concern (interactive menu in A, autonomous triage in B/C/D). The block does not
  branch on caller.

### 5. Optional Go-side spike: protect the history-shape invariant under uniform dispatch

`_pipeline.md` guards a `.status.yaml` history-shape invariant (verbatim, `_pipeline.md` line 86):

> *"This fail+reset pair repeats on **every** failed review verdict that starts a new cycle — not
> just the first failure — so every conforming run leaves the same `.status.yaml` history shape."*

This invariant is enforced by the **CALL SEQUENCE**, not by caller identity. A uniformly-dispatched
apply MUST NOT break it: the orchestrator still issues the same `fail → reset → finish` sequence on
every rework cycle.

**RECOMMENDED** (per the description's critical-invariant note): before finalizing the plan, do a
Go-side spike — **add or confirm a test** that the rework-loop status sequence under uniform
dispatch produces the **same `.status.yaml` history** as the manual path. Because the Go state
machine is already caller-agnostic (see § KEY VERIFIED EVIDENCE below), the manual path and the
dispatched path issue the *same* CLI calls and thus produce the same history; the test pins this so a
future regression in the skills layer that diverges the two call sequences is caught.

Where such a test would live (from repo survey):

- `src/go/fab/internal/status/mutators_test.go` — already hosts
  `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` (the rework-cycle accumulation test,
  ~lines 300–336), which drives the exact `Finish(intake) → Finish(apply) → [Fail(review) →
  Reset(apply) → Finish(apply)]×N` sequence. This is the natural host for a "history shape is
  identical regardless of driver string" assertion.
- `src/go/fab/internal/status/status_test.go` — hosts `TestResetCascade_PreservesReviewIterations`
  (~lines 372–422), which exercises the `Fail(review) → Reset(apply)` choreography and the cascade.
- The transition log itself is written by `log.Transition` in
  `src/go/fab/internal/log/log.go` (`appendJSON` writes `{change-dir}/.history.jsonl`). The `driver`
  field is written as an *optional* entry key only when non-empty.

A concrete assertion shape for the spike: drive the rework sequence twice — once with `driver="""`
(the foreground/manual path: `Finish`'s side-effect passes `driver=""` for the finishing stage; see
evidence) and once with `driver="fab-fff"` — and assert the produced `.history.jsonl` entries are
**identical in count, stage, action, from, and reason**, differing **only** in the optional `driver`
field. That difference is expected and acceptable (it is the recorded driver name, not a state
difference).

**If the spike confirms an existing test already covers this** (e.g.,
`TestStageMetrics_IterationsAccumulateAcrossReworkCycles` is deemed sufficient), the Go component may
be a one-line confirmation in the plan rather than new code — keep it minimal. This change **MAY**
grow a small Go test component; it MUST NOT modify Go production code.

## Affected Memory

<!-- pipeline + runtime domains per the description's "Affected Memory" note. These hydrate AFTER
     apply/review (during the hydrate stage); listing them here scopes the plan. -->

- `pipeline/execution-skills.md`: (modify) Rewrite the `/fab-continue` execution model to a single
  always-dispatch post-intake mode; remove documentation of the dual foreground-vs-subagent fork and
  the three `do NOT run fab status` conditionals; document orchestrators-as-pure-sequencers and the
  single shared Apply/Review/Hydrate block definitions; clarify that the interactive Verdict rework
  menu (Path A) and the autonomous Auto-Rework Loop (Paths B/C/D) are the invocation-level
  failure-policy fork that remains.
- `pipeline/schemas.md`: (modify, if needed) Note that the `.status.yaml` history-shape invariant
  (fail+reset on every rework cycle) is enforced by the call sequence and is caller-identity-blind —
  byte-identical state regardless of foreground vs. dispatched origin; only the recorded `driver`
  differs. Modify only if the spike adds/clarifies a Go test that the memory should reference.
- `runtime/operator.md`: (modify) Reconcile the operator's multi-agent dispatch model with the
  single-dispatch post-intake model. **Committed via /fab-clarify (2026-06-13, Assumption #8)** — the
  operator coordinates post-intake agent dispatch, so it is expected to need alignment. Hydrate
  verifies the precise edit; if operator.md turns out to make no foreground-path assumption, hydrate
  records that and the edit is a no-op — but the entry is no longer conditional.

> Determine the precise edits during hydrate against the actual diff. All three entries
> (`pipeline/execution-skills.md`, `pipeline/schemas.md`, `runtime/operator.md`) are now committed
> targets; `runtime/operator.md`'s exact change is hydrate-verified per Assumption #8.

## Impact

### Source files (skills — canonical at `src/kit/skills/`, NEVER edit `.claude/skills/`)

- **`src/kit/skills/fab-continue.md`** — primary edit target. Remove the three conditionals
  (lines ~100/156/176); rewrite Normal Flow + Apply/Review/Hydrate Behavior so the post-intake path
  always dispatches a subagent and runs `fab resolve-agent <stage>` before dispatch; keep the
  interactive § Verdict rework menu (Path A) intact; keep the foreground header note
  (currently line 19) coherent — with no foreground execution path for apply/review/hydrate, the
  "foreground = advisory-only" framing for those stages no longer applies and should be reconciled
  (the note may shrink to "the *one-stage sequencer* still resolves the tier and dispatches").
- **`src/kit/skills/_pipeline.md`** — confirm/affirm the orchestrator-as-pure-sequencer shape
  (already largely present: § Behavior Dispatch note line 48, Steps 1–3, Auto-Rework Loop). Likely
  light edits to align prose with the new single-mode block (e.g., the "do NOT run `fab status`"
  prompt instruction is now the universal block contract, not a per-caller override). The Auto-Rework
  Loop per-cycle choreography (the fail+reset pair, line 86 invariant) MUST be preserved verbatim in
  semantics.
- **`src/kit/skills/fab-ff.md` / `fab-fff.md` / `fab-proceed.md`** — survey for any prose that
  presumes the *block* owns transitions in some paths; these are thin wrappers over `_pipeline.md`
  and already subagent-only, so expect little/no change. fab-proceed's prefix steps are NOT pipeline
  stages and are out of scope.
- **`src/kit/skills/_preamble.md`** — the § Subagent Dispatch → Per-Stage Model Resolution
  "**Foreground is advisory-only**" paragraph (line ~350) describes the dual mode being collapsed.
  Reconcile: with no foreground execution path for post-intake stages, the advisory-only carve-out
  for apply/hydrate/ship narrows or disappears. Edit carefully — this paragraph is referenced by
  multiple skills and by `stage-models.md`. (Note: full closure of Gap 1a / uniform tiering is
  **Change C**; this change need only stop *contradicting* the single-mode model — it does not have
  to fully rewrite the tiering story.)

### Spec files (MUST be updated — constitution: skill changes MUST update `docs/specs/skills/SPEC-*.md`)

- **`docs/specs/skills/SPEC-fab-continue.md`** — remove/replace the three conditional descriptions
  (mirrors at lines 100/156/176 of the spec); update the "foreground = advisory-only" summary
  (line 9) and the Summary to reflect single-mode post-intake dispatch; keep the Verdict
  rework-menu spec (lines ~158–169) as the Path A behavior.
- **`docs/specs/skills/SPEC-_pipeline.md`** — align the dispatch/transition-ownership prose with the
  pure-sequencer framing.
- **`docs/specs/skills/SPEC-fab-ff.md` / `SPEC-fab-fff.md` / `SPEC-fab-proceed.md`** — update only if
  the corresponding skill prose changes.
- **`docs/specs/stage-models.md`** — the "Foreground limitation (advisory only)" section
  (~lines 263–272) describes the exact scenario being collapsed for post-intake stages. Reconcile
  with the single-mode model (note that full Gap-1a/uniform-tier closure is Change C; this change
  must not leave the section contradicting the single execution mode).

### Go (production code: NO change; test: optional small addition)

- **`src/go/fab/internal/status/status.go`** — **NOT modified**. Read-only evidence: `Finish`
  (160–209), `Reset` (211–243), `Skip`, `Fail`, `applyMetricsSideEffect` (617+).
- **`src/go/fab/internal/status/mutators_test.go`** and/or **`status_test.go`** — optional small
  test addition (the rework-loop history-shape spike, § 5).
- **`src/kit/skills/_cli-fab.md`** — NOT expected to change (no command signatures change). Confirm.

### Out of scope

- Change B (extract `_intake.md`) — pre-boundary de-duplication; independent of this change.
- Change C (uniform per-stage model tier) — depends on this change; full Gap-1a closure / effort-knob
  work belongs there.
- intake / ship / review-pr behavioral rewrites — intake is the boundary; ship/review-pr already
  self-manage transitions.

## Open Questions

- How much of `_pipeline.md` actually needs editing vs. merely re-affirming? It already implements
  the orchestrator-as-sequencer pattern (line 48, Steps 1–3, Auto-Rework Loop). The plan should
  determine whether this change is mostly **deletion in `fab-continue.md`** + **light prose
  alignment elsewhere**, or whether `_pipeline.md` needs structural change. (Leaning: mostly
  fab-continue.md deletion + alignment.)
- ~~Does `runtime/operator.md` actually assume the foreground post-intake path anywhere?~~
  **RESOLVED (/fab-clarify 2026-06-13): operator.md WILL be reconciled** to the single-dispatch model
  (Assumption #8, now Confident). Hydrate verifies the specific edit; if operator.md makes no
  foreground assumption, hydrate records that and the edit is a no-op — but the direction (reconcile,
  not "decide later") is decided.
- ~~Is `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` already sufficient to pin the
  history-shape invariant, making the § 5 spike a confirmation rather than a new test?~~
  **RESOLVED (/fab-clarify 2026-06-13): a new test IS added** in `mutators_test.go` regardless of
  spike outcome (Assumption #7, now Certain). The spike only informs the exact assertion shape, not
  whether the test exists. No production Go change — test-only.
- What is the right resting shape of the `_preamble.md` "Foreground is advisory-only" paragraph and
  the `stage-models.md` foreground section, given that full uniform-tier closure is deferred to
  Change C? (Minimum: stop contradicting single-mode; do not over-reach into Change C's territory.)

## KEY VERIFIED EVIDENCE (confirmed against the Go source on 2026-06-13, fab-kit v2.3.1)

The Go state machine is **ALREADY caller-agnostic**, which is why this is a smaller refactor than it
looks — the duality is **100% a skill-prose artifact, not a CLI one**.

In `src/go/fab/internal/status/status.go`:

- `Start`, `Advance`, `Finish`, `Reset`, `Skip`, `Fail` take a `driver` string, but it flows **ONLY**
  into `applyMetricsSideEffect` (status.go:617) — it sets `sm.Driver` (stage-metrics, line 629) and
  the transition log via `log.Transition` (line 638). **NO state transition reads `driver`.**
  `lookupTransition` (called at lines 169 and 218) keys on the event + current state only.
- `Finish`'s auto-activate-next (status.go:179–189) is computed purely from `NextStage` / `StageOrder`
  + the current progress map — caller-identity-blind. (Subtlety, verified: `Finish` passes
  `driver=""` to the side-effect for the **finishing** stage at line 177, and the caller's `driver`
  only to the **auto-activated next** stage at line 187 — so even within a single `Finish` call the
  driver is purely an annotation on the newly-activated stage, never a transition input.)
- `Reset`'s downstream cascade (status.go:228–240) walks `StageOrder` and sets each downstream stage
  to `pending` with `driver=""` — caller-identity-blind.
- **CONSEQUENCE**: whether `/fab-continue` (foreground) or `_pipeline` (orchestrator) calls
  `fab status finish apply`, the resulting `.status.yaml` **STATE** is byte-identical; only the
  recorded `driver` name differs. The transition log entry differs only in its optional `driver`
  field.

This is the structural reason the refactor is safe: collapsing the two skill-prose modes into one
cannot change `.status.yaml` state, because the CLI never branched on caller in the first place.

## Constitution constraints (apply during apply/review)

- `src/kit/` is canonical; `.claude/skills/` is gitignored deployed copies — **edit ONLY source**
  (`src/kit/skills/*.md`). The deployed `.claude/skills/` copies are produced by `fab sync` and must
  never be hand-edited.
- **Skill file changes MUST update the corresponding `docs/specs/skills/SPEC-*.md`** (constitution
  Additional Constraints) — and reconcile `docs/specs/stage-models.md` where it describes the
  collapsed foreground mode.
- Any Go change MUST include test updates (Test Integrity principle VII) and MUST update
  `src/kit/skills/_cli-fab.md` if command signatures change — **none are expected here** (this is
  behavior reorganization, not new commands). The only anticipated Go change is a *test*; no
  production Go change is in scope.
- Docs are source of truth (principle II); specs are pre-implementation design intent (principle VI)
  — update specs and memory to match shipped behavior.

## Bootstrapping caution

These are skill-architecture changes to **fab-kit itself**, executed BY fab-kit's pipeline. If this
change is mid-flight and breaks `_pipeline.md`, the very pipeline running the change is affected.
Memory note carried forward: **fresh-worktree `.claude/skills` lag `src/kit` — orchestrate from
`src/kit`** (and run `fab sync` to redeploy before exercising the changed skills). Favor **small,
independently-revertable steps**: the natural ordering is (1) delete the three conditionals + rewrite
`fab-continue.md` post-intake dispatch, (2) align `_pipeline.md` / wrappers / `_preamble.md` /
`stage-models.md` prose, (3) update the SPEC-*.md mirrors, (4) the Go history-shape test (now a
committed test, not optional — see Assumption #7) — each a revertable unit.

## Clarifications

### Session 2026-06-13 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 9 | Confirmed | After explanation (minimal-reconcile vs. full rewrite; A neutralizes contradiction, C writes the uniform-tiering story) |

### Session 2026-06-13 (tentative resolution)

| # | Q | A |
|---|---|---|
| 7 | Add a new Go history-shape test, or rely on the existing iteration test? | **Add a new test** in `mutators_test.go` regardless of spike outcome — promoted Tentative → Certain. |
| 8 | Update `runtime/operator.md`, or leave it? | **Reconcile it** (will-update direction) — promoted Tentative → Confident on a follow-up pass; hydrate verifies the precise edit. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Slug `collapse-post-intake-dual-mode`; change type `refactor` | Description is prefixed `refactor:` and is a behavior reorganization; the intake-write hook infers `refactor` from the keyword. Determined by template/hook rules. | S:95 R:90 A:95 D:95 |
| 2 | Certain | `src/kit/skills/*.md` edited (never `.claude/skills/`); SPEC-*.md mirrors updated; `stage-models.md` reconciled | Constitution mandates canonical-source-only edits and SPEC updates for skill changes. No interpretation needed. | S:90 R:85 A:100 D:95 |
| 3 | Certain | No new `fab` CLI commands; `_cli-fab.md` unchanged; no Go production-code change | Description states explicitly "behavior reorganization, not new commands"; Go evidence shows the state machine is already caller-agnostic. | S:95 R:90 A:95 D:90 |
| 4 | Confident | The three conditionals to remove are at lines ~100 / 156 / 176 of `fab-continue.md` (source and deployed copy verified to match) | Clarified — user confirmed (2026-06-13). Verified by grep against `src/kit/skills/fab-continue.md`; line numbers may drift but the three blockquotes are uniquely identifiable by their text. | S:95 R:80 A:85 D:85 |
| 5 | Confident | Orchestrator owns `fab status` + `fab resolve-agent`; the block never owns transitions (orchestrator-as-pure-sequencer) | Clarified — user confirmed (2026-06-13). The finding's open-seam #3 recommends this; `_pipeline.md` already implements it. | S:95 R:70 A:85 D:80 |
| 6 | Confident | Interactive Verdict menu (Path A) and autonomous Auto-Rework Loop (Paths B/C/D) both REMAIN as the invocation-level failure-policy fork; no "skip §Verdict when subagent" flag is reintroduced | Clarified — user confirmed (2026-06-13). Description § 4 and finding open-seam #1 state this explicitly as a hard constraint. | S:95 R:75 A:90 D:85 |
| 7 | Certain | A new Go test (history-shape-under-uniform-dispatch) **IS added**, hosted in `mutators_test.go` near `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` — it drives the rework sequence with `driver=""` and `driver="fab-fff"` and asserts the `.history.jsonl` entries are identical except the optional `driver` field. This is no longer contingent on the spike outcome: the test is added regardless; the spike only informs its exact assertion shape. | Clarified — user confirmed (2026-06-13): commit to adding the new test rather than relying on the existing iteration test. A test-only addition, no production Go change. <!-- clarified: new Go history-shape test committed — user-decided 2026-06-13 --> | S:95 R:85 A:90 D:90 |
| 8 | Confident | `runtime/operator.md` **will be reconciled** to the single-dispatch post-intake model — the operator coordinates post-intake agent dispatch, so it almost certainly documents/assumes how those stages run and must align with uniform dispatch. Hydrate verifies and performs the reconcile; if hydrate finds operator.md genuinely makes no foreground-path assumption, it records that and the edit is a no-op (the direction, not a contingency, is what's now decided). | Clarified — user confirmed (2026-06-13): commit to reconciling operator.md rather than leaving it a hydrate-time if/else. Direction decided; hydrate is the verifier, not the decider. <!-- clarified: operator.md reconcile committed (will-update direction) — user-decided 2026-06-13; hydrate verifies --> | S:80 R:75 A:70 D:75 |
| 9 | Confident | `_preamble.md` "Foreground is advisory-only" paragraph and `stage-models.md` foreground section are minimally reconciled (stop contradicting single-mode) rather than fully rewritten — full uniform-tier closure is deferred to Change C | Clarified — user confirmed after explanation (2026-06-13): minimal-reconcile is the chosen approach; A must neutralize the contradiction but NOT write Change C's uniform-tiering story (avoids docs-ahead-of-code + edit-collision with C). Wording is an apply-time judgment band, hence Confident not Certain. <!-- clarified: minimal reconcile of advisory-only prose; full tiering rewrite deferred to Change C — user-confirmed 2026-06-13 --> | S:95 R:75 A:80 D:80 |

9 assumptions (4 certain, 5 confident, 0 tentative). All resolved via /fab-clarify (2026-06-13): row 7 (Go history-shape test) → Certain (test committed regardless of spike outcome); row 8 (`runtime/operator.md`) → Confident (reconcile committed in the will-update direction; hydrate verifies). No Tentative or Unresolved rows remain.
