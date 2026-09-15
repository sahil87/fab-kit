# Plan: Operator Auto-Merge Choreography

**Change**: 260823-2c92-operator-auto-merge-choreography
**Intake**: `intake.md`

## Requirements

### Operator Skill: Auto-Merge Choreography (Phase C2)

#### R1: Auto-merge choreography section with five MUST rules
`src/kit/skills/fab-operator.md` §6 SHALL gain an `#### Auto-Merge Choreography` subsection adjacent to `#### Ordered Merge`, scoped to the `cherry-pick-ladder` and `merge-auto` merge modes (where PRs target main). It MUST state the `stacked-prs` exclusion with its reasons (inter-merge choreography — retarget-verify, `rebase --onto`, force-push — is operator-sequenced anyway; an armed stacked PR can merge into its dependency's *branch*, destroying the stack silently, or fire on stale-green checks after GitHub's no-re-CI base retarget). It MUST carry all five rules as MUSTs:

1. **Sequential arming** — at most one armed PR per repo-sequence; arm PR_n only after PR_{n-1}'s merge is verified (timeline event, not assumption); never arm a PR whose base is another PR's branch.
2. **Arming-failure shapes** — draft PR → `gh pr ready` first (fab's `/git-pr` creates drafts, so this is every autopilot PR); "already clean" rejection (no required checks) → merge directly; auto-merge disabled on the repo → today's CI-wait choreography.
3. **Stall rule** — an armed PR unmerged after 3 consecutive ticks → check `mergeableState`; `CONFLICTING` → escalate (auto-merge fails silently — there is no event to observe).
4. **Persisted sequence** — starting a merge sequence writes a `kind: coordination` note via `fab operator note add --kind coordination` (sequence, position, armed PR); the note is updated as the sequence advances (`fab operator note update`) and resolved at sequence end (`fab operator note resolve`); a restarted operator re-orients from the note.
5. **Disarm on halt** — any halt/escalation runs `gh pr merge --disable-auto` on the remaining armed PRs (halt-dependents-only assumes unstarted merges stay unstarted, which armed auto-merge violates).

Arming rides `gh pr merge --auto` with the repo's standard merge method (squash unless the user directs otherwise).

- **GIVEN** a completed `cherry-pick-ladder` queue and the user says "merge all"
- **WHEN** the operator starts the merge sequence
- **THEN** it writes the coordination note, arms exactly the first PR in the sequence (readying a draft first), and verifies each merge via a timeline event before arming the next
- **AND** on a halt or escalation it disarms every remaining armed PR

#### R2: Ordered Merge rewired for in-scope modes; CI-wait retained as fallback and stacked-prs path
`#### Ordered Merge` SHALL make the auto-merge choreography the merge-all path for `cherry-pick-ladder` (and note it for `merge-auto`), replacing the foreground "wait for CI to pass" step for those modes: the operator arms the next PR and returns to its loop, verifying the merge on later ticks instead of foreground-waiting. The existing CI-wait choreography MUST remain, verbatim in intent, as (a) the `stacked-prs` merge-all path and (b) the rule-2 fallback when auto-merge is disabled on the repo. Cross-repo ordering barriers and the halt-dependents-only CI-failure policy are unchanged (a CI failure on an armed PR surfaces as a stall/escalation → disarm per rule 5).

- **GIVEN** an in-scope merge-all with auto-merge available on the repo
- **WHEN** the operator processes the sequence
- **THEN** each merge is a passive tick check (arm → tick-verify → arm next), not a foreground CI wait
- **GIVEN** the repo has auto-merge disabled, or the queue is `stacked-prs`
- **WHEN** merge-all runs
- **THEN** today's CI-wait choreography applies unchanged

#### R3: `merge-auto` arms the just-shipped PR
The `merge-auto` mode text (mode bullet and the per-change loop's merge-on-completion step) SHALL arm the just-shipped PR via the choreography instead of foreground CI-waiting on it — the rebase-next-onto-`origin/{default_branch}` step then follows the verified merge on a later tick.

- **GIVEN** a `merge-auto` queue where a change's PR just shipped
- **WHEN** the operator reaches its merge-on-completion step
- **THEN** it arms the PR (rules 1–5 apply, a one-PR sequence position) and proceeds; the next change's rebase waits for the verified merge

#### R4: Loop run-condition and tick gain "merge sequence in progress"
The loop run-condition SHALL gain a fourth condition — a merge sequence is in progress (an open `coordination` note for a merge sequence) — swept across every co-stated site in `fab-operator.md`: §2 Init step 4 ("If any tracked items exist…"), the §4 The Loop opening paragraph ("runs as long as… When all three are empty, stop"), §4 Tick Behavior step 7 ("stop when no tracked state remains"), and the §9 Key Properties `Uses /loop?` row. The tick SHALL drive the sequence: while a merge sequence is in progress, the tick runs the merge-sequence check (verify the armed PR's merge via timeline event; merged → advance the note and arm the next; unmerged → count toward the rule-3 stall).

- **GIVEN** a merge sequence in progress with an empty monitored set, no autopilot queue, and no watches
- **WHEN** the tick's loop-lifecycle step evaluates
- **THEN** the loop keeps running until the sequence's coordination note resolves

#### R5: In-file consistency sweep and safety-tier alignment
All in-file restatements of the merge-all behavior SHALL be reconciled: `#### Queue Completion Summary` ("Or ask me to merge all."), the per-change loop's `merge-auto` note (step 5–8 sentence), the §6 mode bullets, and §3 Safety (merging a PR stays Destructive tier; the existing merge-all confirmation covers the whole sequence — arming is part of the confirmed sequence, requiring no per-PR re-confirmation). Grep `merge all` / `wait for CI` / `CI to pass` in the file and reconcile every occurrence before finishing apply (code-quality.md § Sibling Sweeps).

- **GIVEN** the finished skill text
- **WHEN** grepping `wait for CI|CI to pass|merge all` in `fab-operator.md`
- **THEN** every occurrence is consistent with the choreography (in-scope modes arm; stacked-prs and the disabled-auto-merge fallback CI-wait)

### Non-Goals

- No Go changes, no new/changed `fab` commands, no `_cli-fab.md` or `_cli-agents.md` edits (C2 introduces no command surface).
- Phases B1 (`tick-start --diff`) and B2 (`fab pane questions`) — separate later changes.
- The plan File map's other `docs/memory/runtime/operator.md` rewrites (Design Constraints bullets, detection-policy sentences, Full Mediation extension, level-triggered deltas) — they belong to B1/B2; hydrate here touches only merge-all behavior + the C2 sequential-arming decision.
- No `docs/specs/` edits (aggregate specs do not restate merge-all mechanics — verified at intake).

### Design Decisions

#### Sequential arming with verified-merge gating, scoped to modes that target main
**Decision**: At most one armed PR per repo-sequence, armed only after the predecessor's merge is verified via a timeline event; `stacked-prs` keeps manual merge-all.
**Why**: An armed PR outlives the operator and fires without an event; arming more than one at a time (or arming a stacked PR) lets GitHub merge in an unverified order, into a wrong base (destroying a stack silently), or on stale-green checks after a no-re-CI base retarget. Sequential arming keeps merge order operator-verified while converting the foreground CI wait into a passive tick check.
**Rejected**: Arm-all-upfront (unordered merges, wrong-base hazard). Including `stacked-prs` (the win was always marginal there — its inter-merge choreography is operator-sequenced anyway — and the failure modes are the red-team's worst).
*Introduced by*: 260823-2c92-operator-auto-merge-choreography

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `#### Auto-Merge Choreography` subsection to `src/kit/skills/fab-operator.md` §6 (adjacent to `#### Ordered Merge`): scope statement + stacked-prs exclusion with reasons, the five MUST rules (sequential arming, arming-failure shapes, 3-tick stall rule, persisted coordination note via the `fab operator note` verbs, disarm-on-halt), and the squash-unless-directed merge-method line <!-- R1 -->
- [x] T002 Rewire `#### Ordered Merge` in `src/kit/skills/fab-operator.md`: in-scope modes merge-all via the choreography (arm → tick-verify → arm next); CI-wait retained as the stacked-prs path and the rule-2 disabled-auto-merge fallback; CI-failure/halt policy hooked to rule 5 <!-- R2 -->
- [x] T003 Update the `merge-auto` mode bullet and the per-change loop's merge-on-completion sentence in `src/kit/skills/fab-operator.md` to arm the just-shipped PR instead of foreground CI-waiting <!-- R3 -->

### Phase 2: Integration & Edge Cases

- [x] T004 Sweep the loop run-condition's four co-stated sites in `src/kit/skills/fab-operator.md` (§2 Init step 4, §4 opening paragraph, §4 Tick Behavior step 7, §9 Key Properties loop row) to add "merge sequence in progress", and wire the tick's merge-sequence check (tick step 4 or the choreography section) <!-- R4 -->
- [x] T005 In-file consistency sweep of `src/kit/skills/fab-operator.md`: reconcile `#### Queue Completion Summary`, §3 Safety (arming covered by the merge-all confirmation), and every `merge all` / `wait for CI` / `CI to pass` occurrence; run the grep to verify <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §6 carries an `#### Auto-Merge Choreography` subsection with all five rules stated as MUSTs, the cherry-pick-ladder/merge-auto scope, and the stacked-prs exclusion with both red-team reasons
- [x] A-002 R2: `#### Ordered Merge` routes in-scope merge-all through the choreography and retains the CI-wait choreography as the stacked-prs path and the disabled-auto-merge fallback
- [x] A-003 R3: the `merge-auto` texts arm the just-shipped PR; the next change's rebase keys off the verified merge
- [x] A-004 R4: all four co-stated loop run-condition sites include "merge sequence in progress", and the tick drives the merge-sequence check

### Behavioral Correctness

- [x] A-005 R1: rule 4 uses the merged Phase A verbs exactly (`fab operator note add --kind coordination` / `update` / `resolve`) and states that a restarted operator re-orients from the note
- [x] A-006 R2: the choreography replaces foreground CI-wait only for `cherry-pick-ladder`/`merge-auto`; `stacked-prs` merge-all text is functionally unchanged

### Scenario Coverage

- [x] A-007 R1: the arming-failure shapes cover all three cases (draft → `gh pr ready`; already-clean → merge directly; repo-disabled → CI-wait fallback) with the /git-pr-creates-drafts note
- [x] A-008 R5: grep of `merge all` / `wait for CI` / `CI to pass` in `fab-operator.md` shows every occurrence consistent with the choreography

### Edge Cases & Error Handling

- [x] A-009 R1: the stall rule names the 3-tick threshold, `mergeableState`, the `CONFLICTING` → escalate path, and why polling is needed (auto-merge fails silently); disarm-on-halt names `gh pr merge --disable-auto`

### Code Quality

- [x] A-010 Pattern consistency: the new subsection matches the file's existing §6 heading/prose/MUST conventions and the canonical source is `src/kit/skills/fab-operator.md` (never `.claude/skills/`)
- [x] A-011 No unnecessary duplication: the choreography is stated once; other sites point at it rather than restating rules (owner-or-pointer)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the foreground CI-wait choreography is retained as the `stacked-prs` path and the rule-2 fallback, per R2)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The tick's merge-sequence check hooks into Tick Behavior step 4 (the autopilot-dispatch step) rather than adding a new numbered step | The sequence is autopilot aftermath; step 4 already runs "the next autopilot action" and the choreography section owns the per-tick check detail | S:55 R:90 A:75 D:65 |
| 2 | Confident | Stall-rule N stated as 3 consecutive ticks in the skill text | Carried from intake assumption 9 — the plan leaves N symbolic; a trivially adjustable skill knob | S:35 R:85 A:60 D:50 |
| 3 | Tentative | Merge method stated as squash-unless-directed <!-- assumed: gh pr merge --auto requires a method flag unless repo settings force one; existing stacked-prs text assumes squash --> | Carried from intake assumption 12 | S:30 R:70 A:45 D:35 |
| 4 | Confident | §3 Safety needs only a clarifying clause (arming rides the existing merge-all confirmation), not a new tier row | Merging stays Destructive; the sequence is confirmed once at "merge all", matching the autopilot upfront-confirmation precedent | S:60 R:85 A:80 D:70 |

4 assumptions (0 certain, 3 confident, 1 tentative).
