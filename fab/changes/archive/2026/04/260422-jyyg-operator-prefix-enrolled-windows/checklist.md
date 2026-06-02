# Quality Checklist: Operator prefix on window enrollment

**Change**: 260422-jyyg-operator-prefix-enrolled-windows
**Generated**: 2026-04-22
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Rename on enrollment: §4 Enrollment bullet covers the `display-message '#W'` read, literal `»` prefix guard, `rename-window -t <pane> "»…"` call, and notes the rename "applies to every enrollment path" (covers direct send, user request, autopilot, watch spawn).
- [x] CHK-002 Idempotent prefix guard: Skill explicitly says "prefix `»` (U+00BB)" and "literal `»` prefix check — windows that already carry it … get no rename." All 10 `»` in the skill file verified as U+00BB via python codepoint check.
- [x] CHK-003 No rename on removal: §4 Removal bullet now ends with "The window name is **not** restored on removal — the `»` prefix persists. Users who want it removed rename the window manually (`Ctrl-b ,`)."
- [x] CHK-004 Rename failure resilience: Skill states "If `tmux rename-window` fails … the operator logs one line and continues — enrollment itself is already durable" with the exact log format `{change}: window rename skipped ({error}).`
- [x] CHK-005 Spawn path compatibility: §6 step 3 (`tmux new-window -n "»<wt>" …`) is unchanged; step 4 now carries the parenthetical cross-reference to §4's rename rule.
- [x] CHK-006 Skill file documents the new behavior: §4 Enrollment (new paragraph), §4 Removal (additive sentence), §6 step 4 (parenthetical) all updated. Diff inspection confirms additive-only — no other lines in §4/§6 altered aside from adding "(including autopilot and watch spawns)" in the Enrollment trigger sentence, which the spec permits as part of "all enrollment paths" coverage.
- [x] CHK-007 Spec file gets additive bullet: SPEC-fab-operator.md line 22 is the only changed line; the new content is appended to item 4 of the Section Structure list. Stale `fab-operator4` references remain untouched.

## Behavioral Correctness

- [x] CHK-008 Rename timing order: Skill opens the new paragraph with "After writing the monitored entry to `.fab-operator.yaml`, the operator MUST prefix `»` …" — matches Design Decision 3.
- [x] CHK-009 Scope of rename: Skill uses `#W` (window name) in the `display-message` command and `rename-window` (window scope). Pane/session naming is not mentioned, consistent with Design Decision 1.

## Scenario Coverage

- [x] CHK-010 "New enrollment, plain name": A reader can trace `display-message` → guard-miss → `rename-window -t <pane> "»${name}"` yielding `»dev-tab`.
- [x] CHK-011 "Enrollment of operator-spawned window": Explicitly listed — "windows that already carry it (operator-spawned windows from §6, `/clear`-restored entries, re-enrolled changes) get no rename."
- [x] CHK-012 "/clear recovery over prefixed windows": Same sentence above explicitly names `/clear`-restored entries as a guard-hit case.
- [x] CHK-013 "Pane vanishes between refresh and rename": Explicitly covered — "e.g., the pane vanished between refresh and the rename call" with log line documented.

## Edge Cases & Error Handling

- [x] CHK-014 Partial-failure ordering: Rename occurs only after YAML write ("After writing the monitored entry …"); rename failure leaves enrollment in place and emits the skipped-rename line.
- [x] CHK-015 Re-enrollment after transient removal: "re-enrolled changes" is explicitly listed as a guard-hit case in the skill prose.

## Code Quality

- [x] CHK-016 Readability and maintainability: Prose is concise, one `sh` block for the command and one code block for the log line — reader can copy-paste the exact commands.
- [x] CHK-017 Follow existing project patterns: New paragraph sits between **Enrollment** and **Removal** bullets, uses the same bolded-lead + prose style. Code fences match §6's `tmux new-window` style. `sh` fence and single-line log format match §4 "plain lines below the frame" convention.
- [x] CHK-018 No unnecessary duplication: Spec file's new clause is a one-liner ("idempotent — skipped if already prefixed. Removal does not restore the original name."), not a copy of the skill's algorithm. No duplication.

## Documentation Accuracy

- [x] CHK-019 Canonical source respected: Only `src/kit/skills/fab-operator.md` and `docs/specs/skills/SPEC-fab-operator.md` modified (git diff + `.claude/skills/fab-operator.md` does not exist locally — gitignored deployed copy untouched).
- [x] CHK-020 Memory file flagged for hydrate: Tasks and spec explicitly call out `docs/memory/fab-workflow/execution-skills.md` (lines ~260 and ~563) as the hydrate-stage target; no edit expected at apply.
- [x] CHK-021 Character fidelity: Python codepoint check confirms 10 U+00BB in the skill file and 1 U+00BB in the spec file — zero non-U+00BB angle-like glyphs in either file.

## Cross-References

- [x] CHK-022 Skill ↔ spec consistency: Both agree on (a) literal `»` prefix, (b) idempotent skip, (c) no restore on removal. Spec bullet is a pointer; skill is authoritative.
- [x] CHK-023 Spec ↔ constitution: Constitution "Additional Constraints" satisfied — skill edit accompanied by SPEC-fab-operator.md edit.
- [x] CHK-024 §4 → §6 cross-reference: §6 step 4's parenthetical — "(Enrollment applies the §4 window-rename rule; the `»<wt>` name produced in step 3 already satisfies the idempotent prefix guard, so no duplicate rename occurs.)" — explicitly names §4 as the canonical definition.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
