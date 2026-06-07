# Plan: Restore Color to Operator Status Output

**Change**: 260607-8mrz-operator-status-color
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Skill-documentation change. The operator status frame is LLM-rendered text;
     these requirements govern the rendering instructions in the skill source and
     the corresponding canonical spec. No Go code is in scope. -->

### Operator Status Frame: Colored Health Indicators

#### R1: ANSI-color the health glyphs in the status frame
The `/fab-operator` skill source (`src/kit/skills/fab-operator.md`, §4 Tick Behavior) SHALL instruct the agent to wrap each status-frame **health glyph** in an ANSI SGR color escape, while keeping the glyph characters exactly `● ◌ ✗ – ✓` (single-width BMP — NOT reverted to SMP emoji). Only the health glyph is colored; the autopilot marker `▶`, the `[change]`/`[watch]` type prefix, IDs, the `⚠` stuck-idle marker, and all detail text remain uncolored.

- **GIVEN** the operator renders a tick status frame
- **WHEN** it emits a row whose health glyph is `●` (active/healthy) or `✓` (complete)
- **THEN** the glyph is wrapped green as `\e[32m●\e[0m` / `\e[32m✓\e[0m`
- **AND** an `◌` (idle/new-items) glyph is wrapped yellow `\e[33m◌\e[0m`, an `✗` (stuck/errored) glyph is wrapped red `\e[31m✗\e[0m`, and a `–` (paused) glyph is left uncolored (or grey `\e[90m–\e[0m`)
- **AND** no other element of the row (marker, prefix, ID, detail, `⚠`) carries a color escape

#### R2: Color mapping carried over from the original pre-#333 semantics
The color mapping SHALL be: green = active / healthy / complete; yellow = idle / has-new-items; red = stuck / errored; grey-or-default = paused. This is a 1:1 carry-over of the original SMP-emoji semantics (🟢→green, 🟡→yellow, 🔴→red, ⏸→neutral), not a new convention.

- **GIVEN** the health-state-to-color mapping
- **WHEN** the skill's example frame and both health legends are read
- **THEN** every health state maps to exactly the color above, consistently across the example and both legends

#### R3: Both health legends describe the color
The two legend lines in §4 (Change health, Watch health) SHALL state the color of each glyph in addition to its meaning, so a reader knows green/yellow/red/grey correspond to the health states.

- **GIVEN** the §4 "Change health" and "Watch health" legend lines
- **WHEN** they are read after this change
- **THEN** each glyph entry names its color: `●` active (green), `◌` idle (yellow), `✗` stuck (red), `✓` complete (green); and for watches `●` healthy (green), `◌` has-new (yellow), `✗` errored (red), `–` paused (grey/default)

#### R4: Graceful-degradation rendering note
§4 SHALL carry a one-line note stating the health glyph is wrapped in an ANSI SGR color code (`\e[3Xm…\e[0m`) and that terminals without color support degrade to the bare single-width BMP glyph — which alone still disambiguates every state. Color is redundant reinforcement, not the sole signal; this preserves the terminal-safety guarantee that motivated PR #333.

- **GIVEN** a terminal with no ANSI color support
- **WHEN** the status frame renders
- **THEN** the bare glyph (`● ◌ ✗ – ✓`) still renders correctly and uniquely identifies each state, with no width corruption

#### R5: Autopilot queue reference (§6) consistency
The §6 line describing which queued change is current ("the one showing ●/◌ is current") SHALL keep its glyphs unchanged; if it references color it SHALL use green/yellow consistent with R2.

- **GIVEN** the §6 autopilot queue-progress sentence
- **WHEN** it is read after this change
- **THEN** the `●`/`◌` glyphs are unchanged and any color mention is consistent with the R2 mapping

#### R6: Constitution skill→spec rule satisfied
Per the constitution ("Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"), the change SHALL update `docs/specs/skills/SPEC-fab-operator.md` to reflect that the status-frame health indicators are ANSI-colored.

- **GIVEN** `src/kit/skills/fab-operator.md` is edited
- **WHEN** the change is reviewed against the constitution
- **THEN** `docs/specs/skills/SPEC-fab-operator.md` carries a note documenting the colored health indicators

### Non-Goals

- The `»` (active-monitoring) and `›` (done-marker) tmux window-name prefixes are out of scope — they are window-name characters, not status-frame health indicators, and remain uncolored.
- No Go code change — the frame is LLM-rendered; `fab operator tick-start`/`time` are unaffected.
- No reversion to SMP emoji (`🟢🟡🔴⏸`). Glyphs stay single-width BMP.
- Memory updates (`docs/memory/...`) are owned by hydrate, not apply.

### Design Decisions

1. **ANSI SGR wrap on existing BMP glyphs**: restore color without changing glyph width — *Why*: reconciles "colored" with the variable-width tmux corruption PR #333 fixed; color cannot reintroduce width corruption because the glyphs are unchanged single-cell BMP — *Rejected*: reverting to SMP emoji (reintroduces variable width), emoji-with-padding (adds per-column width bookkeeping).
2. **`\e[3Xm…\e[0m` literal representation**: write escapes literally in the markdown — *Why*: matches the established repo idiom in `src/kit/skills/fab-status.md` (`\e[33m...\e[0m` for the yellow true-impact line); a reader/agent rendering the frame knows to emit real SGR codes — *Rejected*: `\033[`/`\x1b[` forms (less readable, inconsistent with existing usage).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Edit the §4 example status frame in `src/kit/skills/fab-operator.md` (the fenced block ~lines 193-205) to wrap each health glyph in its ANSI color (`\e[32m●\e[0m`, `\e[33m◌\e[0m`, `\e[31m✗\e[0m`, `\e[32m✓\e[0m`; `–` left uncolored); leave `▶`, type prefixes, IDs, detail text, and `⚠` uncolored <!-- R1 R2 -->
- [x] T002 Update the "Change health" and "Watch health" legend lines in §4 of `src/kit/skills/fab-operator.md` to name each glyph's color per the R2 mapping <!-- R3 R2 -->
- [x] T003 Add a one-line graceful-degradation rendering note to §4 of `src/kit/skills/fab-operator.md` (health glyph wrapped in `\e[3Xm…\e[0m`; no-color terminals degrade to the bare single-width BMP glyph which alone disambiguates every state) <!-- R4 -->
- [x] T004 Update the §6 autopilot queue-progress sentence in `src/kit/skills/fab-operator.md` so the `●`/`◌` reference notes green/yellow, consistent with R2 (glyphs unchanged) <!-- R5 R2 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Update `docs/specs/skills/SPEC-fab-operator.md` §4 (Monitoring System) to note the status-frame health indicators are ANSI-colored (green=healthy/active/complete, yellow=idle/new, red=stuck/errored, grey/default=paused), satisfying the constitution's skill→spec rule <!-- R6 -->
- [x] T006 Deploy the edited skill via `fab sync` if safe/expected, and verify the edits: ANSI codes present, glyphs unchanged (`● ◌ ✗ – ✓`, not emoji), legends mention color. `fab sync` ran clean/idempotent; it deploys from the kit cache (not uncommitted worktree source), so the live copy picks up this change at the next kit release — the committed artifact is the `src/kit/` source, as intended <!-- R1 R3 R4 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: Each health glyph in the §4 example frame is wrapped in the correct ANSI SGR color; glyph characters are unchanged (`● ◌ ✗ – ✓`) and not emoji
- [ ] A-002 R2: The green/yellow/red/grey mapping is applied consistently across the example frame and both legends
- [ ] A-003 R3: Both legend lines name the color of each glyph
- [ ] A-004 R4: A graceful-degradation note is present in §4 describing ANSI wrap + bare-glyph fallback
- [ ] A-005 R5: The §6 autopilot reference keeps `●`/`◌` glyphs and is color-consistent with R2
- [ ] A-006 R6: `docs/specs/skills/SPEC-fab-operator.md` documents the colored health indicators

### Behavioral Correctness

- [ ] A-007 R1: Only the health glyph carries color — the autopilot `▶`, `[change]`/`[watch]` prefix, IDs, detail text, and `⚠` remain uncolored in the example frame

### Edge Cases & Error Handling

- [ ] A-008 R4: With zero color support the bare single-width BMP glyphs still render without width corruption and uniquely identify each state (terminal-safety guarantee from PR #333 preserved)

### Code Quality

- [ ] A-009 Pattern consistency: The ANSI escape representation (`\e[3Xm…\e[0m`) matches the existing repo idiom in `src/kit/skills/fab-status.md`; markdown style/idiom matches surrounding `fab-operator.md` content
- [ ] A-010 No unnecessary duplication: The color mapping is stated once authoritatively (legends + example) without redundant per-row restatement

### Documentation Accuracy

- [ ] A-011 R3: The legends, example frame, and degradation note are mutually consistent — no contradictory color/glyph claims

### Cross References

- [ ] A-012 R6: The skill source edit and the `SPEC-fab-operator.md` edit are consistent; the spec note accurately reflects the skill's colored health indicators

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Edit ONLY the `src/kit/` source; `.claude/skills/fab-operator.md` is a gitignored deployed copy regenerated by `fab sync`.

## Assumptions

<!-- Apply-time graded decisions. fab score reads intake.md only; this section is the
     apply-agent's record, not a scoring source. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use the literal `\e[3Xm…\e[0m` ANSI representation (matching the `\e[33m...\e[0m` idiom already in `src/kit/skills/fab-status.md`) rather than `\033[`/`\x1b[` | Established repo precedent found via grep; consistency over invention. | S:95 R:90 A:95 D:92 |
| 2 | Certain | `–` (paused) is left uncolored in the example frame (the `\e[90m…\e[0m` grey form is offered as an option in the legend only) | Intake says "none (or `\e[90m…\e[0m`)" for paused; default-none keeps the diff minimal and matches the original ⏸ neutral semantics. | S:90 R:85 A:90 D:88 |
| 3 | Confident | Satisfy the constitution skill→spec rule by adding a one-line colored-indicators note to `SPEC-fab-operator.md` §4 (Monitoring System), and leave `docs/specs/operator.md` untouched | `operator.md` is a command/version-history doc that never enumerated the health glyphs, so it documents nothing this change touches; `SPEC-fab-operator.md` is the per-skill spec the constitution names explicitly and its §4 already describes the monitoring tick. | S:80 R:88 A:85 D:80 |
| 4 | Confident | Run `fab sync` to deploy after editing the source (the committed artifact is the `src/kit/` source; the deployed copy is gitignored) | `fab sync` regenerates the gitignored deployed copy and is idempotent/safe; keeps the live skill in step with the source. If `fab sync` is unavailable or errors, it is noted and does not block — the committed change is the source. | S:78 R:90 A:82 D:85 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
