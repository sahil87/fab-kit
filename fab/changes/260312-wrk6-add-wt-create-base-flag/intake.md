# Intake: Add --base flag to wt create

**Change**: 260312-wrk6-add-wt-create-base-flag
**Created**: 2026-03-12
**Status**: Draft

## Origin

> Arose from a `/fab-discuss` session analyzing how `/fab-operator2` handles dependent changes. The user asked: "Does the operator ensure that, when the worktree for change B is created, it's actually based on the branch that was created by change A?" The answer was no — operator2 uses a merge-then-rebase sequential model. The user then asked whether `wt create` should support a `--base` flag, and decided yes — it should accept both git branch names and fab change IDs.

Key decisions from the discussion:
- `--base` should accept fab change IDs (4-char ID, substring, full name) in addition to raw git branch names — this keeps the interface consistent with how the rest of fab resolves changes
- When `--base` is passed with an existing local/remote branch (not a new branch), warn and ignore
- The operator2 autopilot per-change loop should use `--base` when the queue has known ordering dependencies

## Why

1. **Problem**: When change B depends on change A (e.g., B builds on code introduced by A), the operator2 autopilot currently creates B's worktree based on HEAD/main — which doesn't include A's code. B only gets A's code after A merges into main and B rebases. This forces strictly sequential execution: A must fully complete and merge before B can even start.

2. **Consequence**: Dependent changes cannot be worked on in parallel. The autopilot queue processes one change at a time, waiting for merge before starting the next. This bottleneck is especially painful when A is in review and B's planning/spec work could proceed independently.

3. **Approach**: Add a `--base` flag to `wt create` that specifies the start-point for the new branch. This maps directly to git's `git worktree add -b <branch> <path> <start-point>` — the underlying primitive already supports it, `wt create` just doesn't expose it. The change ID resolution layer on top makes it ergonomic for the operator, which thinks in change IDs not branch names.

## What Changes

### `wt create` — new `--base` flag

Add a `--base <ref>` flag that specifies the start-point when creating a new branch:

```
wt create feature-B --base feature-A --worktree-name bravo
# → git worktree add -b feature-B <path> feature-A

wt create feature-B --base r3m7 --worktree-name bravo
# → resolves r3m7 to its branch name, then uses that as start-point
```

**Resolution logic for `--base` value:**
1. Try fab change resolution first — resolve the value as a change ID/name via `fab resolve`, then look up the change's branch from its git integration (the branch naming convention: `YYMMDD-XXXX-slug`)
<!-- assumed: Branch name derivation — use the change folder name directly as the branch name, since fab's naming convention makes folder name == branch name -->
2. If fab resolution fails, treat the value as a raw git ref (branch name, tag, commit SHA)
3. If the raw git ref also doesn't resolve, error out

**Interaction with existing branch modes:**

| Scenario | `--base` | Behavior |
|----------|----------|----------|
| New branch (branch doesn't exist locally or remotely) | provided | Create branch from `--base` ref instead of HEAD |
| New branch | omitted | Current behavior: branch from HEAD |
| Existing local branch | provided | Warn: "--base ignored: branch already exists locally" |
| Existing remote branch | provided | Warn: "--base ignored: fetching existing remote branch" |
| Exploratory (no branch arg) | provided | Create exploratory branch from `--base` ref instead of current HEAD |
| Exploratory | omitted | Current behavior: branch from current HEAD |

### `CreateWorktree` function — start-point parameter

The low-level `CreateWorktree` function in `crud.go` needs a start-point parameter:

```go
// Current:
func CreateWorktree(path, branch string, newBranch bool) error

// New:
func CreateWorktree(path, branch string, newBranch bool, startPoint string) error
```

When `startPoint` is non-empty and `newBranch` is true: `git worktree add -b <branch> <path> <startPoint>`
When `startPoint` is non-empty and `newBranch` is false: ignored (existing branch checkout)
When `startPoint` is empty: current behavior unchanged

### Operator2 autopilot integration

Update the operator2 skill's autopilot per-change loop (step 1: Spawn) to use `--base` when the queue has sequential dependencies. The previous change in the queue provides the base:

```
# Current:
wt create --non-interactive --reuse --worktree-name <change-B> <branch-B>

# With dependencies:
wt create --non-interactive --reuse --worktree-name <change-B> <branch-B> --base <change-A-id>
```

The operator already knows the ordering (user-provided, confidence-based, or hybrid). For user-provided ordering, the operator assumes each change depends on the previous one in the queue. For confidence-based ordering, no `--base` is used (independent changes sorted by score).

### wt package spec update

Update `docs/specs/packages.md` to document the new `--base` flag in the wt section.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator2 autopilot behavior to document `--base` usage
- `fab-workflow/kit-architecture`: (modify) Update wt command reference if it lists wt create flags

## Impact

- **`src/go/wt/cmd/create.go`** — new `--base` flag, resolution logic, pass-through to crud functions
- **`src/go/wt/internal/worktree/crud.go`** — `CreateWorktree`, `CreateBranchWorktree`, `CreateExploratoryWorktree` gain start-point parameter
- **`src/go/wt/cmd/create_test.go`** — new test cases for `--base` with new branch, existing branch (warn), exploratory, and change ID resolution
- **`fab/.kit/skills/fab-operator2.md`** — autopilot per-change loop updated to use `--base`
- **`docs/specs/packages.md`** — wt create documentation updated

## Open Questions

- Should `--base` with confidence-based ordering ever be used? (Currently proposed: no — confidence-based assumes independent changes)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go implementation in existing wt package | Discussed — wt is a Go binary, this extends it | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use git worktree add start-point argument | Discussed — this is the direct git primitive | S:90 R:95 A:95 D:95 |
| 3 | Confident | Branch name == change folder name for resolution | Discussed — fab naming convention makes this reliable | S:80 R:80 A:75 D:80 |
| 4 | Confident | Warn-and-ignore for --base with existing branches | Discussed — existing branches already have a history, --base is meaningless | S:85 R:90 A:80 D:75 |
| 5 | Confident | Operator uses --base only for user-provided ordering | Discussed — confidence-based ordering implies independence | S:75 R:80 A:70 D:70 |
| 6 | Confident | fab resolve as first resolution step for --base value | Discussed — keeps interface consistent with rest of fab | S:80 R:85 A:80 D:75 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
