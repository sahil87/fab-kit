# Spec: git-pr Shipped Sentinel and Status Commit

**Change**: 260222-trdc-git-pr-shipped-sentinel-and-status-commit
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/pipeline-orchestrator.md`

## git-pr Skill: Status Commit and Sentinel File

### Requirement: Second commit+push after recording PR URL to .status.yaml

After Step 4 (Record Shipped) writes the PR URL to `.status.yaml` via `stageman ship`, `git-pr` SHALL perform a second commit+push cycle:

1. Stage the `.status.yaml` file: `git add fab/changes/{name}/.status.yaml`
2. Commit with message: `git commit -m "Record shipped URL in .status.yaml"`
3. Push: `git push`

This ensures the branch tip includes the shipped metadata and the PR diff contains the complete, final state of the change folder.

#### Scenario: Status file recorded and pushed
- **GIVEN** the PR has been created and its URL written to `.status.yaml`
- **WHEN** the second commit+push cycle runs
- **THEN** `.status.yaml` changes are staged, committed, and pushed to the remote branch
- **AND** the PR's latest commit includes the `.status.yaml` update with the shipped URL

#### Scenario: Normal flow unchanged for manual git-pr use
- **GIVEN** a user runs `/git-pr` manually (not via orchestrator)
- **WHEN** git-pr completes all steps including the second commit+push
- **THEN** the PR branch is clean (no uncommitted changes) and ready for the next operation

### Requirement: Write `.shipped` sentinel file as final action

After all git operations are complete (including the second commit+push), `git-pr` SHALL write a sentinel file as its very last action:

1. Write sentinel: `echo "$PR_URL" > "$change_dir/.shipped"`

The sentinel file:
- Location: `fab/changes/{name}/.shipped`
- Content: the PR URL (same string written to `.status.yaml`)
- Gitignored: never committed to the repository
- Purpose: provides a race-free filesystem signal that all git operations are complete

#### Scenario: Sentinel file created after all git operations
- **GIVEN** all commits have been pushed to the remote branch
- **WHEN** the sentinel write executes
- **THEN** the sentinel file exists at `fab/changes/{name}/.shipped` containing the PR URL
- **AND** the file is not added to the git index (gitignored)

#### Scenario: Orchestrator polls sentinel to detect ship completion
- **GIVEN** `run.sh`'s polling loop is waiting for ship completion
- **WHEN** it checks for the existence of `fab/changes/{name}/.shipped`
- **THEN** file existence signals that all git operations are done
- **AND** the orchestrator can immediately proceed to branch the next dependent change

## .gitignore: Sentinel Pattern

### Requirement: Ignore sentinel files

The repository's `.gitignore` SHALL include patterns to exclude all `.shipped` sentinel files and temp files:

```
fab/changes/**/.shipped
```

This pattern ensures sentinel files never appear in git tracking or status output.

#### Scenario: Sentinel files are ignored
- **GIVEN** sentinel files exist in change folders
- **WHEN** running `git status`
- **THEN** no `.shipped` or `.shipped.tmp` files appear in the output

## Pipeline Orchestrator: Sentinel-Based Ship Detection

### Requirement: Poll for `.shipped` sentinel instead of `stageman is-shipped`

In `fab/.kit/scripts/pipeline/run.sh`, the `poll_change()` function's `shipping` state (currently line ~407) SHALL replace the `stageman is-shipped` check with a file-existence check:

Before:
```bash
if bash "$STAGEMAN" is-shipped "$status_file" 2>/dev/null; then
```

After:
```bash
local shipped_sentinel="$wt_path/fab/changes/$resolved_id/.shipped"
if [[ -f "$shipped_sentinel" ]]; then
```

This change makes the completion detection synchronous with the final git push, eliminating the TOCTOU race.

#### Scenario: run.sh detects sentinel file and marks change done
- **GIVEN** `/git-pr` has run and created the `.shipped` sentinel
- **WHEN** `poll_change()` checks for the file in the `shipping` state
- **THEN** the file exists and is readable
- **AND** `run.sh` marks the change `done` and proceeds to dispatch the next change

#### Scenario: Next change branches from clean parent tip
- **GIVEN** the first change's `.shipped` sentinel exists (all git ops complete)
- **WHEN** `dispatch.sh` creates a worktree for the dependent change
- **THEN** the worktree branches from the parent change's latest commit (which includes `.status.yaml` with shipped URL)
- **AND** no dirty state or uncommitted changes leak into the child worktree

## Design Decisions

### Use `.shipped` sentinel file instead of polling `.status.yaml`

**Decision**: Sentinel file provides the race-free signal. Git operations (`commit`, `push`) are fully atomic — once the file exists, all ops are done.

**Why**: Polling `.status.yaml` directly has a TOCTOU window: the file write happens before the second commit+push completes. A sentinel file created *after* the push eliminates this race entirely.

**Rejected**:
- Polling `.status.yaml` — TOCTOU race (file appears before push completes)
- Long fixed delays (20 seconds) — brittle, wasteful, unreliable
- `tmux wait-for` — adds tmux coupling to git-pr, complicates dispatch/run.sh interaction

### Sentinel file in `fab/changes/{name}/`, gitignored

**Decision**: Sentinel file lives inside the change folder, gitignored, never committed.

**Why**:
1. Change folder is the locus of change state (`.status.yaml`, `intake.md`, `spec.md` live there)
2. Gitignoring prevents noise in commits and PR diffs
3. Cleanup is automatic when worktree/change folder is deleted
4. No need for separate `fab/pipeline/` directory or orchestration-specific artifacts

**Rejected**: Separate `fab/pipeline/` directory — creates a new namespace for orchestration artifacts, but change folder is already the natural container for change-specific state.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Second commit+push in git-pr after recording shipped URL | Intake explicitly specified; eliminates dirty branch state | S:95 R:90 A:95 D:95 |
| 2 | Certain | Sentinel file at `fab/changes/{name}/.shipped` | User explicitly chose this location in discussion | S:95 R:95 A:95 D:95 |
| 3 | Certain | Sentinel is gitignored, unconditionally created | Discussed; matches change-folder-as-container pattern | S:95 R:95 A:95 D:95 |
| 4 | Certain | run.sh polls sentinel existence, not `.status.yaml` | Core design intent: eliminate TOCTOU race | S:95 R:90 A:95 D:95 |
| 5 | Confident | Sentinel content is the PR URL (not just touch) | Provides debugging value; atomic write via mv | S:75 R:90 A:80 D:85 |
| 6 | Confident | Commit message is "Record shipped URL in .status.yaml" | Low-stakes, easily changed if convention shifts | S:60 R:95 A:85 D:85 |
| 7 | Confident | `stageman is-shipped` command stays (backward compat) | Other consumers may use it; removal is separate cleanup | S:70 R:90 A:85 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved). Confidence score: 4.7/5.0 (high confidence, no unresolved items).
