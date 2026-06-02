# Fab Skills Reference

> Detailed behavior for each `/fab-*` skill. For a quick overview, see the [Quick Reference](overview.md#quick-reference).

---

## Terminology: "spec" vs "memory"

Fab uses two distinct terms to avoid confusion:

| Term | Location | Meaning |
|------|----------|---------|
| **Memory files** | `docs/memory/` | Source-of-truth documentation for the system. Contains both requirements (what) and durable design decisions (why). Updated by `/docs-hydrate-memory` (from external sources) and `/fab-continue` (hydrate) (from change artifacts). |
| **plan.md** | `fab/changes/{name}/plan.md` | Change-level plan. Carries the `## Requirements` (RFC-2119 + GIVEN/WHEN/THEN), `## Tasks`, and `## Acceptance` sections, co-generated at apply entry. |

As of 1.10.0 the `spec` stage and the separate `spec.md` artifact are removed. Requirement capture lives in `plan.md`'s `## Requirements` section, co-generated with tasks and acceptance at apply entry. The canonical artifact flow is `intake.md → plan.md → code`.

---

## Skill Helpers (`helpers:` Frontmatter)

Every skill MAY declare additional helper files it needs to load via a `helpers:` frontmatter list. The agent reads each declared helper's `.claude/skills/{helper}/SKILL.md` after reading `_preamble` and before executing the skill body.

**Allowed values** (4): `_generation`, `_review`, `_cli-fab`, `_cli-external`.

**Default**: omitted (or `[]`) — the skill loads only `_preamble`.

**Example**:

```yaml
---
name: fab-continue
description: ...
helpers: [_generation, _review]
---
```

**Not allowed**: `_naming` and `_cli-rk` — their content is inlined into `_preamble.md` (§ Naming Conventions and § Run-Kit (rk) Reference respectively).

**Implicit**: `_preamble` itself is loaded universally — never list it.

**Current mapping** (post-2026-04-18):

| Skill | `helpers:` |
|-------|------------|
| `fab-new`, `fab-draft` | `[_generation]` |
| `fab-continue`, `fab-ff`, `fab-fff` | `[_generation, _review]` |
| `fab-operator` | `[_cli-fab, _cli-external]` |
| All other 19 skills | omitted (load only `_preamble`) |

Validation is **convention-only** — `fab sync` does not reject skills with unknown helper values. Drift surfaces as runtime behavior (agent loads an unexpected file or fails to find a needed one).

---

## Context Loading Convention

Every skill that generates or validates artifacts MUST load relevant context before proceeding. This ensures agents produce accurate, grounded output rather than hallucinating requirements or ignoring existing patterns.

**Always loaded** (by every skill except `/fab-setup`, `/fab-switch`, `/fab-status`, `/docs-hydrate-memory`):
- `fab/project/config.yaml` — project configuration, naming conventions, model tiers
- `fab/project/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- `fab/project/context.md` — free-form project context: tech stack, conventions, architecture *(optional — no error if missing)*
- `fab/project/code-quality.md` — coding standards for apply/review: principles, anti-patterns, test strategy *(optional — no error if missing)*
- `fab/project/code-review.md` — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
- `docs/memory/index.md` — memory landscape (which domains and memory files exist)
- `docs/specs/index.md` — specifications landscape (pre-implementation design intent, human-curated)

**Change context** (loaded by skills operating on an active change):
- `.status.yaml` — current stage, progress
- All completed artifacts in the active change folder (`intake.md`, `plan.md`)

**Memory file lookup** (loaded by skills operating on an active change):
- Read the intake's "Affected Memory" section to identify relevant domains
- Read domain indexes (`docs/memory/{domain}/index.md`) for each relevant domain
- Read the specific memory file(s) referenced by the Affected Memory entries
- If a referenced file doesn't exist yet (listed under New Files), note this and proceed — it will be created by `/fab-continue` (hydrate)
- This grounds all artifact generation (spec, plan, reviews) in the real current state, not assumptions

**Source code** (loaded during implementation and review):
- Read relevant source files referenced in the task descriptions
- Scope to files actually touched by the change — don't load the entire codebase

Each skill section below lists its specific context requirements under a **Context** field.

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab-command` or `Next: /fab-commandA or /fab-commandB (description)`

**Lookup table**:

| After | Stage reached | Next line |
|-------|---------------|-----------|
| `/fab-setup` | initialized | `Next: /fab-new <description> or /docs-hydrate-memory <sources>` |
| `/docs-hydrate-memory` | memory hydrated | `Next: /fab-new <description> or /docs-hydrate-memory <more-sources>` |
| `/fab-new` | intake ready (activated) | `Next: /fab-continue or /fab-clarify (refine intake) or /fab-ff or /fab-fff` |
| `/fab-draft` | intake ready (not activated) | `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-clarify or /fab-ff or /fab-fff` |
| `/fab-continue` (from intake ready) | apply active/done | `Next: /fab-continue (apply co-generates plan.md — requirements + tasks + acceptance — and runs tasks)` |
| `/fab-ff` | apply done | `Next: /fab-continue (review)` |
| `/fab-clarify` | same stage | `Next: /fab-clarify (refine further) or /fab-continue or /fab-ff` |
| `/fab-continue` → apply | apply done | `Next: /fab-continue (review)` |
| `/fab-continue` → review (pass) | review done | `Next: /fab-continue (hydrate)` |
| `/fab-continue` → review (fail) | review failed | *(contextual — see [Review Behavior](#review-behavior-via-fab-continue) for fix options)* |
| `/fab-continue` → hydrate | hydrated | `Next: /fab-archive` |

---

## `/fab-setup`

**Purpose**: Bootstrap `fab/` in an existing project and manage ongoing configuration. Delegates structural setup to `fab-sync.sh` and adds interactive configuration on top. Safe to run repeatedly (idempotent). Also provides subcommands for config, constitution, and migrations.

**Prerequisite**: Fab Kit must be installed (`brew install fab-kit`) and `fab init` or `fab sync` must have been run in the project.

**Subcommands**:

| Subcommand | Purpose |
|------------|---------|
| `/fab-setup config [section]` | Create or update `fab/project/config.yaml` interactively, preserving comments |
| `/fab-setup constitution` | Create or amend `fab/project/constitution.md` with semantic versioning |
| `/fab-setup migrations [file]` | Run version migrations against the current project |

When called without arguments, `/fab-setup` runs the full bootstrap: invokes `fab-sync.sh` for structural setup, then delegates to `config` and `constitution` subcommands for any missing artifacts. Unrecognized arguments are rejected with a message listing valid subcommands.

**Creates** (first run only — skipped if already present):
- `fab/project/config.yaml` — project configuration (via `/fab-setup config`)
- `fab/project/constitution.md` — project principles and constraints (via `/fab-setup constitution`)
- `fab/.kit-migration-version` — migration version (via `fab-sync.sh`)
- `docs/memory/index.md` — initial memory index (via `fab-sync.sh`)
- `docs/specs/index.md` — specifications index (via `fab-sync.sh`)
- `fab/changes/` — empty, ready for change folders (via `fab-sync.sh`)
- `.claude/skills/` — symlinks pointing into `src/kit/skills/` (via `fab-sync.sh`)

**Delegation pattern**: `fab-sync.sh` handles all non-interactive structural setup (directories, symlinks, indexes, `.envrc`, `.gitignore`). `/fab-setup` adds the interactive parts (config, constitution). `fab-sync.sh` can be run independently (e.g., in CI or after bootstrap download) without requiring `/fab-setup`.

**Examples**:
```
# First run — full bootstrap
/fab-setup
→ "Running fab-sync.sh... structure created."
→ "What's the project name?"
→ "Describe the tech stack and conventions..."
→ "fab/ initialized with config, constitution, and empty memory index."
→ "Next: /fab-new <description> or /docs-hydrate-memory <sources>"

# Re-run — structural health check
/fab-setup
→ "fab/ already initialized. Verified structure."

# Subcommand — update config
/fab-setup config
→ "Updating fab/project/config.yaml..."

# Subcommand — run migrations after kit upgrade
/fab-setup migrations
→ "Applying migration 0.2.0-to-0.3.0... done."
```

---

## `/docs-hydrate-memory [sources...]`

**Purpose**: Ingest external sources into `docs/memory/` with domain mapping and index maintenance. Safe to run repeatedly — content is merged into existing memory files without duplication.

**Prerequisite**: `docs/memory/` must exist (run `/fab-setup` first). If missing, abort with: *"docs/memory/ not found. Run /fab-setup first to create the memory directory."*

**Arguments**:
- `[sources...]` *(required)* — one or more URLs or local paths containing documentation to ingest. Supported source types:
  - **Notion URLs** — pages or databases (fetched via Notion MCP or API)
  - **Linear URLs** — issues or projects (fetched via Linear MCP or API)
  - **Local files/directories** — markdown, text, or directories of files (read from filesystem)

**Creates/Updates**:
- `docs/memory/{domain}/{topic}.md` — memory files (created or merged)
- `docs/memory/{domain}/index.md` — domain indexes (created or updated)
- `docs/memory/index.md` — top-level index (updated with new domains/files)

**Examples**:
```
# Hydrate memory from a Notion page
/docs-hydrate-memory https://notion.so/myteam/API-Spec-abc123
→ "Fetched: API Spec (Notion)"
→ "Created: docs/memory/api/endpoints.md, docs/memory/api/authentication.md"
→ "Updated: docs/memory/index.md"

# Ingest local legacy documentation
/docs-hydrate-memory ./legacy-docs/payments/
→ "Fetched: 3 files from ./legacy-docs/payments/"
→ "Created: docs/memory/payments/checkout.md, docs/memory/payments/refunds.md"

# Multiple sources at once
/docs-hydrate-memory https://notion.so/myteam/Auth-xyz ./legacy-docs/payments/
→ "Fetched: Auth Design (Notion), 3 files from ./legacy-docs/payments/"
→ "Created: docs/memory/auth/oauth.md, docs/memory/payments/checkout.md"
→ "Updated: docs/memory/index.md"
```

**Behavior**:

1. **Pre-flight check**: Verify `docs/memory/` and `docs/memory/index.md` exist (abort with guidance if not). If no sources are provided, abort with usage message.
2. **Fetch/read** each source:
   - Notion URLs → fetch page content via Notion MCP or API
   - Linear URLs → fetch issue/project content via Linear MCP or API
   - Local paths → read files; if directory, read all markdown files recursively
3. **Analyze** fetched content to identify domains and topics
4. **Create or merge** memory files — for each identified topic, either create a new file in `docs/memory/{domain}/` or merge into an existing file. Follow the [Memory File Format](templates.md#memory-file-format-fabmemory) and [Hydration Rules](templates.md#hydration-rules).
5. **Update domain indexes** — create or update `docs/memory/{domain}/index.md` for each affected domain
6. **Update top-level index** — update `docs/memory/index.md` with new domains and expanded file lists
7. **Report** what was created and updated

---

## `/fab-new <description>`

**Purpose**: Start a new change — creates the intake and activates it.

**Context**: config, constitution, `docs/memory/index.md` (to understand existing memory landscape)

**Creates**:
- Change folder named `{YYMMDD}-{XXXX}-{slug}`
- `.status.yaml` manifest
- `intake.md` from template (with clarifying questions if ambiguous)
- `.fab-status.yaml` symlink (auto-activation)

**Arguments**:
- `<description>` — natural language description of the change, Linear ticket ID (e.g., `DEV-988`), or backlog ID (e.g., `90g5`) (required)

**Examples**:
```
/fab-new Add OAuth2 support for Google and GitHub sign-in
→ Created fab/changes/260115-a7k2-add-oauth/
→ Activated: 260115-a7k2-add-oauth
```

**Behavior**:
1. Generate folder name: today's date (`YYMMDD`) + 4 random alphanumeric chars + 2-6 word slug from description
2. Create `fab/changes/{name}/`
3. Initialize `.status.yaml` with all stages `pending`, then make the intake stage active
4. Generate `intake.md` using template (loading `fab/project/constitution.md` and `fab/project/config.yaml` as context)
5. Perform gap analysis — check whether the change is already covered by existing mechanisms
6. Use SRAD-driven adaptive questioning (no fixed cap) to resolve ambiguities conversationally
7. Advance intake to `ready` — the artifact exists and is open for `/fab-clarify` refinement
8. Activate the change via `fab change switch` — creates the `.fab-status.yaml` symlink so `/fab-continue` works immediately

> **Create without activating**: Use `/fab-draft` to queue a change for later without switching context.

---

## `/fab-draft <description>`

**Purpose**: Create a change intake without activating it. Use this to queue a change for later work.

**Context**: config, constitution, `docs/memory/index.md` (to understand existing memory landscape)

**Creates**:
- Change folder named `{YYMMDD}-{XXXX}-{slug}`
- `.status.yaml` manifest
- `intake.md` from template (with clarifying questions if ambiguous)

**Arguments**:
- `<description>` — natural language description of the change, Linear ticket ID (e.g., `DEV-988`), or backlog ID (e.g., `90g5`) (required)

**Examples**:
```
/fab-draft Add OAuth2 support for Google and GitHub sign-in
→ Created fab/changes/260115-a7k2-add-oauth/
→ Next: /fab-switch 260115-a7k2-add-oauth to make it active
```

**Behavior**: Identical to `/fab-new` Steps 1-7, but does NOT activate the change. The user must run `/fab-switch {name}` to make it active before proceeding.

---

## `/fab-continue [<stage>]`

**Purpose**: Advance through the pipeline — finishing the current `ready` planning stage, generating the next artifact, and leaving it at `ready` for optional `/fab-clarify` refinement. Or, when called with a stage argument, reset to that stage and regenerate from there.

**Arguments**:
- `<stage>` *(optional)* — target stage to reset to (`apply` is the typical reset). The legacy `tasks` and `spec` targets are removed and error with a pointer to `apply` / `/fab-clarify intake`. Used after `/fab-continue` (review) identifies issues upstream. When provided, resets `.status.yaml` to this stage and re-runs from that point forward.

**Context** (varies by target stage):
- **Apply stage**: config, constitution, `intake.md` (used to co-generate `plan.md` at apply entry); plus the resumable plan + source code on subsequent invocations

**Examples**:
```
/fab-continue
→ (intake ready) Finishes intake, starts apply, co-generates plan.md (## Requirements + ## Tasks + ## Acceptance), executes the unchecked tasks under ## Tasks, finishes apply.

/fab-continue apply
→ "Resetting to apply. plan.md persists (delete it to force regeneration); re-running unchecked tasks."
```

**Behavior** (no argument — normal forward flow):
1. Read `.status.yaml` to determine current stage and state
2. Intake in `ready` state: finish intake, start apply, then execute apply
3. Intake in `active` state (backward compat): generate intake if missing, advance to `ready`
4. For execution stages: execute the stage's behavior and finish it
5. Load relevant template + context (including `fab/project/constitution.md` for project principles)
6. Apply entry: invoke the unified Plan Generation Procedure — co-generate `plan.md` `## Requirements` (from `intake.md`) + `## Tasks` + `## Acceptance` (skipped on resume when `plan.md` already exists)
7. Update `.status.yaml`

**Behavior** (with stage argument — reset and regenerate):
1. **Guard**: target stage must be a valid 6-pipeline stage (typically `apply`). Reset to `tasks` or `spec` errors with `"tasks"/"spec" stages were removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify intake.`
2. Reset `.status.yaml` stage to the target. Mark all stages from target onward as `pending`.
3. For an intake reset, regenerate the intake artifact in place; for execution resets, re-run from that stage.
4. Downstream artifacts are invalidated only by re-running apply: `plan.md` persists across resets (deleting it forces regeneration); task checkboxes are NOT auto-cleared.
5. For an intake reset, advance intake to `ready` (not `done`) to preserve the `/fab-clarify` opportunity.

---

## `/fab-ff` (Fast Forward)

**Purpose**: Fast-forward apply → review → hydrate (everything after intake). Gated on the single intake confidence gate (flat 3.0), with sub-agent review, auto-rework loop (up to 3 cycles with prioritized findings), and stop on exhaustion. Accepts `--force` to bypass the gate. No `/fab-clarify` runs inside the bracket.

**Context**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/` (loaded once for the apply → hydrate run)

**Flow**: apply (co-generates `plan.md`, executes tasks) → review → hydrate

**When to use**:
- Small, well-understood changes
- Clear requirements upfront
- Want to reach implementation quickly

**Example**:
```
/fab-new Add a logout button to the navbar that clears session
/fab-ff         # fast-forward: apply → review → hydrate
```

**Behavior**:
1. Check the intake gate (confidence >= 3.0, flat). Abort if below threshold. Skip if `--force`.
2. Run apply (single subagent invocation): co-generate `plan.md` (## Requirements from `intake.md` + ## Tasks + ## Acceptance), then execute unchecked tasks under `## Tasks` in dependency order, running tests after each. Under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md` — no clarify step.
3. **Review** — dispatch to sub-agent (fresh context). Sub-agent returns prioritized findings (must-fix / should-fix / nice-to-have); inward sub-agent inspects items under `plan.md` `## Acceptance` against `## Requirements`
4. **On pass** — advance to hydrate
5. **On fail** — auto-rework loop (up to 3 cycles): triage findings by priority, autonomously select rework path (fix code, revise plan, revise requirements), re-apply, spawn fresh sub-agent for re-review. Escalation after 2 consecutive fix-code attempts. Stop after 3 failed cycles with summary.
6. Hydrate into `docs/memory/`

---

## `/fab-fff` (Full Autonomous Pipeline)

**Purpose**: Run the entire automated Fab pipeline — apply → review → hydrate → ship → review-pr — in a single invocation (everything after intake). Gated on the single intake confidence gate (flat 3.0, same as `/fab-ff`). No `/fab-clarify` runs inside the bracket. Autonomously reworks on review failure using sub-agent review with prioritized findings (3-cycle retry cap, escalation after 2 consecutive fix-code failures). Accepts `--force` to bypass the gate.

**Prerequisite**: Active change with completed `intake.md`.

**Context**: Same as `/fab-ff` — all context loaded upfront (config, constitution, intake, memory index, affected memory files).

**Example**:
```
/fab-fff
→ --- Implementation ---
→ ... (apply: plan.md co-generated — requirements + tasks + acceptance — then tasks executed)
→ --- Review ---
→ ... (validation passed)
→ --- Hydrate ---
→ ... (memory hydrated)
→ --- Ship ---
→ ... (PR created)
→ --- Review-PR ---
→ ... (PR review processed)
→ "Pipeline complete."
```

**Behavior**:
1. **Intake gate** (skip if `--force`): Check confidence >= 3.0 (flat). Abort if below threshold.
2. **Resumability**: Check `progress` map — skip any stage already marked `done` or `skipped`. Re-invoking after interruption picks up from the first incomplete stage.
3. **Step 1 — Implementation**: Run apply (one subagent call) — co-generate `plan.md` (## Requirements from `intake.md` + ## Tasks + ## Acceptance), then execute unchecked tasks under `## Tasks` in dependency order, running tests after each. Under-specified requirements are resolved inline as graded SRAD assumptions — no clarify step.
4. **Step 2 — Review**: Dispatch to review sub-agent (fresh context, prioritized findings). On failure, triage findings by priority and autonomously select rework path (fix code, revise plan, revise requirements). Re-review via fresh sub-agent. Retry up to 3 cycles (escalation after 2 consecutive fix-code). Bail with summary after 3 failed cycles.
5. **Step 3 — Hydrate**: Hydrate into memory.
6. **Step 4 — Ship**: Dispatch `/git-pr` to commit, push, and create PR.
7. **Step 5 — Review-PR**: Dispatch `/git-pr-review` to process PR review comments.

**Key difference from `/fab-ff`**: The difference is scope only. `/fab-fff` extends through ship and review-pr; `/fab-ff` stops at hydrate. Both have the identical single intake gate, no in-bracket clarify, and identical auto-rework (3-cycle cap with escalation). Both accept `--force` to bypass the gate.

---

## `/fab-proceed`

**Purpose**: Context-aware orchestrator — detects the current pipeline state (active change, branch, conversation context, unactivated intakes) and automatically runs whatever prefix steps are needed (fab-new, fab-switch, git-branch) before delegating to `/fab-fff` for the full pipeline. Conversation context is the interpretive lens for any unactivated intakes: an unrelated draft never hijacks the pipeline when the current conversation is about a different topic.

**Prerequisite**: None — can bootstrap from conversation context alone.

**Context**: No direct context loading — delegates all pipeline context loading to `/fab-fff`.

**Example**:
```
/fab-proceed
→ /fab-proceed — detecting state...
→ Activated: 260325-kxw7-fab-proceed-orchestrator
→ Branch: 260325-kxw7-fab-proceed-orchestrator (created)
→ Handing off to /fab-fff...
→ {fab-fff output follows}
```

**Behavior**:
1. **State detection** — 5-step pipeline: (1) active change check (`fab resolve --folder`), (2) branch check (`git branch --show-current`, runs only if active change found), (3) conversation classification as substantive/empty-thin, (4) unactivated intake scan (`fab/changes/`, retain full candidate list), (5) dispatch decision combining Steps 1–4 via the 7-row dispatch table. Steps 3 and 4 are order-independent and both run whenever no active change was found.
2. **Relevance assessment** — when substantive conversation AND ≥1 unactivated intake exist, score each candidate by reading its title + Origin + Why + What Changes sections; clearly relevant requires shared topic + overlapping terminology + consistent scope (no partial/vague overlap); asymmetric-bias rule: ambiguous → not clearly relevant → fall through to `/fab-new`; date-descending tiebreak used only among equally-relevant candidates.
3. **Prefix dispatch** — subagent dispatch for prefix steps (fab-new, fab-switch, git-branch) per `_preamble.md` § Subagent Dispatch
4. **Terminal delegation** — invoke `/fab-fff` via the Skill tool (not subagent) for full user visibility
5. **Bypass notes** — when `/fab-new` runs despite ≥1 unactivated intake being present, emit `Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.` for each scanned draft (date-descending order, before any step reports)

**Key properties**:
- No arguments, no flags — infers everything from context
- Zero-prompt — ambiguous relevance resolved by asymmetric-bias rule, never by asking
- Idempotent — re-running detects completed steps and skips them
- Does not run preflight or load `_preamble.md` context — delegates to `/fab-fff`
- Errors on empty context + no intake: "Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first."

---

## `/fab-clarify`

**Purpose**: Deepen and refine the **intake** artifact (`intake.md`) without advancing. Clarification is intake-only (1.10.0) — it is where human judgment lives, gated by the single intake confidence gate. There is no post-intake clarify; inside apply the agent resolves ambiguity inline as graded SRAD assumptions in `plan.md`.

**Context**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

**Example**:
```
/fab-clarify
→ "Stage: intake (active). Reviewing intake.md for gaps..."
→ "Found 2 [NEEDS CLARIFICATION] markers. Resolving..."
→ "Resolved scope boundaries; recomputed intake confidence."
```

**When to use**:
- Intake has unresolved ambiguities or [NEEDS CLARIFICATION] markers
- You want deeper technical research before unlocking the automated bracket
- Intake scope needs sharpening before the intake gate will pass

**Behavior**:
1. Read `.status.yaml` to determine current stage
2. **Guard**: `/fab-clarify` operates only at intake. At apply or later, STOP and point the user to `/fab-continue` (rework) or editing `plan.md` `## Requirements`; to re-clarify the intake, reset via `/fab-continue intake` first. The legacy `spec`/`plan`/`tasks` targets are removed.
3. Load `intake.md` + relevant context
4. Analyze the intake for gaps, ambiguities, and opportunities to deepen: [NEEDS CLARIFICATION] markers, `<!-- assumed: ... -->` markers, scope boundaries, affected areas, impact, memory coverage
5. Refine the artifact **in place** — edit the existing file, don't regenerate from scratch
6. Recompute the intake score (`fab score --stage intake`) and report what was clarified/refined
7. Do **not** advance the stage or update `.status.yaml` stage field

**Key property**: Idempotent and non-advancing. Calling `/fab-clarify` multiple times is safe — it refines further each time. It never transitions to the next stage. Use `/fab-continue` when satisfied.

---

## Apply Behavior (via `/fab-continue`)

**Purpose**: Co-generate `plan.md` (`## Requirements` from `intake.md` + `## Tasks` + `## Acceptance`) at the entry sub-step, then execute the unchecked tasks in `plan.md` `## Tasks` (main sub-step). Both run in a single skill invocation.

**Context**: config, constitution, `intake.md`, `plan.md` (read on resume; written at entry), relevant source code (files referenced in tasks)

**Example**:
```
/fab-continue
→ "Apply entry: co-generating plan.md (requirements + tasks + acceptance) from intake.md..."
→ "Starting implementation. 12 tasks remaining."
```

**Behavior**:
1. **Entry sub-step (Plan Generation)**: If `plan.md` does not exist, run the unified Plan Generation Procedure — co-generate `## Requirements` (from `intake.md`; or fold a legacy `spec.md` if present), `## Tasks`, and `## Acceptance` in one pass, with required `<!-- R# -->` trace annotations. Skipped when `plan.md` already exists (resumability).
2. **Main sub-step (Task Execution)**: Parse `plan.md` `## Tasks` for unchecked items `- [ ]`. The `## Acceptance` section is OUT OF SCOPE for apply.
3. Execute tasks in dependency order
4. Respect parallel markers `[P]`
5. After completing each task, run relevant tests (e.g., the test file for the module just modified). Fix failures before moving on.
6. Mark each task `[x]` immediately upon completion (not batched at the end)
7. Update `.status.yaml` progress after each task

**Resumability**: `/fab-continue` (apply) is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-continue` picks up from the first unchecked item under `## Tasks`. Plan Generation is skipped when `plan.md` already exists. The markdown checklist *is* the progress state — no separate tracking needed.

---

## Review Behavior (via `/fab-continue`)

**Purpose**: Validate implementation against the plan's `## Requirements` and `## Acceptance` items using a **review sub-agent** running in a separate execution context.

**Context**: config, constitution, `plan.md` (containing `## Requirements`, `## Tasks`, and `## Acceptance` sections), target memory file(s) from `docs/memory/`, relevant source code (files touched by the change)

**Sub-agent dispatch**: Review validation is dispatched to a sub-agent that runs in a fresh context — no shared state with the applying agent beyond the explicitly provided artifacts. The orchestrating LLM may use any review agent available (a `code-review` skill, a general-purpose sub-agent with review instructions, or any equivalent). No specific agent is prescribed.

**Example**:
```
/fab-continue
→ "Dispatching review to sub-agent..."
→ "✓ 12/12 tasks complete"
→ "✓ 10/12 acceptance items passed"
→ "✗ 2 items need attention: [A-007, A-011]"
→ "  must-fix: A-007 — missing error handling (src/api.ts:42)"
→ "  should-fix: A-011 — inconsistent naming (src/utils.ts:15)"
```

**Checks** (the sub-agent performs all of these):
1. All tasks in `plan.md` `## Tasks` marked `[x]`
2. All acceptance items in `plan.md` `## Acceptance` verified and checked off — the sub-agent re-reads each `A-*` (or legacy `CHK-*` for in-flight migrated plans) item, inspects the relevant code/tests, and marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match requirements (spot-check key scenarios from `plan.md` `## Requirements`)
5. No memory drift detected (implementation doesn't contradict memory files)
6. Code quality check — naming consistency, function size, error handling, utility reuse

**Structured output**: The sub-agent returns prioritized findings using a three-tier scheme:
- **Must-fix**: Requirements mismatches, failing tests, acceptance violations — always addressed
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

**Pass/fail**: If any must-fix findings exist, the review fails. If only should-fix/nice-to-have remain, the review may pass.

**On failure** (manual rework in `/fab-continue`), the findings are presented with priority annotations and the user chooses where to loop back:

- **Fix code** → `/fab-continue` (apply)
  Implementation bug. The agent identifies which tasks need rework, unchecks them in `plan.md` `## Tasks` (marks `- [ ]` again with a `<!-- rework: reason -->` comment), and re-runs `/fab-continue` which picks up the unchecked items.

- **Revise plan** → edit `plan.md`, then `/fab-continue` (apply)
  Missing or wrong tasks/acceptance items. The agent adds/modifies entries in `plan.md` (new tasks get the next sequential ID; new acceptance items use the next `A-NNN`). Completed tasks that are unaffected stay `[x]`. Only new or revised tasks are executed.

- **Revise requirements** → edit `plan.md` `## Requirements`, then `/fab-continue` (apply)
  Requirements were wrong or incomplete. The agent edits the `## Requirements` section plus the downstream `## Tasks`/`## Acceptance` it affects, then re-runs apply. For a fundamentally wrong intake, fix `intake.md` via `/fab-clarify intake` first (then re-enter apply, which regenerates requirements from the corrected intake).

The applying agent triages review comments by priority — not all comments need to be implemented. The `.status.yaml` stage is reset to the chosen re-entry point. The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

---

## Hydrate Behavior (via `/fab-continue`)

**Purpose**: Validate review passed and hydrate change artifacts into memory files. The change folder remains in `fab/changes/` after hydrate — archiving is a separate step via `/fab-archive`.

**Context**: `plan.md` (its `## Requirements`), `intake.md`, target memory file(s) from `docs/memory/`, `docs/memory/index.md` and relevant domain indexes

**Example**:
```
/fab-continue
→ "Hydrated memory: docs/memory/auth/authentication.md"
→ "Next: /fab-archive"
```

**Behavior**:
1. **Final validation** — review must pass (all tasks under `plan.md` `## Tasks` are `[x]`, all acceptance items under `## Acceptance` are `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose plans reference the same memory files. If found, warn the user: *"Change {name} also modifies {file}. After this hydrate, that change's plan was written against a now-stale base. Re-review with `/fab-continue` after switching to it."*
3. **Hydrate into `docs/memory/`**:
   The agent reads `plan.md` `## Requirements` and the current memory file, then rewrites the memory file to incorporate the changes:
   - **From `plan.md` `## Requirements`** → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements that the plan's `### Deprecated Requirements` explicitly deprecates. Extract durable design decisions into Design Decisions section.
   The agent compares against the existing memory file to determine what's new vs changed vs removed — no explicit delta markers needed. Minimize edits to unchanged sections to prevent drift.
4. **Update status** to `hydrate: done` in `.status.yaml`

**Recovery**: Hydration modifies memory files in-place. If the merge goes wrong (garbled text, incorrect removals), the only recovery is `git checkout` on the affected memory files. Commit (or at least review the diff) before pushing after hydrate.

---

## `/fab-archive [<change-name>]`

**Purpose**: Standalone housekeeping command — not a pipeline stage. Moves completed changes to the archive directory, updates the archive index, marks backlog items done, and clears the pointer.

**Prerequisite**: `hydrate: done` in `.status.yaml`. If hydrate is not done, stop with: *"Hydrate has not completed. Run /fab-continue to hydrate memory first."*

**Arguments**:
- `<change-name>` *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`

**Example**:
```
/fab-archive
→ "Archived to fab/changes/archive/260115-a7k2-add-oauth/"
→ "Next: /fab-new <description>"
```

**Behavior**:
1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. No rename.
2. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first.
3. **Mark backlog items done** — exact-ID check (always), then keyword scan with interactive confirmation.
4. **Clear pointer** — remove `.fab-status.yaml` symlink only if the archived change is the active one.

**Order of operations**: Steps 1–4 execute in this order for safety. Folder move first (recoverable if interrupted — re-run detects folder already in archive and completes remaining steps). Index after folder is in place. Backlog marking after index. Pointer last.

**Restore mode** (`/fab-archive restore <change-name> [--switch]`): Moves an archived change back to `fab/changes/`. Preserves all artifacts and `.status.yaml` without modification. Optionally activates via `--switch` flag.

---

## `/fab-switch <change-name>`

**Purpose**: Switch the active change when multiple changes exist.

**Example**:
```
/fab-switch fix-checkout
→ ".fab-status.yaml → 260202-m3x1-fix-checkout-bug"
```

**Behavior**:
1. Match `change-name` against `fab/changes/` (supports partial/slug match)
2. **Ambiguous match** — if multiple changes match the input (e.g., `/fab-switch add` matches both `260115-a7k2-add-oauth` and `260202-m3x1-add-dark-mode`), list the matches and ask the user to pick one. Never guess.
3. **No match** — if nothing matches, list available changes and ask
4. Create the `.fab-status.yaml` symlink pointing to the change's `.status.yaml`
5. Display the switched change's status summary

---

## `/git-branch [change-name]`

**Purpose**: Create or check out a git branch matching the active (or specified) change. Standalone git command — does not modify fab state.

**Example**:
```
/git-branch
→ "Branch: 260224-vx4k-decouple-git-from-fab-switch (created)"
```

**Behavior**:
1. Check inside a git repo (`git rev-parse --is-inside-work-tree`)
2. Resolve change name (from argument or `.fab-status.yaml`)
3. Derive branch name: `{change-name}` (no prefix)
4. Context-dependent action:
   - **Already on target** → no-op
   - **Target branch exists** → switch to it (`git checkout`)
   - **On `main`/`master`** → auto-create branch
   - **On other branch, no upstream** → rename current branch (`git branch -m`)
   - **On other branch, has upstream** → create new branch (leaving current intact)
5. Report result

**Key properties**:
- Does not modify `.fab-status.yaml` or `.status.yaml`
- Idempotent — checking out an already-active branch is a no-op
- Always enabled if in a git repo

---

## `/fab-status`

**Purpose**: Show current change state at a glance.

**Example output**:
```
Change: 260115-a7k2-add-oauth
Branch: 260115-a7k2-add-oauth
Stage:  intake (1/6)

Progress:
  ◉ intake      active
  ○ apply       pending
  ○ review      pending
  ○ hydrate     pending
  ○ ship        pending
  ○ review-pr   pending

Plan: not yet generated (created at apply entry)

Next: Complete intake.md, then /fab-continue
```

---

## `/fab-discuss`

**Purpose**: Prime the agent with project context for a discussion session. Loads the standard always-load layer and presents an orientation summary of the project landscape — memory domains, specs, active change (if any). Session entry point for exploratory conversations, not a pipeline stage.

**Context**: Same as always-load (`_preamble.md` §1) — `config.yaml`, `constitution.md`, `context.md` (optional), `code-quality.md` (optional), `code-review.md` (optional), `docs/memory/index.md`, `docs/specs/index.md`. Also reads `.fab-status.yaml` symlink for active change awareness (light touch).

**Key properties**:
- No active change required — works without `.fab-status.yaml`, without `fab/changes/`
- Read-only — modifies no files
- Idempotent — safe to invoke repeatedly
- Does not run preflight
- Does not output a `Next:` pipeline command — ends with "Ready to discuss. What would you like to explore?"

**Output**: Structured orientation summary with project identity, memory domains (with file counts), specs landscape, optional file status, active change name/stage (if any), and a ready signal.

---

## `/docs-hydrate-specs [domain]`

**Purpose**: Identify structural gaps between `docs/memory/` and `docs/specs/` and propose concise additions back to specs with interactive confirmation.

**Context**: `docs/memory/index.md`, `docs/specs/index.md`, all memory files, all spec files

**Arguments**:
- `[domain]` *(optional)* — scope to a single memory domain. Scans all domains if omitted.

**Example**:
```
/docs-hydrate-specs
→ "Found 5 structural gaps (showing top 3):"
→ Gap 1: Preflight Script — Source: preflight.md, Target: architecture.md
→ Shows exact markdown preview, asks: "Add this? (yes / no / done)"
```

**Behavior**:
1. Read all memory files to build a topic inventory (headings + summaries)
2. Read all spec files to build a coverage inventory (headings + inline mentions)
3. Cross-reference at section level — a gap is a memory topic with no spec coverage at all
4. Rank by impact (core behaviors > supporting concepts > implementation detail)
5. Present top 3 with exact markdown previews
6. Per-gap interactive confirm: yes (write), no (skip), done (stop)
7. Only confirmed additions are written to spec files

**Key properties**: No active change required. No git operations. Idempotent. Specs modified only with user confirmation.

---

## `/docs-reorg-memory`

**Purpose**: Analyze memory files across all domains for themes and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

**Context**: `docs/memory/index.md`, all domain indexes and memory files. Does NOT require `.fab-status.yaml`, config, or constitution.

**Prerequisite**: `docs/memory/index.md` must exist and `docs/memory/` must contain at least one domain with `.md` files besides `index.md`.

**Behavior**:
1. Read all memory files — extract headings, section summaries, approximate line counts
2. Identify themes (up to 10) with cohesion assessment (concentrated / scattered)
3. Diagnose current structure — what works, pain points, missing connections
4. Propose reorganization with migration map and updated index previews
5. User confirmation — apply all, cherry-pick specific migrations, or skip

**Key properties**: No active change required. No git operations. Idempotent. Memory files modified only with explicit confirmation.

---

## `/docs-reorg-specs`

**Purpose**: Analyze spec files for themes and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

**Context**: `docs/specs/index.md` and all spec files. Does NOT require `.fab-status.yaml`, config, or constitution.

**Prerequisite**: `docs/specs/index.md` must exist and `docs/specs/` must contain at least one `.md` file besides `index.md`.

**Behavior**:
1. Read all spec files — extract headings, section summaries, approximate line counts
2. Identify themes (up to 10) with cohesion assessment (concentrated / scattered)
3. Diagnose current structure — what works, pain points, missing connections
4. Propose reorganization with migration map and updated index preview
5. User confirmation — apply all, cherry-pick specific migrations, or skip

**Key properties**: No active change required. No git operations. Idempotent. Spec files modified only with explicit confirmation.

---

## `/git-pr [type]`

**Purpose**: Autonomously commit, push, and create a GitHub PR. No questions, no prompts. Covers stage 5 (Ship) of the pipeline.

**Arguments**:
- `[type]` *(optional)* — PR type prefix: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`. If omitted, type is inferred from `.status.yaml`, `intake.md`, or the diff (in that order).

**Example**:
```
/git-pr
→ "/git-pr — shipping to PR"
→ "  ✓ commit — Add loading spinner to submit button"
→ "  ✓ push   — origin/260101-abcd-add-spinner"
→ "  ✓ pr     — https://github.com/org/repo/pull/42"
→ "Shipped."
```

**Behavior**:
1. Resolve PR type (argument → `.status.yaml` → `intake.md` → diff → `chore`)
2. Check for uncommitted changes, unpushed commits, existing PR
3. Stage and commit any uncommitted changes (message matches repo style)
4. Push to remote (sets upstream if none)
5. Create a draft PR via `gh pr create` with title derived from intake and body including Summary, Changes, pipeline stats, and stage progress
6. Record PR URL in `.status.yaml`, mark ship stage done

**Key properties**:
- Requires `gh` CLI authenticated (`gh auth login`)
- Stops immediately on `main`/`master` branch — run `/git-branch` first
- Idempotent — skips steps already done (no PR created if one exists)
- Marks the `ship` stage done, auto-activates `review-pr`

**Context**: Does not require an active fab change — works as a standalone git tool. With an active change, reads `intake.md` for PR title/summary.

---

## `/git-pr-review [--tool <name>]`

**Purpose**: Process GitHub PR review comments on the current branch's PR. Handles feedback from any reviewer — human or bot. Covers stage 6 (Review-PR) of the pipeline.

**Arguments**:
- `--tool <name>` *(optional)* — force a specific review tool: `copilot`, `codex`, or `claude`. Bypasses the default cascade.

**Example**:
```
/git-pr-review
→ "3 comments triaged: 2 fix, 1 defer, 0 skip, 0 informational (no reply)"
→ "Fixed 2 comment(s) across 2 file(s)"
→ "Replied to 3 comment(s): 2 fix, 1 defer, 0 skip"
```

**Behavior**:
1. Resolve the PR for the current branch via `gh pr view`
2. **If no reviews exist** — request an automated review via cascade: Copilot (remote) → Codex (local) → Claude (local). First available tool wins. Re-run after Copilot to process its comments.
3. **If reviews with inline comments exist** — fetch all comments, triage each:
   - **fix**: applies a targeted code change, then posts `Fixed — {description}. ({sha})` as a reply
   - **defer**: posts `Deferred — {reason}.`
   - **skip**: posts `Skipped — {reason}.`
   - **informational**: no reply
4. Commit and push any fixes, then post all replies
5. Mark the `review-pr` stage done in `.status.yaml`

**Key properties**:
- Fully autonomous — never asks questions, never presents options
- Targeted fixes only — does not modify code beyond what each comment addresses
- Idempotent — re-running after fixes finds no new modifications; re-running after replies skips already-replied comments
- Cascade is configurable via `review_tools` in `fab/project/config.yaml`
