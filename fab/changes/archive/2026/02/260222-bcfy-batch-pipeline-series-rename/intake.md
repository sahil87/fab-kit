# Intake: Batch Pipeline Series & Rename

**Change**: 260222-bcfy-batch-pipeline-series-rename
**Created**: 2026-02-22
**Status**: Draft

## Origin

> Rename batch-fab-pipeline to batch-pipeline, add finite-exit as default (watch: true manifest field for infinite loop), fix dispatch.sh to use local branch refs instead of origin/, create batch-pipeline-series.sh that takes inline change list and generates a temp manifest with sequential dependencies using current branch as base, add .gitignore pattern for generated series manifests, update example.yaml docs and memory.

Conversational — emerged from a `/fab-discuss` session where the user proposed two changes: (1) rename `batch-fab-pipeline` to `batch-pipeline`, and (2) add `batch-pipeline-series` for inline sequential chains. Discussion surfaced the finite-exit default and local-branch fix as additional scope items. All design decisions were explicitly confirmed through back-and-forth.

## Why

The pipeline orchestrator has three friction points:

1. **Naming inconsistency** — `batch-fab-pipeline` follows `batch-fab-{verb}-{noun}` convention, but unlike the other batch scripts (which wrap a single fab skill), the pipeline orchestrator coordinates multi-change flows. The `fab-` infix is misleading.

2. **Ceremony for simple chains** — Running 3 changes in sequence (A → B → C) currently requires writing a YAML manifest file. For the common case of "run these changes in order," the manifest is unnecessary overhead. A CLI-first interface that takes change IDs as positional arguments would eliminate this.

3. **Fragile remote branch dependency** — `dispatch.sh` branches dependent nodes from `origin/<parent-branch>`, but this assumes the parent branch has been pushed to the remote. While the current flow happens to push before marking `done`, this is an implementation coincidence, not an explicit contract. Using local branch refs is both more correct and more robust — git branches are shared across all worktrees of the same repo.

4. **Infinite loop as default** — `run.sh` loops forever, which makes sense for live-editing workflows but is wrong for pre-defined manifests. Most runs have a finite set of changes; the daemon-like behavior should be opt-in.

## What Changes

### 1. Rename `batch-fab-pipeline.sh` → `batch-pipeline.sh`

Rename `fab/.kit/scripts/batch-fab-pipeline.sh` to `fab/.kit/scripts/batch-pipeline.sh`. Update all references:
- `fab/.kit/scripts/fab-help.sh` — help text
- `docs/memory/fab-workflow/pipeline-orchestrator.md` — memory file
- `docs/memory/fab-workflow/kit-architecture.md` — architecture memory

Internal script comments and usage text updated to match.

### 2. Finite exit as default in `run.sh`

Add `watch` field support to the manifest format:

```yaml
base: main
watch: true   # opt-in to infinite loop (live-editing mode)
changes:
  - id: ...
```

Behavior:
- `watch: true` → current infinite-loop behavior (poll for new entries, never exit)
- `watch: false` or absent → **finite mode** (new default): after each dispatch cycle, if no changes are dispatchable AND all changes are terminal (`done`, `failed`, `invalid`), print summary and exit 0

Implementation in `run.sh`:
- Read `watch` field: `yq -r '.watch // false' "$MANIFEST"`
- In the `else` branch of the main loop (nothing dispatchable), check if all changes are terminal. If yes and `watch` is not `true`, call `print_summary` and `exit 0`

### 3. Local branch refs in `dispatch.sh`

Replace `origin/` remote branch lookup with local branch ref in `create_worktree()`:

```bash
# Current (fragile — depends on remote push)
if git ls-remote --exit-code --heads origin "$PARENT_BRANCH" &>/dev/null; then
  git branch "$CHANGE_BRANCH" "origin/$PARENT_BRANCH" >/dev/null 2>&1
fi

# New (local-first — branches shared across worktrees)
if git show-ref --verify --quiet "refs/heads/$PARENT_BRANCH" 2>/dev/null; then
  git branch "$CHANGE_BRANCH" "$PARENT_BRANCH" >/dev/null 2>&1
fi
```

This works because `wt-create` for the parent worktree created the branch locally, and git branches are shared across all worktrees of the same repo. The parent branch being checked out in another worktree is not a problem — git only blocks checking out a branch that's already checked out, not branching from it.

### 4. New `batch-pipeline-series.sh`

New script at `fab/.kit/scripts/batch-pipeline-series.sh`.

**Interface**:
```
batch-pipeline-series change1 change2 change3 [--base <branch>]
```

- Positional arguments: change names/IDs (same resolution as `fab-switch` — partial slug, 4-char ID, full name)
- `--base <branch>`: branch that the first change branches from. Defaults to current branch (`git branch --show-current`)
- Requires at least 2 change arguments

**Behavior**:
1. Parse arguments — separate change IDs from `--base` flag
2. Resolve `--base` default to current branch
3. Generate a temporary manifest at `fab/pipelines/.series-{timestamp}.yaml`:
   ```yaml
   base: <current-branch>
   changes:
     - id: change1
       depends_on: []
     - id: change2
       depends_on: [change1]
     - id: change3
       depends_on: [change2]
   ```
   First change depends on `[]` (root node), each subsequent depends on its predecessor.
4. No `watch: true` — finite mode by default
5. Delegate to `run.sh` via `exec bash "$SCRIPT_DIR/pipeline/run.sh" "$manifest_path"`

The temp manifest is **not** cleaned up — left in place for debugging/inspection.

### 5. `.gitignore` pattern

Add to the repo `.gitignore`:
```
fab/pipelines/.series-*.yaml
```

### 6. Update `example.yaml`

Add documentation for the `watch` field:
```yaml
# watch — Controls loop behavior.
#   true    — Infinite loop: poll for new entries (live-editing mode)
#   false   — Default: exit when all changes are terminal (done/failed/invalid)
# watch: true
```

### 7. Update memory

Update `docs/memory/fab-workflow/pipeline-orchestrator.md`:
- Rename references from `batch-fab-pipeline` to `batch-pipeline`
- Document `watch` field and finite-exit behavior
- Document `batch-pipeline-series.sh` script and its manifest generation
- Document local branch ref change in dispatch.sh

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Rename references, add watch/finite-exit, add series script, document local branch fix

## Impact

- **Scripts**: `batch-fab-pipeline.sh` (rename), `run.sh` (finite exit logic), `dispatch.sh` (local branch refs), new `batch-pipeline-series.sh`
- **Config/docs**: `example.yaml` (watch field docs), `.gitignore` (series pattern), `fab-help.sh` (rename ref)
- **Memory**: `pipeline-orchestrator.md`, `kit-architecture.md`
- **Tests**: Existing BATS tests in `src/scripts/pipeline/test.bats` — `create_worktree` tests may need updating for local branch ref change. New tests for finite-exit logic and series manifest generation.

## Open Questions

None — all design decisions were resolved in the discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `batch-pipeline.sh` | Discussed — user explicitly requested this rename | S:95 R:90 A:95 D:95 |
| 2 | Certain | Finite exit as default, `watch: true` for infinite | Discussed — user specified "manifest can have a watch: true that turns this mode on" | S:95 R:85 A:90 D:95 |
| 3 | Certain | Local branch refs instead of `origin/` | Discussed — user specified "we should not work with origin/* branches, just the local branches" | S:95 R:80 A:90 D:95 |
| 4 | Certain | Series generates temp manifest, delegates to `run.sh` | Discussed — thin wrapper approach agreed upon (option A) | S:90 R:90 A:90 D:90 |
| 5 | Certain | Don't remove temp manifest, gitignore the pattern | Discussed — user specified "No, don't remove the temp manifest. But git ignore the pattern" | S:95 R:95 A:95 D:95 |
| 6 | Certain | `--base` defaults to current branch | Discussed — user specified "It should assume the base branch is the 'current' branch" | S:95 R:85 A:90 D:95 |
| 7 | Confident | Series requires at least 2 change arguments | Inferred — a single change doesn't need a pipeline; use `fab-ff` directly | S:70 R:90 A:85 D:80 |
| 8 | Confident | Manifest timestamp format uses epoch seconds | Convention from similar patterns; simple and collision-free | S:60 R:95 A:80 D:75 |
| 9 | Confident | `--base` is the only flag; no `--watch` override for series | Series is inherently finite — watch mode doesn't apply to inline lists | S:70 R:90 A:85 D:80 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
