---
name: _review
description: "Review behavior — inward sub-agent (spec/tasks/checklist) and outward sub-agent (Codex→Claude cascade with full repo access), dispatched in parallel during the review stage."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Review Dispatch

> This file defines shared review logic used by `/fab-continue`, `/fab-ff`, and `/fab-fff`.
> Orchestrators reference this file by name rather than inlining review dispatch logic, ensuring
> review behavior is authoritative in one location — the same pattern as `_generation.md` for
> artifact generation procedures.
>
> **Orchestration** (stage guards, Verdict pass/fail transitions, rework options, rework loop)
> remains in each orchestrator's own file. This partial covers only the mechanics of dispatching
> the review sub-agents and merging their findings.

---

## Preconditions

- `tasks.md` and `checklist.md` MUST exist
- All tasks MUST be `[x]`. If not: STOP with "{N} of {total} tasks are incomplete."

---

## Inward Sub-Agent Dispatch

The inward sub-agent validates implementation against the spec, tasks, and checklist. It provides a fresh perspective — no shared context with the applying agent beyond the explicitly provided artifacts.

**Dispatch**: Via the Agent tool (`subagent_type: "general-purpose"`).

**Context provided to the sub-agent**: Standard subagent context files (per `_preamble.md` § Standard Subagent Context), plus change-specific files: `spec.md`, `tasks.md`, `checklist.md`, relevant source files (files touched by the change), and target memory file(s) from `docs/memory/`.

### Validation Steps

The inward sub-agent performs all of these checks:

1. **Tasks complete**: All `[x]` in `tasks.md`
2. **Quality checklist**: Inspect code/tests per CHK item. Mark `[x]` if met, `[x] **N/A**: {reason}` if N/A, leave `[ ]` with reason if not met
3. **Run affected tests**: Scoped to touched modules/files
4. **Spot-check spec**: Verify key requirements and GIVEN/WHEN/THEN scenarios
5. **Memory drift check**: Compare implementation against referenced memory (warning only)
6. **Code quality check**: For each file modified during apply, verify:
   - Naming conventions consistent with surrounding code
   - Functions focused and appropriately sized
   - Error handling consistent with codebase style
   - Existing utilities reused where applicable
   - If `fab/project/code-quality.md` exists, check each applicable principle from `## Principles`
   - If `fab/project/code-quality.md` exists, check for violations listed in `## Anti-Patterns`

### Structured Output

The inward sub-agent SHALL return structured findings with a **three-tier priority scheme**:

- **Must-fix**: Spec mismatches, failing tests, checklist violations
- **Should-fix**: Code quality issues, pattern inconsistencies
- **Nice-to-have**: Style suggestions, minor improvements

Each finding includes: severity tier, description, and file:line reference where applicable.

---

## Outward Sub-Agent Dispatch

The outward sub-agent performs a holistic diff review with full repository access. It is given the diff of all changed files and the list of changed file paths, and is permitted to read any file in the repo to explore context.

**Dispatch**: Via the Agent tool (`subagent_type: "general-purpose"`).

**Context provided to the sub-agent**:
- The diff of all changed files: compute the merge-base against the default branch (`git merge-base HEAD origin/main` or the resolved default), then use `git diff <base>...HEAD`
- The list of changed file paths: use the same resolved base with `git diff --name-only <base>...HEAD`
- Standard subagent context files (per `_preamble.md` § Standard Subagent Context)
- Full tool access (Read, Bash, Agent) — the sub-agent MAY read any file in the repo

**Cascade**: The outward sub-agent uses a **Codex → Claude cascade**, controlled by `review_tools` in `fab/project/config.yaml`:

```yaml
review_tools:
    codex: true    # first in cascade — set to false to skip
    claude: true   # fallback — set to false to skip
    copilot: true  # used by /git-pr-review only, not this cascade
```

When `review_tools` is absent, all tools default to `true`.

1. **Check config**: Read `review_tools.codex` — if `false`, skip Codex
2. **Attempt Codex**: `command -v codex` — if found and enabled, run Codex as the reviewer
3. **Check config**: Read `review_tools.claude` — if `false`, skip Claude
4. **If Codex unavailable/disabled or fails**, attempt Claude: `command -v claude` — if found and enabled, run Claude as the reviewer
5. If all enabled tools are unavailable or fail, return an empty findings set (graceful no-op — not an error condition). The review stage continues normally.

### Focus Areas

The outward sub-agent prompt instructs it to look for:

1. **Interface contract violations** — types, return values, API shape mismatches between changed code and callers/dependents
2. **Inconsistencies with documented patterns** — naming conventions, error handling style, or structural patterns described in memory files (`docs/memory/`) that the changed code violates
3. **Missing cross-references** — memory files or spec files that should reference the changed behavior but do not
4. **Behavioral regressions requiring full-repo context** — issues that the inward reviewer (scoped to changed files) would miss but are visible with full codebase access
5. **Structural issues** — duplication of existing utilities, abstraction violations, or architectural drift visible only in the broader codebase context

### Structured Output

The outward sub-agent returns findings in the same three-tier format as the inward sub-agent:

- **Must-fix**: Interface violations, regressions, or structural issues that must be resolved before ship
- **Should-fix**: Pattern inconsistencies or missing cross-references — addressed when clear and low-effort
- **Nice-to-have**: Minor improvements, optional refactors

Each finding includes: severity tier, description, and file:line reference where applicable.

---

## Parallel Dispatch

Both sub-agents (inward and outward) are dispatched **in parallel**. The orchestrator waits for both to return before proceeding to the Findings Merge step.

---

## Findings Merge

After both sub-agents return, their findings are merged into a single prioritized set:

1. **Collect**: Gather all findings from both sub-agents
2. **Deduplicate**: If both sub-agents flag the same file:line issue, consolidate into a single finding (use the higher severity if they differ)
3. **Merge by severity**: Combine into a unified must-fix / should-fix / nice-to-have list
4. **Pass/fail determination**:
   - If **any must-fix** finding exists (from either sub-agent) → review **fails**
   - If only should-fix and/or nice-to-have findings remain (from either sub-agent) → review **may pass**

The merged findings set is returned to the orchestrator for verdict and rework decisions.

> **Note**: The rework loop (bounded retry, escalation rule, pass/fail state transitions) is defined
> in the orchestrator (`fab-continue.md` Verdict section, `fab-ff.md` Step 6, `fab-fff.md` Step 6),
> not in this file. This file defines only the dispatch and merge mechanics.
