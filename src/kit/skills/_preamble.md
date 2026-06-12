---
name: _preamble
description: "Shared context preamble loaded by every Fab skill — defines path conventions, context loading, the skill helper model, naming conventions, common fab commands, and the confidence gate."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Context Preamble

> This file defines shared conventions for all Fab skills. Each skill file should begin with:
> ``Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.``

---

## Path Convention

All script and file paths in skills are **relative to the repo root** (the agent's CWD). Never expand them to absolute paths.

```
# correct
fab preflight

# wrong
bash /home/user/.fab-kit/versions/0.47.0/fab-go preflight
```

---

## Context Loading

Before generating or validating any artifact, load the relevant context layers below. This ensures output is grounded in the actual project state, not assumptions.

### 1. Always Load

This layer applies to every skill **unless the skill's own Context Loading section says otherwise** — the skill file wins. Current exceptions: `/fab-setup`, `/fab-status`, `/fab-switch`, and `/docs-hydrate-memory` skip the layer entirely; `/fab-operator` loads only `config.yaml`, `constitution.md`, and `context.md`.

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/project/config.yaml`** — project configuration: identity (name/description), `source_paths`/`test_paths`, true-impact excludes, plan-acceptance extra categories, `review_tools` toggles, agent spawn command, optional `stage_hooks`
- **`fab/project/constitution.md`** — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- **`fab/project/context.md`** — free-form project context: tech stack, conventions, architecture *(optional — no error if missing)*
- **`fab/project/code-quality.md`** — coding standards for apply/review: principles, anti-patterns, test strategy *(optional — no error if missing)*
- **`fab/project/code-review.md`** — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
- **`docs/memory/index.md`** — memory landscape (which domains exist; a domain may contain sub-domains — see § Memory File Lookup)
- **`docs/specs/index.md`** — specifications landscape (pre-implementation design intent, human-curated)

> **Note**: If the skill runs `fab preflight` (Section 2 above), the init check (config.yaml and constitution.md existence) is already covered by the script. Skills using preflight don't need separate existence checks for these files — they only need to read them for content.

Additional helpers beyond this preamble are declared by each skill in its frontmatter `helpers:` list (see **Skill Helper Declaration** below). `_preamble` loads nothing extra by default.

### 2. Change Context (when operating on an active change)

Resolve the active change and load its state by running the preflight script:

1. **Run preflight**: Execute `fab preflight [change-name]` via Bash — pass the optional change-name argument if the skill received one
2. **Check exit code**: If the script exits non-zero, STOP and surface the stderr message to the user (it contains the specific error and suggested fix)
3. **Parse stdout YAML**: On success, parse the YAML output for `id`, `name`, `change_dir`, `stage`, `display_stage`, `display_state`, `progress`, `plan`, and `confidence` fields — use these for all subsequent change context instead of re-reading `.status.yaml`. Use `id` (4-char change ID) for script invocations; use `name` for display, path construction, and artifact metadata.
4. **Log command**: Call `fab log command "<skill-name>" "<id>"` where `<skill-name>` is the invoking skill (e.g., `fab-continue`) and `<id>` is the `id` field from the preflight YAML output. This is best-effort — the command always exits 0 given valid usage (internal failures surface only as a stderr warning; cobra arg-count errors exit non-zero before RunE), so no shell guard is needed.
5. Load all completed artifacts in the change folder (`intake.md`, `plan.md`) — read each file that exists so you have full context of what has been decided so far. (A leftover `spec.md` may exist in pre-1.10.0 changes; read it for context if present, but `spec.md` is no longer a generated artifact.)

> **Change-name override**: When a `[change-name]` argument is passed to the preflight script, it resolves the change using case-insensitive substring matching against `fab/changes/` folder names (excluding `archive/`) instead of reading the `.fab-status.yaml` symlink. The override is **transient** — `.fab-status.yaml` is never modified. This enables parallel workflows where multiple tabs target different changes concurrently. Supports full folder names, partial slugs, or 4-char IDs (e.g., `r3m7`).

> **What the script validates internally** (for reference — agents do not need to duplicate these checks):
> 1. `fab/project/config.yaml` and `fab/project/constitution.md` exist (project initialized)
> 2. `.fab-status.yaml` symlink exists (active change set) — OR `$1` override resolves to a valid change
> 3. Change directory `fab/changes/{name}/` exists
> 4. `.status.yaml` exists within the change directory

### 3. Memory File Lookup (when operating on an active change)

Selectively load relevant memory files based on the change's scope. An Affected Memory entry is either flat (`{domain}/{name}`) or sub-domained (`{domain}/{sub-domain}/{name}` — used after a domain has been split). Load via an **up-to-3-hop walk**:

1. Read the intake's **Affected Memory** section to identify which domains (and sub-domains) are relevant
2. **Domain index**: for each referenced domain, read `docs/memory/{domain}/index.md` to understand the domain's memory files and any sub-domains it lists
3. **Sub-domain index (only if the entry is sub-domained)**: if the referenced file lives in a sub-domain (3-part `{domain}/{sub-domain}/{name}` form), read `docs/memory/{domain}/{sub-domain}/index.md` next
4. **File**: read the specific memory file referenced by each Affected Memory entry (those marked `(new)`, `(modify)`, or `(remove)`) — `docs/memory/{domain}/{name}.md` for a flat entry, or `docs/memory/{domain}/{sub-domain}/{name}.md` for a sub-domained entry — for each listed file that exists
5. If a referenced file, sub-domain, or domain does not exist yet (e.g., listed as `(new)`), note this and proceed without error — it will be created during hydrate (via `/fab-continue` or `/fab-ff`)
6. Use this context to ground all artifact generation (plan, reviews) in the real current state, not assumptions

### 4. Source Code Loading (during implementation and review)

Load only the source files relevant to the current work:

1. Read the relevant source files referenced in the task descriptions or the plan's `## Requirements` affected areas
2. Scope to files actually touched by the change — do not load the entire codebase
3. This applies primarily to apply and review behavior in `/fab-continue`
4. **Apply stage**: Also read neighboring files in the same directories to extract pattern context (naming conventions, error handling style, typical structure, reusable utilities). This supports Pattern Extraction in `/fab-continue` Apply Behavior
5. **Review stage**: Re-read all files modified during apply, plus their surrounding code in the same directories, to validate consistency with codebase patterns

---

## Skill Helper Declaration

A skill MAY declare additional helper files it needs to load via frontmatter:

```yaml
---
name: fab-ff
description: ...
helpers: [_generation, _review, _srad, _pipeline]
---
```

**Allowed values**: `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`.

**Not allowed** (inlined into this preamble): `_naming`, `_cli-rk`.

**Implicit** (never list): `_preamble` itself is loaded universally.

**Semantics**: After reading `_preamble` and before executing the skill body, the agent MUST read `.claude/skills/{helper}/SKILL.md` for each declared helper. Skills that declare no `helpers:` list (or an empty list) load only `_preamble`.

**Stage-conditional loading**: A skill MAY instead load a helper at its point of use via an explicit in-body read instruction (e.g., "read `.claude/skills/_review/SKILL.md` before entering Review Behavior"). Frontmatter `helpers:` declares unconditional pre-body loads; in-body read instructions declare conditional ones — a helper loaded this way is intentionally absent from the frontmatter list. `/fab-continue` uses this for `_generation` (apply entry / intake regeneration) and `_review` (review stage).

---

## Naming Conventions

> Defines naming patterns shared across the workflow.

### Change Folder

| Field | Value |
|-------|-------|
| **Pattern** | `{YYMMDD}-{XXXX}-{slug}` |
| **Example** | `260226-jq7a-slim-config-decouple-naming` |
| **Generated by** | `fab change new` |

Components:
- `YYMMDD` — date (always today)
- `XXXX` — 4 random lowercase alphanumeric chars (uniqueness guarantee)
- `slug` — 2-6 word kebab-case description (caller-provided via `--slug`)

The `{YYMMDD}-{XXXX}` prefix is immutable. Only the slug can be changed via `fab change rename`.

### Git Branch

| Field | Value |
|-------|-------|
| **Pattern** | `{change-folder-name}` |
| **Example** | `260226-jq7a-slim-config-decouple-naming` |
| **Created by** | `/git-branch` |

The branch name equals the change folder name directly. No prefix. For standalone branches (no matching change), the raw argument is used as-is.

### Worktree Directory

| Field | Value |
|-------|-------|
| **Pattern** | `{adjective}-{noun}` |
| **Example** | `swift-fox` |
| **Generated by** | `wt create` |

Random adjective-noun combo from predefined word lists. Overridable via `--worktree-name`. Worktrees are created at `$(dirname {repo_root})/{repo_name}.worktrees/`.

---

## Run-Kit (rk) Reference

> All rk usage MUST fail silently if rk is not installed — check `command -v rk` before any rk operation. Do not surface errors or warnings to the user when rk is absent.

### Detection

Before using any rk capability, check availability:

```sh
command -v rk >/dev/null 2>&1 || return  # in functions
command -v rk >/dev/null 2>&1            # in conditionals
```

If `rk` is not available, skip all rk operations silently. Never error, never warn.

### Iframe Windows

Create a tmux window that displays a web page instead of a terminal:

```sh
tmux new-window -n <name>
tmux set-option -w @rk_type iframe
tmux set-option -w @rk_url <url>
```

Change the URL of an existing iframe window:

```sh
tmux set-option -w @rk_url <new-url>
```

The rk server detects `@rk_type` and `@rk_url` changes automatically via SSE polling — no manual refresh needed.

### Proxy

Access local services through the rk server using the proxy URL pattern:

```
{server_url}/proxy/{port}/...
```

For example, a service on port 8080 is available at `{server_url}/proxy/8080/`.

### Server URL Discovery

Discover the server URL at **use-time** by running:

```sh
rk context 2>/dev/null | grep 'Server URL' | awk '{print $NF}'
```

Never hardcode the server URL — it can change between sessions.

### Visual Display Recipe

The canonical recipe for displaying HTML content in an iframe window is documented by `rk context` — run-kit owns this workflow because every step (loopback HTTP server, relative `/proxy/<port>/...` path, `@rk_type`/`@rk_url` tmux options) is run-kit-specific. Keeping the recipe in one place eliminates drift between fab-kit and run-kit.

At use-time, call `rk context` and read the `### Visual Display Recipe` subsection of the output for the current 4-step flow (generate HTML → loopback HTTP server → iframe window with relative `@rk_url` → fail silently). Any step SHALL fail silently if its prerequisite is unavailable (rk missing, port in use, server start fails) — skip remaining steps without surfacing an error.

#### Visual-Explainer Integration

When the `visual-explainer` plugin is available, skills MAY delegate HTML generation to it (Step 1 of the `rk context` recipe), then follow the remaining steps to display the result. If `visual-explainer` is not available, skip the visual display entirely — no error, no fallback.

---

## Common fab Commands

These command families cover ~90% of skill usage. See `_cli-fab` for the full reference (argument formats, every subcommand, flag details).

| Command | Purpose | Canonical form |
|---------|---------|----------------|
| `fab preflight [<change>]` | Validate init + resolve active change; outputs YAML with `id`/`name`/`change_dir`/`stage`/`display_stage`/`display_state`/`progress`/`plan`/`confidence`. Non-zero exit on error. | `fab preflight` |
| `fab score [--check-gate] [--stage intake] <change>` | Compute SRAD confidence from `intake.md` (the sole scoring source). `--check-gate` returns non-zero below the single intake gate (flat 3.0 for all types). `--stage` defaults to `intake`. | `fab score --check-gate --stage intake <id>` |
| `fab log command "<skill>" [<change>]` | Best-effort command telemetry — always exits 0 given valid usage (internal failures become a stderr warning, never an error; cobra arg-count errors exit non-zero before RunE). No shell guard needed. | `fab log command "fab-continue" "<id>"` |
| `fab change <sub>` | Change lifecycle: `new --slug <slug>`, `switch <name>\|--none`, `resolve [<override>]`, `rename`, `list [--archive] [--show-stats]`, `archive <change>`, `restore <change> [--switch]`. | `fab resolve --folder` *(note: the query flags live on top-level `fab resolve` only — `fab change resolve` takes a bare `[<override>]`, no flags)* |
| `fab resolve [--id\|--folder\|--dir\|--status\|--pane] [<change>]` | Pure query — converts change reference to canonical output (4-char ID by default). No side effects. | `fab resolve --folder 2>/dev/null` |
| `fab status <sub> <change>` | State machine + metadata. Key subcommands: `finish <stage>` (auto-activates next), `advance <stage>`, `start <stage>`, `reset <stage>`, `skip <stage>`, `fail <stage>` (review/review-pr only), `set-change-type <type>`, `set-acceptance <field> <value>` (updates `plan:` block), `add-issue <id>`, `add-pr <url>`. | `fab status finish <id> <stage>` |

**Key behaviors** to remember without loading `_cli-fab`:

- `fab status finish <change> <stage>` auto-activates the next pending stage — never call `start` after `finish`.
- `fab status finish <change> review` auto-logs review `"passed"`; `fab status fail <change> review` auto-logs `"failed"`.
- `fab log command` is best-effort and always exits 0 (given valid usage — cobra arg-count errors exit non-zero before RunE) — no shell guard needed (internal failures print a stderr warning only).
- `<change>` argument everywhere accepts 4-char ID, folder substring, or full folder name.
- **Failure rule**: any fab command that exits non-zero → STOP and surface stderr; resumability handles the re-run. (`fab log command` can never trip this rule through internal failure — it owns its best-effort contract and exits 0 given valid usage; a cobra arg-count error still exits non-zero before RunE.) This rule defers to explicit per-skill handling where a skill intentionally branches on a non-zero exit.

---

## Next Steps Convention

Skills MUST end their output with a `Next:` line derived from the State Table below, unless the skill's own Output or Key Properties section defines a different ending (e.g., `/fab-discuss`'s ready signal, `/fab-operator`'s status frame, the `/git-*` skills' own completion output) — the skill file wins, mirroring the §1 context-loading contract. Look up the state reached (not the skill name) and list the available commands. The default command SHOULD be listed first.

**Format**: `Next: /fab-command` or `Next: /fab-commandA, /fab-commandB, or /fab-commandC`

### State Table

| State | Available commands | Default |
|-------|-------------------|---------|
| (none) | /fab-setup | /fab-setup |
| initialized | /fab-new, /fab-proceed, /docs-hydrate-memory | /fab-new |
| intake | /fab-continue, /fab-ff, /fab-fff, /fab-proceed, /fab-clarify | /fab-continue |
| apply | /fab-continue | /fab-continue |
| review (pass) | /fab-continue | /fab-continue |
| review (fail) | *(rework menu)* | — |
| hydrate | /git-pr, /fab-archive | /git-pr |
| ship | /git-pr-review | /git-pr-review |
| review-pr (pass) | /fab-archive | /fab-archive |
| review-pr (fail) | /git-pr-review | /git-pr-review |

**State derivation**:
- **(none)**: `fab/project/config.yaml` does not exist
- **initialized**: `fab/project/config.yaml` exists AND no active change (`.fab-status.yaml` symlink is absent)
- **intake** / **apply**: Derived from the active change's `.status.yaml` progress map (the stage with `active` or `ready` state)
- **review (pass)**: `progress.review == done`
- **review (fail)**: `progress.review == failed`
- **hydrate**: `progress.hydrate == done`

### Lookup Procedure

1. Determine the state reached after the skill's action
2. Look up that state in the State Table
3. Output `Next:` with the default command listed first, followed by other available commands

### Activation Preamble

When a skill creates or restores a change without activating it (no `.fab-status.yaml` symlink created), the `Next:` line SHALL include a switch instruction followed by the state-derived commands:

```
Next: /fab-switch {name} to make it active, then {default}, {other commands}
```

This applies to `/fab-draft` (always) and `/fab-archive restore` (without `--switch`). `/fab-new` auto-activates and does not need the activation preamble.

---

## Skill Invocation Protocol (pointer)

The `[AUTO-MODE]` inter-skill invocation protocol (prefix signaling autonomous mode when one skill invokes another) is defined in `fab-clarify.md` § Skill Invocation Protocol — its sole referencer. No skill currently invokes another with the prefix; user-invoked skills always run interactive.

---

## Subagent Dispatch (Orchestrator Skills)

Orchestrator skills (`/fab-ff`, `/fab-fff`) run multi-stage pipelines that invoke other skills as sub-operations. To preserve the orchestrator's pipeline context, sub-skills are dispatched as **subagents** using the Agent tool (`subagent_type: "general-purpose"`) — never the Skill tool.

**Why not the Skill tool?** The Skill tool expands the sub-skill's prompt into the orchestrator's execution context. After the sub-skill completes, the pipeline context is lost and execution halts. The Agent tool runs the sub-skill in a **separate context** and returns a structured result, keeping the pipeline intact.

**Dispatch pattern** — each subagent prompt includes:

1. The skill file to read (deployed to `.claude/skills/{skill}/SKILL.md`)
2. The specific behavior section to follow (e.g., "Apply Behavior", "Auto Mode")
3. The change ID for resolution
4. Any mode prefix (e.g., `[AUTO-MODE]` — defined in `fab-clarify.md` § Skill Invocation Protocol)
5. The expected return format
6. The standard subagent context files (see below)

### Standard Subagent Context

Every subagent prompt MUST instruct the subagent to read the following project files **before** executing its task. This ensures subagents operate with full awareness of project principles, constraints, and conventions — regardless of nesting depth.

**Required** (subagent reports error if missing):
- `fab/project/config.yaml`
- `fab/project/constitution.md`

**Optional** (skip gracefully if missing):
- `fab/project/context.md`
- `fab/project/code-quality.md`
- `fab/project/code-review.md`

**Nested dispatch**: When a subagent dispatches its own sub-subagent (e.g., review sub-agent within `/fab-continue`), the inner prompt MUST also include the standard subagent context instruction. The same 5 files are loaded at every nesting level.

`general-purpose` subagents have full tool access (Read, Edit, Write, Bash, Agent) and can execute any skill behavior including file modifications and nested subagent dispatch.

---

## SRAD Autonomy Framework (pointer)

SRAD is the decision framework planning skills use to score decision points (Signal, Reversibility, Agent Competence, Disambiguation → Certain/Confident/Tentative/Unresolved) and decide when to ask vs. assume. The full framework — scoring dimensions, grade thresholds, Critical Rule, artifact markers, and the Assumptions Summary block — lives in the `_srad` helper, declared via `helpers:` by the six planning skills (`fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`). Non-planning skills do not need it.

---

## Confidence Scoring

Confidence scoring provides a numeric measure of how well-resolved a change's decisions are, used as the single intake gate for fast-forward pipeline execution via `/fab-ff` and `/fab-fff`. Scoring reads `intake.md` (the sole scoring source) — there is no separate spec score.

Agents never compute the score — `fab score` (Go) does. The `.status.yaml` schema, the score formula, and the status-template details live in `_cli-fab.md` § fab score (extended).

### Gate Threshold

There is exactly **one** confidence gate, evaluated at intake before the automated bracket proceeds. Both `/fab-ff` and `/fab-fff` check it via `fab score --check-gate --stage intake`. The `--force` flag on either skill bypasses it.

**Intake gate**: threshold **3.0 for all seven change types** (flat). `CheckGate` obtains it via `getGateThreshold(changeType)` (which returns 3.0 for every type today), so future per-type divergence is a data-only change. There is no separate spec gate — it was removed in 1.10.0 along with the spec stage.

A change whose intake fails the gate never reaches `done`, so gate-checking orchestrators cannot enter apply. This intake gate is the only "bounce" guard: there is no runtime mechanism inside apply that detects an SRAD Unresolved and resets to intake. The SRAD Critical Rule (Unresolved must be asked/bailed) applies at intake-time skills only (`/fab-new`, `/fab-clarify`).

See `docs/specs/change-types.md` for the full taxonomy.

### Invocation

Confidence is computed by `fab score` (reading `intake.md`), invoked by:
- `/fab-new` (after intake generation, `--stage intake`) — persists the intake score
- `/fab-clarify` (intake target, suggest mode) — re-persists the intake score after resolving assumptions

`/fab-continue` does NOT score at apply entry — intake is authoritative, and there is no scoring at any post-intake stage.

Both `/fab-ff` and `/fab-fff` gate at a single point: the intake gate via `fab score --check-gate --stage intake` before starting the automated bracket. The `--force` flag on either skill bypasses it.

### Bulk Confirm (pointer)

`/fab-clarify` offers a bulk-confirm flow for Confident assumptions — defined in `fab-clarify.md` (Step 2, Suggest Mode), the sole authority for its trigger and semantics.
