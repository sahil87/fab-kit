# Spec: Redesign Hooks Strategy

**Change**: 260306-6bba-redesign-hooks-strategy
**Created**: 2026-03-06
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/setup.md`

## Non-Goals

- User-configured stage hooks — enforcement stays in `project/*` files
- Language-specific templates — fab-kit stays language-neutral
- Absorbing pipeline orchestrator yq usage (~40 calls) — separate concern
- Removing bookkeeping instructions from skills — hooks are additive, skills keep instructions for agent-agnostic portability

## Hooks: PostToolUse Artifact Bookkeeping

### Requirement: PostToolUse Hook Script

The kit SHALL provide a single hook script `fab/.kit/hooks/on-artifact-write.sh` registered for both `PostToolUse Write` and `PostToolUse Edit` Claude Code hook events. The script SHALL detect fab artifact writes and trigger bookkeeping commands automatically.

#### Scenario: Intake Write Triggers Type Inference and Scoring

- **GIVEN** the agent writes or edits `fab/changes/{name}/intake.md`
- **WHEN** the PostToolUse hook fires
- **THEN** the hook runs `fab status set-change-type <change> <inferred-type>` using keyword matching on the file content
- **AND** the hook runs `fab score --stage intake <change>`

#### Scenario: Spec Write Triggers Confidence Scoring

- **GIVEN** the agent writes or edits `fab/changes/{name}/spec.md`
- **WHEN** the PostToolUse hook fires
- **THEN** the hook runs `fab score <change>`

#### Scenario: Tasks Write Triggers Task Count

- **GIVEN** the agent writes `fab/changes/{name}/tasks.md`
- **WHEN** the PostToolUse hook fires
- **THEN** the hook runs `fab status set-checklist <change> total <N>` where N is the count of `- [ ]` lines in the file

#### Scenario: Checklist Write Triggers Checklist Metadata

- **GIVEN** the agent writes `fab/changes/{name}/checklist.md`
- **WHEN** the PostToolUse hook fires
- **THEN** the hook runs `fab status set-checklist <change> generated true`
- **AND** the hook runs `fab status set-checklist <change> total <N>` where N is the count of `- [ ]` and `- [x]` lines
- **AND** the hook runs `fab status set-checklist <change> completed 0`

#### Scenario: Non-Artifact Write Is Ignored

- **GIVEN** the agent writes a file that does not match `fab/changes/*/intake.md|spec.md|tasks.md|checklist.md`
- **WHEN** the PostToolUse hook fires
- **THEN** the hook exits 0 immediately (fast path, no-op)

#### Scenario: Edit Event Uses File Path Only

- **GIVEN** the agent edits a fab artifact via the Edit tool
- **WHEN** the PostToolUse hook fires with `tool_name: Edit`
- **THEN** the hook uses `tool_input.file_path` to match and derive the change name
- **AND** for bookkeeping that needs file content (intake type inference, tasks count), the hook reads the file from disk

### Requirement: Hook Reliability Properties

The hook script MUST exit 0 always — bookkeeping failures SHALL NOT interrupt the agent. All fab CLI commands invoked by the hook MUST be idempotent. The hook SHOULD return `additionalContext` in stdout JSON to inform the agent of what was auto-handled (e.g., `"Bookkeeping: score 4.2/5.0, type: refactor"`).

#### Scenario: Fab CLI Unavailable

- **GIVEN** `fab/.kit/bin/fab` is not found or not executable
- **WHEN** the hook fires
- **THEN** the hook exits 0 without running any commands

#### Scenario: Change Resolution Fails

- **GIVEN** the file path matches an artifact pattern but the change folder cannot be resolved
- **WHEN** the hook attempts to derive the change name
- **THEN** the hook exits 0 without running bookkeeping commands

#### Scenario: Hook Returns Context

- **GIVEN** the hook successfully runs bookkeeping commands
- **WHEN** processing completes
- **THEN** the hook writes a JSON object to stdout with an `additionalContext` field summarizing what was done

### Requirement: Hook Does Not Replace Skills

Skills SHALL keep their existing bookkeeping instructions unchanged. The hook supplements them as a reliability layer — it catches what the agent forgets. This preserves agent-agnostic portability: non-Claude-Code agents (Codex, Gemini CLI, Cursor) continue to work with skill-instructed bookkeeping only. Since all bookkeeping commands are idempotent, both the hook and the skill running the same command produces no conflict.

#### Scenario: Both Hook and Skill Run Same Command

- **GIVEN** the agent writes `spec.md` and the PostToolUse hook fires
- **AND** the skill also instructs the agent to run `fab score <change>`
- **WHEN** both the hook and the agent run `fab score <change>`
- **THEN** the result is identical to running it once (idempotent)

#### Scenario: Non-Claude-Code Agent

- **GIVEN** an agent (Codex, Gemini CLI) without Claude Code hooks
- **WHEN** the agent follows skill instructions to run bookkeeping commands
- **THEN** bookkeeping works correctly (current behavior, unchanged)

### Requirement: Intake Type Inference in Hook

The hook SHALL infer the change type from intake content using keyword matching (case-insensitive, first match wins): fix/bug/broken/regression → `fix`, refactor/restructure/consolidate/split/rename → `refactor`, docs/document/readme/guide → `docs`, test/spec/coverage → `test`, ci/pipeline/deploy/build → `ci`, chore/cleanup/maintenance/housekeeping → `chore`, otherwise → `feat`.

#### Scenario: Refactor Keyword Detected

- **GIVEN** `intake.md` contains the word "Redesign" in its content
- **WHEN** the hook processes the intake artifact
- **THEN** the hook runs `fab status set-change-type <change> refactor`

## Go Runtime: `fab runtime` Subcommands

### Requirement: Runtime Set-Idle Command

The Go binary SHALL provide `fab runtime set-idle <change>` which writes the `agent.idle_since` timestamp (Unix epoch seconds) to `.fab-runtime.yaml` at the repository root, keyed by the change's full folder name.

#### Scenario: Set Idle for Active Change

- **GIVEN** a valid change reference
- **WHEN** `fab runtime set-idle <change>` is executed
- **THEN** `.fab-runtime.yaml` at the repo root is updated with `{folder_name}.agent.idle_since` set to the current Unix timestamp
- **AND** the file is created with `{}` as initial content if it does not exist

#### Scenario: Runtime File Already Exists

- **GIVEN** `.fab-runtime.yaml` exists with idle state for another change
- **WHEN** `fab runtime set-idle <change>` is executed for a different change
- **THEN** the existing change's entry is preserved and the new change's entry is added

### Requirement: Runtime Clear-Idle Command

The Go binary SHALL provide `fab runtime clear-idle <change>` which deletes the `agent` block from `.fab-runtime.yaml` for the resolved change's folder name.

#### Scenario: Clear Idle for Active Change

- **GIVEN** `.fab-runtime.yaml` contains an `agent.idle_since` entry for the change
- **WHEN** `fab runtime clear-idle <change>` is executed
- **THEN** the `agent` block for that change is removed from `.fab-runtime.yaml`
- **AND** other changes' entries are preserved

#### Scenario: No Runtime File

- **GIVEN** `.fab-runtime.yaml` does not exist
- **WHEN** `fab runtime clear-idle <change>` is executed
- **THEN** the command exits 0 (no-op)

### Requirement: Change Resolution

Both `fab runtime` subcommands SHALL accept the standard `<change>` argument (4-char ID, folder name substring, or full folder name) and resolve it to the full folder name via the existing resolve logic.

#### Scenario: Resolve by 4-Char ID

- **GIVEN** a change with folder `260306-6bba-redesign-hooks-strategy`
- **WHEN** `fab runtime set-idle 6bba` is executed
- **THEN** the entry is keyed by `260306-6bba-redesign-hooks-strategy`

## Hooks: Existing Hook Migration

### Requirement: Stop Hook Uses Runtime Commands

`fab/.kit/hooks/on-stop.sh` SHALL use `fab runtime set-idle` instead of `yq` for writing idle state. The script MUST NOT depend on `yq`. The `command -v yq` guard SHALL be removed.

#### Scenario: yq Not Installed

- **GIVEN** `yq` is not installed on the system
- **WHEN** the Stop hook fires
- **THEN** the hook successfully records idle state via `fab runtime set-idle`

#### Scenario: Migrated Stop Hook Structure

- **GIVEN** the Stop hook script
- **WHEN** inspecting the script
- **THEN** it contains `"$fab_cmd" runtime set-idle "$change_folder"` instead of `yq -i` commands
- **AND** the total line count is approximately 15 lines (down from 31)

### Requirement: SessionStart Hook Uses Runtime Commands

`fab/.kit/hooks/on-session-start.sh` SHALL use `fab runtime clear-idle` instead of `yq` for clearing idle state. The script MUST NOT depend on `yq`. The `command -v yq` guard SHALL be removed.

#### Scenario: Migrated SessionStart Hook Structure

- **GIVEN** the SessionStart hook script
- **WHEN** inspecting the script
- **THEN** it contains `"$fab_cmd" runtime clear-idle "$change_folder"` instead of `yq -i del()` commands
- **AND** the `runtime_file` variable and file existence check are removed

## Sync: Hook Registration with Matchers

### Requirement: Matcher Support in Sync Script

`fab/.kit/sync/5-sync-hooks.sh` SHALL support registering hooks with tool-name matchers for PostToolUse events. The `map_event()` function SHALL return event+matcher pairs.

#### Scenario: PostToolUse Hook Registration

- **GIVEN** `fab/.kit/hooks/on-artifact-write.sh` exists
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** `.claude/settings.local.json` contains two PostToolUse entries: one with `"matcher": "Write"` and one with `"matcher": "Edit"`, both pointing to `on-artifact-write.sh`

#### Scenario: Existing Hooks Unaffected

- **GIVEN** existing Stop and SessionStart hooks are registered
- **WHEN** `5-sync-hooks.sh` runs with the new PostToolUse hook
- **THEN** the Stop and SessionStart entries are preserved (matcher remains `""`)
- **AND** the new PostToolUse entries are added

### Requirement: Filename-to-Event Mapping

The `map_event()` function SHALL be extended to support a naming convention where filenames encode the event and optional matcher. For the new hook, the mapping SHALL be:

| Filename | Event | Matcher |
|----------|-------|---------|
| `on-session-start.sh` | `SessionStart` | `""` |
| `on-stop.sh` | `Stop` | `""` |
| `on-posttooluse-write-artifact-write.sh` | `PostToolUse` | `Write` |
| `on-posttooluse-edit-artifact-write.sh` | `PostToolUse` | `Edit` |

Alternatively, if a single script handles both matchers, the sync script MAY use a mapping table approach instead of pure filename convention.

#### Scenario: Single Script, Multiple Matchers

- **GIVEN** `on-artifact-write.sh` is a single script for both Write and Edit matchers
- **WHEN** `5-sync-hooks.sh` processes hooks
- **THEN** two `.claude/settings.local.json` entries are created, both pointing to the same script but with different matchers

## Constitution: Wording Update

### Requirement: Section I Wording

Constitution §I ("Pure Prompt Play") SHALL change the sentence "All workflow logic MUST live in markdown skill files and shell scripts" to "All workflow logic MUST live in markdown skill files and scripts". The version SHALL be bumped and `Last Amended` updated.

#### Scenario: Constitution Wording Updated

- **GIVEN** the current Constitution §I contains "shell scripts"
- **WHEN** the change is applied
- **THEN** §I reads "markdown skill files and scripts"
- **AND** the Governance line reflects a version bump

## Setup: Remove Language Detection

### Requirement: Remove Phase 1b-lang

`fab/.kit/skills/fab-setup.md` SHALL remove the entire `#### 1b-lang. Language Detection and Convention Inference` section and its subsections. The bootstrap flow proceeds directly from 1b (constitution) to 1b2 (context.md).

#### Scenario: Clean Setup Without Language Detection

- **GIVEN** a new project with `Cargo.toml` in the repo root
- **WHEN** `/fab-setup` runs the bootstrap flow
- **THEN** no language detection occurs
- **AND** `fab/project/context.md` is created from the scaffold template (not language-inferred content)

### Requirement: Renumber Bootstrap Steps

After removing 1b-lang, the steps 1b2, 1b3, 1b4 SHALL be renumbered to 1c, 1d, 1e (or their current numbering adjusted to close the gap). Subsequent steps SHALL be renumbered accordingly.

#### Scenario: Step Numbering After Removal

- **GIVEN** fab-setup.md with 1b-lang removed
- **WHEN** examining the bootstrap sequence
- **THEN** steps flow continuously with no gaps (1a → 1b → 1c → 1d → ...)

## Deprecated Requirements

### Template-Driven Language Detection (fab-setup Phase 1b-lang)

**Reason**: Language-specific customization rejected — fab-kit stays language-neutral per user discussion. Detection logic has no purpose without templates. Agent-inferred conventions (260306-143f) superseded by full removal.
**Migration**: Projects that want language-specific conventions can add them manually to `fab/project/*` files.

## Design Decisions

1. **Hooks as reliability layer, not replacement**: Skills keep all bookkeeping instructions for agent-agnostic portability. The PostToolUse hook catches what the agent forgets. Doubling up is harmless due to idempotency.
   - *Why*: Claude Code hooks are platform-specific — no equivalent in Codex, Gemini CLI, Cursor. Constitution §I requires agent-agnostic portability.
   - *Rejected*: Removing bookkeeping from skills (breaks non-Claude-Code agents), making hooks the sole mechanism (ties fab-kit to Claude Code).

2. **Single script for Write and Edit matchers**: `on-artifact-write.sh` handles both PostToolUse Write and PostToolUse Edit events with the same logic — detect artifact path, run bookkeeping.
   - *Why*: The bookkeeping logic is identical regardless of whether the artifact was created or edited. Separate scripts would duplicate code.
   - *Rejected*: Separate scripts per matcher (unnecessary duplication).

3. **`fab runtime` absorbs yq from hooks only**: The Go binary gets `set-idle` and `clear-idle` subcommands to replace yq usage in hooks. Pipeline orchestrator keeps yq (~40 calls) as a separate concern.
   - *Why*: Hook yq usage is a dependency liability (silent failure on missing yq). Pipeline orchestrator is a heavy manifest parser where absorbing yq is a much larger change.
   - *Rejected*: Absorbing all yq usage in one change (too large), keeping yq in hooks (unnecessary external dependency).

4. **Hook returns additionalContext**: The PostToolUse hook writes JSON to stdout with `additionalContext` to inform the agent what bookkeeping was performed, reducing redundant agent-directed bookkeeping in practice.
   - *Why*: PostToolUse hooks can return `additionalContext` in their stdout JSON. The agent sees this context and can skip redundant commands, though it's not required to.
   - *Rejected*: Silent hooks (agent has no visibility into what was auto-handled).

5. **Stale changes already deleted**: The template change folders (4vj0, qg80, rwt1) and shk2 no longer exist in the repository (deleted in prior changes). No deletion action needed.
   - *Why*: Git history confirms these folders are absent from both the worktree and main branch.

6. **Full removal of language detection over keeping agent inference**: Removes 1b-lang entirely from fab-setup.md rather than keeping the agent-inferred approach from 260306-143f.
   - *Why*: Language-specific customization was rejected as a concept — fab-kit stays language-neutral. Users who want conventions can add them manually. Agent inference (143f) was a stepping stone that this change supersedes.
   - *Rejected*: Keeping agent-inferred conventions (still adds language-specific content to `project/*` files).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Hooks are for kit-internal mechanics only — user enforcement stays in project/* files | Confirmed from intake #1 — user explicitly rejected user-configured hooks | S:95 R:85 A:95 D:95 |
| 2 | Certain | PostToolUse (Write/Edit) is the only new hook event | Confirmed from intake #2 — all 18 events assessed | S:95 R:85 A:95 D:95 |
| 3 | Certain | Constitution §I changes "shell scripts" → "scripts" | Confirmed from intake #4 — user explicitly requested | S:95 R:90 A:95 D:95 |
| 4 | Certain | Stage transitions remain agent-directed, not hookable | Confirmed from intake #5 — advance/finish/fail/reset are intentional decisions | S:90 R:85 A:90 D:90 |
| 5 | Certain | `fab status advance` stays in fab-new (not hookable) | Confirmed from intake #6 — must happen after SRAD questions | S:90 R:80 A:90 D:90 |
| 6 | Certain | Existing hooks migrate from yq to `fab runtime` subcommands | Confirmed from intake #7 — eliminates silent-fail yq dependency | S:90 R:85 A:90 D:90 |
| 7 | Certain | Single script handles both Write and Edit matchers | Confirmed from intake #8 — same logic, no reason for separate scripts | S:85 R:90 A:85 D:90 |
| 8 | Certain | Hooks are a reliability layer, not a replacement | Confirmed from intake #9 — agent-agnostic portability, idempotent doubling | S:95 R:90 A:95 D:90 |
| 9 | Certain | Stale changes (4vj0, qg80, rwt1, shk2) already deleted | Confirmed — git inspection shows folders absent from worktree and main | S:95 R:95 A:95 D:95 |
| 10 | Confident | Hook returns `additionalContext` to inform agent | Confirmed from intake #10 — PostToolUse hooks support this mechanism | S:80 R:90 A:80 D:85 |
| 11 | Confident | Pipeline orchestrator keeps yq — separate concern | Confirmed from intake #11 — ~40 uses, much larger change | S:80 R:85 A:85 D:80 |
| 12 | Confident | Remove language detection from fab-setup.md entirely | Upgraded from intake #12 — supersedes 260306-143f agent inference approach | S:85 R:80 A:80 D:80 |
| 13 | Confident | Sync script uses mapping table for multi-matcher hooks | Codebase signal — `map_event()` case statement needs extending, mapping table is cleaner for event+matcher pairs | S:75 R:90 A:80 D:75 |

13 assumptions (9 certain, 4 confident, 0 tentative, 0 unresolved).
