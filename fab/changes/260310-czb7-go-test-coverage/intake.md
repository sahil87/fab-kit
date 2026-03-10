# Intake: Go Test Coverage and Backend Priority

**Change**: 260310-czb7-go-test-coverage
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Fix Go test coverage — every fab subcommand should have test cases. Restore `just test-go` and `just test-go-v` justfile targets to run all Go tests (unit + integration), not just the old parity tests which have been removed. Also reverse the backend priority in the `fab` dispatcher from rust > go to go > rust, since Go is now the actively maintained backend.

The Go parity tests (`src/fab-go/test/parity/`) were removed because the bash scripts they tested against no longer exist. This left `just test-go` missing from the justfile. The existing Go unit tests (`./...`) cover some packages but several internal packages have zero test files.

## Why

1. **Coverage gaps**: 6 internal packages have `[no test files]`:
   - `internal/archive` — archive/restore logic
   - `internal/change` — change lifecycle (new, rename, switch, list)
   - `internal/log` — append-only JSONL history logging
   - `internal/preflight` — validation + structured YAML output
   - `internal/resolve` — change reference resolution
   - `internal/score` — confidence scoring from Assumptions tables

2. **No test target**: After removing the parity tests, `just test-go` no longer exists. The existing Go unit tests are only discoverable via manual `go test ./...`.

3. **Parity with Rust**: The Rust binary has 36 integration tests covering all subcommands. The Go binary has unit tests for only `cmd/fab` (panemap, sendkeys), `internal/config`, `internal/hooks`, `internal/status`, and `internal/statusfile`.

4. **Backend priority is backwards**: The `fab` dispatcher (`fab/.kit/bin/fab`) currently prioritizes `fab-rust` over `fab-go`. Since Go is the actively maintained backend and Rust will fall behind, the priority should be reversed to go > rust.

## What Changes

### Restore justfile targets

Add `test-go` and `test-go-v` targets that run all Go tests:

```just
# Run Go unit and integration tests
test-go:
    cd {{go_src}} && go test ./... -count=1

# Run Go tests with verbose output
test-go-v:
    cd {{go_src}} && go test ./... -v -count=1
```

Add `test-go` back to the `test` recipe.

### Add tests for untested packages

For each of the 6 untested packages, add a `*_test.go` file with tests covering the public API:

**`internal/resolve/`** — change reference resolution:
- Resolve via active symlink
- Resolve via 4-char ID
- Resolve via substring
- Resolve via full folder name
- Ambiguous match error
- No match error
- `fab_root()` detection

**`internal/log/`** — JSONL logging:
- Log command event
- Log transition event
- Log review event
- Log confidence event
- Append-only (doesn't overwrite existing entries)
- Timestamp format

**`internal/preflight/`** — validation:
- Valid repo with active change
- Missing config.yaml
- Missing constitution.md
- Missing .fab-status.yaml
- Override change name resolution
- YAML output structure

**`internal/score/`** — confidence scoring:
- Parse Assumptions table from intake
- Parse Assumptions table from spec
- Compute score with formula
- Gate check pass/fail
- Cover factor calculation
- Zero score when unresolved > 0

**`internal/archive/`** — archive/restore:
- Archive a change (move + index update)
- Restore an archived change
- Archive list
- Idempotent index updates
- Date-based directory structure

**`internal/change/`** — change lifecycle:
- Create new change (slug, ID generation)
- Rename change
- Switch active change
- Switch blank
- List changes with stage info
- Resolve change name

### Reverse backend priority in `fab` dispatcher

Update `fab/.kit/bin/fab` to prioritize `fab-go` over `fab-rust`:

```sh
# Default priority: go > rust
if [ -x "$SCRIPT_DIR/fab-go" ]; then
  exec "$SCRIPT_DIR/fab-go" "$@"
fi
if [ -x "$SCRIPT_DIR/fab-rust" ]; then
  exec "$SCRIPT_DIR/fab-rust" "$@"
fi
```

Also update the `--version` detection order to match:

```sh
if [ -x "$SCRIPT_DIR/fab-go" ]; then
  backend="go"
elif [ -x "$SCRIPT_DIR/fab-rust" ]; then
  backend="rust"
fi
```

The `FAB_BACKEND` env var and `.fab-backend` file override remain unchanged — users can still force either backend.

### Existing tests (no changes needed)

These packages already have tests:
- `cmd/fab/` — panemap (5 tests), sendkeys (4 tests)
- `internal/config/` — 4 tests
- `internal/hooks/` — 6 tests
- `internal/status/` — 4 tests (hooks integration)
- `internal/statusfile/` — 4 tests

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document test strategy for Go binary

## Impact

- **justfile**: Restore `test-go`, `test-go-v` targets; add `test-go` to `test` recipe
- **src/fab-go/internal/**: 6 new `*_test.go` files (~600-800 lines total)
- **fab/.kit/bin/fab**: Dispatcher priority reversed (go > rust)
- **No behavioral changes** to the fab commands themselves — additive test coverage + dispatcher priority swap

## Open Questions

- None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `go test ./...` as the test target (not parity tests) | Parity tests removed — bash scripts no longer exist. Unit/integration tests against Go code directly | S:90 R:90 A:95 D:95 |
| 2 | Certain | Test each untested internal package | 6 packages with `[no test files]` — `go test ./...` output confirms the gaps | S:95 R:90 A:90 D:95 |
| 3 | Confident | Use temp directories with fixture copies (same pattern as existing tests) | Consistent with `cmd/fab/panemap_test.go` and `internal/statusfile/statusfile_test.go` patterns | S:75 R:85 A:85 D:80 |
| 4 | Confident | Test public API surface, not internal implementation details | Tests should exercise exported functions via realistic inputs, not reach into private state | S:70 R:80 A:80 D:75 |
| 5 | Certain | Reverse dispatcher priority to go > rust | Go is actively maintained; Rust binary will fall behind. FAB_BACKEND override preserved for manual selection | S:90 R:90 A:90 D:95 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
