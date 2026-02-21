# Spec: Batch Script Frontmatter for fab-help Discovery

**Change**: 260221-alng-batch-script-frontmatter
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Shell Script Frontmatter Convention

### Requirement: Shell-Comment Frontmatter Format

Shell scripts MAY include a frontmatter block using `# ---` delimiters. Each field SHALL be a `# key: value` line between the opening and closing `# ---` markers. The block MUST appear before `set -euo pipefail` (or the first non-comment code line). Supported fields: `name` (string), `description` (quoted string).

```bash
#!/usr/bin/env bash
# ---
# name: batch-fab-switch-change
# description: "Open tmux tabs in worktrees for one or more changes"
# ---
set -euo pipefail
```

#### Scenario: Script with valid frontmatter
- **GIVEN** a shell script with `# ---` delimiters and `# name:` / `# description:` fields
- **WHEN** `shell_frontmatter_field` is called with the file and a field name
- **THEN** the unquoted field value is returned

#### Scenario: Script without frontmatter
- **GIVEN** a shell script with no `# ---` block
- **WHEN** `shell_frontmatter_field` is called
- **THEN** an empty string is returned

#### Scenario: Script with regular `---` (not `# ---`)
- **GIVEN** a shell script containing `---` markers (not `# ---`)
- **WHEN** `shell_frontmatter_field` is called
- **THEN** an empty string is returned (only `# ---` delimiters are recognized)

### Requirement: Batch Script Frontmatter Content

Each `batch-*.sh` script in `fab/.kit/scripts/` SHALL include a `# ---` frontmatter block with `name` and `description` fields. The existing comment header line (e.g., `# batch-fab-switch-change.sh — ...`) SHALL be removed to avoid duplication with the frontmatter description.

| Script | name | description |
|--------|------|-------------|
| `batch-fab-switch-change.sh` | `batch-fab-switch-change` | Open tmux tabs in worktrees for one or more changes |
| `batch-fab-archive-change.sh` | `batch-fab-archive-change` | Archive multiple completed changes in one session |
| `batch-fab-new-backlog.sh` | `batch-fab-new-backlog` | Create worktree tabs from backlog items |

#### Scenario: Frontmatter replaces comment header
- **GIVEN** `batch-fab-switch-change.sh` with the old comment `# batch-fab-switch-change.sh — Per change ID/name: ...`
- **WHEN** the frontmatter block is added
- **THEN** the old comment header line is removed
- **AND** the `# ---` block appears between the shebang and `set -euo pipefail`

## Frontmatter Parser Extension

### Requirement: `shell_frontmatter_field` Function

`fab/.kit/scripts/lib/frontmatter.sh` SHALL export a new function `shell_frontmatter_field` with the same interface as `frontmatter_field`: `shell_frontmatter_field <file> <field_name>` → unquoted value or empty string.

The function SHALL:
1. Match `# ---` (not bare `---`) as opening and closing delimiters
2. Strip the leading `# ` from each line within the block before matching the field
3. Handle both quoted (`"value"`) and unquoted values
4. Strip trailing inline comments (` #...`)

The existing `frontmatter_field` function MUST NOT be modified.

#### Scenario: Parse name field from shell frontmatter
- **GIVEN** a file containing `# ---\n# name: batch-fab-new-backlog\n# ---`
- **WHEN** `shell_frontmatter_field <file> name` is called
- **THEN** `batch-fab-new-backlog` is returned

#### Scenario: Parse quoted description field
- **GIVEN** a file containing `# description: "Archive multiple completed changes in one session"`
- **WHEN** `shell_frontmatter_field <file> description` is called
- **THEN** `Archive multiple completed changes in one session` is returned (quotes stripped)

#### Scenario: Field not found
- **GIVEN** a file with shell frontmatter that does not contain a `group` field
- **WHEN** `shell_frontmatter_field <file> group` is called
- **THEN** an empty string is returned

## Help Output Integration

### Requirement: Batch Script Discovery in fab-help.sh

`fab-help.sh` SHALL scan `"$kit_dir"/scripts/batch-*.sh` for shell-comment frontmatter using `shell_frontmatter_field`. Discovered scripts SHALL be rendered in the COMMANDS section under a "Batch Operations" group.

#### Scenario: Batch scripts appear in help output
- **GIVEN** three `batch-*.sh` scripts with valid `# ---` frontmatter
- **WHEN** `fab-help.sh` is executed
- **THEN** a "Batch Operations" group appears in the COMMANDS section
- **AND** each script is listed with its name and description
- **AND** names are displayed without a `/` prefix (they are shell commands, not slash-commands)

#### Scenario: Batch script without frontmatter is skipped
- **GIVEN** a `batch-*.sh` file without `# ---` frontmatter
- **WHEN** `fab-help.sh` is executed
- **THEN** that script does not appear in the help output

### Requirement: Centralized Group Mapping

Group assignment for batch scripts SHALL be maintained in `fab-help.sh` via the same centralized mapping pattern used for skills. The `skill_to_group` associative array (or a parallel `batch_to_group` array) SHALL map batch script names to the `"Batch Operations"` group. Batch scripts SHALL NOT self-declare their group in frontmatter.

#### Scenario: Group mapping in fab-help.sh
- **GIVEN** `fab-help.sh` contains `"Batch Operations"` in `group_order`
- **AND** batch script names are mapped to `"Batch Operations"` in the group mapping
- **WHEN** a new batch script is added with frontmatter
- **THEN** it must also be added to the group mapping in `fab-help.sh` to appear under "Batch Operations"

### Requirement: Alignment Consistency

The dynamic alignment computation in `fab-help.sh` SHALL include batch script display names when calculating `max_len`. This ensures consistent column alignment across skills and batch scripts.

#### Scenario: Long batch script name affects alignment
- **GIVEN** `batch-fab-switch-change` (24 chars) is longer than any skill display name
- **WHEN** `fab-help.sh` computes alignment
- **THEN** all entries (skills and batch scripts) align to the same column

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `# ---` delimiters for shell-comment frontmatter | Explicitly discussed and chosen as Option 1. Confirmed from intake #1 | S:95 R:95 A:90 D:95 |
| 2 | Certain | Centralized grouping in fab-help.sh | User explicitly said "Let fab-help own the grouping". Confirmed from intake #2 | S:95 R:90 A:90 D:95 |
| 3 | Confident | Add `shell_frontmatter_field` as a new function in frontmatter.sh | Same file, parallel function — keeps parser library cohesive. Confirmed from intake #3 | S:75 R:90 A:85 D:75 |
| 4 | Confident | Display batch scripts without `/` prefix | Shell commands run directly, not agent slash-commands. Confirmed from intake #4 | S:70 R:95 A:80 D:75 |
| 5 | Certain | Group name "Batch Operations" | Mentioned in discussion, no objection. Confirmed from intake #5 | S:85 R:95 A:85 D:90 |
| 6 | Certain | Remove old comment headers from batch scripts | Frontmatter description replaces the existing `# script-name — description` comment. Avoids duplication | S:80 R:95 A:90 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
