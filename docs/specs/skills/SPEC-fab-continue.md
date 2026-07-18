# fab-continue

## Contents

- [Summary](#summary)
- [Flow](#flow)

## Summary

Advances through the 6-stage pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. Supports reset to a given stage (legacy `tasks`/`spec` targets error with a pointer to the `apply` and `intake` reset routes). Handles all six stages: intake (the only planning stage), apply (co-generates `plan.md` `## Requirements` + `## Tasks` + `## Acceptance` at entry then runs tasks), review (sub-agent), hydrate, ship (delegates to `/git-pr` behavior), and review-pr (delegates to `/git-pr-review` behavior).

**Single post-intake execution mode — one-stage sequencer** (260613-fgxx): intake is the sole context boundary. Intake is the only stage `/fab-continue` runs in the main session; **every post-intake stage (apply / review / hydrate) is always dispatched as a sub-agent**, the same dispatch the orchestrators (`_pipeline.md`) perform. There is no foreground execution path for apply/review/hydrate. In this mode `/fab-continue` is a **one-stage sequencer**: it runs `fab resolve-agent <stage> --alias` immediately before the dispatch and applies the resolved tier (the two-seam model-param + effort-via-prompt mechanics are described in the **Per-stage model** paragraph below), includes the universal **block-contract carve-out** prompt contract, reads the returned status/findings, and owns the `finish`/`fail`/`reset` transition itself — identical sequencer/block split whether the caller is manual `/fab-continue` or an orchestrator. As of **260702-aetz (3d)** the Step 1 dispatch contract **branches on the resolved `dispatch=` line** (surfaced alongside `model=/effort=`): absent ⇒ native Agent-tool dispatch (the two seams); present ⇒ the CLI adapter (`fab dispatch`, per `_preamble.md` § CLI-Adapter Dispatch). The block-contract carve-out the prompt carries is refined: it prohibits `fab status` *transition* commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) but REQUIRES a terminal `fab status refresh` (a pull-based recompute, not a transition; the sequencer still owns every transition). As of **260704-pag2** the review stage dispatches a **single** review sub-agent whose prompt carries both checklists (plan-conformance steps + holistic-diff focus areas), so there is no nested reviewer dispatch, no parallel dispatch, and no review-stage nesting degradation — native and CLI dispatch are structurally identical for review (one worker runs the whole review inline). The three former dual-mode "When invoked as a subagent: do NOT run `fab status`" conditionals (Apply/Review/Hydrate Behavior) are removed; the instruction is now the universal block contract, not a per-caller override, and is NOT re-encoded as any "skip §Verdict when subagent" flag. Ship and review-pr are excluded from this rewrite — they self-manage their transitions (the dispatch exception).

**Failure recovery + idempotent reset** (260612-w7dp): a `review-pr`/`failed` dispatch row — keyed off `progress.review-pr == failed`, the same progress-map guard mechanism as the review row — re-executes `/git-pr-review` behavior (its Step 0 `start` accepts `failed → active` for review-pr; never `reset`, whose From-set `{done, ready, skipped}` excludes `failed`), so a failed PR review no longer falls through to "Change is complete." The ship and review-pr rows (incl. the failed row) pass the resolved change **explicitly** to `/git-pr`/`/git-pr-review` (`{name}` as the `<change>` argument — the explicit-arg contract); the ship and review-pr **`active`** rows key on `active` only — `ready` is not in either stage's AllowedStates — while the review-pr failed row keys on the progress map's `failed`. The Reset Flow handles all non-resettable target states (reset From-set `{done, ready, skipped}`): already-`active` → skip the call and proceed (re-running a reset is a state-wise no-op — idempotency, a fab-kit design principle); `failed` → route via the matching failed dispatch row (`start` owns failed→active, review/review-pr only); `pending` → error with advance guidance. All recovery pointers are executable: the unexecutable `/fab-clarify intake` form is replaced by `/fab-continue intake` then argless `/fab-clarify` (argless is correct in fab-continue's own messages — the change reference of the current invocation is implied, active or `[change-name]` override, and an Error Handling note tells override users to re-run with the same `<change-name>`; cross-context sites like `_pipeline.md`'s stop guidance instead name the change in every command), with an explicit delete-`plan.md` note where plan regeneration is the intent; the `intake.md`-missing error points at `/fab-continue intake` instead of looping through plain `/fab-continue`. The **sequencer** (Normal Flow Step 1's review dispatch) reads `change_type` from `.status.yaml` and carries it in the block dispatch prompt per `_review.md`'s context contract — the dispatched review block does not read it itself (the parsimony/deletion-candidate skip condition keys on the prompt value).

**Per-stage model** (260613-l3ja, 260613-fgxx, 260613-m3d4, 260613-yky7): the one-stage sequencer resolves `fab resolve-agent <stage> --alias` immediately before dispatching each post-intake stage's block (`--alias` since 260613-yky7 — emits the Agent-tool-valid short alias on the `model=` line), **surfaces** the resolved `model=/effort=` (visibility — a skip is detectable, not silent; m3d4), and applies both halves via two seams: model on the Agent `model` param (empty ⇒ omit/inherit) and effort as an imperative instruction in the sub-agent's prompt (no Agent effort param; omitted when empty; m3d4). Since every post-intake stage now dispatches, per-stage selection applies uniformly across apply/review/hydrate regardless of caller — there is no foreground post-intake path left to be the advisory-only exception (this closes Gap 1a of the model-tier finding; Gap 1b visibility + Gap 2's effort half are closed by m3d4 via the surface + prompt-injection seams; the lone residual is a first-class per-sub-agent effort param on the Agent tool — a harness ask, not built). Review is unexceptional (260704-pag2): the sequencer resolves `fab resolve-agent review --alias` once for the **single** review sub-agent, exactly like every other stage — there is no second nested resolution for reviewers + merge (the Claude Code adapter is the Agent tool `model` param, effort rides the prompt; resolution is provider-neutral — see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution). Intake is pre-boundary and is not tiered by `/fab-continue`.

**Helpers**: Declares `helpers: [_srad]` in frontmatter; `_generation` and `_review` are loaded **stage-conditionally** at point of use (apply entry / intake regeneration → `_generation`; Review Behavior entry → `_review`) per `_preamble.md` § Skill Helper Declaration stage-conditional loading. Hydrate/ship/review-pr invocations and apply-resumes load neither.

**FKF hydrate prose** (260615-8fr5, 260616-2fm8): Hydrate Behavior authors memory files to the FKF contract — the shipped normative extract at `$(fab kit-path)/reference/fkf.md` (260616-frlo; mirror of the dev-repo design doc `docs/specs/fkf.md`). New memory files are created from the canonical memory-file template shipped at `$(fab kit-path)/templates/memory.md` (260616-2fm8) — read on demand the same way `_generation.md`/`_intake.md` read `$(fab kit-path)/templates/intake.md` — the single source of truth for the FKF frontmatter pair — `type: memory` (constant, §3.1) plus a curated `description:` one-liner (§3.2) — and the body skeleton; not `description:` alone. As of **260715-xu0k** Hydrate Step 4 states the `description:` **500-character one-liner cap** (§3.2 — a routing signal, not a summary of record; detail belongs in the body, `fab memory-index` warns over the cap) and carries the **never-hand-merge pointer** on the `fab memory-index` regen bullet (a generated `docs/memory/**/index.md`/`log.md` conflict is resolved by fixing topic files + re-running, taking output wholesale — FKF §5, never hand-merged). Hydrate no longer writes a per-file `## Changelog` section (§3.3): it records what changed once via `fab status set-summary {change} "<one-line what-changed>"` (the C-lite `summary:` source line, §6.3, authored once at hydrate), which `fab memory-index` joins with git history to generate the per-folder `log.md` (§6). Memory↔memory cross-links use the bundle-relative `/...` form (§7); links out of the bundle stay repo-relative/absolute-URL. The "update existing" section list drops `Changelog` (now Requirements/Design Decisions only); the merge-without-duplication contract is unchanged. When hydrate edits an existing/legacy memory file missing `type: memory`, it stamps the constant in so the touched file becomes FKF-conforming (§2/§3.1 require `type: memory` on every memory file, stamped by every memory writer — not just on creation). This is FKF migration Change 3/4 — it stops *new* changelog writes; the strip of the 20 existing `## Changelog` sections is Change 4/4.

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts — the ~5 near-identical "dispatched block / sequencer owns transitions" blockquotes collapsed to one canonical statement (Normal Flow Step 1's dispatch contract) plus per-section references, the per-stage model paragraphs reduced to references, the Step 3 procedure table folded into prose, and Hydrate Step 4's long paragraph reformatted as a bullet list (same content); a `## Contents` TOC added to both the skill and this SPEC. No behavioral change (Flow / Tools / Sub-agents unchanged).

**Writer-contract additions** (260718-wrct): Hydrate Behavior carries the residual leak-class rules on top of the present-truth core. Step 4's merge bullets gain (1) a **heading change-id ban** (a heading names its topic, never a change — `### Dispatch States (xu0k)` is prohibited; change-ids stay citation-only in body text), (2) a **post-body-edit `description:` re-check** (after any body edit, re-read the `description:` — one line, ≤500 chars, change-id-free — so it doesn't drift stale), and (3) a **rationale → Design Decisions** bullet (any why / rejected alternative goes into a four-field `## Design Decisions` entry — Decision / Why / Rejected / *Introduced by* — never inline narration; the changelog-bullet shape `- **{change-id} — retired X**` is banned inside `## Design Decisions`, that being `log.md`'s job, §6). A new **post-hydrate self-check** numbered step (now step 5, before the regen step 6) re-reads every file touched *this run* and strips any transition phrasing / change-keyed delta paragraph / change-id heading reflexively introduced, confirming descriptions still route — scoped to this run's own writes, not a corpus sweep (`/docs-distill-memory`'s job). The Hydrate step insertion renumbers Return (now step 7) and Pattern capture (now step 8); Pattern capture is reworded to name the four-field DD entry shape. The FKF §3.3 body-style rules these cite are amended identically in both `docs/specs/fkf.md` and `src/kit/reference/fkf.md`. No behavioral change to the sequencer/block split, dispatch, or transitions — only the writer contract the dispatched hydrate block applies is tightened.

**Present-truth hydrate merge** (260717-3plm): Hydrate Behavior now authors memory bodies as **current truth in present tense** (FKF §3.3, amended by this change). The former "Merge without duplication" bullet is rewritten as **merge as current truth**: the dedup unit is the **topic/section**, not the change — hydrate locates the section documenting a topic and **rewrites it to current truth** rather than appending (or updating-in-place) a change-keyed delta entry. Superseded statements are removed, not narrated (no "renamed X→Y in {id}", "this inverts {id}'s claim", "was `old.value`" in the body); the change's dated *what* is already captured once via `fab status set-summary` → `log.md`, and body provenance is citation-only (trailing `(change-id)` / `*Introduced by*`). The rewrite preserves the idempotency the by-change-name dedup gave (a re-run rewrites the same section to the same truth). Pattern-capture (Hydrate step 6) drops the "with the change name for traceability" phrasing for citation-form provenance (`*Introduced by*` / a trailing citation). Hydrate's FKF-frontmatter bullet also records the §3.2 **no-change-ids-in-`description:`** ban on both of its description clauses (create-path cap clause + update-existing clause) — the `description:` is a routing signal, not a provenance record, so change-id citations belong in the body. Flow-diagram wording refreshed to match (the HYDRATE box's merge note now reads "merge as current truth"); no structural change — the merge/idempotency semantics are unchanged in shape, only the keying (topic vs change), the change-id-free description, and the ban on transition narration are new.

## Flow

```
User invokes /fab-continue [change-name] [stage]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ [if reset arg] Reset Flow
│  └─ Bash: fab status reset <change> <stage> fab-continue
│     (non-resettable target states handled first, 260612-w7dp —
│      reset From = {done, ready, skipped}: already-active → skip
│      the call, proceed (re-run is a no-op); failed → route via the
│      matching failed dispatch row (start owns failed→active);
│      pending → error with advance guidance)
│     └─ (cascades downstream to pending)
│
├─ Dispatch on current stage + state
│  (review-failed dispatch — 260611-szxd f019: progress.review == failed
│   [exhausted ff/fff rework or interrupted fail→reset] →
│   fab status reset <change> apply fab-continue, then present the
│   rework menu directly and stop for the user's choice — do NOT
│   re-run review; orchestrators re-running /fab-ff//fab-fff recover
│   via fab status start <change> review per _pipeline.md Resumability
│   instead — that autonomous path is theirs, not this skill's)
│  (review-pr-failed dispatch — 260612-w7dp: progress.review-pr ==
│   failed → re-execute /git-pr-review behavior; its Step 0 start
│   recovers failed→active — never reset, and never falls through
│   to "Change is complete.")
│
│  ┌─────────────────────────────────────────────────┐
│  │ INTAKE STAGE (the only planning stage)          │
│  │                                                 │
│  │  Read: templates, intake, memory files          │
│  │  (agent generates intake artifact via SRAD)     │
│  │  Write: intake.md                               │
│  │  (no scoring here — intake score is written by  │
│  │   /fab-new and /fab-clarify)                    │
│  │  Bash: fab status advance <stage>               │
│  │  (intake ready → finish intake — auto-activates │
│  │   apply; no start call)                         │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ APPLY STAGE                                     │
│  │                                                 │
│  │  Entry sub-step (skip if plan.md exists):       │
│  │    Read: intake.md, _generation.md              │
│  │    Write: plan.md                               │
│  │      (## Requirements + ## Tasks +              │
│  │       ## Acceptance, R#/T###/A-### IDs)         │
│  │      (under-spec → inline SRAD assumption)      │
│  │                                                 │
│  │  Main sub-step (Task Execution):                │
│  │    Read: plan.md ## Tasks, source files         │
│  │    (pattern extraction from neighboring files)  │
│  │    For each unchecked task:                     │
│  │      Read: relevant source files                │
│  │      Edit/Write: implementation files           │
│  │      Bash: run tests                            │
│  │      Edit: plan.md ## Tasks (mark [x])          │
│  │    Bash: fab status finish <change> apply       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW STAGE                                    │
│  │  (the dispatched review block IS the single     │
│  │   review agent: it reads _review.md at entry    │
│  │   and runs the whole review inline — no nested  │
│  │   sub-agent. Sequencer reads change_type from   │
│  │   .status.yaml and carries it in the prompt.)   │
│  │                                                 │
│  │  Framing (in _review.md, which the worker       │
│  │   reads): conformance to plan.md is necessary   │
│  │   but not sufficient; also judge the diff on    │
│  │   its own merits against the repo               │
│  │  Read: standard subagent context, git diff +    │
│  │        changed file list, plan.md               │
│  │        (## Requirements + ## Tasks +            │
│  │        ## Acceptance), source + memory;         │
│  │        full repo access                         │
│  │  Plan-conformance steps (full mode) +           │
│  │   holistic-diff focus areas +                   │
│  │   Codex→Claude cascade (graceful no-op)         │
│  │  Bash: run tests                                │
│  │  Edit: plan.md ## Acceptance (mark [x])         │
│  │  Returns: ONE unified must-fix/                 │
│  │   should-fix/nice-to-have set                   │
│  │                                                 │
│  │  Verdict from the single findings set           │
│  │                                                 │
│  │  Pass:                                          │
│  │    Bash: fab status finish <change> review      │
│  │    Bash: fab status set-acceptance              │
│  │          <change> acceptance_completed N        │
│  │  Fail:                                          │
│  │    Bash: fab status fail <change> review        │
│  │    Bash: fab status reset <change> apply        │
│  │    (present rework options to user)             │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ HYDRATE STAGE                                   │
│  │                                                 │
│  │  Read: docs/memory/ files, intake.md,           │
│  │    $(fab kit-path)/templates/memory.md (shape)  │
│  │  Write/Edit: docs/memory/{domain}/{file}.md     │
│  │    (from template: FKF frontmatter type: memory │
│  │     + curated description:; NO ## Changelog —   │
│  │     bundle-relative /... memory↔memory links;   │
│  │     merge as current truth — rewrite the topic  │
│  │     section, not a change-keyed delta; headings │
│  │     carry no change-ids; why → 4-field Design   │
│  │     Decisions, not narration)                   │
│  │  Self-check (files touched this run): strip any │
│  │    transition phrasing / change-id heading just │
│  │    introduced; descriptions still route         │
│  │  Bash: fab status set-summary <change> "<one-   │
│  │     line what-changed>"  (C-lite summary:       │
│  │     source; fab memory-index joins it with git  │
│  │     history into the per-folder log.md)         │
│  │  Bash: fab memory-index --check (refuse-before- │
│  │   regen guard, defense-in-depth: refuse on exit │
│  │   2; no-op on born-compatible trees) →          │
│  │  Bash: fab memory-index — regenerates the root  │
│  │  (domains-only), domain, and sub-domain indexes │
│  │  Bash: fab status finish <change> hydrate       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ SHIP STAGE                                      │
│  │  (delegates to /git-pr behavior, passing the    │
│  │   resolved change as the explicit <change>      │
│  │   argument — 260612-w7dp)                       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW-PR STAGE                                 │
│  │  (delegates to /git-pr-review behavior, passing │
│  │   the resolved change as the explicit <change>  │
│  │   argument — 260612-w7dp; it                    │
│  │   routes all terminal paths through its Step 6  │
│  │   and runs its own transitions; finish or fail  │
│  │   only if the stage is still active after it    │
│  │   returns; timeout outcome: stage deliberately  │
│  │   left active — report and stop, no re-finish)  │
│  └─────────────────────────────────────────────────┘
│
└─ Output: summary + Next: line
```

> **Dispatch annotation** (260613-fgxx; 260702-aetz): in the APPLY / REVIEW / HYDRATE boxes above, the stage *work* runs inside a dispatched sub-agent (resolved via `fab resolve-agent <stage> --alias`, then **branched on `dispatch=`** — native Agent-tool dispatch when absent, the CLI adapter `fab dispatch` when present, 260702-aetz — dispatched by the one-stage sequencer). The `Bash: fab status finish/fail/reset` lines shown in those boxes are run by the **sequencer** after the block returns — the dispatched block runs **no `fab status` transition command** (its prompt carries the carve-out: no transition commands, but a terminal `fab status refresh` — a pull-based recompute — is required). The boxes show the end-to-end stage picture, not block-internal actions. INTAKE is the only box that runs in the main session.

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, templates, artifacts, source files, memory |
| Write | Plan (`plan.md`), memory files |
| Edit | Plan (mark `## Tasks` and `## Acceptance` items [x]), memory files |
| Bash | All `fab status` transitions, `fab preflight`, `fab status set-summary` (hydrate — the C-lite `summary:` source for the generated `log.md`), `fab memory-index` (+ a `fab memory-index --check` refuse-before-regen guard at the hydrate stage — defense-in-depth, refuses on exit 2, a no-op on born-compatible trees), test execution — no `fab score` (no scoring at any stage `/fab-continue` runs; intake scoring belongs to `/fab-new`/`/fab-clarify`) |
| Agent | Single review sub-agent (general-purpose) — the sequencer dispatches one worker that reads `_review.md` and runs the whole review inline |

### Sub-agents

| Agent | Stage | Purpose |
|-------|-------|---------|
| Single review sub-agent (`_review.md`) | review | Runs the whole review inline: `plan.md` validation (`## Requirements` + `## Tasks` + `## Acceptance`) with test execution (full mode) + holistic full-repo diff review via Codex→Claude cascade; returns one unified findings set |

> Review Behavior reads `.claude/skills/_review/SKILL.md` (if not already loaded) and executes its **Shared Review Dispatch** end-to-end (Review Mode → Preconditions → Review Agent Dispatch → Findings & Verdict) — `_review.md` is the single source of truth for the single review sub-agent's dispatch and findings shape. `fab-continue.md` retains the Verdict section (pass/fail state transitions, rework options).

> **Universal block contract** (f006, revised 260613-fgxx; carve-out refined 260702-aetz): the Apply/Review/Hydrate behavior sections are **always** dispatched as sub-agents (by the manual `/fab-continue` one-stage sequencer in Path A and by `/fab-ff`/`/fab-fff` orchestrators in Paths B/C/D — identical dispatch; native Agent-tool or the CLI adapter `fab dispatch` per the `dispatch=` branch). The dispatched block runs **no `fab status` transition command** (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) and takes no §Verdict-style decision itself; it returns results/findings only — but its prompt DOES end with a terminal `fab status refresh` (a pull-based recompute, not a transition, per `_preamble.md` § Dispatch-Prompt Obligations). The owning sequencer (the manual `/fab-continue` invocation, or `_pipeline.md`) runs all `finish`/`fail`/`reset` transitions. This is no longer a per-caller conditional baked into the block — the former three "When invoked as a subagent: do NOT run `fab status`" blockquotes are removed and the instruction is the universal block contract, carried in the dispatch prompt. It is NOT re-encoded as a "skip §Verdict when subagent" flag — the Review block always returns findings; **who** acts on a fail verdict (interactive § Verdict menu in Path A vs. autonomous Auto-Rework Loop in B/C/D) is the orchestrator's concern. The ship dispatch row likewise only runs `finish <change> ship` if the stage is still `active` after `/git-pr` returns (git-pr finishes ship internally), and the review-pr row's Pass and Fail branches both carry the same only-if-still-active guard (git-pr-review's Step 6 runs its own finish/fail).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Plan generation | `fab status refresh` recomputes `plan.task_count`, `plan.acceptance_count`, sets `plan.generated=true` | Self-healed at the next advance/finish/preflight after plan.md write (no scoring at apply — intake is authoritative) |
| Review pass | `fab status set-acceptance <change> acceptance_completed N` | After review validation |
