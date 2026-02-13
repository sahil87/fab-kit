# Execution Skills

**Domain**: fab-workflow

## Overview

Execution behavior (apply, review, archive) is accessed via `/fab-continue`, which dispatches to the appropriate behavior based on the active stage. The standalone skills `/fab-apply`, `/fab-review`, and `/fab-archive` no longer exist as separate commands. All execution skills inherit the optional `[change-name]` argument from `/fab-continue`, which is passed to the preflight script for transient change resolution without modifying `fab/current`.

**Pipeline invocation**: Both `/fab-ff` and `/fab-fff` use the same execution behavior internally as part of their full-pipeline runs. `/fab-ff` presents interactive rework options on review failure; `/fab-fff` bails immediately. Both accept an optional `[change-name]` argument.

## Requirements

### Apply Behavior (via `/fab-continue`)

`/fab-continue` dispatches to apply behavior when the active stage is `tasks` or `apply`. It executes tasks from `tasks.md` in dependency order, running tests after each completed task.

#### Task Execution

1. Parse `tasks.md` for unchecked items (`- [ ]`)
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. After completing each task, run relevant tests (e.g., the test file for the module just modified). Fix failures before moving on
5. Mark each task `- [x]` immediately upon completion (not batched at the end)
6. Update `.status.yaml` progress after each task

#### Resumability

Apply behavior is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-continue` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

#### Context

Loads: config, constitution, `design/index.md`, `tasks.md`, `spec.md`, relevant source code (files referenced in tasks).

### Review Behavior (via `/fab-continue`)

`/fab-continue` dispatches to review behavior after apply completes. It validates implementation against specs and checklists. On pass, it advances to archive readiness. On failure, it presents rework options.

#### Validation Checks

The agent SHALL perform all of these checks:
1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklist.md` verified and checked off — the agent re-reads each `CHK-*` item, inspects relevant code/tests, and marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No doc drift detected (implementation doesn't contradict centralized docs)

#### On Pass

All checks succeed → stage advances to review done.

#### On Failure

The agent presents options and the user chooses where to loop back:

- **Fix code** → `/fab-continue` — Implementation bug. The agent identifies which tasks need rework, unchecks them in `tasks.md` (marks `- [ ]` with a `<!-- rework: reason -->` comment), and re-runs `/fab-continue`
- **Revise tasks** → edit `tasks.md`, then `/fab-continue` — Missing or wrong tasks. New tasks get next sequential ID. Completed unaffected tasks stay `[x]`
- **Revise spec** → `/fab-continue spec` — Requirements were wrong or incomplete. Resets to spec stage, updates `spec.md`. Tasks subsequently regenerated

The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

#### Context

Loads: config, constitution, `design/index.md`, `tasks.md`, `checklist.md`, `spec.md`, target centralized doc(s) from `fab/docs/`, relevant source code (files touched by the change).

### Archive Behavior (via `/fab-continue`)

`/fab-continue` dispatches to archive behavior after review passes. It completes a change: validates review passed, hydrates learnings into centralized docs, and moves the change to archive.

#### Behavior

1. **Final validation** — review MUST have passed (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same centralized doc files. If found, warn: "Change {name} also modifies {doc}. After this archive, that change's spec was written against a now-stale base. Re-review with `/fab-continue` after switching to it."
3. **Hydrate into `fab/docs/`**:
   - From `spec.md` → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements the spec explicitly deprecates. Extract durable design decisions into Design Decisions section
   - Compare against existing doc to determine what's new vs changed vs removed — no explicit delta markers needed
   - Minimize edits to unchanged sections to prevent drift
4. **Update status** to `archive: done` in `.status.yaml`
5. **Move change folder** to `archive/` (no rename — date already in folder name)
6. **Update archive index** — append an entry to `fab/changes/archive/index.md` (create with backfill of all existing entries if it doesn't exist). Entry format: `- **{folder-name}** — {1-2 sentence description from brief Why section}`. Most-recent-first ordering.
7. **Clear pointer** — delete `fab/current` (no active change)

#### Fail-Safe Order of Operations

Steps 3–7 are ordered to fail safely. Status is updated *before* the folder move, so if the move is interrupted, the change is marked archived but still in `changes/` — the agent can detect and complete the move on next invocation. The index is updated after the folder is in place but before the pointer is cleared, so mid-archive, `/fab-status` still reports the active change.

#### Recovery

Hydration modifies centralized docs in-place. If the merge goes wrong, the only recovery is `git checkout` on the affected doc files. Commit (or at least review the diff) before pushing after an archive.

#### Context

Loads: config, constitution, `design/index.md`, `spec.md`, target centralized doc(s) from `fab/docs/`, `fab/docs/index.md` and relevant domain indexes.

## Design Decisions

### Checklist Tests Implementation Fidelity, Not Spec Quality
**Decision**: The quality checklist validates "does the code match the spec?" rather than "is the spec well-written?"
**Why**: Fab has explicit spec review during the spec stage (via `/fab-clarify`). A separate requirement-quality checklist is redundant. The checklist focuses on what matters at review time: implementation correctness.
**Rejected**: SpecKit-style requirement-quality checklist — duplicates work already done during planning stages.
*Source*: doc/fab-spec/TEMPLATES.md

### Review Failure Offers Multiple Re-Entry Points
**Decision**: On review failure, the agent presents three options (fix code, revise tasks, revise spec) and the user chooses where to loop back.
**Why**: Not all review failures are implementation bugs. Some require revisiting upstream artifacts. Giving the user explicit choice prevents the agent from guessing wrong about where the problem originated.
**Rejected**: Always looping back to apply — misses cases where the spec was wrong.
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

### Execution Stage Reset Preserves Task Checkboxes
**Decision**: `/fab-continue apply` re-runs apply behavior starting from the first unchecked task. It does NOT uncheck all tasks.
**Why**: Task checkboxes reflect actual implementation progress. Silently unchecking them would discard valid work. Review rework (Option 1: "Fix code") handles targeted unchecking with `<!-- rework: reason -->` annotations.
**Rejected**: Resetting all checkboxes on apply reset — too destructive, discards completed work.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Review Active Triggers Forward Progression
**Decision**: When the active stage is `review`, `/fab-continue` runs the review behavior (advancing toward archive), not re-review. Re-review is available via `/fab-continue review` reset.
**Why**: The normal flow always advances. `review: active` means "review needs to run"; `review: done` means "review passed, archive is next." This avoids ambiguity about whether the command should redo or advance.
**Rejected**: Having review active trigger re-review — conflicts with the forward-progression model.
*Introduced by*: 260212-a4bd-unify-fab-continue

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | Execution skills now inherit optional `[change-name]` argument via `/fab-continue` preflight override; `fab-status.sh` also accepts change-name override directly |
| 260212-a4bd-unify-fab-continue | 2026-02-12 | Restructured: apply, review, and archive behavior now accessed via `/fab-continue` instead of standalone skills. Updated all section headings, requirements, and cross-references |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Updated checklist path references from `checklists/quality.md` to `checklist.md` in `/fab-review` and `/fab-archive` |
| 260212-bk1n-rework-fab-ff-archive | 2026-02-12 | Added note that `/fab-ff` and `/fab-fff` invoke execution skills internally as part of their full-pipeline behavior |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated stage references from proposal/specs to brief/spec |
| 260211-endg-add-created-by-field | 2026-02-11 | `fab-status.sh` now displays `Created by:` line when `created_by` field is present in `.status.yaml` |
| 260210-7wxx-add-specs-index-context-loading | 2026-02-10 | Added `fab/design/index.md` to context loading for all three execution skills, aligning with the always-load protocol in `_context.md` |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Added archive index maintenance step to `/fab-archive` — creates/updates `fab/changes/archive/index.md` with searchable change summaries |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Removed auto-guess soft gate from `/fab-apply` — replaced by confidence gating on `/fab-fff` |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added auto-guess soft gate to `/fab-apply` (subsequently removed by 260208-k3m7-add-fab-fff) |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| — | 2026-02-07 | Generated from doc/fab-spec/ (SKILLS.md, TEMPLATES.md) |
