# Spec: Fold resolve-change into changeman

**Change**: 260216-oinh-DEV-1045-fold-resolve-into-changeman
**Created**: 2026-02-17
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/preflight.md`

## Non-Goals

- Changing resolve behavior (matching algorithm, case-insensitivity, archive exclusion) — the logic is ported verbatim
- Moving interactive selection (no-arg `/fab-switch`) into shell — Claude UI prompting stays in the skill layer
- Adding new config.yaml fields — `switch` reads existing `git.enabled` and `git.branch_prefix`

## changeman: `resolve` subcommand

### Requirement: Resolve from `fab/current`

`changeman.sh resolve` (no override) SHALL read `fab/current`, strip whitespace, and print the folder name to stdout. It SHALL exit 1 with a diagnostic to stderr if `fab/current` is missing or empty.

#### Scenario: Active change exists

- **GIVEN** `fab/current` contains `260216-a7k2-add-oauth\n`
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** stdout contains `260216-a7k2-add-oauth` (whitespace stripped)
- **AND** exit code is 0

#### Scenario: No active change

- **GIVEN** `fab/current` does not exist
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** stderr contains a diagnostic message
- **AND** exit code is 1

#### Scenario: Empty fab/current

- **GIVEN** `fab/current` exists but contains only whitespace
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** stderr contains a diagnostic message
- **AND** exit code is 1

### Requirement: Resolve from override

`changeman.sh resolve <override>` SHALL perform case-insensitive substring matching against folder names in `fab/changes/` (excluding `archive/`). Exact match wins over partial. Single partial match resolves. Multiple partial matches SHALL exit 1 listing the matches to stderr. No match SHALL exit 1 with a diagnostic.

#### Scenario: Exact match

- **GIVEN** `fab/changes/260216-a7k2-add-oauth/` exists
- **WHEN** `changeman.sh resolve "260216-a7k2-add-oauth"` is invoked
- **THEN** stdout contains `260216-a7k2-add-oauth`
- **AND** exit code is 0

#### Scenario: Single partial match (case-insensitive)

- **GIVEN** `fab/changes/260216-a7k2-DEV-100-add-oauth/` is the only folder containing "dev-100"
- **WHEN** `changeman.sh resolve "dev-100"` is invoked
- **THEN** stdout contains `260216-a7k2-DEV-100-add-oauth`
- **AND** exit code is 0

#### Scenario: Multiple partial matches

- **GIVEN** `fab/changes/260216-a7k2-add-oauth/` and `fab/changes/260216-b3k2-add-auth/` both contain "add"
- **WHEN** `changeman.sh resolve "add"` is invoked
- **THEN** stderr lists the matching folder names
- **AND** exit code is 1

#### Scenario: No match

- **GIVEN** no folder in `fab/changes/` contains "zzz"
- **WHEN** `changeman.sh resolve "zzz"` is invoked
- **THEN** stderr contains a diagnostic message
- **AND** exit code is 1

#### Scenario: Archive folder excluded

- **GIVEN** `fab/changes/archive/` exists
- **WHEN** `changeman.sh resolve "archive"` is invoked
- **THEN** the `archive/` folder SHALL NOT be considered a match
- **AND** the result depends on whether other folders match "archive"

### Requirement: Missing changes directory

`changeman.sh resolve` SHALL exit 1 with a diagnostic if `fab/changes/` does not exist (override mode only — no-override mode reads `fab/current` directly).

#### Scenario: No changes directory

- **GIVEN** `fab/changes/` does not exist
- **WHEN** `changeman.sh resolve "anything"` is invoked
- **THEN** stderr contains a diagnostic message
- **AND** exit code is 1

## changeman: `switch` subcommand

### Requirement: Normal switch flow

`changeman.sh switch <name>` SHALL resolve the change name (via internal resolve logic), write the folder name to `fab/current`, perform git branch integration, and output a structured summary to stdout.

#### Scenario: Switch to existing change

- **GIVEN** `fab/changes/260216-a7k2-add-oauth/` exists with a valid `.status.yaml` at intake stage
- **AND** `config.yaml` has `git.enabled: true` and `branch_prefix: ""`
- **WHEN** `changeman.sh switch "a7k2"` is invoked
- **THEN** `fab/current` contains `260216-a7k2-add-oauth`
- **AND** git checks out or creates branch `260216-a7k2-add-oauth`
- **AND** stdout contains the change name, branch, stage, and next command suggestion

#### Scenario: Switch with branch prefix

- **GIVEN** `config.yaml` has `git.enabled: true` and `branch_prefix: "feat/"`
- **WHEN** `changeman.sh switch "a7k2"` is invoked and resolves to `260216-a7k2-add-oauth`
- **THEN** git checks out or creates branch `feat/260216-a7k2-add-oauth`

#### Scenario: Git disabled

- **GIVEN** `config.yaml` has `git.enabled: false`
- **WHEN** `changeman.sh switch "a7k2"` is invoked
- **THEN** `fab/current` is written
- **AND** no git operations are performed
- **AND** stdout omits the branch line

### Requirement: Deactivation flow

`changeman.sh switch --blank` SHALL delete `fab/current` (no-op if absent) and output a confirmation.

#### Scenario: Deactivate active change

- **GIVEN** `fab/current` exists
- **WHEN** `changeman.sh switch --blank` is invoked
- **THEN** `fab/current` is deleted
- **AND** stdout contains a deactivation confirmation

#### Scenario: Deactivate when already blank

- **GIVEN** `fab/current` does not exist
- **WHEN** `changeman.sh switch --blank` is invoked
- **THEN** stdout indicates already blank
- **AND** exit code is 0

### Requirement: Git branch integration

When `git.enabled` is `true` (or `config.yaml` is absent — default to enabled), `switch` SHALL check out the branch if it exists (`git checkout <branch>`) or create it if it does not (`git checkout -b <branch>`). The branch name is `{branch_prefix}{folder_name}`.

#### Scenario: Branch exists

- **GIVEN** git branch `260216-a7k2-add-oauth` exists
- **WHEN** switch resolves to `260216-a7k2-add-oauth`
- **THEN** `git checkout 260216-a7k2-add-oauth` is executed

#### Scenario: Branch does not exist

- **GIVEN** git branch `260216-a7k2-add-oauth` does not exist
- **WHEN** switch resolves to `260216-a7k2-add-oauth`
- **THEN** `git checkout -b 260216-a7k2-add-oauth` is executed

#### Scenario: Git operation fails

- **GIVEN** `git checkout` fails (e.g., uncommitted changes)
- **WHEN** switch attempts branch integration
- **THEN** stderr contains the git error
- **AND** `fab/current` is still written (switch succeeds despite git failure)
- **AND** stdout notes the branch integration failure

#### Scenario: Not inside a git repo

- **GIVEN** the working directory is not a git repository
- **WHEN** `changeman.sh switch` is invoked
- **THEN** git integration is skipped silently
- **AND** the switch completes without branch lines in output

### Requirement: Config reading

`switch` SHALL read `fab/config.yaml` for `git.enabled` and `git.branch_prefix`. If `config.yaml` is missing, `switch` SHALL default to `git.enabled: true` and `branch_prefix: ""`.

Config values SHALL be extracted using `yq` (already a system dependency via stageman).
<!-- assumed: yq for config parsing — yq is already required by stageman and available on PATH; using it in changeman is consistent and avoids fragile grep/sed parsing of YAML -->

#### Scenario: Config present with values

- **GIVEN** `config.yaml` has `git.enabled: false` and `branch_prefix: "feat/"`
- **WHEN** `switch` reads config
- **THEN** `git.enabled` is `false` and `branch_prefix` is `feat/`

#### Scenario: Config missing

- **GIVEN** `config.yaml` does not exist
- **WHEN** `switch` reads config
- **THEN** defaults apply: `git.enabled: true`, `branch_prefix: ""`

### Requirement: Output format

`switch` SHALL output a structured summary to stdout. The format is:

```
fab/current → {name}

Stage:  {stage} ({N}/6)
Branch: {name} ({created|checked out|skipped})

Next: {per state table}
```

Branch line is omitted when git is disabled or not in a repo. Deactivation outputs `No active change.` (or `No active change (already blank).`).

Stage number mapping: intake=1, spec=2, tasks=3, apply=4, review=5, hydrate=6.

The `Next:` suggestion SHALL follow the state table in `_context.md`.

#### Scenario: Normal output

- **GIVEN** switch resolves to `260216-a7k2-add-oauth` at spec stage, branch created
- **WHEN** output is emitted
- **THEN** stdout contains:
  ```
  fab/current → 260216-a7k2-add-oauth

  Stage:  spec (2/6)
  Branch: 260216-a7k2-add-oauth (created)

  Next: /fab-continue
  ```

#### Scenario: Deactivation output

- **GIVEN** `--blank` is invoked and `fab/current` existed
- **WHEN** output is emitted
- **THEN** stdout contains `No active change.`

## Caller migration

### Requirement: Preflight migration

`preflight.sh` SHALL replace `source resolve-change.sh` + `resolve_change()` with a subprocess call: `name=$("$CHANGEMAN" resolve "$override")`. The `CHANGEMAN` variable SHALL be set alongside the existing `STAGEMAN` variable. Error handling SHALL preserve the existing guidance messages (appended after changeman's stderr).

#### Scenario: Preflight resolves via changeman

- **GIVEN** `preflight.sh` invokes `$CHANGEMAN resolve "a7k2"`
- **WHEN** changeman exits 0 with `260216-a7k2-add-oauth` on stdout
- **THEN** preflight uses `260216-a7k2-add-oauth` as the change name
- **AND** the rest of preflight proceeds unchanged

#### Scenario: Preflight resolution failure

- **GIVEN** `$CHANGEMAN resolve "zzz"` exits 1
- **WHEN** preflight handles the error
- **THEN** stderr includes changeman's diagnostic plus preflight's guidance ("Provide a more specific name." or "Run /fab-new to start one.")

### Requirement: Batch script migration

`batch-fab-switch-change.sh` and `batch-fab-archive-change.sh` SHALL replace `source resolve-change.sh` + `resolve_change()` with `$CHANGEMAN resolve`. The `CHANGEMAN` variable SHALL point to the lib script. The `RESOLVED_CHANGE_NAME` variable pattern is replaced by capturing stdout.

#### Scenario: Batch switch resolves via changeman

- **GIVEN** `batch-fab-switch-change.sh` calls `$CHANGEMAN resolve "$change"`
- **WHEN** changeman exits 0
- **THEN** the resolved name is captured from stdout and used for worktree creation

#### Scenario: Batch archive resolves via changeman

- **GIVEN** `batch-fab-archive-change.sh` calls `$CHANGEMAN resolve "$change"`
- **WHEN** changeman exits 0
- **THEN** the resolved name is captured from stdout and used for archive filtering

### Requirement: `/fab-switch` skill simplification

The `/fab-switch` skill (`.kit/skills/fab-switch.md`) SHALL be updated to delegate switch operations to `changeman.sh switch` instead of orchestrating 5-6 tool calls. Interactive selection (no-arg flow, multi-match selection) remains in the skill layer. The skill's Argument Flow becomes: `changeman.sh switch "$arg"` + display output.

#### Scenario: Skill delegates to changeman

- **GIVEN** user invokes `/fab-switch a7k2`
- **WHEN** the skill processes the argument
- **THEN** it runs `changeman.sh switch "a7k2"` via a single Bash tool call
- **AND** displays the stdout output

#### Scenario: Skill handles no-arg interactively

- **GIVEN** user invokes `/fab-switch` with no argument
- **WHEN** the skill scans `fab/changes/`
- **THEN** it lists changes and prompts for selection (stays in skill layer)
- **AND** after selection, delegates to `changeman.sh switch "<selected>"`

## Deletion and test migration

### Requirement: Delete resolve-change.sh

`fab/.kit/scripts/lib/resolve-change.sh` SHALL be deleted after all callers are migrated to `changeman.sh resolve`.

#### Scenario: No remaining callers

- **GIVEN** `preflight.sh`, `batch-fab-switch-change.sh`, and `batch-fab-archive-change.sh` all use `$CHANGEMAN resolve`
- **WHEN** the migration is complete
- **THEN** `resolve-change.sh` is deleted
- **AND** no script in `fab/.kit/scripts/` references `resolve-change.sh`

### Requirement: Test migration

The resolve-change test suite (`src/lib/resolve-change/test.bats`) SHALL be migrated into `src/lib/changeman/test.bats`. Tests SHALL be adapted to invoke `changeman.sh resolve` instead of sourcing `resolve_change()`. The `src/lib/resolve-change/` dev directory MAY be deleted after migration. `src/lib/changeman/SPEC-changeman.md` SHALL be updated with `resolve` and `switch` subcommand documentation.

#### Scenario: Resolve tests in changeman suite

- **GIVEN** `src/lib/changeman/test.bats` contains migrated resolve tests
- **WHEN** `bats src/lib/changeman/test.bats` is run
- **THEN** all resolve scenarios pass (exact match, partial, multiple, no match, archive exclusion, missing directory)

## Deprecated Requirements

### Shared Change Resolution Library (in `preflight.md`)

**Reason**: `resolve-change.sh` is absorbed into `changeman.sh resolve`. The standalone sourced library pattern is no longer needed — all callers use the subprocess CLI pattern consistent with stageman.
**Migration**: Replace `source resolve-change.sh; resolve_change()` with `$CHANGEMAN resolve`.

## Design Decisions

1. **yq for config parsing in changeman**: changeman gains a `yq` dependency for reading `config.yaml` (`git.enabled`, `git.branch_prefix`).
   - *Why*: yq is already a required system dependency (via stageman). Using it avoids fragile grep/sed YAML parsing. Keeps config access consistent across scripts.
   - *Rejected*: grep/sed-based parsing — brittle for nested YAML, inconsistent with the yq convention established by stageman.

2. **`resolve` outputs to stdout (not shell variable)**: The `resolve` subcommand prints the resolved name to stdout, consumed via `name=$(changeman.sh resolve ...)`.
   - *Why*: CLI-subprocess convention established by the stageman migration. Eliminates the only remaining sourced library in `lib/`. Compatible with eventual Rust binary replacement.
   - *Rejected*: Keeping the sourced variable-setting pattern — inconsistent with the CLI convention, requires `source` which prevents binary replacement.

3. **switch writes fab/current before git operations**: `fab/current` is written first, then git branch integration runs. Git failure does not roll back the pointer.
   - *Why*: The pointer is the primary output of switch. Git integration is convenience — a failure should not prevent the core operation. The skill already tolerates git failures today.
   - *Rejected*: Atomic pointer+git — adds rollback complexity for no user benefit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `resolve` outputs to stdout, not via shell variable | Confirmed from intake #1. CLI-subprocess convention established by stageman migration; eliminates the only remaining sourced library | S:95 R:90 A:95 D:95 |
| 2 | Certain | `switch` handles git branch integration | Confirmed from intake #2. User explicitly agreed branch logic is deterministic and should be in shell | S:90 R:85 A:90 D:90 |
| 3 | Certain | `resolve-change.sh` is deleted after migration | Confirmed from intake #3. All callers migrate to `changeman.sh resolve`; no reason to keep the standalone file | S:90 R:90 A:90 D:95 |
| 4 | Confident | `switch` uses yq for config.yaml parsing | Upgraded from intake #4. yq is already a system dependency via stageman; grep/sed-based YAML parsing would be fragile and inconsistent | S:80 R:80 A:75 D:75 |
| 5 | Certain | Interactive selection stays in skill layer | Upgraded from intake #5 (was Confident). Shell cannot prompt through Claude's UI — this is a hard constraint, not a design choice | S:95 R:90 A:95 D:95 |
| 6 | Confident | Resolve tests merge into changeman test suite | Confirmed from intake #6. Single test file per script; resolve is now a changeman subcommand | S:80 R:85 A:75 D:80 |
| 7 | Confident | `switch` writes fab/current before git, no rollback on git failure | Derived from intake's switch flow description. Git failure is non-fatal, consistent with current `/fab-switch` behavior | S:85 R:85 A:80 D:75 |
| 8 | Confident | Config missing defaults to git enabled, empty prefix | Reasonable default — matches current `/fab-switch` behavior when config.yaml is absent | S:75 R:85 A:75 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
