# Tasks: Operator prefix on window enrollment

**Change**: 260422-jyyg-operator-prefix-enrolled-windows
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  Pure documentation change against two markdown files (skill + spec).
  No source-code, test, build, or config modifications.
  No scaffolding/setup needed — Phase 1 is skipped.
-->

## Phase 2: Core Implementation

- [x] T001 Update `src/kit/skills/fab-operator.md` §4 Monitored Set **Enrollment** bullet to describe the window rename on enrollment. Add one paragraph covering: (a) after writing the monitored entry to `.fab-operator.yaml`, the operator reads the window name via `tmux display-message -p -t <pane> '#W'`; (b) if it does not already start with `»`, the operator runs `tmux rename-window -t <pane> "»<current-name>"`; (c) the guard is a literal `»` (U+00BB) prefix check; (d) rename failure logs a single line `"{change}: window rename skipped ({error})."` and continues — enrollment is not rolled back. Cover all enrollment paths (direct command dispatch, user-requested monitoring, autopilot spawn, watch spawn) in one rule. Do not reorder or remove surrounding content. (Spec: fab-operator → Rename on enrollment, Idempotent prefix guard, Rename failure resilience)

- [x] T002 Update `src/kit/skills/fab-operator.md` §4 Monitored Set **Removal** bullet to add the explicit non-behavior: "The window name is **not** restored on removal — the `»` prefix persists. Users who want it removed rename manually." No other edits to the Removal bullet. (Spec: fab-operator → No rename on removal)

- [x] T003 Update `src/kit/skills/fab-operator.md` §6 Spawning an Agent, step 4 (enrollment), to cross-reference the new §4 rename behavior. Add a short parenthetical: "(enrollment applies the §4 window-rename rule; the `»<wt>` name produced in step 3 already satisfies the idempotent prefix guard, so no duplicate rename occurs)". Do not restructure step 3 or the surrounding list. (Spec: fab-operator → Spawn path compatibility)

- [x] T004 [P] Update `docs/specs/skills/SPEC-fab-operator.md` with one additive bullet under the **Monitoring System** section (item 4 of the "Section Structure" list). Wording: `- Window-name rename on enrollment: prefix `»` to the tmux window name (idempotent — skipped if already prefixed). Removal does not restore the original name.` Do not rewrite, reorder, or delete any existing content — including the stale `fab-operator4` references. (Spec: Documentation → Spec file gets an additive bullet)

## Phase 3: Integration & Edge Cases

- [x] T005 Cross-read the two updated files to verify consistency: the skill's §4 rename paragraph and the spec's new bullet MUST agree on (a) the U+00BB prefix character, (b) idempotent guard semantics, (c) no rename on removal. No duplicated full requirement text — spec bullet is a one-line pointer. If a drift is found, fix in the skill file (canonical per constitution §V Additional). (Spec: Documentation → Skill file documents the new behavior, Spec file gets an additive bullet)

- [x] T006 Verify the idempotent-guard example in the skill file uses the literal `»` character (U+00BB), not a similar-looking glyph (e.g., `>>`, `»»`, `»` escaped). Reading via `Read` should show the exact character in the markdown source. (Spec: fab-operator → Idempotent prefix guard, Design Decision 2)

## Phase 4: Polish

*(Omitted — pure doc change, no cleanup warranted beyond the consistency check in T005.)*

---

## Execution Order

- T001, T002, T003 are sequential (all touch `src/kit/skills/fab-operator.md` — same file, no parallel edits)
- T004 is independent of T001–T003 (different file) and marked `[P]`
- T005 and T006 run after T001–T004 complete
