# Spec: Rename _context.md to _preamble.md

**Change**: 260221-5tj7-rename-context-to-preamble
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/clarify.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing any content within the preamble file itself — scope is rename + reference updates only
- Updating archived change artifacts — archives are historical records per Constitution II

## Skill Preamble: File Rename

### Requirement: Rename preamble file

The shared skill preamble file SHALL be renamed from `fab/.kit/skills/_context.md` to `fab/.kit/skills/_preamble.md`. The file content SHALL remain unchanged.

#### Scenario: File exists at new path after rename
- **GIVEN** the file `fab/.kit/skills/_context.md` exists
- **WHEN** the rename is applied
- **THEN** `fab/.kit/skills/_preamble.md` exists with identical content
- **AND** `fab/.kit/skills/_context.md` no longer exists

## Skill Preamble: Reference Updates

### Requirement: Update preamble instruction line in skill files

Every skill file in `fab/.kit/skills/` that contains the instruction `Read and follow the instructions in fab/.kit/skills/_context.md before proceeding.` SHALL be updated to reference `fab/.kit/skills/_preamble.md` instead.

Affected files:
- `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`
- `fab-clarify.md`, `fab-switch.md`, `fab-setup.md`, `fab-status.md`
- `fab-archive.md`, `fab-discuss.md`
- `docs-hydrate-memory.md`, `docs-hydrate-specs.md`
- `_generation.md`, `internal-retrospect.md`, `internal-skill-optimize.md`

#### Scenario: Skill file references updated preamble path
- **GIVEN** a skill file contains `fab/.kit/skills/_context.md`
- **WHEN** the reference update is applied
- **THEN** all occurrences of `_context.md` in the instruction line are replaced with `_preamble.md`
- **AND** no other content in the skill file is modified

### Requirement: Update self-reference in preamble file

The preamble file (`_preamble.md`) contains a self-referencing blockquote instructing skill files to read it. This self-reference SHALL be updated from `_context.md` to `_preamble.md`.

#### Scenario: Preamble self-reference is consistent
- **GIVEN** `_preamble.md` contains a blockquote referencing `_context.md`
- **WHEN** the self-reference update is applied
- **THEN** the blockquote references `_preamble.md`

### Requirement: Update references in memory files

All live memory files under `docs/memory/` that reference `_context.md` by path SHALL be updated to reference `_preamble.md`.

#### Scenario: Memory file path references updated
- **GIVEN** a memory file contains a path reference to `_context.md`
- **WHEN** the reference update is applied
- **THEN** all path references to `_context.md` are replaced with `_preamble.md`
- **AND** the descriptive text around the reference (e.g., "shared preamble", "context loading convention") MAY be updated for consistency but is not required

### Requirement: Update references in spec files

All spec files under `docs/specs/` that reference `_context.md` by path SHALL be updated to reference `_preamble.md`.

#### Scenario: Spec file path references updated
- **GIVEN** a spec file contains a path reference to `_context.md`
- **WHEN** the reference update is applied
- **THEN** all path references to `_context.md` are replaced with `_preamble.md`

### Requirement: Update directory tree listing in kit-architecture memory

The `docs/memory/fab-workflow/kit-architecture.md` file contains a directory tree listing `_context.md` under `fab/.kit/skills/`. This listing SHALL be updated to show `_preamble.md`.

#### Scenario: Directory tree reflects renamed file
- **GIVEN** `kit-architecture.md` contains `├── _context.md` in the directory tree
- **WHEN** the tree update is applied
- **THEN** the tree shows `├── _preamble.md` with the same description

### Requirement: Preserve archived change artifacts

Archived change artifacts under `fab/changes/archive/` SHALL NOT be modified, even if they contain references to `_context.md`.

#### Scenario: Archive files untouched
- **GIVEN** archived change files reference `_context.md`
- **WHEN** the rename change is fully applied
- **THEN** all archived files retain their original `_context.md` references

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New name is `_preamble.md` | Confirmed from intake #1 — explicitly agreed, matches file's own heading | S:95 R:90 A:95 D:95 |
| 2 | Certain | Archives not updated | Confirmed from intake #2 — Constitution II: docs are source of truth, archives are historical | S:90 R:95 A:90 D:95 |
| 3 | Certain | No content changes to preamble file | Confirmed from intake #3 — rename + reference updates only | S:90 R:95 A:90 D:95 |
| 4 | Confident | Scaffold `context.md` unaffected | Confirmed from intake #4 — different file (project context template vs skill preamble) | S:80 R:90 A:85 D:85 |
| 5 | Certain | Shell scripts not affected | No shell script references `_context.md` — it's consumed only by agent-read markdown files | S:90 R:95 A:95 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
