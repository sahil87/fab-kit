# Intake: Operator Spawn Add Fab Sync

**Change**: 260405-xh08-operator-spawn-add-fab-sync
**Created**: 2026-04-06
**Status**: Draft

## Origin

> fix: operator spawn sequence missing fab sync — skills not deployed before agent start causes "Unknown skill" on initial slash command

One-shot invocation. Investigation context added during intake (see Why section).

The user reported that operator-spawned agents sometimes fail with "Unknown skill" on their first slash command. The initial hypothesis was that the spawn sequence was missing `fab sync` entirely. Investigation of `src/go/wt/cmd/create.go` revealed the real root cause: `fab sync` IS run by `wt create --non-interactive` via `RunWorktreeSetup`, but the `--reuse` path hits an **early return** that bypasses the init block.

## Why

### How `wt create` Normally Deploys Skills

`wt create --non-interactive` runs `fab sync` (via `$WORKTREE_INIT_SCRIPT`) in the new worktree as the final step of creation — `RunWorktreeSetup` sets `cmd.Dir = wtPath`, so skills are deployed to the new worktree's `.claude/skills/`. For fresh worktrees, this works correctly.

### The `--reuse` Early Return Bug

In `src/go/wt/cmd/create.go` (lines 179–185), when `--reuse` is passed and the worktree already exists by name, the collision check triggers an early return:

```go
if wt.CheckNameCollision(ctx.WorktreesDir, finalName) {
    if reuse {
        fmt.Fprintf(os.Stderr, "Reusing existing worktree: %s\n", finalName)
        rb.Disarm()
        fmt.Println(filepath.Join(ctx.WorktreesDir, finalName))
        return nil   // ← early return — skips init block at line 222
    }
}
```

The init script block at line 222 (`if worktreeInit == "true"`) is never reached. So for any `wt create --reuse` call that finds an existing worktree, `fab sync` is silently skipped.

### When This Causes "Unknown skill"

The operator uses `wt create --reuse` for autopilot respawns (§6 Autopilot, step 1: "create worktree (`--reuse` for respawns)"). If the existing worktree's `.claude/skills/` is stale or empty (e.g., the worktree was created before `fab sync` was last run, or skills were cleared), the respawned agent opens with missing skills and fails on the first slash command.

### Why "Sometimes"

- **Fresh spawn**: `wt create --non-interactive` (no `--reuse`) → `RunWorktreeSetup` runs → `fab sync` deploys skills → works.
- **Respawn (`--reuse`) into worktree with current skills**: early return, but skills already exist from the prior session → works.
- **Respawn (`--reuse`) into worktree with stale/empty skills**: early return, `fab sync` skipped, skills missing → "Unknown skill".

### Fix — Two Parts

**Part 1**: Change the default `WORKTREE_INIT_SCRIPT` in `InitScriptPath()` from `"fab-kit sync"` to `"fab sync"`. Currently `context.go` defaults to `"fab-kit sync"` but the canonical command since the routing consolidation is `fab sync` (which routes to `fab-kit` internally). Any environment that doesn't have `WORKTREE_INIT_SCRIPT` set gets the stale default.

**Part 2**: Add a `fab sync` call inside the `--reuse` early-return path in `wt create`, so skills are refreshed even when reusing an existing worktree.

Both are idempotent (per Constitution §III) — `fab sync` is safe to run even when skills are already current.

## What Changes

### `src/go/wt/internal/worktree/context.go` — Default Init Script

Change the fallback in `InitScriptPath()` from `"fab-kit sync"` to `"fab sync"`:

```go
// Before
func InitScriptPath() string {
    if v := os.Getenv("WORKTREE_INIT_SCRIPT"); v != "" {
        return v
    }
    return "fab-kit sync"
}

// After
func InitScriptPath() string {
    if v := os.Getenv("WORKTREE_INIT_SCRIPT"); v != "" {
        return v
    }
    return "fab sync"
}
```

Also update `context_test.go` — `TestInitScriptPath_Default` currently asserts `"fab-kit sync"`; update to assert `"fab sync"`.

### `src/go/wt/cmd/create.go` — `--reuse` Early Return Path

In the collision-check block, before returning, run the init script on the existing worktree path:

```go
if wt.CheckNameCollision(ctx.WorktreesDir, finalName) {
    if reuse {
        fmt.Fprintf(os.Stderr, "Reusing existing worktree: %s\n", finalName)
        rb.Disarm()
        existingWtPath := filepath.Join(ctx.WorktreesDir, finalName)
        // Run init script on reuse — ensures skills are current even in existing worktrees
        if worktreeInit == "true" {
            initScript := wt.InitScriptPath()
            _ = wt.RunWorktreeSetup(existingWtPath, "force", initScript, ctx.RepoRoot)
            // Non-fatal: reuse proceeds even if init fails (existing worktree may be functional)
        }
        fmt.Println(existingWtPath)
        return nil
    }
}
```

The init failure is non-fatal here — the worktree exists and may already have working skills. The failure is logged to stderr by `RunWorktreeSetup` before returning the error.

### Test Coverage

`src/go/wt/cmd/create_test.go` — add `TestCreate_ReuseRunsInitScript` verifying that `--reuse` on an existing worktree executes the init script. Parallel to the existing `TestCreate_InitScriptRuns` test.

### Memory Update

`docs/memory/fab-workflow/kit-architecture.md` — update the `wt create` description (in the "wt package" section) to document the `--reuse` init behavior.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) document that `--reuse` now also runs the init script on the existing worktree

## Impact

- `src/go/wt/internal/worktree/context.go` — default changed from `"fab-kit sync"` to `"fab sync"`
- `src/go/wt/internal/worktree/context_test.go` — update `TestInitScriptPath_Default` assertion
- `src/go/wt/cmd/create.go` — `--reuse` early-return block: add init call before `return nil`
- `src/go/wt/cmd/create_test.go` — one new test for `--reuse` init behavior
- `docs/memory/fab-workflow/kit-architecture.md` — memory update (hydrate)
- No changes to `fab-go`, `fab-kit`, operator skill, batch commands, status schemas, or templates
- No breaking changes — init script is idempotent; `fab sync` is the canonical routing command

## Open Questions

- Should init failure on `--reuse` be fatal (abort the reuse, surface error) or non-fatal (warn and proceed)? Non-fatal is proposed since the existing worktree may have functional skills; fatal would be more defensive but could break autopilot respawns if `fab` is temporarily unavailable.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change type is `fix` | Keyword "fix" in description; identified regression in `--reuse` path and stale default | S:90 R:90 A:95 D:90 |
| 2 | Certain | Fix #1: default `WORKTREE_INIT_SCRIPT` changed to `"fab sync"` in `context.go` | User confirmed; `fab sync` is the canonical routing command since the three-binary consolidation; `"fab-kit sync"` is the stale pre-consolidation default | S:95 R:90 A:95 D:95 |
| 3 | Certain | Fix #2: `--reuse` path in `create.go` must run init before early return | User confirmed; root cause is the early return at line 183 bypassing the init block at line 222 | S:95 R:90 A:95 D:95 |
| 4 | Confident | Init failure on `--reuse` is non-fatal | Existing worktree may already have working skills; failing hard would break autopilot respawns over a transient init issue | S:70 R:65 A:75 D:70 |
| 5 | Tentative | `--worktree-init false` flag suppresses init even on `--reuse` | Consistent with the non-reuse path behavior. <!-- assumed: flag check added to reuse path; same gate as line 222 --> | S:65 R:70 A:70 D:60 |

5 assumptions (3 certain, 1 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
