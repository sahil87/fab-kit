# Spec: Port Idea Script to Go Binary

**Change**: 260310-pl72-port-idea-to-go
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Removing the existing Bash `idea` script — coexistence is intentional for rollback safety
- Adding new features beyond what the Bash script supports — this is a straight port
- Changing the `fab/backlog.md` line format — backward compatibility is required

## Idea Package: Core Types and Parsing

### Requirement: Idea Struct

The `internal/idea/` package SHALL define an `Idea` struct with fields: `ID` (string, 4-char alphanumeric), `Date` (string, YYYY-MM-DD), `Text` (string), `Done` (bool).

#### Scenario: Parse a valid open idea line

- **GIVEN** a line `- [ ] [a7k2] 2025-06-15: Add dark mode to settings page`
- **WHEN** the line is parsed
- **THEN** the result is `Idea{ID: "a7k2", Date: "2025-06-15", Text: "Add dark mode to settings page", Done: false}`

#### Scenario: Parse a valid done idea line

- **GIVEN** a line `- [x] [e5f6] 2025-06-08: Fix login redirect bug`
- **WHEN** the line is parsed
- **THEN** the result is `Idea{ID: "e5f6", Date: "2025-06-08", Text: "Fix login redirect bug", Done: true}`

#### Scenario: Parse an invalid line

- **GIVEN** a line that does not match the backlog format (e.g., `some random text`)
- **WHEN** the line is parsed
- **THEN** a parse error is returned and the line is skipped

### Requirement: Line Serialization

An `Idea` struct SHALL be serializable back to the markdown line format: `- [ ] [{id}] {date}: {text}` (open) or `- [x] [{id}] {date}: {text}` (done).

#### Scenario: Round-trip serialization

- **GIVEN** a parsed `Idea` struct
- **WHEN** it is serialized back to a line
- **THEN** the output matches the original line format

### Requirement: Query Matching

Query matching SHALL be case-insensitive substring search against both the idea ID and text fields. A query matches if it is a substring of either the ID or the text.

#### Scenario: Match by ID

- **GIVEN** ideas `[a7k2] Add dark mode` and `[c3d4] Fix redirect`
- **WHEN** queried with `a7k2`
- **THEN** only the first idea matches

#### Scenario: Match by text (case-insensitive)

- **GIVEN** ideas `[a7k2] Add dark mode` and `[c3d4] Fix redirect`
- **WHEN** queried with `dark`
- **THEN** only the first idea matches

#### Scenario: Multiple matches

- **GIVEN** ideas `[a7k2] Add dark mode` and `[c3d4] Add light mode`
- **WHEN** queried with `mode`
- **THEN** both ideas match

#### Scenario: No matches

- **GIVEN** ideas `[a7k2] Add dark mode`
- **WHEN** queried with `nonexistent`
- **THEN** zero ideas match

## Idea Package: File Operations

### Requirement: Load Backlog File

The package SHALL load a backlog file by reading all lines, parsing valid idea lines, and skipping non-idea lines (headers, blank lines, comments). The order of ideas SHALL be preserved as-is from the file.

#### Scenario: Load a file with mixed content

- **GIVEN** a file with a header line, blank lines, and valid idea lines
- **WHEN** loaded
- **THEN** only the idea lines are parsed into `Idea` structs; headers and blanks are preserved for write-back

#### Scenario: Load an empty file

- **GIVEN** an empty file
- **WHEN** loaded
- **THEN** zero ideas are returned with no error

#### Scenario: Load a missing file

- **GIVEN** a path to a file that does not exist
- **WHEN** loaded
- **THEN** an appropriate error is returned

### Requirement: Save Backlog File

The package SHALL write the backlog by reconstructing the full file content: non-idea lines are preserved in their original positions, and idea lines are serialized from the current `Idea` slice.

#### Scenario: Save preserves non-idea lines

- **GIVEN** a file with headers and idea lines that was loaded, modified, and saved
- **WHEN** the file is re-read
- **THEN** non-idea lines (headers, blanks) are in their original positions

### Requirement: Auto-Create on Add

When adding a new idea, if the backlog file does not exist, the file and its parent directories SHALL be created automatically.

#### Scenario: Add to nonexistent file

- **GIVEN** no file exists at the backlog path
- **WHEN** `add` is called with a text
- **THEN** the file is created with a single idea line

## Idea Package: CRUD Operations

### Requirement: Add

`Add` SHALL append a new idea line to the end of the backlog file. A random 4-char alphanumeric ID SHALL be generated unless `--id` is provided. The date SHALL default to today unless `--date` is provided.

#### Scenario: Add with defaults

- **GIVEN** an existing backlog file with 2 ideas
- **WHEN** `add "Build search feature"` is called
- **THEN** a new idea line is appended with a generated 4-char ID, today's date, and the given text

#### Scenario: Add with custom ID and date

- **GIVEN** an existing backlog file
- **WHEN** `add "My idea" --id ab12 --date 2025-01-01` is called
- **THEN** the idea line uses ID `ab12` and date `2025-01-01`

#### Scenario: Add with ID collision

- **GIVEN** a backlog file containing idea `[ab12]`
- **WHEN** `add "New idea" --id ab12` is called
- **THEN** an error is returned: `ID 'ab12' already exists`

### Requirement: List

`List` SHALL return ideas filtered by status: open (default), done (`--done`), or all (`-a`). Output defaults to the raw markdown lines. `--json` outputs a JSON array of objects with `id`, `date`, `status`, `text` fields. `--sort` sorts by `id` or `date` (default: `date`). `--reverse` reverses sort order.

#### Scenario: List open ideas (default)

- **GIVEN** a backlog with 2 open and 1 done idea
- **WHEN** `list` is called
- **THEN** only the 2 open ideas are returned

#### Scenario: List with --json

- **GIVEN** a backlog with ideas
- **WHEN** `list --json` is called
- **THEN** output is a JSON array with objects containing `id`, `date`, `status`, `text`

#### Scenario: List with --sort id --reverse

- **GIVEN** ideas with IDs `a1b2`, `c3d4`, `e5f6`
- **WHEN** `list --sort id --reverse` is called
- **THEN** ideas are returned in reverse ID order: `e5f6`, `c3d4`, `a1b2`

#### Scenario: List empty file

- **GIVEN** a backlog file that does not exist
- **WHEN** `list` is called
- **THEN** plain: `No ideas file yet. Add one with: fab idea add "your idea"`, JSON: `[]`

#### Scenario: List no matches for filter

- **GIVEN** a backlog with only open ideas
- **WHEN** `list --done` is called
- **THEN** plain: `No ideas found.`, JSON: `[]`

### Requirement: Show

`Show` SHALL find exactly one idea matching the query and display it. If zero or multiple ideas match, an error is returned. `--json` outputs the single idea as a JSON object.

#### Scenario: Show single match

- **GIVEN** a backlog with idea `[a7k2] Add dark mode`
- **WHEN** `show a7k2` is called
- **THEN** the full idea line is displayed

#### Scenario: Show with --json

- **GIVEN** a backlog with idea `[a7k2] Add dark mode`
- **WHEN** `show a7k2 --json` is called
- **THEN** output is `{"id":"a7k2","date":"...","status":"open","text":"Add dark mode"}`

#### Scenario: Show multiple matches

- **GIVEN** ideas matching `mode`
- **WHEN** `show mode` is called
- **THEN** an error listing the matches is returned with guidance to be more specific

#### Scenario: Show no matches

- **GIVEN** no idea matching `nonexistent`
- **WHEN** `show nonexistent` is called
- **THEN** error: `No idea matching 'nonexistent'`

### Requirement: Done

`Done` SHALL mark a single matching open idea as done by changing `[ ]` to `[x]`.

#### Scenario: Mark open idea as done

- **GIVEN** an open idea `[a7k2] Add dark mode`
- **WHEN** `done a7k2` is called
- **THEN** the line changes to `- [x] [a7k2] ...` and a confirmation is printed

#### Scenario: Done on already-done idea

- **GIVEN** a done idea `[a7k2]`
- **WHEN** `done a7k2` is called (filter: open only)
- **THEN** error: `No idea matching 'a7k2'`

### Requirement: Reopen

`Reopen` SHALL mark a single matching done idea as open by changing `[x]` to `[ ]`.

#### Scenario: Reopen done idea

- **GIVEN** a done idea `[a7k2]`
- **WHEN** `reopen a7k2` is called
- **THEN** the line changes to `- [ ] [a7k2] ...` and a confirmation is printed

#### Scenario: Reopen already-open idea

- **GIVEN** an open idea `[a7k2]`
- **WHEN** `reopen a7k2` is called (filter: done only)
- **THEN** error: `No idea matching 'a7k2'`

### Requirement: Edit

`Edit` SHALL modify the text of a single matching idea. Optional `--id` and `--date` flags allow changing those fields. ID change SHALL be validated for collisions.

#### Scenario: Edit text only

- **GIVEN** idea `[a7k2] 2025-06-15: Add dark mode`
- **WHEN** `edit a7k2 "Add dark mode with toggle"` is called
- **THEN** the line text changes, ID and date preserved

#### Scenario: Edit with --id (no collision)

- **GIVEN** idea `[a7k2]` and no idea `[z9y8]`
- **WHEN** `edit a7k2 "same text" --id z9y8` is called
- **THEN** the ID changes to `z9y8`

#### Scenario: Edit with --id (collision)

- **GIVEN** ideas `[a7k2]` and `[z9y8]`
- **WHEN** `edit a7k2 "text" --id z9y8` is called
- **THEN** error: `ID 'z9y8' already exists`

### Requirement: Remove

`Rm` SHALL delete a single matching idea line from the file. Without `--force`, the Go implementation SHALL require `--force` (no interactive prompt).
<!-- clarified: rm requires --force, no interactive prompt — confirmed from Design Decision #1 and agent-context usage pattern -->

#### Scenario: Remove with --force

- **GIVEN** idea `[a7k2]`
- **WHEN** `rm a7k2 --force` is called
- **THEN** the line is deleted from the file and a confirmation is printed

#### Scenario: Remove without --force

- **GIVEN** idea `[a7k2]`
- **WHEN** `rm a7k2` is called (no --force)
- **THEN** error: `Use --force to confirm deletion`

## Cobra Command Integration

### Requirement: Subcommand Registration

`fab idea` SHALL be registered as a top-level Cobra subcommand in `src/go/fab/cmd/fab/main.go`. It SHALL have subcommands: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`.

#### Scenario: Help output

- **GIVEN** the `fab` binary
- **WHEN** `fab idea --help` is called
- **THEN** all subcommands are listed with descriptions

#### Scenario: Unknown subcommand

- **GIVEN** the `fab` binary
- **WHEN** `fab idea unknown` is called
- **THEN** Cobra returns an error with usage

### Requirement: Git Root Resolution

The `idea` command SHALL resolve the git repository root via `git rev-parse --path-format=absolute --git-common-dir` (matching the Bash script's approach). The backlog file path SHALL be relative to the repo root.

#### Scenario: Run inside a git repo

- **GIVEN** the current directory is inside a git repository
- **WHEN** any `fab idea` subcommand is called
- **THEN** the backlog file is resolved relative to the git repo root

#### Scenario: Run outside a git repo

- **GIVEN** the current directory is not inside a git repository
- **WHEN** any `fab idea` subcommand is called
- **THEN** error: `not in a git repository`

### Requirement: File Override

The `--file` flag SHALL override the backlog file path (relative to git root). The `IDEAS_FILE` environment variable SHALL also override the path. Flag takes precedence over env var, which takes precedence over the default `fab/backlog.md`.

#### Scenario: --file flag

- **GIVEN** `--file backlog/ideas.md`
- **WHEN** `fab idea list --file backlog/ideas.md` is called
- **THEN** the file at `{repo_root}/backlog/ideas.md` is used

#### Scenario: IDEAS_FILE env var

- **GIVEN** `IDEAS_FILE=custom/backlog.md` in environment
- **WHEN** `fab idea list` is called
- **THEN** the file at `{repo_root}/custom/backlog.md` is used

### Requirement: Output Format

All confirmation messages SHALL match the Bash script's format:
- Add: `Added: [{id}] {date}: {text}`
- Done: `Done: - [x] [{id}] {date}: {text}`
- Reopen: `Reopened: - [ ] [{id}] {date}: {text}`
- Edit: `Updated: - [{status}] [{id}] {date}: {text}`
- Rm: `Removed: - [{status}] [{id}] {date}: {text}`

#### Scenario: Add confirmation

- **GIVEN** a successful add operation
- **WHEN** the output is printed
- **THEN** it matches `Added: [{id}] {date}: {text}`

## Test Coverage

### Requirement: Comprehensive Go Tests

Tests SHALL be placed in `src/go/fab/internal/idea/idea_test.go` following the table-driven test pattern used in existing packages (e.g., `internal/change/change_test.go`).

#### Scenario: Tests use temp directories

- **GIVEN** any test that reads/writes files
- **WHEN** the test runs
- **THEN** it uses `t.TempDir()` for file paths, not real git repos

#### Scenario: Tests cover all CRUD operations

- **GIVEN** the test suite
- **WHEN** all tests pass
- **THEN** add, list, show, done, reopen, edit, rm operations are covered

#### Scenario: Tests cover query matching edge cases

- **GIVEN** the test suite
- **WHEN** all tests pass
- **THEN** match-by-ID, match-by-text, case-insensitive, multiple matches, no matches are covered

#### Scenario: Tests cover flag combinations

- **GIVEN** the test suite
- **WHEN** all tests pass
- **THEN** `--json`, `-a`, `--done`, `--sort`, `--reverse`, `--force`, `--id`, `--date` are covered

## Design Decisions

1. **`rm` requires `--force`, no interactive prompt**: In the Bash version, `rm` without `--force` prompts for confirmation via stdin. The Go version runs in non-interactive agent contexts where stdin prompts are unreliable. Requiring `--force` is safer and more predictable.
   - *Why*: Agent-driven workflows cannot reliably answer interactive prompts
   - *Rejected*: Keeping interactive prompt (breaks non-interactive use cases)

2. **`fab idea add` instead of bare positional arg**: The Bash script allows `idea "text"` as shorthand for add. Cobra's subcommand parsing conflicts with positional-arg-as-default. Explicit `add` is idiomatic for Cobra.
   - *Why*: Cobra requires explicit subcommand resolution before arg parsing
   - *Rejected*: Custom pre-run to detect bare args (fragile, non-idiomatic)

3. **Git root via exec, not Go library**: Use `git rev-parse` via `os/exec` rather than a Go git library (e.g., go-git). Matches the Bash script's approach, avoids adding a dependency.
   - *Why*: Consistency with existing Bash behavior; no new dependency
   - *Rejected*: go-git library (adds ~15MB dependency for one call)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Backlog file format unchanged | Confirmed from intake #1 — backward compatibility required, format consumed by `/fab-new` and batch scripts | S:90 R:20 A:95 D:95 |
| 2 | Certain | Use Cobra framework | Confirmed from intake #2 — all existing `fab` subcommands use Cobra | S:95 R:85 A:95 D:95 |
| 3 | Certain | Place in `internal/idea/` | Confirmed from intake #3 — matches `internal/change/`, `internal/status/`, etc. | S:90 R:90 A:95 D:95 |
| 4 | Certain | `fab idea add` instead of bare positional | Confirmed from intake #4 — verified Cobra parsing constraints | S:85 R:85 A:90 D:85 |
| 5 | Confident | Keep old Bash script, don't remove | Confirmed from intake #5 — coexistence is safe, removal is a separate change | S:75 R:90 A:70 D:75 |
| 6 | Certain | Comprehensive Go test coverage | Confirmed from intake #6 — user explicitly requested tests | S:95 R:90 A:90 D:95 |
| 7 | Certain | Use `git rev-parse` for root detection | Confirmed from intake #7 — verified codebase has no Go git library dep | S:85 R:85 A:90 D:90 |
| 8 | Confident | JSON output matches Bash format | Confirmed from intake #8 — `{"id","date","status","text"}` shape | S:65 R:75 A:70 D:75 |
| 9 | Confident | `rm` requires `--force`, no interactive prompt | Go binary runs in agent contexts; interactive stdin is unreliable | S:70 R:80 A:75 D:65 |
| 10 | Certain | File override: --file flag > IDEAS_FILE env > default | Matches Bash script precedence exactly | S:90 R:85 A:90 D:90 |
| 11 | Certain | show supports --json flag | Bash script supports `show --json` — ported as-is | S:85 R:85 A:90 D:90 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
