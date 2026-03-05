# Tasks: Copy-Template Skills, Drop Agents

**Change**: 260219-d2y2-copy-template-skills-drop-agents
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Move `claude_fast_model` resolution before sync calls in `fab/.kit/sync/2-sync-workspace.sh` — extract the model resolution logic from Section 4 (lines 391-400) and place it after skill classification (Section 3b) but before the `sync_agent_skills` calls. Keep the same `yaml_value` lookup + `haiku` fallback pattern. Guard with `if [ ${#fast_skills[@]} -gt 0 ]` so it only runs when fast-tier skills exist.

## Phase 2: Core Implementation

- [x] T002 Enhance `sync_agent_skills` copy path to support sed templating in `fab/.kit/sync/2-sync-workspace.sh` — modify the copy branch to accept an optional sed expression via the 5th parameter (currently `rel_prefix`, unused in copy mode). When provided in copy mode: apply the sed expression to produce expected content, compare against existing dest for idempotency, and write the templated result. Update the function comment to document dual-purpose 5th parameter.
- [x] T003 Change Claude Code sync call from symlink to copy-with-template in `fab/.kit/sync/2-sync-workspace.sh` — update line 375 from `sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "symlink" "../../../"` to `sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy" "s/^model_tier: .*/model: $claude_fast_model/"`. This passes the sed expression that replaces `model_tier: fast` with `model: haiku` (or configured model).
- [x] T004 Delete Section 4 (agent file generation) from `fab/.kit/sync/2-sync-workspace.sh` — remove the entire block from `# ── 4. Model tier agent files` through the closing `fi` (lines ~386-458). This removes the generation loop, stale agent cleanup, and the "Agents:" output line.
- [x] T005 Add transitional agent cleanup in `fab/.kit/sync/2-sync-workspace.sh` — after the `clean_stale_skills` calls and where Section 4 was, add a cleanup block: iterate `.claude/agents/*.md`, check if basename (without `.md`) matches any entry in `skills[]`, remove matches, count and report removed files. Skip if `.claude/agents/` doesn't exist. Preserve non-matching files (user-created agents).

## Phase 3: Integration & Edge Cases

- [x] T006 Run `fab/.kit/sync/2-sync-workspace.sh` end-to-end and verify: (a) `.claude/skills/*/SKILL.md` are regular files, not symlinks; (b) fast-tier skills contain `model: haiku` instead of `model_tier: fast`; (c) capable-tier skills are exact copies; (d) no `.claude/agents/fab-*.md` files exist; (e) re-run shows all skills as "already valid" (idempotency)

---

## Execution Order

- T001 blocks T003 (model variable must be available before the sync call uses it)
- T002 blocks T003 (function must support sed before the call passes one)
- T004 is independent of T002/T003 but should follow T003 (remove old code after new code works)
- T005 depends on T004 (replaces Section 4's position)
- T006 depends on all prior tasks
