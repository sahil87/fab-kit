# Plan: Review-PR Recovery — Terminalize a Stuck Review-PR Stage, One Bounded Recovery Shape for the Operator

**Change**: 260916-vb4l-review-pr-recovery
**Intake**: `intake.md`

## Requirements

### Pipeline: `/git-pr-review` timeout budget (D1, D5)

#### R1: Second consecutive Copilot timeout on the same PR terminalizes the stage
`src/kit/skills/git-pr-review.md` Step 2 Phase 2 ("Copilot request and poll", the 20-attempts-exhausted branch) MUST, when Step 0 resolved a change, (a) record the timeout with `fab log command "git-pr-review" {name} "timeout pr={number}"`, (b) count `timeout pr={number}` markers in `fab/changes/{name}/.history.jsonl` written **after** the most recent `review-pr` `stage-transition` line (one `awk`, substring match, key order irrelevant), and (c) branch: count `< 2` → today's behavior verbatim (pending message, Step 6 outcome **timeout**, stage left `active`); count `≥ 2` → print the review-gate-unavailable message, record `fab log command "git-pr-review" {name} "review-gate-unavailable pr={number}"`, and exit through Step 6 with outcome **no-reviews** (the existing `fab status finish <change> review-pr git-pr-review 2>/dev/null || true`). The 30 s × 20 poll, the synchronous-poll discipline note, the two-login predicate, and the REST-not-GraphQL note MUST NOT change. With no resolved change the pre-change behavior stands (no history file, no stage).

- **GIVEN** a change at `review-pr` `active` whose history has zero `timeout pr=N` markers since the last `review-pr` transition
- **WHEN** the Copilot poll exhausts 20 attempts on PR N
- **THEN** one `timeout pr=N` marker is logged, the pending message is printed, and the stage stays `active` (outcome **timeout**)

- **GIVEN** the same change on a later invocation, the poll exhausts again on PR N (count becomes 2)
- **WHEN** the branch evaluates
- **THEN** the skill prints `Review gate unavailable — no Copilot review landed after 2 requests on PR #N (~20 minutes). Finishing review-pr unreviewed (reason: review-gate-unavailable).`, logs the `review-gate-unavailable pr=N` marker, and finishes `review-pr` via the **no-reviews** row

- **GIVEN** `fab status reset <change> review-pr` or a re-ship wrote a fresh `review-pr` `stage-transition` line
- **WHEN** the count runs
- **THEN** earlier markers are not counted ("consecutive" is per activation, never lifetime)

#### R2: Outcome table, description, result summary, idempotency
The Step 6 outcome table's **no-reviews** "Includes" cell MUST add "second consecutive Copilot timeout on the same PR (reason `review-gate-unavailable`, logged)" and the **timeout** cell MUST read "first timeout in this activation only; the `timeout pr=<n>` marker is logged so the next invocation can count it" — stage-action and commit columns unchanged. The frontmatter `description` MUST gain the clause "a second consecutive timeout on the same PR finishes the stage unreviewed (reason `review-gate-unavailable`)". The dispatched-arm result guidance MUST state that the terminalized exit writes `outcome: no-reviews` with a `summary` naming the reason. A short idempotency note MUST state that a re-run after the terminalized finish is a clean no-op on every path (`start` no-ops on `done`; a late review is processed normally; another timeout exits via `no-reviews` and `finish` no-ops). No new `phase` value is added.

- **GIVEN** the edited skill
- **WHEN** a reader consults Step 6
- **THEN** both rows name the first-vs-second timeout split and no stage action changed

### Operator: `#### Review-PR Recovery` (D2, D3, D4, D5)

#### R3: One new section holds the reason table and the three bounded actions
`src/kit/skills/fab-operator.md` MUST gain exactly one new `#### Review-PR Recovery` sub-section under §6 Coordination Patterns immediately after `#### Auto-Merge Choreography`, containing: the lead paragraph (one condition / one reason / one bounded action / per item, not a queue / halt-dependents-only still applies / detection = `timeout` outcome, `gh pr checks` failed required check, `gh pr view --json mergeable` == `CONFLICTING` — never GraphQL timeline or a new drop counter); the reason table with rows `no-reviewer` (one routed `/git-pr-review <change>` re-send to the idle authoring pane past the §8 stuck threshold, pane gone → one recovery spawn; the operator never finishes/skips the stage itself), `ci-failed` (one fix round — the fix request text below sent to the authoring pane, PR stays armed, `fab operator track update pr-<n> --scope '{"recovery":"ci-fix","fix_head":"<headRefOid>"}'`; green → continue with ` · after CI fix round`; still red on a head ≠ `fix_head` or 30 m without a push → today's halt-dependents-only + rule 5 disarm + escalate **with the agent's diagnosis** from `rk mux capture`), and `conflicting` (in the change's worktree — live pane's worktree, else `wt create … --checkout <branch>` — `git fetch origin && git rebase origin/{default_branch} && git push --force-with-lease`, re-arm `gh pr merge --auto --squash <n>` idempotently, `--scope '{"recovery":"rebase"}'`; Recoverable tier, no confirm; `CONFLICTING` again with `scope.recovery == "rebase"` or a rebase conflict (`git rebase --abort`) → disarm + escalate naming files); the verbatim fix request text; the shared **Recovery spawn** paragraph (§6 spawn steps 1–8 with the existing-branch route, tracked as a change-less `<change>-fix` pane item, `track rm` when resolved, budget 1 spawn per item per reason); **Locating the authoring pane** (`fab pane map --all-sessions` join on `change`; PR→change link via `scope.change` on the `github-pr` item); the **Report wording** list; and the **Recovery state** note (`scope.recovery` durable across compaction; `merge-auto`'s one-PR degenerate case). Contents list unchanged (§6 sub-headings are not listed).

- **GIVEN** an armed `github-pr` item whose required check fails
- **WHEN** the tick probes it
- **THEN** the operator sends the fix request once, records `recovery: ci-fix` + `fix_head`, and does NOT disarm

- **GIVEN** the fix round pushed a new head and CI failed again
- **WHEN** the tick probes it
- **THEN** the operator disarms, applies halt-dependents-only, and escalates with the agent's one-line diagnosis

- **GIVEN** an armed item at `unchanged ≥ 3`, no failed check, `mergeable == CONFLICTING`, no `scope.recovery`
- **WHEN** the tick evaluates rule 3
- **THEN** the operator rebases, force-pushes with lease, re-arms once, records `recovery: rebase`, and announces without confirming

- **GIVEN** a tracked pane item at stage `review-pr`, agent `idle` past the stuck threshold
- **WHEN** the tick's stuck check fires
- **THEN** the operator re-sends `/git-pr-review <change>` once (routed skill send, `idle` pre-send gate) and a later idle-at-`review-pr` is a §3 stuck-agent report, not another send

#### R4: Existing owners route through the new section (owner-or-pointer, no duplicated text)
The following sites in `fab-operator.md` MUST be amended as pointers, not restatements: §6 Ordered Merge "CI failure during ordered merge (halt-dependents-only)" opens with "A failed required check first runs § Review-PR Recovery's `ci-failed` round; only an exhausted round halts:" and its example escalation gains the diagnosis clause; Auto-Merge Choreography rule 3 replaces "disarm per rule 5 and apply the halt-dependents-only policy" with a pointer to `ci-failed` and "`CONFLICTING` → disarm and escalate" with a pointer to `conflicting`; rule 4's `track add` `--scope` carries `"change":"<id>"`; rule 5 reads "Any halt or escalation — CI failure, stall, conflict — **after § Review-PR Recovery's bounded action is exhausted** — MUST run `gh pr merge --disable-auto`…"; §1 "Coordinate, don't execute" allowlist adds "and § Review-PR Recovery"; §3 Bounded Retries gains one row `Review-PR Recovery (§6) | 1 per reason | per its table`; the Pipeline-first bullet (§6 The pane Kind and Spawn Rules) gains the one-clause carve-out naming the CI fix request as the single sanctioned raw send that amends a shipped change. §3 Confirmation Tiers and §9 Key Properties MUST NOT change.

- **GIVEN** the edited file
- **WHEN** `grep -n 'Review-PR Recovery' src/kit/skills/fab-operator.md` runs
- **THEN** it hits the new heading plus each of the seven pointer sites, and no site restates the reason table's action text

#### R5: Unreviewed merges are reported, never confirmed, never silent
The new section MUST specify: on the verified-merge tick, for every PR merged this tick with zero non-`PENDING` reviews (`gh pr view <n> --json reviews -q '[.reviews[] | select(.state != "PENDING")] | length'` == `0`) the per-merge line appends ` · unreviewed` and the tick emits one summary line `merged with review gate unavailable: #{n} ({change}), #{m} ({change})`. No confirmation prompt is added anywhere for merging an unreviewed PR; no `rk notify` is added.

- **GIVEN** two PRs merged on one tick, one with a landed Copilot review and one with none
- **WHEN** the tick reports
- **THEN** only the second line carries ` · unreviewed` and exactly one summary line names it

### Sweep: sibling sites describing the review-pr `timeout` outcome (D6)

#### R6: Every site that restates the timeout outcome names the first-vs-second split
`src/kit/skills/fab-fff.md` (Step 5 "If timeout" paragraph, Error Handling "Review-PR timeout" row, CLI-arm `outcome: timeout` / `outcome: no-reviews` rows), `src/kit/skills/fab-continue.md` (the `review-pr`/`active` timeout clause), and `src/kit/skills/_preamble.md` (§ Dispatch-Prompt Obligations review-pr result comment) MUST gain the parenthetical "(first timeout in the activation; a second consecutive timeout on the same PR is a `no-reviews` outcome with reason `review-gate-unavailable` — report its `summary`)" or an equivalent one-clause pointer. `/fab-fff` MUST still stop (not re-dispatch) on the first timeout. `src/kit/skills/_pipeline.md` MUST be re-checked and left alone if it has no timeout text. A repo-wide grep for `timeout`, `no-reviews`, and `Copilot review requested` under `src/kit/skills/` MUST be run before apply finishes and every review-pr-outcome hit updated; hits meaning dispatch-wait/poll timeouts are left alone.

- **GIVEN** the sweep grep
- **WHEN** it runs after the edits
- **THEN** no `src/kit/skills/` site describes the Copilot timeout as an unconditional leave-`active` outcome

### Constraints (Constitution I, III, V; code-quality.md)

#### R7: Canonical sources only, deployable citations, guard passes
Only `src/kit/skills/*.md` MUST be edited (never `.agents/skills/` or `.claude/skills/`). No edited skill may cite `docs/specs/*`, `docs/memory/*` (outside the host convention paths), `docs/site/*`, or `src/go/*` as an authority. No Go, no new `fab` verb, no new `.status.yaml` field, no migration, no template change. The Go portability guard test in `src/go/fab-kit/cmd/fab/` MUST pass after the edits, and `gofmt` is moot (no Go touched).

- **GIVEN** the finished apply
- **WHEN** `git status --porcelain` and the guard test run
- **THEN** only `src/kit/skills/{git-pr-review,fab-operator,fab-fff,fab-continue,_preamble}.md` and `fab/changes/260916-vb4l-review-pr-recovery/` are modified and the guard passes

### Non-Goals

- No change to the 10-minute poll window or the synchronous-poll discipline
- No autonomous second `/git-pr-review` dispatch inside `/fab-fff` — the second invocation comes from the operator's `no-reviewer` re-send or a user re-run
- No dependency-chain merge-ordering change; `stacked-prs` keeps its own `rebase --onto` step and conflict escalation (D4 is armed-PR-only)
- No fleet-level "review gate degraded" condition, no batch confirm, no timeline-based detection (D7)
- Backlog marks (`[vb4l]` by archive, `[hf6x]` by hand) happen at archive, not apply

### Design Decisions

#### Review-PR Always Terminalizes: the Second Consecutive Timeout Finishes Unreviewed
**Decision**: `/git-pr-review` counts `timeout pr=<n>` markers it writes to the change history with `fab log command`, windowed by the last `review-pr` `stage-transition` line; the second consecutive timeout on the same PR exits through the existing **no-reviews** outcome with reason `review-gate-unavailable`.
**Why**: the `timeout` outcome was the only stage exit that was neither `done` nor `failed`, and it was absorbing — the operator's completion predicate (`review-pr` done/skipped) could never fire once the review gate dropped requests. A degraded gate should behave like an absent one (`copilot: false` already finishes the stage unreviewed), with the reason logged and reported. `fab log command` is pure telemetry that already exists, so no new field or verb is needed.
**Rejected**: parking `review-pr` as `failed` (still not `done`/`skipped`, so items stay unreachable and the user is paged); a new `stage_metrics.review-pr.phase` value (single-valued, would overload an existing field); a batch confirm before merging unreviewed PRs (merge-all / merge-auto already carry the user's Destructive-tier confirm; `review` already passed).
*Introduced by*: 260916-vb4l-review-pr-recovery

#### Review-PR Recovery: One Bounded Action per Stuck Reason, No Batch Confirm
**Decision**: the operator treats a stuck `review-pr` as one condition with a reason (`no-reviewer` / `ci-failed` / `conflicting`), each with exactly one bounded action (re-send once / one CI fix round / rebase-and-re-arm once) recorded on the `github-pr` item's `scope.recovery`, after which the existing Ordered Merge halt-dependents-only + rule 5 disarm + escalation applies with more evidence attached. Detection uses only the `timeout` outcome, `gh pr checks`, and `mergeable == CONFLICTING`.
**Why**: the three reasons already had owners (the review skill's outcome table, Ordered Merge, Auto-Merge rule 3); plugging one action into each keeps the merge choreography intact and avoids a second loop. Per-item recovery matches the observed failure (independent PRs, not a queue). The CI fix request is the single sanctioned raw send into a fab-project agent because it amends a change that already completed the pipeline.
**Rejected**: a standalone "review gate degraded" playbook with its own rebase-wait-merge loop (duplicates Ordered Merge + Auto-Merge); GraphQL timeline / `reviewRequests` detection (omits bot reviewers); an in-context recovery counter (lost at compaction — `scope` on the item is durable).
*Introduced by*: 260916-vb4l-review-pr-recovery

## Tasks

### Phase 1: Setup

- [x] T001 Baseline the sweep class: run `grep -rn -i 'timeout\|no-reviews\|Copilot review requested' src/kit/skills/` and record every hit that describes the review-pr Copilot timeout outcome (vs. dispatch-wait/poll timeouts) as the definitive sweep list for T006; confirm `src/kit/skills/_pipeline.md` has none <!-- R6 -->

### Phase 2: Core Implementation

- [x] T002 Edit `src/kit/skills/git-pr-review.md` Step 2 Phase 2 "Copilot request and poll" step 2's 20-attempts-exhausted branch: log the `timeout pr={number}` marker, count markers after the last `review-pr` `stage-transition` line with the single `awk` from the intake (§ What Changes 1), branch `< 2` → today's text verbatim / `≥ 2` → the review-gate-unavailable message + `review-gate-unavailable pr={number}` marker + Step 6 outcome **no-reviews**; guard the whole branch on Step 0 having resolved a change <!-- R1 -->
- [x] T003 Edit `src/kit/skills/git-pr-review.md` Step 6 outcome table (**no-reviews** and **timeout** "Includes" cells only), frontmatter `description`, the dispatched-arm result-file guidance (`outcome: no-reviews` + reason-naming `summary`), and add the idempotency note; leave stage actions, the poll, the discipline note, and Phase Sub-State Tracking untouched <!-- R2 -->
- [x] T004 [P] Add `#### Review-PR Recovery` to `src/kit/skills/fab-operator.md` under §6 immediately after `#### Auto-Merge Choreography`: lead paragraph, reason table (`no-reviewer` / `ci-failed` / `conflicting` with Detected-by / One bounded action / Then columns), verbatim fix request text, Recovery spawn paragraph, Locating the authoring pane, Report wording list (including the D2 ` · unreviewed` per-merge suffix and `merged with review gate unavailable: …` summary line), Recovery state note <!-- R3, R5 -->
- [x] T005 Amend the seven pointer sites in `src/kit/skills/fab-operator.md` (Ordered Merge CI-failure paragraph + example line; Auto-Merge rules 3, 4 `--scope` `"change"`, 5 exhausted-clause; §1 "Coordinate, don't execute" allowlist; §3 Bounded Retries pointer row; §6 Pipeline-first carve-out clause) as pointers to § Review-PR Recovery — no restated action text; verify §3 Confirmation Tiers and §9 Key Properties are byte-identical <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Sweep the T001 list: `src/kit/skills/fab-fff.md` (Step 5 "If timeout", Error Handling "Review-PR timeout" row, CLI-arm `outcome: timeout` / `outcome: no-reviews` rows), `src/kit/skills/fab-continue.md` (`review-pr`/`active` timeout clause), `src/kit/skills/_preamble.md` (§ Dispatch-Prompt Obligations review-pr result comment) — add the first-vs-second parenthetical/pointer; `/fab-fff` still stops on the first timeout; re-run the grep and confirm zero unconditional leave-`active` descriptions remain <!-- R6 -->
- [x] T007 Verify constraints: `git status --porcelain` shows only the five `src/kit/skills/` files + the change folder; grep the five edited files for `docs/specs/`, `docs/memory/` (non-convention paths), `docs/site/`, `src/go/` citations and remove any; run the Go portability guard test in `src/go/fab-kit/cmd/fab/` (`go test ./src/go/fab-kit/cmd/fab/ -run 'Portab|Guard|Cite|Path'` — widen to the package if the name filter matches nothing) and record the result <!-- R7 -->

## Execution Order

- T001 blocks T006 (the sweep list)
- T002 blocks T003 (same file, sequential edits)
- T004 blocks T005 (the pointers target the new heading)
- T002–T003 and T004–T005 are independent file pairs and may run in parallel
- T007 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: The poll-exhausted branch logs `timeout pr={number}`, counts markers after the last `review-pr` `stage-transition` line with one `awk`, and routes `< 2` to **timeout** (today's text) and `≥ 2` to **no-reviews** with the `review-gate-unavailable pr={number}` marker and message
- [x] A-002 R2: Step 6's **no-reviews** and **timeout** "Includes" cells, the frontmatter `description`, the result-file `summary` guidance, and the idempotency note are present; stage-action and commit columns are unchanged
- [x] A-003 R3: `#### Review-PR Recovery` exists once, directly after `#### Auto-Merge Choreography`, with the lead paragraph, the three-row reason table, the verbatim fix request text, Recovery spawn, Locating the authoring pane, Report wording, and Recovery state
- [x] A-004 R4: All seven pointer sites route to § Review-PR Recovery; none restates the reason table's action text
- [x] A-005 R5: The ` · unreviewed` per-merge suffix and the `merged with review gate unavailable: …` summary line are specified, with the zero-non-`PENDING`-reviews probe, and no confirmation or `rk notify` is added
- [x] A-006 R6: `fab-fff.md`, `fab-continue.md`, and `_preamble.md` each carry the first-vs-second timeout pointer; `_pipeline.md` is unchanged
- [x] A-007 R7: Only the five canonical skill files and the change folder are modified; the Go portability guard passes

### Behavioral Correctness

- [x] A-008 R1: The first timeout in an activation still leaves `review-pr` `active` and prints the existing pending message (explicit-change form preserved when `<change>` was passed)
- [x] A-009 R1: A `review-pr` `stage-transition` line written after earlier markers resets the count (per-activation, not lifetime)
- [x] A-010 R3: In the `ci-failed` row the armed PR stays armed during the round and is disarmed only when the round is exhausted; the exhausted escalation carries the agent's diagnosis line
- [x] A-011 R3: In the `conflicting` row the rebase + `--force-with-lease` + re-arm runs at most once per item (`scope.recovery == "rebase"` guards the second pass) and is announced without a confirm
- [x] A-012 R4: Rule 5's disarm is conditioned on the bounded action being exhausted; rule 3 no longer says "disarm and escalate" for `CONFLICTING`

### Scenario Coverage

- [x] A-013 R1: The `awk` window logic is spelled out so a reader can verify it against a sample `.history.jsonl` with a `stage-transition` line followed by two `timeout pr=N` command events
- [x] A-014 R3: The `no-reviewer` row's budget is one re-send per item per activation and a later idle-at-`review-pr` is a §3 stuck-agent report
- [x] A-015 R3: The Recovery spawn tracks a change-less `<change>-fix` pane item (no `--change`) and `track rm`s it when the round resolves; budget 1 spawn per item per reason

### Edge Cases & Error Handling

- [x] A-016 R1: With no resolved change (Step 0 found none) the branch is skipped and the pre-change pending-message behavior stands
- [x] A-017 R2: Re-running `/git-pr-review` after the terminalized finish is a clean no-op on every path (late review processed, `finish` no-ops, another timeout exits via `no-reviews`)
- [x] A-018 R3: Pane gone (`pane_death` / `agent_exited` acked, or item removed) routes each row to the shared Recovery spawn; the `conflicting` row needs no agent
- [x] A-019 R3: A rebase that itself conflicts is aborted (`git rebase --abort`) and escalates naming the conflicting files
- [x] A-020 R5: A `copilot: false` project's merges also read ` · unreviewed` (same probe, still true)

### Code Quality

- [x] A-021 Pattern consistency: New prose follows `fab-operator.md` conventions (§-numbered references, Destructive/Recoverable tier vocabulary, bare-markdown report lines) and `git-pr-review.md`'s step/phase structure
- [x] A-022 No unnecessary duplication: The reason table's action text appears once (in the new section); every other site is a pointer (owner-or-pointer)
- [x] A-023 Canonical source only: No edits under `.agents/skills/` or `.claude/skills/`
- [x] A-024 Deployable citations: No edited skill cites `docs/specs/*`, `docs/memory/*` (non-convention), `docs/site/*`, or `src/go/*` as an authority
- [x] A-025 Sibling sweep complete: The T001 grep list is fully applied; `fab-ff.md` checked (no review-pr step, expected untouched)
- [x] A-026 No magic numbers unnamed: the budget (2 invocations ≈ 20 minutes), the stuck threshold (§8, 15 m default), and the fix-round bound (30 m, §6 Failures) are each tied to their named source in prose

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change edits prose in place (timeout branch, outcome-table cells, Ordered Merge paragraph, Auto-Merge rules) and adds one new section; no existing file, symbol, or block is left redundant or unused

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The history path in the `awk` is `fab/changes/{name}/.history.jsonl` relative to the repo root, matching the skill's existing path convention | `_preamble.md` § Path Convention; `fab log command` writes there per `_cli-fab.md` | S:90 R:95 A:95 D:90 |
| 2 | Confident | `gh pr checks <n> --required --json name,state,link` is the failed-check probe; `link` is a supported field | `gh pr checks --json` exposes `link` alongside `name`/`state` in current gh; the intake names it; apply verifies with `gh pr checks --help` and falls back to `gh run view` links if absent | S:70 R:85 A:85 D:75 |
| 3 | Confident | `fab pane map --all-sessions` is the pane→change join for locating the authoring pane | Named in the intake; apply verifies the flag in `_cli-fab-pane.md` and substitutes the documented all-sessions form if it differs | S:65 R:85 A:85 D:70 |
| 4 | Certain | `fab-operator.md`'s Contents list is not amended (§6 sub-headings are not listed today) | Verified in the intake against the current file | S:90 R:95 A:95 D:90 |
| 5 | Certain | The sweep touches exactly `fab-fff.md`, `fab-continue.md`, `_preamble.md`; `_pipeline.md` and `fab-ff.md` are expected untouched, subject to the T001 grep | Intake § What Changes 3 enumerated the sites; T001 is the guard against under-coverage | S:85 R:90 A:90 D:85 |
| 6 | Certain | The Go guard is exercised by running the `src/go/fab-kit/cmd/fab/` package tests; no Go is edited so no `gofmt` step is needed | code-quality.md § Test Strategy; the guard lives in that package per the anti-pattern note | S:85 R:95 A:90 D:85 |

6 assumptions (4 certain, 2 confident, 0 tentative).
