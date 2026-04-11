# Intake: Operator Tab Prefix Character

**Change**: 260411-57k0-operator-tab-prefix-char
**Created**: 2026-04-11
**Status**: Draft

## Origin

> The lightning symbol (⚡) used as tmux tab prefix for operator-managed agents makes console output wonky due to its horizontal width. Switch to a more console-friendly character.

One-shot request. The ⚡ prefix was introduced by `260328-iqt8-standardize-tmux-tab-naming` as a visual differentiator for operator-spawned agent tabs. In practice, the emoji's double-width rendering in terminals causes alignment issues in tmux tab bars and other console output that reads or displays tab names.

## Why

The ⚡ (U+26A1, "High Voltage") is a multi-byte Unicode character that renders as double-width in most terminal emulators. This means:

1. **Tmux tab bar misalignment** — the tab bar's width calculations assume single-width characters, causing tab names to overlap or clip
2. **Console output wonkiness** — any tool that reads or formats tmux window names (e.g., `fab pane map` table alignment) can misrender when the name contains a double-width character
3. **Copy-paste friction** — some terminals and clipboard managers struggle with emoji characters in tab names

The fix is straightforward: replace ⚡ with a single-width ASCII or narrow Unicode character that is still visually distinct.

## What Changes

### 1. Skill: `src/kit/skills/fab-operator.md`

Replace all 4 `tmux new-window -n "⚡<wt>"` invocations with `tmux new-window -n "»<wt>"`. Affected lines:

- Line ~305: Generic "Open agent tab" example
- Line ~367: "From existing change" spawn path
- Line ~376: "From raw text" spawn path
- Line ~384: "From backlog ID or Linear issue" spawn path

### 2. Memory: `docs/memory/fab-workflow/execution-skills.md`

Update the tab naming convention documentation (around line 555-571) to reflect the new prefix character, replacing all `⚡<wt>` references.

### 3. Spec: `docs/specs/operator.md`

Update the version table entry referencing `⚡<wt>` tab naming (line ~22).

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator tab naming convention from `⚡<wt>` to new prefix

## Impact

- **Skill file**: `src/kit/skills/fab-operator.md` — 4 string replacements in `tmux new-window -n` invocations
- **Memory doc**: `docs/memory/fab-workflow/execution-skills.md` — documentation update
- **Spec doc**: `docs/specs/operator.md` — version table entry update
- **No Go code changes**: The `⚡` prefix is only used in the skill markdown (not in `src/go/`). Note: `fab batch new` uses `fab-<id>` naming, which is a separate concern

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `»` (right guillemet, U+00BB) as replacement prefix | User chose from options — single-width Unicode, visually distinct "points forward" appearance, not used in shells, renders consistently | S:95 R:90 A:95 D:95 |
| 2 | Certain | No space between prefix and worktree name | Carries forward from original ⚡ convention — confirmed in prior change `260328-iqt8` | S:95 R:95 A:95 D:95 |
| 3 | Certain | Same 4 locations in `fab-operator.md` need updating | Grep confirms exactly 4 `⚡<wt>` occurrences in the skill file | S:95 R:95 A:95 D:95 |
| 4 | Certain | No Go binary changes needed | Grep confirms ⚡ does not appear in `src/go/` — it's purely in the skill markdown | S:95 R:95 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
