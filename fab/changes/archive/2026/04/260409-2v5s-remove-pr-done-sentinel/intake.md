# Intake: Remove .pr-done Sentinel

**Change**: 260409-2v5s-remove-pr-done-sentinel
**Created**: 2026-04-09
**Status**: Draft

## Origin

> Discussion session: user asked "Check if we use .pr-done file anywhere" then "Can we remove it in favour of reading .status.yaml (via a fab command) directly?" Agent analyzed all 30 files referencing `.pr-done` and confirmed it is write-only — nothing reads it for logic. User confirmed removal.

Interaction mode: conversational (discuss → new). Key decision: `.status.yaml` via `fab status get-prs` is the authoritative source for "has a PR been created?" — the `.pr-done` sentinel is redundant.

## Why

The `.pr-done` file is a write-only sentinel: `/git-pr` creates it (step 4d) and `fab archive` deletes it, but nothing in between reads it to gate any behavior. It was introduced as a "race-free filesystem signal that all git operations are complete," but by the time it's written, `.status.yaml` already has the PR URL committed and pushed (steps 4a-4c). The file adds dead code paths, complicates archiving, and required a now-stale `.gitignore` entry. The 0.46.0-to-1.1.0 migration already strips `.pr-done` from `.gitignore` and deletes existing files, signaling intent to deprecate — this change finishes the job.

## What Changes

### 1. Remove `.pr-done` write from `/git-pr` skill

In `src/kit/skills/git-pr.md`, delete Step 4d ("Write PR Sentinel") entirely. The step currently writes `echo "$PR_URL" > "$change_dir/.pr-done"`. Steps 4a-4c (record PR URL via `fab status add-pr`, finish ship stage, commit+push `.status.yaml`) are unchanged and already provide the authoritative record.

### 2. Remove `.pr-done` cleanup from `fab archive` Go command

In `src/go/fab/internal/archive/archive.go`, remove the "Clean: delete .pr-done" logic (lines ~66-71). The archive YAML output currently includes a `clean` field (`removed` / `not_present`) — this field should be removed from the output entirely.

### 3. Update `/fab-archive` skill

In `src/kit/skills/fab-archive.md`, remove the "Clean" bullet from the command description and the `clean: removed` / `clean: not_present` rows from the report format table. Update the example output to remove the `Cleaned:` line.

### 4. Update `_cli-fab.md` command reference

In `src/kit/skills/_cli-fab.md`, update the `archive` row description from "Clean .pr-done, move to archive/, update index, clear pointer" to "Move to archive/, update index, clear pointer".

### 5. Update spec diagrams

In `docs/specs/skills/SPEC-git-pr.md`, remove step 4d from the flow diagram. In `docs/specs/skills/SPEC-fab-archive.md`, remove the `.pr-done` cleanup step from the flow diagram.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update `/git-pr` section to remove step 4d and `/fab-archive` section to remove clean step

## Impact

- **`src/kit/skills/git-pr.md`** — remove step 4d
- **`src/go/fab/internal/archive/archive.go`** — remove `.pr-done` cleanup logic
- **`src/kit/skills/fab-archive.md`** — remove clean step from docs and report
- **`src/kit/skills/_cli-fab.md`** — update archive command description
- **`docs/specs/skills/SPEC-git-pr.md`** — update flow diagram
- **`docs/specs/skills/SPEC-fab-archive.md`** — update flow diagram
- No migration needed — the 0.46.0-to-1.1.0 migration already handles cleanup of existing `.pr-done` files and `.gitignore` entries

## Open Questions

None — the discussion resolved all questions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.status.yaml` prs field is the authoritative PR record | `fab status add-pr` writes it in step 4a, before `.pr-done` would be written in step 4d — discussed and confirmed | S:95 R:90 A:95 D:95 |
| 2 | Certain | No migration needed for `.pr-done` removal | The 0.46.0-to-1.1.0 migration already deletes `.pr-done` files and strips `.gitignore` entries — verified in codebase | S:90 R:95 A:95 D:95 |
| 3 | Certain | Remove `clean` field from archive YAML output entirely | Nothing reads the `clean` field downstream — the skill just maps it to a report line | S:85 R:85 A:90 D:90 |
| 4 | Certain | Historical changelog entries in memory files are left as-is | Constitution says memory records what happened — past changes that referenced `.pr-done` are historical fact | S:90 R:95 A:90 D:95 |
| 5 | Confident | Archive Go test updates needed | `archive.go` likely has tests that assert `.pr-done` cleanup behavior — these need updating | S:70 R:85 A:70 D:85 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).