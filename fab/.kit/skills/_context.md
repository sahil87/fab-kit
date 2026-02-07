# Shared Context Preamble

> This file defines shared conventions for all Fab skills. Each skill file should begin with:
> `Read and follow the instructions in fab/.kit/skills/_context.md before proceeding.`

---

## Context Loading

Before generating or validating any artifact, load the relevant context layers below. This ensures output is grounded in the actual project state, not assumptions.

### 1. Always Load (every skill except `/fab:init`, `/fab:switch`, `/fab:status`, `/fab:hydrate`)

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/config.yaml`** — project configuration, tech stack, naming conventions, stage configuration
- **`fab/constitution.md`** — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- **`fab/docs/index.md`** — documentation landscape (which domains and docs exist)

> **Note**: If the skill runs `fab-preflight.sh` (Section 2 above), the init check (config.yaml and constitution.md existence) is already covered by the script. Skills using preflight don't need separate existence checks for these files — they only need to read them for content.

### 2. Change Context (when operating on an active change)

Resolve the active change and load its state by running the preflight script:

1. **Run preflight**: Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. **Check exit code**: If the script exits non-zero, STOP and surface the stderr message to the user (it contains the specific error and suggested fix)
3. **Parse stdout YAML**: On success, parse the YAML output for `name`, `change_dir`, `stage`, `branch`, `progress`, and `checklist` fields — use these for all subsequent change context instead of re-reading `.status.yaml`
4. Load all completed artifacts in the change folder (e.g., `proposal.md`, `spec.md`, `plan.md`, `tasks.md`) — read each file that exists so you have full context of what has been decided so far

> **What the script validates internally** (for reference — agents do not need to duplicate these checks):
> 1. `fab/config.yaml` and `fab/constitution.md` exist (project initialized)
> 2. `fab/current` exists and is non-empty (active change set)
> 3. Change directory `fab/changes/{name}/` exists
> 4. `.status.yaml` exists within the change directory

### 3. Centralized Doc Lookup (when operating on an active change)

Selectively load relevant domain docs based on the change's scope:

1. Read the proposal's **Affected Docs** section (or spec's **Affected docs** metadata) to identify which domains are relevant
2. For each referenced domain, read `fab/docs/{domain}/index.md` to understand the domain's docs
3. Read the specific centralized doc(s) referenced by the Affected Docs entries (the New, Modified, and Removed entries) — read `fab/docs/{domain}/{name}.md` for each listed doc that exists
4. If a referenced doc or domain does not exist yet (e.g., listed under New Docs), note this and proceed without error — it will be created by `/fab:archive`
5. Use this context to ground all artifact generation (specs, plans, tasks, reviews) in the real current state, not assumptions

### 4. Source Code Loading (during implementation and review)

Load only the source files relevant to the current work:

1. Read the relevant source files referenced in the plan's **File Changes** section (New, Modified, Deleted) or in the task descriptions
2. Scope to files actually touched by the change — do not load the entire codebase
3. This applies primarily to `/fab:apply` and `/fab:review`

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab:command` or `Next: /fab:commandA or /fab:commandB (description)`

### Lookup Table

| After skill | Stage reached | Next line |
|-------------|---------------|-----------|
| `/fab:init` | initialized | `Next: /fab:new <description> or /fab:hydrate <sources>` |
| `/fab:hydrate` | docs hydrated | `Next: /fab:new <description> or /fab:hydrate <more-sources>` |
| `/fab:new` | proposal done | `Next: /fab:continue or /fab:ff (fast-forward all planning)` |
| `/fab:continue` → specs | specs done | `Next: /fab:continue (plan) or /fab:ff (fast-forward) or /fab:clarify (refine spec)` |
| `/fab:continue` → plan | plan done | `Next: /fab:continue (tasks) or /fab:clarify (refine plan)` |
| `/fab:continue` → tasks | tasks done | `Next: /fab:apply` |
| `/fab:ff` | tasks done | `Next: /fab:apply` |
| `/fab:clarify` | same stage | `Next: /fab:clarify (refine further) or /fab:continue or /fab:ff` |
| `/fab:apply` | apply done | `Next: /fab:review` |
| `/fab:review` (pass) | review done | `Next: /fab:archive` |
| `/fab:review` (fail) | review failed | *(contextual — see /fab:review for fix options)* |
| `/fab:archive` | archived | `Next: /fab:new <description> (start next change)` |
