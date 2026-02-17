# Spec: Fix Stageman Skill Path References

**Change**: 260217-eywl-fix-stageman-skill-path-refs
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Skill Files: Path Reference Consistency

### Requirement: Repo-Root-Relative Stageman Paths

All references to `stageman.sh` in skill markdown files under `fab/.kit/skills/` SHALL use the repo-root-relative form `fab/.kit/scripts/lib/stageman.sh`, consistent with how `preflight.sh` is referenced in `_context.md`.

The short-form `lib/stageman.sh` (relative to `fab/.kit/scripts/`) SHALL NOT appear in any skill file.

#### Scenario: Stageman references in fab-continue.md

- **GIVEN** the file `fab/.kit/skills/fab-continue.md` contains 10 occurrences of `lib/stageman.sh`
- **WHEN** the path update is applied
- **THEN** all 10 occurrences are replaced with `fab/.kit/scripts/lib/stageman.sh`
- **AND** no other content in the file is modified

#### Scenario: Stageman references in fab-ff.md

- **GIVEN** the file `fab/.kit/skills/fab-ff.md` contains 8 occurrences of `lib/stageman.sh`
- **WHEN** the path update is applied
- **THEN** all 8 occurrences are replaced with `fab/.kit/scripts/lib/stageman.sh`

#### Scenario: Stageman references in fab-fff.md

- **GIVEN** the file `fab/.kit/skills/fab-fff.md` contains 9 occurrences of `lib/stageman.sh`
- **WHEN** the path update is applied
- **THEN** all 9 occurrences are replaced with `fab/.kit/scripts/lib/stageman.sh`

#### Scenario: Stageman references in remaining files

- **GIVEN** the following files each contain short-form stageman references:
  - `fab/.kit/skills/fab-clarify.md` (1 occurrence)
  - `fab/.kit/skills/_generation.md` (3 occurrences)
  - `fab/.kit/skills/fab-status.md` (1 occurrence)
- **WHEN** the path update is applied
- **THEN** all occurrences in each file are replaced with `fab/.kit/scripts/lib/stageman.sh`

### Requirement: Repo-Root-Relative Preflight Paths

All references to `preflight.sh` in skill markdown files SHALL use the repo-root-relative form `fab/.kit/scripts/lib/preflight.sh`. Two remaining short-form references SHALL be updated:

- `fab/.kit/skills/_context.md` line 21: `lib/preflight.sh` in the note about init checks
- `fab/.kit/skills/fab-status.md` line 41: `lib/preflight.sh` in the tool usage description

Existing full-form references (`_context.md` line 27, `fab-archive.md` line 36, `fab-status.md` line 38) SHALL NOT be modified.

#### Scenario: Preflight short-form in _context.md

- **GIVEN** `fab/.kit/skills/_context.md` line 21 contains `lib/preflight.sh`
- **WHEN** the path update is applied
- **THEN** it is replaced with `fab/.kit/scripts/lib/preflight.sh`
- **AND** the full-form reference on line 27 remains unchanged

#### Scenario: Preflight short-form in fab-status.md

- **GIVEN** `fab/.kit/skills/fab-status.md` line 41 contains `lib/preflight.sh`
- **WHEN** the path update is applied
- **THEN** it is replaced with `fab/.kit/scripts/lib/preflight.sh`
- **AND** the full-form reference on line 38 remains unchanged

### Requirement: No Shell Script Modifications

Shell scripts (`stageman.sh`, `preflight.sh`, `changeman.sh`, `calc-score.sh`) SHALL NOT be modified. These scripts resolve paths internally via `$0`/`BASH_SOURCE` and are unaffected by how skill files reference them.

#### Scenario: Scripts remain untouched

- **GIVEN** the shell scripts under `fab/.kit/scripts/lib/`
- **WHEN** the path update is applied across skill files
- **THEN** no files under `fab/.kit/scripts/` are modified

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fab/.kit/scripts/lib/stageman.sh` as the canonical path | Confirmed from intake #1. Matches `_context.md` preflight convention and `settings.local.json` allowlist | S:95 R:95 A:95 D:95 |
| 2 | Certain | Only modify skill markdown files, not shell scripts | Confirmed from intake #2. Scripts resolve paths internally via `$0` | S:90 R:95 A:95 D:95 |
| 3 | Certain | Also fix short-form `lib/preflight.sh` references | Upgraded from intake Confident #3. Verified: 2 short-form refs exist in `_context.md` and `fab-status.md` | S:95 R:90 A:90 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
