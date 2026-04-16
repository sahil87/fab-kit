# Intake: Operator Terminal-Safe Status Symbols

**Change**: 260416-edq9-operator-terminal-safe-status-symbols
**Created**: 2026-04-16
**Status**: Draft

## Origin

> Replace operator status emoji (🟢🟡🔴⏸) with terminal-safe single-width Unicode symbols to fix terminal display corruption in tmux.

One-shot change request with specific symbol mappings already decided. The problem was identified during operator usage — emoji characters in the status frame cause rendering corruption in tmux sessions.

## Why

The operator status frame in `fab-operator.md` uses emoji characters (🟢, 🟡, 🔴, ⏸) for health indicators. These characters live in the Supplementary Multilingual Plane (SMP) and cause three concrete problems:

1. **Variable-width rendering** — some terminals render SMP emoji as 2 cells, others as 1. This breaks the column alignment of the status frame, making the output unreadable when columns shift unpredictably.
2. **tmux cursor miscalculation** — tmux tracks cursor position by character width. When emoji width doesn't match tmux's expectation, the entire frame corrupts on redraw, requiring a manual `clear` or pane kill.
3. **Font gaps** — monospace terminal fonts often lack emoji glyphs, rendering them as empty boxes or tofu characters, which defeats the purpose of visual status indicators.

The ⏸ character (U+23F8 DOUBLE VERTICAL BAR) for paused watches carries the same risk despite being in the BMP — its width behavior varies across terminal emulators.

If unfixed, the operator — a long-running coordination tool — produces garbled output that undermines its core value: at-a-glance status visibility.

## What Changes

### Symbol Replacement in `src/kit/skills/fab-operator.md`

Replace all SMP emoji and variable-width Unicode status indicators with single-width BMP Unicode symbols that render reliably in monospace terminal fonts:

| Current | Replacement | Unicode | Meaning |
|---------|-------------|---------|---------|
| 🟢 | ● | U+25CF BLACK CIRCLE | active/healthy |
| 🟡 | ◌ | U+25CC DOTTED CIRCLE | idle/new items |
| 🔴 | ✗ | U+2717 BALLOT X | stuck/errored |
| ⏸ | – | U+2013 EN DASH | paused |
| ✓ | ✓ | (unchanged) | complete |

The ✓ (U+2713 CHECK MARK) is already a single-width BMP character and stays as-is.

### Affected Locations

Three areas in `fab-operator.md` require updates:

1. **Status frame example** (lines 178-184) — the code block showing the tick output with status indicators for changes and watches
2. **Health legend** (lines 203, 205) — the inline documentation defining what each symbol means for change health and watch health
3. **Autopilot reference** (line 420) — prose mentioning 🟢/🟡 as current-item indicators in autopilot queue progress

### Design Decision: Why These Specific Symbols

Three options were evaluated:

- **Option 1 (ANSI-colored ●)**: Single circle with ANSI color codes for green/yellow/red. Rejected because Claude Code's markdown renderer may strip ANSI escape sequences, making the symbols indistinguishable.
- **Option 2 (Distinct BMP shapes)**: Different single-width Unicode shapes (●, ◌, ✗, –) with no color dependency. Selected — each symbol is visually distinct by shape alone, not just color.
- **Option 3 (ASCII-only)**: Plain ASCII characters like `[OK]`, `[!!]`, `[--]`. Rejected as too noisy and less scannable compared to single-character symbols.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator skill documentation to reflect new status symbols

## Impact

- **`src/kit/skills/fab-operator.md`** — the only file modified. All changes are to documentation/skill prose, not executable code.
- **No migration needed** — `.fab-operator.yaml` stores no emoji; the symbols exist only in the skill file that guides the agent's output format.
- **Deployed copies** — `.claude/skills/fab-operator/SKILL.md` will reflect the change after `fab sync`.
- **Spec file** — `docs/specs/skills/SPEC-fab-operator.md` needs a corresponding update per constitution.

## Open Questions

(None — symbol mappings and rationale are fully specified.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace 🟢 with ● (U+25CF) for active/healthy | Discussed — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 2 | Certain | Replace 🟡 with ◌ (U+25CC) for idle/new items | Discussed — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 3 | Certain | Replace 🔴 with ✗ (U+2717) for stuck/errored | Discussed — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 4 | Certain | Replace ⏸ with – (U+2013) for paused | Discussed — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 5 | Certain | Keep ✓ unchanged | Discussed — user confirmed it stays as-is | S:95 R:95 A:95 D:95 |
| 6 | Certain | Option 2 (distinct BMP shapes) over ANSI-colored or ASCII-only | Discussed — user chose Option 2 with rationale (ANSI stripping risk, ASCII noise) | S:95 R:85 A:90 D:95 |
| 7 | Confident | SPEC-fab-operator.md needs corresponding update | Constitution mandates spec updates for skill changes | S:70 R:80 A:90 D:85 |
| 8 | Confident | No migration needed — symbols are in skill prose only, not persisted state | No emoji stored in .fab-operator.yaml or .status.yaml | S:75 R:85 A:85 D:90 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
