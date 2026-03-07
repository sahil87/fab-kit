# Spec: Fix git-pr Ship Finish Ordering

**Change**: 260307-8ggm-git-pr-ship-finish-ordering
**Created**: 2026-03-07
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## git-pr: Ship Stage Step Ordering

### Requirement: All status mutations MUST occur before the commit boundary

The `/git-pr` skill's post-PR steps (Step 4 through Step 4d) SHALL execute all `.status.yaml` and `.history.jsonl` mutations before the `git commit && git push` operation. This ensures no uncommitted fab state files remain in the working tree after PR creation completes.

The new step ordering SHALL be:

1. **Step 4a** (Record PR URL): `fab status add-pr <change> <url>` — mutates `.status.yaml`
2. **Step 4b** (Finish ship stage): `fab status finish <change> ship git-pr` — mutates `.status.yaml` and `.history.jsonl`
3. **Step 4c** (Commit + push): `git add` both `.status.yaml` and `.history.jsonl`, then `git commit && git push`
4. **Step 4d** (Write sentinel): `echo "$PR_URL" > .pr-done` — gitignored, stays last

#### Scenario: Normal PR creation flow
- **GIVEN** a `/git-pr` run that successfully creates a PR
- **WHEN** Step 4a records the PR URL and Step 4b finishes the ship stage
- **THEN** both `.status.yaml` and `.history.jsonl` are staged in Step 4c
- **AND** the commit captures all status mutations atomically
- **AND** `git status` shows a clean working tree after push completes

#### Scenario: Ship finish is best-effort
- **GIVEN** Step 4b runs `fab status finish <change> ship git-pr 2>/dev/null || true`
- **WHEN** the finish command fails (e.g., stage already done, no active change)
- **THEN** Step 4c proceeds normally
- **AND** `git diff --cached --quiet` may show no changes if neither 4a nor 4b produced mutations
- **AND** the commit+push is skipped silently when there are no staged changes

#### Scenario: Already-shipped early exit
- **GIVEN** a `/git-pr` run where the PR already exists and no uncommitted/unpushed changes exist
- **WHEN** the skill executes Steps 4a–4d as part of the early-exit path
- **THEN** the same step ordering applies (4a → 4b → 4c → 4d)
- **AND** no uncommitted files remain

### Requirement: Step 4c MUST stage both `.status.yaml` and `.history.jsonl`

The commit step SHALL stage both `fab/changes/{name}/.status.yaml` and `.history.jsonl` (repo-root), since `fab status finish` auto-logs transitions to `.history.jsonl`. The previous implementation only staged `.status.yaml`.

#### Scenario: Both files committed
- **GIVEN** Step 4a and 4b have mutated `.status.yaml` and `.history.jsonl`
- **WHEN** Step 4c runs `git add`
- **THEN** both files are included in the staging area
- **AND** the commit message reflects the broader scope (e.g., "Update ship status and record PR URL")

#### Scenario: .history.jsonl has no changes
- **GIVEN** `.history.jsonl` does not exist or has no new entries
- **WHEN** Step 4c runs `git add .history.jsonl`
- **THEN** git silently ignores the no-op add
- **AND** Step 4c proceeds normally

### Requirement: Step renumbering for clarity

The post-PR steps SHALL be renumbered as 4a/4b/4c/4d (replacing the previous 4/4b/4c/4d numbering) to avoid confusion with the old ordering and to reflect the logical grouping.

#### Scenario: Spec file step references
- **GIVEN** `docs/specs/skills/SPEC-git-pr.md` documents the step flow
- **WHEN** this change is applied
- **THEN** the spec file reflects the new 4a/4b/4c/4d ordering with updated descriptions

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Reorder steps rather than add a second commit+push | Confirmed from intake #1 — consolidating into one commit is cleaner and captures all mutations atomically | S:95 R:90 A:95 D:95 |
| 2 | Certain | Stage `.history.jsonl` alongside `.status.yaml` | Confirmed from intake #2 — `fab status finish` writes to both files via auto-logging | S:90 R:90 A:90 D:95 |
| 3 | Certain | Keep `.pr-done` sentinel as the last step | Confirmed from intake #3 — gitignored, signals completion of all git operations | S:90 R:95 A:95 D:95 |
| 4 | Certain | `finish ship` is best-effort (failure silently ignored) | Confirmed from intake #4 — existing `2>/dev/null \|\| true` pattern preserved | S:85 R:90 A:90 D:90 |
| 5 | Confident | Renumber steps as 4a/4b/4c/4d | Confirmed from intake #5 — convention choice, low stakes | S:70 R:95 A:70 D:75 |
| 6 | Certain | No changes to `/fab-ff` or `/fab-fff` orchestration | Confirmed from intake #6 — they delegate to git-pr skill behavior | S:90 R:95 A:90 D:95 |
| 7 | Certain | Update `SPEC-git-pr.md` with new step ordering | Confirmed from intake #7 — constitution requires spec updates for skill changes | S:85 R:90 A:90 D:90 |
| 8 | Certain | Same bug class as 260222-trdc fix | Confirmed from intake #8 — same principle: all mutations before commit boundary | S:90 R:90 A:90 D:95 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
