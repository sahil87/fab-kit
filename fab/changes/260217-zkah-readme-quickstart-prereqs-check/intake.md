# Intake: README Quick Start Restructure + fab-sync Prerequisites Check

**Change**: 260217-zkah-readme-quickstart-prereqs-check
**Created**: 2026-02-17
**Status**: Draft

## Origin

> Restructure README Quick Start and add prerequisites check to fab-sync.
>
> Discussion concluded with three agreed changes: (1) fold Initialize and Updating under Install as sub-sections, (2) update TOC to match, (3) add prerequisites checking as a sync step.

## Why

The README Quick Start currently presents Install and Initialize as separate top-level steps, but they're part of the same "getting fab into your project" flow. Separating them creates a false mental model — users think initialization is a distinct phase rather than part of setup. Similarly, the standalone Updating section is disconnected from the Install section it logically belongs with.

Additionally, the sync pipeline (`fab/.kit/sync/`) has no prerequisites validation. If `yq`, `jq`, or other required tools are missing, downstream sync steps either silently fail or produce confusing errors. Users should get a clear, actionable error message up front.

## What Changes

### 1. Restructure Quick Start section in README.md

Collapse the current 4-step Quick Start (Install → Initialize → Your first change → Going parallel) into a 3-step flow:

```
### 1. Install
#### New project          (existing gh release download)
#### From a local clone   (existing cp -r)
#### Initialize           (existing fab-sync + direnv + /fab-setup — moved from step 2)
#### Updating from a previous version  (existing fab-upgrade.sh — moved from standalone section)

### 2. Your first change  (renumbered from 3)
### 3. Going parallel     (renumbered from 4)
```

Remove the standalone `## Updating` section entirely — its content moves under Install.

### 2. Update Contents TOC

Update the TOC line at the top to reflect the new structure. The `#updating` anchor should point to the sub-section within Install (GitHub auto-generates anchors from headings, so `#updating-from-a-previous-version` will work).

### 3. Add prerequisites check as a sync step

Create `fab/.kit/sync/1-prerequisites.sh` that validates required tools are available. Rename existing sync steps to maintain sort order: `1-direnv.sh` → `2-direnv.sh`, `2-sync-workspace.sh` → `3-sync-workspace.sh`.

The prerequisites step runs first (before direnv and workspace sync) and checks that all required tools are present. Any missing tool is **fatal** (exit 1):

- `yq` — used by stageman/changeman for `.status.yaml` operations
- `jq` — used by sync for `.claude/settings.local.json` merge
- `gh` — used for install, releases, and PR workflows
- `direnv` — used for PATH setup (sync step 2 runs `direnv allow`)
- `bats` — used for the test suite

Output should point users to the Prerequisites section in the README.

## Affected Memory

- `fab-workflow/distribution`: (modify) Update to reflect prerequisites check in fab-sync and README restructuring

## Impact

- **README.md** — structural reorganization of Quick Start, TOC update, Updating section removal
- **`fab/.kit/sync/1-prerequisites.sh`** — new sync step for prerequisites validation
- **`fab/.kit/sync/1-direnv.sh`** → renamed to **`fab/.kit/sync/2-direnv.sh`**
- **`fab/.kit/sync/2-sync-workspace.sh`** → renamed to **`fab/.kit/sync/3-sync-workspace.sh`**

## Open Questions

None — the approach was discussed and agreed in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All prerequisites are fatal (yq, jq, gh, direnv, bats) | Keep it simple — if the tool is listed as a prerequisite, it's required. Partial environments cause confusing downstream failures | S:90 R:90 A:95 D:95 |
| 2 | Certain | GitHub auto-generates heading anchors | Standard GitHub markdown behavior — `#### Updating from a previous version` becomes `#updating-from-a-previous-version` | S:95 R:95 A:95 D:95 |
| 3 | Confident | Prerequisites check runs as first sync step (`1-prerequisites.sh`) | Sync steps run in sorted order; prerequisites must pass before direnv or workspace sync. Existing steps renumber to `2-direnv.sh` and `3-sync-workspace.sh` | S:80 R:90 A:85 D:80 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
