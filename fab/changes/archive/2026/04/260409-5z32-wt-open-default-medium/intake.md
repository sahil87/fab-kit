# Intake: wt open — Support "default" as App Value

**Change**: 260409-5z32-wt-open-default-medium
**Created**: 2026-04-09
**Status**: Draft

## Origin

> I want to understand how wt open chooses its default option to open a worktree. The default should generally be the current medium. Is this happening?

Conversational investigation. User confirmed `DetectDefaultApp()` already handles current-medium detection correctly via `TERM_PROGRAM` and session checks. The real gap: there's no way to use that detection non-interactively. `wt open --app` and `wt create --worktree-open` require a specific app name — neither accepts `"default"` to mean "auto-detect the current medium."

User also confirmed `--app` is the right flag name (mirrors macOS `open -a`).

## Why

When scripting or automating worktree workflows (e.g., `fab batch new`, operator spawning), callers need to open worktrees in the current medium without showing an interactive menu. Today the only options are:

1. `--app <specific-name>` — requires hardcoding an app, breaks portability across environments
2. `--worktree-open prompt` — shows interactive menu, unusable in automation
3. `--worktree-open skip` — skips opening entirely

There's no way to say "open with whatever the detected default would be" non-interactively. This forces automation scripts to either skip the open step or hardcode environment-specific app names.

## What Changes

### Add `"default"` as a Recognized App Value

Both `wt open --app` and `wt create --worktree-open` SHALL accept `"default"` as a value. When received:

1. Call `BuildAvailableApps()` to get the available app list
2. Call `DetectDefaultApp(apps)` to resolve the default index
3. If a default is found (index > 0), open with that app (call `SaveLastApp` + `OpenInApp`)
4. If no default is found (index == -1), warn and skip

### `wt open --app default`

In `src/go/wt/cmd/open.go`, the `--app` code path at lines 81-101 currently calls `ResolveApp(appFlag, apps)`. When `appFlag == "default"`, it should instead resolve via `DetectDefaultApp()`.

### `wt create --worktree-open default`

In `src/go/wt/cmd/create.go`, the `else if worktreeOpen != "skip"` branch at lines 273-286 currently calls `ResolveApp(worktreeOpen, apps)`. When `worktreeOpen == "default"`, it should resolve via `DetectDefaultApp()`.

### Helper: `ResolveDefaultApp()`

Extract the shared logic into a new function in `apps.go`:

```go
// ResolveDefaultApp resolves the "default" keyword to an app using DetectDefaultApp.
// Returns the resolved AppInfo or an error if no default can be determined.
func ResolveDefaultApp(apps []AppInfo) (*AppInfo, error) {
    idx := DetectDefaultApp(apps)
    if idx < 1 || idx > len(apps) {
        return nil, fmt.Errorf("no default app detected")
    }
    return &apps[idx-1], nil
}
```

### Test Updates

- `src/go/wt/internal/worktree/apps_test.go` — test `ResolveDefaultApp()` returns the correct app based on environment
- `src/go/wt/cmd/open_test.go` — test `--app default` resolves and opens without menu
- `src/go/wt/cmd/create_test.go` — test `--worktree-open default` resolves and opens without menu

## Affected Memory

- `fab-workflow/distribution`: (modify) Document the new `"default"` keyword for `--app` and `--worktree-open` flags

## Impact

- **`src/go/wt/internal/worktree/apps.go`** — new `ResolveDefaultApp()` helper
- **`src/go/wt/cmd/open.go`** — `"default"` branch before `ResolveApp()` call
- **`src/go/wt/cmd/create.go`** — `"default"` branch before `ResolveApp()` call
- **Test files** — new test cases for the `"default"` keyword
- No config changes, no migration needed

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keyword is `"default"` (case-sensitive, lowercase) | Consistent with `"skip"` and `"prompt"` — all lowercase keywords in `--worktree-open` | S:85 R:95 A:90 D:90 |
| 2 | Certain | Reuse existing `DetectDefaultApp()` logic | Already implements the correct priority chain (TERM_PROGRAM → session → cache → fallback); no need to duplicate | S:90 R:95 A:95 D:95 |
| 3 | Certain | `SaveLastApp` is called when `"default"` resolves | Consistent with menu selection behavior — the resolved app becomes the cached last-app | S:80 R:95 A:90 D:90 |
| 4 | Confident | Warn and skip (not error) when no default detected | Matches the non-fatal pattern in `wt create` open step; erroring would break automation | S:75 R:85 A:80 D:75 |
| 5 | Certain | `--app` flag name is correct as-is | Discussed — user confirmed; mirrors macOS `open -a` convention, no rename needed | S:95 R:95 A:95 D:95 |
| 6 | Confident | Extract `ResolveDefaultApp()` as shared helper | Both `open.go` and `create.go` need the same logic; avoids duplication | S:80 R:90 A:85 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
