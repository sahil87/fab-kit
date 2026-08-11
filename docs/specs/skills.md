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

**Allowed values** (8): `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake`.

**Default**: omitted (or `[]`) — the skill loads only `_preamble`.

**Stage-conditional loading** (260611-zc9m): a skill MAY instead load a helper at its point of use via an explicit in-body read instruction; such a helper is intentionally absent from the frontmatter list. `/fab-continue` uses this for `_generation` (apply entry / intake regeneration) and `_review` (review stage).

**Example**:

```yaml
---
name: fab-ff
description: ...
helpers: [_generation, _review, _srad, _pipeline]
---
```

**Not allowed**: `_naming` and `_cli-rk` — their content is inlined into `_preamble.md` (§ Naming Conventions and § Run-Kit (rk) Reference respectively).

**Implicit**: `_preamble` itself is loaded universally — never list it.

**Current mapping** (post-260611-zc9m):

| Skill | `helpers:` |
|-------|------------|
| `fab-new`, `fab-draft`, `fab-dedupe` | `[_generation, _srad, _intake]` |
| `fab-ff`, `fab-fff` | `[_generation, _review, _srad, _pipeline]` (the shared bracket lives in `_pipeline.md`) |
| `fab-adopt` | `[_srad, _generation, _review, _pipeline]` (orchestrator — reuses the diff-generation procedures, diff-only review, and the auto-rework budget) |
| `fab-continue` | `[_srad]` (+ `_generation`/`_review` stage-conditionally, in-body) |
| `fab-clarify` | `[_srad]` |
| `fab-operator` | `[_cli-agents, _cli-fab, _cli-external]` (`_cli-agents` carries the generic agent-CLI interaction procedures extracted from the skill in 260805-nvad, plus the four-provider grammar/discovery dictionary) |
| All other skills | omitted (load only `_preamble`) |

Validation is **convention-only** — `fab sync` does not reject skills with unknown helper values. Drift surfaces as runtime behavior (agent loads an unexpected file or fails to find a needed one).


### Partial Flow Skeletons

The behavioral partials carry their condensed flow skeletons here; the pure-reference partials (`_cli-fab`, `_cli-external`, `_cli-agents`) carry no flow.

`_preamble` — shared context preamble loaded by every skill (path/context/helper conventions, per-stage profile resolution, cross-adapter dispatch, worker continuation, pane readiness gate, confidence scoring):

```text
Skill reads _preamble.md
├─ Context Loading (4 layers): always-load → change context (fab preflight, fab log command) → memory lookup (≤3-hop) → source code
├─ Conventions: helpers: frontmatter, naming, rk reference, common fab commands, next steps
├─ Subagent Dispatch
│  ├─ Dispatch pattern + Standard Subagent Context
│  ├─ Per-stage model resolution: fab resolve-agent <stage> --alias → branch on dispatch= (absent ⇒ native Agent / present ⇒ fab dispatch CLI adapter)
│  ├─ Worker continuation (apply rework only): native SendMessage / pane deliver / else fresh
│  ├─ CLI-adapter dispatch: fab dispatch start → wait; pane open → ready gate → deliver; stage-aware reap
│  └─ Dispatch-prompt obligations (all adapters): {stage}-result.yaml, context files, terminal fab status refresh, no status TRANSITIONs
├─ SRAD Autonomy Framework (pointer → _srad.md)
└─ Confidence scoring — fab score <change>
```

`_generation` — shared artifact generation procedures (Intake Generation, Plan Generation, and the diff-based adoption variants):

```text
Consumer reads _generation.md (via helpers: declaration)
├─ Intake Generation: read intake template → fill every section substantively → append ## Assumptions per _srad → write intake.md
├─ Plan Generation: read plan template → ## Requirements (RFC-2119, stable R#) → Task + Acceptance entry per requirement (traceability R# → T# → test → A#) → ## Tasks / ## Acceptance / ## Assumptions → write plan.md
└─ Adoption variants (fab-adopt only, diff read once): Intake-from-Diff / Plan-from-Diff (headings only; no R#/T#/A# ceremony)
```

`_intake` — shared pre-boundary Create-Intake Procedure (Steps 0–9) for /fab-new, /fab-draft, /fab-dedupe, and /fab-proceed's create-new dispatch:

```text
├─ 0 Parse input: Linear ID → MCP fetch / backlog ID → read fab/backlog.md / natural language as-is
├─ 1 Generate slug (2–6 word kebab)
├─ 2 Gap analysis
├─ 3 Create change: collision pre-checks → [existing] route to resume, STOP / else fab change new [--change-id] (+ fab status add-issue if Linear)
├─ 4 Mine conversation context → Certain/Confident assumption rows
├─ 5 Generate intake.md → _generation.md § Intake Generation
├─ 6 Verify change type — [if wrong] fab status set-change-type
├─ 7 Confidence — fab score --stage intake <change>
├─ 8 SRAD question selection: interactive → ask via SRAD / promptless-defer → "Deferred" rows returned to the dispatcher
└─ 9 Advance — fab status advance <change> intake
```

`_pipeline` — shared pipeline bracket for /fab-ff and /fab-fff (/fab-adopt partially consumes the rework loop + hydrate dispatch):

```text
Driver (fab-ff / fab-fff) reads _pipeline.md with {driver}/{terminal} bound
├─ Pre-flight: fab preflight; gate (skip if --force) fab score --check-gate --stage intake → STOP if < 3.0
├─ Resumability: skip done stages; per-stage dispatch via fab resolve-agent <stage> --alias (dispatch= absent ⇒ native Agent / present ⇒ CLI adapter)
├─ Step 1 Apply → subagent /fab-continue Apply → fab status finish intake/apply
├─ Step 2 Review → subagent /fab-continue Review (_review.md)
│  ├─ Pass: finish review → Step 3
│  └─ Fail: auto-rework loop ≤{max_cycles} (resume apply worker when reachable, fresh review each cycle); exhaustion: fab status fail review → STOP
├─ Step 3 Hydrate → subagent /fab-continue Hydrate → fab status finish hydrate
└─ {terminal} = hydrate → complete / review-pr → driver Steps 4–5
```

`_review` — shared review logic run by the dispatched review worker (a `mode` parameter — full | diff-only — selects whether plan-conformance steps run):

```text
Review worker reads _review.md at entry, runs the whole review inline
├─ Mode: full (default) / diff-only (holistic diff only)
├─ Preconditions [full]: plan.md tasks all [x], acceptance present
├─ Context: full diff (git diff <base>...HEAD), full repo access; full mode also plan.md + touched sources + memory
├─ Plan conformance [full]: acceptance items → run affected tests → spot-check requirements → memory drift → code quality → parsimony pass → deletion-candidate prompt
├─ Holistic-diff focus [both]: contract violations, pattern inconsistencies, missing cross-references, regressions, structural issues
└─ Verdict: one three-tier list (must/should/nice); any must-fix → fail, else pass
```

`_srad` — SRAD autonomy framework helper (four-dimension scoring, indicative grades, the Critical Rule, per-skill autonomy levels, artifact markers, Assumptions Summary format):

```text
Planning skill declares helpers: [..., _srad] → reads _srad.md before its body
├─ SRAD scoring: 4 dimensions 0–100 → composite → grade (Certain/Confident/Tentative/Unresolved)
├─ Critical Rule: genuine unknowns MUST be asked (promptless dispatch → "Deferred" rows)
├─ Skill-specific autonomy levels; worked examples
└─ Artifact markers (<!-- assumed --> / <!-- clarified -->) + ## Assumptions block (Scores column required)
```

---

## Context Loading Convention

Every skill that generates or validates artifacts MUST load relevant context before proceeding. This ensures agents produce accurate, grounded output rather than hallucinating requirements or ignoring existing patterns.

**Always loaded** — descriptive, not exhaustive: the layer applies unless the skill's own Context Loading section says otherwise (the skill file wins). Exceptions: `/fab-setup`, `/fab-switch`, `/fab-status`, and `/docs-hydrate-memory` skip it; `/fab-operator` loads only `config.yaml`, `constitution.md`, and `context.md`. The default layer:
- `fab/project/config.yaml` — project configuration: identity (name/description), `source_paths`/`test_paths`, true-impact excludes, plan-acceptance extra categories, independent provider capabilities (`interactive_command`, `headless_command`, and `native`) plus per-role fills (`providers:`), the two agent depth knobs (`agent.session`/`agent.workers`) and sparse `agent.profiles` overrides, `dispatch.mode` (the `pane → native → headless` preference ceiling), pane placement/reaping settings, optional `stage_hooks`
- `fab/project/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- `fab/project/context.md` — free-form project context: tech stack, conventions, architecture *(optional — no error if missing)*
- `fab/project/code-quality.md` — coding standards for apply/review: principles, anti-patterns, test strategy *(optional — no error if missing)*
- `fab/project/code-review.md` — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
- `docs/memory/index.md` — memory landscape (which domains and memory files exist)
- `docs/specs/index.md` — specifications landscape (pre-implementation design intent, human-curated)

**Change context** (loaded by skills operating on an active change):
- `.status.yaml` — current stage, progress
- All completed artifacts in the active change folder (`intake.md`, `plan.md`)

**Memory file lookup** (loaded by skills operating on an active change) — an up-to-3-hop walk, since a domain may be split into sub-domains:
- Read the intake's "Affected Memory" section to identify relevant domains (and sub-domains); entries are either flat (`{domain}/{file}`) or sub-domained (`{domain}/{sub-domain}/{file}`)
- Read domain indexes (`docs/memory/{domain}/index.md`) for each relevant domain
- If an entry is sub-domained, read the sub-domain index (`docs/memory/{domain}/{sub-domain}/index.md`) next
- Read the specific memory file(s) referenced by the Affected Memory entries (`docs/memory/{domain}/{file}.md`, or `docs/memory/{domain}/{sub-domain}/{file}.md` for a sub-domained entry)
- If a referenced file doesn't exist yet (listed under New Files), note this and proceed — it will be created by `/fab-continue` (hydrate)
- This grounds all artifact generation (plan, reviews) in the real current state, not assumptions

**Source code** (loaded during implementation and review):
- Read relevant source files referenced in the task descriptions
- Scope to files actually touched by the change — don't load the entire codebase

Each skill section below lists its specific context requirements under a **Context** field.

---

## Next Steps Convention

Skills MUST end their output with a `Next:` line suggesting the available follow-up commands, unless the skill's own Output or Key Properties section defines a different ending (e.g., `/fab-discuss`'s ready signal, `/fab-operator`'s status frame, the `/git-*` skills' own completion output) — the skill file wins, mirroring the context-loading contract. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab-command` or `Next: /fab-commandA or /fab-commandB (description)`

**Lookup table**:

| After | Stage reached | Next line |
|-------|---------------|-----------|
| `/fab-setup` | initialized | `Next: /fab-new <description>, /fab-proceed, or /docs-hydrate-memory <sources>` |
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

## New Skill Checklist

Adding a skill to the kit touches eight integration points. Work through all of them — drift in any one is invisible until an agent hits it.

1. **Frontmatter fields** — `name` (matches the filename) and `description` (the one-liner agents use for model invocation — name the actual behavior, including non-obvious modes like draft PRs or `--none` flags). Internal partials additionally set `user-invocable: false`, `disable-model-invocation: true`, and `metadata.internal: true`.
2. **Preamble-read line** — the body opens with the standard blockquote: ``> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.``
3. **`helpers:` declaration** — list any additional partials the skill needs (`_generation`, `_review`, `_cli-fab`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake`) in frontmatter; skills without the list load only `_preamble`. See § Skill Helpers.
4. **`Next:` line** — the skill's output ends with a state-derived `Next:` line per `_preamble.md` § Next Steps Convention (or documents an explicit opt-out, as `fab-discuss` and `fab-operator` do).
5. **Error Handling + Key Properties tables** — the body closes with the two standard tables (skill-specific errors only; idempotency, write surface, stage effects).
6. **Flow skeleton in skills.md** — add the skill's Flow skeleton to its `skills.md` section: the Flow diagram, plus the Tools and Sub-agents one-liners where they add information beyond the section's prose. Behavioral partials carry theirs in § Skill Helpers (§ Partial Flow Skeletons); pure-reference partials carry none.
7. **skills.md row** — add the skill's section to this file (and its `helpers:` row to § Skill Helpers when it declares any).
8. **Help grouping** — add the skill to `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` so `/fab-help` lists it under the right group (unmapped skills fall into the "Other" bucket).

---

## `/fab-setup`

**Purpose**: Bootstrap `fab/` in an existing project and manage ongoing configuration. Delegates structural setup to `fab sync` (the `fab-kit` Go binary) and adds interactive configuration on top. Safe to run repeatedly (idempotent). Also provides subcommands for config, constitution, and migrations.

**Prerequisite**: Fab Kit must be installed (`brew install fab-kit`) and `fab init` or `fab sync` must have been run in the project.

**Subcommands**:

| Subcommand | Purpose |
|------------|---------|
| `/fab-setup config [section]` | Create or update `fab/project/config.yaml` interactively, preserving comments |
| `/fab-setup constitution` | Create or amend `fab/project/constitution.md` with semantic versioning |
| `/fab-setup migrations [file]` | Run version migrations against the current project |

When called without arguments, `/fab-setup` runs the full bootstrap: invokes `fab sync` for structural setup, then delegates to `config` and `constitution` subcommands for any missing artifacts. Unrecognized arguments are rejected with a message listing valid subcommands.

**Creates** (idempotent — setup is re-runnable; whatever already exists is skipped):
- `fab/project/config.yaml` — project configuration (via `/fab-setup config`)
- `fab/project/constitution.md` — project principles and constraints (via `/fab-setup constitution`)
- `fab/.kit-migration-version` — migration version (via `fab sync`)
- `docs/memory/index.md` — initial memory index (via `fab sync`)
- `docs/specs/index.md` — specifications index (via `fab sync`)
- `fab/changes/` — empty, ready for change folders (via `fab sync`)
- `.claude/skills/` — deployed skill copies from the kit cache (via `fab sync`)

**Delegation pattern**: `fab sync` handles non-interactive structural setup
(directories, scaffolding, skill deployment, and `.envrc`/`.gitignore`
fragments). It does not register hooks or write `.claude/settings.local.json`;
migrations clean legacy hook entries. `/fab-setup` adds interactive config and
constitution work, while `fab sync` remains independently re-runnable.

**Examples**:
```
# First run — full bootstrap
/fab-setup
→ "Running fab sync... structure created."
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


**Flow**:

```text
User invokes /fab-setup [subcommand]
├─ Pre-flight: src/kit/ + VERSION exist; Bash: fab log command "fab-setup"
├─── No argument: Bootstrap
│  ├─ Bash: fab doctor → [non-zero] STOP
│  ├─ config.yaml: fab init --project seeds identity; refine in place
│  ├─ Read scaffold + project context → Write constitution.md
│  └─ Bash: fab sync (files, skill deploy, .gitignore) → [non-zero] STOP
├─── config: Read → interactive menu → Edit: fab/project/config.yaml
├─── constitution: Read → amendment menu → Edit: constitution.md
└─── migrations
   ├─ Bash: fab migrations-status --json (binary discovers; skill applies each file)
   ├─ [overlaps non-empty] STOP
   └─ Per applicable file: Read migration → execute → Write: fab/.kit-migration-version
```

**Tools**: Read (scaffolds, project files, migration files), Write (project files, migration version), Edit (config, constitution), Bash (`fab doctor`, `fab sync`, `fab log command`, `fab migrations-status --json`).

**Sub-agents**: None.

---

## `/docs-hydrate-memory [sources...]`

**Purpose**: Ingest external sources into `docs/memory/` with domain mapping and index maintenance. Safe to run repeatedly — content is merged into existing memory files as current truth without duplication (the affected topic section is rewritten to current truth, not appended as a change-keyed delta).

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


**Flow**:

```text
User invokes /docs-hydrate-memory [sources...|folders...|backfill]
├─ Read: _preamble.md; Pre-flight: docs/memory/ + index.md exist
├─ Ingest (URLs/.md): WebFetch/Read sources → Write topic files from template + index stubs → self-check → regen
├─ Generate (folders/none): Glob/Read codebase → gap report → Write from template + index stubs → self-check → regen
└─ Backfill (keyword / reorg dispatch): re-scan for missing description: → Edit: prepend frontmatter (body preserved, idempotent); regen deferred when reorg-dispatched
   (regen = Bash: fab memory-index --check refuse-guard → fab memory-index)
```

**Tools**: Read/Glob/Grep/WebFetch (sources, codebase scan, memory re-scan, `templates/memory.md` shape), Write/Edit (new memory files + index stubs; backfilled frontmatter), Bash (`fab memory-index --check` refuse-guard; `fab memory-index` regen).

**Sub-agents**: None — backfill mode is dispatched by `/docs-reorg-memory`; this skill spawns none.

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


**Flow**:

```text
User invokes /fab-new <description>
├─ Read: _preamble.md, .claude/skills/_intake/SKILL.md (+helpers)
├─ Create-Intake Procedure Steps 0–9 (interactive — see `_intake.md`)
├─ Bash: fab change switch "{name}"
└─ Create Git Branch (same cases as git-branch): probe repo/branch/dirty/target-exists/rename-guard → checkout, checkout --track, checkout -b, or branch -m; dirty tree on create/rename → non-blocking note
```

**Tools**: Steps 0–9 per `_intake.md`; own tail: Read (`_intake` skill + helpers), Bash (`fab change switch`, `fab resolve`, `git`).

**Sub-agents**: None.

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

**Behavior**: A thin delta over `/fab-new` — reads its deployed skill file and executes its Pre-flight, Arguments, and Steps 0–9 with the documented deltas, skipping Steps 10–11 entirely (no activation, no git branch). The user must run `/fab-switch {name}` to make it active before proceeding.


**Flow**:

```text
User invokes /fab-draft <description>
├─ Read: _preamble.md, .claude/skills/_intake/SKILL.md (+helpers)
└─ Create-Intake Procedure Steps 0–9 (interactive — see `_intake.md`); STOP after Step 9 (no activation, no branch)
```

**Tools**: Per the Create-Intake Procedure (see `_intake.md`). No `fab change switch`, no git.

**Sub-agents**: None.

---

## `/fab-continue [<stage>]`

**Purpose**: Advance the active change one pipeline stage — intake, apply (co-generates `plan.md` at entry then runs tasks), review, hydrate, ship (delegates to `/git-pr`), or review-pr (delegates to `/git-pr-review`). Or, when called with a stage argument, reset to that stage and re-run from there.

**Arguments**:
- `<stage>` *(optional)* — target stage to reset to (`apply` is the typical reset). The legacy `tasks` and `spec` targets are removed and error with a pointer to the `apply` and `intake` reset routes (`/fab-continue apply` to re-run apply — delete `plan.md` first to force regeneration — or `/fab-continue intake` then `/fab-clarify` to rework the intake). Used after `/fab-continue` (review) identifies issues upstream. When provided, resets `.status.yaml` to this stage and re-runs from that point forward.

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
2. Intake in `ready` state: finish intake (auto-activates apply), then execute apply
3. Intake in `active` state (backward compat): generate intake if missing, advance to `ready`
4. For execution stages: execute the stage's behavior and finish it
5. Load relevant template + context (including `fab/project/constitution.md` for project principles)
6. Apply entry: invoke the unified Plan Generation Procedure — co-generate `plan.md` `## Requirements` (from `intake.md`) + `## Tasks` + `## Acceptance` (skipped on resume when `plan.md` already exists)
7. Update `.status.yaml`

**Behavior** (with stage argument — reset and regenerate):
1. **Guard**: target stage must be a valid 6-pipeline stage (typically `apply`). Reset to `tasks` or `spec` errors with `"tasks"/"spec" stages were removed — use /fab-continue apply to re-run apply (delete plan.md first to force regeneration), or /fab-continue intake then /fab-clarify to rework the intake.`
2. Reset `.status.yaml`: the target stage → `active`; all stages **after** it → `pending` (stages before the target are preserved). Non-resettable current states are handled first (reset From-set is `{done, ready, skipped}`): target already `active` → skip the call and proceed (a reset re-run is a state-wise no-op); target `failed` → handled by the failed dispatch rows instead (`start` owns failed→active, review/review-pr only); target `pending` → error with advance guidance.
3. For an intake reset, regenerate the intake artifact in place; for execution resets, re-run from that stage.
4. Downstream artifacts are invalidated only by re-running apply: `plan.md` persists across resets (deleting it forces regeneration); task checkboxes are NOT auto-cleared.
5. For an intake reset, advance intake to `ready` (not `done`) to preserve the `/fab-clarify` opportunity.


**Flow**:

```text
User invokes /fab-continue [change-name] [stage]
├─ Read: _preamble.md; Bash: fab preflight
├─ [reset arg] Bash: fab status reset <change> <stage>
├─ [review-failed] reset apply + rework menu, stop; [review-pr-failed] re-run /git-pr-review
├─ Dispatch on current stage:
│  INTAKE (main session): Read templates/memory → Write intake.md (SRAD) → advance; finish intake
│  APPLY (dispatched): no plan.md → Write plan.md; per unchecked task: Edit/Write sources → Bash: tests → check off plan.md → finish apply
│  REVIEW (dispatched; worker reads _review.md, runs review inline): read diff/plan/source/memory → tests → unified findings
│    pass → finish review + set-acceptance / fail → fail review + reset apply (rework options)
│  HYDRATE (dispatched): Write/Edit docs/memory/** → set-summary → fab memory-index → finish hydrate
│  SHIP: delegate to /git-pr <change>
│  REVIEW-PR: delegate to /git-pr-review <change> (timeout → stage left active)
└─ Output: summary + Next: line
```

**Tools**: Read (preamble, templates, artifacts, source, memory), Write (`plan.md`, memory files), Edit (plan checkboxes, memory), Bash (`fab status` transitions, `fab preflight`, `fab memory-index`, tests), Agent (review sub-agent).

**Sub-agents**: Single review sub-agent (reads `_review.md`, runs the review inline; returns one unified findings set).

---

## `/fab-ff` (Fast Forward)

**Purpose**: Fast-forward apply → review → hydrate (everything after intake). Gated on the single intake confidence gate (flat 3.0), with sub-agent review, auto-rework loop (up to `{max_cycles}` cycles — the code-review.md Rework Budget knob, default 3 — with prioritized findings), and stop on exhaustion. Accepts `--force` to bypass the gate. No `/fab-clarify` runs inside the bracket.

**Source ownership**: `_pipeline.md` owns the shared arguments, framing, output
skeleton, Steps 1–3, and Stage Dispatch Procedure. `fab-ff.md` binds only the
`fab-ff`/`hydrate` driver delta.

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
3. **Review** — dispatch to a single sub-agent (fresh context). The sub-agent returns prioritized findings (must-fix / should-fix / nice-to-have); it inspects items under `plan.md` `## Acceptance` against `## Requirements` and judges the diff on its own merits
4. **On pass** — advance to hydrate
5. **On fail** — auto-rework loop (up to `{max_cycles}` cycles, default 3): triage findings by priority, autonomously select rework path (fix code, revise plan, revise requirements), re-apply (resume-first: continue the named `apply-{id}` worker on the native arm when reachable, else dispatch fresh), spawn fresh sub-agent for re-review. Escalation after 2 consecutive fix-code attempts. Stop after `{max_cycles}` failed cycles with summary.
6. Hydrate into `docs/memory/`


**Flow**:

```text
User invokes /fab-ff [change-name] [--force]
├─ Read: _preamble.md, helpers incl. _pipeline.md
└─ Execute the _pipeline.md bracket ({driver}=fab-ff, {terminal}=hydrate) — see `_pipeline.md`
```

**Tools**: Read (`_preamble.md`, helpers); all other tool use lives in the bracket (see `_pipeline.md`).

**Sub-agents**: Per the bracket: `/fab-continue` Apply, Review (via `_review.md`), Hydrate.

---

## `/fab-fff` (Full Autonomous Pipeline)

**Purpose**: Run the entire automated Fab pipeline — apply → review → hydrate → ship → review-pr — in a single invocation (everything after intake). Gated on the single intake confidence gate (flat 3.0, same as `/fab-ff`). No `/fab-clarify` runs inside the bracket. Autonomously reworks on review failure using sub-agent review with prioritized findings (`{max_cycles}`-cycle retry cap — code-review.md Rework Budget knob, default 3 — escalation after 2 consecutive fix-code failures). Accepts `--force` to bypass the gate.

**Source ownership**: `_pipeline.md` owns shared framing and Steps 1–3;
`fab-fff.md` owns only the `fab-fff`/`review-pr` binding plus ship and PR review.

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
4. **Step 2 — Review**: Dispatch to review sub-agent (fresh context, prioritized findings). On failure, triage findings by priority and autonomously select rework path (fix code, revise plan, revise requirements), then re-apply resume-first (continue the named `apply-{id}` worker on the native arm when reachable, else dispatch fresh). Re-review via fresh sub-agent. Retry up to `{max_cycles}` cycles (default 3; escalation after 2 consecutive fix-code). Bail with summary after `{max_cycles}` failed cycles.
5. **Step 3 — Hydrate**: Hydrate into memory.
6. **Step 4 — Ship**: Dispatch `/git-pr` to commit, push, and create PR.
7. **Step 5 — Review-PR**: Dispatch `/git-pr-review` to process PR review comments.

**Key difference from `/fab-ff`**: The difference is scope only. `/fab-fff` extends through ship and review-pr; `/fab-ff` stops at hydrate. Both have the identical single intake gate, no in-bracket clarify, and identical auto-rework (`{max_cycles}`-cycle cap with escalation, default 3). Both accept `--force` to bypass the gate.


**Flow**:

```text
User invokes /fab-fff [change-name] [--force]
├─ Read: _preamble.md, helpers incl. _pipeline.md
├─ Execute the _pipeline.md bracket ({driver}=fab-fff, {terminal}=review-pr)
├─ Ship: dispatch /git-pr {name} (own ship transitions)
└─ Review-PR: dispatch /git-pr-review {name}
   ├─ [success / no-reviews] stage done; [failure] STOP with the error
   └─ [timeout] stage left active; report pending + re-run guidance
```

**Tools**: Read (`_preamble.md`, helpers); all other tool use lives in the bracket and the dispatched skills.

**Sub-agents**: Bracket sub-agents per `_pipeline.md` (`/fab-continue` Apply, Review, Hydrate) plus `/git-pr` and `/git-pr-review`.

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
1. **State detection** — 5-step pipeline: (1) active change check (`fab resolve --folder --or-none` — branch on the output token: a folder name = active change, `(none)` = none; a non-zero exit is a real error, not the no-active-change state), (2) branch check (`git branch --show-current`, runs only if active change found), (3) conversation classification as substantive/empty-thin, (4) unactivated intake scan (`fab/changes/`, retain full candidate list), (5) dispatch decision combining Steps 1–4 via the 7-row dispatch table. Steps 3 and 4 are order-independent and both run whenever no active change was found.
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


**Flow**:

```text
User invokes /fab-proceed
├─ Bash: fab resolve --folder --or-none
│  ├─ folder → branch check: matches → dispatch /fab-fff only / mismatch → dispatch /git-branch → /fab-fff
│  └─ "(none)" → classify conversation substantive vs empty/thin
├─ Scan unactivated intakes; substantive + candidates → relevance assessment (clearly-relevant wins; ambiguous → not relevant; date-descending tiebreak)
├─ Prefix dispatch (subagents): _intake Steps 0–9 in promptless-defer mode (no questions; unresolved decisions deferred, surfaced before /fab-fff) → /fab-switch → /git-branch
└─ Terminal delegation: Skill: /fab-fff (main context, NOT subagent)
```

**Tools**: Bash (`fab resolve --folder --or-none`, `git branch --show-current`, intake scan), Agent (prefix dispatches), Skill (`/fab-fff`).

**Sub-agents**: `_intake` (Create-Intake, promptless-defer — stops at `ready`), `/fab-switch` (activate), `/git-branch` (branch) — dispatched per the dispatch table.

---

## `/fab-adopt`

**Purpose**: Bring a **completed-but-off-pipeline** change into the Fab pipeline (scenario B — a feature branch authored without fab, with an **OPEN** or **not-yet-created** PR). It is the *real* pipeline entered late, with **apply** marked `skipped` (the only stage that cannot meaningfully re-run when the code already exists); intake/review/hydrate/ship/review-pr all genuinely run. A **MERGED** PR (scenario A — retroactive backfill) is out of scope and STOPs at Step 0. A thin orchestrator on the `/fab-proceed`/`/fab-ff` pattern — `helpers: [_srad, _generation, _review, _pipeline]`.

**Prerequisite**: An active branch (not detached HEAD, not the default branch) with a non-empty diff against the default-branch merge-base, and no fab change already mapping to that branch.

**Context**: config, constitution; the branch diff (`git diff {base}...HEAD`) and PR body — read once in the one main-session generation pass.

**Behavior**:
1. **Step 0 — Guards & diff base**: reuse `/git-pr`'s guard idioms — STOP on detached HEAD / default branch / MERGED PR (scenario A) / branch-already-maps-to-a-change (point at `/fab-continue`) / empty diff. `OPEN` and `none` PR states proceed. Resolve `base=$(git merge-base HEAD origin/{default})` and capture the diff.
2. **Steps 1+2 — one main-session generation pass** (same agent, not dispatched): `fab change new --slug {slug}` + activate (branch exists — `/fab-new` Step 11 row 1/2); reconstruct `intake.md` via the **Intake-from-Diff Procedure** (`_generation.md`); **human-confirmation checkpoint** (confirm/correct the reconstructed intent — the late deliberation the bypass skipped) → `fab status advance/finish {name} intake`; write a deliberately MINIMAL `plan.md` via the **Plan-from-Diff Procedure**.
3. **Step 2 (state)**: `fab status skip {name} apply` (cascades downstream → skipped) then `fab status reset {name} review fab-adopt` (skipped → active, downstream → pending) — yields `apply=skipped, review=active`, **no Go change**; record the fact via `fab status set-summary`.
4. **Step 3 — Review** (dispatched, `mode: diff-only` — the `_review.md` parameter): the orchestrator owns the verdict (pass incl. zero-findings best-effort → `finish review`; fail → auto-rework per `_pipeline.md` budget when autonomous, hand findings back when interactive).
5. **Step 4 — Hydrate** (dispatched, verbatim per `_pipeline.md` Step 3): the permanent-loss recovery — `docs/memory/` finally reflects what shipped → `finish hydrate`.
6. **Step 5 — Ship**: `/git-pr {name}` retrofits `## Meta` onto the OPEN PR (its Step 3d, gated on body-lacks-`## Meta`) or creates the PR fresh when `none`; `finish ship` auto-activates review-pr.
7. **Step 6**: land in review-pr; print the honest-state summary and `Next: /git-pr-review`.

**Key properties**:
- Only **apply** is `skipped`; every other stage runs for real (just late)
- Diff-only review via the general `mode` parameter on `_review.md` — not an adopt-specific branch
- State composed from existing `skip`/`reset` transitions; PR Meta retrofit reuses `fab pr-meta` + `gh pr edit` — **no Go change**
- Idempotent guards: re-run after the change is created routes to `/fab-continue` via the collision guard; the Meta retrofit is body-gated


**Flow**:

```text
User invokes /fab-adopt [<slug>]
├─ Guards (STOP before any mutation): detached/default branch, MERGED PR, already in pipeline, or empty diff vs base
├─ Intake (one main-session pass): fab change new + activate; reconstruct intake.md from the diff; human confirmation; write MINIMAL all-[x] plan.md
├─ Bash: fab status skip apply / reset review / set-summary
├─ Review (dispatched, mode: diff-only): pass → finish review / fail → auto-rework or hand back
├─ Hydrate (dispatched) → finish hydrate
├─ Ship: dispatch /git-pr {name} (retrofit existing PR or fresh PR)
└─ Land in review-pr → summary + Next: /git-pr-review
```

**Tools**: Bash (`git`, `gh pr view`, `fab change new`, `fab status`, `fab score`), Read (diff, PR body, templates), Write (`intake.md`, `plan.md`), Agent (review + hydrate, `/git-pr`).

**Sub-agents**: `/fab-continue` Review (`mode: diff-only`), `/fab-continue` Hydrate, `/git-pr {name}` (ship).

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


**Flow**:

```text
User invokes /fab-clarify [change-name]  — or —  [AUTO-MODE] invocation
├─ Read: _preamble.md; Bash: fab preflight
├─ Target is intake.md; at apply or later → STOP (→ /fab-continue rework)
├─── SUGGEST MODE (interactive)
│  ├─ Read intake.md → taxonomy scan (gaps, markers)
│  ├─ Bulk-confirm confident items → Edit: intake.md
│  ├─ Ask questions, process answers → Edit: intake.md (append ## Clarifications)
│  └─ Coverage summary → Bash: fab score --stage intake
├─── AUTO MODE (machine-readable)
│  ├─ Scan + resolve contextual gaps; mark blocking user-input gaps
│  └─ Return {resolved, blocking, non_blocking, blocking_issues?} → Bash: fab score
└─ Does NOT advance stage
```

**Tools**: Read (preamble, artifacts, memory), Edit (intake.md in place), Bash (`fab preflight`, `fab score`).

**Sub-agents**: None.

---

## `/fab-dedupe [scope]`

**Purpose**: Sweep a scoped area for duplicated utilities, cluster them by behavioral shape, and draft one change intake per accepted cluster group. Read-only until the user accepts clusters. It **does not refactor** — the consolidation runs through the normal pipeline with review in the loop.

**Context**: config, constitution, `docs/memory/_shared/utilities.md` (if present — so the sweep does not re-propose finished work). No change artifacts — there is no active change. Runs **no** `fab preflight`.

**Creates**: 0..N change folders with `.status.yaml` + `intake.md`, each at intake `ready`. No activation, no git branch. Zero is a valid outcome.

**Arguments**:
- `[scope]` *(optional)* — natural language (`"test setup helpers in src/go"`), or a path/glob. Natural language resolves against `source_paths`/`test_paths` and is confirmed with the user; a bare invocation defaults to `source_paths` with a length warning. Scope filters *where clusters may be found*, not *where their members may live* — out-of-scope members of an in-scope cluster MUST be reported.

**Examples**:
```
/fab-dedupe src/go
→ Consolidation sweep — src/go · Detectors: jscpd (skipped — not installed)
→ 2 clusters. Which should become changes? (all / 1,3 / none)
→ Drafted 1 change: 260728-a1b2-consolidate-test-fixtures  Confidence: 4.2 / 5.0
```

**Behavior** (5 steps, then the shared procedure):
1. **Resolve scope** → concrete paths; echo the resolved set and file count.
2. **Run detectors** — `consolidate.detectors` from project config (default: jscpd alone). Each is probed with `command -v` and **skipped silently when absent** (the `rk` fail-silent discipline); a **non-zero exit is a finding, not a STOP** — an explicit per-skill exception to `_preamble.md` § Common fab Commands' failure rule, which governs `fab` commands rather than third-party tools. `{paths}` and `{out}` are substituted before execution as **shell-quoted** values, so a scope path containing a space or a shell metacharacter stays one intact argument.
3. **Cluster by behavioral shape** — seeded by detector output but not limited to it. Per cluster: members, shared behavior, divergences, canonical home, call sites, out-of-scope members, and a **layered decomposition** — a shared core plus named opt-in variation layers with their member lists, and the unified API expressing that shape. Never one flat signature per cluster. Do not cluster on name similarity; a cluster of one is not a cluster.
4. **Rank and present** — call sites ↑ / divergence ↓, where a member needing only the base layer counts as **LOW** divergence even when its body reads differently. Then ask which clusters to act on as a **plain conversational reply** (`all` / `1,3` / `none`) — not a structured multi-select.
5. **Draft intakes** — per accepted cluster group, execute the `_intake` **Create-Intake Procedure** (Steps 0–9, `{questioning-mode} = interactive`), preferring one intake per cluster, and **STOP after Step 9**.

**Configuration**: one registry key, `consolidate.detectors` (project scope, advertised, no built-in default). The utilities memory home is **hardcoded** to `docs/memory/_shared/utilities.md` — there is no override key, and the skill never writes the file (hydrate does, on the normal pipeline path).

**Key property**: read-only until Step 5; the sweep is idempotent but drafting is not (natural-language input creates a fresh change each run) — Step 2's gap analysis is the duplicate-work guard.


**Flow**:

```text
User invokes /fab-dedupe [scope]
├─ Pre-flight: config.yaml + constitution.md exist (no fab preflight); Read: _preamble.md
├─ Bash: fab log command "fab-dedupe"
├─ Resolve scope → paths; probe + run configured detectors (fail-silent)
├─ Cluster by behavioral shape; rank → report → ASK (all / 1,3 / none)
└─ Per accepted group → Create-Intake Procedure Steps 0–9 (see `_intake.md`); STOP after Step 9 (no activation)
```

**Tools**: Read (memory, source), Bash (`fab log command`, detector probes), Write (`intake.md` via the shared procedure). No `fab preflight`, no `fab change switch`, no git.

**Sub-agents**: None.

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

**Pass/fail** (deterministic): If any must-fix findings exist, the review fails. No must-fix findings (including zero findings) → the review passes; should-fix and nice-to-have findings are reported but never block.

**On failure** (manual rework in `/fab-continue`), the findings are presented with priority annotations and the user chooses where to loop back:

- **Fix code** → `/fab-continue` (apply)
  Implementation bug. The agent identifies which tasks need rework, unchecks them in `plan.md` `## Tasks` (marks `- [ ]` again with a `<!-- rework: reason -->` comment), and re-runs `/fab-continue` which picks up the unchecked items.

- **Revise plan** → edit `plan.md`, then `/fab-continue` (apply)
  Missing or wrong tasks/acceptance items. The agent adds/modifies entries in `plan.md` (new tasks get the next sequential ID; new acceptance items use the next `A-NNN`). Completed tasks that are unaffected stay `[x]`. Only new or revised tasks are executed.

- **Revise requirements** → edit `plan.md` `## Requirements`, then `/fab-continue` (apply)
  Requirements were wrong or incomplete. The agent edits the `## Requirements` section plus the downstream `## Tasks`/`## Acceptance` it affects, then re-runs apply. For a fundamentally wrong intake, run `/fab-continue intake` first (resets to intake and regenerates it), refine via `/fab-clarify`, and delete `plan.md` so re-entering apply re-derives `## Requirements` from the revised intake — `plan.md` is otherwise preserved on reset; there is no automatic regeneration.

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
→ "Archived to fab/changes/archive/2026/01/260115-a7k2-add-oauth/"
→ "Next: /fab-new <description>"
```

**Behavior** — the skill delegates all mechanical operations to a single `fab change archive <change>` call and formats its YAML output into the report:
1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{yyyy}/{mm}/{name}/` (date-bucketed by the folder's embedded date). No rename.
2. **Delete dispatch state** — remove the change's `.fab-dispatch/{id}/` dispatch state dir (shared by both `fab dispatch` modes — headless and pane; transient comms, not history; one of the two `fab dispatch` cleanup paths, **not recreated on restore**; best-effort — an absent dir is a no-op).
3. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first. Description derived from the intake title (humanized-slug fallback).
4. **Mark backlog item done** — exact change-ID match in `fab/backlog.md` (`- [ ]` → `- [x]`), in place; reported as `marked`/`already`/`not_found`.
5. **Clear pointer** — remove `.fab-status.yaml` symlink only if the archived change is the active one.

**Order of operations**: the Go command executes move → dispatch-state deletion → index → backlog → pointer. Re-archiving an already-archived change is a soft skip (exit 0) that still re-attempts the backlog mark; interrupted runs are recovered by re-running.

**Restore mode** (`/fab-archive restore <change-name> [--switch]`): Moves an archived change back to `fab/changes/`. Preserves all artifacts and `.status.yaml` without modification. Optionally activates via `--switch` flag.


**Flow**:

```text
User invokes /fab-archive [change-name | restore <name> [--switch]]
├─ Read: _preamble.md
├─ Archive: Bash: fab preflight → [hydrate not done] STOP → Bash: fab change archive <change>
├─ Restore: Bash: fab change restore <name> [--switch] (preflight + hydrate guard waived)
└─ Format report (index/pointer results; failed = op done, exit non-zero)
```

**Tools**: Read (preamble), Bash (`fab preflight`, `fab change archive`, `fab change restore`, `fab change archive-list`).

**Sub-agents**: None.

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


**Flow**:

```text
User invokes /fab-switch [change-name] [--none]
├─ Read: _preamble.md (§1 always-load exception)
├─ No argument → Bash: fab change list → user selects; --none → fab change switch --none
└─ change-name → Bash: fab change switch "<name>" ([multi-match] ask; [no match] list) → Bash: fab log command
```

**Tools**: Bash (`fab change switch`, `fab change list`, `fab log command`).

**Sub-agents**: None.

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
   - **On other branch, no upstream** → rename guard: rename the current branch (`git branch -m`) only when it resolves to no other change (`fab change resolve <current-branch>` fails); if it belongs to another change, create a new branch instead (`git checkout -b`, leaving the other change's branch intact — caveat: the new branch inherits its HEAD)
   - **On other branch, has upstream** → create new branch (leaving current intact)
5. Report result

**Key properties**:
- Does not modify `.fab-status.yaml` or `.status.yaml`
- Idempotent — checking out an already-active branch is a no-op
- Always enabled if in a git repo


**Flow**:

```text
User invokes /git-branch [change-name]
├─ Bash: git rev-parse --is-inside-work-tree; fab change resolve "<name>" → [multi-match] STOP with candidates / [no match, explicit arg] standalone fallback
├─ Probes (current branch, dirty count, local + origin existence) → [on target] no-op / [local] checkout / [origin-only] checkout --track / [on main] checkout -b
├─ [other branch, no upstream] rename guard: resolve current branch → branch -m (same/no change) or checkout -b (different change)
└─ Report; create/rename with dirty tree → carried-over note
```

**Tools**: Bash — `fab change resolve` (resolution + rename guard; strict exit-code form kept deliberately); all git operations.

**Sub-agents**: None.

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


**Flow**:

```text
User invokes /fab-status [change-name]
├─ Bash: fab preflight; Read: kit VERSION + fab/.kit-migration-version; Bash: git branch --show-current
└─ Render: stage progress table, plan counts, confidence, impact line (⚠️+bold over threshold), refactor-growth warning
```

**Tools**: Bash (`fab preflight`, `git branch --show-current`), Read (VERSION, migration-version).

**Sub-agents**: None.

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


**Flow**:

```text
User invokes /fab-discuss
├─ Read: always-load layer (per _preamble.md §1)
├─ Bash: fab resolve --folder --or-none; Read: .status.yaml if a change is active
├─ Bash: fab log command "fab-discuss"
└─ Output: orientation summary
```

**Tools**: Read (always-load files, `.status.yaml`), Bash (`fab resolve --folder --or-none`, `fab log command`).

**Sub-agents**: None.

---

## `/fab-help`

**Purpose**: Orient the user in the fab workflow — show how the stages fit together and list every deployed command with a one-line description: the `/fab-*`, `/git-*`, and `/docs-*` skills grouped by category, plus the `fab batch` operations and the companion packages (wt, idea).

**Context**: None. `/fab-help` loads no config, constitution, or change artifacts.

**Behavior**: Log the invocation (`fab log command "fab-help"`), then shell out to `fab fab-help`. The subcommand reads the kit version from the cache (falling back to `unknown` when absent), scans skill frontmatter for command descriptions, and prints the complete help text. **The subcommand is the single source of truth for help content** — the skill does not restate it, so a newly added skill surfaces automatically once it is mapped in `skillToGroupMap` (§ New Skill Checklist item 8).

**Key properties**: Read-only — creates and modifies nothing. Idempotent. Requires no active change, no config, and no constitution.


**Flow**:

```text
User invokes /fab-help
├─ Bash: fab log command "fab-help"
└─ Bash: fab fab-help (scans skill frontmatter, prints grouped help)
```

**Tools**: Bash (`fab log command`, `fab fab-help`).

**Sub-agents**: None.

---

## `/fab-operator`

**Purpose**: Multi-agent coordination layer. Runs in a dedicated tmux pane, observes agents across every session on its tmux server via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, auto-answers routine prompts, drives autopilot queues, and spawns dependency-aware agents. Started via `fab operator` (a singleton tmux tab named `operator`, one per tmux server).

**Context**: A deliberate exception to the always-load layer — loads only `config.yaml`, `constitution.md`, and `context.md` (optional). It runs no `fab preflight` and never reads change artifacts, keeping a long-lived context window reserved for coordination state. Declares `helpers: [_cli-agents, _cli-fab, _cli-external]`.

**Key properties**:
- **Coordinates, never executes** — all pipeline work is spawned into a freshly created worktree agent (`wt create --non-interactive`), never run in the operator's own pane. Operational maintenance (merge PR, archive, delete worktree) is the one direct-execution exception.
- **Pipeline-first routing** — new work always enters through `/fab-new` then a pipeline command; raw inline implementation instructions are never dispatched to agent panes.
- **State is re-derived, never remembered** — live state is re-queried before every action; continuity across `/clear` comes from the server-keyed operator state file.
- Ends with its own status frame rather than a `Next:` line.

**Full spec**: [operator.md](operator.md) (the top-level behavioral spec). The agent-CLI mechanics it builds on — spawn composition, pre-send validation, delivery probe, peek, await — live in `_cli-agents.md`; this skill owns *when and whether* to use them (confirmation tiers, retry budgets, repo targeting, enrollment, dependency resolution, autopilot).


**Flow**:

```text
Started via `fab operator`; runs a continuous /loop cycle
├─ Re-derive state each cycle: Bash: fab pane map --all-sessions (never trust cached values)
├─ Auto-answer routine agent questions; nudge stalled agents; route commands via tmux send-keys
└─ Drive autopilot queues (/fab-new → /fab-fff); spawn each task in a fresh worktree
```

**Tools**: Bash (`fab pane map`, `tmux send-keys`, `wt create`), Skill (`/loop`); helpers `_cli-agents`, `_cli-fab`, `_cli-external`.

**Sub-agents**: None (spawns agent sessions, not sub-agents).

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


**Flow**:

```text
User invokes /docs-hydrate-specs [domain]
├─ Read: _preamble.md; pre-flight: memory + specs indexes exist
├─ Read: all memory + spec files → identify gaps, rank top 3
└─ Per gap: preview → yes/no/done → Edit: existing spec, or Write: new file + Edit: specs/index.md row; summary {N} of {M} applied
```

**Tools**: Read (memory files, spec files, indexes), Edit/Write (approved spec edits; new spec file + index row for no-target gaps).

**Sub-agents**: None.

---

## `/docs-reorg-memory`

**Purpose**: Analyze memory files across all domains for themes and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

**Context**: `docs/memory/index.md`, all domain indexes and memory files. Does NOT require `.fab-status.yaml`, config, or constitution.

**Prerequisite**: `docs/memory/index.md` must exist and `docs/memory/` must contain at least one domain with `.md` files besides `index.md`.

**Behavior**:
1. Read all memory files — extract headings, section summaries, approximate line counts. One `fab memory-index --check --json` call feeds three consumers: `losses[]` (compatibility detection), `warnings[]` `file-size` (Shape Report over-size file rows), `warnings[]` `unsorted-nonempty` (`_unsorted/` triage); older-binary ⇒ prose fallback / read-pass line counts / folder listing
2. Identify themes (up to 10) with cohesion assessment (concentrated / scattered); detect **duplicate coverage** — the same topic in 2+ files (near-identical filenames/descriptions, same filename in two domains, heavy heading overlap) → `## Duplicate Coverage` table (remediation: `merge-file` or `move-section`; cross-references the open single-sourcing seam audit, not scope)
3. Diagnose current structure — a **Shape Report** flagging over-width/over-depth/under-floor **folders AND over-size topic files** (~400 lines / ~15KB → `split-file` candidates when ≥2 topic clusters; long-but-cohesive reported, not split); an `_unsorted/` triage (per-file `move`-to-domain default / `delete` with per-file confirmation; `_unsorted/` keeps its bounds exemption)
4. Propose reorganization with a Migration Map (`Kind` ∈ `move-section` / `split-domain` / `merge-domain` / `flatten` / `move` / `split-file` / `merge-file`) + a Link Impact note for every move-bearing migration. `split-file` fans one multi-topic file into ≥2 topic files (verbatim bodies — restyling stays `/docs-distill-memory`'s; new `type: memory` + change-id-free `description:`; anchored inbound links follow their heading, un-anchored retarget to the dominant-topic file, ambiguity → abort escape). `merge-file` folds a duplicate-coverage file into a canonical sibling
5. User confirmation — apply all, cherry-pick specific migrations, or skip (a `delete` needs explicit per-file confirmation)
6. **Completion chain → `/docs-distill-memory`**: at completion (whether or not any migration ran), emit `Next: /docs-distill-memory (N files flagged across M domains)` (listed first) when N ≥ 1 — the fixed *structure-then-prose* order made self-guiding. The chain **reuses the Step 1 `--check --json` call's `warnings[]`** (no second call) and aggregates with distill's four-kind rule (dedupe by path, sub-domain roll-up, exclusion set). N = 0 → the normal completion `Next:`; older-binary (no `warnings[]`) → a plain pointer without counts. Distill points back at reorg in its own `Next:` line (the bidirectional chain).

**Key properties**: No active change required. No git operations. Idempotent (a well-shaped tree with no over-size files, duplicate coverage, or `_unsorted/` staging proposes nothing). Memory files modified only with explicit confirmation. Owns structure at **file** granularity (`split-file`/`merge-file`) as well as folder; body-prose restyling belongs to `/docs-distill-memory`. Completion chains to `/docs-distill-memory` (reusing the Step 1 `--check --json` call, no second call) so the fixed reorg → distill composition order is self-guiding.


**Flow**:

```text
User invokes /docs-reorg-memory
├─ Pre-flight; Read: all memory files + one Bash: fab memory-index --check --json (feeds compatibility, shape, _unsorted/, completion chain)
├─ Diagnose: Shape Report + compatibility + duplicate coverage + _unsorted/ triage → propose reorg → approval gate
├─ [compat approved] Write: _shared/removed-domains.md; dispatch /docs-hydrate-memory backfill
└─ [approved] per migration: Write/Edit moves/splits/merges + link rewrites → Bash: fab memory-index → verify (no lost headings/dangling links) → Next: /docs-distill-memory with flagged counts
```

**Tools**: Read (all memory files and indexes), Write/Edit (approved moves/splits/merges, link rewrites, tombstone file, `_unsorted/` triage), Bash (one `fab memory-index --check --json` — four consumers; `fab memory-index` regen), Agent (dispatch `/docs-hydrate-memory` backfill during compatibility orchestration).

**Sub-agents**: `/docs-hydrate-memory` (backfill mode) — synthesizes `description:` frontmatter; defer-regen so reorg owns the single regen.

---

## `/docs-distill-memory [<domain>]`

**Purpose**: Rewrite an existing `docs/memory/` domain's topic files to the FKF present-truth style (`$(fab kit-path)/reference/fkf.md` §3.2, §3.3) — strip transition narration and superseded-state prose, cap/de-id `description:` frontmatter, and relocate rationale into Design Decisions. The corpus-remediation counterpart to the forward-looking memory writers (step 3 of the present-truth effort; steps 1–2 shipped in `260717-3plm`). Read-only by default — files only rewritten with explicit user approval.

**Context**: `docs/memory/index.md`, the target domain's index and topic files, and `$(fab kit-path)/reference/fkf.md`. Does NOT require `.fab-status.yaml`, config, or constitution. Declares no `helpers:`.

**Prerequisite**: `docs/memory/index.md` must exist and the resolved (named or survey-picked) `docs/memory/{domain}/` must contain at least one topic file.

**Arguments**: `<domain>` is **optional**. Named explicitly, it forces a full read of that one domain (survey skipped) and runs the one-domain flow once (no loop). Omitted, it runs **survey mode** — a cheap heuristic scan across all domains that reports per-domain candidate counts, builds a flagged-domain worklist, and then **loops every flagged domain sequentially** in `docs/memory/index.md` domain-table order, running the one-domain flow (full read → per-file report → per-domain approval → apply → regen) as the loop body per domain (main session, no per-domain dispatch); or reports the terminal all-distilled case when nothing is flagged. No-arg no longer aborts and no longer stops after one domain.

**Behavior**:
1. **Survey mode (no-arg only)**: a single `fab memory-index --check --json` call (the canonical machine surface, not an agent-side grep) counts flagged files per domain (in `docs/memory/index.md` domain-table order) by aggregating four finding kinds — `malformed[]` `description-change-id` + `description-over-cap` (blocking) and `warnings[]` `description-length` (501–1000 advisory) + `narration-density`; a file with multiple findings counts once, a sub-domain file rolls up to its domain (first path segment), and the check's exit code does NOT gate the survey (a missing `type: memory` is not a survey signal). Older-binary fallback (no `--json`/`warnings`): the legacy grep of three classes (`description:` over the 500-char cap, change-ids in `description:`, body narration markers) + an "upgrade fab" warning. Report per-domain counts with the heuristic caveat, build the flagged-domain worklist, then **loop every flagged domain sequentially** (index-table order) — the survey runs **once**, no re-survey between domains — running the one-domain flow (steps 2–5) as the loop body per domain in the main session. A **skipped** domain stays untouched and the loop continues; an already-distilled worklist domain reports "already distilled" and continues; an exit-2 within one domain follows per-domain handling without swallowing the rest; the terminal state is all-distilled or a skipped/remaining summary. If nothing is flagged, report "all domains distilled (survey heuristic)" with the caveat and stop. An explicit `<domain>` skips this step and runs the one-domain flow once (no loop).
2. Read the resolved domain's topic files read-only; classify transition narration, superseded-state prose, `description:` defects (over-cap, change-ids), **change-id heading suffixes** (strip, registry-gated), **byte-identical duplicate blocks** (dedup; near-duplicates flagged, never auto-merged), **Design-Decisions changelog bullets** (rewrite to four-field or remove pure history; never fabricate rationale), **embedded operational TODOs** (relocate → `fab/backlog.md`), rationale-carrying narration (relocate), and allowed provenance (keep)
3. Report per-file proposed rewrites (before/after for the non-obvious; every relocation shown, incl. TODO → backlog relocations; near-duplicates flagged not auto-merged); state per file whether content is deleted vs. relocated and where deleted content is already recorded
4. User confirmation — apply all, cherry-pick specific files, or skip
5. On approval, rewrite bodies to present truth (removing narration, stripping change-id heading suffixes, deduping byte-identical blocks, rewriting DD changelog bullets, relocating rationale into Design Decisions `Why`/`Rejected`, preserving trailing `(change-id)` + `*Introduced by*`), relocate operational TODOs to `fab/backlog.md` (never delete; create with a `# Backlog` header when absent), fix `description:` frontmatter, then regenerate indexes via `fab memory-index` — consulting `fab memory-index --check` first and refusing on exit 2 (destructive loss)
6. Emit a **dynamic `Next:` line** reporting surveyed **skipped/remaining** domains (with flagged-file counts, in index.md order) as a follow-up targeted-run pointer, or "all domains distilled" when none remain. It reports surveyed truth; it **no longer drives per-domain re-invocation** (the no-arg loop already processes every flagged domain in one invocation). No-arg reflects the initial survey minus every domain fully distilled this run (a skipped/partially-cherry-picked domain stays listed while still flagged); an explicit `<domain>` runs the survey at completion to populate it.

**Key properties**: No active change required. `<domain>` optional (named = single-domain full-read override, no loop; omitted = survey mode + all-domains loop). One domain per **approval/apply unit**, iterated within a single invocation — a property of the analysis+apply/approval unit, not the invocation (each domain is read-in-full, reported, approved, and rewritten as its own unit; a no-arg invocation loops that unit over every flagged domain, an explicit `<domain>` runs it once; the per-domain approval gate is retained, no bulk approval). Idempotent (an already-distilled domain proposes nothing; a fully-distilled tree surveys clean so the no-arg loop's worklist is empty; `fab memory-index` is byte-stable). Rationale is relocated, never deleted; deletion is confined to narration recorded elsewhere (log.md/git/archive), and **never fabricated** when rewriting a DD bullet. `_shared/removed-domains.md` is exempt (§3.3 tombstone carve-out). The generated files `index.md`/`log.md` are never hand-edited; `log.seed.md` is a curated read-only seed input (never written by the generator) that distillation excludes like a ledger. Writes one file outside `docs/memory/` — `fab/backlog.md` (operational-TODO relocation). Moves no files, auto-merges no near-/cross-file duplicates (structural moves + cross-file merges belong to `/docs-reorg-memory`).


**Flow**:

```text
User invokes /docs-distill-memory [<domain>]
├─ [omitted] Survey: Bash: fab memory-index --check --json → flagged-domain worklist → per-domain loop in main session; [none flagged] STOP
├─ [given] skip survey, full read of that one domain (no loop); Pre-flight: index.md + ≥1 topic file
├─ Read: domain files + $(fab kit-path)/reference/fkf.md → classify (narration / superseded prose / description: defects / duplicates / DD changelog bullets / TODOs / rationale)
├─ Report → approval gate → [declined] stop, no mutation
└─ [approved] Edit rewrites; relocate TODOs → fab/backlog.md; then once: Bash: fab memory-index --check → fab memory-index; Next: remaining flagged domains
```

**Tools**: Read (domain files + fkf.md reference; survey JSON output), Edit/Write (approved present-truth rewrites; TODO relocation into `fab/backlog.md`), Bash (`fab memory-index --check --json` survey; refuse-guarded `--check` + `fab memory-index` regen).

**Sub-agents**: None — runs inline, including the no-arg all-domains loop.

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


**Flow**:

```text
User invokes /docs-reorg-specs
├─ Pre-flight; Read: all spec files (recursing subfolders)
├─ Propose reorganization → approval gate
└─ [approved] Write/Edit: moved files (bytes verbatim) + Edit: docs/specs/index.md (hand-rewritten)
```

**Tools**: Read (all spec files and index), Write/Edit (approved reorganizations).

**Sub-agents**: None.

---

## `/git-pr [<change>] [<type>]`

**Purpose**: Autonomously commit, push, and create a draft GitHub PR. No questions, no prompts. Covers stage 5 (Ship) of the pipeline.

**Arguments** (both optional, in any order — classified by value):
- `[<change>]` *(optional)* — explicit change to target instead of the active one: any argument that is NOT one of the 7 PR types. Resolved transiently (`.fab-status.yaml` untouched); an explicit argument that fails to resolve STOPs (caller error — never a silent fallback to the active change). Pass the change folder name, not a bare 4-char id: an id spelling a type word (`feat`, `docs`, `test`) would be classified as a type.
- `[<type>]` *(optional)* — PR type prefix: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`. If omitted, type is inferred from `.status.yaml`, `intake.md`, or the diff (in that order).

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
- Branch-matches-change guard: when a change is resolved, the current branch must equal its folder name (or contain it as a substring) — a mismatch STOPs before any status mutation, commit, or push; no autonomous checkout
- Idempotent — skips steps already done (no PR created if one exists)
- Marks the `ship` stage done, auto-activates `review-pr`

**Context**: Does not require an active fab change — works as a standalone git tool. With an active change, reads `intake.md` for PR title/summary.


**Flow**:

```text
/git-pr [<change>] [<type>]
├─ Resolve: Bash: fab change resolve → {has_fab}, {name} (explicit-arg failure STOPs); branch-matches-change guard → STOP on mismatch/detached
├─ Bash: fab status start {name} ship git-pr (if {has_fab}); PR type: status.yaml → intake keywords → git diff fallback
├─ Gather: branch/status/log, gh pr view → {pr_state}, default branch, fab status get-issues
├─ Guards: detached HEAD / on default branch / {pr_state} MERGED → STOP
├─ 3a Commit: expected-area guard for untracked files → git add -u + in-area untracked → commit
├─ 3a-bis (if {has_fab} + committed): Bash: fab memory-index → commit docs/memory drift (no --amend)
├─ 3b Push; 3c Create PR (no OPEN PR): Read intake → Bash: fab pr-meta → ## Meta block → gh pr create --draft (--fill fallback)
├─ 3d Retrofit ## Meta onto existing OPEN PR (idempotent prepend)
└─ 4a–4c: fab status add-pr + finish ship stage; commit + push .status.yaml/.history.jsonl
```

**Tools**: Read (intake — PR title, Summary, Changes), Bash (git, gh, fab status; `fab pr-meta` renders the entire `## Meta` block — self-contained; non-zero/empty → Meta omitted).

**Sub-agents**: None.

---

## `/git-pr-review [<change>] [--tool <name>]`

**Purpose**: Process GitHub PR review comments on the current branch's PR. Handles feedback from any reviewer — human or bot. Covers stage 6 (Review-PR) of the pipeline.

**Arguments**:
- `[<change>]` *(optional)* — explicit change to target instead of the active one: any positional (non-flag) argument; `--tool` and its value are consumed as the flag, never as a change reference. Resolved transiently (`.fab-status.yaml` untouched); an explicit argument that fails to resolve STOPs (caller error), while argless resolution failure proceeds with no change context. When a change is resolved, the branch-matches-change guard STOPs on mismatch before any status mutation.
- `--tool <name>` *(optional)* — force a specific review tool. Valid values: `copilot` (only). Bypasses the Review Tools check (`code-review.md` § Review Tools).

**Example**:
```
/git-pr-review
→ "3 comments triaged: 2 fix, 1 defer, 0 skip, 0 informational (no reply)"
→ "Fixed 2 comment(s) across 2 file(s)"
→ "Replied to 3 comment(s): 2 fix, 1 defer, 0 skip"
```

**Behavior**:
1. Resolve the PR for the current branch via `gh pr view`
2. **If no reviews exist** — request a Copilot review (`gh pr edit --add-reviewer copilot-pull-request-reviewer`) and poll every 30 seconds for up to 10 minutes (20 attempts). If the review arrives, process its comments in the same run; if not, the timeout outcome leaves `review-pr` `active` with a re-run message. Copilot is the only automated reviewer, honoring the Copilot toggle in `code-review.md` § Review Tools (absent = enabled).
3. **If reviews with inline comments exist** — fetch all comments, triage each:
   - **fix**: applies a targeted code change, then posts `Fixed — {description}. ({sha})` as a reply
   - **defer**: posts `Deferred — {reason}.`
   - **skip**: posts `Skipped — {reason}.`
   - **informational**: no reply
4. Commit and push any fixes, then post all replies
5. Route every terminal outcome through Step 6: success / no-reviews → `fab status finish review-pr`; failure → `fab status fail review-pr`; timeout → stage deliberately left `active` (no finish, no fail). Two direct-STOP exceptions never reach Step 6: invalid `--tool` value (Step 1.5) and commit/push failure (Step 5, after `git reset`).

**Key properties**:
- Fully autonomous — never asks questions, never presents options
- Targeted fixes only — does not modify code beyond what each comment addresses
- Idempotent — re-running after fixes finds no new modifications; re-running after replies skips already-replied comments
- The Copilot request honors the Copilot toggle in `fab/project/code-review.md` § Review Tools (absent = enabled)


**Flow**:

```text
/git-pr-review [<change>] [--tool <name>]
├─ Start: Bash: fab change resolve → {name}; branch-matches-change guard → STOP on mismatch/detached; fab status start review-pr
├─ Resolve PR (gh pr view, gh repo view); validate --tool (copilot only) or STOP
├─ Detect: [comments exist] → triage / [none] → request Copilot review, poll gh pr view 30s×20 synchronously → [timeout] Step 6 timeout (stage stays active)
├─ Fetch: Bash: gh api --paginate pulls/{n}/comments (reply comments skipped)
├─ Triage fix/defer/skip/informational → Read + Edit fixes → commit + push ([commit fails] reset + STOP; [push fails] keep commit, no replies)
├─ Post disposition replies (dedup existing, best-effort POSTs); Step 6: fab status finish / fail / timeout-left-active
└─ Step 6.5 (success only, idempotent): commit + best-effort push .status.yaml/.history.jsonl; yq phase tracking on .status.yaml
```

**Tools**: Read/Edit (source files for review fixes), Bash (gh API — REST only; git incl. Step 6.5 status commit; fab status; yq phase tracking).

**Sub-agents**: None.

---

## `/internal-consistency-check`

**Purpose**: Scan for inconsistencies across the three sources of truth — implementation (the `source_paths` from config), memory (`docs/memory/`), and specs (`docs/specs/`) — flagging conflicts and suggesting fixes. A kit-maintenance skill, not a pipeline stage.

**Context**: `fab/project/config.yaml` for `source_paths`. STOPs with `No source_paths defined in fab/project/config.yaml.` when the key is missing or empty.

**Behavior**: Spawns **three parallel `Explore` sub-agents** (thoroughness `very thorough`), each receiving the resolved `source_paths` and auditing one pairing — specs ↔ implementation, memory ↔ implementation, and specs ↔ memory. Every finding cites specific files and lines. The results are synthesized into a summary table (findings / critical / minor per dimension), the critical and minor finding lists, and suggested actions grouped as **Fix** (actively wrong), **Add** (missing coverage), **Remove** (stale or orphaned), **Rename** (terminology alignment).

**Classification**: *Critical* — implementation contradicts a spec, memory instructs something that fails, or a referenced file/command/path does not exist. *Minor* — naming mismatch without behavioral impact, coverage gap, stale reference that would not confuse a user, orphaned content.

**Key properties**: Read-only — reports findings, applies no fixes. No active change required.


**Flow**:

```text
User invokes /internal-consistency-check
├─ Pre-flight: Read config.yaml source_paths (missing → STOP)
├─ Parallel dispatch: 3 Explore agents — Specs↔Impl, Memory↔Impl, Specs↔Memory
└─ Synthesis (no writes): summary table → critical findings → minor findings → suggested actions
```

---

## `/internal-retrospect`

**Purpose**: Analyze a completed session and produce a retrospective. A kit-maintenance skill, not a pipeline stage.

**Behavior**: Reviews the session end-to-end across four areas, citing actual moments from the conversation rather than generic advice: (1) **scriptable repetition** — manual steps repeated 2+ times that could become a script, with a suggested name; (2) **skill and prompt quality** — skills whose output needed significant correction, re-runs, or manual fixup, with a concrete prompt change to prevent it; (3) **context and documentation gaps** — wrong assumptions traceable to something missing from memory, specs, `_preamble.md`, `CLAUDE.md`, or the constitution, plus where that knowledge should live; (4) **workflow friction** — awkward stage transitions or clarification loops, classified as a workflow-design, missing-command, or documentation problem.

**Output**: Per area, either **Findings** with citations and suggested actions or **Clean**. Ends with a consolidated **Suggested Actions** list of concrete next steps (e.g. `Run /internal-skill-optimize {skill}`, `Add {X} to docs/memory/{domain}/{name}.md`). A session with no findings in any area reports `Clean session — no actions needed.`

**Key properties**: Read-only — writes no files. No active change required.


**Flow**:

```text
User invokes /internal-retrospect (agent reasoning over conversation history; no tool calls required)
└─ Output: per-area Findings (citations + suggested actions) or Clean → Suggested Actions, or "Clean session — no actions needed."
```

**Tools**: None — pure conversation analysis (may Read cited files to verify a claim).

**Sub-agents**: None.

---

## `/internal-skill-optimize [<skill-name>]`

**Purpose**: Condense a skill to its core — remove verbosity, redundant examples, and re-explained concepts without losing functionality — and apply two structural health checks. A kit-maintenance skill, not a pipeline stage.

**Arguments**: `<skill-name>` *(optional)* — a single skill, resolved to `src/kit/skills/{skill-name}.md`. Omitted, it processes all skills in batch mode.

**Two passes with different scoping** — this split is the skill's defining rule:
- **Content optimization** (the bloat-signal trim) — a consumer-skill or batch pass treats every `_*.md` partial as read-only shared reference context and never trims one as a side effect. A dedicated pass invoked explicitly with a partial's name (for example, `/internal-skill-optimize _preamble`) legitimately applies the same content signals to that partial. The partial set is derived by globbing `src/kit/skills/_*.md`, never from a hardcoded list.
- **Structural checks** (table of contents + reference depth) — **all** skill files, partials included. A 700-line partial with no TOC, or one that chains references more than one level deep, is a real defect the content rule would wrongly exempt. These checks only add a `## Contents` block or report a finding; they never trim a partial's prose.

**Signals**: Content — redundant re-explanation of partial-defined concepts; transition narration rewritten as present truth per `$(fab kit-path)/reference/fkf.md` §3.3; excessive output examples; obvious instructions; duplicated argument docs; over-specified error tables; verbose step narration; duplicate examples; and mode-scoped sibling duplication. Every content pass compares the analyzed file with the already-loaded `_*.md` partials, so the signal operates only on files loaded for that pass. Ownership rule: a file may state a rule it owns or point at the owner, never both. Duplication findings are report-only. Structural — a file **over 100 lines** with no `## Contents` block (inserted after the H1 and preamble blockquote, listing `##` headings verbatim), and **reference depth > 1 level** (a SKILL → A → B chain), which is **report-only** — flagging the chain for the maintainer, never auto-restructuring. The one-output-example rule applies only to illustrative examples; grep/string-match literals such as `git-pr`'s checkmark lines and `git-pr-review`'s disposition prefixes are contracts and are never consolidated or reworded.

**Behavior**: Analyze, then report a before/after line count, a one-line-per-change content summary, the TOC action (added / already present / not needed), and any duplication or depth findings. Confirm with the user before writing. Batch mode sorts by line count descending, skips content optimization for files under 80 lines (`Already lean — skipped`) and for partials, runs structural checks on everything, compares the analyzed skills for cross-skill/twin clusters after per-file analysis, and presents one consolidated table plus report-only duplication and depth findings.

**Constraints**: Never removes functionality, error handling, or edge-case coverage; preserves frontmatter, the H1, and the preamble blockquote exactly; keeps the imperative tone; keeps a TOC in sync when a trim changes a `##` heading; never merges skills or moves content between them.


**Flow**:

```text
User invokes /internal-skill-optimize [<skill-name>]
├─ Pre-flight: Read _*.md partials; [named skill missing] STOP
├─ Single: Read target → content + structural analysis → AskUserQuestion → [approved] Write optimized file
└─ Batch: Read all skills → content trim skips <80-line files + partials (structural checks on all) → summary table → approval → Write approved files
```

**Tools**: Read (partials as read-only reference context + target files), Write (approved optimized files — a partial only in a dedicated partial-optimization pass), AskUserQuestion (approval gate before any write).

**Sub-agents**: None.
