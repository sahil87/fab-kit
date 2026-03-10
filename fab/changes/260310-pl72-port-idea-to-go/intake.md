# Intake: Port Idea Script to Go Binary

**Change**: 260310-pl72-port-idea-to-go
**Created**: 2026-03-10
**Status**: Draft

## Origin

> User requested: port the "idea" script to a go binary. User also explicitly requested that proper Go-based test cases be added.

## Why

The `idea` command is currently a ~350-line Bash script at `fab/.kit/packages/idea/bin/idea`. It's the only package still implemented as a shell script while the core `fab` CLI has been fully ported to Go (`src/fab-go/`). Porting `idea` to Go:

1. **Consistency** — all fab CLI tooling runs through a single compiled binary, matching the existing pattern (`fab resolve`, `fab change`, `fab status`, etc.)
2. **Testability** — Go's testing infrastructure enables proper unit and integration tests, whereas the Bash script has no test coverage. The user explicitly wants comprehensive Go test cases.
3. **Portability** — eliminates cross-platform `sed -i` workarounds and shell behavior differences between macOS and Linux
4. **Maintainability** — Go's type system and error handling make the CRUD logic easier to modify and debug

If we don't do this, `idea` remains a Bash outlier in an otherwise Go-based CLI, and the backlog management logic stays untested.

## What Changes

### New `idea` subcommand on the `fab` CLI

Add `fab idea` as a new top-level Cobra subcommand in `src/fab-go/`, implementing all current `idea` commands:

| Current shell command | New Go command |
|---|---|
| `idea "text"` | `fab idea add "text"` |
| `idea list` | `fab idea list` |
| `idea show <query>` | `fab idea show <query>` |
| `idea done <query>` | `fab idea done <query>` |
| `idea reopen <query>` | `fab idea reopen <query>` |
| `idea edit <query> "text"` | `fab idea edit <query> "text"` |
| `idea rm <query>` | `fab idea rm <query>` |

**Note**: The bare `idea "text"` add syntax becomes `fab idea add "text"` — the default-to-add behavior is not idiomatic for Cobra subcommands and would conflict with subcommand parsing.

### Backlog file format (unchanged)

The markdown format in `fab/backlog.md` remains identical:

```markdown
- [ ] [a7k2] 2025-06-15: Add dark mode to settings page
- [ ] [c3d4] 2025-06-10: DES-123 Link to a Linear ticket
- [x] [e5f6] 2025-06-08: Fix login redirect bug
```

### Internal package: `internal/idea/`

Create `src/fab-go/internal/idea/` with:

- **`idea.go`** — core types (`Idea` struct with ID, Date, Text, Done fields), parsing logic (line ↔ struct), file read/write, query matching (case-insensitive substring on ID and text)
- **`idea_test.go`** — unit tests covering: parsing valid/invalid lines, query matching (by ID, by text, case-insensitive, multiple matches, no matches), CRUD operations against a temp file, ID collision detection, `--json` output format

### Cobra command wiring: `cmd/fab/idea.go`

Register `ideaCmd()` in `cmd/fab/main.go` alongside existing commands. Each subcommand (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`) wired as a Cobra subcommand of `idea`.

Flags carried over:
- `--file <path>` — override backlog file (relative to git root)
- `list --json` — JSON output
- `list -a` — show all (open + done)
- `list --done` — show only done
- `list --sort <field>` — sort by `id` or `date`
- `list --reverse` — reverse sort order
- `add --id <id>` — custom 4-char ID
- `add --date <date>` — custom date
- `edit --id <id>` — change the ID
- `edit --date <date>` — change the date
- `rm --force` — skip confirmation

### Git root resolution

Reuse the existing `resolve` or `worktree` internal package pattern for finding the git repo root. The Bash script uses `git rev-parse --git-common-dir` — the Go version should use the same approach (exec `git rev-parse`) or leverage an existing helper if one exists in the codebase.

### Dispatcher update

The `fab/.kit/bin/fab` shell dispatcher already routes to `fab-go`. Adding `idea` as a Cobra subcommand means `fab idea ...` works automatically through the dispatcher — no dispatcher changes needed.

### Package removal (deferred)

The old `fab/.kit/packages/idea/bin/idea` Bash script is NOT removed in this change. It can be deprecated and removed in a follow-up once the Go version is validated. The `idea` shell command (via PATH) and `fab idea` Go command can coexist.

### Test requirements

The user explicitly requested proper Go-based test cases. Tests SHALL:

- Cover all CRUD operations (add, list, show, done, reopen, edit, rm)
- Test query matching: by ID, by text, case-insensitive, multiple matches (disambiguation), no matches
- Test edge cases: empty file, missing file (auto-create on add), ID collision, invalid line format
- Test flag combinations: `--json`, `-a`, `--done`, `--sort`, `--reverse`, `--force`
- Use temp directories and files — no reliance on actual git repos
- Follow the existing test patterns in `src/fab-go/internal/` (e.g., table-driven tests with `t.Run`)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document `idea` subcommand as part of the Go binary
- `fab-workflow/distribution`: (modify) Note that `idea` is now part of the compiled binary, not a separate package script

## Impact

- **`src/fab-go/`** — new `internal/idea/` package + `cmd/fab/idea.go` command file
- **`src/fab-go/cmd/fab/main.go`** — add `ideaCmd()` to root command registration
- **`fab/.kit/bin/fab` dispatcher** — no changes needed (routes to `fab-go` already)
- **`fab/.kit/packages/idea/bin/idea`** — unchanged (coexists, deprecated later)
- **`docs/specs/packages.md`** — may need a note about Go port availability

## Open Questions

- Should `rm` without `--force` still prompt for confirmation interactively, or should the Go version require `--force` always (since it may run in non-interactive contexts)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Backlog file format unchanged | Backward compatibility with existing `fab/backlog.md` files; format is consumed by `/fab-new` and batch scripts | S:90 R:20 A:95 D:95 |
| 2 | Certain | Use Cobra framework | All existing `fab` subcommands use Cobra; deviation would be inconsistent | S:95 R:85 A:95 D:95 |
| 3 | Certain | Place in `internal/idea/` | Matches existing package layout (`internal/change/`, `internal/status/`, etc.) | S:90 R:90 A:95 D:95 |
| 4 | Confident | `fab idea add` instead of bare `fab idea "text"` | Cobra's subcommand parsing conflicts with positional-arg-as-default; explicit `add` is idiomatic | S:70 R:85 A:80 D:70 |
| 5 | Confident | Keep old Bash script, don't remove yet | Safer rollout — users can fall back to `idea` while `fab idea` is validated | S:75 R:90 A:70 D:75 |
| 6 | Certain | Comprehensive Go test coverage | User explicitly requested proper Go-based test cases | S:95 R:90 A:90 D:95 |
| 7 | Confident | Use `git rev-parse` for root detection | Matches the Bash script's approach and avoids adding a Go git library dependency | S:70 R:85 A:80 D:80 |
| 8 | Confident | `--json` output matches current Bash format | Existing consumers (if any) expect `{"id","date","status","text"}` shape | S:65 R:75 A:70 D:75 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
