# Spec: Standardize Script Names and Add fab-help.sh

**Change**: 260207-cg03-standardize-script-names
**Created**: 2026-02-07
**Affected docs**: None (kit-internal change; no centralized docs in `fab/docs/` affected)

<!--
  CHANGE SPECIFICATION
  Three domains: script naming, new fab-help.sh script, reference updates.
  Requirements use RFC 2119 keywords.
  Every requirement has at least one GIVEN/WHEN/THEN scenario.
-->

## Script Naming: Prefix Convention

### Requirement: All scripts in `fab/.kit/scripts/` MUST use a `fab-` prefix

Every shell script in `fab/.kit/scripts/` SHALL be named with a `fab-` prefix followed by a descriptive slug (e.g., `fab-setup.sh`, `fab-status.sh`). This ensures consistent naming with the `fab-*` convention used for skill files.

#### Scenario: Renamed scripts exist at new paths
- **GIVEN** the change has been applied
- **WHEN** listing files in `fab/.kit/scripts/`
- **THEN** the following files MUST exist: `fab-setup.sh`, `fab-status.sh`, `fab-update-claude-settings.sh`, `fab-help.sh`
- **AND** the following files MUST NOT exist: `setup.sh`, `status.sh`, `update-claude-settings.sh`

#### Scenario: Renamed scripts are executable
- **GIVEN** the renamed scripts exist at their new paths
- **WHEN** checking file permissions with `test -x`
- **THEN** all four scripts MUST be executable

### Requirement: Renamed scripts MUST preserve identical behavior

Each renamed script SHALL produce identical output and exit codes as its predecessor. The rename is purely cosmetic — no functional changes to script logic.

#### Scenario: fab-setup.sh behaves identically to setup.sh
- **GIVEN** a repository with `fab/.kit/` in place
- **WHEN** running `bash fab/.kit/scripts/fab-setup.sh`
- **THEN** the output, side effects (directories, symlinks, .gitignore), and exit code MUST be identical to the former `setup.sh`

#### Scenario: fab-status.sh behaves identically to status.sh
- **GIVEN** no `fab/current` file exists
- **WHEN** running `bash fab/.kit/scripts/fab-status.sh`
- **THEN** it MUST output `No active change` and exit 0

#### Scenario: fab-status.sh with active change
- **GIVEN** `fab/current` contains a valid change name with a `.status.yaml` that has `stage: specs` and `branch: my-branch`
- **WHEN** running `bash fab/.kit/scripts/fab-status.sh`
- **THEN** it MUST output `Active: {name} (stage: specs, branch: my-branch)` and exit 0

#### Scenario: fab-status.sh with missing .status.yaml <!-- clarified: added missing error-state scenario from existing status.sh behavior -->
- **GIVEN** `fab/current` contains a change name but the corresponding `.status.yaml` does not exist
- **WHEN** running `bash fab/.kit/scripts/fab-status.sh`
- **THEN** it MUST output `Active: {name} (missing — run /fab:switch or /fab:new)` and exit 1

#### Scenario: fab-update-claude-settings.sh behaves identically
- **GIVEN** `.claude/settings.local.json` exists at the repo root
- **WHEN** running `bash fab/.kit/scripts/fab-update-claude-settings.sh`
- **THEN** it MUST copy the file to `fab/worktree-init/assets/settings.local.json` and output the same confirmation message as before

#### Scenario: fab-update-claude-settings.sh with missing source <!-- clarified: added error-state scenario -->
- **GIVEN** `.claude/settings.local.json` does NOT exist at the repo root
- **WHEN** running `bash fab/.kit/scripts/fab-update-claude-settings.sh`
- **THEN** it MUST output `No .claude/settings.local.json found` and exit 1

### Requirement: Self-referencing comments MUST use the new name

Each script's internal comment header (shebang, description, usage) SHALL reference its new filename, not the old one.

#### Scenario: fab-setup.sh comment header
- **GIVEN** the file `fab/.kit/scripts/fab-setup.sh`
- **WHEN** reading the comment block (lines 4 and 9)
- **THEN** line 4 MUST contain `fab/.kit/scripts/fab-setup.sh` (not `setup.sh`)
- **AND** line 9 MUST contain `fab/.kit/scripts/fab-setup.sh` (not `setup.sh`)

## Script Naming: New fab-help.sh

### Requirement: A `fab-help.sh` script MUST be created

A new executable script `fab/.kit/scripts/fab-help.sh` SHALL be created that outputs the Fab help text. It MUST read the version from `fab/.kit/VERSION` and print the complete help output currently defined in the `fab-help.md` skill.

#### Scenario: fab-help.sh outputs help text with version
- **GIVEN** `fab/.kit/VERSION` contains `0.1.0`
- **WHEN** running `bash fab/.kit/scripts/fab-help.sh`
- **THEN** the first line of output MUST be `Fab Kit v0.1.0 — Specification-Driven Development`
- **AND** the output MUST include the `WORKFLOW`, `COMMANDS`, and `TYPICAL FLOW` sections

#### Scenario: fab-help.sh with missing VERSION <!-- clarified: specified graceful handling under set -e -->
- **GIVEN** `fab/.kit/VERSION` does not exist
- **WHEN** running `bash fab/.kit/scripts/fab-help.sh`
- **THEN** the first line of output MUST be `Fab Kit vunknown — Specification-Driven Development`
- **AND** the script MUST exit 0 (not fail due to `set -e`; it SHALL check for the file before reading)

### Requirement: fab-help.sh output MUST match the canonical help text <!-- clarified: pinned output to exact content -->

The output of `fab-help.sh` SHALL be byte-for-byte identical to the help text block currently defined in the `fab-help.md` skill's Output section (lines 25–62), with `{version}` substituted from `fab/.kit/VERSION`. This script is the single source of truth for help content — the skill file references it rather than duplicating.

#### Scenario: Output content matches canonical help text
- **GIVEN** `fab/.kit/VERSION` contains `0.1.0`
- **WHEN** running `bash fab/.kit/scripts/fab-help.sh`
- **THEN** the output MUST contain all of these exact strings:
  - `WORKFLOW`
  - `/fab:new ─→ /fab:continue (or /fab:ff) ─→ /fab:apply ─→ /fab:review ─→ /fab:archive`
  - `COMMANDS`
  - `Start & Navigate`
  - `Planning`
  - `Execution`
  - `Completion`
  - `Setup`
  - `TYPICAL FLOW`

### Requirement: fab-help.sh MUST follow kit script conventions

The script SHALL use `#!/usr/bin/env bash`, `set -euo pipefail`, and resolve paths relative to its own location (via `$(dirname "$0")`), consistent with the existing scripts.

#### Scenario: Script header matches conventions
- **GIVEN** the file `fab/.kit/scripts/fab-help.sh`
- **WHEN** reading the first two lines
- **THEN** line 1 MUST be `#!/usr/bin/env bash`
- **AND** line 2 MUST be `set -euo pipefail`

## Skill Integration: fab-help.md

### Requirement: fab-help.md MUST delegate to fab-help.sh

The `/fab:help` skill definition SHALL instruct the agent to execute `fab/.kit/scripts/fab-help.sh` and display its output, rather than containing the help text inline. The skill's Output section MUST reference the script as the source of truth for help content.

#### Scenario: Skill instructs agent to run the script
- **GIVEN** the file `fab/.kit/skills/fab-help.md`
- **WHEN** reading the Behavior section
- **THEN** it MUST contain an instruction to execute `fab/.kit/scripts/fab-help.sh`
- **AND** it MUST NOT contain the literal help text inline (the `WORKFLOW`, `COMMANDS`, `TYPICAL FLOW` content)

#### Scenario: Key properties preserved
- **GIVEN** the updated `fab-help.md`
- **WHEN** reading the Key Properties table
- **THEN** `Advances stage?` MUST be `No`
- **AND** `Idempotent?` MUST be `Yes`
- **AND** `Modifies any files?` MUST be `No`

## Reference Updates: Internal References

### Requirement: All references to old script names MUST be updated

Every file that references the old script names (`setup.sh`, `status.sh`) by their former paths MUST be updated to use the new `fab-` prefixed names. This applies to skill files, config files, spec docs, and permission patterns.

#### Scenario: fab-init.md references fab-setup.sh
- **GIVEN** the file `fab/.kit/skills/fab-init.md`
- **WHEN** reading the file
- **THEN** all references to `scripts/setup.sh` MUST be replaced with `scripts/fab-setup.sh`
- **AND** there MUST be zero occurrences of the string `scripts/setup.sh`

#### Scenario: settings.local.json permission pattern updated
- **GIVEN** the file `fab/worktree-init/assets/settings.local.json`
- **WHEN** reading the permissions allow list
- **THEN** the entry MUST be `Bash(fab/.kit/scripts/fab-setup.sh:*)` (not `Bash(fab/.kit/scripts/setup.sh:*)`)

#### Scenario: ARCHITECTURE.md updated
- **GIVEN** the file `doc/fab-spec/ARCHITECTURE.md`
- **WHEN** reading the file
- **THEN** the tree listing MUST show `fab-status.sh`, `fab-setup.sh`, `fab-help.sh`, and `fab-update-claude-settings.sh` under `scripts/`
- **AND** all prose references to `scripts/status.sh` MUST be replaced with `scripts/fab-status.sh`
- **AND** all prose references to `scripts/setup.sh` MUST be replaced with `scripts/fab-setup.sh`

#### Scenario: README.md updated
- **GIVEN** the file `doc/fab-spec/README.md`
- **WHEN** reading the file
- **THEN** all references to `scripts/status.sh` MUST be replaced with `scripts/fab-status.sh`

### Requirement: Historical files MUST NOT be modified

Files in `.ralph/`, `.agents/tasks/`, and `fab/backlog.md` are historical records. They SHALL NOT be updated — they retain the old script names as-is.

#### Scenario: Historical files unchanged
- **GIVEN** the files `.ralph/progress.md`, `.agents/tasks/prd-fab-kit.json`, and `fab/backlog.md`
- **WHEN** comparing their content before and after this change
- **THEN** they MUST be byte-identical (no modifications)

## Deprecated Requirements

<!-- No existing requirements are being removed — this is additive (rename + new script). -->

None.
