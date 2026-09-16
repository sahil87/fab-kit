# Plan: Fetch the Base Before Spawning a Worktree, Rebase onto the Base at Ship

**Change**: 260916-bq00-fetch-base-spawn-rebase-ship
**Intake**: `intake.md`

## Requirements

### Operator: Spawn-Time Base Freshness

#### R1: Spawn step 3 fetches and pins the new branch to the fetched default-branch tip
Before running any **new-branch** `wt create` form, the operator MUST run `git fetch origin` in the target repo's main-worktree root, resolve the default branch via the chain Dependency Resolution step 0 owns (pointer, not restatement), and pass `--base "$(git rev-parse origin/{default_branch})"` — the **commit SHA** — to `wt create`. The rule MUST state that the SHA, never the ref name, is passed, and why (a ref start-point sets upstream tracking, which breaks `/fab-new` Step 11 row 5's rename guard and confuses `push.default=simple`). The `--checkout <branch>` form is untouched.

- **GIVEN** an operator spawn whose change branch does not yet exist (positional form) or a bare/no-`fab/` spawn (no branch argument)
- **WHEN** §6 step 3 runs
- **THEN** `git fetch origin` runs first in the target repo's main-worktree root, and `wt create` receives `--base <sha>` where `<sha>` is the fetched `origin/{default_branch}` commit
- **AND** the resulting branch has no upstream set (`git config branch.<name>.remote` empty)

#### R2: `_cli-external.md` § wt shows the fetch + `--base <sha>` forms
The Operator Spawning Rules' two new-branch `wt create` example lines (positional, and the no-branch-argument sentence) MUST carry `--base "$base_sha"` with a preceding fetch + rev-parse, and MUST NOT restate the default-branch chain (point at `fab-operator.md` § Dependency Resolution step 0).

- **GIVEN** the routing rule in `_cli-external.md` § wt → Operator Spawning Rules
- **WHEN** an agent composes a new-branch `wt create`
- **THEN** the example it copies includes the fetch and `--base "$base_sha"`, and the `--checkout` example is unchanged

#### R3: `merge-auto` prose describes the spawn-time mechanism, not a pre-spawn rebase
The three `merge-auto` sites in `fab-operator.md` § Queues (the mode bullet, the "steps 5–8" paragraph, and the arming item's `then` description) MUST drop "rebase the next change onto `origin/{default_branch}`" wording and MUST keep the defer-spawn-until-merge-verified rule; the "starts from the advanced line" outcome is delivered by step 3's fetch + `--base <sha>`.

- **GIVEN** a `merge-auto` queue whose item n's PR merge was just verified on a tick
- **WHEN** the operator reads § Queues for what to do next
- **THEN** it spawns item n+1 through §6 (whose step 3 fetches and pins the base) and is not told to rebase a branch that does not exist yet

### Ship: Rebase onto the Base Before Push

#### R4: `/git-pr` rebases onto `origin/<base_branch>` immediately before 3b Push
`/git-pr` Step 3 MUST gain a sub-step between 3a-bis and 3b that runs `git fetch origin` and, when `refs/remotes/origin/$base_branch` resolves, `git rebase origin/$base_branch`; when the ref does not resolve it MUST skip with a one-line warning naming `$base_branch` and continue. It applies whenever `$base_branch` is resolved (Step 1 always resolves it), not gated on `{has_fab}`. It runs only on paths that reach 3b (i.e. not on the "already shipped" / MERGED-STOP paths).

- **GIVEN** a ship whose branch is behind `origin/<base_branch>`
- **WHEN** Step 3 reaches the new sub-step
- **THEN** the branch is rebased onto the fetched base before anything is pushed, and the PR diff is against the current base
- **GIVEN** `origin/<base_branch>` does not resolve after the fetch
- **WHEN** the sub-step runs
- **THEN** it prints one warning line and continues to 3b without rebasing

#### R5: Conflict policy for the ship rebase
On a rebase conflict: generated `docs/memory/**/index.md` / `log.md` conflicts follow the existing FKF §5 rule (resolve topic files, re-run `fab docs-index docs/memory`, take output wholesale, `git rebase --continue`); any other conflict is resolved by the ship agent only when the resolution is clear from the change's own intent; otherwise `git rebase --abort`, report the conflicting file list, and STOP the ship stage (nothing pushed, no PR created). The pointer to the 3a-bis never-hand-merge rule is a pointer, not a restatement.

- **GIVEN** a conflict only in `docs/memory/pipeline/log.md`
- **WHEN** the rebase stops
- **THEN** the agent regenerates via `fab docs-index docs/memory`, stages, continues, and ships
- **GIVEN** a conflict in `src/kit/skills/x.md` whose resolution is not clear
- **WHEN** the rebase stops
- **THEN** `git rebase --abort` runs, the file list is reported, and the skill STOPs with the ship stage left `active`

#### R6: Push after a rebase uses `--force-with-lease` when an upstream exists
3b MUST push with `git push --force-with-lease` when an upstream exists and the rebase sub-step ran (a rebase rewrites history); the no-upstream first push (`git push -u origin <branch>`) is unchanged. Step 4c's status push stays a plain `git push` (it follows 3b on the same rebased branch).

- **GIVEN** a re-ship on a branch with an upstream, after the rebase moved HEAD
- **WHEN** 3b runs
- **THEN** the push is `git push --force-with-lease` and succeeds; a plain `git push` would have been rejected

### Docs: Memory, Specs, Site Sweep

#### R7: fab-kit docs describe the new behavior and the g8st decision is extended
`docs/memory/runtime/operator.md` (spawn sequence step, § Queues `merge-auto`, the g8st Design Decision with an *Updated by* trailer), `docs/memory/pipeline/execution-skills.md` (`/git-pr` ship behavior), `docs/memory/pipeline/change-lifecycle.md` § Git Integration, `docs/specs/skills.md` (`/git-pr` Behavior list + Flow, `/fab-operator` Tools line if it names `wt create`), and `docs/site/merge-topologies.md` (`merge-auto` mechanism wording) MUST reflect Fix 1–3 with no stale "rebase the next change" claim left anywhere in the repo (excluding archived changes and generated log files). Memory `index.md`/`log.md` are regenerated by `fab docs-index docs/memory`, never hand-edited.

- **GIVEN** `grep -rn "rebase the next change\|fetches, rebases" src docs --include='*.md'` after apply
- **WHEN** archived changes and `docs/specs/findings/` are excluded
- **THEN** no hit remains

### Non-Goals

- `/fab-new` Step 11 and `/git-branch` Step 4 gain no fetch — ship's rebase reconciles those flows (intake assumption 10)
- `/git-pr-review`'s re-push gains no rebase (intake assumption 11)
- No Go/CLI change, no `_cli-fab*.md` reference update, no migration, no `wt` change
- `fab batch new`'s `wt create --worktree-name` launcher in `_cli-fab.md` (Go-owned, already a recorded follow-up) is not touched

### Design Decisions

#### Spawn Base Is the Fetched SHA, Not the Ref
**Decision**: §6 step 3 passes `--base "$(git rev-parse origin/{default_branch})"` after `git fetch origin`, never `--base origin/{default_branch}`.
**Why**: A remote-tracking start-point makes `git branch`/`git worktree add -b` set upstream tracking (verified git 2.53.0: `branch.<name>.remote=origin`), which flips `/fab-new` Step 11 row 5's rename guard (`upstream` must be empty) to row 6 and leaves the worktree on the wrong branch; a SHA start-point sets no upstream.
**Rejected**: `--base origin/{default_branch}` + `git branch --unset-upstream` afterwards (two steps, and the window between them is exactly where `/fab-new` runs in a raw-text spawn); `git pull --ff-only` in the main checkout (the checkout may be on another branch or dirty).
*Introduced by*: 260916-bq00-fetch-base-spawn-rebase-ship

#### Ship Rebases onto the Recorded Base, and a Conflict Stops Ship
**Decision**: `/git-pr` fetches and rebases onto `origin/$base_branch` right before 3b; unclear conflicts abort and STOP the ship stage.
**Why**: Ship is the last moment the change's author context is present; a conflict there is recoverable by the agent that wrote the code, whereas the same conflict at merge-all lands on the operator with no context. Using the recorded base keeps `stacked-prs` dependents on their dependency branch.
**Rejected**: Rebase only at spawn (misses long-lived changes and non-operator flows); skip the rebase when an OPEN PR exists (a re-ship is exactly when the base has drifted most); recording a commit SHA in `base_branch` (inverts vy27's PR-target semantics and needs Go + migration).
*Introduced by*: 260916-bq00-fetch-base-spawn-rebase-ship

## Tasks

### Phase 2: Core Implementation

- [x] T001 Edit `src/kit/skills/fab-operator.md`: §6 spawn step 3 gains the pre-`wt create` fetch + `base_sha` rule (SHA-not-ref, pointer to Dependency Resolution step 0 for the chain, `--checkout` untouched); the bare-agent sentence names `--base "$base_sha"`; § Queues `merge-auto` bullet, the "steps 5–8" paragraph, and the arming `then` wording drop the pre-spawn rebase <!-- R1, R3 -->
- [x] T002 [P] Edit `src/kit/skills/_cli-external.md` § wt → Operator Spawning Rules: fetch + `base_sha` preamble, `--base "$base_sha"` on the positional and no-branch forms, `--checkout` form unchanged, pointer to the chain owner <!-- R2 -->
- [x] T003 [P] Edit `src/kit/skills/git-pr.md`: new `#### 3a-ter. Rebase onto Base` sub-step (fetch, ref check + warning skip, rebase, conflict policy pointing at 3a-bis's never-hand-merge rule), 3b's upstream branch pushes `--force-with-lease` after a rebase, Key Properties "Modifies git state" row names rebase <!-- R4, R5, R6 -->

### Phase 4: Polish

- [x] T004 Sweep memory: `docs/memory/runtime/operator.md` (spawn step 2 sentence, § Queues `merge-auto` paragraph, g8st DD body + `*Updated by*: 260916-bq00-…`), `docs/memory/pipeline/execution-skills.md` (`/git-pr` ship paragraph gains the rebase sub-step, conflict policy, force-with-lease), `docs/memory/pipeline/change-lifecycle.md` § Git Integration default-branch paragraph; then `fab docs-index docs/memory` <!-- R7 -->
- [x] T005 [P] Sweep specs/site: `docs/specs/skills.md` `/git-pr` Behavior list + Flow skeleton; `docs/site/merge-topologies.md` `merge-auto` prose; repo-wide grep for `rebase the next change` / `fetches, rebases` / `rebase the next change onto` with zero live hits <!-- R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §6 step 3 instructs `git fetch origin` in the target repo's main-worktree root and `--base "$base_sha"` on both new-branch `wt create` forms, states the SHA-not-ref rule with its rename-guard rationale, and leaves `--checkout` untouched
- [x] A-002 R2: `_cli-external.md` § wt's positional and no-branch `wt create` examples carry `--base "$base_sha"` preceded by the fetch + rev-parse; the `--checkout` example is unchanged
- [x] A-003 R3: all three `merge-auto` sites in `fab-operator.md` § Queues describe the spawn-time mechanism and keep the defer-until-verified rule
- [x] A-004 R4: `git-pr.md` has a rebase sub-step between 3a-bis and 3b with fetch, ref-resolves check, warning-skip branch, and `git rebase origin/$base_branch`
- [x] A-005 R5: the sub-step's conflict policy has the three rows (generated index/log → pointer to 3a-bis rule; clear → resolve; unclear → abort + report + STOP)
- [x] A-006 R6: 3b pushes `--force-with-lease` on the upstream-exists branch after a rebase; the no-upstream push is unchanged
- [x] A-007 R7: memory, specs, and site files listed in T004/T005 reflect Fix 1–3; g8st DD carries `*Updated by*: 260916-bq00-fetch-base-spawn-rebase-ship`

### Behavioral Correctness

- [x] A-008 R1: the skill text names the commit SHA (`git rev-parse origin/{default_branch}`) as the `--base` value and forbids the ref name
- [x] A-009 R4: the rebase is not gated on `{has_fab}` and does not run on the "already shipped" / MERGED-STOP paths

### Scenario Coverage

- [x] A-010 R7: `grep -rn "rebase the next change\|fetches, rebases" src docs --include='*.md'` excluding `fab/changes/archive` and `docs/specs/findings` returns nothing

### Edge Cases & Error Handling

- [x] A-011 R4: an unresolvable `origin/$base_branch` after fetch produces a one-line warning and continues to 3b
- [x] A-012 R5: the unclear-conflict path runs `git rebase --abort` before STOP, leaving nothing pushed

### Code Quality

- [x] A-013 Pattern consistency: new skill prose follows the surrounding sub-step / numbered-step conventions (headings `#### 3x.`, `Print:` lines, STOP blocks)
- [x] A-014 No unnecessary duplication: the default-branch chain is not restated at any new site (owner-or-pointer); the never-hand-merge rule is pointed at, not copied
- [x] A-015 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-016 Constitution V: no new `src/kit/**` text cites `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` as authority (Go guard `go test ./src/go/fab-kit/cmd/fab/...` passes)
- [x] A-017 Sibling sweep: every `merge-auto` restatement (skill, memory, site) updated together

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change edits prose in place (the `merge-auto` pre-spawn rebase wording was replaced, not left orphaned); no existing file, symbol, or block became redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The new git-pr sub-step is named `3a-ter. Rebase onto Base` and sits between 3a-bis and 3b | Intake fixes the position; naming follows the existing `3a` / `3a-bis` lettering scheme | S:80 R:90 A:90 D:85 |
| 2 | Confident | `--force-with-lease` is used only when the rebase sub-step actually ran (not on every upstream push) | Minimal blast radius: a plain push stays plain when nothing was rewritten; a rebase that was skipped (unresolvable base) leaves history intact | S:70 R:85 A:80 D:75 |
| 3 | Confident | Step 4c's status push stays a plain `git push` | It runs after 3b on the same branch, so the remote already matches the rebased history | S:70 R:90 A:85 D:80 |
| 4 | Confident | `wt create --reuse` respawns also carry `--base "$base_sha"` (harmless — the existing worktree is reused) | wt documents no `--reuse` × `--base` conflict; a spurious flag on a reused worktree is inert (intake assumption 15) | S:50 R:90 A:60 D:65 |

4 assumptions (1 certain, 3 confident, 0 tentative).
