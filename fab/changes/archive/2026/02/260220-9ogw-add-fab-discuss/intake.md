# Intake: Add fab-discuss Skill

**Change**: 260220-9ogw-add-fab-discuss
**Created**: 2026-02-20
**Status**: Draft

## Origin

> User conversation, 2026-02-20:
>
> "I want a skill that I can load that primes the agent with the knowledge of the repo. Something like fab-discuss."
>
> Discussion concluded:
> - Minimum info needed: location of docs/memory and docs/specs
> - Option A (just the standard always-load 7 files) is correct — the paths are fixed constants across every fab-kit repo
> - Adding overview.md etc. would be repo-specific content, not a generic skill
> - The always-load indexes already give the agent the map to navigate anywhere on demand
> - Value of the skill: triggers context loading explicitly, presents an orientation summary, signals "discussion mode" — no artifact generation, no stage advancement
> - Works without an active change (unlike most fab-* skills)

## Why

Starting a discussion session in a fab-kit repo currently requires either (a) invoking a workflow skill which loads context as a side effect of doing something, or (b) manually telling the agent what to read. Neither is a good fit for exploratory conversations about the project ("how does X work?", "I'm thinking about Z, help me think through it").

Without `fab-discuss`, an agent entering a discussion session has no guaranteed awareness of:
- Where memory files live or what domains exist
- Where specs live or what's been designed
- Project configuration, constraints, or constitution

The always-load layer (`_context.md` Layer 1) already defines exactly the right 7 files for baseline orientation. They're fixed-path constants across every fab-kit repo. `fab-discuss` makes that layer the explicit entry point for discussion, rather than a side effect of a workflow skill.

If we don't add this: users have to manually prime agents, risk inconsistent context loading, and have no clean way to signal "I want to talk, not execute a pipeline stage."

## What Changes

### New skill: `fab/.kit/skills/fab-discuss.md`

A new skill that:
1. Loads the standard always-load layer (same 7 files as every other skill, per `_context.md`)
2. Prints an orientation summary: what was loaded, what memory domains exist, what specs exist
3. Checks for an active change (reads `fab/current` if it exists) and mentions its name/stage without deep-loading it
4. Signals discussion mode — ends with "Ready to discuss. What would you like to explore?" rather than a `Next:` pipeline command
5. **Does not** require an active change, does not run preflight, does not advance any stage
6. **Does not** modify any files — fully read-only

Skill frontmatter:
```yaml
---
name: fab-discuss
description: "Prime the agent with project context for a discussion session — loads the always-load layer and orients to the repo landscape."
model_tier: capable
---
```

The skill body loads `_context.md` for the always-load list, then explicitly reads the 7 files (gracefully skipping the 3 optional ones if missing), and presents a structured orientation.

### Update `fab/.kit/scripts/fab-help.sh`

Add `fab-discuss` to the `skill_to_group` mapping under `"Start & Navigate"`:
```bash
[fab-discuss]="Start & Navigate"
```

The help script auto-discovers skill descriptions from frontmatter, so no description text changes needed — just the group assignment.

### Update `docs/specs/skills.md`

Add a `## /fab-discuss` section documenting:
- Purpose: prime agent with project context for discussion
- Context: same as always-load (the full 7-file layer)
- Key properties: no active change required, read-only, idempotent
- What it outputs: orientation summary + ready signal

Also update the **Context Loading Convention** section at the top — it currently only lists 3 files in the always-load list (it's stale vs. the 7 files in `_context.md`). Fix it as part of this change.

### Update `docs/specs/overview.md`

Add `/fab-discuss` row to the Quick Reference table:

```markdown
| `/fab-discuss` | Prime agent with project context for discussion | — (read-only) |
```

### Update `docs/specs/user-flow.md`

Add `/fab-discuss` to the Setup & Utilities diagram (Section 3A, "Utility (anytime)" subgraph):
```
DISCUSS["/fab-discuss"] ~~~ STATUS["/fab-status"] ~~~ HELP["/fab-help"]
```

### Update `docs/memory/fab-workflow/context-loading.md`

Add a note in the **Exception Skills** section clarifying `fab-discuss`'s relationship to the always-load layer: it is the *only* skill whose entire purpose is to surface the always-load layer — it loads all 7 files and presents them as its output, rather than as a preamble to something else. Also add a changelog entry.

## Affected Memory

- `fab-workflow/context-loading`: (modify) Document `fab-discuss` skill — clarify it as the only skill whose primary output IS the always-load layer (vs. loading it as preamble)

## Impact

- `fab/.kit/skills/fab-discuss.md` — new file
- `fab/.kit/scripts/fab-help.sh` — one-line addition to group mapping
- `docs/specs/skills.md` — new section + stale context loading section fix
- `docs/specs/overview.md` — one-row addition to Quick Reference
- `docs/specs/user-flow.md` — one node addition to diagram 3A
- `docs/memory/fab-workflow/context-loading.md` — paragraph + changelog entry

No changes to `_context.md` (the exception list does not change — `fab-discuss` loads the full always-load layer). No changes to templates, configuration, or scripts beyond `fab-help.sh`.

## Open Questions

- Should `fab-discuss` mention the active change's name/stage in its output, or omit it entirely to keep the skill simpler? (Current assumption: mention it lightly — "Active change: X (stage: intake)" — as useful orientation, without deep-loading the change.)
- Should `fab-discuss` be listed as a `Next:` suggestion anywhere in the state table? (Current assumption: no — it's a session-entry point, not a pipeline step. The state table covers the pipeline only.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Load exactly the standard 7-file always-load layer | Explicitly agreed in conversation — the always-load paths are fixed constants across all fab-kit repos | S:95 R:90 A:95 D:95 |
| 2 | Certain | Do not require an active change | Explicitly agreed — discussion sessions don't need a change in flight | S:90 R:90 A:95 D:90 |
| 3 | Confident | `model_tier: capable` | Discussion requires reasoning and synthesis, not just script delegation | S:75 R:85 A:80 D:80 |
| 4 | Confident | Add to "Start & Navigate" group in fab-help.sh | Closest semantic fit — it's a session-entry/orientation skill alongside fab-status | S:70 R:90 A:80 D:75 |
| 5 | Confident | Fix the stale context-loading section in docs/specs/skills.md as part of this change | It's already wrong (3 files listed vs. 7 actual); fixing it is low-risk and keeps docs accurate | S:80 R:85 A:85 D:80 |
| 6 | Tentative | Mention active change name/stage in output (light touch) | Useful orientation, but could be omitted for simplicity. User didn't specify either way. | S:50 R:80 A:65 D:55 |
| 7 | Tentative | Do not add fab-discuss to the state table Next: suggestions | Feels wrong in a pipeline context, but could be useful at "initialized" state | S:55 R:75 A:70 D:60 |

7 assumptions (2 certain, 3 confident, 2 tentative, 0 unresolved). Run /fab-clarify to review.
