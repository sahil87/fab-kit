# Spec: Go Test Coverage and Backend Priority

**Change**: 260310-czb7-go-test-coverage
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing behavior of any fab subcommand — this is additive test coverage only
- Testing private/unexported functions — tests target public API surface
- Achieving 100% line coverage — focus on meaningful behavioral coverage of exported functions
- Adding tests for the `internal/worktree` package (not listed in intake)

## Testing: Justfile Targets

### Requirement: Restore `test-go` and `test-go-v` justfile targets

The justfile SHALL include `test-go` and `test-go-v` targets that run all Go tests via `go test ./... -count=1`. The `test-go` target SHALL be included in the `test` recipe.

#### Scenario: Run Go tests via justfile
- **GIVEN** a justfile with `test-go` defined
- **WHEN** a user runs `just test-go`
- **THEN** all Go unit tests execute via `cd src/fab-go && go test ./... -count=1`
- **AND** the exit code reflects test pass/fail status

#### Scenario: Verbose Go tests
- **GIVEN** a justfile with `test-go-v` defined
- **WHEN** a user runs `just test-go-v`
- **THEN** all Go tests execute with verbose output via `cd src/fab-go && go test ./... -v -count=1`

#### Scenario: Combined test target includes Go
- **GIVEN** the `test` recipe in the justfile
- **WHEN** a user runs `just test`
- **THEN** Go tests are included alongside other test suites

## Testing: resolve Package

### Requirement: Change reference resolution via `resolve.ToFolder`

`resolve.ToFolder(fabRoot, override)` SHALL resolve a change reference (4-char ID, substring, full folder name) to the canonical folder name. When `override` is empty, it SHALL read the `.fab-status.yaml` symlink.

#### Scenario: Resolve via active symlink
- **GIVEN** a `.fab-status.yaml` symlink pointing to `fab/changes/260310-abcd-my-change/.status.yaml`
- **WHEN** `ToFolder(fabRoot, "")` is called with no override
- **THEN** it returns `"260310-abcd-my-change"`

#### Scenario: Resolve via 4-char ID
- **GIVEN** a change folder `260310-abcd-my-change` exists in `fab/changes/`
- **WHEN** `ToFolder(fabRoot, "abcd")` is called
- **THEN** it returns `"260310-abcd-my-change"`

#### Scenario: Resolve via substring
- **GIVEN** a change folder `260310-abcd-my-change` exists
- **WHEN** `ToFolder(fabRoot, "my-change")` is called
- **THEN** it returns `"260310-abcd-my-change"`

#### Scenario: Resolve via full folder name
- **GIVEN** a change folder `260310-abcd-my-change` exists
- **WHEN** `ToFolder(fabRoot, "260310-abcd-my-change")` is called
- **THEN** it returns `"260310-abcd-my-change"`

#### Scenario: Ambiguous match error
- **GIVEN** two folders `260310-abcd-my-change` and `260310-efgh-my-other-change` exist
- **WHEN** `ToFolder(fabRoot, "my")` is called
- **THEN** it returns an error containing "Multiple changes match"

#### Scenario: No match error
- **GIVEN** no change folders match the override
- **WHEN** `ToFolder(fabRoot, "nonexistent")` is called
- **THEN** it returns an error containing "No change matches"

### Requirement: Change ID extraction via `resolve.ExtractID`

`ExtractID(folder)` SHALL extract the 4-char change ID from a `YYMMDD-XXXX-slug` folder name.

#### Scenario: Valid folder name
- **GIVEN** folder name `"260310-abcd-my-change"`
- **WHEN** `ExtractID` is called
- **THEN** it returns `"abcd"`

### Requirement: FabRoot detection via `resolve.FabRoot`

`FabRoot()` SHALL search upward from the current directory for a `fab/` directory and return the repo root path.

#### Scenario: FabRoot from nested directory
- **GIVEN** a repo with `fab/` at the root
- **WHEN** `FabRoot()` is called from a nested subdirectory
- **THEN** it returns the repo root path containing `fab/`

### Requirement: Path construction helpers

`ToDir`, `ToStatus`, `ToAbsDir`, and `ToAbsStatus` SHALL compose paths from a resolved folder name.

#### Scenario: ToDir returns relative change directory
- **GIVEN** a resolvable change reference
- **WHEN** `ToDir(fabRoot, override)` is called
- **THEN** it returns `"fab/changes/{folder}/"`

#### Scenario: ToStatus returns relative status path
- **GIVEN** a resolvable change reference
- **WHEN** `ToStatus(fabRoot, override)` is called
- **THEN** it returns `"fab/changes/{folder}/.status.yaml"`

## Testing: log Package

### Requirement: JSONL event logging

The `log` package SHALL append JSON-Lines entries to `.history.jsonl` in the change directory. Each entry SHALL include an ISO 8601 timestamp and event type.

#### Scenario: Log command event
- **GIVEN** a valid change directory with `.history.jsonl`
- **WHEN** `Command(fabRoot, "fab-continue", "abcd", "")` is called
- **THEN** a JSON line is appended with `"event": "command"`, `"cmd": "fab-continue"`, and an ISO 8601 timestamp

#### Scenario: Log transition event
- **GIVEN** a valid change directory
- **WHEN** `Transition(fabRoot, "abcd", "spec", "finish", "", "", "fab-ff")` is called
- **THEN** a JSON line is appended with `"event": "transition"`, `"stage": "spec"`, `"action": "finish"`
- **AND** the `driver` field is `"fab-ff"`

#### Scenario: Log review event
- **GIVEN** a valid change directory
- **WHEN** `Review(fabRoot, "abcd", "passed", "")` is called
- **THEN** a JSON line is appended with `"event": "review"`, `"result": "passed"`

#### Scenario: Log confidence event
- **GIVEN** a valid change directory
- **WHEN** `ConfidenceLog(fabRoot, "abcd", 4.2, "+0.3", "spec")` is called
- **THEN** a JSON line is appended with `"event": "confidence"`, `"score": 4.2`

#### Scenario: Append-only behavior
- **GIVEN** a `.history.jsonl` file with existing entries
- **WHEN** a new event is logged
- **THEN** the existing entries are preserved and the new entry is appended

#### Scenario: Optional fields omitted when empty
- **GIVEN** a command event with empty `args`
- **WHEN** `Command(fabRoot, "fab-status", "abcd", "")` is called
- **THEN** the JSON line does NOT contain an `"args"` field

## Testing: preflight Package

### Requirement: Pre-operation validation

`preflight.Run(fabRoot, changeOverride)` SHALL validate project initialization, resolve the active change, and return a structured `Result`.

#### Scenario: Valid repo with active change
- **GIVEN** a valid project with `config.yaml`, `constitution.md`, active `.fab-status.yaml`, and a valid change directory
- **WHEN** `Run(fabRoot, "")` is called
- **THEN** it returns a `Result` with populated `ID`, `Name`, `ChangeDir`, `Stage`, `Progress`, `Checklist`, and `Confidence`

#### Scenario: Missing config.yaml
- **GIVEN** a repo root without `fab/project/config.yaml`
- **WHEN** `Run(fabRoot, "")` is called
- **THEN** it returns an error mentioning `config.yaml`

#### Scenario: Missing constitution.md
- **GIVEN** a repo with `config.yaml` but no `fab/project/constitution.md`
- **WHEN** `Run(fabRoot, "")` is called
- **THEN** it returns an error mentioning `constitution.md`

#### Scenario: Missing active change
- **GIVEN** a valid project with no `.fab-status.yaml` symlink and no override
- **WHEN** `Run(fabRoot, "")` is called
- **THEN** it returns an error about no active change

#### Scenario: Override change name resolution
- **GIVEN** a valid project with change folder `260310-abcd-my-change`
- **WHEN** `Run(fabRoot, "abcd")` is called with an override
- **THEN** it returns a `Result` with `Name: "260310-abcd-my-change"` without requiring `.fab-status.yaml`

### Requirement: YAML output formatting

`FormatYAML(result)` SHALL produce YAML matching the expected preflight output format.

#### Scenario: Formatted output structure
- **GIVEN** a valid `Result` struct
- **WHEN** `FormatYAML` is called
- **THEN** the output contains `id:`, `name:`, `change_dir:`, `stage:`, `progress:`, `checklist:`, `confidence:` fields

## Testing: score Package

### Requirement: Assumptions table parsing and confidence scoring

`score.Compute(fabRoot, changeArg, stage)` SHALL parse the Assumptions table from the relevant artifact (`intake.md` or `spec.md`), count grades, compute a confidence score, and persist results to `.status.yaml`.

#### Scenario: Compute score from spec with all certain
- **GIVEN** a `spec.md` with 5 Certain assumptions and 0 Tentative/Unresolved
- **WHEN** `Compute(fabRoot, "abcd", "")` is called
- **THEN** it returns `Score: 5.0` (no penalties)

#### Scenario: Compute score with confident assumptions
- **GIVEN** a `spec.md` with 3 Certain and 2 Confident assumptions
- **WHEN** `Compute` is called
- **THEN** the score reflects the `5.0 - 0.3 * confident` penalty formula

#### Scenario: Zero score when unresolved > 0
- **GIVEN** a `spec.md` with 1 Unresolved assumption
- **WHEN** `Compute` is called
- **THEN** it returns `Score: 0.0`

#### Scenario: Cover factor reduces score for thin specs
- **GIVEN** a `spec.md` for a `feat` change with only 3 decisions (expected_min=7)
- **WHEN** `Compute` is called
- **THEN** the score is reduced by `cover = 3/7` factor

### Requirement: Gate checking

`score.CheckGate(fabRoot, changeArg, stage)` SHALL compare the computed confidence score against the per-type threshold and return a pass/fail result.

#### Scenario: Gate check passes
- **GIVEN** a `fix` change type with confidence score 3.5
- **WHEN** `CheckGate(fabRoot, "abcd", "")` is called
- **THEN** it returns `Gate: "pass"` (threshold for fix is 2.0)

#### Scenario: Gate check fails
- **GIVEN** a `feat` change type with confidence score 2.5
- **WHEN** `CheckGate(fabRoot, "abcd", "")` is called
- **THEN** it returns `Gate: "fail"` (threshold for feat is 3.0)

#### Scenario: Intake gate with fixed threshold
- **GIVEN** any change with intake score 2.5
- **WHEN** `CheckGate(fabRoot, "abcd", "intake")` is called
- **THEN** it returns `Gate: "fail"` (fixed threshold 3.0)

### Requirement: SRAD dimension parsing

When the Assumptions table includes a `Scores` column with `S:nn R:nn A:nn D:nn` entries, `Compute` SHALL extract and average the per-dimension scores.

#### Scenario: Parse dimension scores
- **GIVEN** a `spec.md` with Assumptions containing `S:80 R:90 A:70 D:85`
- **WHEN** `Compute` is called
- **THEN** the result includes `MeanS`, `MeanR`, `MeanA`, `MeanD` averages

## Testing: archive Package

### Requirement: Change archival

`archive.Archive(fabRoot, changeArg, description)` SHALL move a change folder to `archive/yyyy/mm/`, update `archive/index.md`, and clear the active pointer if the archived change was active.

#### Scenario: Archive a change
- **GIVEN** an active change `260310-abcd-my-change` in `fab/changes/`
- **WHEN** `Archive(fabRoot, "abcd", "Completed feature")` is called
- **THEN** the folder is moved to `fab/changes/archive/2026/03/260310-abcd-my-change/`
- **AND** `archive/index.md` is updated with the change entry
- **AND** the `.fab-status.yaml` symlink is removed

#### Scenario: Archive list
- **GIVEN** archived changes in `fab/changes/archive/`
- **WHEN** `List(fabRoot)` is called
- **THEN** it returns folder names of all archived changes

### Requirement: Change restoration

`archive.Restore(fabRoot, changeArg, doSwitch)` SHALL move a change from the archive back to `fab/changes/` and remove it from `archive/index.md`.

#### Scenario: Restore an archived change
- **GIVEN** an archived change `260310-abcd-my-change` in the archive
- **WHEN** `Restore(fabRoot, "abcd", false)` is called
- **THEN** the folder is moved back to `fab/changes/260310-abcd-my-change/`
- **AND** the index entry is removed

#### Scenario: Restore with switch
- **GIVEN** an archived change
- **WHEN** `Restore(fabRoot, "abcd", true)` is called
- **THEN** the folder is restored AND `.fab-status.yaml` symlink is created

## Testing: change Package

### Requirement: Create new change

`change.New(fabRoot, slug, changeID, logArgs)` SHALL create a new change directory with a valid folder name (`YYMMDD-XXXX-slug`), initialize `.status.yaml` from template, and start the intake stage.

#### Scenario: Create with valid slug
- **GIVEN** a valid project setup and slug `"my-feature"`
- **WHEN** `New(fabRoot, "my-feature", "", "")` is called
- **THEN** a directory `fab/changes/YYMMDD-XXXX-my-feature/` is created
- **AND** `.status.yaml` is initialized from template
- **AND** intake stage is started

#### Scenario: Create with explicit change ID
- **GIVEN** a valid slug and explicit 4-char ID `"ab12"`
- **WHEN** `New(fabRoot, "my-feature", "ab12", "")` is called
- **THEN** the folder name contains the explicit ID: `YYMMDD-ab12-my-feature`

#### Scenario: Invalid slug rejected
- **GIVEN** a slug with invalid characters `"my feature!"`
- **WHEN** `New(fabRoot, "my feature!", "", "")` is called
- **THEN** it returns an error about invalid slug format

#### Scenario: ID collision detected
- **GIVEN** an existing change with ID `"ab12"`
- **WHEN** `New(fabRoot, "other-thing", "ab12", "")` is called
- **THEN** it returns an error about ID collision

### Requirement: Rename change

`change.Rename(fabRoot, currentFolder, newSlug)` SHALL update the folder name, `.status.yaml` name field, and symlink if the change is active.

#### Scenario: Rename active change
- **GIVEN** an active change `260310-abcd-old-name`
- **WHEN** `Rename(fabRoot, "260310-abcd-old-name", "new-name")` is called
- **THEN** the folder is renamed to `260310-abcd-new-name`
- **AND** `.status.yaml` name field is updated
- **AND** `.fab-status.yaml` symlink is updated

### Requirement: Switch active change

`change.Switch(fabRoot, name)` SHALL create or update the `.fab-status.yaml` symlink to point to the resolved change's `.status.yaml`.

#### Scenario: Switch to a change
- **GIVEN** a change `260310-abcd-my-change` exists
- **WHEN** `Switch(fabRoot, "abcd")` is called
- **THEN** `.fab-status.yaml` symlink points to `fab/changes/260310-abcd-my-change/.status.yaml`

#### Scenario: Switch blank (deactivate)
- **GIVEN** an active `.fab-status.yaml` symlink
- **WHEN** `SwitchBlank(fabRoot)` is called
- **THEN** the symlink is removed

### Requirement: List changes

`change.List(fabRoot, archive)` SHALL return change entries with stage, state, score, and indicative flag.

#### Scenario: List active changes
- **GIVEN** two changes exist in `fab/changes/`
- **WHEN** `List(fabRoot, false)` is called
- **THEN** it returns entries for both changes with format `name:stage:state:score:indicative`

## Dispatcher: Backend Priority

### Requirement: Reverse default backend priority to go > rust

The `fab` shell dispatcher SHALL check for `fab-go` before `fab-rust` in its default priority chain. The `FAB_BACKEND` env var and `.fab-backend` file override SHALL continue to work unchanged.

#### Scenario: Default priority with both backends
- **GIVEN** both `fab-go` and `fab-rust` are present and executable
- **WHEN** `fab` is invoked without `FAB_BACKEND` or `.fab-backend`
- **THEN** `fab-go` is executed

#### Scenario: Default priority with only rust
- **GIVEN** only `fab-rust` is present
- **WHEN** `fab` is invoked
- **THEN** `fab-rust` is executed as fallback

#### Scenario: Override still works
- **GIVEN** both backends are present
- **WHEN** `FAB_BACKEND=rust fab` is invoked
- **THEN** `fab-rust` is executed despite go being default

#### Scenario: Version shows correct backend
- **GIVEN** both backends present, no override
- **WHEN** `fab --version` is invoked
- **THEN** output shows `(go backend)` — detection order matches execution priority

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `go test ./...` as the test runner | Confirmed from intake #1 — parity tests removed, direct Go tests are the standard | S:90 R:90 A:95 D:95 |
| 2 | Certain | Test all 6 untested internal packages | Confirmed from intake #2 — `go test ./...` output confirms gaps | S:95 R:90 A:90 D:95 |
| 3 | Confident | Use temp directories with fixture copies (same pattern as existing tests) | Confirmed from intake #3 — consistent with panemap_test.go, statusfile_test.go, config_test.go patterns | S:80 R:85 A:85 D:80 |
| 4 | Confident | Test public API surface only | Confirmed from intake #4 — tests exercise exported functions, not private state | S:70 R:80 A:80 D:75 |
| 5 | Certain | Reverse dispatcher priority to go > rust | Confirmed from intake #5 — Go is actively maintained | S:90 R:90 A:90 D:95 |
| 6 | Certain | No behavioral changes to fab subcommands | Scope is additive tests + dispatcher priority — spec explicitly excludes behavior changes | S:95 R:95 A:90 D:95 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
