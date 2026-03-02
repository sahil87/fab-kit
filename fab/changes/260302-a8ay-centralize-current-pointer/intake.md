# Intake: Centralize Current Pointer Format

**Change**: 260302-a8ay-centralize-current-pointer
**Created**: 2026-03-02
**Status**: Draft

## Origin

> Change fab/current to a two-line plain text format storing both the 4-char change ID (line 1) and full folder name (line 2). Update resolve.sh to read line 2 for folder name resolution. Update changeman.sh switch to write both lines, rename to update line 2 only. Update dispatch.sh to poll against line 1 (ID). Update fab-discuss and fab-archive skills to go through resolve.sh/changeman.sh instead of reading fab/current directly — zero direct contact with the file format from skills. Update logman.sh direct-read path similarly.

Conversational. The discussion traced the full read/write chain for `fab/current`, identified that all scripts already delegate to `resolve.sh` internally, explored three format options (4-char ID only, YAML, two-line plain text), and converged on two-line plain text as the right balance of simplicity and zero-cost reads.

## Why

1. **Agent output noise**: Skills currently pass full folder names (e.g., `260302-ye59-tu-fresh-flag-reduced-ttl`) to every script call. The 4-char ID (`ye59`) is sufficient and produces cleaner, shorter output.

2. **Format knowledge leaks**: Two skills (`fab-discuss`, `fab-archive`) and one script (`logman.sh` command subcommand) read `fab/current` directly instead of going through `resolve.sh`/`changeman.sh`. This couples them to the file's format — any format change requires updating every reader. The file format should be an implementation detail known only to `resolve.sh` (reads) and `changeman.sh` (writes).

3. **Rename no-op opportunity**: Today `changeman.sh rename` must update `fab/current` when the renamed change is active (compare stored folder name, write new folder name). With the two-line format, the ID on line 1 is immutable across renames — only line 2 (folder name) needs updating, and the comparison is simpler.

If we don't fix this: every future format change to `fab/current` requires auditing skills and scripts that bypass the centralized access layer. The coupling accumulates.

## What Changes

### `preflight.sh` — Emit `id:` Field in YAML Output

This is the root change that enables shorter agent script calls. Currently preflight emits:

```yaml
name: 260302-ye59-tu-fresh-flag-reduced-ttl
change_dir: fab/changes/260302-ye59-tu-fresh-flag-reduced-ttl
```

Add an `id:` field extracted from the folder name:

```yaml
id: ye59
name: 260302-ye59-tu-fresh-flag-reduced-ttl
change_dir: fab/changes/260302-ye59-tu-fresh-flag-reduced-ttl
```

Use `resolve.sh --id` (which calls `extract_id` internally) to derive it from the resolved name.

### `_preamble.md` §2 — Agent Uses `id` for Script Calls

Update the Change Context instructions so that after parsing preflight's YAML output, the agent uses the `id` field (not `name`) when calling scripts. All scripts already accept 4-char IDs via `resolve.sh`.

Before (current preamble instructs):
```bash
bash fab/.kit/scripts/lib/statusman.sh finish 260302-ye59-tu-fresh-flag-reduced-ttl intake fab-continue
bash fab/.kit/scripts/lib/logman.sh command "fab-continue" "260302-ye59-tu-fresh-flag-reduced-ttl"
```

After:
```bash
bash fab/.kit/scripts/lib/statusman.sh finish ye59 intake fab-continue
bash fab/.kit/scripts/lib/logman.sh command "fab-continue" "ye59"
```

The `name` field remains available for display, path construction, and artifact generation (e.g., writing `{YYMMDD-XXXX-slug}` in intake/spec headers). The `id` field is for script invocations.

### `fab/current` — Two-Line Plain Text Format

Change from single-line (full folder name) to two-line format:

```
ye59
260302-ye59-tu-fresh-flag-reduced-ttl
```

- **Line 1**: 4-char change ID (for display, polling, quick comparison)
- **Line 2**: Full folder name (for path construction, zero-cost reads)

No trailing newline after line 2. No YAML, no parsing dependency. Readable with `sed -n '1p'` / `sed -n '2p'` or `head -1` / `tail -1`.

### `resolve.sh` — Read Line 2

The default mode (no override argument) currently does:

```bash
name=$(tr -d '[:space:]' < "$current_file")
```

Change to read line 2 specifically for the folder name. Line 1 is available but not used by resolve's default mode (it resolves to folder name, then output modes derive from that).

The `extract_id` function and `--id` output mode remain unchanged — they work on the folder name.

### `changeman.sh switch` — Write Both Lines

Currently writes: `printf '%s' "$resolved" > "$FAB_ROOT/current"`

Change to write both the 4-char ID (extracted from the resolved folder name) and the full folder name on two lines.

### `changeman.sh rename` — Update Line 2 Only

Currently compares stored folder name to old name, writes new folder name. The ID doesn't change during rename (the `YYMMDD-XXXX` prefix is immutable), so:

- Compare line 1 (ID) to the rename source's ID — simpler, always matches
- Update line 2 (folder name) to the new folder name
- Line 1 stays unchanged

### `changeman.sh switch --blank` — No Change

Already deletes the file entirely. Two-line format doesn't affect this.

### `dispatch.sh` — Poll Line 1

Currently polls `fab/current` and compares content to `CHANGE_ID` (the full folder name). Change to compare against line 1 (the 4-char ID). Simpler, shorter comparison.

The `CHANGE_ID` variable in dispatch.sh is currently the `fs-change-id` (full folder name). Either extract the 4-char ID from it for comparison, or add a new variable.

### `logman.sh` — Remove Direct `fab/current` Read

The `command` subcommand has a best-effort direct-read path (when no change argument is provided):

```bash
# No change arg: read fab/current directly, silently exit 0 on any failure.
```

Change to delegate to `resolve.sh` instead, maintaining the same silent-exit-on-failure behavior. This eliminates the last script-level direct reader.

### `fab-discuss` Skill — Use `resolve.sh`

Currently instructs the agent to read `fab/current` directly and construct paths from the stored value. Change to instruct the agent to call `resolve.sh` (or `resolve.sh --folder`) to get the active change name, then use that for `.status.yaml` lookup.

### `fab-archive` Skill — Use `changeman.sh`

Currently instructs the agent to:
- Read `fab/current` to check if the archived change is active
- Delete `fab/current` to clear the pointer
- Write `fab/current` on restore with `--switch`

Change all three operations to go through `changeman.sh`:
- Use `changeman.sh resolve` (or `resolve.sh`) to check the active change
- Use `changeman.sh switch --blank` to clear the pointer
- Use `changeman.sh switch <name>` on restore with `--switch`

### `_preamble.md` — Update `fab/current` References

The preamble references `fab/current` in the Change-name override section and context loading. These are descriptive (explaining what preflight does internally) and may not need changes, but should be reviewed for accuracy.

### Memory and Specs — Update After Implementation

- `docs/memory/fab-workflow/change-lifecycle.md` — "Active Change Tracking" section documents the current single-line format
- `docs/memory/fab-workflow/kit-scripts.md` — documents resolve.sh's default mode reading `fab/current`

These will be updated during hydrate.

## Test Changes

Tests use bats (bash automated testing). Test files live alongside source in `src/lib/{script}/test.bats` and `src/scripts/pipeline/test.bats`.

### Existing Tests That Need Updating

**`src/lib/resolve/test.bats`**:
- `"no argument reads fab/current"` (line 107) — currently writes single-line `printf '260228-a1b2-test-change'`. Update to write two-line format, assert resolve still returns the folder name from line 2.
- `"fab/current with trailing whitespace"` (line 115) — same: update to two-line format with whitespace handling on line 2.

**`src/lib/changeman/test.bats`**:
- `"switch: writes fab/current"` (line 523) — currently asserts `cat "$FAB_ROOT/current"` equals the full folder name. Update to assert line 1 is the 4-char ID and line 2 is the full folder name.
- `"rename updates fab/current when it points to old folder"` (line 305) — currently writes single-line format and asserts new single-line value. Update to write two-line format, assert line 1 unchanged (same ID), line 2 updated to new folder name.
- `"rename does not modify fab/current when it points to different change"` (line 315) — update to two-line format, assert both lines unchanged.
- `"resolve: reads fab/current when no override"` (line 467) — update to write two-line format.
- `"resolve: fab/current with trailing whitespace resolves"` (line 475) — update to two-line format.

**`src/lib/logman/test.bats`**:
- `"command with cmd only resolves via fab/current"` (line 91) — currently writes `echo "$CHANGE_NAME"`. Update to two-line format.
- `"command with cmd only and stale fab/current exits 0 silently"` (line 125) — writes invalid name. Update to test the new resolution path (logman delegates to resolve.sh instead of direct read).

**`src/lib/preflight/test.bats`**:
- `set_current` helper (line 48) — currently writes single-line. Update to write two-line format. All 15+ tests using this helper are affected.

### New Tests to Add

**`src/lib/resolve/test.bats`** — new cases:
- `"fab/current two-line format: reads folder name from line 2"` — write two-line fab/current (ID on line 1, folder name on line 2), assert `--folder` returns the folder name.
- `"fab/current two-line format: --id extracts from line 2 folder name"` — same setup, assert `--id` returns the 4-char ID.
- `"fab/current single-line backward compat"` — (if backward compat is desired) write old single-line format, assert it still resolves. Otherwise, this test documents the intentional break.

**`src/lib/changeman/test.bats`** — new cases:
- `"switch: fab/current line 1 is 4-char ID"` — after switch, assert `sed -n '1p'` of fab/current matches the 4-char ID.
- `"switch: fab/current line 2 is full folder name"` — after switch, assert `sed -n '2p'` matches the resolved folder name.
- `"rename: fab/current line 1 unchanged after rename"` — rename a change, assert line 1 (ID) is identical before and after.
- `"rename: fab/current line 2 updated after rename"` — assert line 2 reflects the new folder name.

**`src/lib/preflight/test.bats`** — new cases:
- `"YAML output includes id field"` — run preflight on a `YYMMDD-XXXX-slug` change, assert output contains `id: XXXX`.
- `"id field matches 4-char portion of change name"` — parse the id field, assert it matches the second segment of the folder name.

**`src/scripts/pipeline/test.bats`** — new cases (if dispatch.sh polling logic changes):
- `"poll_switch: matches 4-char ID in fab/current line 1"` — test the polling comparison uses line 1 instead of full content match.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update "Active Change Tracking" section — two-line format, centralized access pattern
- `fab-workflow/kit-scripts`: (modify) Update resolve.sh default mode description, logman.sh direct-read removal

## Impact

- **Scripts**: `preflight.sh` (new `id:` field), `resolve.sh`, `changeman.sh`, `logman.sh`, `dispatch.sh` — all in `fab/.kit/scripts/lib/` or `fab/.kit/scripts/pipeline/`
- **Preamble**: `_preamble.md` §2 — agent uses `id` for script calls instead of `name`
- **Skills**: `fab-discuss`, `fab-archive` — centralize access through resolve.sh/changeman.sh
- **Tests**: Any existing tests for these scripts need updating for the new format

## Open Questions

- None — all design decisions were resolved during the discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Two-line plain text format (ID line 1, folder name line 2) | Discussed — user chose over YAML (yq hot-path cost) and single-ID-only (requires folder scan on every default-mode call) | S:95 R:85 A:90 D:95 |
| 2 | Certain | Skills must not read fab/current directly — go through resolve.sh/changeman.sh | Discussed — user stated "zero contact with implementation details keeps resolve.sh more flexible" | S:95 R:80 A:90 D:95 |
| 3 | Certain | resolve.sh reads line 2 for folder name (zero-cost path preserved) | Discussed — avoids matching overhead on every default-mode call, same performance as today | S:90 R:90 A:85 D:90 |
| 4 | Certain | changeman.sh rename pointer update simplified (ID unchanged, update folder name only) | Discussed — user confirmed the YYMMDD-XXXX prefix is immutable across renames | S:90 R:90 A:90 D:95 |
| 5 | Certain | dispatch.sh polls line 1 (4-char ID) for switch confirmation | Discussed — simpler short-string comparison | S:85 R:85 A:85 D:90 |
| 6 | Certain | logman.sh direct-read path replaced with resolve.sh delegation | Discussed — same centralization principle as skills | S:85 R:85 A:85 D:90 |
| 7 | Confident | fab-archive uses changeman.sh switch/switch --blank instead of direct file ops | Discussed — follows the centralization principle, changeman.sh already has these subcommands | S:85 R:80 A:80 D:85 |
| 8 | Confident | No trailing newline after line 2 | Inferred from existing convention (current fab/current uses printf '%s' with no newline) | S:70 R:95 A:80 D:85 |
| 9 | Certain | preflight.sh emits `id:` field in YAML output | Discussed — user chose this over agent-side extraction from `name` field | S:90 R:90 A:90 D:95 |
| 10 | Certain | Preamble §2 instructs agent to use `id` (not `name`) for script calls | Discussed — this is the root fix for the original problem (agent output noise from full folder names) | S:95 R:85 A:90 D:95 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
