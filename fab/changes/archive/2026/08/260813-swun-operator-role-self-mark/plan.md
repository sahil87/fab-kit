# Plan: Operator Role Self-Mark

**Change**: 260813-swun-operator-role-self-mark
**Intake**: `intake.md`

## Requirements

### Operator Skill: Startup Role Self-Mark

#### R1: Fail-Silent Role Mark at Startup
`src/kit/skills/fab-operator.md` §2 Startup SHALL gain a `### Role Mark` subsection, placed after `### Tmux Gate` and before `### wt Gate`, instructing the operator to mark its tmux window via run-kit's `@rk_role` window option with the exact command:

```bash
command -v rk >/dev/null 2>&1 && rk role operator >/dev/null 2>&1 || true
```

The step MUST be fail-silent in both absence and version-skew cases (the installed run-kit may predate the `role` subcommand — probed true on 2026-08-13), MUST NOT block or noise startup, and MUST state the ownership split: rk owns the option contract, pinned rendering, and one-operator-per-server radio semantics; fab is only the producer. No unmark step and no radio-conflict handling are added.

- **GIVEN** an operator starting inside tmux with run-kit installed and current
- **WHEN** the startup sequence passes the Tmux Gate
- **THEN** `rk role operator` marks the window, and startup proceeds to the wt Gate

- **GIVEN** run-kit absent, or installed but predating the `role` subcommand
- **WHEN** the Role Mark step runs
- **THEN** it degrades to a silent no-op — no error output, no startup interruption

#### R2: `_cli-external.md` § rk Enumeration Accuracy
`src/kit/skills/_cli-external.md` SHALL acknowledge the role self-mark as the second fab-owned rk usage via a **pointer** to `fab-operator.md` §2 Startup — never a restatement of the command (owner-or-pointer rule, code-quality.md). The enumerations of fab-owned rk content SHALL be updated consistently: the frontmatter `description:` ("the escalation rk-notify usage"), the § Reference Model intro list, and the § rk (run-kit) section body.

- **GIVEN** a reader of `_cli-external.md` § rk (run-kit)
- **WHEN** they enumerate fab-owned rk usages
- **THEN** they find both the escalation `rk notify` send (owned in place) and a pointer naming the startup role self-mark as owned by `fab-operator.md` §2

#### R3: No Stale Sole-Usage Claims
A repo-wide sweep SHALL confirm no document still claims the escalation `rk notify` send is the *only* fab-owned rk usage, and the aggregate specs (`docs/specs/skills.md` § `/fab-operator`, `glossary.md`, `architecture.md`) remain consistent. The skills.md Flow skeleton does not enumerate startup gates (verified at plan time), so it likely needs no edit — the task verifies rather than assumes.

- **GIVEN** the two skill edits are complete
- **WHEN** grepping for phrases enumerating fab-owned rk usage (e.g. "escalation `rk notify` usage", "rk-notify usage")
- **THEN** every occurrence either includes the role self-mark or is a context where the enumeration is not exhaustive-by-claim

### Non-Goals

- No change to the `fab operator` Go launcher, no new `fab` subcommand, no tests (markdown-only change), no migration
- Nothing on the run-kit side — the `@rk_role` option contract, pinned rendering, and radio semantics ship in run-kit separately
- No unmark/cleanup on operator exit — staleness is rk's concern

### Design Decisions

#### Skill-Side Producer, Not Launcher-Side
**Decision**: The mark is produced by the skill's startup step, not by the `fab operator` Go launcher.
**Why**: The skill runs in the window regardless of how it was launched (via `fab operator` or manually), so the self-mark covers both paths; it also keeps the change Pure Prompt Play (Constitution I) with zero Go surface.
**Rejected**: Marking from the Go launcher — misses manually-started operators and adds a binary change for a one-line producer.
*Introduced by*: 260813-swun-operator-role-self-mark

#### Full Suppression Beyond the command -v Gate
**Decision**: The `rk role` call carries `>/dev/null 2>&1 || true` in addition to the `command -v rk` gate.
**Why**: Probed 2026-08-13 — the installed run-kit errors with `unknown command "role"`; the `_preamble.md` fail-silent rule must extend to version skew or every startup on a current rk prints an error.
**Rejected**: Bare `command -v rk && rk role operator` — fails loudly on any rk predating the companion feature.
*Introduced by*: 260813-swun-operator-role-self-mark

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `### Role Mark` subsection to `src/kit/skills/fab-operator.md` §2 Startup, between `### Tmux Gate` and `### wt Gate`: the fail-silent command, the rk-owns-contract/fab-is-producer split, idempotency, and the version-skew rationale <!-- R1 -->
- [x] T002 Update `src/kit/skills/_cli-external.md`: add a pointer line in § rk (run-kit) naming the startup role self-mark as fab-owned usage owned by `fab-operator.md` §2; sweep the frontmatter `description:` and § Reference Model intro enumerations <!-- R2 -->

### Phase 2: Integration & Edge Cases

- [x] T003 Repo-wide sweep: grep for fab-owned-rk-usage enumerations and `rk role`/`@rk_role` mentions across `src/kit/` and `docs/`; verify `docs/specs/skills.md` § `/fab-operator`, `glossary.md`, and `architecture.md` need no edit (or fix them) <!-- R3 --> *(sweep also fixed `_preamble.md` § Run-Kit's singular "the fab-owned rk usage" claim; aggregate specs verified clean)*

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §2 Startup contains a `### Role Mark` subsection after the Tmux Gate and before the wt Gate, carrying the exact fail-silent command with both the `command -v` gate and full output/exit suppression
- [x] A-002 R2: `_cli-external.md` § rk names the role self-mark as the second fab-owned usage via a pointer to `fab-operator.md` §2 — the command itself is not restated there

### Behavioral Correctness

- [x] A-003 R1: The Role Mark prose states the ownership split (rk owns option contract, rendering, radio; fab produces) and adds no unmark step or radio-conflict handling
- [x] A-004 R2: The `_cli-external.md` frontmatter `description:` and § Reference Model intro enumerations both reflect the second fab-owned usage consistently

### Scenario Coverage

- [x] A-005 R1: The version-skew scenario is documented in the step (an installed rk predating `role` degrades to a silent no-op)

### Edge Cases & Error Handling

- [x] A-006 R3: Repo-wide grep finds no remaining claim that the escalation `rk notify` send is the sole fab-owned rk usage; aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) are consistent with the new step or verified as not enumerating startup steps

### Code Quality

- [x] A-007 Pattern consistency: The new subsection matches the surrounding Startup gates' structure and tone (short heading, imperative prose, fenced command block)
- [x] A-008 Owner-or-pointer: No file both states the Role Mark command and points at its owner; `fab-operator.md` §2 is the single owner
- [x] A-009 Canonical source only: All edits land in `src/kit/skills/` — nothing under `.claude/skills/` is touched

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new documentation without making existing content redundant (the only rewording, `_preamble.md` § Run-Kit's singular "fab-owned rk usage" sentence, was updated in place by the apply sweep)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Subsection is titled `### Role Mark` and sits between Tmux Gate and wt Gate | Matches the gate-style subsection naming in §2 Startup; trivially renameable | S:70 R:95 A:80 D:75 |
| 2 | Confident | `docs/specs/skills.md` § `/fab-operator` likely needs no edit | Its Flow skeleton shows the loop, not startup gates (verified at plan time); T003 confirms rather than assumes | S:75 R:90 A:80 D:70 |

2 assumptions (0 certain, 2 confident, 0 tentative).
