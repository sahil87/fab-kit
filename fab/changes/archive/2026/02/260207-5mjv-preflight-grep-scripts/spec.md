# Spec: Add fab-preflight.sh and update skills to consume it

**Change**: 260207-5mjv-preflight-grep-scripts
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/preflight.md` (new), `fab/docs/fab-workflow/context-loading.md` (modified)

## Preflight Script: Output Contract

### Requirement: Structured YAML Output

`fab-preflight.sh` SHALL output a YAML document to stdout containing the active change's resolved state. The output MUST include the following fields:

- `name` — the change folder name (from `fab/current`)
- `change_dir` — path to `fab/changes/{name}/`, relative to `fab/` (e.g., `changes/260207-5mjv-...`) <!-- clarified: relative to fab/, not absolute — matches existing script conventions -->
- `stage` — current stage (from `.status.yaml`)
- `branch` — branch name (from `.status.yaml`, empty string if unset)
- `progress` — full progress map (all 7 stages with their status)
- `checklist.generated` — boolean
- `checklist.completed` — integer
- `checklist.total` — integer

The output MUST be valid YAML parseable by any YAML-consuming tool. Agents consume this output by running the script via Bash and parsing the stdout directly. <!-- clarified: agents execute script via Bash tool and parse stdout YAML -->

#### Scenario: Normal active change

- **GIVEN** `fab/current` contains `260207-5mjv-preflight-grep-scripts`
- **AND** `fab/changes/260207-5mjv-preflight-grep-scripts/.status.yaml` exists and is valid
- **WHEN** `fab-preflight.sh` is executed via Bash
- **THEN** it outputs YAML with all fields populated from `.status.yaml`
- **AND** `change_dir` is `changes/260207-5mjv-preflight-grep-scripts`
- **AND** exits with code 0

#### Scenario: Change with no branch

- **GIVEN** `.status.yaml` has no `branch:` field
- **WHEN** `fab-preflight.sh` is executed
- **THEN** the output includes `branch: ""`

#### Scenario: Example output <!-- clarified: added concrete output example for implementability -->

```yaml
name: 260207-5mjv-preflight-grep-scripts
change_dir: changes/260207-5mjv-preflight-grep-scripts
stage: specs
branch: 260207-5mjv-preflight-grep-scripts
progress:
  proposal: done
  specs: done
  plan: pending
  tasks: pending
  apply: pending
  review: pending
  archive: pending
checklist:
  generated: false
  completed: 0
  total: 0
```

## Preflight Script: Validation

### Requirement: Project Initialization Validation <!-- clarified: preflight is full gate — also validates config.yaml and constitution.md -->

`fab-preflight.sh` SHALL validate that `fab/config.yaml` and `fab/constitution.md` exist before any other checks. If either is missing, the script MUST exit with code 1 and print `fab/ is not initialized. Run /fab:init first.` to stderr.

#### Scenario: Missing config.yaml

- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `fab/ is not initialized. Run /fab:init first.` to stderr
- **AND** exits with code 1

#### Scenario: Missing constitution.md

- **GIVEN** `fab/constitution.md` does not exist
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `fab/ is not initialized. Run /fab:init first.` to stderr
- **AND** exits with code 1

### Requirement: fab/current Validation

`fab-preflight.sh` SHALL validate that `fab/current` exists and is non-empty. If validation fails, the script MUST exit with a non-zero exit code and output a diagnostic message to stderr.

#### Scenario: No active change

- **GIVEN** `fab/current` does not exist
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `No active change. Run /fab:new to start one.` to stderr
- **AND** exits with code 1

#### Scenario: Empty fab/current

- **GIVEN** `fab/current` exists but is empty (or whitespace-only) <!-- clarified: whitespace-only edge case -->
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `No active change. Run /fab:new to start one.` to stderr
- **AND** exits with code 1

### Requirement: Change Directory Validation

`fab-preflight.sh` SHALL validate that the change directory `fab/changes/{name}/` exists. If validation fails, the script MUST exit with a non-zero exit code and output a diagnostic message to stderr.

#### Scenario: Missing change directory

- **GIVEN** `fab/current` contains `260207-abcd-missing`
- **AND** `fab/changes/260207-abcd-missing/` does not exist
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `Change directory not found: changes/260207-abcd-missing/` to stderr
- **AND** exits with code 1

### Requirement: .status.yaml Validation

`fab-preflight.sh` SHALL validate that `.status.yaml` exists within the change directory. If validation fails, the script MUST exit with a non-zero exit code.

#### Scenario: Missing .status.yaml

- **GIVEN** the change directory exists
- **AND** `.status.yaml` is missing from it
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it prints `Active change "{name}" is corrupted — .status.yaml not found.` to stderr
- **AND** exits with code 1

### Requirement: Validation Order <!-- clarified: explicit validation order for predictable error messages -->

Validations SHALL run in this order, stopping at the first failure:

1. `config.yaml` and `constitution.md` exist (project initialized)
2. `fab/current` exists and is non-empty (active change set)
3. Change directory exists
4. `.status.yaml` exists within change directory

## Preflight Script: Portability

### Requirement: No External Dependencies

`fab-preflight.sh` SHALL use only POSIX-standard tools (`grep`, `sed`, `awk`, `cat`, `cut`) and Bash builtins. It MUST NOT require `yq`, `jq`, Python, or any tool not included in a standard macOS or Linux installation.

#### Scenario: Runs on a fresh macOS system

- **GIVEN** a macOS system with only default tools
- **WHEN** `fab-preflight.sh` is executed
- **THEN** it runs without errors (no missing command failures)

### Requirement: Idempotent and Read-Only

`fab-preflight.sh` SHALL NOT modify any files. It MUST be safe to run any number of times without side effects. This aligns with Constitution Principle III (Idempotent Operations).

#### Scenario: Multiple invocations

- **GIVEN** a valid active change
- **WHEN** `fab-preflight.sh` is executed twice in succession
- **THEN** both invocations produce identical output
- **AND** no files in `fab/` are modified

## Preflight Script: Path Resolution

### Requirement: Relative Path Resolution

`fab-preflight.sh` SHALL resolve all internal paths relative to its own location in `fab/.kit/scripts/`, navigating up to the `fab/` root via `$(dirname "$0")/../..`. It MUST work regardless of the caller's working directory. All paths in stdout output SHALL be relative to the `fab/` directory. <!-- clarified: output paths relative to fab/ -->

#### Scenario: Called from repo root

- **GIVEN** the user's working directory is the repo root
- **WHEN** `fab/.kit/scripts/fab-preflight.sh` is executed
- **THEN** it correctly resolves `fab/current` and the change directory

#### Scenario: Called from an unrelated directory

- **GIVEN** the user's working directory is `/tmp`
- **WHEN** the script is called by absolute path
- **THEN** it correctly resolves `fab/current` and the change directory

## _context.md: Preflight Integration

### Requirement: Document fab-preflight.sh in Change Context Layer

`_context.md` Section 2 ("Change Context") SHALL reference `fab-preflight.sh` as the standard mechanism for resolving the active change and validating its state. The section SHALL instruct agents to run the script via Bash, check the exit code, and parse the stdout YAML for change context. The existing 4-step inline sequence SHALL remain as documentation of what the script does internally (not removed). <!-- clarified: explicit consumption model — run via Bash, parse stdout -->

#### Scenario: Agent reads _context.md

- **GIVEN** an agent reads `fab/.kit/skills/_context.md`
- **WHEN** it reaches the "Change Context" section
- **THEN** it finds a directive to run `fab/.kit/scripts/fab-preflight.sh` via Bash
- **AND** instructions to stop if the script exits non-zero (surfacing the stderr message)
- **AND** instructions to parse the stdout YAML for change name, stage, progress, and checklist state
- **AND** the existing 4-step inline sequence remains as a reference for what the script validates

### Requirement: Always-Load Layer References Preflight for Init Check <!-- clarified: since preflight now validates config/constitution, _context.md should note this -->

`_context.md` Section 1 ("Always Load") SHALL note that `fab-preflight.sh` covers the initialization check (config.yaml and constitution.md existence). Skills that run preflight do not need to separately verify these files exist — only that they can be read for content.

#### Scenario: Agent uses preflight then loads config

- **GIVEN** an agent runs `fab-preflight.sh` and it exits 0
- **WHEN** the agent proceeds to read `fab/config.yaml`
- **THEN** it can trust the file exists (preflight already validated)
- **AND** it reads the file for its content (project name, tech stack, etc.)

## Skill Files: Preflight References

### Requirement: Update Pre-flight Sections in Skills

Each skill that performs inline pre-flight checks (ff, apply, review, archive, continue, clarify) SHALL be updated so that its "Pre-flight Check" section directs the agent to run `fab/.kit/scripts/fab-preflight.sh` via Bash. On non-zero exit, the agent SHALL stop and surface the stderr message. On success, the agent SHALL use the stdout YAML for change context (name, stage, progress) instead of re-reading `.status.yaml`. <!-- clarified: concrete consumption instructions per skill -->

The inline validation steps (check current, check directory, check .status.yaml, check config/constitution) SHALL be replaced with the single preflight directive.

Skills that are exempt from the always-load convention (`init`, `switch`, `status`, `hydrate`, `help`) SHALL NOT be modified.

`fab-new` SHALL NOT be modified — it has a distinct pre-flight (checking config.yaml and constitution.md exist, not checking fab/current).

#### Scenario: fab-ff pre-flight section

- **GIVEN** an agent reads `fab/.kit/skills/fab-ff.md`
- **WHEN** it reaches the "Pre-flight Check" section
- **THEN** it finds a directive to run `fab-preflight.sh` via Bash and parse the stdout YAML
- **AND** the section describes what to do on script failure (stop with the error message)

#### Scenario: fab-apply pre-flight section

- **GIVEN** an agent reads `fab/.kit/skills/fab-apply.md`
- **WHEN** it reaches the "Pre-flight Check" section
- **THEN** it finds the same preflight directive pattern as fab-ff
- **AND** any stage-specific validation (e.g., "tasks must be done") remains as an additional check after the preflight, using the `progress` field from preflight output

#### Scenario: Skill still validates stage-specific preconditions

- **GIVEN** a skill requires a specific stage to be complete (e.g., apply requires tasks done)
- **WHEN** the pre-flight section is updated
- **THEN** the stage-specific check is preserved as a separate step after preflight
- **AND** the preflight output's `stage` and `progress` fields are used for this check instead of re-reading .status.yaml

### Requirement: Preserve Skill-Specific Context Loading

The "Context Loading" or "Load Context" sections within each skill (which load proposal.md, spec.md, plan.md, source code, etc.) SHALL NOT be modified. Preflight replaces only the validation boilerplate and `.status.yaml` reading, not the artifact-loading logic.

#### Scenario: fab-review context loading unchanged

- **GIVEN** `fab-review.md` has a context loading section that reads spec.md, plan.md, tasks.md, checklists, and source code
- **WHEN** the preflight update is applied
- **THEN** that context loading section remains intact
- **AND** only the pre-flight validation section references the preflight script

### Requirement: Skills Remove Redundant config/constitution Existence Checks <!-- clarified: since preflight now covers init validation, skills don't need separate checks -->

Skills that currently check for `fab/config.yaml` or `fab/constitution.md` existence in their pre-flight section SHALL remove those checks, since `fab-preflight.sh` validates them. Skills still read these files for content in their context-loading step.

#### Scenario: fab-clarify pre-flight simplified

- **GIVEN** `fab-clarify.md` currently checks config.yaml and constitution.md in its pre-flight
- **WHEN** the preflight update is applied
- **THEN** those existence checks are removed from the pre-flight section
- **AND** the context loading section still reads config.yaml and constitution.md for content

## Deprecated Requirements

<!-- None — this change adds new capabilities without removing existing ones. -->
