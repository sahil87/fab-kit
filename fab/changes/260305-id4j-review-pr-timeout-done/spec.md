# Spec: Review-PR Timeout Treated as Done

**Change**: 260305-id4j-review-pr-timeout-done
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Execution Skills: PR Review Timeout and Polling

### Requirement: Copilot Polling Window

`/git-pr-review` Phase 3 SHALL poll `GET /reviews` for `copilot-pull-request-reviewer[bot]` every 30 seconds, with a maximum of **16 attempts** (8 minutes total).

The timeout message SHALL read: `Copilot review did not arrive within 8 minutes.`

#### Scenario: Copilot review arrives within window

- **GIVEN** Copilot was requested as a reviewer (Phase 2 succeeded)
- **WHEN** a review from `copilot-pull-request-reviewer[bot]` appears within 16 poll attempts
- **THEN** the review ID is captured and the skill proceeds to Step 3 (Path B)

#### Scenario: Copilot review does not arrive within window

- **GIVEN** Copilot was requested as a reviewer (Phase 2 succeeded)
- **WHEN** 16 poll attempts complete without a review from `copilot-pull-request-reviewer[bot]`
- **THEN** the skill prints `Copilot review did not arrive within 8 minutes.` and STOPs

### Requirement: Stage Update Routing on Timeout

When the Copilot review times out (Phase 3 exhausted), `/git-pr-review` Step 6 SHALL call `statusman.sh finish <change> review-pr` (marking the stage `done`), NOT `statusman.sh fail`.

The absence of external review feedback is a graceful completion — the internal `review` stage already passed.

#### Scenario: Copilot timeout results in done

- **GIVEN** an active change with `review-pr: active`
- **AND** Phase 3 polling timed out (no Copilot review arrived)
- **WHEN** Step 6 executes
- **THEN** `statusman.sh finish <change> review-pr git-pr-review` is called
- **AND** `.status.yaml` shows `review-pr: done`

#### Scenario: Copilot unavailable results in done (existing behavior, unchanged)

- **GIVEN** an active change with `review-pr: active`
- **AND** Phase 2 POST to `/requested_reviewers` returned non-2xx
- **WHEN** Step 6 executes
- **THEN** `statusman.sh finish <change> review-pr git-pr-review` is called
- **AND** `.status.yaml` shows `review-pr: done`

#### Scenario: No PR found remains a failure

- **GIVEN** an active change with `review-pr: active`
- **AND** `gh pr view` fails with "no pull requests found"
- **WHEN** Step 6 executes
- **THEN** `statusman.sh fail <change> review-pr git-pr-review` is called
- **AND** `.status.yaml` shows `review-pr: failed`

### Requirement: Step 6 Case Classification

Step 6 SHALL classify outcomes into exactly three cases:

1. **Success**: comments processed and pushed, or no actionable comments → `finish`
2. **Failure**: no PR found, or processing error → `fail`
3. **No reviews**: Copilot unavailable, Copilot timeout, or no reviews at all → `finish`

The previous classification placed Copilot timeout under case 2 (failure). This change moves it to case 3 (no reviews).

#### Scenario: All three cases map correctly

- **GIVEN** the Step 6 classification
- **WHEN** the outcome is "Copilot timeout"
- **THEN** it falls under case 3 (no reviews) and calls `finish`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only skill markdown and memory file change, no scripts | Confirmed from intake #1 — statusman commands are correct, only routing logic changes; poll count is prose in the skill file | S:90 R:95 A:95 D:95 |
| 2 | Certain | "No PR found" and "processing error" remain failure cases | Confirmed from intake #2 — these are actual errors distinct from "no review arrived" | S:85 R:90 A:90 D:90 |
| 3 | Certain | Copilot timeout follows same pattern as "Copilot unavailable" | Confirmed from intake #3 — both represent "no external review feedback" | S:90 R:95 A:85 D:95 |
| 4 | Certain | 16 polls at 30s = 8 minutes total | Arithmetic: 16 × 30s = 480s = 8 min. No change to poll interval, only the attempt count | S:95 R:95 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
