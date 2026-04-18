# Tasks: Add _cli-rk Skill

**Change**: 260416-mgsm-add-cli-rk-skill
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Create `src/kit/skills/_cli-rk.md` with frontmatter and content: detection (`command -v rk`), iframe windows (tmux `@rk_type`/`@rk_url`), proxy URL pattern (`/proxy/{port}/...`), server URL discovery (`rk context`), visual display recipe (4-step centralized pattern), visual-explainer integration note
- [x] T002 Update `src/kit/skills/_preamble.md` — add `_cli-rk` entry to Section 1 (Always Load), after `_naming`, as an optional always-load skill

## Phase 2: Specs & Docs

- [x] T003 [P] Create `docs/specs/skills/SPEC-_cli-rk.md` — summary, flow, tools used, sub-agents (following existing pattern from `SPEC-fab-discuss.md`)
- [x] T004 [P] **N/A** — `docs/specs/skills.md` does not maintain a list of internal/underscore skills

---

## Execution Order

- T001 blocks T002 (preamble references the file T001 creates)
- T003 and T004 are independent of each other and can run after T001-T002
