# Quality Checklist: Operator Numbered-Menu Classification + Idle-Escalation Auto-Default

**Change**: 260422-hin2-operator-strategic-menu-escalation
**Generated**: 2026-04-22
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Classify before answering: `src/kit/skills/fab-operator.md` §5 Answer Model rule 4 names Routine and Strategic classes, lists all four signals (option text length, semantic distinctness, surrounding agent context, reversibility), and prohibits a hardcoded keyword list.
- [ ] CHK-002 Escalate on classification uncertainty: rule 4 text states that uncertainty MUST be treated as Strategic and escalate.
- [ ] CHK-003 Revised rule 4 text: the new rule 4 text in `src/kit/skills/fab-operator.md` matches the canonical wording from spec §"Revised rule 4 text" requirement — Routine and Strategic classes, four signals enumerated, no keyword list, uncertainty escalates.
- [ ] CHK-004 30-minute idle threshold: `src/kit/skills/fab-operator.md` §5 Idle Auto-Default subsection states the hardcoded 30-minute threshold and explicitly rules out `.fab-operator.yaml`, per-change override, and environment-variable configurability.
- [ ] CHK-005 Idle clock reset rule: subsection states that the idle timer resets on any terminal-state change — new pane content OR user keystrokes.
- [ ] CHK-006 Auto-default answer selection: subsection states the priority — stated default in prompt (e.g., `(default: 2)`, `Press enter for 2`, `[2]`) → otherwise option `1`.
- [ ] CHK-007 Rule-6 exclusion: subsection explicitly carves out rule-6 "cannot determine keystrokes" escalations from the auto-default.
- [ ] CHK-008 Distinct log line format: §5 Logging includes the bullet with exact format `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`, distinct from `auto-answered`.
- [ ] CHK-009 SPEC mirror: `docs/specs/skills/SPEC-fab-operator.md` documents both classification (Routine/Strategic with four signals) and the idle auto-default (threshold, reset rule, answer priority, rule-6 exclusion, distinct log format).
- [ ] CHK-010 Memory hydrate: hydrate stage updates `docs/memory/fab-workflow/execution-skills.md` (or target(s) identified by hydrate's diff scan) to reflect classification and auto-default behavior.
- [ ] CHK-011 Backlog cleanup: `fab/backlog.md` has `[hin2]` and `[i1l6]` transitioned from `[ ]` to `[x]` at hydrate, entry text unchanged.

## Behavioral Correctness

- [ ] CHK-012 Rule 4 behavior change: a reader of §5 Answer Model can tell that rule 4's behavior changed from "always answer 1" to "classify then route" — not merely that classification was added as flavor text.
- [ ] CHK-013 Idle subsection is additive: existing §5 rules 1-3, 5, 6 and existing Logging bullets (`auto-answered`, `can't determine`) are unchanged.

## Scenario Coverage

- [ ] CHK-014 Routine tool-permission prompt scenario from spec is traceable to rule 4 text behavior.
- [ ] CHK-015 Strategic rework menu scenario from spec is traceable to rule 4 text behavior.
- [ ] CHK-016 "Prompt states a default" scenario is traceable to the answer-selection priority in the Idle subsection.
- [ ] CHK-017 "User types partial keystrokes" / "Agent emits output mid-wait" reset scenarios are traceable to the reset-rule wording in the Idle subsection.
- [ ] CHK-018 "Rule-6 escalation does not auto-default" scenario is traceable to the rule-6 carve-out wording.

## Edge Cases & Error Handling

- [ ] CHK-019 Ambiguous classification handling: the revised rule 4 text makes explicit that borderline prompts escalate rather than auto-answer.
- [ ] CHK-020 Option-length heuristic is qualitative (spec assumption #14): no "~30 chars" or equivalent numeric threshold appears in the shipped skill text.

## Code Quality

- [ ] CHK-021 Pattern consistency: new rule 4 text and Idle subsection follow the markdown structure/voice of surrounding §5 content in `src/kit/skills/fab-operator.md`.
- [ ] CHK-022 No unnecessary duplication: the classification description does not repeat §5 Question Detection content — it references it.
- [ ] CHK-023 God-function anti-pattern N/A: skill-doc-only change, no executable code.
- [ ] CHK-024 Magic numbers: the 30-minute threshold is stated in prose with a clear rationale reference; not referenced as an unlabeled literal.
- [ ] CHK-025 Duplication avoidance across files: SPEC-fab-operator.md mirror and `src/kit/skills/fab-operator.md` describe the same four signals, same threshold, same log format, same rule-6 carve-out — no drift (tasks T010 verifies).

## Documentation Accuracy

<!-- From config.yaml checklist.extra_categories -->

- [ ] CHK-026 Canonical source vs deployed copy: only `src/kit/skills/fab-operator.md` is edited; any changes to `.claude/skills/fab-operator/SKILL.md` come from `fab sync`, not direct edits.
- [ ] CHK-027 SPEC-fab-operator.md mirror is current: every behavior added to the skill file is also documented in the SPEC file (constitution §Additional Constraints).
- [ ] CHK-028 Log format string is byte-exact: `{change}: auto-defaulted after 30m idle: '{summary}' → {answer}` — including the Unicode `→` character (U+2192), single quotes around summary, and literal `30m`.

## Cross-References

<!-- From config.yaml checklist.extra_categories -->

- [ ] CHK-029 Rule 4 revision references the four classifier signals in the same order documented in spec §"Classification mechanics" — option text length, semantic distinctness, surrounding agent context, reversibility.
- [ ] CHK-030 Rule-6 carve-out uses the same phrasing or an exact paraphrase of spec's "cannot determine keystrokes" language so grep-based cross-checks succeed.
- [ ] CHK-031 Memory hydrate target alignment: if hydrate's diff scan selects a memory file other than `fab-workflow/execution-skills.md`, both the spec's Affected Memory metadata and the hydrate commit explain the redirect.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-XXX **N/A**: {reason}`
