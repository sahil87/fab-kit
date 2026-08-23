# Intake: Operator Auto-Merge Choreography

**Change**: 260823-2c92-operator-auto-merge-choreography
**Created**: 2026-08-23

## Origin

Promptless dispatch from `/fab-proceed` (no interactive questioning; would-be questions are recorded as deferred Unresolved rows — none arose, see Assumptions).

**Traceability**: This is **Change 2 of 4** in the same-repo ordered autopilot queue executing the plan doc `fab/plans/sahil/26-08-23-operator-offload-plan.md`, specifically its **§ Phase C2 — auto-merge, re-specced (promoted into the queue)** section, plus the queue ordering in § Critical couplings & sequencing item 2 and the File map row for `src/kit/skills/fab-operator.md` ("C2 merge rules" portion only). The plan doc was written 2026-08-23 from a /fab-discuss brainstorm and revised the same day after a four-agent review; C2 was promoted by the scope review (cheapest large win) and re-specced after the failure red-team showed the original one-liner spec was dangerous.

**Dependency**: Change 1 (Phase A, `fab operator note` verb family) is **MERGED to main as PR #615** — the `fab operator note add/resolve/update/list` verbs and the `kind: coordination` note are available as a hard dependency (verified present in `src/kit/skills/fab-operator.md` §4 Notes and `docs/memory/runtime/operator.md`, whose worked example already shows a coordination note with text `Phase 2 of 4 — auto-merge armed behind s2gw`).

**Out of scope**: Changes 3 (Phase B1 `tick-start --diff`) and 4 (Phase B2 `fab pane questions`) are separate later changes. The plan File map's other `docs/memory/runtime/operator.md` rewrites (Design Constraints bullets, detection-policy sentences, Full Mediation extension, level-triggered-deltas decision) and the `pane-commands.md` re-scope belong to B1/B2, not this change.

## Why

1. **The pain point**: after a `cherry-pick-ladder` autopilot queue completes and the user says "merge all", today's § Ordered Merge choreography has the operator merge each PR and **foreground-wait for CI to pass** before proceeding to the next. These CI-watch stretches are the longest operator-busy periods in the whole coordination lifecycle — the operator sits blocked on GitHub checks doing purely mechanical waiting.
2. **The consequence of not fixing it**: the operator's queue (its attention) stays consumed by CI polling; strategic questions from other agents queue behind a merge sequence; and the plan's whole offload goal (free the operator's context for judgment work) is defeated at exactly the moment the fleet is otherwise idle.
3. **Why this approach**: GitHub auto-merge (`gh pr merge --auto`) turns the foreground CI-watch into a **passive tick check** — arm the PR, let GitHub merge it when checks pass, verify on a later tick, arm the next. The original one-line spec ("just use auto-merge") was shown dangerous by the plan's red-team review — an armed PR outlives the operator, can merge into a wrong base, and fails silently on conflicts — hence this change ships the **re-specced five-MUST-rule choreography**, scoped to the modes where PRs target main. Honest sizing (from the plan): the win is real for `cherry-pick-ladder`; it was always marginal for `stacked-prs`, hence the scope cut.

## What Changes

**Skill text only.** The sole file changed is `src/kit/skills/fab-operator.md` (canonical source; deployed via `fab sync` — never edit `.claude/skills/` directly). NO Go changes, NO new/changed `fab` commands — so the constitution's simultaneous-signature rule does NOT require a `_cli-fab.md` update (C2 introduces no command surface). No `_cli-agents.md` changes.

### New auto-merge choreography section in `fab-operator.md`

Add an auto-merge choreography section in/near the autopilot merge-all guidance (§6, adjacent to `#### Ordered Merge`), **scoped to the `cherry-pick-ladder` merge mode (and trivially `merge-auto`)**, where PRs target main. In these modes the choreography becomes the merge-all path: arm via GitHub auto-merge instead of foreground CI-waiting, with fallback to today's CI-wait choreography when auto-merge is unavailable (rule 2, third shape).

**`stacked-prs` keeps today's manual merge-all unchanged** — its inter-merge choreography (retarget-verify, `rebase --onto`, force-push) is operator-sequenced anyway, and an armed stacked PR can merge into its dependency's *branch* (destroying the stack silently) or fire on stale-green checks after GitHub's no-re-CI base retarget. The section must state this exclusion and its reasons.

**All five rules are MUSTs in the skill text:**

1. **Sequential arming**: at most one armed PR per repo-sequence. Arm PR_n only after PR_{n-1}'s merge is **verified** (timeline event, not assumption). Never arm a PR whose base is another PR's branch.
2. **Arming-failure shapes**: draft PR → `gh pr ready` first (fab's `/git-pr` creates drafts, so this is every autopilot PR); "already clean" rejection (no required checks on the repo, so auto-merge has nothing to wait for) → merge directly; auto-merge disabled on the repo → today's CI-wait choreography.
3. **Stall rule**: an armed PR unmerged after N ticks → check `mergeableState`; `CONFLICTING` → escalate (auto-merge fails silently — there is no event to observe). <!-- assumed: N fixed at 3 consecutive ticks in the skill text — the plan leaves N symbolic; ~10 minutes at the 3m cadence balances noise vs. latency and is a trivially adjustable skill knob -->
4. **Persisted sequence** (the Phase A dependency): merge-all state is conversational today, and an armed PR **outlives the operator** (survives `/clear`, crash, abandonment) — so starting a merge sequence writes a `kind: coordination` note via `fab operator note add --kind coordination` recording the sequence, current position, and the armed PR; the note is updated as the sequence advances (`fab operator note update`) and resolves at sequence end via `fab operator note resolve`. A restarted operator re-orients from the note and resumes verification/arming.
5. **Disarm on halt**: any halt/escalation runs `gh pr merge --disable-auto` on the remaining armed PRs — the existing halt-dependents-only CI-failure policy assumes unstarted merges stay unstarted, which armed auto-merge violates.

Arming rides `gh pr merge --auto` with the repo's standard merge method (squash unless the user directs otherwise — today's stacked-prs text already assumes squash merges). <!-- assumed: merge method — gh requires a method flag with --auto unless repo settings permit exactly one; skill text states squash-unless-directed as the default -->

### Loop run-condition gains "merge sequence in progress"

At merge-all time the autopilot queue is exhausted and the monitored set is typically empty — today the `/loop` may not even be running to do the "verify on later ticks" work. The §4 loop run-condition ("runs as long as the monitored set is non-empty, an autopilot queue is active, or any watch is configured. When all three are empty, stop the loop") gains a fourth condition: **a merge sequence is in progress** (an open coordination note for a merge sequence). Sweep every co-stated site of that run-condition in the same file: §2 Init step 4 ("If any tracked items exist, start the single loop"), §4 The Loop opening paragraph, §4 Tick Behavior step 7 ("Loop lifecycle — stop when no tracked state remains"), and §9 Key Properties' loop row.

### Sibling/consistency sweep (within `fab-operator.md`)

The five rules touch text that is restated in the same file: `#### Ordered Merge` (the CI-wait choreography stays as the stacked-prs path and the rule-2 fallback), `#### Queue Completion Summary` ("Or ask me to merge all"), the § Autopilot per-change loop's mode notes, §3 Safety (merging a PR is Destructive tier — the merge-all confirmation already covers the sequence; arming is part of the confirmed sequence), and the `merge-auto` paragraph (its per-PR merge-on-completion trivially arms the just-shipped PR instead of foreground CI-waiting). Grep `merge all` / `wait for CI` / `CI to pass` in the file and reconcile every occurrence before finishing apply (code-quality.md § Sibling Sweeps).

## Affected Memory

- `runtime/operator`: (modify) Autopilot/Ordered Merge behavioral text (merge-all for `cherry-pick-ladder`/`merge-auto` becomes sequential auto-merge arming with the five MUST rules; stacked-prs unchanged; loop run-condition gains merge-sequence-in-progress) plus one new Design Decision entry: **C2 sequential arming** (why one-armed-PR-at-a-time with verified-merge gating, and the stacked-prs scope cut). This is the single decision the plan's File map assigns to C2; the file's other rewrites listed there belong to B1/B2 and are NOT in scope.

## Impact

- **Files**: `src/kit/skills/fab-operator.md` only (plus `docs/memory/runtime/operator.md` at hydrate). No Go code, no tests (no `.go` change), no migrations (no user-data restructuring), no `_cli-fab.md` (no command-surface change), no `docs/specs/` (aggregate specs do not restate merge-all mechanics — verified `skills.md`/`glossary.md`).
- **Behavior contract**: operator-facing only — the merge-all flow for `cherry-pick-ladder`/`merge-auto` queues changes from foreground CI-wait to sequential auto-merge arming; `stacked-prs` merge-all and everything else is untouched.
- **Dependencies**: `gh` (already a constitution-sanctioned single-binary dependency; `gh pr merge --auto/--disable-auto`, `gh pr ready`, `gh pr view --json mergeableState`, timeline verification) and the merged `fab operator note` verb family (PR #615).
- **Risk**: skill-text change; misordered arming is the danger the five MUST rules exist to prevent — review should check each rule is stated as a MUST and the stacked-prs exclusion is explicit.

## Open Questions

- None — the plan doc's § Phase C2 is a post-red-team re-spec; the remaining low-stakes choices (stall-rule N, merge method) are graded in Assumptions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope to `cherry-pick-ladder` + trivially `merge-auto`; `stacked-prs` keeps today's manual merge-all | Plan explicit, with red-team reasons (stack destruction, stale-green after base retarget) | S:95 R:70 A:90 D:95 |
| 2 | Certain | All five rules land as MUSTs in the skill text | Plan states "Rules, all MUSTs in the skill text" verbatim | S:95 R:85 A:90 D:95 |
| 3 | Certain | Skill-only: sole edited file is `src/kit/skills/fab-operator.md`; no `_cli-fab.md` update (no command surface) | Dispatch description + constitution rule keys on CLI signature changes, of which there are none | S:90 R:80 A:90 D:90 |
| 4 | Certain | Persisted sequence uses `fab operator note add --kind coordination` / `update` / `resolve` (Phase A, merged PR #615) | Dependency verified present in skill §4 Notes and memory; the memory example even shows this exact note shape | S:95 R:80 A:95 D:90 |
| 5 | Certain | Disarm-on-halt (`gh pr merge --disable-auto` on remaining armed PRs) hooks the existing halt-dependents-only escalation path | Plan rule 5 explicit; halt policy already in skill § Ordered Merge | S:90 R:80 A:85 D:85 |
| 6 | Confident | New section placed in §6 adjacent to `#### Ordered Merge`, as the merge-all path for in-scope modes | Plan says "in/near its autopilot merge-all guidance"; exact heading/position is apply's call | S:70 R:90 A:80 D:70 |
| 7 | Confident | Auto-merge choreography **replaces** foreground CI-wait as the default merge-all path for in-scope modes; repo-disabled auto-merge falls back to today's CI-wait (rule 2) | The stated win ("foreground CI-watch becomes a passive tick check") only exists if it is the default path; fallback shape is explicit in rule 2 | S:75 R:75 A:80 D:75 |
| 8 | Certain | Loop run-condition sweep covers all co-stated sites (§2 Init step 4, §4 opening, Tick step 7, §9 Key Properties) | Plan names the run-condition addition; the sweep class is the repo's standing code-quality rule | S:85 R:85 A:85 D:80 |
| 9 | Confident | Stall-rule N fixed at 3 consecutive ticks (~10m at the 3m cadence) | Plan leaves N symbolic; a skill-text knob, trivially adjusted; 3 balances conflict-detection latency vs. false stalls on slow CI | S:35 R:85 A:60 D:50 |
| 10 | Confident | Affected Memory is `runtime/operator` (modify) only, limited to merge-all behavior + the C2 sequential-arming decision | Plan File map assigns exactly that decision bullet to C2; other operator.md rewrites belong to B1/B2; pane-commands.md untouched | S:75 R:85 A:80 D:75 |
| 11 | Confident | `change_type` = `feat`, set as an explicit override (re-inference had flipped it to `fix` on a keyword hit in this intake's prose) | New operator capability (auto-merge arming), not a fix/refactor/docs correction, despite being skill-text-only — skill text is behavior in this repo | S:60 R:90 A:80 D:70 |
| 12 | Tentative | Arming uses the repo's standard merge method — squash unless the user directs otherwise | `gh pr merge --auto` needs a method unless repo settings force one; existing stacked-prs text assumes squash merges, but per-repo conventions genuinely vary and the codebase does not settle it | S:30 R:70 A:45 D:35 |

12 assumptions (6 certain, 5 confident, 1 tentative, 0 unresolved).
