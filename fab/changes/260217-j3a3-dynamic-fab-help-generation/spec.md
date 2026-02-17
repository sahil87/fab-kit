# Spec: Dynamic Fab Help Generation

**Change**: 260217-j3a3-dynamic-fab-help-generation
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Adding a `group` field to skill frontmatter — grouping is a display concern owned by the help script
- Changing the workflow diagram or "Typical Flow" section — those are layout, not derived data
- Modifying any skill file frontmatter — the existing `name` and `description` fields are already sufficient

## Help Script: Dynamic Command Listing

### Requirement: Dynamic command extraction from skill frontmatter

The help script (`fab/.kit/scripts/fab-help.sh`) SHALL extract command names and descriptions from skill file YAML frontmatter at runtime instead of hardcoding them. The script SHALL source `fab/.kit/scripts/lib/frontmatter.sh` for the `frontmatter_field()` function.

#### Scenario: Normal help output

- **GIVEN** skill files exist in `fab/.kit/skills/` with valid `name` and `description` frontmatter fields
- **WHEN** the user runs `fab/.kit/scripts/fab-help.sh`
- **THEN** the output SHALL contain each skill's name (prefixed with `/`) and description, organized under group headings
- **AND** the output SHALL include the version header, workflow diagram, and "Typical Flow" footer unchanged from the current format

#### Scenario: Newly added skill appears automatically

- **GIVEN** a new skill file `fab/.kit/skills/fab-example.md` is added with `name: fab-example` and `description: "Does something useful."`
- **WHEN** the user runs `fab/.kit/scripts/fab-help.sh`
- **THEN** the output SHALL include `/fab-example` with its description under the default group
- **AND** no manual edit to `fab-help.sh` is required

### Requirement: Skill file filtering

The help script SHALL exclude files matching these patterns from the command listing:

1. Files prefixed with `_` (shared partials: `_context.md`, `_generation.md`)
2. Files prefixed with `internal-` (internal tooling: `internal-consistency-check.md`, `internal-retrospect.md`, `internal-skill-optimize.md`)

#### Scenario: Partials excluded

- **GIVEN** `fab/.kit/skills/_context.md` and `fab/.kit/skills/_generation.md` exist
- **WHEN** the help script scans skill files
- **THEN** neither file SHALL appear in the output

#### Scenario: Internal skills excluded

- **GIVEN** `fab/.kit/skills/internal-consistency-check.md` exists
- **WHEN** the help script scans skill files
- **THEN** it SHALL NOT appear in the output

### Requirement: Group assignment

The help script SHALL maintain a hardcoded mapping of skill names to display groups. The groups and their members SHALL be:

| Group | Skills |
|-------|--------|
| Start & Navigate | `fab-new`, `fab-switch`, `fab-status` |
| Planning | `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify` |
| Completion | `fab-archive` |
| Maintenance | `docs-hydrate-specs`, `docs-reorg-specs`, `docs-reorg-memory` |
| Setup | `fab-setup`, `fab-help`, `docs-hydrate-memory` |

Skills not present in any group SHALL appear under a "Other" group at the end. This handles future skills added to `fab/.kit/skills/` without requiring a help script update — they'll show up under "Other" until the mapping is updated.

#### Scenario: All current skills appear in assigned groups

- **GIVEN** the current set of 14 user-facing skill files exist (excluding `_*` and `internal-*`)
- **WHEN** the help script runs
- **THEN** each skill SHALL appear under its assigned group heading
- **AND** no "Other" group SHALL appear (all current skills have assignments)

#### Scenario: Unknown skill gets default group

- **GIVEN** a new skill `fab/.kit/skills/fab-example.md` exists and is not in the group mapping
- **WHEN** the help script runs
- **THEN** `/fab-example` SHALL appear under an "Other" group at the end of the commands section

### Requirement: Non-skill hardcoded entries

The help script SHALL include `fab-sync.sh` as a hardcoded entry in the "Setup" group with the description "Repair directories, symlinks, and agents (no LLM needed)". This entry is not derived from frontmatter because `fab-sync.sh` is a shell script, not a skill file.
<!-- clarified: fab-sync.sh description confirmed as-is -->

#### Scenario: fab-sync.sh appears in output

- **GIVEN** the help script runs
- **WHEN** the "Setup" group is rendered
- **THEN** it SHALL include a line for `fab-sync.sh` (without `/` prefix) alongside the dynamically-generated skill entries

### Requirement: Output format

The help script output SHALL preserve the current format conventions:

1. Version header: `Fab Kit v{version} — Specification-Driven Development`
2. `WORKFLOW` section with the diagram (static)
3. `COMMANDS` section with group headings and indented command entries
4. `TYPICAL FLOW` section (static)

Each command entry SHALL be formatted as: `    /name` followed by enough spaces to align descriptions, then the description text. The alignment column SHALL be computed dynamically based on the longest command name in the current output.
<!-- clarified: Dynamic alignment confirmed by user -->

#### Scenario: Description alignment

- **GIVEN** the longest command name is `/docs-hydrate-memory` (20 chars with prefix)
- **WHEN** the help script formats the command list
- **THEN** all description text SHALL start at the same column position

## Shared Library: frontmatter.sh

### Requirement: Extract frontmatter_field() to shared library

The `frontmatter_field()` function SHALL be moved from `fab/.kit/sync/3-sync-workspace.sh` to `fab/.kit/scripts/lib/frontmatter.sh`. The function's behavior SHALL be identical to the current implementation: extract a field value from YAML frontmatter delimited by `---` markers, returning the unquoted value or empty string if not found.

#### Scenario: Sourced by fab-help.sh

- **GIVEN** `fab/.kit/scripts/lib/frontmatter.sh` exists and defines `frontmatter_field()`
- **WHEN** `fab-help.sh` sources it via `source "$kit_dir/scripts/lib/frontmatter.sh"`
- **THEN** `frontmatter_field <file> <field>` SHALL return the correct field value

#### Scenario: Sourced by 3-sync-workspace.sh

- **GIVEN** `fab/.kit/scripts/lib/frontmatter.sh` exists
- **WHEN** `3-sync-workspace.sh` replaces its inline `frontmatter_field()` definition with `source "$kit_dir/scripts/lib/frontmatter.sh"`
- **THEN** all existing sync behavior SHALL remain identical — skill classification, agent file generation, etc.

### Requirement: frontmatter.sh file structure

`fab/.kit/scripts/lib/frontmatter.sh` SHALL be a sourceable shell library (no shebang, no `set -euo pipefail`). It SHALL define only `frontmatter_field()` with the same signature and implementation as the current inline version in `3-sync-workspace.sh`.

#### Scenario: Guard against double-sourcing

- **GIVEN** a script sources `frontmatter.sh` twice
- **WHEN** `frontmatter_field` is called
- **THEN** the function SHALL work correctly (function redefinition is idempotent in bash)

## Agent Cleanup

### Requirement: Delete redundant fab-help agent file

`.claude/agents/fab-help.md` SHALL be deleted. This file is a near-duplicate of the skill file `fab/.kit/skills/fab-help.md` with an outdated `model: haiku` frontmatter key (should be `model_tier: fast`). The agent version causes a subprocess spawn instead of inline execution, breaking output display.

#### Scenario: Agent file removed

- **GIVEN** `.claude/agents/fab-help.md` exists
- **WHEN** the apply stage executes this task
- **THEN** the file SHALL be deleted
- **AND** the skill at `fab/.kit/skills/fab-help.md` SHALL continue to function as the sole entry point for `/fab-help`

### Requirement: Corrected agent file via sync
<!-- clarified: Removed contradictory scenario and internal analysis — the requirement below captures the correct behavior -->

After the hand-authored `.claude/agents/fab-help.md` is deleted, the next `fab-sync.sh` run SHALL regenerate it via the model-tier agent file logic (section 6 of `3-sync-workspace.sh`). The regenerated file SHALL contain `model: haiku` (translated from `model_tier: fast` via `model-tiers.yaml`) and the full skill content — resolving the stale/broken state.

#### Scenario: Sync regenerates correct agent file

- **GIVEN** `.claude/agents/fab-help.md` was manually deleted
- **WHEN** `fab-sync.sh` runs
- **THEN** section 6 SHALL regenerate `.claude/agents/fab-help.md` from the skill file with `model_tier: fast` translated to `model: haiku`
- **AND** the regenerated content SHALL match the skill file exactly except for the `model_tier` → `model` substitution

## Design Decisions

1. **Group mapping lives in the script, not in frontmatter**: Adding a `group:` field to every skill would be over-engineering. Groups are a display concern that changes rarely. A short associative array or case statement in `fab-help.sh` is simpler and keeps the skill files focused on behavior.
   - *Why*: Follows the "don't add features beyond what's needed" principle. The group structure has been stable since the project started.
   - *Rejected*: `group:` frontmatter field — requires touching every skill file, adds a field that only one consumer (fab-help.sh) reads.

2. **Extract `frontmatter_field()` rather than duplicate**: Both `3-sync-workspace.sh` and `fab-help.sh` need this function. Constitution I (Pure Prompt Play) favors shared utilities via shell scripts.
   - *Why*: Avoids duplicating a non-trivial sed expression that parses YAML frontmatter.
   - *Rejected*: Copying the function into `fab-help.sh` — two copies to maintain, inevitable drift.

3. **"Other" catch-all group for unmapped skills**: Rather than silently dropping new skills from help output, unmapped skills appear under "Other". This ensures every user-facing skill is always visible.
   - *Why*: Eliminates the failure mode where a new skill is added but never appears in help.
   - *Rejected*: Requiring the mapping to be updated when adding skills — that's exactly the manual step this change aims to reduce.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exclude `_*` and `internal-*` prefixed files from help output | Confirmed from intake #1. Convention is established: `_` = partial, `internal-` = internal tooling | S:70 R:95 A:90 D:85 |
| 2 | Certain | Delete `.claude/agents/fab-help.md` (sync regenerates correct version) | Confirmed from intake #2. The hand-authored copy is stale; sync section 6 handles regeneration | S:95 R:90 A:90 D:95 |
| 3 | Confident | Keep group headings hardcoded in the script | Confirmed from intake #3. Groups are display layout, not data | S:60 R:90 A:70 D:50 |
| 4 | Confident | Extract `frontmatter_field()` to `lib/frontmatter.sh` | Confirmed from intake #4. Avoids duplication, both consumers source the same file | S:50 R:95 A:80 D:60 |
| 5 | Certain | Keep `fab-sync.sh` as hardcoded help entry | Clarified — user confirmed description as-is | S:90 R:90 A:90 D:85 |
| 6 | Certain | Dynamic column alignment for description text | Clarified — user confirmed dynamic over fixed-width | S:90 R:95 A:90 D:85 |
| 7 | Confident | "Other" catch-all group for unmapped skills | Ensures new skills always appear in help even if the mapping isn't updated. Low risk — can be changed to "skip" if preferred | S:50 R:90 A:75 D:50 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-18

1. **Spec hygiene: embedded reasoning** — Removed contradictory "Wait — this is a conflict" analysis block (lines 149-160). The "Corrected agent file via sync" requirement already captures the correct behavior.
2. **fab-sync.sh description** — Confirmed "Repair directories, symlinks, and agents (no LLM needed)" as the correct hardcoded description. Upgraded assumption #5 to Certain.
3. **Dynamic alignment** — Confirmed dynamic column alignment (computed from longest command name) over fixed-width. Upgraded assumption #6 to Certain.
4. **Dropped Execution group** — Confirmed removal of the "Execution" group with non-existent `/fab-apply` and `/fab-review` commands. These are sub-behaviors of `/fab-continue`, not separate skills.
