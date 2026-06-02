# Tasks: Operator Numbered-Menu Classification + Idle-Escalation Auto-Default

**Change**: 260422-hin2-operator-strategic-menu-escalation
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

<!-- This is a skill-doc-only change — no code scaffolding, dependencies, or configuration needed. -->

- [x] T001 Read the current text of `src/kit/skills/fab-operator.md` §5 Answer Model (rule 4) and §5 Logging; record the exact pre-change wording in a scratch note for the apply step so replacements are surgical and leave surrounding rules untouched.
- [x] T002 [P] Read the current text of `docs/specs/skills/SPEC-fab-operator.md` and identify the exact section(s) that describe §5 Answer Model behavior today; record section headings and anchor text so the mirror edits are targeted.

## Phase 2: Core Implementation

<!-- Edits to canonical skill source and its mirror spec. Ordered by dependency: skill source first, then mirror, then supporting edits. -->

- [x] T003 Replace the current rule 4 text in `src/kit/skills/fab-operator.md` §5 Answer Model with the classification-aware variant. The new text MUST: (a) name Routine and Strategic as the two classes; (b) list all four classifier signals (option text length, semantic distinctness of options, surrounding agent context, reversibility of the choice); (c) state the "no hardcoded keyword list" constraint; (d) state the escalate-on-uncertainty rule. Do NOT include a "~30 chars per option" number — option length stays qualitative (spec assumption #14). Preserve rule numbering; rules 1-3 and 5-6 are unchanged.
- [x] T004 Append the auto-default logging bullet to `src/kit/skills/fab-operator.md` §5 Logging — format exactly: `- Auto-default (after 30m idle on strategic escalation): `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"``. The existing two bullets (`auto-answered`, `can't determine`) remain untouched.
- [x] T005 Add an "Idle Auto-Default on Strategic Escalations" subsection to `src/kit/skills/fab-operator.md` §5 after the Answer Model and before/within the Logging subsection as appropriate to the file's current structure. The subsection MUST specify: 30-minute hardcoded threshold; idle clock resets on any terminal-state change (new content OR user keystrokes); auto-default answer selection priority (stated default in prompt → `1`); rule-6 exclusion; no `.fab-operator.yaml` schema change.
- [x] T006 Mirror the behavior changes from T003-T005 into `docs/specs/skills/SPEC-fab-operator.md`. Update the section(s) identified in T002 to document both classification (Routine/Strategic with all four signals) and the idle auto-default (threshold, reset rule, answer priority, rule-6 exclusion, distinct log format). Style should match the existing SPEC-fab-operator.md conventions identified in T002.

## Phase 3: Integration & Edge Cases

<!-- Verify the canonical source / deployed copy relationship, and confirm no unintended surface area was touched. -->

- [x] T007 Run `fab sync` to deploy the updated `src/kit/skills/fab-operator.md` to `.claude/skills/fab-operator/SKILL.md`. Verify via diff that the deployed copy reflects the new rule 4 text and the new logging bullet. Per constitution, the canonical source is `src/kit/` — never edit the deployed copy directly.
- [x] T008 [P] Verify `.fab-operator.yaml` schema documentation in `src/kit/skills/fab-operator.md` §4 is unchanged (spec assumption #12): grep or diff §4 before/after; no threshold field should have been introduced.
- [x] T009 [P] Verify no CLI / Go binary files were touched (spec assumption #13): confirm `git status` shows changes only under `src/kit/skills/`, `docs/specs/skills/`, `.claude/skills/` (sync output), and the change folder — no edits to `src/go/` or `src/kit/lib/`.

## Phase 4: Polish

<!-- Keep to essentials. Hydrate stage handles memory file edits and backlog checkbox flip; tasks cover only the apply-phase deliverables. -->

- [x] T010 Cross-read the revised `src/kit/skills/fab-operator.md` and `docs/specs/skills/SPEC-fab-operator.md` for internal consistency — same four signals named, same threshold value, same log format, same rule-6 carve-out wording. Fix any drift between the two.

---

## Execution Order

- T001 and T002 are parallelizable — both are read-only captures.
- T003 blocks T004 and T005 (all edit the same file; sequential for clean diffs).
- T005 depends on T003 conceptually (references the classification system the subsection relies on).
- T006 depends on T003–T005 (mirror must reflect all source changes).
- T007 depends on T003–T005 (sync deploys what's been written to src/kit/).
- T008 and T009 are parallelizable post-T007 verifications.
- T010 depends on T006 (final cross-read after both files are authored).

Hydrate-stage tasks (not listed here): update `docs/memory/fab-workflow/execution-skills.md` per hydrate's diff scan; flip `[hin2]` and `[i1l6]` to `[x]` in `fab/backlog.md`. These are handled by /fab-continue Hydrate Behavior, not by the apply phase.
