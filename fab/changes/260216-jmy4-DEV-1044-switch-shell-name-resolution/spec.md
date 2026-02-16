# Spec: Delegate fab-switch Name Resolution to Shell

**Change**: 260216-jmy4-DEV-1044-switch-shell-name-resolution
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/preflight.md`

## Name Resolution: Shell Delegation

### Requirement: Argument Flow MUST delegate to resolve-change.sh

When `<change-name>` is provided, the `/fab-switch` skill SHALL call `fab/.kit/scripts/lib/resolve-change.sh` via Bash for name resolution instead of performing in-prompt string matching. The skill MUST source the script, invoke `resolve_change "fab" "<change-name>"`, and use `$RESOLVED_CHANGE_NAME` on success (exit 0).

#### Scenario: Exact match by full folder name

- **GIVEN** `fab/changes/260216-ymvx-DEV-1043-envrc-line-sync/` exists
- **WHEN** `/fab-switch 260216-ymvx-DEV-1043-envrc-line-sync` is invoked
- **THEN** `resolve_change` returns exit 0 with `RESOLVED_CHANGE_NAME=260216-ymvx-DEV-1043-envrc-line-sync`
- **AND** the skill proceeds with the Switch Flow using that folder name

#### Scenario: Single partial match

- **GIVEN** `fab/changes/260216-ymvx-DEV-1043-envrc-line-sync/` is the only folder containing `DEV-1043`
- **WHEN** `/fab-switch DEV-1043` is invoked
- **THEN** `resolve_change` returns exit 0 with `RESOLVED_CHANGE_NAME=260216-ymvx-DEV-1043-envrc-line-sync`
- **AND** the skill proceeds with the Switch Flow

#### Scenario: No match

- **GIVEN** no folder in `fab/changes/` matches `nonexistent`
- **WHEN** `/fab-switch nonexistent` is invoked
- **THEN** `resolve_change` returns exit 1 with stderr `No change matches "nonexistent".`
- **AND** the skill lists all available changes and informs the user

### Requirement: Multiple matches MUST be presented for user selection

When `resolve_change` returns exit 1 with stderr containing `Multiple changes match`, the skill SHALL parse the comma-separated folder names from stderr and present them as numbered options for the user to pick.

#### Scenario: Multiple partial matches

- **GIVEN** `fab/changes/260216-a-foo/` and `fab/changes/260216-b-foo/` both exist
- **WHEN** `/fab-switch foo` is invoked
- **THEN** `resolve_change` returns exit 1 with stderr `Multiple changes match "foo": 260216-a-foo, 260216-b-foo.`
- **AND** the skill presents both as numbered options with their current stages

### Requirement: No Argument Flow SHALL remain LLM-driven

When no `<change-name>` is provided (and no `--blank`), the skill SHALL NOT call `resolve-change.sh`. Instead it SHALL scan `fab/changes/` directly (excluding `archive/`), list folders with their stages, and prompt the user to select.

#### Scenario: No argument invocation

- **GIVEN** `fab/changes/` contains two change folders
- **WHEN** `/fab-switch` is invoked with no arguments
- **THEN** the skill lists both changes with stages as numbered options
- **AND** waits for user selection without invoking `resolve-change.sh`

### Requirement: Context Loading MUST document shell script dependency

The Context Loading section of both files SHALL note that `fab/.kit/scripts/lib/resolve-change.sh` is used for name resolution when a `<change-name>` argument is provided.

#### Scenario: Context Loading documentation

- **GIVEN** the updated skill file
- **WHEN** a reader inspects the Context Loading section
- **THEN** it mentions `resolve-change.sh` alongside the existing config/status dependencies

### Requirement: Both skill and agent files MUST be updated in sync

Changes to `fab/.kit/skills/fab-switch.md` SHALL be mirrored exactly in `.claude/agents/fab-switch.md`. The two files MUST have identical content in the Argument Flow and Context Loading sections (differing only in YAML frontmatter).

#### Scenario: File sync verification

- **GIVEN** the updated `fab/.kit/skills/fab-switch.md`
- **WHEN** compared to `.claude/agents/fab-switch.md` (ignoring YAML frontmatter)
- **THEN** the body content is identical

## Design Decisions

1. **Source-and-invoke pattern over script-as-subprocess**: The skill instructs the LLM to run `source resolve-change.sh && resolve_change "fab" "<arg>" && echo "$RESOLVED_CHANGE_NAME"` in a single Bash invocation.
   - *Why*: `resolve-change.sh` uses a variable-setting pattern (`RESOLVED_CHANGE_NAME`) designed for sourcing, not standalone execution. This is the same pattern used by `preflight.sh` (documented in `docs/memory/fab-workflow/preflight.md`).
   - *Rejected*: Running `resolve-change.sh` as a subprocess — the script has no `main()` or standalone entry point; it's a library file.

2. **Stderr parsing for multi-match handling**: On multi-match (exit 1), the skill reads the comma-separated list from stderr rather than re-running its own folder scan.
   - *Why*: `resolve-change.sh` already formats the match list. Duplicating the scan would defeat the purpose of delegating.
   - *Rejected*: Adding a structured output mode to `resolve-change.sh` — changes the script, violating the "no modifications needed" constraint from the intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use existing `resolve-change.sh` without modification | Confirmed from intake #1 — script handles exact + substring matching correctly; battle-tested by `preflight.sh` and `batch-fab-switch-change.sh` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep No Argument Flow as LLM-driven listing | Confirmed from intake #2 — listing is a presentation concern; `resolve-change.sh` doesn't cover enumeration | S:90 R:95 A:90 D:90 |
| 3 | Certain | Mirror changes in both skill and agent files | Confirmed from intake #3 — both files contain identical body content | S:95 R:95 A:95 D:95 |
| 4 | Confident | Multiple-match handling via stderr parsing | Confirmed from intake #4 — `resolve-change.sh` prints comma-separated matches to stderr; parsing is simpler than re-scanning | S:80 R:85 A:75 D:70 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
