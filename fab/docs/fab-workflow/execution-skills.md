# Execution Skills

**Domain**: fab-workflow

## Overview

The execution skills (`/fab-apply`, `/fab-review`, `/fab-archive`) handle the implementation, validation, and completion stages of the Fab workflow. They execute tasks from the planning phase, validate the result against specs and checklists, and hydrate learnings into centralized docs on completion.

## Requirements

### `/fab-apply`

`/fab-apply` executes tasks from `tasks.md` in dependency order, running tests after each completed task.

#### Task Execution

1. Parse `tasks.md` for unchecked items (`- [ ]`)
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. After completing each task, run relevant tests (e.g., the test file for the module just modified). Fix failures before moving on
5. Mark each task `- [x]` immediately upon completion (not batched at the end)
6. Update `.status.yaml` progress after each task

#### Resumability

`/fab-apply` is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-apply` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

#### Auto-Guess Soft Gate

Before beginning implementation, `/fab-apply` SHALL scan `tasks.md` for `<!-- auto-guess: ... -->` markers. If any are found, the skill warns with a count, lists each auto-guess description, and prompts `Continue with implementation? (y/n)`. If the user declines, the skill aborts with a suggestion to run `/fab-clarify` first. If no markers are found, the gate is skipped entirely.

#### Context

Loads: config, constitution, `tasks.md`, `spec.md`, `plan.md` (if exists), relevant source code (files referenced in tasks).

### `/fab-review`

`/fab-review` validates implementation against specs and checklists. On pass, it advances to archive readiness. On failure, it presents rework options.

#### Validation Checks

The agent SHALL perform all of these checks:
1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklists/quality.md` verified and checked off — the agent re-reads each `CHK-*` item, inspects relevant code/tests, and marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No doc drift detected (implementation doesn't contradict centralized docs)

#### On Pass

All checks succeed → stage advances to review done.

#### On Failure

The agent presents options and the user chooses where to loop back:

- **Fix code** → `/fab-apply` — Implementation bug. The agent identifies which tasks need rework, unchecks them in `tasks.md` (marks `- [ ]` with a `<!-- rework: reason -->` comment), and re-runs `/fab-apply`
- **Revise tasks** → edit `tasks.md`, then `/fab-apply` — Missing or wrong tasks. New tasks get next sequential ID. Completed unaffected tasks stay `[x]`
- **Revise plan** → `/fab-continue plan` — Architecture was wrong. Resets to plan stage, updates `plan.md` in place, invalidates downstream (tasks reset, checklist regenerated)
- **Revise specs** → `/fab-continue specs` — Requirements were wrong or incomplete. Resets to specs stage, updates `spec.md`. Plan and tasks subsequently regenerated

The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

#### Context

Loads: config, constitution, `tasks.md`, `checklists/quality.md`, `spec.md`, target centralized doc(s) from `fab/docs/`, relevant source code (files touched by the change).

### `/fab-archive`

`/fab-archive` completes a change: validates review passed, hydrates learnings into centralized docs, and moves the change to archive.

#### Behavior

1. **Final validation** — review MUST have passed (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same centralized doc files. If found, warn: "Change {name} also modifies {doc}. After this archive, that change's spec was written against a now-stale base. Re-review with `/fab-review` after switching to it."
3. **Hydrate into `fab/docs/`**:
   - From `spec.md` → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements the spec explicitly deprecates
   - From `plan.md` → extract durable design decisions into Design Decisions section. Skip tactical details (file paths, setup steps, library install commands)
   - Compare against existing doc to determine what's new vs changed vs removed — no explicit delta markers needed
   - Minimize edits to unchanged sections to prevent drift
4. **Update status** to `archive: done` in `.status.yaml`
5. **Move change folder** to `archive/` (no rename — date already in folder name)
6. **Clear pointer** — delete `fab/current` (no active change)

#### Fail-Safe Order of Operations

Steps 3–6 are ordered to fail safely. Status is updated *before* the folder move, so if the move is interrupted, the change is marked archived but still in `changes/` — the agent can detect and complete the move on next invocation. The pointer is cleared last so mid-archive, `/fab-status` still reports the active change.

#### Recovery

Hydration modifies centralized docs in-place. If the merge goes wrong, the only recovery is `git checkout` on the affected doc files. Commit (or at least review the diff) before pushing after an archive.

#### Context

Loads: `spec.md`, `plan.md` (if exists), target centralized doc(s) from `fab/docs/`, `fab/docs/index.md` and relevant domain indexes.

## Design Decisions

### Checklist Tests Implementation Fidelity, Not Spec Quality
**Decision**: The quality checklist validates "does the code match the spec?" rather than "is the spec well-written?"
**Why**: Fab has explicit spec review during the specs stage (via `/fab-clarify`). A separate requirement-quality checklist is redundant. The checklist focuses on what matters at review time: implementation correctness.
**Rejected**: SpecKit-style requirement-quality checklist — duplicates work already done during planning stages.
*Source*: doc/fab-spec/TEMPLATES.md

### Review Failure Offers Multiple Re-Entry Points
**Decision**: On review failure, the agent presents four options (fix code, revise tasks, revise plan, revise specs) and the user chooses where to loop back.
**Why**: Not all review failures are implementation bugs. Some require revisiting upstream artifacts. Giving the user explicit choice prevents the agent from guessing wrong about where the problem originated.
**Rejected**: Always looping back to apply — misses cases where the spec or plan was wrong.
*Source*: doc/fab-spec/SKILLS.md

### Archive Hydrates Semantically, Not by Delta Markers
**Decision**: The agent compares `spec.md` against existing centralized docs to determine what's new, changed, or removed. No ADDED/MODIFIED/REMOVED markers in the spec.
**Why**: The spec reads as a straightforward requirements document. Delta markers would clutter the spec and couple it to the hydration mechanism.
**Rejected**: Explicit delta markers — clutters specs, requires discipline to maintain, fragile to editing.
*Source*: doc/fab-spec/TEMPLATES.md

### Concurrent Change Warning on Archive
**Decision**: Before hydrating, scan for other active changes that reference the same docs and warn the user.
**Why**: Hydration updates the centralized docs, which may invalidate assumptions in other active changes. The warning prompts re-review rather than allowing silent drift.
**Rejected**: Blocking archive if concurrent changes exist — too restrictive, especially for independent changes that happen to touch the same domain.
*Source*: doc/fab-spec/SKILLS.md

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-09sj-autonomy-framework | 2026-02-08 | Added auto-guess soft gate to `/fab-apply` — warns and prompts before implementing when `<!-- auto-guess: ... -->` markers exist |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab:xxx` colon format to `/fab-xxx` hyphen format |
| — | 2026-02-07 | Generated from doc/fab-spec/ (SKILLS.md, TEMPLATES.md) |
