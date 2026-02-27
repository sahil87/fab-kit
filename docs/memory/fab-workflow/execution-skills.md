# Execution Skills

**Domain**: fab-workflow

## Overview

Execution behavior (apply, review, hydrate) is accessed via `/fab-continue`, which dispatches to the appropriate behavior based on the active stage. The standalone skills `/fab-apply` and `/fab-review` no longer exist as separate commands — their behavior is consolidated into `/fab-continue`. `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for moving completed changes to the archive. All execution behaviors in `/fab-continue` inherit the optional `[change-name]` argument, which is passed to the preflight script for transient change resolution without modifying `fab/current`.

**Status mutations**: All `.status.yaml` progress transitions, checklist updates, and confidence writes use `fab/.kit/scripts/lib/stageman.sh` CLI event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-checklist`, `set-confidence`) via the Bash tool, rather than direct file editing. This centralizes validation and ensures atomic writes with `last_updated` refresh. The `driver` parameter is optional but skills always pass it. Stage metrics (started_at, completed_at, driver, iterations) are updated automatically as side-effects.

**Pipeline invocation**: Both `/fab-fff` and `/fab-ff` use the same execution behavior internally as part of their pipeline runs. All three pipeline skills (`/fab-continue`, `/fab-ff`, `/fab-fff`) dispatch review to a sub-agent in a separate execution context, producing structured findings with three-tier priority (must-fix / should-fix / nice-to-have). `/fab-continue` preserves manual rework on failure; `/fab-ff` auto-loops between apply and review (up to 3 cycles) with interactive fallback on exhaustion; `/fab-fff` uses autonomous rework with bounded retry (3-cycle cap, escalation after 2 consecutive fix-code failures, bail on exhaustion). Both `/fab-ff` and `/fab-fff` accept an optional `[change-name]` argument.

**PR type system**: `/git-pr` supports 7 PR types (feat, fix, refactor, docs, test, ci, chore) derived from Conventional Commits. Types are resolved via a four-step chain: explicit argument → read from `.status.yaml` → infer from fab change intake → infer from diff. The type controls the PR title prefix (`{type}: {title}`) and template tier: Tier 1 (fab-linked: feat/fix/refactor) shows a Context table with Type, Change, Confidence score, Pipeline (completed stages joined with `→`), and separate Intake/Spec blob URL rows. Tier 2 (lightweight: docs/test/ci/chore) shows "No design artifacts — housekeeping change." Blob URLs use `https://github.com/{owner}/{repo}/blob/{branch}/...` to resolve against the feature branch instead of main.

## Requirements

### Apply Behavior (via `/fab-continue`)

`/fab-continue` dispatches to apply behavior when the active stage is `tasks` or `apply`. It executes tasks from `tasks.md` in dependency order, running tests after each completed task.

#### Pattern Extraction

Before executing the first unchecked task, the agent reads existing source files in the areas the change will touch and extracts: naming conventions, error handling style, typical structure, and reusable utilities. These patterns are held as context for all subsequent task execution. If `config.yaml` defines a `code_quality` section, its `principles` are loaded as additional constraints and `test_strategy` governs test timing (default: `test-alongside`). Pattern extraction is skipped when resuming mid-apply.

#### Task Execution

1. Parse `tasks.md` for unchecked items (`- [ ]`)
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. For each unchecked task:
   1. Read source files relevant to this task
   2. Implement per spec, constitution, and extracted patterns
   3. Prefer reusing existing utilities over creating new ones
   4. Keep functions focused — consider extracting if implementation exceeds the codebase's typical function size
   5. Write tests per `code_quality.test_strategy` (default: `test-alongside`)
   6. Run tests, fix failures
   7. Mark `[x]` immediately
5. Update `.status.yaml` progress after each task

#### Resumability

Apply behavior is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-continue` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

#### Context

Loads: config, constitution, `specs/index.md`, `tasks.md`, `spec.md`, relevant source code (files referenced in tasks), neighboring files for pattern extraction.

### Review Behavior (via `/fab-continue`)

`/fab-continue` dispatches to review behavior after apply completes. Review validation is dispatched to a **sub-agent running in a separate execution context**. The sub-agent provides a fresh perspective — it has no shared context with the applying agent beyond the explicitly provided artifacts.

The orchestrating LLM MAY use any review agent available in its environment (e.g., a `code-review` skill, a general-purpose sub-agent with review instructions, or any equivalent). The skill files do not hardcode a specific agent name or tool.

The review sub-agent performs capable-tier work: deep reasoning, code analysis, spec comparison, and checklist validation.

#### Validation Checks

The sub-agent performs all of these checks:
1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklist.md` verified and checked off — inspects relevant code/tests per `CHK-*` item, marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No memory drift detected (implementation doesn't contradict memory files)
6. Code quality check — for each file modified during apply: naming conventions consistent with surrounding code, functions focused and appropriately sized, error handling consistent with codebase style, existing utilities reused. If `config.yaml` defines `code_quality.principles`, check each applicable principle. If `code_quality.anti_patterns` defined, check for violations. Report code quality issues with specific file:line references; classify as should-fix by default, and as must-fix only when they correspond to spec mismatches, functional defects, or violations of hard project constraints

#### Structured Review Output

The sub-agent returns structured findings with a three-tier priority scheme:

- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed during rework
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

Each finding includes: severity tier, description, and file:line reference where applicable. Pass/fail determination: if any must-fix findings exist, the review fails. If only should-fix and/or nice-to-have remain, the review MAY pass.

#### On Pass

All checks succeed → stage advances to review done. The skill calls `fab/.kit/scripts/lib/stageman.sh log-review <change_dir> "passed"` to record the review outcome in `.history.jsonl`.

#### On Failure

The skill calls `fab/.kit/scripts/lib/stageman.sh log-review <change_dir> "failed" "<rework-option>"` to record the review outcome. Rework behavior differs by invoking skill:

**`/fab-continue` (manual rework)**: Presents the sub-agent's prioritized findings to the user, then offers three rework options:

- **Fix code** — the agent identifies affected tasks, unchecks them in `tasks.md` with `<!-- rework: reason -->` annotations, re-runs apply, then spawns a fresh sub-agent for re-review
- **Revise tasks** — the user edits `tasks.md` (add/modify tasks), then the agent re-runs apply for unchecked tasks and spawns a fresh sub-agent for re-review
- **Revise spec** → `/fab-continue spec` — resets to spec stage, regenerates downstream, re-runs apply, then spawns a fresh sub-agent for re-review

**`/fab-ff` (auto-loop + interactive fallback)**: The agent triages the sub-agent's prioritized findings and autonomously selects the rework path (up to 3 cycles). Each cycle = one rework action + one re-review by a fresh sub-agent. On exhaustion, falls back to interactive rework (same 3 options presented to the user, no further retry cap).

**`/fab-fff` (auto-loop + bail)**: Same autonomous triage and rework as `/fab-ff`, but on exhaustion after 3 cycles, bails with a cycle summary instead of falling back to interactive.

**Escalation rule** (applies to `/fab-ff` and `/fab-fff` auto-loops): After 2 consecutive "fix code" attempts, the agent MUST escalate to "revise tasks" or "revise spec". Non-fix-code actions reset the consecutive counter.

**Comment triage**: The applying agent triages review comments by priority — not all comments need to be implemented. Must-fix items are always addressed. Should-fix items are addressed when clear and low-effort. Nice-to-have items may be acknowledged but deferred.

The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

#### Context

Loads: config, constitution, `specs/index.md`, `tasks.md`, `checklist.md`, `spec.md`, target memory file(s) from `docs/memory/`, relevant source code (files touched by the change).

### Hydrate Behavior (via `/fab-continue`)

`/fab-continue` dispatches to hydrate behavior after review passes. It completes the pipeline: validates review passed and hydrates learnings into memory files. The change folder remains in `fab/changes/` after hydrate — archiving is a separate step via `/fab-archive`.

#### Behavior

1. **Final validation** — review MUST have passed (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same memory files. If found, warn: "Change {name} also modifies {file}. After this hydrate, that change's spec was written against a now-stale base. Re-review with `/fab-continue` after switching to it."
3. **Hydrate into `docs/memory/`**:
   - From `spec.md` → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements the spec explicitly deprecates. Extract durable design decisions into Design Decisions section
   - Compare against existing memory file to determine what's new vs changed vs removed — no explicit delta markers needed
   - Minimize edits to unchanged sections to prevent drift
4. **Update status** to `hydrate: done` in `.status.yaml`
5. **Pattern capture** *(optional)* — if the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section with the change name for traceability. Skip for implementations that follow existing patterns

#### Recovery

Hydration modifies memory files in-place. If the merge goes wrong, the only recovery is `git checkout` on the affected memory files. Commit (or at least review the diff) before pushing after hydrate.

#### Context

Loads: config, constitution, `specs/index.md`, `spec.md`, `intake.md`, target memory file(s) from `docs/memory/`, `docs/memory/index.md` and relevant domain indexes.

### `/fab-archive` (Standalone Skill)

`/fab-archive` is a standalone housekeeping command — not a pipeline stage. It supports two modes: **archive** (default) moves completed changes to the archive; **restore** moves archived changes back to active.

#### Archive Mode

##### Precondition

Requires `hydrate: done` in `.status.yaml`. If hydrate is not done, it stops with: "Hydrate has not completed. Run /fab-continue to hydrate memory first."

##### Behavior

1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. No rename.
2. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first.
3. **Mark backlog items done** — exact-ID check (always), then keyword scan with interactive confirmation
4. **Clear pointer** — delete `fab/current` only if the archived change is the active one

##### Fail-Safe Order of Operations

Steps 1–4 execute in this order for safety. Folder move first (recoverable if interrupted — re-run detects folder already in archive and completes remaining steps). Index after folder is in place. Backlog marking after index. Pointer last.

#### Restore Mode (`/fab-archive restore <change-name> [--switch]`)

Restores an archived change back to `fab/changes/`. Inverse of the archive operation. Preserves all artifacts and `.status.yaml` without modification — no status reset, no artifact regeneration.

##### Precondition

`<change-name>` is required. Resolved via case-insensitive substring matching against folder names in `fab/changes/archive/`. Supports exact/single/ambiguous/no-match flows (same pattern as `/fab-switch`).

##### Behavior

1. **Move change folder** — `fab/changes/archive/{name}/` → `fab/changes/{name}/`. No rename. All artifacts preserved.
2. **Remove archive index entry** — remove the entry for `{name}` from `fab/changes/archive/index.md`. Preserve empty index file.
3. **Update pointer** (conditional) — if `--switch` flag provided, write `{name}` to `fab/current`. Otherwise no-op.

Steps execute 1→3 for safety. If interrupted, re-run detects folder already in `fab/changes/` and completes remaining steps (index cleanup, optional pointer update).

#### Key Properties

- Does NOT modify `.status.yaml` progress (may update `last_updated`)
- Accepts optional `[change-name]` argument for targeting a specific change (archive mode)
- Conditional pointer clearing in archive mode — only clears `fab/current` when the archived change is the active one
- Restore mode requires explicit `<change-name>` — no "restore most recent" convenience
- Restore mode optionally activates via `--switch` flag

## Design Decisions

### Checklist Tests Implementation Fidelity and Code Quality
**Decision**: The quality checklist validates "does the code match the spec?" (implementation fidelity) and "is the code well-written?" (code quality). Code Quality is always included with at least two baseline items (pattern consistency, no unnecessary duplication); additional items derive from `config.yaml` `code_quality` section when present.
**Why**: Spec quality is addressed during the spec stage (via `/fab-clarify`), but code quality is only observable at review time. The baseline items are universally applicable; project-specific standards come from config.
**Rejected**: Code quality as opt-in only — would miss quality checks on projects without `code_quality` config. SpecKit-style requirement-quality checklist — duplicates planning-stage work.
*Source*: doc/fab-spec/TEMPLATES.md
*Updated by*: 260215-r8k3-DEV-1024-code-quality-layer

### Sub-Agent Over Inline Review
**Decision**: Review validation is dispatched to a sub-agent in a separate execution context, replacing inline review by the applying agent.
**Why**: Same-context review is fundamentally limited by shared cognitive biases. The sub-agent provides a fresh perspective — it has no shared context with the applying agent beyond the explicitly provided artifacts.
**Rejected**: Multiple inline review passes (still shares context), external review tool integration (too prescriptive, not portable).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### Priority-Based Comment Triage
**Decision**: The applying agent triages review comments by severity (must-fix / should-fix / nice-to-have) rather than implementing all of them.
**Why**: Prevents infinite rework loops over diminishing-return suggestions. Must-fix items ensure correctness; nice-to-have items allow pragmatic completion.
**Rejected**: Fix all comments (leads to infinite loops on style disagreements), ignore all non-critical (misses genuine should-fix quality issues).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### fab-ff Gains Auto-Loop with Interactive Fallback
**Decision**: `/fab-ff` auto-loops between apply and review (up to 3 cycles) before falling back to interactive rework. Previously, fab-ff always presented interactive rework immediately on failure.
**Why**: Sub-agent review enables tighter automated feedback cycles. The interactive fallback preserves fab-ff's semi-interactive character — the user is never locked out of control.
**Rejected**: Keep fab-ff fully interactive (wastes the fresh-context benefit on simple fixes), make fab-ff fully autonomous like fab-fff (loses the semi-interactive identity).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### Review Failure Offers Multiple Re-Entry Points
**Decision**: On review failure, the agent presents three options (fix code, revise tasks, revise spec) and the user chooses where to loop back.
**Why**: Not all review failures are implementation bugs. Some require revisiting upstream artifacts. Giving the user explicit choice prevents the agent from guessing wrong about where the problem originated.
**Rejected**: Always looping back to apply — misses cases where the spec was wrong.
*Source*: doc/fab-spec/SKILLS.md

### Hydrate Semantically, Not by Delta Markers
**Decision**: The agent compares `spec.md` against existing memory files to determine what's new, changed, or removed. No ADDED/MODIFIED/REMOVED markers in the spec.
**Why**: The spec reads as a straightforward requirements document. Delta markers would clutter the spec and couple it to the hydration mechanism.
**Rejected**: Explicit delta markers — clutters specs, requires discipline to maintain, fragile to editing.
*Source*: doc/fab-spec/TEMPLATES.md

### Concurrent Change Warning on Hydrate
**Decision**: Before hydrating, scan for other active changes that reference the same memory files and warn the user.
**Why**: Hydration updates the memory files, which may invalidate assumptions in other active changes. The warning prompts re-review rather than allowing silent drift.
**Rejected**: Blocking hydrate if concurrent changes exist — too restrictive, especially for independent changes that happen to touch the same domain.
*Source*: doc/fab-spec/SKILLS.md

### Hydrate is a Pipeline Stage, Archive is Not
**Decision**: Memory hydration (`hydrate`) is a tracked pipeline stage; folder housekeeping (`/fab-archive`) is a standalone skill.
**Why**: Memory hydration is the logical completion of the agent's work — it closes the feedback loop from implementation back to memory files. Folder housekeeping is a user-triggered cleanup action with no bearing on artifact quality.
**Rejected**: Both as pipeline stages — would add a 7th stage for marginal benefit. Neither as pipeline stages — would lose the memory hydration automation.
*Introduced by*: 260213-jc0u-split-archive-hydrate

### Restore as Subcommand, Not Separate Skill
**Decision**: Archive restore is a subcommand of `/fab-archive` (`/fab-archive restore <name>`), not a separate `/fab-restore` skill.
**Why**: Archive and restore are paired inverse operations. Grouping them under the same skill maintains conceptual cohesion and avoids skill proliferation. Users naturally look for restore under the archive command.
**Rejected**: Separate `/fab-restore` skill — adds a new top-level command for a narrow, complementary operation.
*Introduced by*: 260214-v7k3-archive-restore-mode

### fab-archive Clears Pointer Conditionally
**Decision**: `/fab-archive` only clears `fab/current` when the archived change is the active one.
**Why**: If archiving a non-active change (via change-name argument), clearing the pointer would disrupt the user's active work context.
**Rejected**: Always clear — would lose active change context when archiving a different change. Never clear — would leave stale pointer after archiving the active change.
*Introduced by*: 260213-jc0u-split-archive-hydrate

### Two-Tier PR Templates with Type Resolution
**Decision**: `/git-pr` resolves PR type via a three-step chain (explicit argument → intake inference → diff inference) and uses two template tiers: Tier 1 (fab-linked) for feat/fix/refactor with intake/spec blob URLs; Tier 2 (lightweight) for docs/test/ci/chore with explicit "no design artifacts" note. PR titles always get a conventional-commits prefix.
**Why**: Lightweight changes (chores, CI tweaks, doc fixes) don't warrant a full fab change folder. Blob URLs fix broken relative links (which resolved against main where change files don't exist). Type prefix signals to reviewers what to expect before opening the PR.
**Rejected**: Single template with optional links (broken links, no reviewer signal), branch prefix convention for type (branch names are change names, not type-encoded), prompting for type (breaks /git-pr's "no questions" contract).
*Introduced by*: 260225-54vl-smart-git-pr-category-taxonomy

### Execution Stage Reset Preserves Task Checkboxes
**Decision**: `/fab-continue apply` re-runs apply behavior starting from the first unchecked task. It does NOT uncheck all tasks.
**Why**: Task checkboxes reflect actual implementation progress. Silently unchecking them would discard valid work. Review rework (Option 1: "Fix code") handles targeted unchecking with `<!-- rework: reason -->` annotations.
**Rejected**: Resetting all checkboxes on apply reset — too destructive, discards completed work.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Review Active Triggers Forward Progression
**Decision**: When the active stage is `review`, `/fab-continue` runs the review behavior (advancing toward hydrate), not re-review. Re-review is available via `/fab-continue review` reset.
**Why**: The normal flow always advances. `review: active` means "review needs to run"; `review: done` means "review passed, hydrate is next." This avoids ambiguity about whether the command should redo or advance.
**Rejected**: Having review active trigger re-review — conflicts with the forward-progression model.
*Introduced by*: 260212-a4bd-unify-fab-continue

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260227-8q33-richer-git-pr-output | 2026-02-27 | `/git-pr` Tier 1 Context table expanded: added Confidence (`{score} / 5.0`) and Pipeline (`done` stages joined with `→`) rows. Fixed Intake/Spec from sibling cells in one row to separate `Field \| Detail` rows. Spec row omitted (not empty) when `spec.md` absent. Reads `confidence.score` and `progress` from `.status.yaml`. Tier 2 unchanged. |
| 260226-6boq-event-driven-stageman | 2026-02-26 | Status mutations updated: replaced `transition`/`set-state` with event commands (`start`, `advance`, `finish`, `reset`, `fail`). Driver parameter now optional (skills always pass it). |
| 260226-3g6f-git-branch-non-interactive-rename | 2026-02-26 | `/git-branch` non-interactive: replaced 3-option interactive menu with deterministic upstream-tracking logic. Local-only branches renamed (`git branch -m`); tracked branches get new branch (`git checkout -b`). Removed "Adopt this branch" concept. Added `renamed from {old}` and `created, leaving {old} intact` report verbs. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | `fab-status.md` updated progress symbols: replaced `—` skipped with `◷` ready. `/fab-ff` review failure behavior changed from interactive fallback to stop after 3 cycles. |
| 260227-gasp-consolidate-status-field-naming | 2026-02-27 | `/git-pr` reads issues via `stageman.sh get-issues` (replaces direct `issue_id` read), uses `stageman.sh add-pr` (replaces `ship`), PR title supports multiple space-joined issues, commit message updated, `.shipped` sentinel → `.pr-done`. `/fab-new` uses `stageman.sh add-issue` (replaces raw `yq`). `/fab-archive` cleans `.pr-done` instead of `.shipped`. |
| 260226-tnr8-coverage-scoring-change-types | 2026-02-26 | `/git-pr` type resolution chain extended from 3 to 4 steps: explicit argument → read `change_type` from `.status.yaml` → infer from intake → infer from diff. Step 2 (new) reads `change_type` written by `/fab-new` at creation time, avoiding re-inference. Resolution source tracking updated to include `status`. |
| 260225-54vl-smart-git-pr-category-taxonomy | 2026-02-25 | `/git-pr` gains PR type system: 7 types (feat, fix, refactor, docs, test, ci, chore) from Conventional Commits. New Step 0 resolves type via three-step chain (explicit arg → intake keyword inference → diff path inference). Step 3c rewritten with two-tier templates: Tier 1 (fab-linked) shows summary/changes/context with blob URL links; Tier 2 (lightweight) shows auto-generated summary with "no design artifacts" note. PR titles prefixed with `{type}: `. Blob URLs resolve against feature branch instead of main, fixing broken relative links. PR Type Reference table added. |
| 260224-1jkh-smart-resolve-and-pr-summary | 2026-02-25 | `changeman.sh resolve` gains single-change guessing fallback when `fab/current` is missing/empty (enumerate valid change folders, return if exactly one). `/git-pr` Step 3c enhanced with intake-aware PR generation: resolves active change, reads `intake.md`, derives PR title from H1 (strips `Intake: ` prefix), generates body with Summary/Changes/Context sections and links to intake + optional spec. Falls back to `gh pr create --fill` when resolution fails or no intake exists. |
| 260224-vx4k-decouple-git-from-fab-switch | 2026-02-24 | `/git-pr` enhanced with Step 1b (branch mismatch nudge — non-blocking note when current branch doesn't match active change, using prefix-stripped comparison) and enhanced Step 2 branch guard (suggests `/git-branch` when on main with active change). New `/git-branch` command documented in change-lifecycle.md. |
| 260222-trdc-git-pr-shipped-sentinel-and-status-commit | 2026-02-22 | `/git-pr` now performs a second commit+push after recording shipped URL to `.status.yaml` (Step 4b), then writes a `.shipped` sentinel file as its final action (Step 4c). Step 4b guards for "nothing to commit" via `git diff --cached --quiet`. The "already shipped" early-exit path now includes Steps 4–4c. Sentinel is gitignored, written unconditionally. Pipeline orchestrator (`run.sh`) polls sentinel existence instead of `stageman is-shipped`. |
| 260222-s90r-add-shipped-tracking | 2026-02-22 | Updated `/git-pr` skill to record PR URLs via `stageman.sh ship` after PR creation (graceful degradation when no active change). Updated `_preamble.md` state table: hydrate row now routes to `/git-pr` as default, `/fab-archive` as alternative. Updated `changeman.sh` `default_command` for hydrate from `/fab-archive` to `/git-pr`. |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble from `_context.md` to `_preamble.md`. Updated references in status mutations overview and sub-agent context sections. |
| 260217-eywl-fix-stageman-skill-path-refs | 2026-02-18 | All `lib/stageman.sh` references in skill files updated to repo-root-relative `fab/.kit/scripts/lib/stageman.sh`. Also fixed 2 short-form `lib/preflight.sh` references in `_preamble.md` and `fab-status.md`. Memory file references updated to match. |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | All `Next:` lines in execution skills (`/fab-continue`, `/fab-archive`) now derived from canonical state table in `_preamble.md`. Removed hardcoded suggestions from review pass verdict and archive mode output. `/fab-archive` restore without `--switch` now uses activation preamble before state-derived commands. |
| 260216-gqpp-DEV-1040-code-review-loop | 2026-02-16 | Review dispatched to sub-agent in separate execution context for all three pipeline skills. Structured findings with three-tier priority (must-fix / should-fix / nice-to-have). `/fab-ff` gains auto-loop (3 cycles) with interactive fallback; `/fab-fff` retains auto-loop with bail. Comment triage by applying agent. Escalation rule after 2 consecutive fix-code. |
| 260216-knmw-DEV-1030-swap-ff-fff-review-rework | 2026-02-16 | Swapped pipeline invocation note: `/fab-ff` now presents interactive rework on review failure; `/fab-fff` now uses autonomous rework with bounded retry (3-cycle cap, escalation after 2 consecutive fix-code) |
| 260215-237b-DEV-1027-redefine-ff-fff-scope | 2026-02-16 | Updated pipeline invocation note: `/fab-fff` now presents interactive rework on review failure, `/fab-ff` now bails immediately (swapped from previous behavior) |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` stage/artifact to `intake` throughout — stage identifiers, artifact filenames, YAML keys, prose references |
| 260215-r8k3-DEV-1024-code-quality-layer | 2026-02-15 | Added Pattern Extraction to Apply (naming, error handling, structure, utilities), expanded per-task guidance to 7-step sequence, added code quality check as Review step 6, added optional pattern capture to Hydrate step 5, updated "Checklist Tests Implementation Fidelity" design decision to include code quality |
| 260214-r7k3-stageman-yq-metrics | 2026-02-14 | Added `driver` parameter requirement to status mutations overview. Added `log-review` calls to review pass/fail behavior. Stage metrics side-effects documented as automatic |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_stageman.sh` → `lib/stageman.sh` in status mutations overview |
| 260214-w3r8-stageman-write-api | 2026-02-14 | All execution-stage `.status.yaml` transitions now use `_stageman.sh` CLI commands instead of direct file edits |
| 260214-eikh-consistency-fixes | 2026-02-14 | Verified cf13 (contradictory fab-status.sh/stageman.sh changelog entries) — already resolved by prior changes. No behavioral modifications. |
| 260214-r8kv-docs-skills-housekeeping | 2026-02-14 | Removed `fab-status.sh` references from changelog entries (updated to reference `/fab-status` skill instead) |
| 260214-v7k3-archive-restore-mode | 2026-02-14 | Added restore mode to `/fab-archive` — moves archived changes back to active, removes index entry, optional `--switch` flag. Idempotent and resumable. Added Restore as Subcommand design decision. |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Replaced Archive Behavior with Hydrate Behavior (steps 1-4 only, change folder stays). Added `/fab-archive` as standalone housekeeping skill. Updated overview, design decisions. |
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | Execution skills now inherit optional `[change-name]` argument via `/fab-continue` preflight override; `/fab-status` also accepts change-name override directly |
| 260212-a4bd-unify-fab-continue | 2026-02-12 | Restructured: apply, review, and archive behavior now accessed via `/fab-continue` instead of standalone skills. Updated all section headings, requirements, and cross-references |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Updated checklist path references from `checklists/quality.md` to `checklist.md` in `/fab-review` and `/fab-archive` |
| 260212-bk1n-rework-fab-ff-archive | 2026-02-12 | Added note that `/fab-ff` and `/fab-fff` invoke execution skills internally as part of their full-pipeline behavior |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated stage references from proposal/specs to intake/spec |
| 260211-endg-add-created-by-field | 2026-02-11 | `/fab-status` now displays `Created by:` line when `created_by` field is present in `.status.yaml` |
| 260210-7wxx-add-specs-index-context-loading | 2026-02-10 | Added `docs/specs/index.md` to context loading for all three execution skills, aligning with the always-load protocol in `_preamble.md` |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Added archive index maintenance step to `/fab-archive` — creates/updates `fab/changes/archive/index.md` with searchable change summaries |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Removed auto-guess soft gate from `/fab-apply` — replaced by confidence gating on `/fab-fff` |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added auto-guess soft gate to `/fab-apply` (subsequently removed by 260208-k3m7-add-fab-fff) |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| — | 2026-02-07 | Generated from doc/fab-spec/ (SKILLS.md, TEMPLATES.md) |
