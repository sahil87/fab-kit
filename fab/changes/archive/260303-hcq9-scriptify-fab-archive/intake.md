# Intake: Scriptify Fab-Archive

**Change**: 260303-hcq9-scriptify-fab-archive
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Backlog item [hcq9]: "See if the some or all steps for fab-archive skill can be offloaded to a script to make it faster"

## Why

`/fab-archive` is slow because every step runs as agent-driven file reads and edits — sequential tool calls with round-trip latency for each one. Most steps are purely mechanical (move folder, update index, clear pointer) and don't need agent intelligence at all. Offloading these to a shell script eliminates round-trips and lets the agent focus only on the parts that genuinely require reasoning (backlog keyword matching and interactive confirmation).

The same pattern has already been applied successfully elsewhere in fab-kit: `changeman.sh`, `statusman.sh`, and `preflight.sh` all handle mechanical operations in shell, with skills orchestrating only the intelligent parts.

If we don't fix this: every archive operation pays ~10-15 sequential tool calls of latency for work that a single shell script could do in milliseconds.

## What Changes

### Create `archiveman.sh` script

New script at `fab/.kit/scripts/lib/archiveman.sh` with subcommands covering the mechanical steps:

#### `archiveman.sh archive <change>`

Performs Steps 1, 2, 3, 5 of the current skill in one shot:

1. **Clean**: Delete `.pr-done` from change folder if present
2. **Move**: `fab/changes/{name}/` → `fab/changes/archive/{name}/` (create `archive/` if needed, skip if already there)
3. **Index**: Prepend entry to `fab/changes/archive/index.md` (create with header if missing, backfill existing archived folders). Takes a `--description` argument for the entry text — the agent computes this from intake before calling the script
4. **Pointer**: If active change matches, clear via `changeman.sh switch --blank`

Outputs structured YAML (like preflight.sh) so the skill can parse results:

```yaml
clean: removed        # or: not_present
move: moved           # or: already_archived
index: updated        # or: created
pointer: cleared      # or: skipped
name: 260303-xxxx-slug
```

#### `archiveman.sh restore <change> [--switch]`

Performs all 3 restore steps:

1. **Move**: `fab/changes/archive/{name}/` → `fab/changes/{name}/` (skip if already there)
2. **Index**: Remove entry from `fab/changes/archive/index.md`
3. **Pointer**: If `--switch`, run `changeman.sh switch {name}`

Outputs:

```yaml
move: restored        # or: already_in_changes
index: removed        # or: not_found
pointer: switched     # or: skipped
name: 260303-xxxx-slug
```

#### `archiveman.sh list`

List archived changes (for restore mode's fuzzy matching). One folder name per line.

#### Resolution

Uses `resolve.sh` internally for the `<change>` argument. For restore mode, resolves against `fab/changes/archive/` folder names instead of `fab/changes/`.

### Slim down the skill

The `/fab-archive` skill becomes an orchestrator:

**Archive mode**:
1. Run preflight (existing)
2. Extract description from intake's Why section (agent intelligence)
3. Call `archiveman.sh archive <change> --description "..."` (one shell call replaces Steps 1, 2, 3, 5)
4. Run Step 4 backlog matching (agent intelligence — keyword scan + interactive confirmation)
5. Parse YAML output, format user-facing report

**Restore mode**:
1. Call `archiveman.sh list` to get archived folders
2. Fuzzy match the user's argument (agent or script — see assumption #4)
3. Call `archiveman.sh restore <match> [--switch]` (one shell call replaces all 3 restore steps)
4. Parse YAML output, format user-facing report

### Backlog matching stays in the skill

Step 4 (mark backlog items done) remains agent-driven:
- 4a (exact ID match) could be scripted but is trivial
- 4b (keyword extraction + fuzzy matching) genuinely benefits from agent reasoning
- 4c (interactive confirmation) requires agent interaction

Keeping all of Step 4 in the skill avoids splitting the backlog logic across two places.

## Affected Memory

No memory files affected — this is an internal tooling optimization.

## Impact

- `fab/.kit/scripts/lib/archiveman.sh` — new script
- `.claude/skills/fab-archive/SKILL.md` — modified (slimmed to orchestrator)
- `fab/.kit/scripts/lib/resolve.sh` — may need a mode/flag to resolve against archive folder (or archiveman handles this internally)

## Open Questions

- Should `archiveman.sh` handle the archive index backfill (scanning existing archived folders that predate the index)? Or is that edge case rare enough to leave in the skill?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Create `archiveman.sh` following existing script patterns | Consistent with changeman/statusman/preflight pattern; constitution requires shell scripts for workflow logic | S:90 R:85 A:95 D:90 |
| 2 | Certain | Backlog matching (Step 4) stays in the skill | Keyword extraction and fuzzy matching genuinely need agent reasoning; interactive confirmation needs agent | S:85 R:85 A:90 D:90 |
| 3 | Confident | Script outputs structured YAML like preflight.sh | Existing pattern — preflight.sh outputs YAML for skill parsing; consistent and parseable | S:80 R:85 A:85 D:75 |
| 4 | Confident | Restore fuzzy matching handled by script (substring match) not agent | Current restore mode does case-insensitive substring matching — this is mechanical, not reasoning | S:75 R:80 A:75 D:70 |
| 5 | Confident | Description passed as `--description` arg, computed by agent before script call | Agent extracts 1-2 sentences from intake Why section; script just writes it to index | S:80 R:85 A:80 D:75 |
| 6 | Tentative | Index backfill (scanning existing folders without index entries) stays in the script | Edge case but deterministic — list folders, check index, add missing entries | S:55 R:80 A:65 D:55 |

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
