# Brief: Add fab-switch variant to return to main branch

**Change**: 260212-egqa-switch-return-main
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [egqa]
**Linear**: DEV-1016
**User command**: `/fab-new egqa`

From the backlog: "Add a fab-switch variant takes you back to the main branch - ie. the state with no active change. Check if such a flow already exists."

## Why

Currently, fab-switch requires specifying a change name to switch to. There's no built-in way to deactivate all changes and return to a clean main branch state. This creates friction when users want to step out of the fab workflow temporarily or when starting fresh after completing a change.

## What Changes

- Add support for `/fab-switch main` (or similar syntax) to return to the main branch
- Clear `fab/current` when switching to main (deactivate all changes)
- Checkout the main branch if `git.enabled` is true
- Document the new behavior in fab-switch skill documentation
- Ensure the command is idempotent (safe to run when already on main)

## Affected Docs

### Modified Docs
- `fab-workflow/planning-skills`: Add documentation for the deactivation variant of fab-switch
- `fab-workflow/change-lifecycle`: Update the lifecycle diagram/description to show how to exit the active change state

## Impact

**Code areas**:
- `.claude/skills/fab-switch/skill.md` - add new behavior for "main" argument
- Possibly `fab/.kit/scripts/` if any helper scripts are involved in branch switching

**Workflow impact**:
- Provides a clear exit path from the fab workflow
- Complements the existing fab-switch behavior (switching between changes)
- No breaking changes - existing fab-switch behavior remains unchanged

**Edge cases**:
- Behavior when there's uncommitted work in the current change
- Interaction with git.enabled=false (should still clear fab/current)
- Handling when already on main with no active change

## Open Questions

None - the requirement is clear from the Linear description and backlog entry.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Clearing fab/current and checking out main is sufficient; no need to pull latest | Safer to leave the user in control of pulling updates. Reversibility is high. |
| 2 | Tentative | Command syntax is `/fab-switch main` | Most natural and consistent with git conventions. Alternative syntaxes like `--main` or `-` are possible but less intuitive. <!-- assumed: using 'main' as the argument --> |
