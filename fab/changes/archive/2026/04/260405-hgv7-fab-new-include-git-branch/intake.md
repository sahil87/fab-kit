# Intake: fab-new Include Git Branch

**Change**: 260405-hgv7-fab-new-include-git-branch
**Created**: 2026-04-05
**Status**: Draft

## Origin

> Expand fab-new scope to include git-branch step so that fab-new -> fab-fff becomes the complete workflow. Currently fab-new includes fab-draft and fab-switch behavior; we also need to add git-branch after activation. Also add note to constitution.md that src/kit/ is the canonical source for kit content and .claude/skills/ is gitignored.

Initiated conversationally. Key decisions from the session:

1. `fab-new` should create the git branch inline after activating the change — no separate `/git-branch` invocation needed when starting fresh work
2. The git-branch logic should follow the same branching rules as the standalone `git-branch` skill (rename if on a local-only branch, create-new if upstream exists, auto-create from main/master)
3. `src/kit/` is the canonical source for all kit content; `.claude/skills/` is a gitignored deployment artifact produced by `fab sync` — never edited directly
4. Direct edits (made earlier in the session) were reverted so this change can implement them properly via the pipeline

## Why

`/fab-new` was recently expanded to include activation (`fab change switch`) in Step 10, collapsing the old `/fab-new → /fab-switch` two-step into one. The logical next step is to include git-branch creation in the same operation, so that `/fab-new` takes a change all the way from "nothing" to "ready to code":

- change folder created
- intake generated
- change activated (`.fab-status.yaml` pointer set)
- git branch created and checked out

Without this, users who invoke `/fab-new` still need a separate `/git-branch` call before they can start coding. The `/fab-proceed` skill already works around this by dispatching `/git-branch` as a prefix step, but that workaround should become unnecessary once `/fab-new` handles it inline.

**Consequence of not fixing**: The `/fab-new → /fab-fff` sequence is not fully self-contained. Users either forget the branch step (and get a confusing "wrong branch" state) or rely on `/fab-proceed` to paper over the gap.

**Constitution clarification**: During the session, edits were accidentally made to `.claude/skills/` (which is gitignored) rather than `src/kit/` (the canonical source). The constitution does not currently make this distinction explicit. Adding a clear rule prevents the mistake from recurring.

## What Changes

### 1. `src/kit/skills/fab-new.md` — Add Step 11: Create Git Branch

After Step 10 (Activate Change), add a new Step 11 that creates the matching git branch inline. The step follows the same logic as the standalone `git-branch` skill:

```
1. Verify inside a git repo:
   git rev-parse --is-inside-work-tree >/dev/null 2>&1
   If not in a git repo: warn and skip — git is optional, change stays activated.

2. Branch name = {name} (the change folder name)

3. Get current branch: git branch --show-current

4. Check if branch already exists: git rev-parse --verify "{name}" >/dev/null 2>&1

5. Context-dependent action:
   - Already on target branch       → no-op, report "Branch: {name} (already active)"
   - Branch exists, not current     → git checkout "{name}", report "Branch: {name} (checked out)"
   - On main or master              → git checkout -b "{name}", report "Branch: {name} (created)"
   - On another branch, no upstream → git branch -m "{name}", report "Branch: {name} (renamed from {old})"
   - On another branch, has upstream→ git checkout -b "{name}", report "Branch: {name} (created, leaving {old} intact)"
```

Upstream check:
```bash
upstream=$(git config "branch.$(git branch --show-current).remote" 2>/dev/null || true)
```

The step is non-fatal: if git operations fail, surface the error and tell the user to run `/git-branch` manually. The change remains activated.

**Frontmatter changes**:
- `description`: update to `"Start a new change — creates the intake, activates it, and creates the git branch."`
- ~~`allowed-tools`: add `Bash(git:*)`~~ — *superseded: adding this would restrict all Bash calls in fab-new to git-only, breaking Steps 3–10. No `allowed-tools` change needed.*

**Output section**: Add `Branch: {name} (created|created, leaving {old_branch} intact|checked out|renamed from {old_branch}|already active)` line after `Activated: {name}`.

**Error handling table**: Add two rows:
- Not in a git repo (Step 11): Warn and skip branch creation — change is still activated
- `git checkout` / `git branch` failure (Step 11): Report git error; change remains activated — user can run `/git-branch` manually

### 2. `fab/project/constitution.md` — Canonical Source Clarification

Add one bullet to the **Additional Constraints** section (after the existing `src/kit/skills/*.md` rule):

```
- `src/kit/` is the canonical source for all kit content (skills, templates, migrations).
  `.claude/skills/` contains deployed copies produced by `fab sync` and is gitignored —
  never edit files there directly
```

This makes the distinction explicit and co-located with the existing `src/kit` constraint.

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update `/fab-new` requirements to reflect git-branch integration in Step 11, activation behavior, and updated output format. The existing memory text for `/fab-new` is stale in two ways: (a) it says "never activates changes" (activation was added in a prior change), (b) it doesn't mention git-branch step.

## Impact

- `src/kit/skills/fab-new.md` — primary file modified
- `fab/project/constitution.md` — one bullet added
- `docs/memory/fab-workflow/planning-skills.md` — hydrate update needed
- `docs/specs/skills/SPEC-fab-new.md` — spec file update needed (per constitution constraint)
- No Go binary changes; no config schema changes
- `/git-branch` skill remains unchanged — it still works as a standalone command and retains its idempotent behavior (if the branch is already active, it no-ops)

## Open Questions

- Should the `fab-proceed` skill remove its `git-branch` prefix-step dispatch now that `fab-new` handles it inline? (The dispatch is only triggered for new changes from conversation context; since `fab-new` now handles it, the dispatch in `fab-proceed` is redundant for that path.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Git-branch step goes after activation (Step 10), not before | Branch name derives from change folder name — must exist first; activation makes it the active change | S:95 R:90 A:95 D:95 |
| 2 | Certain | Branch name = change folder name (no prefix) | Consistent with `_naming.md` convention and `git-branch` skill behavior | S:95 R:90 A:95 D:95 |
| 3 | Certain | Step is non-fatal — git failure leaves change activated | User discussed this: change activation is the primary action; branch is convenience | S:90 R:85 A:90 D:95 |
| 4 | Certain | `src/kit/` is canonical; `.claude/` is gitignored deployment artifact | User stated explicitly twice, corrected direct edits to `.claude/` | S:100 R:95 A:100 D:100 |
| 5 | Certain | Constitution note goes in Additional Constraints, after existing `src/kit/skills` bullet | Co-location with existing `src/kit` rule makes it easy to find | S:90 R:95 A:95 D:90 |
| 6 | Confident | `fab-proceed` git-branch dispatch for the `fab-new` path should be reviewed for redundancy | Logical consequence — but the open question is whether to remove it now or in a follow-up change | S:70 R:75 A:70 D:65 |
| 7 | Tentative | `docs/specs/skills/SPEC-fab-new.md` needs updating (per constitution rule) | Constitution says skill changes MUST update corresponding SPEC file; file likely exists but content TBD | S:65 R:70 A:75 D:70 |

<!-- assumed: SPEC-fab-new.md update is in scope — constitution mandates it, but exact content will be determined at spec generation -->

7 assumptions (5 certain, 1 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
