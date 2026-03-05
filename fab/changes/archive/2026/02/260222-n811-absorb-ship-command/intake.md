# Intake: Absorb Ship Command

**Change**: 260222-n811-absorb-ship-command
**Created**: 2026-02-22
**Status**: Draft

## Origin

> `[n811] Reduce external dependency - absorb changes:ship command into fabkit. Need suggestion for names`

Backlog item `n811`. User clarifications:
- **Name**: `/git-pr`
- **Invocation**: Pipeline orchestrator continues to use tmux send-keys — sends `/git-pr` instead of `/changes:ship pr`
- **Scope**: Narrower than `changes:ship` — focused single-purpose: commit + push + create GitHub PR
- **Behavior**: Fully autonomous — make all decisions, no prompts, get to PR

## Why

The pipeline orchestrator (`fab/.kit/scripts/pipeline/run.sh`) currently ships completed changes by sending `/changes:ship pr` to an interactive Claude session via `tmux send-keys`. This depends on `prompt-pantry`'s `changes` plugin — an external repo. Fab-kit's constitution (Principle V: Portability) requires that `fab/.kit/` work in any project via `cp -r`, so external plugin dependencies should be eliminated where possible.

The `changes:ship` command is also broader than what the pipeline needs. It handles repo detection, target selection (push/pr/merge), interactive prompts, and sub-agent delegation for each step. The pipeline only ever calls `/changes:ship pr` — it needs a focused command that autonomously gets uncommitted work to a GitHub PR with zero interaction.

## What Changes

### New skill: `/git-pr`

A fab-kit skill that autonomously commits, pushes, and creates a GitHub PR. No questions, no prompts — just execute.

**Behavior**:

1. **Gather state**:
   ```bash
   git status
   git branch --show-current
   git log --oneline -5          # for commit message style
   git diff --stat HEAD           # scope summary
   git log --oneline @{u}..HEAD 2>/dev/null  # unpushed commits
   gh pr view --json number,state,url 2>/dev/null  # existing PR
   ```

2. **Commit** (if uncommitted changes exist):
   - Stage all changes: `git add -A`
   - Generate commit message matching repo's existing style (from `git log`)
   - Commit — no "Co-Authored-By" line, no interactive editing

3. **Push** (if commits ahead of remote):
   - If no upstream: `git push -u origin <branch>`
   - If upstream exists: `git push`

4. **Create PR** (if no PR exists for this branch):
   - `gh pr create --fill` — auto-populate title/body from commits
   - If PR already exists: report its URL and skip

5. **Report result**: Show PR URL on success, error details on failure

**Key design points**:
- No repo detection — always operates on CWD
- No target selection — always goes to PR (the only use case)
- No merge support — out of scope
- No interactive prompts — fully autonomous
- Fails fast on errors — no retry logic

### Pipeline orchestrator update: `fab/.kit/scripts/pipeline/run.sh`

Replace `/changes:ship pr` with `/git-pr` in the tmux send-keys invocation (lines 365-374):

```bash
# Before
tmux send-keys -t "$pane_id" "/changes:ship pr"
sleep 0.5
tmux send-keys -t "$pane_id" Enter

# After
tmux send-keys -t "$pane_id" "/git-pr"
sleep 0.5
tmux send-keys -t "$pane_id" Enter
```

The rest of the pipeline mechanics (ship delay, PR polling via `gh pr view`, ship timeout) remain unchanged — they work the same regardless of which command is sent.

### Skill file location

The skill lives in fab-kit's skill directory and gets synced to `.claude/skills/` by the existing sync mechanism. File: `fab/.kit/skills/git-pr/git-pr.md` (or wherever the sync convention places standalone skills).

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update shipping command from `/changes:ship pr` to `/git-pr`
- `fab-workflow/kit-architecture`: (modify) Add `/git-pr` skill to skills inventory

## Impact

- **`fab/.kit/scripts/pipeline/run.sh`**: One-line change — replace command string in tmux send-keys
- **New skill file**: `/git-pr` skill definition
- **Dependencies**: `git` and `gh` CLI (both already required)
- **No breaking changes**: Pipeline manifest format, stage states, polling mechanics all unchanged
- **Removes dependency**: `prompt-pantry` `changes` plugin no longer needed for pipeline operation

## Open Questions

(None — all decisions resolved by user input.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Name is `/git-pr` | User explicitly specified | S:100 R:90 A:100 D:100 |
| 2 | Certain | Pipeline still uses tmux send-keys | User explicitly specified: "orchestrator invocations should still be via tmux only" | S:100 R:80 A:100 D:100 |
| 3 | Certain | No merge support | User scoped to "get to github PR" — merge is out of scope | S:90 R:90 A:90 D:95 |
| 4 | Certain | Fully autonomous, no prompts | User specified: "make all decisions" | S:100 R:85 A:100 D:100 |
| 5 | Confident | Use `gh pr create --fill` for PR creation | Standard gh CLI approach; `--fill` auto-populates from commits which suits autonomous operation | S:70 R:85 A:80 D:75 |
| 6 | Confident | Commit message derived from git log style + diff stat | No AI-generated messages needed; conventional format matching repo history is sufficient for autonomous operation | S:65 R:85 A:75 D:70 |
| 7 | Confident | No repo detection — operate on CWD | Fab-kit skills always run from project root; pipeline dispatches to worktree paths. Repo detection from `changes:ship` is unnecessary | S:70 R:90 A:90 D:85 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
