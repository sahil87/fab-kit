# Intake: Unified Tick Status List

**Change**: 260326-oxgu-unified-tick-status-list
**Created**: 2026-03-26
**Status**: Draft

## Origin

> Update fab-operator7.md skill: the tick status frame should use a single unified list for all tracked items — monitored changes, watches, and other tracked sources should render as entries in one list with type indicators

## Why

The current operator tick status frame (`fab/.kit/skills/fab-operator7.md` §4 Tick Behavior) renders monitored changes and watches as visually separate groups with different formatting. Monitored changes appear as a list of change IDs with stage/status indicators, followed by a separate watches section prefixed with a 👁 icon. This separation means the user must mentally merge two lists to get the full operational picture, and adding future tracked-source types (e.g., scheduled triggers, PR freshness checks) would require yet another visual section.

A single unified list with type indicators gives the operator a flat, scannable view of everything it's tracking — every entry gets the same structural treatment (type badge + identifier + status detail), making it trivial to add new source types later.

## What Changes

### Tick Status Frame Rendering (§4 Tick Behavior)

The status frame currently renders two separate blocks:

```
── Operator ── 17:32 ── tick #47 ── 3 monitored · autopilot 1/3 · 1 watch ──

  r3m7  🟢 apply → review
  k8ds  🟡 review · idle 18m ⚠
  ab12  🟢 hydrate ✓

  👁 linear-bugs  2 known · 1 completed · last check 17:29

───────────────────────────────────────────────────────────
```

This SHALL be replaced with a single unified list where every tracked item is an entry with consistent column structure:

```
── Operator ── 17:32 ── tick #47 ── 7 tracked ──

  [change]  r3m7         ▶ 🟢 apply → review
  [change]  k8ds         ▶ 🟡 review · idle 18m ⚠
  [change]  ab12           🟢 hydrate ✓
  [change]  ef56           🔴 spec · idle 32m ⚠
  [watch]   linear-bugs    🟢 2 known · 1 completed · 3m ago
  [watch]   slack-alerts   🟢 0 new · 1m ago
  [watch]   gmail-deploys  🟡 1 new · 2m ago

───────────────────────────────────────────────────────────
```

### Column Structure

Every row follows the same column layout:

| Column | Content |
|--------|---------|
| Type | `[change]` / `[watch]` — bracketed type prefix |
| ID | Change ID or watch name |
| Autopilot | `▶` if autopilot-driven, blank otherwise |
| Health | Status emoji — universal across all types |
| Detail | Type-specific status text |

### Specific changes

1. **Header line**: Replace the separate counts (`3 monitored · autopilot 1/3 · 1 watch`) with a single total count (`N tracked`). Autopilot is no longer a header-level concept — it's a per-change property shown via the `▶` symbol on autopilot-driven entries.

2. **Type indicators**: Each entry gets a bracketed type prefix:
   - `[change]` — monitored pipeline changes
   - `[watch]` — watch sources (Linear, Slack, etc.)
   - Future types follow the same `[type]` pattern

3. **Autopilot as a change property**: The `▶` symbol marks changes being driven by the autopilot queue. This replaces the header-level `autopilot 1/3` indicator — queue position and progress are readable directly from the list (which entries have `▶`, which are complete). Non-autopilot changes (manually enrolled or watch-spawned) have no arrow.

4. **Universal health emoji**: Both changes and watches get a status emoji in the same column position:
   - **Changes**: 🟢 active, 🟡 idle, 🔴 stuck (>15m idle at non-terminal), ✓ complete (existing semantics)
   - **Watches**: 🟢 healthy (last query succeeded), 🟡 has new unprocessed items, 🔴 errored (`last_error` set), ⏸ paused (`enabled: false`)

5. **Watch detail format update**: Watches use relative timestamps (`3m ago`) instead of absolute (`last check 17:29`) for consistency and scannability.

6. **Single list**: All entries render in one flat list, no blank-line separators between types. Entries are ordered: changes first (sorted by enrollment time), then watches (sorted alphabetically by name).

### Downstream References

The status frame format is referenced in the tick behavior description and the example output block in §4. Both need updating. The watch indicator note (`Watch indicator: 👁`) and stage indicator list in the paragraph after the example should be replaced with the column structure table and unified health emoji definitions.

The `autopilot` field in `.fab-operator.yaml` header summary and the §6 Autopilot section's references to `autopilot 1/3` in the header need updating to reflect the per-entry `▶` approach.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator tick status frame format description

## Impact

- `fab/.kit/skills/fab-operator7.md` — primary change (§4 Tick Behavior example + description)
- `.claude/skills/fab-operator7.md` — deployed copy (updated by fab-sync)
- `docs/specs/skills/SPEC-fab-operator7.md` — spec update if it documents the frame format (per constitution)

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Type indicators use `[type]` bracket syntax | Discussed — user confirmed bracket prefix approach | S:90 R:90 A:85 D:90 |
| 2 | Certain | Changes ordered before watches in the list | Changes are the primary tracked items; watches are secondary/meta — natural reading order | S:70 R:95 A:85 D:80 |
| 3 | Certain | Header uses single "N tracked" count, no per-type counts, no autopilot in header | Discussed — user chose `▶` per-entry symbol over header-level `autopilot 1/3` | S:95 R:90 A:90 D:95 |
| 4 | Certain | Autopilot is a per-change property shown via `▶` symbol | Discussed — user proposed moving autopilot from header to entry-level, chose arrow symbol over queue numbers and `[auto]` prefix | S:95 R:85 A:90 D:95 |
| 5 | Certain | Universal health emoji column for all entry types | Discussed — user confirmed watches should get status emoji (🟢/🟡/🔴/⏸) matching the change emoji column position | S:90 R:90 A:85 D:90 |
| 6 | Confident | Watch detail uses relative timestamps (`3m ago`) instead of absolute | Follows from unified column structure — relative is more scannable at a glance | S:65 R:95 A:75 D:70 |
| 7 | Confident | No blank-line separators between types | "single unified list" implies flat rendering without visual grouping | S:65 R:95 A:70 D:70 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
