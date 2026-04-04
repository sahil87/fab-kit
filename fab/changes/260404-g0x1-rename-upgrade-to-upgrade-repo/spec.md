# Spec: Rename upgrade to upgrade-repo

**Change**: 260404-g0x1-rename-upgrade-to-upgrade-repo
**Created**: 2026-04-05
**Affected memory**:
- `docs/memory/fab-workflow/distribution.md`
- `docs/memory/fab-workflow/kit-architecture.md`
- `docs/memory/fab-workflow/migrations.md`
- `docs/memory/fab-workflow/configuration.md`

## Non-Goals

- Rename the Go function `Upgrade()` in `internal/upgrade.go` — internal names don't face users; renaming adds churn without user benefit
- Add a backward-compatibility alias for `fab upgrade` — clean break preferred given low usage frequency
- Change any behavior of the upgrade operation itself — this is a pure rename

---

## CLI: Subcommand Rename

### Requirement: upgrade-repo Command Registration

The `fab-kit` binary SHALL register the workspace upgrade command as `upgrade-repo` (not `upgrade`). All cobra command definitions SHALL use `upgrade-repo` as the `Use` field.

#### Scenario: User runs fab upgrade-repo (no version)
- **GIVEN** the user is in a fab-managed repo
- **WHEN** the user runs `fab upgrade-repo`
- **THEN** the command resolves, downloads the latest version if needed, updates `fab_version` in `config.yaml`, and calls `Sync()` — identical behavior to the former `fab upgrade`

#### Scenario: User runs fab upgrade-repo with explicit version
- **GIVEN** the user is in a fab-managed repo
- **WHEN** the user runs `fab upgrade-repo 0.44.0`
- **THEN** the command installs version 0.44.0, updating `fab_version` and re-syncing

#### Scenario: User runs old fab upgrade command
- **GIVEN** the user runs the old `fab upgrade` command
- **WHEN** `fab` receives `upgrade` as the first argument
- **THEN** `upgrade` is NOT in the `fabKitArgs` allowlist, so it routes to `fab-go`, which will return an unknown-command error

### Requirement: fab Router Allowlist

The `fab` router (`cmd/fab/main.go`) SHALL update its `fabKitArgs` allowlist to contain `upgrade-repo` instead of `upgrade`. This ensures `fab upgrade-repo` is routed to `fab-kit` and not forwarded to `fab-go`.

#### Scenario: Router dispatches upgrade-repo to fab-kit
- **GIVEN** the user runs `fab upgrade-repo [version]`
- **WHEN** the `fab` router receives the command
- **THEN** `fabKitArgs["upgrade-repo"]` is `true`, the command is dispatched to `fab-kit`

#### Scenario: Router does not dispatch plain upgrade
- **GIVEN** the user runs `fab upgrade`
- **WHEN** the `fab` router receives the command
- **THEN** `fabKitArgs["upgrade"]` is `false` (key absent), the command is forwarded to `fab-go`

### Requirement: Help Text

The `fab` router's built-in help text (`printHelp()`) SHALL display `upgrade-repo` in the workspace commands section. The `fab-kit` root command's `Short` description SHALL reference `upgrade-repo`.

#### Scenario: User runs fab --help
- **GIVEN** the user runs `fab --help`
- **WHEN** the help output is rendered
- **THEN** the workspace commands section lists `upgrade-repo` (not `upgrade`) with its description

---

## Tests: Updated Expectations

### Requirement: Router Test

The `TestFabKitArgs` test in `cmd/fab/main_test.go` SHALL verify `upgrade-repo` is in `fabKitArgs` and `upgrade` is NOT in `fabKitArgs`.

#### Scenario: TestFabKitArgs passes with renamed command
- **GIVEN** `fabKitArgs` has been updated to `upgrade-repo`
- **WHEN** `TestFabKitArgs` runs
- **THEN** the test passes for `upgrade-repo` and does not check for plain `upgrade`

### Requirement: fab-kit Commands Test

The `TestFabKitCommands` test in `cmd/fab-kit/main_test.go` SHALL verify `upgrade-repo` is in `fabKitCommands` and `upgrade` is NOT expected.

#### Scenario: TestFabKitCommands passes with renamed command
- **GIVEN** `fabKitCommands` has been updated to `upgrade-repo`
- **WHEN** `TestFabKitCommands` runs
- **THEN** the test passes

---

## Documentation: All User-Facing References

### Requirement: README Update

`README.md` SHALL use `fab upgrade-repo` in the "Updating from a previous version" section. Both the no-argument form and the explicit-version form SHALL be shown. The tool table entry at line 85 (workspace lifecycle description) SHALL also be updated to list `upgrade-repo` instead of `upgrade`.

<!-- clarified: README also has a tool table at line 85 referencing `init`/`upgrade`/`sync` — this inline description must also be updated to `upgrade-repo` -->

#### Scenario: Developer reads README upgrade section
- **GIVEN** the README "Updating from a previous version" section
- **WHEN** the developer reads the code block
- **THEN** they see `fab upgrade-repo` and `fab upgrade-repo 0.44.0` (not `fab upgrade`)

#### Scenario: Developer reads README tool table
- **GIVEN** the README prerequisites tool table
- **WHEN** the developer reads the fab-kit row
- **THEN** the workspace lifecycle list reads `init`/`upgrade-repo`/`sync` (not `upgrade`)

### Requirement: Memory File Updates

The four affected memory files SHALL replace all user-facing `fab upgrade` references with `fab upgrade-repo`. **Exception**: Historical changelog entries (rows in `## Changelog` tables) SHALL NOT be modified — they are immutable records of what was true at the time of the change.

The affected memory files and the specific reference types:
- `docs/memory/fab-workflow/distribution.md` — requirements prose, scenarios, command examples, design decisions
- `docs/memory/fab-workflow/kit-architecture.md` — requirements prose, command examples
- `docs/memory/fab-workflow/migrations.md` — requirements prose, design decisions
- `docs/memory/fab-workflow/configuration.md` — requirements prose

#### Scenario: Agent reads distribution.md for upgrade flow
- **GIVEN** `distribution.md` has been updated
- **WHEN** the agent reads the "`fab upgrade-repo` (Shim Subcommand)" section
- **THEN** the heading and all prose within reference `fab upgrade-repo`

#### Scenario: Historical changelog rows untouched
- **GIVEN** `distribution.md` has been updated
- **WHEN** an agent reads the `## Changelog` section
- **THEN** entries predating this change still reference `fab upgrade` (they are historical records)

### Requirement: Skill Source Updates

`src/kit/skills/fab-setup.md` SHALL replace the `fab upgrade` reference in the kit-not-found error message with `fab upgrade-repo`.

`src/kit/skills/_cli-fab.md` SHALL update the workspace commands routing description and the `fab-kit` table entry to reference `upgrade-repo` instead of `upgrade`.

#### Scenario: fab-setup error message
- **GIVEN** the kit cache check in `/fab-setup` fails
- **WHEN** the error message is displayed
- **THEN** it reads: `Kit not found. Run 'fab sync' or 'fab upgrade-repo' to populate the cache.`

#### Scenario: _cli-fab routing table
- **GIVEN** an agent reads `_cli-fab.md` for routing conventions
- **WHEN** it checks the workspace commands list
- **THEN** `upgrade-repo` appears (not `upgrade`) in the routing list

### Requirement: Deployed Skill Copy

Since `src/kit/skills/_cli-fab.md` is the canonical source and `.claude/skills/_cli-fab/SKILL.md` is the deployed copy, BOTH SHALL be updated in this change. This keeps the deployed copy consistent with the canonical source within the worktree until the next `fab sync`.

#### Scenario: Deployed skill in sync with source
- **GIVEN** both files are updated
- **WHEN** `fab sync` runs again
- **THEN** the deployed copy already matches the source — no regression

---

## Design Decisions

1. **Historical changelog entries preserved**: Command references in `## Changelog` table rows are immutable records. Updating them would misrepresent what was true at that time. Rejected: updating all occurrences uniformly — historically inaccurate.

2. **Internal Go function name unchanged**: `Upgrade()` in `internal/upgrade.go` is not user-facing. Renaming adds churn without user benefit, and would require updating the comment. The comment on line 10 IS updated (it refers to the CLI command), but the function signature stays `Upgrade()`. Rejected: renaming to `UpgradeRepo()` — no benefit.

3. **No backward-compatibility alias**: Adding `fabKitArgs["upgrade"] = true` alongside `upgrade-repo` would silently route `fab upgrade` to `fab-kit`, masking the rename. A clean break lets users discover the new name immediately via an error. Rejected: alias — perpetuates the confusion.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `upgrade-repo`, not other candidates | Confirmed from intake #1 — user explicitly chose `upgrade-repo` over `use`, `pin`, `repo-upgrade` | S:95 R:90 A:95 D:95 |
| 2 | Certain | `fab update` remains unchanged | Confirmed from intake #2 — user confirmed it's a cross-tool pattern | S:95 R:90 A:95 D:95 |
| 3 | Certain | Historical change artifacts not updated | Confirmed from intake #3 — constitution principle: change artifacts are transient records | S:90 R:95 A:90 D:95 |
| 4 | Certain | Historical changelog entries in memory docs not updated | Same principle as #3: changelog rows are immutable records of past state | S:90 R:95 A:90 D:95 |
| 5 | Confident | Go function name `Upgrade()` stays as-is; only comment updated | Confirmed from intake #5 — internal naming doesn't face users; renaming adds churn. Comment on line 10 updated (CLI reference) | S:65 R:85 A:80 D:75 |
| 6 | Confident | No backward-compatibility alias for `fab upgrade` | Confirmed from intake #4 — clean break preferred over aliases | S:60 R:75 A:70 D:70 |
| 7 | Confident | Deployed skill copy `.claude/skills/_cli-fab/SKILL.md` updated alongside source | Context.md states canonical source is `src/kit/skills/`; deployed copy updated for worktree consistency until next sync | S:70 R:85 A:80 D:75 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
