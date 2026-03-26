# Spec: Allow idea shorthand

**Change**: 260326-p4ki-allow-idea-shorthand
**Created**: 2026-03-26
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## CLI: Bare Idea Shorthand

### Requirement: Default-to-Add on Root Command

The `idea` binary's root Cobra command SHALL accept positional arguments. When one or more positional arguments are provided and no subcommand is matched, the root command SHALL delegate to the `add` logic, treating the first positional argument as the idea text.

The shorthand SHALL use the same `idea.Add()` function as the `add` subcommand with default values for optional parameters (empty `customID`, empty `customDate`).

The shorthand SHALL NOT support `--id` or `--date` flags — those remain exclusive to `idea add`.

#### Scenario: Bare shorthand adds an idea

- **GIVEN** `fab/backlog.md` exists
- **WHEN** the user runs `idea "refactor auth middleware"`
- **THEN** a new idea is appended to the backlog with a generated 4-char ID and today's date
- **AND** stdout shows `Added: [{id}] {date}: refactor auth middleware`

#### Scenario: Bare shorthand is equivalent to add subcommand

- **GIVEN** `fab/backlog.md` exists
- **WHEN** the user runs `idea "some text"` and then `idea add "some other text"`
- **THEN** both ideas are added with the same format and behavior
- **AND** the only difference is the text content

#### Scenario: No arguments shows help

- **GIVEN** the `idea` binary is invoked
- **WHEN** no arguments and no subcommand are provided
- **THEN** the root command's help text is displayed (existing Cobra default behavior, unchanged)

#### Scenario: Known subcommands still work

- **GIVEN** the `idea` binary is invoked
- **WHEN** the user runs `idea add "text"`, `idea list`, `idea done <query>`, or any other known subcommand
- **THEN** the subcommand executes normally (no regression)

### Requirement: Persistent Flags on Shorthand

The `--main` and `--file` persistent flags SHALL be available when using the bare shorthand, since they are defined on the root command and Cobra propagates them.

#### Scenario: Shorthand with --main flag

- **GIVEN** the user is in a linked worktree
- **WHEN** the user runs `idea --main "fix login bug"`
- **THEN** the idea is added to the main worktree's `fab/backlog.md`

### Requirement: Error on Empty Text

When the root command's `RunE` is invoked with positional arguments, it SHALL validate that the joined text is non-empty after trimming. If empty, it SHALL return an error.

#### Scenario: Empty string argument

- **GIVEN** the user runs `idea ""`
- **WHEN** the root command's `RunE` processes the args
- **THEN** an error is returned (not a no-op)

## Documentation: CLI Reference Updates

### Requirement: Update _cli-external.md

`fab/.kit/skills/_cli-external.md` SHALL document the bare shorthand in the idea section. The subcommand table SHALL include a row for the shorthand usage: `idea <text>` as equivalent to `idea add <text>`.

#### Scenario: _cli-external.md reflects shorthand

- **GIVEN** the spec is implemented
- **WHEN** a reader consults `_cli-external.md`'s idea section
- **THEN** the bare shorthand is documented alongside the `add` subcommand

### Requirement: Update docs/specs/packages.md

`docs/specs/packages.md` SHALL mention the bare shorthand in the idea section's command table or common workflows.

#### Scenario: packages.md reflects shorthand

- **GIVEN** the spec is implemented
- **WHEN** a reader consults `docs/specs/packages.md`'s idea section
- **THEN** the bare shorthand is documented

## Design Decisions

1. **Root `RunE` with `Args: cobra.ArbitraryArgs`**: The root command gets a `RunE` handler and `Args: cobra.ArbitraryArgs`. When args are present, it delegates to `idea.Add()`. Cobra resolves subcommands before falling through to `RunE`, so `idea list` still dispatches to the `list` subcommand — only unrecognized first args hit `RunE`.
   - *Why*: This is Cobra's standard pattern for "default command" behavior. It preserves subcommand routing and persistent flag parsing.
   - *Rejected*: `DisableFlagParsing` on root — would break `--main` and `--file` persistent flags. Custom `PersistentPreRunE` with subcommand detection — over-engineered for this use case.

2. **Reuse `resolveFile()` and `idea.Add()` directly**: The root `RunE` calls `resolveFile()` (already a package-level function in `cmd/`) and `idea.Add()` with empty strings for `customID` and `customDate` — the same defaults `addCmd` uses.
   - *Why*: Zero code duplication. The add logic is already factored into the `idea` package.
   - *Rejected*: Extracting a shared helper — unnecessary indirection for a 5-line function.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Shorthand delegates to `idea.Add()` with same defaults | Confirmed from intake #1 — direct reuse of existing add logic | S:90 R:95 A:95 D:95 |
| 2 | Certain | No `--id`/`--date` flags on bare shorthand | Confirmed from intake #2 — shorthand is for quick capture | S:85 R:90 A:90 D:90 |
| 3 | Certain | Root `RunE` with `Args: cobra.ArbitraryArgs` | Upgraded from intake #3 Confident — verified Cobra dispatches subcommands before `RunE`, so no conflict | S:85 R:90 A:85 D:85 |
| 4 | Certain | Update `_cli-external.md` and `packages.md` | Confirmed from intake #4 — constitution requires docs updates | S:90 R:95 A:95 D:95 |
| 5 | Certain | `--main` and `--file` work with shorthand | Cobra persistent flags propagate to root `RunE` automatically | S:90 R:95 A:95 D:95 |
| 6 | Certain | No args = show help (existing behavior unchanged) | Cobra default: no `RunE` args and no subcommand → help. We only set `RunE` to fire when `len(args) > 0` | S:90 R:95 A:90 D:95 |

6 assumptions (6 certain, 0 confident, 0 tentative, 0 unresolved).
