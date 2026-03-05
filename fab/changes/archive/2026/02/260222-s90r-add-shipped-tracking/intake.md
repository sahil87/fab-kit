# Intake: Add Shipped Tracking

**Change**: 260222-s90r-add-shipped-tracking
**Created**: 2026-02-22
**Status**: Draft

## Origin

> Add "shipped" tracking to the fab pipeline — extend stageman with ship/is-shipped subcommands, update git-pr to record PR URLs after creation.

Discussion mode: conversational (`/fab-discuss` → exploration → `/fab-new`). Multiple design options were evaluated collaboratively before arriving at the chosen approach.

Key decisions from conversation:
- **Option 2 chosen** over full stage (Option 1) and file-marker (Option 3) — a side-band `shipped` field in `.status.yaml` rather than a 7th pipeline stage or a loose file
- **Simple string array** — no metadata objects (no `url`/`at` keys), just bare URL strings
- **List by default** — supports multiple PRs per change (stacked PRs, split repos)
- **Built into `/git-pr`** — the skill already knows the PR URL from `gh pr create` output
- **Extend `stageman.sh`** — single-writer principle for `.status.yaml`; avoid two scripts mutating the same file
- **Archive guard unchanged** — remains `hydrate: done` only, no dependency on `shipped`

## Why

The pipeline currently defines "done" as `hydrate: done` — a housekeeping step. The actual deliverable of a change is the PR. There's no way to query whether a change has been shipped, no PR URL recorded in the change's status, and the state table routes directly from hydrate to archive with no acknowledgment that shipping happened. This creates a gap between "change complete" (memory hydrated) and "change delivered" (PR created).

If we don't fix it: the pipeline has no formal concept of shipping. `/git-pr` remains a fire-and-forget side channel with no status integration. There's no queryable record of which changes produced which PRs.

## What Changes

### stageman.sh — Two new subcommands

#### `ship <file> <url>`

Appends a PR URL to the `shipped` array in `.status.yaml`. Behavior:

```bash
stageman.sh ship .status.yaml "https://github.com/org/repo/pull/42"
```

- Reads the current `shipped` array (creates it if missing)
- **Deduplication**: if the URL already exists in the array, skip silently (idempotent)
- Appends the URL as a plain string
- Updates `last_updated` timestamp
- Atomic write pattern: temp file → `mv` (same as existing write commands)

Resulting YAML:

```yaml
shipped:
  - "https://github.com/org/repo/pull/42"
  - "https://github.com/org/repo/pull/43"
```

#### `is-shipped <file>`

Query command that checks whether the change has been shipped:

```bash
stageman.sh is-shipped .status.yaml
# exit 0 if shipped array has >= 1 entry
# exit 1 if shipped array is empty or missing
```

No stdout output — purely exit-code based (consistent with query patterns like `grep -q`).

### status.yaml template — New field

Add `shipped: []` to `fab/.kit/templates/status.yaml`, positioned after `stage_metrics` and before `last_updated`:

```yaml
stage_metrics: {}
shipped: []
last_updated: {CREATED}
```

### workflow.yaml schema — Document shipped field

Add a `shipped` section documenting the field's purpose, type, and semantics. This is descriptive documentation within the schema, not a new stage.

### /git-pr skill — Call stageman after PR creation

After successful PR creation (step 3c) or when a PR already exists (the "already shipped" path), call:

```bash
fab/.kit/scripts/lib/stageman.sh ship <status_file> <pr_url>
```

This requires `/git-pr` to resolve the active change's `.status.yaml` path. The skill should:
1. Attempt to resolve the active change via `changeman.sh resolve`
2. If a change is active, call `stageman.sh ship` with the PR URL
3. If no active change (or resolution fails), skip silently — `/git-pr` should work even outside the fab pipeline

### _preamble.md state table — Route hydrate to git-pr

Update the hydrate row to include `/git-pr` as an available command alongside `/fab-archive`:

| State | Available commands | Default |
|-------|-------------------|---------|
| hydrate | `/git-pr, /fab-archive` | `/git-pr` |

This signals that after hydrate, shipping is the expected next action before archiving.

### changeman.sh — Update default_command for hydrate

The `default_command` function in `changeman.sh` maps `hydrate` → `/fab-archive`. Update to map `hydrate` → `/git-pr`.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the `shipped` field in `.status.yaml` structure
- `fab-workflow/execution-skills`: (modify) Document `/git-pr` integration with stageman

## Impact

- **stageman.sh** — two new subcommands (`ship`, `is-shipped`), new CLI dispatch entries, new help text
- **status.yaml template** — one new field
- **workflow.yaml schema** — documentation addition
- **/git-pr skill** — new step after PR creation to record URL
- **_preamble.md** — state table row update
- **changeman.sh** — one-line change in `default_command`
- **No impact on**: archive guard, existing 6-stage progression, preflight, existing changes in archive

## Open Questions

None — all design decisions were resolved in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Simple string array for shipped field | Discussed — user explicitly chose bare URLs over structured objects with url/at keys | S:95 R:90 A:90 D:95 |
| 2 | Certain | Extend stageman.sh rather than separate script | Discussed — user confirmed single-writer principle for .status.yaml | S:90 R:85 A:90 D:90 |
| 3 | Certain | Built into /git-pr skill | Discussed — user confirmed this is the natural integration point | S:90 R:90 A:85 D:95 |
| 4 | Certain | Archive guard unchanged (hydrate:done only) | Discussed — user explicitly stated no dependency on shipped for archive | S:95 R:95 A:90 D:95 |
| 5 | Certain | List by default (not single value) | Discussed — user specified list to support multiple PRs per change | S:95 R:90 A:85 D:95 |
| 6 | Confident | is-shipped uses exit codes only (no stdout) | Convention inferred from grep -q pattern; consistent with CLI query tools | S:60 R:90 A:80 D:75 |
| 7 | Confident | git-pr silently skips ship call when no active change | Graceful degradation — git-pr should work outside fab pipeline context | S:65 R:85 A:80 D:70 |
| 8 | Confident | shipped field positioned after stage_metrics in template | Logical grouping — post-pipeline metadata. Easily moved if preferred | S:50 R:95 A:70 D:80 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
