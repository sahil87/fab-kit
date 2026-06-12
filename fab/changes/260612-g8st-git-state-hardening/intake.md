# Intake: Git-State Hardening for Autonomous Skills

**Change**: 260612-g8st-git-state-hardening
**Created**: 2026-06-12

## Origin

> g8st

One-shot `/fab-new g8st` (backlog ID). The `[g8st]` entry lives in the **main worktree's** `fab/backlog.md` (uncommitted as of 2026-06-12) — this worktree's copy predates it. It is batch 3/5 of the 2026-06-12 skills audit (253-agent audit, 175 verified findings). The full report is at `docs/specs/findings/skills-review-2026-06-12.md` **in the main worktree only** (also uncommitted); §2 Theme 3 is the section batched here. Line numbers cited below are vs commit `1431a9c3` (this branch's HEAD — verified accurate).

Backlog entry (verbatim):

> Skills-audit batch 3/5 — autonomous git skill state hardening. PARALLEL: wave 1 — safe alongside k4ge and c5tr (same-file-different-section touches only; coordinate the fab-archive exit-semantics seam with k4ge, which owns the Go/doc side of that contract). COLLIDES with w7dp on git-pr.md Step 0/guard block, git-pr-review.md failure paths, and fab-operator.md §6 — do NOT run concurrently with w7dp; w7dp branches after this merges. GOAL: the no-questions skills (git-pr, git-pr-review, git-branch, fab-new step 11, fab-archive, operator git ops) stop assuming a clean/attached/main-defaulted/local-only world. ACTIONS (report §2 Theme 3): detached-HEAD STOP before the autonomous commit path — git-pr.md:104-121/156-163 currently passes the branch guard then commits and emits a refspec-less push (must-fix). Split push-failure handling in git-pr-review.md:139-149/180 — "(no partial state)" is false (git reset cannot undo the commit) and the re-run path declares "No changes needed" / posts "Fixed" replies citing an unpushed SHA, permanently stranding fixes (must-fix): keep the commit, document recovery, add an unpushed-commit check to the re-run gate. Scope git add -A (git-pr.md:143-150) — it sweeps every untracked repo file into a pushed commit; stage known paths or guard with a stop-on-unexpected-untracked check. Branch has_pr on PR state (git-pr.md:70-127 — state/number are fetched and never read; a closed/merged PR short-circuits creation): branch OPEN/MERGED/CLOSED explicitly. git-branch.md:53-62/85-90: STOP with a disambiguation list on ambiguous multi-match instead of silently creating a junk standalone branch with a false message; check out remote-only branches with --track origin/<branch> instead of recreating divergent locals. Resolve the actual default branch instead of literal main/master (git-pr.md:106) — also fixes operator cherry-pick/rebase hardcoding origin/main with no fetch step (fab-operator.md:429-445/485/499, autopilot unusable on non-main-default repos). Dirty-tree caveat at change creation: fab-new.md:154-160 / git-branch.md:140 — uncommitted work silently rides into the new change's branch; warn or stash-prompt. fab-archive: archive/restore move tracked files + edit fab/backlog.md with no commit step (the "safe" claim contradicts the dirty tree it leaves); the archive-ok/backlog-mark-failed exit has no recovery path (re-run can never mark the backlog — make backlog marking idempotent on re-run). CONSTRAINTS: these are behavior changes to autonomous paths — flag each in the PR; SPEC mirror per touched skill. REPORT: docs/specs/findings/skills-review-2026-06-12.md §2 Theme 3.

## Why

1. **Pain point**: The no-questions-asked skills (`/git-pr`, `/git-pr-review`, `/git-branch`, `/fab-new` Step 11, `/fab-archive`, operator git ops) run autonomously — no prompts, no confirmation. Every one of them assumes a clean, attached, main-defaulted, local-only git world. When that assumption breaks, they don't fail — they **corrupt state silently**: a detached HEAD ships a refspec-less push; a rejected push posts "Fixed — {sha}" replies citing a commit that never reached the remote (permanently stranding the fix, since the re-run gate sees a clean tree and declares "No changes needed"); `git add -A` sweeps unrelated untracked files into a pushed commit; autopilot cherry-picks against a hardcoded, never-fetched `origin/main`.

2. **Consequence of inaction**: Two of these are audit must-fixes that strand work product irrecoverably (the git-pr-review push-failure path) or push garbage refs (detached-HEAD path). The operator autopilot is unusable on any repo whose default branch isn't `main`. As ff/fff/operator adoption grows, these paths run unattended more often — the blast radius compounds.

3. **Why this approach**: These are audit-verified findings (each survived adversarial verification; line refs reproduced against HEAD during this intake). The audit's Theme 3 batches them as one coherent change because they share a single design stance — *autonomous git paths must verify state before mutating and report failures honestly* — and shared file sections (the git-pr guard block, the git-branch/fab-new branch table). Fixing them piecemeal would re-edit the same w7dp-colliding sections repeatedly.

## What Changes

All edits target canonical sources in `src/kit/skills/` (never `.claude/skills/` deployed copies), each with its `docs/specs/skills/SPEC-*.md` mirror updated in the same PR. Items 1–2 are audit must-fixes.

### 1. git-pr: detached-HEAD STOP before the autonomous commit path (must-fix)

**Current** (`src/kit/skills/git-pr.md:104-121, 156-163`): Step 2's branch guard only checks for literal `main`/`master`. On a detached HEAD, `git branch --show-current` prints an empty string → the guard passes → Step 3a commits → Step 3b finds no upstream and runs `git push -u origin $(git branch --show-current)`, which expands to a refspec-less `git push -u origin`.

**Target**: Detect detached HEAD in Step 1 (empty `git branch --show-current`, or `git symbolic-ref -q HEAD` exit 1). STOP before any commit/push:

```
Cannot ship from a detached HEAD — check out a branch first (run /git-branch).
```

Placed ahead of the Step 2 branch guard. This edits the w7dp-colliding guard block — g8st owns this edit; w7dp branches after g8st merges.

### 2. git-pr-review: split push-failure handling + unpushed-commit re-run gate (must-fix)

**Current** (`src/kit/skills/git-pr-review.md:139-149, 180`): Step 5.7 says "If commit or push fails → run `git reset` to clear any staged changes, then print the error and STOP (no partial state)". The "(no partial state)" claim is **false** for push failures: `git reset` cannot undo a commit that already succeeded. On re-run, Step 5.1 sees a clean tree → prints "No changes needed" → proceeds to Step 5.5 and posts "Fixed — {sha}" replies citing the **unpushed** SHA. The fix is permanently stranded.

**Target** — split the two failure modes:

- **Commit fails**: `git reset` to clear staged changes, print error, STOP. ("No partial state" is true here.)
- **Push fails**: **keep the commit.** Print the push error plus documented recovery (e.g., `git pull --rebase && git push`, then re-run `/git-pr-review`), and STOP **without posting replies** — no "Fixed" reply may cite an unpushed SHA.
- **Re-run gate**: before Step 5.2 can declare "No changes needed", check for unpushed commits (`git log --oneline @{u}..HEAD`, treating no-upstream as unpushed). If unpushed commits exist, push them first, then proceed to Step 5.5 replies with the now-pushed SHA.
- Update Step 6's exception list (`:180` — "Step 5 … leaving no partial state") to match the split semantics.

### 3. git-pr: scope the autonomous `git add -A`

**Current** (`src/kit/skills/git-pr.md:143-150`): Step 3a runs `git add -A`, sweeping every untracked file in the repo — scratch files, logs, unrelated work — into an autonomously pushed commit.

**Target**: stage tracked changes (`git add -u`) unconditionally; enumerate untracked files via `git status --porcelain` and apply the **expected-area guard**: include untracked files that fall inside the project's expected write areas (`source_paths` entries, `docs/`, `fab/` — derived from `config.yaml`), STOP with the file list when any untracked file falls outside them. Rejected alternatives: hard stop-on-any-untracked (blocks every change that legitimately creates files); keep-`add -A`-and-disclose (the sweep finding would remain live).
<!-- clarified: expected-area guard confirmed — git add -u + untracked included only inside source_paths/docs/fab, STOP listing unexpected files outside those areas -->

### 4. git-pr: branch `has_pr` on PR state (OPEN/MERGED/CLOSED)

**Current** (`src/kit/skills/git-pr.md:70-127`): Step 1 fetches `gh pr view --json number,state,url` but only ever reads existence — `state` and `number` are fetched and never read. A closed or merged PR makes `has_pr` true, so Step 3 short-circuits with "already shipped" and new commits never get a PR.

**Target**: branch explicitly on `state`:

- **OPEN** → current behavior (short-circuit "already shipped" when nothing else to do).
- **CLOSED** → proceed to Step 3c and create a fresh PR (`gh pr create` works after a closed PR — shipping intent is explicit: the user/orchestrator just invoked `/git-pr`).
- **MERGED** → STOP with guidance: the branch's PR already merged; new work needs a new change/branch.

<!-- clarified: per-state policy confirmed — OPEN short-circuit / CLOSED fresh PR / MERGED stop; reopen-on-CLOSED and stop-on-both alternatives rejected -->

### 5. git-branch: multi-match disambiguation STOP + `--track` for remote-only branches

**Current** (`src/kit/skills/git-branch.md:53-62, 85-90`): when an explicit argument fails `fab change resolve`, the skill unconditionally enters standalone fallback, printing `No matching change found — using standalone branch '{name}'` and creating a junk branch. For an **ambiguous multi-match** that message is false — matches exist, the resolution was just ambiguous. Separately, Step 4 only checks local existence (`git rev-parse --verify "{branch_name}"`); a branch existing only on `origin` gets recreated as a divergent local via `checkout -b`.

**Target**:

- Distinguish the two resolve failures by stderr (verified against the binary, both exit 1): `ERROR: Multiple changes match "x": <candidate list>.` vs `ERROR: No change matches "x".` On multi-match → STOP and show the candidate list (no branch created). On true no-match → existing standalone fallback. No Go change needed.
- In Step 4, before creating: check `git rev-parse --verify "origin/{branch_name}"`; if the branch exists remote-only → `git checkout --track "origin/{branch_name}"` instead of `git checkout -b`.

### 6. Default-branch resolution: git-pr guard + operator cherry-pick/rebase + fetch step

**Current**: `src/kit/skills/git-pr.md:106` guards on literal `main`/`master`. `src/kit/skills/fab-operator.md:429-445, 485, 499` hardcodes `origin/main` in the dependency cherry-pick range (`git cherry-pick --no-commit origin/main..<dep-branch>`), the "Why `origin/main` as base" rationale, and the `--merge-on-complete` rebase — with **no fetch step**, so even on main-defaulted repos the base can be stale. Autopilot is unusable on non-main-default repos.

**Target**: resolve the actual default branch — `git symbolic-ref --short refs/remotes/origin/HEAD` (strip `origin/`), falling back to `gh repo view --json defaultBranchRef` then literal `main`/`master` when both fail. Use the resolved name in git-pr's Step 2 guard (and Step 1b nudge) and in all three operator sites; add `git fetch origin` before the operator's cherry-pick/rebase sequences. The operator §6-adjacent edits are w7dp-colliding — g8st owns them now.

### 7. Dirty-tree warning at change creation (fab-new Step 11 / git-branch)

**Current** (`src/kit/skills/fab-new.md:154-160` Step 11 branch table / `git-branch.md:140` caveat): the only caveat covers **committed** work ("the `checkout -b` fallback inherits the old change's HEAD — unpushed commits carry over"). Uncommitted changes silently ride into the new change's branch on `checkout -b` / `branch -m` with no mention.

**Target**: before the branch operation, check `git status --porcelain`; if dirty, append a non-blocking warning to the report line, e.g.:

```
Branch: {name} (created) — note: {N} uncommitted change(s) carried over from {old_branch}
```

Warn, not stash-prompt: these paths run inside no-questions/orchestrated flows (ff/fff/operator), where a blocking prompt would violate the autonomy posture. Both files carry an explicit keep-in-sync comment — the same edit lands in both tables.

### 8. fab-archive: honest dirty-tree claim + backlog-mark recovery on re-run

**Current**: `fab-archive.md` claims "safe to re-run" while `fab change archive` moves tracked files and edits `fab/backlog.md` with **no commit step** — every archive leaves a dirty tree the claim doesn't disclose. Separately (verified in `src/go/fab/internal/archive/archive.go:129-140`): `ArchiveWithBacklog` returns early when `Archive` yields `ErrAlreadyArchived`, **before** ever reaching `backlog.MarkDone` — so after an archive-ok/backlog-mark-failed exit, no re-run can ever mark the backlog.

**Target**:

- **Doc side (g8st)**: qualify the "safe to re-run" claim in `fab-archive.md` (and its SPEC mirror) with explicit dirty-tree disclosure — archive/restore leave uncommitted moves + backlog edits for the caller to commit. Do **not** add an autonomous commit step (that would change blast radius and overlap `/git-pr`'s commit ownership).
- **Go side (g8st, coordinated with k4ge)**: on `ErrAlreadyArchived`, still attempt `backlog.MarkDone` (already idempotent — returns `already` when marked) so a re-run recovers the failed mark. Small change + unit test + `_cli-fab.md` row update per constitution. **Seam**: k4ge owns archive exit-code semantics and the adjacent defect that makes the soft skip CLI-unreachable today (archived changes fail resolution with "No change matches" exit 1) — g8st's fix is unit-testable now but only CLI-observable after k4ge lands. Do not touch exit codes here.
<!-- clarified: Go fix lands in g8st — on ErrAlreadyArchived still attempt backlog.MarkDone (unit-tested, _cli-fab.md row updated), exit-code semantics untouched (k4ge's seam); defer-to-k4ge rejected -->

## Affected Memory

- `pipeline/execution-skills`: (modify) `/git-pr` ship pipeline (detached-HEAD stop, add -A scoping, PR-state branching, default-branch guard), `/git-pr-review` failure semantics, `/fab-archive` safety claims
- `pipeline/planning-skills`: (modify) fab-new Step 11 branch table gains the dirty-tree warning
- `pipeline/change-lifecycle`: (modify) git integration — branch creation caveats, default-branch resolution convention
- `runtime/operator`: (modify) cherry-pick/rebase base resolution + fetch step in dependency resolution and `--merge-on-complete`

## Impact

- **Skills (canonical sources)**: `src/kit/skills/git-pr.md`, `git-pr-review.md`, `git-branch.md`, `fab-new.md` (Step 11 table only), `fab-archive.md`, `fab-operator.md` (git-ops sections)
- **SPEC mirrors (constitution-mandated, same PR)**: `docs/specs/skills/SPEC-git-pr.md`, `SPEC-git-pr-review.md`, `SPEC-git-branch.md`, `SPEC-fab-new.md`, `SPEC-fab-archive.md`, `SPEC-fab-operator.md`
- **Go (item 8 only)**: `src/go/fab/internal/archive/archive.go` + `archive_test.go`, plus the matching `src/kit/skills/_cli-fab.md` row (constitution: Go changes need tests + same-PR `_cli-fab.md` updates)
- **Coordination**: COLLIDES with w7dp on git-pr.md Step 0/guard block, git-pr-review.md failure paths, fab-operator.md §6 — w7dp must branch after this merges. Wave-1 parallel-safe with k4ge and c5tr (same-file-different-section; second-to-merge rebases). fab-archive exit-semantics seam coordinated with k4ge per item 8.
- **PR requirement**: every item changes autonomous-path behavior — the PR description must flag each behavior change explicitly (backlog CONSTRAINTS).

## Open Questions

None — the backlog entry fully specifies scope; the three design choices it left open were resolved in the 2026-06-12 clarification session (see `## Clarifications`).

## Clarifications

### Session 2026-06-12

| Q | Question | Answer |
|---|----------|--------|
| 1 | `git add -A` replacement mechanism in `/git-pr` Step 3a | Expected-area guard — `git add -u` + untracked files included only inside `source_paths`/`docs/`/`fab/`; unexpected untracked elsewhere → STOP with file list |
| 2 | Per-state policy when `/git-pr` finds an existing PR | OPEN → short-circuit; CLOSED → create fresh PR; MERGED → STOP with new-change/branch guidance |
| 3 | Where the fab-archive backlog-mark recovery fix lands | Go-side in g8st: on `ErrAlreadyArchived` still attempt `backlog.MarkDone`; exit codes untouched (k4ge seam) |

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 10 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the 8 ACTIONS in the `[g8st]` backlog entry — no other Theme 3 or audit findings pulled in | Backlog entry is exhaustive and audit-curated; w7dp/c5tr own the adjacent themes | S:95 R:85 A:90 D:90 |
| 2 | Certain | g8st edits the w7dp-colliding sections now (git-pr guard block, git-pr-review failure paths, operator §6); w7dp branches after merge | Backlog states the collision and ordering explicitly | S:90 R:80 A:90 D:90 |
| 3 | Certain | Every touched skill gets its SPEC mirror updated same-PR; src/kit is canonical (never edit `.claude/skills/`); each behavior change flagged in the PR | Constitution constraints + backlog CONSTRAINTS verbatim | S:95 R:90 A:100 D:95 |
| 4 | Certain | Default branch resolved via `git symbolic-ref --short refs/remotes/origin/HEAD`, fallback `gh repo view --json defaultBranchRef`, then literal main/master; operator adds `git fetch origin` before cherry-pick/rebase | Clarified — user confirmed | S:95 R:80 A:80 D:65 |
| 5 | Certain | Dirty-tree handling = non-blocking warn (not stash-prompt) at fab-new Step 11 / git-branch | Clarified — user confirmed | S:95 R:75 A:70 D:55 |
| 6 | Certain | git-branch disambiguation branches on `fab change resolve` stderr text (Multiple vs No match) — doc-only, no Go change | Clarified — user confirmed | S:95 R:80 A:85 D:70 |
| 7 | Certain | `git add -A` replacement: `git add -u` + expected-area guard — untracked included inside `source_paths`/`docs/`/`fab/`, STOP with file list outside | Clarified — user confirmed expected-area guard | S:95 R:70 A:50 D:45 |
| 8 | Certain | PR-state policy: OPEN → short-circuit, CLOSED → create fresh PR, MERGED → STOP with new-branch guidance | Clarified — user confirmed recommended policy | S:95 R:75 A:55 D:40 |
| 9 | Certain | Backlog-mark re-attempt on `ErrAlreadyArchived` lands Go-side within g8st; exit-code semantics untouched (k4ge's seam) | Clarified — user confirmed Go fix in g8st | S:95 R:55 A:60 D:50 |
| 10 | Certain | fab-archive dirty-tree finding resolved by honest disclosure, not an autonomous commit step | Clarified — user confirmed | S:95 R:75 A:70 D:60 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).
