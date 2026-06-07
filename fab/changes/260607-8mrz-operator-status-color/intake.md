# Intake: Restore Color to Operator Status Output

**Change**: 260607-8mrz-operator-status-color
**Created**: 2026-06-07
**Status**: Draft

## Origin

Initiated from backlog item `[8mrz]` (2026-06-01):

> I had shy'd away from special utf8 chars in the operator's status outputs every 3 mins. There are these in older version of the operator. Now that a lot of run-kit's display bugs are fixed, we can get the utf8 colored characters back.

**Mode**: conversational (`/fab-new 8mrz`). Before generating this intake, the agent traced the history of the operator's status indicators:

- **`9d79b053`** (`feat: add tick output format with status frame and color indicators`) introduced the box-drawn tick status frame with **colored SMP emoji** health indicators: `🟢` active, `🟡` idle, `🔴` stuck, `⏸` paused.
- **PR #333 / `82ec6835`** (`fix: Replace operator status emoji with terminal-safe BMP symbols`) **removed** those emoji because *SMP emoji are variable-width and corrupted tmux rendering*. It substituted single-width BMP glyphs: `●` active/healthy, `◌` idle/new-items, `✗` stuck/errored, `–` paused. (`✓` complete was already single-width BMP and was left unchanged.)

So the regression the user is asking to reverse is **the loss of color**, and the *original* removal reason was **variable glyph width**, not color itself.

**Key decision reached in conversation** (presented as three options, user chose option 1):

The user chose to **restore color via ANSI escape codes wrapped around the existing single-width BMP symbols** — *not* to revert to the SMP emoji. This reconciles the two competing constraints: color is restored (satisfying the backlog request) while the glyphs remain fixed single-width (so the variable-width tmux corruption that motivated #333 cannot recur). The user's run-kit display-bug fixes are leveraged for *rendering ANSI color*, not for tolerating variable-width emoji.

## Why

**Problem.** The operator emits a status frame every loop tick (default 3 minutes). After PR #333 the health column is monochrome (`● ◌ ✗ – ✓`). For an at-a-glance multi-agent dashboard that the user stares at across long autopilot runs, color is the fastest channel for conveying health: green=healthy, yellow=attention, red=problem. Monochrome glyphs force the user to read the shape of each glyph rather than scan color, which is slower and more error-prone when many agents are tracked.

**Consequence of not fixing.** The operator dashboard stays harder to scan than it was originally. The user explicitly recalls the colored version being better and wants it back now that the blocker (run-kit display bugs) is resolved.

**Why this approach over alternatives.** Three approaches were weighed:
1. **ANSI color on the existing BMP symbols** *(chosen)* — color returns, width stays a fixed single cell, so #333's corruption cannot recur. Safest reconciliation of "colored" + "terminal-safe".
2. **Revert to SMP emoji** (`🟢🟡🔴⏸`) — maximal match to "get the colored characters back", but reintroduces the exact variable-width glyphs that caused the corruption; depends entirely on run-kit fixes holding.
3. **Colored emoji with explicit fixed-width padding** — keeps the emoji look but adds width-discipline bookkeeping to every column.

Option 1 was chosen because it delivers the requested color while structurally eliminating the failure mode (rather than relying on downstream fixes to compensate for variable width).

## What Changes

This is a **skill-documentation change** to `src/kit/skills/fab-operator.md`. The operator status frame is **LLM-rendered text** — the agent following the skill emits the frame; there is no Go code that prints it (the Go side only provides `fab operator tick-start`, which emits the tick counter and timestamp). Therefore the change is entirely in the skill's rendering instructions and legends, plus the canonical spec, with the memory file updated at hydrate.

### Change area 1: ANSI color the health indicators in the status frame (§4 Tick Behavior)

The health glyphs stay exactly as they are (`● ◌ ✗ – ✓`) — single-width BMP, never reverted to emoji. What's added is an **ANSI color wrap** per health state. The mapping:

| State | Glyph | ANSI color | Code |
|-------|-------|-----------|------|
| active / healthy | `●` | green | `\e[32m…\e[0m` |
| idle / has-new-items | `◌` | yellow | `\e[33m…\e[0m` |
| stuck / errored | `✗` | red | `\e[31m…\e[0m` |
| complete | `✓` | green | `\e[32m…\e[0m` |
| paused | `–` | grey/default | none (or `\e[90m…\e[0m`) |

Rendered example of the status frame (ANSI shown literally for the spec; the terminal displays color):

```
── Operator ── 17:32 ── tick #47 ── 7 tracked ──

  [change]  r3m7         ▶ \e[32m●\e[0m apply → review
  [change]  k8ds         ▶ \e[33m◌\e[0m review · idle 18m ⚠
  [change]  ab12           \e[32m●\e[0m hydrate \e[32m✓\e[0m
  [change]  ef56           \e[31m✗\e[0m apply · idle 32m ⚠
  [watch]   gmail-deploys  \e[33m◌\e[0m 1 new · 2m ago
  [watch]   linear-bugs    \e[32m●\e[0m 2 known · 1 completed · 3m ago
  [watch]   slack-alerts   \e[32m●\e[0m 0 new · 1m ago

───────────────────────────────────────────────────────────
```

Note: only the **health glyph** is colored. The autopilot marker `▶`, the bracketed type prefix, the IDs, and the detail text remain uncolored to keep color signal concentrated on health (the one dimension color is meant to encode). The `⚠` stuck-idle warning marker stays uncolored (its meaning is already carried by the red `✗`).

### Change area 2: Update the health legends (§4)

The two legend lines must describe the color, not just the glyph:

- **Change health**: `● active` (green), `◌ idle` (yellow), `✗ stuck` (red, >15m idle at non-terminal), `✓ complete` (green).
- **Watch health**: `● healthy` (green), `◌ has new unprocessed items` (yellow), `✗ errored` (red, `last_error` set), `– paused` (grey/default).

The column-layout table's **Health** row description stays "Status indicator — universal position across all types" (no change needed; it's color-agnostic).

### Change area 3: Update the autopilot queue reference (§6)

The line "the one showing ●/◌ is current" stays as-is (glyphs unchanged); if color is mentioned there, note green/yellow.

### Change area 4: Graceful degradation note

Add a one-line rendering note to §4 stating that the health glyph is wrapped in ANSI SGR color codes (`\e[3Xm…\e[0m`), and that terminals without color support degrade to the bare single-width BMP glyph (the glyph alone still disambiguates every state — color is redundant reinforcement, not the sole signal). This preserves the terminal-safety guarantee that motivated #333: even with zero color, the frame renders correctly because the glyphs are unchanged single-width BMP.

### Change area 5: Window-prefix glyphs are out of scope

The `»` (active-monitoring) and `›` (done-marker) window-name prefixes (§4 Monitored Set / Removal) are **not** part of this change. They are tmux *window-name* characters set via `fab pane window-name`, not status-frame health indicators, and they are not colored. They stay as-is.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) The memory table row `260416-edq9-operator-terminal-safe-status-symbols` records that SMP emoji were replaced by monochrome BMP symbols. After this change ships, a new hydrate row will record that color was restored via ANSI escapes on those same BMP glyphs (glyphs unchanged, color re-added). The existing row is left intact as history; a new row is appended at hydrate.

## Impact

- **`src/kit/skills/fab-operator.md`** (canonical skill source) — §4 status-frame example, both health legends, the rendering/degradation note; §6 autopilot reference if it mentions color. Edit the **source** under `src/kit/`, never the deployed `.claude/skills/` copy (which `fab sync` regenerates).
- **`docs/specs/skills/SPEC-fab-operator.md`** — required by the constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"). This spec is a flow/tool-usage diagram and does not currently enumerate the health symbols; update only if the change touches anything it documents (likely a no-op or a one-line note).
- **`docs/specs/operator.md`** — the prose operator spec; check whether it describes the status-frame health indicators and update if so.
- **No Go code changes.** The frame is LLM-rendered; `fab operator tick-start` and `fab operator time` are unaffected. No CLI signature changes, so no `_cli-fab.md` update is triggered.
- **No migration.** This changes no on-disk user data (config, `.status.yaml`, `.fab-operator.yaml` schema all unchanged).
- **Distribution**: change reaches users via the normal `fab sync` of the next kit release.

## Open Questions

- None blocking. The color-vs-glyph approach (the only high-blast-radius decision) was resolved in conversation: ANSI color on existing BMP glyphs.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Restore color via ANSI escape codes wrapping the existing single-width BMP glyphs (`● ◌ ✗ – ✓`), not by reverting to SMP emoji | Discussed — user chose option 1 over reverting to emoji (option 2) and emoji-with-padding (option 3). Directly reconciles "colored" with the variable-width corruption #333 fixed. | S:98 R:80 A:90 D:95 |
| 2 | Certain | Color mapping: green=active/healthy/complete, yellow=idle/new-items, red=stuck/errored, grey/default=paused | Derived from the original pre-#333 SMP emoji semantics (🟢 green, 🟡 yellow, 🔴 red, ⏸ neutral) — a 1:1 color carry-over, no new convention invented. | S:90 R:85 A:90 D:88 |
| 3 | Certain | This is a skill-doc change in `src/kit/skills/fab-operator.md`; no Go code changes | Verified: the status frame is LLM-rendered per the skill; Go only provides `tick-start`/`time`. Grep confirmed no frame-rendering code in `src/go/`. | S:95 R:90 A:95 D:92 |
| 4 | Confident | Only the health glyph is colored; autopilot marker, type prefix, IDs, detail text, and `⚠` stay uncolored | Concentrating color on the single health dimension maximizes scan value and matches the original design (only the emoji was colored). Easily adjusted later via `/fab-clarify` if the user wants more color. | S:70 R:88 A:75 D:78 |
| 5 | Confident | Add a graceful-degradation note: glyphs alone disambiguate every state, so no-color terminals still render correctly | Preserves #333's terminal-safety guarantee; glyphs are unchanged single-width BMP. Standard ANSI practice (color as redundant reinforcement). | S:72 R:90 A:82 D:80 |
| 6 | Confident | The `»`/`›` window-name prefix glyphs are out of scope | They are tmux window-name characters, not status-frame health indicators, and are not colored. The backlog item is specifically about the 3-minute "status outputs". | S:75 R:85 A:85 D:82 |
| 7 | Confident | Update the canonical spec(s) (`SPEC-fab-operator.md` and/or `operator.md`) per the constitution's skill→spec rule; memory row appended at hydrate | Constitution mandates skill→spec updates; memory is hydrate-owned. The specific spec edit is small/possibly no-op since the specs don't enumerate health glyphs. | S:80 R:88 A:88 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
