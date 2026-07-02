# Intake: Pipeline Choreography Prose Fixes (Group B)

**Change**: 260615-qg64-pipeline-choreography-prose-fixes
**Created**: 2026-06-15

## Origin

> Backlog `[qg64]`: "Pipeline choreography prose fixes (skills only, no Go) — GROUP B of 2
> (recurring-lessons fixes 4,5). GOAL: fix two pipeline-behavior defects that live in skill prose,
> not the binary. ACTIONS: (a) stage_metrics cycle-count [PR-meta under-reports review rework as
> '1 cycle' when 3+ cycles ran; CONFIRMED the bug is orchestrator/skill choreography, NOT
> internal/status — a 3-cycle fail->reset->re-finish regression test PASSES against the unchanged
> state machine. Fix the fail->reset->re-finish sequence in the orchestrator skills (_pipeline.md /
> fab-ff / fab-fff) so the iteration counter isn't reset to 1; do NOT touch internal/status].
> (b) review-pr poll discipline [bake permanently into git-pr-review.md's INITIAL subagent dispatch
> prompt: 'do NOT yield while the Copilot poll is pending — complete synchronously' (the subagent
> stalled/died mid-poll 4x). Encode the correct query: comment-author login is 'Copilot' but the
> review-REQUEST login is 'copilot-pull-request-reviewer'; GraphQL reviewRequests omits bot
> reviewers so confirm requests via REST requested_reviewers; orchestrator owns the wait (request
> review, background-poll 30s x20, dispatch only once review exists). Copilot lands ~4.5-6.5 min].
> CONSTRAINTS: prose-only, no Go, no behavior-schema change; SPEC mirror per touched skill
> (docs/specs/skills/SPEC-*.md) — Copilot enforces this strictly; src/kit is canonical.
> SEQUENCING: develop in parallel with GROUP A [jznd] (disjoint surfaces: skill .md vs Go) but
> MERGE AFTER A. SRAD: likely passes gate (well-scoped prose) — /fab-fff candidate."

**Interaction mode**: one-shot from backlog, with one interactive SRAD clarification on defect (b)'s
poll architecture (see Assumption #7). These two defects are carry-forwards distilled from six
shipped+archived batch efforts (the "fab recurring lessons" set, items 4 and 5). Both have been
pre-investigated: defect (a)'s root-cause locus is confirmed as orchestrator/skill choreography (not
the Go state machine), and defect (b)'s correct GitHub query semantics are spelled out in the
backlog.

This is **Group B of 2**. Group A (`[jznd]`) covers the disjoint Go surface; the two groups are
developed in parallel but Group B **MUST merge after** Group A (operator stacked-merge rule). The
surfaces are disjoint — Group B touches only skill `.md` + SPEC mirrors; it touches **no Go**.

## Why

Two pipeline-behavior defects degrade observability and reliability of the automated bracket. Both
live in **skill prose** (the choreography that orchestrates the Go binary), not in the binary
itself — so both are fixable with prose-only edits.

### Defect (a) — `stage_metrics` review-cycle under-reporting

**Problem**: When the auto-rework loop runs 3+ cycles, `fab pr-meta`'s Review column reports
"✓ 1 cycle" (or "✗ 1 cycle") instead of the true cycle count. The cycle count is read from
`stage_metrics.review.iterations` (`src/go/fab/internal/prmeta/prmeta.go:307` → `reviewCell`,
lines 139–162). An under-count makes a heavily-reworked change look clean in PR metadata,
hiding rework churn from anyone reading the PR.

**Consequence if unfixed**: PR metadata silently misrepresents review effort. Reviewers and the
operator lose a real signal (how many rework cycles a change burned), and the distilled
"recurring lesson" stays unresolved — it will keep biting future batch efforts.

**Why this approach (choreography fix, not Go fix)**: The root cause is **confirmed** to be the
orchestrator skills' `fail → reset → re-finish` choreography, NOT the Go state machine:

- A 3-cycle `fail → reset → re-finish` regression test **PASSES** against the unchanged state
  machine — i.e., when driven through the *correct* event sequence, `iterations` increments
  correctly.
- Code reading confirms the mechanism: `internal/status/status.go:627` does `sm.Iterations++`
  **only** on a transition to `active`; the `reset → pending` cascade (`status.go:646–660`)
  deliberately **preserves** the counter (clears only the timing fields, never zeroes
  `Iterations`). So the Go layer is correct and MUST NOT be touched.

Therefore the bug is that the orchestrator's per-cycle sequence is failing to drive the
counter-incrementing `active` transition once per cycle. The fix is to make the per-cycle
choreography in `_pipeline.md` (and any divergent wording in `fab-ff.md` / `fab-fff.md`) explicit
and correct so the iteration counter advances once per genuine rework cycle and is never reset to 1.

### Defect (b) — review-pr Copilot-poll discipline

**Problem**: The `/git-pr-review` subagent stalled or died mid-poll **4 times** across prior
efforts while waiting for a Copilot review to land. Copilot reviews land ~4.5–6.5 min after the
request — well within the 10-minute (30s × 20) poll window — but the subagent yielded/died before
the review appeared, leaving `review-pr` `active` and the cycle incomplete. Separately, the poll
query semantics are subtle and easy to get wrong (see below), which can cause a poll to never see
a review that has in fact landed.

**Consequence if unfixed**: The review-pr stage repeatedly fails to complete autonomously, forcing
manual re-runs and undercutting the "everything after intake runs unattended" promise.

**Why this approach**: Bake the poll discipline **permanently** into `/git-pr-review`'s behavior
and into the orchestrator's dispatch prompt, plus encode the exact GitHub query semantics, so the
subagent completes the poll **synchronously** without yielding. (The poll stays inside
`/git-pr-review` — the subagent owns request + poll + triage in one synchronous run — per the
interactive SRAD decision; see Assumption #7.)

## What Changes

> **Scope guardrails (constitution-derived, repeated for the apply agent)**: prose-only — edit
> **only** `src/kit/skills/*.md` and their `docs/specs/skills/SPEC-*.md` mirrors. **No Go**
> (`src/go/**` is off-limits). **No behavior-schema change** (the `.status.yaml` schema, the state
> machine, and the `fab` CLI signatures are unchanged). `src/kit/` is canonical; never edit
> `.claude/skills/` (gitignored deployed copies). Each touched skill `.md` MUST update its
> SPEC mirror — **Copilot enforces this strictly** (constitution: "Changes to skill files
> (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file").

### Change Area 1 — Fix the per-cycle iteration choreography (defect a)

**File**: `src/kit/skills/_pipeline.md` (primary). Verify `fab-ff.md` / `fab-fff.md` carry no
divergent per-cycle wording (they are thin wrappers over the `_pipeline` bracket — see § Impact);
edit them only if they restate the choreography.

The Auto-Rework Loop's **per-cycle choreography** (`_pipeline.md` lines 84–90) currently reads:

1. **Status pair**: `fab status fail <change> review` then `fab status reset <change> apply {driver}`.
2. Triage + rework action.
3. **Re-dispatch apply**: dispatch `/fab-continue` Apply Behavior; on success
   `fab status finish <change> apply {driver}` (auto-activates review).
4. **Fresh re-review**: dispatch a fresh `/fab-continue` Review Behavior subagent.
5. **Verdict**: pass → finish; fail → next cycle (back to item 1).

**The defect**: `iterations` increments only when `review` transitions to `active`. In the current
choreography, review is driven to `active` via the `finish apply` auto-activation at item 3. The
under-count arises because the per-cycle sequence does not reliably produce **one** counted
`pending → active` review re-entry per rework cycle that maps to what `pr-meta` should report — the
counter ends up reading as 1 even after 3+ cycles ran. The apply agent MUST trace the exact event
sequence against `applyMetricsSideEffect` (`status.go:617–661`) and make the prose drive **exactly
one counted review re-activation per cycle**, so a run with N rework cycles leaves
`stage_metrics.review.iterations == N` (or N+1 if the initial review entry is also counted — the
agent MUST determine and document the intended baseline so prose and `pr-meta` agree).

**Concrete acceptance signal**: after the fix, a 3-cycle rework run (initial review fail → 3
rework cycles, final pass) MUST leave `stage_metrics.review.iterations` reflecting the true cycle
count, and `fab pr-meta` MUST render "✓ 3 cycles" (or the documented baseline-consistent count),
NOT "✓ 1 cycle". This MUST hold **without any change to `internal/status` or `internal/prmeta`** —
the Go regression test already passes for the correct sequence, so the fix is purely making the
skill prose drive that correct sequence.

> **Pin against the Go contract** (do not re-derive): `status.go:627` `Iterations++` fires **only**
> on `state == "active"`; `status.go:646–660` `reset`/`skip` → `pending`/`skipped` **preserves**
> `Iterations` (clears timing fields only). The choreography must therefore guarantee one `active`
> review transition per cycle; it must never rely on `reset` to bump or zero the counter.

### Change Area 2 — Bake in review-pr poll discipline (defect b)

**Files**: `src/kit/skills/git-pr-review.md` (primary — Step 2 Phase 2, the poll), and
`src/kit/skills/fab-fff.md` (Step 5 — the dispatch prompt that hands review-pr to the subagent).

**2a — Don't-yield-mid-poll directive (git-pr-review.md + fab-fff dispatch prompt)**. The poll
stays inside `/git-pr-review` (the subagent owns request + poll + triage synchronously). Add a
permanent, explicit directive that the Copilot poll MUST run **synchronously to completion** — the
subagent MUST NOT yield, return, or hand back control while the poll is pending (the 30s × 20 /
10-minute window). Mirror this into `fab-fff.md`'s Step 5 dispatch prompt: when fab-fff dispatches
`/git-pr-review` as a subagent, the dispatch prompt MUST instruct the subagent to **complete the
Copilot poll synchronously and not yield mid-poll**. Rationale to encode inline: the subagent
stalled/died mid-poll 4× in prior efforts; Copilot lands ~4.5–6.5 min, comfortably inside the
window, so the correct behavior is patience-to-completion, not early return.

**2b — Correct query semantics (git-pr-review.md Step 2 Phase 2 + Step 3)**. Encode these GitHub
specifics so the poll reliably detects a landed review:

- **Two distinct logins**: the review-**request** assignee login is `copilot-pull-request-reviewer`
  (used by `gh pr edit --add-reviewer copilot-pull-request-reviewer`), but the **comment-author**
  login on a posted Copilot review is `Copilot`. The skill MUST NOT conflate them. (Today's Step 2
  Phase 2 polls `reviews | map(select(.author.login == "copilot-pull-request-reviewer"))` — the
  apply agent MUST verify which login the *review object* actually carries vs. which the
  *requested-reviewer* carries, and correct the poll predicate accordingly, documenting the
  distinction inline.)
- **GraphQL omits bot reviewers**: GraphQL `reviewRequests` does **not** surface bot/app reviewers
  like Copilot. To confirm the **request** succeeded, query REST `requested_reviewers` (e.g.,
  `gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers`), not a GraphQL
  `reviewRequests` field.
- **Poll cadence unchanged**: 30s × 20 (10-minute window) — this matches the existing Step 2
  Phase 2 cadence and Copilot's ~4.5–6.5 min landing time. No schema or timing change; this is a
  prose-clarification of *what to query and how to wait*.

### Change Area 3 — SPEC mirrors (mandatory, all touched skills)

For **every** skill `.md` touched above, update its `docs/specs/skills/SPEC-*.md` mirror in the
same change:

- `_pipeline.md` → `docs/specs/skills/SPEC-_pipeline.md`
- `git-pr-review.md` → `docs/specs/skills/SPEC-git-pr-review.md`
- `fab-fff.md` → `docs/specs/skills/SPEC-fab-fff.md` (only if its Step 5 dispatch prose changes)
- `fab-ff.md` → `docs/specs/skills/SPEC-fab-ff.md` (only if it carries divergent per-cycle wording
  that gets edited)

Copilot enforces the skill↔SPEC mirror strictly on the PR; a missing mirror is a guaranteed review
finding.

## Affected Memory

- `pipeline/execution-skills`: (modify) Update the `_pipeline.md` rework-choreography note to
  reflect the corrected per-cycle iteration-counting choreography, and the `/git-pr-review` note to
  reflect the baked-in synchronous-poll discipline + corrected Copilot query semantics
  (`copilot-pull-request-reviewer` request login vs. `Copilot` comment-author login; REST
  `requested_reviewers` to confirm requests since GraphQL omits bot reviewers).
- `pipeline/schemas`: (modify) Reinforce that the iterations-preserving `reset` cascade
  (`status.go` k4ge) is correct **as-is** and that cycle-count accuracy is an
  orchestrator-choreography property, not a state-machine one — i.e., document that the
  fix lives in skill prose, not `internal/status`. (Low-touch — only if hydrate determines the
  schema note benefits from the clarification.)

## Impact

**Code areas (prose only)**:
- `src/kit/skills/_pipeline.md` — Auto-Rework Loop per-cycle choreography (lines ~84–90).
- `src/kit/skills/git-pr-review.md` — Step 2 Phase 2 (Copilot request + poll) and Step 3 (comment
  fetch login predicate).
- `src/kit/skills/fab-fff.md` — Step 5 review-pr dispatch prompt (don't-yield directive).
- `src/kit/skills/fab-ff.md` — only if it restates per-cycle choreography (it is a thin
  `_pipeline` wrapper; expected: no change beyond verification).
- `docs/specs/skills/SPEC-*.md` mirrors for each of the above that changes.

**Explicitly out of scope**:
- `src/go/**` — NO Go changes. `internal/status` and `internal/prmeta` are confirmed correct.
- `.status.yaml` schema, state-machine transitions, `fab` CLI signatures — unchanged.
- Group A (`[jznd]`) surface (Go) — disjoint; coordinated via merge ordering (B after A).

**Dependencies / sequencing**: Develop in parallel with Group A; **merge Group B after Group A**
(disjoint surfaces, but the operator stacked-merge rule applies). No runtime dependency between the
two during development.

**Verification note**: Because no Go changes, the existing Go test suite (incl. the 3-cycle
`fail → reset → re-finish` regression test and the `prmeta` cycle-rendering tests) MUST continue to
pass unchanged — they are the oracle that the choreography fix targets the right contract.

## Open Questions

- (Resolved via SRAD #7) Poll ownership: keep the Copilot poll inside `/git-pr-review` (subagent
  owns request + poll + triage synchronously) vs. move the wait up to the orchestrator. Resolved:
  keep it in `/git-pr-review`.
- For defect (a), the exact baseline convention — does `iterations` count the **initial** review
  entry plus each rework re-entry (so N rework cycles → N+1), or only rework re-entries (N cycles →
  N)? The apply agent MUST determine the intended convention from the Go regression test's
  expectations and `pr-meta`'s rendering, then make the prose consistent with it. (This is a
  decide-and-record at apply, not a blocking intake question — the Go test is the oracle.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Defect (a)'s root-cause locus is orchestrator/skill choreography, NOT `internal/status`; the fix is prose-only and MUST NOT touch Go. | Confirmed by the backlog (3-cycle regression test passes against unchanged state machine) AND by code reading: `status.go:627` increments on `active` only; `status.go:646–660` `reset` preserves the counter. The constitution forbids unnecessary Go change here; the Go contract is the oracle. | S:95 R:80 A:95 D:90 |
| 2 | Certain | Every touched skill `.md` MUST update its `docs/specs/skills/SPEC-*.md` mirror in the same change. | Constitution: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." Backlog reinforces "Copilot enforces this strictly." Deterministic rule, no judgment. | S:100 R:70 A:100 D:95 |
| 3 | Certain | Edits land in `src/kit/skills/` (canonical), never in `.claude/skills/` (gitignored deployed copies). | Constitution Principle V + the explicit `src/kit is canonical` constraint; `.claude/skills/` is produced by `fab sync`. | S:100 R:90 A:100 D:100 |
| 4 | Confident | Defect (b)'s poll cadence stays 30s × 20 (10-min window); only the query semantics and the don't-yield discipline are clarified — no timing/schema change. | Existing `git-pr-review.md` Step 2 Phase 2 already uses 30s × 20; Copilot lands ~4.5–6.5 min (well inside). Backlog frames this as discipline + correct query, not a timing change. One obvious interpretation. | S:80 R:75 A:85 D:80 |
| 5 | Confident | The corrected Copilot query semantics are: request login `copilot-pull-request-reviewer`, comment-author login `Copilot`, confirm requests via REST `requested_reviewers` (GraphQL omits bot reviewers). | Spelled out verbatim in the backlog; matches GitHub's documented Copilot-reviewer behavior. The apply agent verifies the exact poll predicate against the live API but the semantics are dictated. | S:85 R:75 A:80 D:85 |
| 6 | Certain | `fab-ff.md`/`fab-fff.md` are thin `_pipeline` wrappers; the per-cycle choreography fix lands primarily in `_pipeline.md`, with `fab-ff`/`fab-fff` edited only if they restate the choreography. | Verified by reading both files during intake: neither restates per-cycle wording; both reference the `_pipeline` bracket. Corroborated by memory (`planning-skills.md`: "fab-ff/fab-fff are thin wrappers over the shared `_pipeline` bracket"). Determined by direct evidence, not inference. | S:90 R:80 A:95 D:85 |
| 7 | Confident | Poll ownership stays inside `/git-pr-review` (subagent owns request + poll + triage synchronously); fix = harden git-pr-review.md's poll + add a don't-yield directive to fab-fff's dispatch prompt — NOT move the wait to the orchestrator. | Asked — user selected "Keep poll in git-pr-review" over the orchestrator-owned-wait restructure. Minimal restructure, matches today's flow and "bake into the dispatch prompt" literally. | S:90 R:65 A:70 D:90 |
| 8 | Certain | Group B merges AFTER Group A (`[jznd]`); surfaces are disjoint (skill `.md` + SPEC vs. Go), so parallel development is safe. | Dictated verbatim by the backlog ("MERGE AFTER A") and the fixed operator stacked-merge rule — instruction-determined, not a judgment call. Disjoint surfaces confirmed by scope: Group B touches no `src/go/**`. | S:95 R:75 A:90 D:95 |
| 9 | Confident | The per-cycle prose edit makes `iterations` advance exactly once per rework cycle so the count reflects true cycles; the apply agent traces the choreography against `applyMetricsSideEffect` and aligns the baseline convention (N vs. N+1) to the Go regression test. | The defect direction is confirmed and the target end-state is unambiguous (iterations == true cycle count, validated by the existing Go regression test as a deterministic oracle). Only the literal wording varies — that is true of every prose task and is not genuine design ambiguity; the outcome has one correct interpretation. Reversible (prose). | S:80 R:75 A:80 D:70 |
| 10 | Confident | `pipeline/schemas` memory gets a light clarifying touch (cycle-count accuracy is a choreography property, not a state-machine one), decided at hydrate. | Clearly-bounded, low-risk documentation note with one obvious framing; the existing k4ge schema note already establishes the iterations-preserving reset, so this only adds the choreography-vs-state-machine clarification. Reversible; hydrate confirms final placement. | S:70 R:90 A:80 D:75 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
