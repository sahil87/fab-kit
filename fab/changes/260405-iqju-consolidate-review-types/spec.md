# Spec: Consolidate Review Types

**Change**: 260405-iqju-consolidate-review-types
**Created**: 2026-04-05
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/configuration.md`

## Non-Goals

- Changes to the `fab` Go binary or any CLI command signatures — no new subcommands, no flag changes
- Changes to the rework loop mechanics (max cycles, escalation rule, pass/fail determination) — behavior of the loop is unchanged
- Changes to `/git-pr-review` Phase 1 (human/bot review comment processing) — only Phase 2 is affected
- Auto-disabling the outward sub-agent based on config — always-on is the specified behavior

---

## `_review.md` Shared Skill File

### Requirement: New Shared Skill File

`src/kit/skills/_review.md` SHALL be created as an internal, non-user-invocable skill file following the same structural pattern as `_generation.md`. It SHALL define both the inward and outward sub-agent dispatch behaviors so that orchestrators (`fab-continue`, `fab-ff`, `fab-fff`) reference it by name rather than inlining review logic.

The frontmatter MUST include:
- `name: _review`
- `description:` describing it as shared review logic
- `user-invocable: false`
- `disable-model-invocation: true`
- `metadata.internal: true`

#### Scenario: File Structure Mirrors `_generation.md`
- **GIVEN** the `_generation.md` file defines shared artifact generation logic with a YAML frontmatter declaring it internal and non-user-invocable
- **WHEN** `_review.md` is created
- **THEN** its frontmatter contains `user-invocable: false`, `disable-model-invocation: true`, and `metadata.internal: true`
- **AND** the file body opens with a `> This file defines shared review logic...` block quote citing which orchestrators reference it

### Requirement: Inward Sub-Agent Behavior (Unchanged)

`_review.md` SHALL define the inward sub-agent dispatch behavior — spec/tasks/checklist validation — with content identical to the current Review Behavior in `fab-continue.md`. No behavioral change.

The inward sub-agent:
- Is dispatched via the Agent tool (`subagent_type: "general-purpose"`)
- Receives standard subagent context files per `_preamble.md` § Standard Subagent Context, plus change-specific files: `spec.md`, `tasks.md`, `checklist.md`, relevant source files, and target memory file(s) from `docs/memory/`
- Performs all six validation checks (tasks complete, quality checklist, run affected tests, spot-check spec, memory drift check, code quality check)
- Returns structured findings: must-fix / should-fix / nice-to-have with file:line references

#### Scenario: Inward Sub-Agent Dispatched with Correct Context
- **GIVEN** the review stage is active and all tasks are marked `[x]`
- **WHEN** the review behavior executes per `_review.md`
- **THEN** the inward sub-agent is dispatched via the Agent tool with standard subagent context plus `spec.md`, `tasks.md`, `checklist.md`, source files, and memory files

#### Scenario: Inward Sub-Agent Produces Three-Tier Output
- **GIVEN** the inward sub-agent completes its validation checks
- **WHEN** findings are returned to the orchestrator
- **THEN** findings are structured as must-fix / should-fix / nice-to-have with file:line references where applicable

### Requirement: Outward Sub-Agent Behavior (New)

`_review.md` SHALL define an outward sub-agent dispatch that performs a holistic diff review with full repository access.

The outward sub-agent:
- Is dispatched via the Agent tool (`subagent_type: "general-purpose"`)
- Receives: the diff of all changed files (`git diff` against the base), the list of changed file paths, and standard subagent context files per `_preamble.md` § Standard Subagent Context
- Has full tool access (Read, Bash, Agent) and MAY read any file in the repo to explore context
- Uses a **Codex → Claude cascade**: attempts Codex first (via `codex` CLI); if Codex is unavailable or fails, falls back to Claude as the reviewer. The cascade gracefully no-ops if neither tool is available — this is not an error condition
- Returns structured findings in the same severity format: must-fix / should-fix / nice-to-have

The outward sub-agent prompt instructs it to look for:
1. Interface contract violations (types, return values, API shape mismatches)
2. Inconsistencies with documented patterns in memory files (e.g., naming conventions, error handling style)
3. Missing cross-references (memory files or specs that should reference the changed behavior)
4. Behavioral regressions not caught by the inward reviewer (issues requiring full-repo context to detect)
5. Structural issues visible only with broad codebase context (e.g., duplication of existing utilities)

The outward sub-agent is **always on** — there is no config flag to disable it. The Codex→Claude cascade gracefully handles unavailability, so always attempting incurs no harm.

#### Scenario: Outward Sub-Agent Uses Full Repo Access
- **GIVEN** the review stage is active
- **WHEN** the outward sub-agent is dispatched
- **THEN** it receives the diff and changed file list, and is permitted to read any file in the repo
- **AND** it is not limited to the files changed in the apply stage

#### Scenario: Codex Unavailable — Cascade Falls Back to Claude
- **GIVEN** `codex` CLI is not found (via `command -v codex` or equivalent)
- **WHEN** the outward sub-agent attempts the Codex→Claude cascade
- **THEN** Claude is attempted as the fallback reviewer
- **AND** the absence of Codex is not treated as an error

#### Scenario: Both Tools Unavailable — Graceful No-Op
- **GIVEN** neither `codex` nor `claude` CLI is available
- **WHEN** the outward sub-agent attempts the Codex→Claude cascade
- **THEN** the outward sub-agent returns an empty findings set (no must-fix, no should-fix, no nice-to-have)
- **AND** the review stage continues normally (outward sub-agent absence does not block the review)

#### Scenario: Outward Sub-Agent Always On
- **GIVEN** `review_tools` in `fab/project/config.yaml` has any value or is absent entirely
- **WHEN** the review stage executes
- **THEN** the outward sub-agent is always dispatched regardless of config values

### Requirement: Parallel Dispatch of Both Sub-Agents

Both sub-agents (inward and outward) SHALL be dispatched in parallel. The orchestrator waits for both to return before merging findings.

#### Scenario: Parallel Dispatch Returns Combined Findings
- **GIVEN** the review stage executes per `_review.md`
- **WHEN** both sub-agents are dispatched
- **THEN** they run concurrently (not sequentially)
- **AND** the orchestrator collects findings from both before proceeding to the verdict

### Requirement: Merged Findings Feed a Single Rework Loop

Findings from both sub-agents SHALL be merged and fed into the same must-fix/should-fix/nice-to-have rework loop. The pass/fail determination uses the merged set: if any must-fix finding exists (from either sub-agent), the review fails.

#### Scenario: Must-Fix from Outward Sub-Agent Triggers Rework
- **GIVEN** the inward sub-agent reports no must-fix findings
- **AND** the outward sub-agent reports one must-fix finding
- **WHEN** findings are merged
- **THEN** the review fails (must-fix found in merged set)
- **AND** the rework loop is triggered

#### Scenario: No Must-Fix from Either — Review Passes
- **GIVEN** neither sub-agent reports must-fix findings
- **WHEN** findings are merged
- **THEN** the review passes
- **AND** the rework loop is not triggered

---

## `fab-continue.md` Review Behavior Update

### Requirement: Delegate to `_review.md`

`fab-continue.md` Review Behavior SHALL be updated to delegate to `_review.md` rather than inlining review logic. The inline sub-agent dispatch description SHALL be replaced with a reference: "Follow **Review Behavior** (`_review.md`)."

This follows the same delegation pattern used for planning stages: `| spec | Spec Generation Procedure (_generation.md) |`.

#### Scenario: fab-continue Dispatches Review via `_review.md` Reference
- **GIVEN** the review stage is reached via `/fab-continue`
- **WHEN** the Review Behavior section is executed
- **THEN** the agent reads `_review.md` for the authoritative dispatch instructions
- **AND** does not inline its own sub-agent dispatch logic

### Requirement: Stage Dispatch Table Updated

The Stage dispatch table in `fab-continue.md` (the table mapping stages to procedures) SHALL be updated to add a row: `| review | Review Behavior (_review.md) |`.

#### Scenario: Stage Table Maps Review to `_review.md`
- **GIVEN** the stage dispatch table in fab-continue.md
- **WHEN** the review stage row is read
- **THEN** the procedure references `_review.md` (not inline text)

---

## `git-pr-review.md` Phase 2 Simplification

### Requirement: Phase 2 Is Copilot-Only

Phase 2 of `git-pr-review.md` SHALL be simplified to Copilot-only. The Codex and Claude tool invocations SHALL be removed from Phase 2.

**New Phase 2 behavior**:

1. Attempt: `gh pr edit {number} --add-reviewer copilot`
2. **On success**: poll up to 10 minutes for the Copilot review to appear
   - Poll interval: 30 seconds
   - Check via: `gh pr view --json reviews`
   - When a review from Copilot appears: proceed to Step 3+ (existing processing logic)
   - If 10 minutes elapse with no review: print `Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.` and STOP (clean finish, no error)
3. **On failure** (non-zero exit from the `gh pr edit` command): print `No automated reviewer available. Run /git-pr-review when reviews are added.` and STOP (clean finish, no error)

#### Scenario: Copilot Request Succeeds — Poll for Review
- **GIVEN** Phase 2 runs (no existing reviews found in Phase 1)
- **WHEN** `gh pr edit {number} --add-reviewer copilot` exits 0
- **THEN** the skill polls `gh pr view --json reviews` every 30 seconds
- **AND** polls for up to 10 minutes (20 attempts max)
- **AND** when a Copilot review appears, falls through to Step 3 to process comments

#### Scenario: Copilot Review Appears During Poll
- **GIVEN** polling is in progress after a successful Copilot review request
- **WHEN** `gh pr view --json reviews` returns a review from the `copilot` reviewer
- **THEN** polling stops and execution proceeds to Step 3 (Fetch Comments)

#### Scenario: Copilot Review Does Not Appear Within 10 Minutes
- **GIVEN** polling is in progress after a successful Copilot review request
- **WHEN** 10 minutes elapse without a Copilot review appearing
- **THEN** the skill prints `Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.` and stops
- **AND** the stop is a clean finish (no error status, no `fail` stage event)

#### Scenario: Copilot Request Fails — Clean Finish
- **GIVEN** Phase 2 runs (no existing reviews found in Phase 1)
- **WHEN** `gh pr edit {number} --add-reviewer copilot` exits non-zero
- **THEN** the skill prints `No automated reviewer available. Run /git-pr-review when reviews are added.`
- **AND** execution stops with a clean finish (no error)

### Requirement: Step 2a and Step 2b Removed

Step 2a (Context Enrichment) and Step 2b (Local Review Output Posting) SHALL be removed from `git-pr-review.md`. These steps were used only for local tool invocations (Codex and Claude), which no longer run in Phase 2.

#### Scenario: No Context Enrichment in Phase 2
- **GIVEN** Phase 2 executes
- **WHEN** the Copilot request is made
- **THEN** no diff, PR description, or test output is constructed or passed to any tool
- **AND** no local review output is posted as a PR comment

### Requirement: `--tool` Flag Validation Updated

The `--tool` flag valid values SHALL be updated to reflect the simplified Phase 2. Valid values: `copilot` only (Codex and Claude are removed from Phase 2). The `--tool codex` and `--tool claude` options SHALL be removed.

<!-- assumed: Removing codex/claude from --tool valid values is the correct interpretation — they no longer run locally in Phase 2, so exposing them as valid --tool options would be misleading. If --tool needs to remain for forward-compatibility, this can be revised. -->

#### Scenario: `--tool copilot` Still Valid
- **GIVEN** `/git-pr-review --tool copilot` is invoked
- **WHEN** the flag is parsed
- **THEN** only the Copilot path is attempted (unchanged behavior)

#### Scenario: `--tool codex` Now Invalid
- **GIVEN** `/git-pr-review --tool codex` is invoked
- **WHEN** the flag is parsed
- **THEN** the skill prints an error: `Invalid tool: codex. Valid values: copilot.` and STOP

### Requirement: `review_tools` Config Consumed by Phase 2 Is Now Copilot-Only

The `review_tools` config block read during Phase 2 SHALL only honor the `copilot` key. If `review_tools.copilot` is `false`, Phase 2 skips the Copilot request and immediately prints `No automated reviewer available. Run /git-pr-review when reviews are added.` and stops.

The `codex` and `claude` keys (if still present in a user's config) SHALL be ignored by Phase 2.

#### Scenario: Copilot Disabled in Config — Clean Skip
- **GIVEN** `review_tools.copilot: false` in `fab/project/config.yaml`
- **WHEN** Phase 2 runs
- **THEN** the skill prints `No automated reviewer available. Run /git-pr-review when reviews are added.` and stops without attempting any tool

---

## `fab-ff.md` and `fab-fff.md` Review Stage Dispatch Update

### Requirement: Reference `_review.md` as Authoritative Source

The review stage dispatch description in `fab-ff.md` and `fab-fff.md` SHALL be updated to note that `_review.md` is the authoritative source of review behavior. Specifically, the Step 6 (Review) description SHALL clarify: "The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there."

This is a documentation/pointer update only. The functional dispatch ("Dispatch `/fab-continue` as subagent — Review Behavior") is unchanged.

#### Scenario: fab-ff Step 6 References `_review.md`
- **GIVEN** `fab-ff.md` Step 6 is read
- **WHEN** the review sub-agent dispatch is described
- **THEN** the text notes `_review.md` as the authoritative review behavior source

#### Scenario: fab-fff Step 6 References `_review.md`
- **GIVEN** `fab-fff.md` Step 6 is read
- **WHEN** the review sub-agent dispatch is described
- **THEN** the text notes `_review.md` as the authoritative review behavior source

---

## `config.yaml` `review_tools` Schema Update

### Requirement: Remove `codex` and `claude` Keys

The `review_tools` schema in `fab/project/config.yaml` SHALL be simplified to a single key:

```yaml
review_tools:
    copilot: true/false
```

The `codex` and `claude` keys SHALL be removed. This is a **breaking schema change** for projects that currently have these keys set. A migration file handles key removal.

The meaning of `review_tools.copilot` is unchanged: when `true` (or absent), Copilot is attempted in Phase 2; when `false`, Phase 2 is skipped.

When the entire `review_tools` block is absent, Copilot defaults to enabled (existing default behavior preserved).

#### Scenario: Config Has Only `copilot` Key After Migration
- **GIVEN** an existing config with `review_tools.codex: true` and `review_tools.claude: true`
- **WHEN** the migration runs
- **THEN** both `codex` and `claude` keys are removed
- **AND** `review_tools.copilot` retains its existing value

#### Scenario: New Project Gets Copilot-Only Block
- **GIVEN** a new project is initialized via `/fab-setup`
- **WHEN** `review_tools` is written to `config.yaml`
- **THEN** only the `copilot: true` key is present (no `codex`, no `claude`)

#### Scenario: Absent `review_tools` Block Defaults to Copilot Enabled
- **GIVEN** `fab/project/config.yaml` has no `review_tools` key
- **WHEN** Phase 2 of `git-pr-review` runs
- **THEN** the Copilot reviewer is attempted (defaults to enabled)

---

## Migration File

### Requirement: Migration Removes `codex` and `claude` Keys

A migration file SHALL be created at `src/kit/migrations/` to remove the `review_tools.codex` and `review_tools.claude` keys from `fab/project/config.yaml`. The migration SHALL use direct `yq` key-removal (no dry-run step).

The migration file name SHALL follow the existing version-range convention (e.g., `1.2.0-to-1.3.0.md` or the next applicable version pair), consistent with `src/kit/migrations/` naming.

<!-- clarified: Migration file named 1.3.0-to-1.4.0.md — confirmed: fab_version is 1.3.0, the latest migration in src/kit/migrations/ is 1.1.0-to-1.2.0.md (no 1.2.0-to-1.3.0 migration exists), so the next version is 1.4.0. Rename trivially if version differs at ship time. -->

Migration structure:

**Pre-check**:
1. Confirm `fab/project/config.yaml` exists — if not, skip migration entirely
2. Check if `review_tools` key exists — if not, print `review_tools block absent, nothing to migrate.` and stop

**Changes**:
1. Remove `review_tools.codex` key (if present):
   ```bash
   yq -i 'del(.review_tools.codex)' fab/project/config.yaml
   ```
2. Remove `review_tools.claude` key (if present):
   ```bash
   yq -i 'del(.review_tools.claude)' fab/project/config.yaml
   ```

**Verification**:
1. `fab/project/config.yaml` does NOT contain `review_tools.codex`
2. `fab/project/config.yaml` does NOT contain `review_tools.claude`
3. `fab/project/config.yaml` DOES contain `review_tools.copilot` (if `review_tools` block is present)

#### Scenario: Migration Removes Both Keys
- **GIVEN** `fab/project/config.yaml` contains `review_tools.codex: true` and `review_tools.claude: true`
- **WHEN** the migration runs
- **THEN** both keys are deleted via `yq -i 'del(...)'`
- **AND** `review_tools.copilot` is unaffected

#### Scenario: Migration Is Idempotent
- **GIVEN** the migration has already been applied (keys are absent)
- **WHEN** the migration runs again
- **THEN** `yq del` on absent keys is a no-op — no error, no change

#### Scenario: No `review_tools` Block — Skip Gracefully
- **GIVEN** `fab/project/config.yaml` has no `review_tools` key
- **WHEN** the migration runs
- **THEN** the migration prints a skip message and exits without modifying the file

---

## Spec Files Update

### Requirement: SPEC-fab-continue.md Updated

`docs/specs/skills/SPEC-fab-continue.md` SHALL be updated to:
- Add a new `_review` sub-agent row to the Sub-agents table (inward + outward, spawned in parallel)
- Update the Review stage diagram to show dual parallel sub-agents dispatched via `_review.md`
- Note that Review Behavior is delegated to `_review.md`

#### Scenario: Spec Reflects Dual Sub-Agent Dispatch
- **GIVEN** `SPEC-fab-continue.md` is read
- **WHEN** the Review stage section is inspected
- **THEN** both the inward and outward sub-agents are shown in the flow diagram
- **AND** `_review.md` is cited as the authoritative source

### Requirement: SPEC-git-pr-review.md Updated

`docs/specs/skills/SPEC-git-pr-review.md` SHALL be updated to:
- Update the Configuration section to show `review_tools` with `copilot` key only (remove `codex` and `claude` rows from the example YAML)
- Update the Phase 2 flow diagram to show Copilot-only (remove Codex and Claude branches)
- Update the Review Request Cascade table to show only Copilot
- Remove the Step 2a and Step 2b blocks from the flow diagram
- Update `--tool` valid values to `copilot` only

#### Scenario: SPEC Reflects Copilot-Only Phase 2
- **GIVEN** `SPEC-git-pr-review.md` is read
- **WHEN** the Phase 2 flow is inspected
- **THEN** only the Copilot tool is shown (Codex and Claude branches absent)

### Requirement: SPEC-fab-ff.md and SPEC-fab-fff.md Updated

`docs/specs/skills/SPEC-fab-ff.md` and `docs/specs/skills/SPEC-fab-fff.md` SHALL be updated to note `_review.md` as the authoritative review behavior source in the Step 6 (Review) flow description and/or sub-agents table.

#### Scenario: SPEC-fab-ff Reflects `_review.md` Source
- **GIVEN** `SPEC-fab-ff.md` is read
- **WHEN** the Step 6 Review sub-agent section is inspected
- **THEN** `_review.md` is cited as the source of review behavior

---

## Deprecated Requirements

### Codex/Claude Local Tool Invocations in Phase 2

**Reason**: Phase 2 of `/git-pr-review` previously ran Codex and Claude as local tools to generate review output and post it as a PR comment. These tools are now relocated to the outward sub-agent in `_review.md`, where they run pre-ship with full repo access. Running them again post-ship in git-pr-review is redundant and architecturally incorrect.

**Migration**: Codex and Claude review capabilities are preserved in the outward sub-agent in `_review.md`, which runs during the `review` stage (pre-ship). Phase 2 of git-pr-review is replaced by Copilot-only behavior.

### `review_tools.codex` and `review_tools.claude` Config Keys

**Reason**: These keys controlled whether Codex and Claude were used in git-pr-review Phase 2. With Phase 2 becoming Copilot-only, there is nothing left to configure for those tools. The outward sub-agent in `_review.md` uses Codex→Claude as a cascade fallback, but this is not user-configurable (always-on).

**Migration**: `src/kit/migrations/{version}.md` removes both keys via `yq del`.

---

## Design Decisions

### Outward Sub-Agent Always On (No Config Flag)

**Decision**: The outward sub-agent in `_review.md` runs unconditionally — no `review_tools.codex` or `review_tools.claude` flag controls it.

**Why**: The Codex→Claude cascade gracefully no-ops if neither tool is available. Adding a config flag would add config surface area (another key users must know about) without proportional benefit. The cascade is a fallback mechanism, not a preference — users don't need to toggle it per-project.

**Rejected**: Adding an `outward_review: true/false` flag to `config.yaml` — unnecessary complexity given the graceful no-op behavior.

### Copilot-Only Phase 2 with Poll-and-Wait

**Decision**: Phase 2 waits for Copilot's review to appear (up to 10 minutes, polling every 30 seconds) rather than stopping immediately after the review request.

**Why**: Immediately stopping after requesting Copilot would require the user to re-invoke `/git-pr-review` manually. Waiting in-process allows a single pipeline run (e.g., via `/fab-fff`) to process the Copilot review without user re-intervention when Copilot responds quickly. The 10-minute timeout provides a reasonable upper bound.

**Rejected**: Stop-immediately pattern (original Copilot behavior in the old cascade) — requires user re-invocation and interrupts the `/fab-fff` pipeline. Abandoned in favor of poll-and-wait per intake specification.

### `_review.md` Extraction Following `_generation.md` Pattern

**Decision**: Extract review behavior into a shared `_review.md` skill file, following the same pattern as `_generation.md`.

**Why**: `_generation.md` proved that centralizing shared logic in an internal partial makes it easy to update and maintain — one file, one place. The same reasoning applies to review behavior. Both `fab-continue` and the orchestrators (`fab-ff`, `fab-fff`) reference `_review.md`, ensuring review behavior is authoritative in one location.

**Rejected**: Keeping review logic inlined in `fab-continue.md` — creates divergence risk when behavior needs updating, since orchestrators would need their own copies or cross-references to the inline version.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Inward review sub-agent behavior is unchanged | Confirmed from intake #1 — explicitly stated as "No change" | S:95 R:95 A:95 D:95 |
| 2 | Certain | Outward sub-agent uses Agent tool (general-purpose), not Skill tool | Confirmed from intake #2 — consistent with preamble orchestrator pattern | S:90 R:80 A:90 D:90 |
| 3 | Certain | git-pr-review Phase 1 (human/bot reviews) is unchanged | Confirmed from intake #3 — explicitly stated as unchanged | S:95 R:95 A:95 D:95 |
| 4 | Certain | Canonical skill sources are in src/kit/skills/, not .claude/skills/ | Confirmed from intake #4 — context.md is explicit | S:95 R:95 A:95 D:95 |
| 5 | Certain | Both sub-agents run in parallel during the review stage | Confirmed from intake #5 — user-clarified | S:95 R:70 A:80 D:75 |
| 6 | Certain | Migration is direct yq key-removal of review_tools.codex and review_tools.claude | Confirmed from intake #6 — user confirmed after example review | S:95 R:65 A:85 D:80 |
| 7 | Certain | Review Behavior extracted to _review.md; fab-continue/fab-ff/fab-fff all reference it | Confirmed from intake #7 — user explicitly chose _review.md extraction pattern | S:95 R:70 A:65 D:70 |
| 8 | Certain | Outward sub-agent is always on (no config flag to disable) | Confirmed from intake #8 — user confirmed; cascade gracefully no-ops if tools absent | S:95 R:60 A:70 D:55 |
| 9 | Certain | Migration is direct key-removal (not dry-run-first) | Confirmed from intake #9 — user confirmed; trivial operation does not warrant dry-run | S:95 R:55 A:60 D:55 |
| 10 | Certain | `--tool` flag valid values reduced to `copilot` only (codex, claude removed) | Clarified — Codex and Claude are removed from Phase 2; the spec body fully specifies the `--tool codex` error message and the Non-Goals section confirms no CLI binary changes (flag is skill-level only). | S:95 R:65 A:80 D:75 |
| 11 | Certain | Phase 2 polls up to 10 min (30s interval) when Copilot request succeeds | Clarified — spec body and scenarios explicitly specify 30s interval, 10-minute cap (20 attempts), confirmed from intake as authoritative. | S:95 R:70 A:85 D:90 |
| 12 | Certain | Migration file named 1.3.0-to-1.4.0.md | Clarified — fab_version is 1.3.0; latest migration in src/kit/migrations/ is 1.1.0-to-1.2.0.md (no 1.2.0-to-1.3.0 exists), confirming 1.4.0 as the next target version. | S:95 R:80 A:75 D:80 |
| 13 | Certain | When Copilot times out after 10 min, stage completes as done (clean finish, not fail) | Clarified — spec scenarios explicitly state "clean finish (no error status, no `fail` stage event)"; consistent with intake. | S:95 R:70 A:80 D:85 |

13 assumptions (13 certain, 0 confident, 0 tentative, 0 unresolved).
