# Spec: Unify Pipeline Commands into fab-continue

**Change**: 260212-a4bd-unify-fab-continue
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/execution-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changing any stage's actual behavior — the apply, review, and archive logic stays the same; only the entry point changes
- Modifying the 6-stage pipeline structure or stage ordering
- Combining fab-ff/fab-fff into fab-continue — they remain separate commands with distinct orchestration (frontloaded questions, auto-clarify, bail behavior)

## fab-continue: Stage Dispatch (Normal Flow)

### Requirement: Unified Stage Advancement

`/fab-continue` (no argument) SHALL advance through all 6 pipeline stages. The active stage in `.status.yaml` determines which behavior is dispatched. Each invocation handles one stage's work and transitions to the next.

The stage guard table SHALL be:

| Active stage | Action |
|---|---|
| `brief` | Generate `spec.md` (existing behavior) |
| `spec` | Generate `tasks.md` + checklist (existing behavior) |
| `tasks` | Execute apply behavior (new) |
| `apply` | Resume apply behavior if interrupted, or re-run (new) |
| `review` | Execute archive behavior (new) |
| No active entry (all done) | Block: "Change is complete." |

<!-- assumed: review active triggers archive, not re-review — the normal forward flow always advances. Re-review is handled via reset: fab-continue review -->

#### Scenario: Advance from tasks to apply
- **GIVEN** the active stage is `tasks`
- **WHEN** the user runs `/fab-continue`
- **THEN** fab-continue SHALL set `tasks: done, apply: active`
- **AND** execute apply behavior (parse unchecked tasks, execute in dependency order, run tests, mark complete)
- **AND** on completion set `apply: done, review: active`

#### Scenario: Resume interrupted apply
- **GIVEN** the active stage is `apply` (implementation was interrupted)
- **WHEN** the user runs `/fab-continue`
- **THEN** fab-continue SHALL resume apply behavior from the first unchecked task
- **AND** on completion set `apply: done, review: active`

#### Scenario: All tasks already complete during apply
- **GIVEN** the active stage is `tasks` or `apply`
- **AND** all tasks in `tasks.md` are already checked `[x]`
- **WHEN** the user runs `/fab-continue`
- **THEN** fab-continue SHALL set `apply: done, review: active`
- **AND** output: "All tasks already complete. Implementation finished."

#### Scenario: Advance from review to archive
- **GIVEN** the active stage is `review`
- **WHEN** the user runs `/fab-continue`
- **THEN** fab-continue SHALL execute archive behavior (validate, hydrate docs, move folder, clear pointer)
- **AND** on completion set `archive: done`

#### Scenario: Change already complete
- **GIVEN** all stages are `done` (no active entry)
- **WHEN** the user runs `/fab-continue`
- **THEN** fab-continue SHALL output: "Change is complete."

### Requirement: Review Behavior with Rework Options

When `/fab-continue` dispatches to review behavior (active stage is `apply` with `apply: done`), it SHALL run the same validation as the current `/fab-review`: verify tasks, verify checklist, run tests, spot-check spec requirements, check doc drift.

#### Scenario: Review passes
- **GIVEN** the active stage transitions to review
- **WHEN** all review checks pass
- **THEN** fab-continue SHALL set `review: done, archive: active`
- **AND** output the review report with "Next: /fab-continue (archive)"

#### Scenario: Review fails — rework options
- **GIVEN** the active stage transitions to review
- **WHEN** one or more review checks fail
- **THEN** fab-continue SHALL set `review: failed, apply: active`
- **AND** present three rework options:
  1. **Fix code** — uncheck affected tasks in `tasks.md` (with `<!-- rework: reason -->` comment), then user runs `/fab-continue` to resume apply
  2. **Revise tasks** — user edits `tasks.md`, then runs `/fab-continue` to resume apply
  3. **Revise spec** — run `/fab-continue spec` to reset and regenerate all downstream

## fab-continue: Apply Behavior

### Requirement: Task Execution (Absorbed from fab-apply)

When dispatched to apply behavior, `/fab-continue` SHALL execute tasks from `tasks.md` with identical behavior to the current `/fab-apply`:

1. Parse `tasks.md` for unchecked items (`- [ ]`)
2. Execute in dependency order (phases sequential, `[P]` tasks parallelizable within phase)
3. Respect Execution Order constraints from `tasks.md`
4. After each task: run relevant tests, fix failures, mark `- [x]` immediately
5. Update `.status.yaml` `last_updated` after each task completion
6. On completion: set `apply: done`

#### Scenario: Normal task execution
- **GIVEN** `tasks.md` has unchecked tasks
- **WHEN** fab-continue dispatches to apply behavior
- **THEN** tasks SHALL be executed in dependency order
- **AND** tests SHALL run after each task
- **AND** each completed task SHALL be marked `[x]` immediately

#### Scenario: Resuming after interruption
- **GIVEN** some tasks are `[x]` and some are `[ ]`
- **WHEN** fab-continue dispatches to apply behavior
- **THEN** execution SHALL resume from the first unchecked task
- **AND** output: "Resuming implementation. {M} of {N} tasks already complete, {R} remaining."

### Requirement: Apply Context Loading

When executing apply behavior, `/fab-continue` SHALL load: `config.yaml`, `constitution.md`, `design/index.md`, `tasks.md`, `spec.md`, `brief.md`, and relevant source code referenced in task descriptions.

#### Scenario: Source code scoping
- **GIVEN** a task references specific file paths
- **WHEN** fab-continue loads context for that task
- **THEN** it SHALL read the referenced files (not the entire codebase)

## fab-continue: Review Behavior

### Requirement: Implementation Validation (Absorbed from fab-review)

When dispatched to review behavior, `/fab-continue` SHALL perform the same 5-step validation as the current `/fab-review`:

1. Verify all tasks marked `[x]`
2. Verify all checklist items — inspect code/tests for each `CHK-*` item, mark `[x]` or record failure
3. Run affected tests (scoped to modules touched)
4. Spot-check spec requirements against implementation
5. Check for doc drift against centralized docs

#### Scenario: Checklist verification
- **GIVEN** `checklist.md` has items to verify
- **WHEN** fab-continue runs review behavior
- **THEN** each `CHK-*` item SHALL be inspected against code/tests
- **AND** met items SHALL be marked `[x]`
- **AND** N/A items SHALL be marked `[x]` with `**N/A**: {reason}` prefix
- **AND** unmet items SHALL remain `[ ]` with failure recorded

#### Scenario: Unchecked tasks block review
- **GIVEN** some tasks in `tasks.md` are unchecked
- **WHEN** fab-continue attempts review behavior
- **THEN** it SHALL stop with: "{N} of {total} tasks are incomplete. Run /fab-continue to finish implementation first."

### Requirement: Review Context Loading

When executing review behavior, `/fab-continue` SHALL load: `config.yaml`, `constitution.md`, `design/index.md`, `tasks.md`, `checklist.md`, `spec.md`, `brief.md`, centralized docs from Affected Docs section, and relevant source code.

#### Scenario: Review loads all context
- **GIVEN** fab-continue dispatches to review
- **WHEN** context is loaded
- **THEN** centralized docs listed in the brief's Affected Docs section SHALL be loaded for doc drift checking

## fab-continue: Archive Behavior

### Requirement: Change Completion (Absorbed from fab-archive)

When dispatched to archive behavior, `/fab-continue` SHALL perform the same behavior as the current `/fab-archive`:

1. **Final validation** — verify review passed, all tasks `[x]`, all checklist items `[x]`
2. **Concurrent change check** — warn about other active changes modifying same docs
3. **Hydrate into `fab/docs/`** — integrate new/changed requirements and design decisions from `spec.md`
4. **Update `.status.yaml`** — set `archive: done`
5. **Move change folder** to `fab/changes/archive/`
6. **Update archive index** — create or update `fab/changes/archive/index.md`
7. **Mark backlog item done** — if brief contains a backlog ID
8. **Clear pointer** — delete `fab/current`

Steps 4-8 SHALL execute in fail-safe order (status first, pointer last).

#### Scenario: Successful archive
- **GIVEN** review has passed (progress.review: done)
- **AND** all tasks and checklist items are `[x]`
- **WHEN** fab-continue dispatches to archive behavior
- **THEN** docs SHALL be hydrated, change SHALL be moved to archive, pointer SHALL be cleared
- **AND** output: "Archive complete. Next: /fab-new <description>"

#### Scenario: Review not passed blocks archive
- **GIVEN** progress.review is not `done`
- **WHEN** fab-continue attempts archive behavior
- **THEN** it SHALL stop with: "Review has not passed. Run /fab-continue to validate implementation first."

#### Scenario: Concurrent change warning
- **GIVEN** another active change references the same centralized docs
- **WHEN** fab-continue runs archive behavior
- **THEN** it SHALL warn: "Change {other-name} also modifies {doc-path}. Re-review with /fab-continue after switching."
- **AND** proceed with archiving (warning only, not blocking)

### Requirement: Archive Context Loading

When executing archive behavior, `/fab-continue` SHALL load: `config.yaml`, `constitution.md`, `design/index.md`, `spec.md`, `brief.md`, `docs/index.md`, and target centralized docs from Affected Docs section.

#### Scenario: Hydration context
- **GIVEN** the brief's Affected Docs lists specific doc paths
- **WHEN** fab-continue loads archive context
- **THEN** it SHALL read each listed centralized doc (if it exists) and the domain index

## fab-continue: Extended Reset Flow

### Requirement: Reset to Any Stage

`/fab-continue <stage>` SHALL accept all 6 stages as reset targets: `brief`, `spec`, `tasks`, `apply`, `review`, `archive`.

For **planning stages** (brief, spec, tasks), reset behavior remains the same as current: regenerate the artifact, invalidate downstream.

For **execution stages** (apply, review, archive), reset SHALL: set the target stage to `active`, reset all subsequent stages to `pending`, then re-run the stage's behavior.

<!-- assumed: execution stage resets re-run behavior without resetting task checkboxes — checkbox state reflects real implementation progress and should not be silently undone -->

#### Scenario: Reset to apply
- **GIVEN** the change is at review or later
- **WHEN** the user runs `/fab-continue apply`
- **THEN** fab-continue SHALL set `apply: active`, `review: pending`, `archive: pending`
- **AND** execute apply behavior (resume from first unchecked task)

#### Scenario: Reset to review
- **GIVEN** the change is at archive stage
- **WHEN** the user runs `/fab-continue review`
- **THEN** fab-continue SHALL set `review: active`, `archive: pending`
- **AND** execute review behavior (full validation)

#### Scenario: Reset to archive
- **GIVEN** the change needs re-archiving
- **WHEN** the user runs `/fab-continue archive`
- **THEN** fab-continue SHALL set `archive: active`
- **AND** execute archive behavior

#### Scenario: Reset to brief
- **GIVEN** the user wants to start over
- **WHEN** the user runs `/fab-continue brief`
- **THEN** fab-continue SHALL set `brief: active`, all other stages to `pending`
- **AND** regenerate `brief.md` in place

#### Scenario: Reset to spec (existing behavior)
- **GIVEN** the user runs `/fab-continue spec`
- **WHEN** the change is at tasks or later
- **THEN** existing reset behavior applies: regenerate spec.md, invalidate downstream

#### Scenario: Reset to tasks (existing behavior)
- **GIVEN** the user runs `/fab-continue tasks`
- **WHEN** the change is at apply or later
- **THEN** existing reset behavior applies: regenerate tasks.md, reset checkboxes, regenerate checklist

## fab-ff and fab-fff: Updated References

### Requirement: fab-ff Internal References

`/fab-ff` SHALL update all internal references from standalone skill names to `fab-continue` behavior:

- Step 6 (Implementation): "Execute apply behavior" (remove reference to `/fab-apply`)
- Step 7 (Review): "Execute review behavior" (remove reference to `/fab-review`)
- Step 8 (Archive): "Execute archive behavior" (remove reference to `/fab-archive`)
- Review rework options: reference `/fab-continue` instead of `/fab-apply` or `/fab-continue spec`

#### Scenario: fab-ff review rework references
- **GIVEN** `/fab-ff` review fails
- **WHEN** rework options are presented
- **THEN** "Fix code" SHALL reference `/fab-continue` (not `/fab-apply`)
- **AND** "Revise spec" SHALL reference `/fab-continue spec`

### Requirement: fab-fff Internal References

`/fab-fff` SHALL update all internal references from standalone skill names to `fab-continue` behavior, following the same pattern as `/fab-ff`.

#### Scenario: fab-fff references
- **GIVEN** `/fab-fff` references execution skills
- **WHEN** the skill files are updated
- **THEN** all references to `/fab-apply`, `/fab-review`, `/fab-archive` SHALL be replaced with `fab-continue` equivalents

## Skill Deletion

### Requirement: Remove Standalone Execution Skills

The following skill files SHALL be deleted from `fab/.kit/skills/`:
- `fab-apply.md`
- `fab-review.md`
- `fab-archive.md`

#### Scenario: Skills removed
- **GIVEN** the apply, review, and archive behavior is absorbed into `fab-continue`
- **WHEN** the change is applied
- **THEN** the three standalone skill files SHALL no longer exist
- **AND** no other skill or doc SHALL reference them as invocable commands

## Next Steps Convention

### Requirement: Simplified Next Steps Table

The Next Steps lookup table in `_context.md` SHALL be updated to reflect the unified command:

| After skill | Stage reached | Next line |
|---|---|---|
| `/fab-init` | initialized | `Next: /fab-new <description> or /fab-hydrate <sources>` |
| `/fab-hydrate` | docs hydrated | `Next: /fab-new <description> or /fab-hydrate <more-sources>` |
| `/fab-new` | brief done | `Next: /fab-continue or /fab-ff` |
| `/fab-continue` → spec | spec done | `Next: /fab-continue or /fab-ff or /fab-clarify` |
| `/fab-continue` → tasks | tasks done | `Next: /fab-continue or /fab-ff` |
| `/fab-continue` → apply | apply done | `Next: /fab-continue` |
| `/fab-continue` → review (pass) | review done | `Next: /fab-continue` |
| `/fab-continue` → review (fail) | review failed | *(contextual rework options)* |
| `/fab-continue` → archive | archived | `Next: /fab-new <description>` |
| `/fab-ff` | archived | `Next: /fab-new <description>` |
| `/fab-ff` (bail) | varies | *(contextual)* |
| `/fab-clarify` | same stage | `Next: /fab-clarify or /fab-continue or /fab-ff` |
| `/fab-fff` | archived | `Next: /fab-new <description>` |
| `/fab-fff` (bail) | varies | *(contextual)* |

#### Scenario: Next line after apply
- **GIVEN** fab-continue completes apply (all tasks done)
- **WHEN** the Next line is displayed
- **THEN** it SHALL read: "Next: /fab-continue"

#### Scenario: Next line after review pass
- **GIVEN** fab-continue completes review (pass)
- **WHEN** the Next line is displayed
- **THEN** it SHALL read: "Next: /fab-continue"

## Cross-Reference Updates

### Requirement: Update All References to Removed Skills

All files referencing `/fab-apply`, `/fab-review`, or `/fab-archive` as invocable commands SHALL be updated:

**Skills to update**:
- `fab-continue.md` — remove guards that block at execution stages
- `fab-ff.md` — update internal references and rework options
- `fab-fff.md` — update internal references
- `_context.md` — update Next Steps lookup table
- `fab-new.md` — update any "Next:" output lines
- `fab-clarify.md` — update any "Next:" output lines
- `fab-status.md` — update any command references

**Other files to update**:
- `fab/design/skills.md` — update skill references
- `fab/design/user-flow.md` — update command map
- `fab/design/overview.md` — update quick reference
- `README.md` — update command references
- `.claude/settings.local.json` — remove permission entries for deleted skills

#### Scenario: No dangling references
- **GIVEN** all files have been updated
- **WHEN** searching the codebase for `/fab-apply`, `/fab-review`, or `/fab-archive`
- **THEN** zero results SHALL be found outside of archived changes and changelogs

## Deprecated Requirements

### `/fab-apply` as Standalone Skill
**Reason**: Behavior absorbed into `/fab-continue`. Running `/fab-continue` when apply is the active stage (or using `/fab-continue apply` reset) replaces the standalone invocation.
**Migration**: Use `/fab-continue` instead.

### `/fab-review` as Standalone Skill
**Reason**: Behavior absorbed into `/fab-continue`. Running `/fab-continue` when review is the active stage (or using `/fab-continue review` reset) replaces the standalone invocation.
**Migration**: Use `/fab-continue` instead.

### `/fab-archive` as Standalone Skill
**Reason**: Behavior absorbed into `/fab-continue`. Running `/fab-continue` when archive is the active stage (or using `/fab-continue archive` reset) replaces the standalone invocation.
**Migration**: Use `/fab-continue` instead.

## Design Decisions

1. **Inline execution behavior into fab-continue sections**: Apply, review, and archive behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial like `_generation.md`.
   - *Why*: These are orchestration-heavy stages with distinct flows (task execution, validation with rework, hydration with folder moves). The existing `_generation.md` partial covers artifact generation mechanics, which are fundamentally different. Inlining keeps each stage's full behavior in one readable location.
   - *Rejected*: Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these directly (fab-ff/fff describe the same behavior inline for their own orchestration context).

2. **Execution stage reset does not reset task checkboxes**: `/fab-continue apply` re-runs apply behavior starting from the first unchecked task. It does NOT uncheck all tasks.
   - *Why*: Task checkboxes reflect actual implementation progress. Silently unchecking them would discard valid work. Review rework (Option 1) handles targeted unchecking with `<!-- rework: reason -->` annotations. The user can also manually uncheck specific tasks.
   - *Rejected*: Resetting all checkboxes on `apply` reset — too destructive, discards completed work.

3. **Review active triggers archive in normal flow**: When the active stage is `review`, `/fab-continue` advances to archive behavior, not re-review.
   - *Why*: The normal flow always advances. After review passes, the stage transitions to `review: done, archive: active`. So when archive is the active stage, fab-continue runs archive. If review is still active, it means the review behavior hasn't been run yet (either first time or after a reset), so fab-continue runs the review. The key distinction: `review: active` = "review needs to run", `review: done` = "review passed, archive is next."
   - *Rejected*: Having review active trigger re-review — conflicts with the forward-progression model. Users can use `/fab-continue review` reset for explicit re-review.

4. **fab-ff/fff keep their own behavioral descriptions**: Rather than literally calling `/fab-continue` as a sub-skill, fab-ff and fab-fff describe the same behavior inline within their own orchestration context.
   - *Why*: fab-ff/fff have fundamentally different orchestration: frontloaded questions, auto-clarify interleaving, bail behavior, resumability across all stages. Their execution steps (apply, review, archive) use the same logic as fab-continue but within a different control flow. Literal sub-skill invocation would add complexity (nested preflight checks, status conflicts) without benefit.
   - *Rejected*: Literal `/fab-continue` invocation from fab-ff/fff — orchestration mismatch, nested state management issues.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Inline execution behavior as fab-continue sections, not partials | Brief says "absorb"; orchestration-heavy stages benefit from colocated descriptions |
| 2 | Confident | fab-ff/fff keep behavioral descriptions, update references only | Different orchestration (frontloaded questions, auto-clarify, bail) makes literal invocation impractical |
| 3 | Confident | Execution stage reset re-runs behavior without resetting task checkboxes | Checkboxes reflect real progress; targeted unchecking is handled by review rework flow |
| 4 | Confident | `fab-continue brief` supported as reset target | Brief says "any specific stage"; consistent with all-stages reset model |
| 5 | Confident | Review active triggers forward progression to archive, not re-review | Normal flow always advances; re-review available via `/fab-continue review` reset |

5 assumptions made (5 confident, 0 tentative). Run /fab-clarify to review.
