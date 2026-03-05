# Intake: Batch Pipeline — Single Change Support & Default Base Branch

**Change**: 260223-xiuk-batch-pipeline-single-change-and-base-branch
**Created**: 2026-02-23
**Status**: Draft

## Origin

> batch-pipeline-series should accept a single change (drop the >= 2 guard), and run.sh should default to current branch when the manifest's base field is missing/empty (instead of erroring)

Conversational — discussed during a `/fab-discuss` session. Both issues were identified through code review of the batch pipeline scripts. The branch naming consistency question was raised and resolved (already consistent across `batch-fab-switch-change.sh` and `dispatch.sh`).

## Why

1. **Single-change guard is artificial**: `batch-pipeline-series.sh` requires `>= 2` changes (line 77), but the entire downstream pipeline (manifest generation, `run.sh`, `dispatch.sh`) handles a single-entry manifest without issue. Users who want the orchestrator machinery (worktree isolation, tmux pane, status polling, `/git-pr` shipping) for a single change must currently hand-write a manifest or use `batch-pipeline` with a manifest file. The series shorthand should cover this common case.

2. **Hardcoded `main` fallback in `run.sh`**: `validate_manifest()` errors out if the `base` field is missing or empty (lines 87-91). `batch-pipeline-series.sh` already defaults to the current branch (line 84-86), but hand-written manifests that omit `base` get a hard error instead of a sensible default. Pipelines should operate relative to where you are, not relative to a hardcoded remote default.

## What Changes

### 1. Relax minimum change count in `batch-pipeline-series.sh`

**File**: `fab/.kit/scripts/batch-pipeline-series.sh`

Drop the minimum from 2 to 1:

```bash
# Before (line 77):
if [[ ${#changes[@]} -lt 2 ]]; then
  echo "Error: at least 2 changes required (got ${#changes[@]})" >&2

# After:
if [[ ${#changes[@]} -lt 1 ]]; then
  echo "Error: at least 1 change required" >&2
```

Update the usage text accordingly:
- Arguments line: `<change1> <change2> [...]` → `<change> [<change>...]`
- Description: `at least 2 required` → remove or update
- Examples: add a single-change example

### 2. Default `base` to current branch in `run.sh`

**File**: `fab/.kit/scripts/pipeline/run.sh`

In `validate_manifest()` (lines 87-91), instead of erroring on a missing/empty `base`, default to `git branch --show-current` with `main` as the last-resort fallback:

```bash
# Before:
base=$(yq -r '.base // ""' "$manifest")
if [[ -z "$base" || "$base" == "null" ]]; then
  echo "Error: manifest missing 'base' field" >&2
  return 1
fi

# After:
base=$(yq -r '.base // ""' "$manifest")
if [[ -z "$base" || "$base" == "null" ]]; then
  base=$(git branch --show-current 2>/dev/null) || base="main"
  # Write the resolved base back so get_parent_branch() reads it consistently
  yq -i ".base = \"$base\"" "$manifest"
fi
```

The write-back ensures `get_parent_branch()` reads the resolved value from the manifest without needing its own fallback logic.

### 3. Update example manifest and documentation

**File**: `fab/pipelines/example.yaml`

Document `base` as optional with the default behavior:

```yaml
# base — Branch that root nodes (depends_on: []) branch from.
# Optional. Defaults to the current branch if omitted.
# base: "main"
```

### 4. Update memory file

**File**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

- Update `batch-pipeline-series.sh` section: remove "Requires at least 2 change arguments", note single-change support
- Update Manifest Format section: note `base` is optional with current-branch default
- Add changelog entry

### 5. Update tests

**File**: `src/scripts/pipeline/test.bats`

- Add test for `validate_manifest` with missing `base` field (should succeed and write resolved base)
- Verify the single-change manifest path through `batch-pipeline-series.sh` argument validation (if testable without infrastructure)

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update series script docs, manifest format, add changelog entry

## Impact

- `fab/.kit/scripts/batch-pipeline-series.sh` — argument validation, usage text
- `fab/.kit/scripts/pipeline/run.sh` — manifest validation
- `fab/pipelines/example.yaml` — documentation
- `docs/memory/fab-workflow/pipeline-orchestrator.md` — memory
- `src/scripts/pipeline/test.bats` — test coverage

No behavioral change for existing manifests that already specify `base`. No change to `dispatch.sh`, `batch-pipeline.sh`, or `batch-fab-switch-change.sh`.

## Open Questions

None — both changes are well-scoped with clear implementation paths.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop minimum from 2 to 1 in series script | Discussed — user explicitly requested single-change support | S:95 R:95 A:95 D:95 |
| 2 | Certain | Default base to current branch, fallback to main | Discussed — user explicitly requested current-branch default; matches existing series script behavior | S:90 R:90 A:90 D:90 |
| 3 | Confident | Write resolved base back to manifest for consistency | Avoids duplicating fallback logic in get_parent_branch(); manifest becomes self-documenting after resolution | S:70 R:85 A:80 D:75 |
| 4 | Certain | No changes to dispatch.sh or batch-fab-switch-change.sh | Discussed — branch naming already consistent across scripts, confirmed during review | S:95 R:95 A:95 D:95 |
| 5 | Confident | Detached HEAD fallback to "main" | Standard git convention; detached HEAD returns empty from git branch --show-current, main is the safe last-resort | S:60 R:85 A:80 D:80 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
