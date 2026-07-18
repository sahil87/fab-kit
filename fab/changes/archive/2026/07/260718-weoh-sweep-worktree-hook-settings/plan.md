# Plan: Sweep Worktree Hook Settings

**Change**: 260718-weoh-sweep-worktree-hook-settings
**Intake**: `intake.md`

## Requirements

### Migration: Worktree hook-settings sweep

#### R1: New migration file with standard structure
The change SHALL add a new migration `src/kit/migrations/2.15.7-to-2.15.8.md` following the standard Summary / Pre-check / Changes / Verification structure used by sibling migrations (`2.13.6-to-2.14.0.md`, `2.10.1-to-2.11.0.md`, `1.4.0-to-1.5.0.md`).

- **GIVEN** the migration catalog in `src/kit/migrations/`
- **WHEN** `2.15.7-to-2.15.8.md` is authored
- **THEN** it carries the four canonical sections (Summary, Pre-check, Changes, Verification) with the tone, sentinel/pre-check conventions, and print-line style of the neighboring settings-editing migrations

#### R2: Target-set definition by prefix + legacy shims
The migration SHALL define a stale hook action as one whose `command` starts with the prefix `fab hook ` (prefix match, NOT an enumeration of the four known subcommands), OR matches the legacy script-shim forms `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-<script>.sh` and `bash fab/.kit/hooks/on-<script>.sh`.

- **GIVEN** a `.claude/settings.local.json` carrying `fab hook <anything>` or a legacy `on-*.sh` shim under any event
- **WHEN** the migration evaluates the target set
- **THEN** every such action matches (any `fab hook <x>` variant, plus both legacy shim spellings), and non-matching custom commands do not

#### R3: All-worktrees sweep including the main checkout
The migration SHALL enumerate all worktrees via `git worktree list --porcelain` and sweep each — the main checkout is the first porcelain entry and MUST be covered (it is the live poison, since Claude Code resolves settings through worktrees to the main repo root). If the current directory is not a git repository, the migration SHALL handle only the current directory and skip the sibling sweep (mirroring `2.13.6-to-2.14.0` §2).

- **GIVEN** a repo with a main checkout plus sibling worktrees, some carrying stale hook entries
- **WHEN** the migration runs
- **THEN** every worktree's `.claude/settings.local.json` (including the main checkout's) is inspected and cleaned; a non-git directory sweeps the current directory only

#### R4: Sentinel-guarded, idempotent, preserve-non-fab-hooks editing
The migration SHALL be sentinel-guarded (skip with `Skipped: no stale fab hook entries found in any worktree.` when no worktree carries a target entry), idempotent (re-run is a complete no-op), and MUST preserve editing discipline: mixed entries keep their custom commands; an entry whose `hooks[]` becomes empty is removed; an emptied event array is left empty or the key omitted; the `hooks` object is never deleted; all non-hook top-level keys are preserved verbatim; writes are atomic (temp file in the same directory + rename); one report line is printed per cleaned worktree (`Removed stale fab hook entries from <worktree-path>/.claude/settings.local.json.`).

- **GIVEN** a settings file mixing a target action with an unrelated custom command
- **WHEN** the migration cleans it
- **THEN** only the target action is dropped, the custom command survives, non-hook top-level keys are untouched, and the write is atomic
- **AND GIVEN** the migration is re-run after a successful sweep
- **THEN** the sentinel trips and nothing is changed

#### R5: No binary pre-check, no status/git changes
The migration SHALL require no binary capability pre-check (pure prompt/JSON-edit logic), make no `.status.yaml` change, no `fab/` data change, and no commit (the edited `.claude/settings.local.json` files are gitignored — unlike `2.15.1-to-2.15.2`).

- **GIVEN** the migration's Changes section
- **WHEN** it is authored
- **THEN** it contains no binary pre-check, no `.status.yaml` edit, and no `git add`/`git commit` step

### Versioning

#### R6: VERSION bump 2.15.7 → 2.15.8
`src/kit/VERSION` SHALL be bumped `2.15.7` → `2.15.8` (patch). The migration filename FROM SHALL equal the real current released VERSION; if `src/kit/VERSION` is not `2.15.7` at apply time, the filename is re-slotted per the `2.11.0-to-2.12.0` slot-note precedent and the re-slot noted in the result.

- **GIVEN** `src/kit/VERSION` currently contains `2.15.7`
- **WHEN** the bump is applied
- **THEN** `src/kit/VERSION` contains `2.15.8` and the migration is named `2.15.7-to-2.15.8.md` (FROM = current VERSION, satisfying the range rule `FROM <= local < TO`)

### Spec claim accuracy

#### R7: Update stale "done by 2.13.6-to-2.14.0" spec sentences
The two spec sentences that state hook-entry cleanup "is done by the `2.13.6-to-2.14.0` migration" SHALL be extended to also name the new sweep migration, and the whole `docs/specs/` occurrence class SHALL be swept per code-quality.md § Sibling & Mirror Sweeps.

- **GIVEN** `docs/specs/architecture.md` (~line 448) and `docs/specs/skills.md` (~line 157) assert cleanup "is done by" the 2.13.6-to-2.14.0 migration
- **WHEN** the sweep runs
- **THEN** both sentences (plus any other `docs/specs/` occurrence framing 2.13.6-to-2.14.0 as the resolution of lingering hook entries) name the new `2.15.7-to-2.15.8` worktree-sweep migration as the complement, and the incompleteness across worktrees is corrected

### Non-Goals

- **`docs/memory/` edits** — deferred to the hydrate stage (`distribution/migrations.md` catalog + Design Decision). Apply MUST NOT touch `docs/memory/`.
- **Skill source files (`src/kit/skills/*.md`)** — not changed, so no `SPEC-*.md` mirror obligation is triggered (per the intake's explicit exclusion). `src/kit/skills/_cli-fab.md:406` describes 2.13.6-to-2.14.0's own behavior accurately (not a cross-worktree completeness claim) and is left as-is.
- **Go code, `.status.yaml` schema, scaffold/fragment changes** — none.

### Design Decisions

1. **Prefix match over enumeration**: Match `fab hook ` as a prefix rather than listing the four known subcommands — *Why*: the entire command family was removed in 2.14.0, so any `fab hook <x>` is a cobra unknown-command error; enumeration could strand an unlisted variant — *Rejected*: enumerate `session-start|stop|user-prompt|artifact-write`.
2. **Sweep all worktrees including the main checkout**: `git worktree list --porcelain` (main = first entry) — *Why*: the version gate (`fab/.kit-migration-version`) is committed/repo-wide, so a current-checkout-only edit permanently strands sibling checkouts; the main checkout is the live poison via settings resolution — *Rejected*: current-checkout-only (what 2.11.0 and 2.13.6-to-2.14.0 §1 did, which is why this change exists).
3. **No commit**: the edited files are gitignored — *Why*: unlike `2.15.1-to-2.15.2` (which committed `fab/.fab-version`), nothing here lands in git — *Rejected*: a pathspec-scoped commit step.

## Tasks

### Phase 1: Version slot

- [x] T001 Verify `src/kit/VERSION` current content is `2.15.7`; if it differs, re-slot the migration filename (FROM = real current VERSION) per the `2.11.0-to-2.12.0` slot-note precedent and note the re-slot in the result <!-- R6 -->

### Phase 2: Core Implementation

- [x] T002 Author `src/kit/migrations/2.15.7-to-2.15.8.md` with the Summary / Pre-check / Changes / Verification structure, encoding: prefix `fab hook ` + legacy `on-*.sh` shim target set (R2), all-worktrees sweep incl. main checkout via `git worktree list --porcelain` + non-git-dir current-dir-only fallback (R3), sentinel/idempotency + preserve-non-fab-hooks + atomic write + per-worktree report line (R4), and the no-binary-precheck / no-status / no-commit constraints (R5) <!-- R1 -->
- [x] T003 Bump `src/kit/VERSION` `2.15.7` → `2.15.8` <!-- R6 -->

### Phase 3: Spec sweep

- [x] T004 [P] Extend the stale sentence in `docs/specs/architecture.md` (~line 448) to also name the `2.15.7-to-2.15.8` worktree-sweep migration <!-- R7 -->
- [x] T005 [P] Extend the stale sentence in `docs/specs/skills.md` (~line 157) to also name the new sweep migration <!-- R7 -->
- [x] T006 [P] Sweep the remaining `docs/specs/` class occurrences (`docs/specs/skills/SPEC-hooks.md`, `docs/specs/skills/SPEC-_cli-fab.md`) that frame 2.13.6-to-2.14.0 as the resolution of lingering hook entries, extending each to name the new migration <!-- R7 -->

### Phase 4: Verification

- [x] T007 Verify internal consistency: filename vs. range rule `FROM <= local < TO`; target-set spellings; the exact stale-entry JSON shapes from the intake; cross-references between the new migration and the sibling migrations; repo-wide grep confirms no un-swept `docs/specs/` occurrence of the stale completeness claim remains <!-- R1 R7 -->

## Execution Order

- T001 precedes T002/T003 (confirms the slot before authoring/bumping)
- T002, T003 sequential (same version concern); T004–T006 are `[P]` (independent files)
- T007 last (whole-change verification)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/migrations/2.15.7-to-2.15.8.md` exists with all four canonical sections (Summary, Pre-check, Changes, Verification)
- [x] A-002 R2: The migration's target set is defined as prefix `fab hook ` (not an enumeration) plus both legacy `on-*.sh` shim spellings (`bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-<script>.sh` and `bash fab/.kit/hooks/on-<script>.sh`)
- [x] A-003 R3: The migration enumerates all worktrees via `git worktree list --porcelain`, explicitly covers the main checkout (first entry), and falls back to current-directory-only for non-git dirs
- [x] A-004 R4: The migration is sentinel-guarded with the exact skip line, idempotent, preserves mixed/custom commands and non-hook top-level keys, never deletes the `hooks` object, uses atomic temp+rename writes, and prints one report line per cleaned worktree
- [x] A-005 R5: The migration has no binary pre-check, no `.status.yaml` change, and no commit step
- [x] A-006 R6: `src/kit/VERSION` contains `2.15.8` and the migration filename is `2.15.7-to-2.15.8.md` (FROM = current released VERSION)
- [x] A-007 R7: `docs/specs/architecture.md` and `docs/specs/skills.md` (and the swept SPEC-* files) name the new `2.15.7-to-2.15.8` migration alongside 2.13.6-to-2.14.0 for hook-entry cleanup

### Behavioral Correctness

- [x] A-008 R3: The migration text makes clear the main checkout is included and is the live poison (settings resolution through worktrees), distinguishing this sweep from 2.13.6-to-2.14.0 §1's current-checkout-only edit
- [x] A-009 R4: A mixed entry (target action + custom command) is documented to keep the custom command; an entry with emptied `hooks[]` is removed; an emptied event is left empty/omitted

### Scenario Coverage

- [x] A-010 R4: The Verification section documents the re-run-is-a-no-op sentinel path (doubling as the local validation path, since fab-kit's own repo was hand-cleaned)
- [x] A-011 R2: The migration includes the stale-shape JSON example (PostToolUse Write/Edit `fab hook artifact-write` + the three session events) matching the intake's example

### Edge Cases & Error Handling

- [x] A-012 R3: Non-git-directory handling (current directory only, sibling sweep skipped) is specified

### Code Quality

- [x] A-013 Pattern consistency: The new migration follows the structure, tone, sentinel conventions, and print-line style of the neighboring settings-editing migrations
- [x] A-014 No unnecessary duplication: The migration references the precedent migrations (`2.13.6-to-2.14.0`, `2.10.1-to-2.11.0`) rather than restating their full rationale
- [x] A-015 Markdown-only artifacts: All edits are plain-markdown/plain-text (no binary/proprietary formats), CommonMark-compliant
- [x] A-016 Migrations for user-data restructuring: The `.claude/settings.local.json` edit ships as a `src/kit/migrations/` file, not an ad-hoc script

### Documentation Accuracy & Cross References

- [x] A-017 Documentation accuracy: No `docs/specs/` occurrence of the stale "cleanup is done by 2.13.6-to-2.14.0" completeness claim remains un-updated
- [x] A-018 Cross references: The new migration's references to sibling migrations and the docs-confirmed settings-resolution behavior are accurate; the filename satisfies the range rule `FROM <= local < TO`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The new migration's target set supersets the settings edits of `2.10.1-to-2.11.0.md` and `2.13.6-to-2.14.0.md` §1, but shipped migrations are append-only version-chain history — they must remain in `src/kit/migrations/` for users at intermediate versions and are not deletion candidates.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Version slot `2.15.7-to-2.15.8` confirmed (VERSION is `2.15.7`); no re-slot needed | Read `src/kit/VERSION` = `2.15.7` at apply time; the free-slot check (`ls src/kit/migrations/`) shows no `2.15.7-to-2.15.8.md` | S:95 R:90 A:95 D:95 |
| 2 | Confident | Spec sweep class = the two explicit `docs/specs/` targets plus `SPEC-hooks.md` and `SPEC-_cli-fab.md` (the `docs/specs/` files framing 2.13.6-to-2.14.0 as the resolution of lingering hook entries) | Grep found these carrying the same "removes any lingering ... entries"/"is done by" completeness framing; code-quality.md § Sibling & Mirror Sweeps puts aggregate/mirror specs in the class | S:70 R:80 A:75 D:70 |
| 3 | Confident | Do NOT edit `src/kit/skills/_cli-fab.md:406` despite it mentioning 2.13.6-to-2.14.0 | The intake explicitly excludes skill files (to avoid the SPEC-mirror obligation) and that sentence describes 2.13.6's own per-checkout behavior accurately, not a cross-worktree completeness claim; leaving it avoids a skill↔spec mirror trigger the intake deliberately scoped out | S:75 R:75 A:70 D:65 |

3 assumptions (1 certain, 2 confident, 0 tentative).
