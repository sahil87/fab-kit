# Tasks: Operator 7 — Dependency-Aware Agent Spawning

**Change**: 260324-prtv-operator7-dep-aware-spawning
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Copy `fab/.kit/skills/fab-operator6.md` to `fab/.kit/skills/fab-operator7.md` — update the skill name/description in frontmatter to reference operator7, update the heading to `/fab-operator7`, update the launcher reference in §2 to `fab-operator7.sh`
- [x] T002 [P] Create `fab/.kit/scripts/fab-operator7.sh` — copy `fab-operator6.sh`, replace `/fab-operator6` with `/fab-operator7` in the `tmux new-window` command and the script name/description frontmatter

## Phase 2: Core Implementation

- [x] T003 Add `.fab-operator.yaml` schema additions to §4 in `fab/.kit/skills/fab-operator7.md` — add `depends_on` and `branch` fields to the monitored entry YAML example, add `branch_map` top-level section to the YAML example, add documentation text for all three new fields
- [x] T004 Add pre-spawn dependency resolution to §6 "Spawning an Agent" in `fab/.kit/skills/fab-operator7.md` — restructure spawn sequence to 4 steps (create worktree → resolve dependencies → open agent tab → enroll), document the full resolve dependencies procedure (branch lookup, redundant dep pruning, already-present check, cherry-pick command, conflict handling)
- [x] T005 Update "Working a Change" flows in §6 of `fab/.kit/skills/fab-operator7.md` — insert "Resolve dependencies" step 3 in the structured flow (backlog/Linear), add note for existing change flow
- [x] T006 Update autopilot dispatch sequence in §6 of `fab/.kit/skills/fab-operator7.md` — insert "Resolve dependencies" as step 2, renumber subsequent steps, add `--base` implies `depends_on` note
- [x] T007 Add dependency declaration documentation to §6 of `fab/.kit/skills/fab-operator7.md` — document three declaration paths (explicit conversational, autopilot queue, --base flag)

## Phase 3: Integration & Edge Cases

- [x] T008 Add cherry-pick conflict row to §3 bounded retries table in `fab/.kit/skills/fab-operator7.md` — `| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |`
- [x] T009 Add idle message timestamp to §4 in `fab/.kit/skills/fab-operator7.md` — update the between-tick idle message format to include `Time: HH:MM · next tick: HH:MM`
- [x] T010 Run `fab/.kit/scripts/fab-sync.sh` to deploy `fab-operator7.md` to `.claude/skills/`

---

## Execution Order

- T001 and T002 are independent (Phase 1 parallelizable)
- T003 through T007 all modify `fab-operator7.md` — execute sequentially (T003 first for schema context, then T004-T007)
- T008 and T009 modify `fab-operator7.md` — execute after T007
- T010 depends on all prior tasks
