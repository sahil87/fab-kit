# Intake: Status Symlink Pointer

**Change**: 260307-x2tx-status-symlink-pointer
**Created**: 2026-03-07
**Status**: Draft

## Origin

> Replace `fab/current` pointer file with a `.fab-status.yaml` symlink at repo root that points to the active change's `.status.yaml`. Reduce the observation surface area for a cross-worktree orchestrator (conductor).

Conversational. The discussion started in `/fab-discuss` exploring whether a symlink could replace the `fab/current` text file. Initial analysis identified tradeoffs (portability, broken symlinks, atomic updates). The user clarified the real motivation: building the conductor orchestrator needs fast cross-worktree state observation, and state is currently spread across three locations (`fab/current`, `fab/changes/{name}/.status.yaml`, `.fab-runtime.yaml`). This reframed the change from "marginal ergonomic gain" to "material reduction in observation cost". The user chose `.fab-status.yaml` (not `.status.yaml`) for naming consistency with `.fab-runtime.yaml`, and repo root (not `fab/`) since `.fab-runtime.yaml` already lives there.

## Why

1. **Observation cost for the conductor**: The conductor (SPEC-fab-conductor) needs to scan N worktrees and build a pane map. Currently each worktree requires three reads: `fab/current` (parse line 2 for change name) → `fab/changes/{name}/.status.yaml` (stage, progress, confidence) → `.fab-runtime.yaml` (agent idle state). The first read is purely indirection — it exists only to locate the second. With a symlink, `readlink .fab-status.yaml` gives the change identity (from the target path) and `cat .fab-status.yaml` gives full status, collapsing two reads into one.

2. **State surface consolidation**: After this change, all ephemeral per-worktree state lives as two sibling files at repo root: `.fab-status.yaml` (symlink → change identity + pipeline state) and `.fab-runtime.yaml` (agent idle state). Both gitignored, both at the same level, scannable with a single glob.

3. **If we don't do this**: The conductor must implement a multi-step resolve chain for every worktree observation — read a text file, parse it, construct a path, read the real file. This adds latency and complexity to every refresh cycle, and the indirection serves no purpose when a symlink can encode both pointer and data access in one filesystem entry.

## What Changes

### Symlink at `<repo_root>/.fab-status.yaml`

Replace `fab/current` (a 2-line text file with change ID on line 1 and folder name on line 2) with a symlink at the repository root:

```
.fab-status.yaml → fab/changes/260307-x2tx-status-symlink-pointer/.status.yaml
```

- **Target**: always a relative path from repo root to the active change's `.status.yaml`
- **Absent symlink**: no active change (same semantics as missing `fab/current`)
- **Broken symlink**: treated as absent — if `os.Stat` fails but `os.Lstat` succeeds, the change folder was deleted/archived; treat as no active change
- **Naming**: `.fab-status.yaml` (matches `.fab-runtime.yaml` convention)
- **Location**: repo root (alongside `.fab-runtime.yaml` — group ephemeral state)

### Add `id` field to `.status.yaml`

Add the 4-char change ID as a top-level field in `.status.yaml`, both in the template and in `fab change new`:

```yaml
id: x2tx
name: 260307-x2tx-status-symlink-pointer
created: 2026-03-07T16:54:29+05:30
...
```

This makes the change ID directly available from reading `.status.yaml` without needing to parse the folder name. The `id` field is derived from the `name` at creation time and is immutable.

### Update `resolve.go` — `resolveFromCurrent()`

Replace the `fab/current` file read with symlink resolution:

```go
// Before: read fab/current, parse line 2
// After:  readlink <repoRoot>/.fab-status.yaml, extract folder name from target path

func resolveFromCurrent(fabRoot, changesDir string) (string, error) {
    symlinkPath := filepath.Join(filepath.Dir(fabRoot), ".fab-status.yaml")
    target, err := os.Readlink(symlinkPath)
    if err != nil {
        // No symlink = no active change, fall through to single-change guess
        ...
    }
    // target is "fab/changes/{name}/.status.yaml"
    // extract {name} from the path
    ...
}
```

The single-change fallback (when exactly one change folder exists) remains unchanged.

### Update `change.go` — Switch and Rename

**Switch**: Instead of writing a 2-line text file to `fab/current`, create a symlink:

```go
// Remove existing symlink (or stale file)
os.Remove(symlinkPath)
// Create new symlink: .fab-status.yaml → fab/changes/{name}/.status.yaml
os.Symlink(target, symlinkPath)
```

**Switch --blank**: Remove the symlink instead of deleting `fab/current`.

**Rename**: When the active change is renamed, update the symlink target. Since the rename changes the folder name, the symlink must point to the new path. Remove and recreate the symlink.

### Update `panemap.go` — `readFabCurrent()`

Replace the `fab/current` file read with symlink resolution. The function currently reads `fab/current` and parses both lines. Change to `readlink .fab-status.yaml` and derive the display name and folder name from the target path.

### Update `.gitignore`

```diff
-fab/current
+.fab-status.yaml
```

### Update `fab/.kit/templates/status.yaml`

Add `id` field:

```diff
+id: {ID}
 name: {NAME}
 created: {CREATED}
```

### Update `fab change new` (Go)

Write the `id` field when initializing `.status.yaml`. The ID is already extracted during folder name generation — persist it to the status file.

### Update Skills and Preamble

All references to `fab/current` in skills and preamble need updating:

- **`_preamble.md`** — Change Context section references `fab/current` in the override explanation and validation list. Update to reference `.fab-status.yaml` symlink.
- **`_scripts.md`** — `fab change` documentation references `fab/current` pointer. Update.
- **`fab-discuss`** — references `fab/current` in context loading. Update to use `fab resolve`.
- **`fab-archive`** — references clearing `fab/current`. Update to symlink removal via `fab change switch --blank`.
- **Glossary** (`docs/specs/glossary.md`) — defines "Pointer file" as `fab/current`. Update to `.fab-status.yaml` symlink.
- **Other specs/memory** — grep for `fab/current` and update references.

### Migration

Ship a migration file in `fab/.kit/migrations/` that:

1. If `fab/current` exists and is non-empty:
   - Parse line 2 for the folder name
   - Create `.fab-status.yaml` symlink pointing to `fab/changes/{name}/.status.yaml`
   - Remove `fab/current`
2. If `fab/current` is empty or absent: no-op
3. Add `id` field to all existing `.status.yaml` files (extract from folder name) if not present
4. Update `.gitignore`: remove `fab/current`, add `.fab-status.yaml`

### Go Tests

Update all tests that write `fab/current`:

**`src/fab-go/internal/resolve/`** — tests that create `fab/current` as a text file need to create a symlink instead.

**`src/fab-go/internal/change/`** — switch tests that assert `fab/current` content need to assert symlink target instead.

**`src/fab-go/cmd/fab/panemap_test.go`** — `WriteFile(tmp+"/fab/current", ...)` needs to become `os.Symlink(...)`.

**`src/fab-go/test/parity/`** — any parity tests referencing `fab/current`.

New test cases:
- Symlink resolution returns correct folder name
- Broken symlink treated as no active change
- Switch creates symlink with correct relative target
- Switch --blank removes symlink
- Rename updates symlink target

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update "Active Change Tracking" — symlink replaces pointer file, new observation model
- `fab-workflow/kit-architecture`: (modify) Update references to `fab/current`
- `fab-workflow/schemas`: (modify) Add `id` field to `.status.yaml` schema documentation
- `fab-workflow/migrations`: (modify) Document the migration for this change

## Impact

- **Go source**: `resolve.go`, `change.go`, `panemap.go`, `panemap_test.go`, `runtime.go` (references to `fab/current` in comments)
- **Skills**: `_preamble.md`, `_scripts.md`, `fab-discuss`, `fab-archive`, and any other skills referencing `fab/current`
- **Specs/docs**: glossary, architecture, overview, various memory files
- **Config**: `.gitignore`, `fab/.kit/templates/status.yaml`
- **Migration**: new migration file in `fab/.kit/migrations/`
- **Downstream**: SPEC-fab-conductor Discovery procedure simplifies (read two sibling files at repo root)

## Open Questions

None — all design decisions resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.fab-status.yaml` symlink at repo root replaces `fab/current` | Discussed — user chose this over keeping `fab/current`, motivated by conductor observation cost | S:95 R:75 A:90 D:95 |
| 2 | Certain | Symlink target is relative path: `fab/changes/{name}/.status.yaml` | Discussed — relative paths work across worktrees, absolute would break | S:90 R:85 A:90 D:90 |
| 3 | Certain | Naming: `.fab-status.yaml` (not `.status.yaml`) | Discussed — user chose for consistency with `.fab-runtime.yaml` naming convention | S:95 R:90 A:95 D:95 |
| 4 | Certain | Location: repo root (not `fab/`) | Discussed — groups ephemeral state with `.fab-runtime.yaml`, already at repo root | S:90 R:85 A:85 D:90 |
| 5 | Certain | Broken symlink = no active change | Discussed — user confirmed same semantics as missing `fab/current` | S:90 R:90 A:90 D:95 |
| 6 | Certain | Add `id` field to `.status.yaml` | Discussed — user proposed this to make change ID directly available without path parsing | S:90 R:90 A:85 D:90 |
| 7 | Certain | Windows compatibility not a concern | Discussed — user explicitly stated this | S:95 R:95 A:95 D:95 |
| 8 | Certain | All symlink lifecycle through `fab` subcommands (switch, archive, new) | Discussed — user confirmed, no direct symlink manipulation from skills | S:90 R:80 A:90 D:90 |
| 9 | Confident | Migration ships in `fab/.kit/migrations/` per project convention | Inferred from `context.md` migration policy and existing pattern | S:80 R:80 A:85 D:85 |
| 10 | Confident | Symlink created with `os.Remove` + `os.Symlink` (not atomic) | Discussed — user accepted this tradeoff since all changes go through `fab` subcommands | S:80 R:85 A:80 D:85 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
