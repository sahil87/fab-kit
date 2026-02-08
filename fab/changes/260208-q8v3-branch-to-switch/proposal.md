# Proposal: Move Branch Integration from fab-new to fab-switch

**Change**: 260208-q8v3-branch-to-switch
**Created**: 2026-02-08
**Status**: Draft

## Why

Branch integration currently lives in `/fab-new` (Step 4), creating a dichotomy: `/fab-new` handles both artifact creation and git context switching, while `/fab-discuss` creates artifacts but has no path to branch integration. The user must manually manage branches after `/fab-discuss`. Meanwhile, the `branch:` field in `.status.yaml` is purely ceremonial — no skill uses it for logic, and it goes stale if the user manually switches branches.

By consolidating branch handling into `/fab-switch` (the "I'm committing to work on this" moment), both `/fab-new` and `/fab-discuss` get consistent branch support through a shared path, and `/fab-new` gets simpler.

## What Changes

- **Remove `branch:` field from `.status.yaml`**: No skill consumes it for logic. `fab-status` can use `git branch --show-current` instead, which is always accurate. Remove from the template, preflight script, status script, and all skill references.
- **Remove Step 4 (Git Integration) from `/fab-new`**: `/fab-new` no longer creates, adopts, or prompts about branches. It creates the folder, generates the proposal, and calls `/fab-switch` internally to set the change active.
- **Remove `--branch` flag from `/fab-new`**: Moves to `/fab-switch --branch <name>`.
- **Remove `branch_prefix` from `config.yaml`**: No longer needed (or move to `/fab-switch` context if branch naming conventions are desired).
- **Add branch integration to `/fab-switch`**: After writing `fab/current`, if `git.enabled` is true and the project is a git repo, `/fab-switch` offers the same branch options currently in `/fab-new`:
  - If on `main`/`master` → auto-create branch named after the change
  - If on a feature branch → ask: Adopt / Create new / Skip
  - `wt/*` branches default to "Create new branch"
  - `--branch <name>` flag for explicit branch name (skip prompt)
- **Add `--branch` flag to `/fab-switch`**: Takes an explicit branch name. Creates if new, checks out if existing.
- **`/fab-new` calls `/fab-switch` internally**: After generating the proposal, `/fab-new` invokes the switch flow to activate the change (including branch integration). From the user's perspective, `/fab-new` still results in an active change with a branch — the internal delegation is transparent.
- **Update `/fab-discuss` next steps**: After creating a new change, the user runs `/fab-switch {name}` which now handles branch creation too. No separate manual branch step needed.
- **Update `fab-status`**: Replace `branch:` field display with `git branch --show-current` for live branch info. Show branch name only when inside a git repo with `git.enabled`.
- **Update `fab-preflight.sh`**: Remove `branch` from YAML output.

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/planning-skills.md`: Remove branch integration from `/fab-new`, add note about internal `/fab-switch` call, remove `--branch` flag
- `fab-workflow/change-lifecycle.md`: Remove `branch:` from `.status.yaml` schema, update `/fab-switch` to include branch integration, remove `branch_prefix` from config references

### Removed Docs
(none)

## Impact

### Modified files
- `fab/.kit/skills/fab-switch.md` — Add branch integration step, `--branch` flag, update context loading (needs `config.yaml` for `git.enabled`)
- `fab/.kit/skills/fab-new.md` — Remove Step 4, remove `--branch` argument, add internal `/fab-switch` call after proposal, remove `branch:` from `.status.yaml` template in Step 5
- `fab/.kit/skills/fab-discuss.md` — Remove references to `branch:` field being omitted (no longer relevant since no skill writes it)
- `fab/.kit/skills/fab-status.md` — Update to use `git branch --show-current` instead of `.status.yaml` branch field
- `fab/.kit/skills/_context.md` — Remove `branch` from preflight YAML fields list, update `/fab-new` interruption budget (no more "0 for branch-on-main")
- `fab/.kit/templates/status.yaml` — Remove `branch:` field
- `fab/.kit/scripts/fab-preflight.sh` — Remove `branch` parsing and YAML output
- `fab/.kit/scripts/fab-status.sh` — Replace branch field display with live git query
- `fab/config.yaml` — Remove or relocate `branch_prefix`
- All skill files that reference `branch` in preflight YAML parsing — update parse instructions

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `/fab-new` calls `/fab-switch` at the end (after proposal), not at the beginning | Proposal generation doesn't need `fab/current` or a branch — safer to defer activation until the proposal is ready |
| 2 | Confident | `/fab-switch` needs `config.yaml` for `git.enabled` and `branch_prefix` | Branch integration requires knowing if git is enabled; this changes `/fab-switch` from "minimal context" to "loads config" |
| 3 | Confident | `branch_prefix` stays in `config.yaml` under `git:` section | It's still a project-level convention, just consumed by a different skill now |
| 4 | Confident | `fab-status` uses `git branch --show-current` for live branch display | More accurate than a static field; only shown when `git.enabled` and inside a git repo |
| 5 | Confident | Existing archived changes with `branch:` in `.status.yaml` are left as-is | Archived changes are read-only historical records; no migration needed |

5 assumptions made (5 confident, 0 tentative).
