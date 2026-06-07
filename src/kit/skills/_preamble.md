---
name: _preamble
description: "Shared context preamble loaded by every Fab skill — defines path conventions, context loading, SRAD framework, and confidence scoring."
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

### 1. Always Load (every skill except `/fab-setup`, `/fab-status`, `/docs-hydrate-memory`; `/fab-switch` loads only `config.yaml`)

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/project/config.yaml`** — project configuration, naming conventions, model tiers
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
3. **Parse stdout YAML**: On success, parse the YAML output for `id`, `name`, `change_dir`, `stage`, `progress`, `plan`, and `confidence` fields — use these for all subsequent change context instead of re-reading `.status.yaml`. Use `id` (4-char change ID) for script invocations; use `name` for display, path construction, and artifact metadata.
4. **Log command**: Call `fab log command "<skill-name>" "<id>" 2>/dev/null || true` where `<skill-name>` is the invoking skill (e.g., `fab-continue`) and `<id>` is the `id` field from the preflight YAML output. This is best-effort — failures are silently ignored.
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
name: fab-continue
description: ...
helpers: [_generation, _review]
---
```

**Allowed values**: `_generation`, `_review`, `_cli-fab`, `_cli-external`.

**Not allowed** (inlined into this preamble): `_naming`, `_cli-rk`.

**Implicit** (never list): `_preamble` itself is loaded universally.

**Semantics**: After reading `_preamble` and before executing the skill body, the agent MUST read `.claude/skills/{helper}/SKILL.md` for each declared helper. Skills that declare no `helpers:` list (or an empty list) load only `_preamble`.

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

### Operator Spawning Rules

When the operator creates a worktree for an agent, the naming strategy depends on whether the change already exists:

### Known change (already exists)

Use the change folder name as the branch argument to `wt create`:

```
wt create --non-interactive --worktree-name <name> <change-folder-name>
```

The worktree gets a random name; the branch matches the change. No `/git-branch` needed.

### New change (from backlog)

The change folder doesn't exist yet, so there's no branch name to use:

1. `wt create --non-interactive` — auto-generates worktree name, creates on default branch
2. Agent runs `/fab-new` to create the change folder
3. Operator sends `/git-branch` to the agent after detecting the intake stage has advanced — this aligns the branch name with the newly created change folder name

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
| `fab preflight [<change>]` | Validate init + resolve active change; outputs YAML with `id`/`name`/`change_dir`/`stage`/`progress`/`plan`/`confidence`. Non-zero exit on error. | `fab preflight` |
| `fab score [--check-gate] [--stage intake] <change>` | Compute SRAD confidence from `intake.md` (the sole scoring source). `--check-gate` returns non-zero below the single intake gate (flat 3.0 for all types). `--stage` defaults to `intake`. | `fab score --check-gate --stage intake <id>` |
| `fab log command "<skill>" [<change>]` | Best-effort command telemetry. Failures silently ignored. | `fab log command "fab-continue" "<id>" 2>/dev/null \|\| true` |
| `fab change <sub>` | Change lifecycle: `new --slug <slug>`, `switch <name>\|--none`, `resolve [<override>]`, `rename`, `list [--archive]`, `archive <change>`, `restore <change> [--switch]`. | `fab change resolve --folder` *(note: `fab resolve` is the pure-query alias)* |
| `fab resolve [--id\|--folder\|--dir\|--status\|--pane] [<change>]` | Pure query — converts change reference to canonical output (4-char ID by default). No side effects. | `fab resolve --folder 2>/dev/null` |
| `fab status <sub> <change>` | State machine + metadata. Key subcommands: `finish <stage>` (auto-activates next), `advance <stage>`, `start <stage>`, `reset <stage>`, `skip <stage>`, `fail <stage>` (review only), `set-change-type <type>`, `set-acceptance <field> <value>` (updates `plan:` block), `add-issue <id>`, `add-pr <url>`. | `fab status finish <id> <stage>` |

**Key behaviors** to remember without loading `_cli-fab`:

- `fab status finish <change> <stage>` auto-activates the next pending stage — never call `start` after `finish`.
- `fab status finish <change> review` auto-logs review `"passed"`; `fab status fail <change> review` auto-logs `"failed"`.
- `fab log command` is best-effort — always trail with `2>/dev/null || true`.
- `<change>` argument everywhere accepts 4-char ID, folder substring, or full folder name.

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line derived from the State Table below. Look up the state reached (not the skill name) and list the available commands. The default command SHOULD be listed first.

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

## Skill Invocation Protocol

When one skill invokes another internally (e.g., `/fab-ff` invoking `/fab-clarify` between stages), the calling skill MUST signal the invocation mode explicitly using an instruction prefix. This makes the contract between skills explicit and testable rather than relying on implicit "call context" interpretation.

### Protocol

1. **Prefix**: `[AUTO-MODE]`
2. **Placement**: The calling skill includes `[AUTO-MODE]` as the **first line** of the invocation prompt / instruction to the called skill.
3. **Detection**: The called skill checks for the `[AUTO-MODE]` prefix at the start of its invocation context.
   - **If present**: Enter autonomous mode (no user interaction, machine-readable result).
   - **If absent**: Enter default/interactive mode (user-facing, structured questions).
4. **Transitivity**: When skills chain, each link applies the prefix independently.

### Currently Applicable

No skill currently invokes another with the `[AUTO-MODE]` prefix. The former
`/fab-fff` → `/fab-clarify` and `/fab-ff` → `/fab-clarify` auto-invocations were
removed in 1.10.0: clarification is an intake-only, human-facing activity, so no
clarify step runs inside the automated post-intake bracket (apply → review →
hydrate → ship → review-pr). The protocol itself remains defined for future use.

User-invoked skills never carry the `[AUTO-MODE]` prefix, so called skills default to interactive mode.

To add new mode signals, define new bracketed prefixes (e.g., `[BATCH-MODE]`) here. Pattern: one prefix per mode, first-line placement, absence means default.

---

## Subagent Dispatch (Orchestrator Skills)

Orchestrator skills (`/fab-ff`, `/fab-fff`) run multi-stage pipelines that invoke other skills as sub-operations. To preserve the orchestrator's pipeline context, sub-skills are dispatched as **subagents** using the Agent tool (`subagent_type: "general-purpose"`) — never the Skill tool.

**Why not the Skill tool?** The Skill tool expands the sub-skill's prompt into the orchestrator's execution context. After the sub-skill completes, the pipeline context is lost and execution halts. The Agent tool runs the sub-skill in a **separate context** and returns a structured result, keeping the pipeline intact.

**Dispatch pattern** — each subagent prompt includes:

1. The skill file to read (deployed to `.claude/skills/{skill}/SKILL.md`)
2. The specific behavior section to follow (e.g., "Apply Behavior", "Auto Mode")
3. The change ID for resolution
4. Any mode prefix (`[AUTO-MODE]`)
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

## SRAD Autonomy Framework

When generating artifacts, planning skills encounter decision points not explicitly addressed by user input. The SRAD framework provides a principled method for deciding when to ask, when to assume, and when to surface assumptions.

### SRAD Scoring

For each decision point, evaluate four dimensions on a **continuous 0–100 scale** (100 = fully safe to assume, 0 = must ask):

| Dimension | High (75–100) | Medium (40–74) | Low (0–39) |
|-----------|--------------|----------------|------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | Moderate detail, some gaps | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Moderate rework, a few files | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Partial signals, some inference | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | 2–3 options, clear front-runner | Multiple valid interpretations with different tradeoffs |

**Aggregation**: Compute a composite score via weighted mean: `composite = 0.25*S + 0.30*R + 0.25*A + 0.20*D`. Map to grade using thresholds: Certain (85–100), Confident (60–84), Tentative (30–59), Unresolved (0–29). Critical Rule override: R < 25 AND A < 25 → always Unresolved.

Record per-dimension scores in the Assumptions table's required `Scores` column (e.g., `S:75 R:80 A:65 D:70`). The Scores column is mandatory for every row. `fab score` parses these and writes aggregate dimension statistics to `.status.yaml`.

### Confidence Grades

Each decision produces an assumption graded on a 4-level scale:

| Grade | Meaning | Artifact Marker | Output Visibility |
|-------|---------|----------------|-------------------|
| **Certain** | Determined by config/constitution/template rules | None | Noted in Assumptions summary |
| **Confident** | Strong signal, one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, multiple valid options | `<!-- assumed: {description} -->` | Noted in Assumptions summary, `/fab-clarify` suggested |
| **Unresolved** | Cannot determine, incompatible interpretations | None — always asked or bailed | Asked as question AND noted in Assumptions summary |

### Critical Rule

**Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked** — even in `/fab-new` and `/fab-continue`. These count toward the skill's question budget (max ~3). The existence of `/fab-clarify` as an escape valve does NOT justify silently assuming high-blast-radius decisions. `/fab-clarify` is for Tentative assumptions, not for Unresolved ones.

### Skill-Specific Autonomy Levels

| Aspect | fab-new (adaptive) | fab-continue (deliberate) | fab-fff (full pipeline) | fab-ff (fast-forward) |
|--------|-------------------|---------------------------|-------------------------|--------------------------|
| **Posture** | SRAD-driven: 0 questions for clear inputs, conversational for vague; gap analysis before folder creation | Surface tentative, ask top ~3 unresolved | Gated on confidence; extends through ship + review-pr | Gated on confidence; stops at hydrate |
| **Interruption budget** | SRAD-driven (no fixed cap); conversational mode for vague inputs | 1-2 per stage | 0 (autonomous rework, then stop) | 0 (autonomous rework, then stop) |
| **Output** | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary + apply/review/hydrate/ship/review-pr output | Tasks + apply/review/hydrate output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify`, `/fab-continue` (after rework cap) | `/fab-clarify`, `/fab-continue` (after rework cap) |
| **Recomputes confidence?** | Yes (intake, via `fab score --stage intake`) | No (no scoring at apply — intake is authoritative) | No | No |

### Worked Examples

#### Example 1: Auth provider selection

> **Decision point**: User says "Add auth." Which provider — OAuth2, SAML, API keys?
>
> | Dimension | Score | Reasoning |
> |-----------|-------|-----------|
> | S — Signal | Low | One word ("auth") — no detail on mechanism |
> | R — Reversibility | Low | Auth architecture cascades into DB schema, middleware, API contracts |
> | A — Agent Competence | Low | Business relationship with identity providers is a user preference |
> | D — Disambiguation | Low | OAuth2, SAML, and API keys all valid with different tradeoffs |
>
> **Grade: Unresolved** — all four dimensions score low. This MUST be asked (Critical Rule applies: low R + low A).

#### Example 2: Error response format

> "Handle errors" in a REST API → S: Medium, R/A/D: High. **Confident** — codebase signal is strong, easily reversed, one obvious default. Note in Assumptions summary, don't ask.

#### Example 3: Test framework selection

> "Which test framework?" → S: Low, R/A/D: High. **Certain** — config deterministically answers this (use existing runner). No marker, no mention.

### Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (fab-new, fab-continue, fab-ff) | `/fab-clarify` (suggest + auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

**Placement**: Insert the marker inline in the artifact, immediately after the assumed or guessed content. The `{description}` MUST be a concise summary of what was assumed/guessed and why.

**Example**:
```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

### Assumptions Summary Block

Every planning skill invocation SHALL end its output with an Assumptions summary and persist it as a trailing `## Assumptions` section in the generated artifact.

**Output format** (displayed to user):

```
## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 2 | Confident | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 3 | Tentative | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 4 | Unresolved | {decision summary} | {status context} | S:nn R:nn A:nn D:nn |

{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved). Run /fab-clarify to review.
```

**Artifact format** (persisted in the generated file): The same table is appended as the last section (`## Assumptions`) of the generated artifact. This ensures `/fab-clarify` can discover and scan assumptions from the artifact file.

**Rules**:
- Include all four grades (Certain, Confident, Tentative, Unresolved) in the summary. The Scores column (`S:nn R:nn A:nn D:nn`) is required for every row.
- Unresolved rows MUST include status context in the Rationale column: `Asked — {outcome}` or `Deferred — {reason}`.
- For `/fab-ff`, the output summary is **cumulative** across all generated stages. Each entry notes its source artifact (e.g., "in plan.md"). Per-artifact `## Assumptions` sections are persisted individually.
- If 0 assumptions were made, omit the Assumptions summary entirely (no empty table).

---

## Confidence Scoring

Confidence scoring provides a numeric measure of how well-resolved a change's decisions are, used as the single intake gate for fast-forward pipeline execution via `/fab-ff` and `/fab-fff`. Scoring reads `intake.md` (the sole scoring source) — there is no separate spec score.

### Schema (in `.status.yaml`)

```yaml
confidence:
  certain: 12      # count of Certain-graded SRAD decisions
  confident: 3     # count of Confident-graded decisions
  tentative: 2     # count of Tentative-graded decisions
  unresolved: 0    # count of Unresolved-graded decisions
  score: 2.1       # derived score (see formula below), computed from intake.md
```

> The `confidence.indicative` flag is retired (1.10.0): intake scoring is now authoritative, not indicative, so the flag's distinction is meaningless with one scoring source. It is no longer written; a legacy `indicative: true` key on disk is tolerated on read and harmlessly dropped on the next save.

### Formula

```
if unresolved > 0:
  score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Where `total_decisions = certain + confident + tentative + unresolved` and `expected_min` is looked up by `change_type` from a single embedded table in `fab score` (`feat:7, refactor:6, fix:5`, default `3` for `docs`/`test`/`ci`/`chore`). The `cover` factor prevents thin intakes from getting inflated scores. When `total_decisions >= expected_min`, `cover = 1.0` and the formula degenerates to the base penalty. Range: 0.0 to 5.0. See `docs/specs/change-types.md` for the full `expected_min` table.

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

### Template

The `status.yaml` template (in the kit cache at `$(fab kit-path)/templates/status.yaml`) includes the confidence block initialized to zero counts and score 0.0. `/fab-new` writes the intake score after intake generation; `/fab-clarify` re-writes it after resolving intake assumptions.

### Bulk Confirm (Confident Assumptions)

When the confidence score is low primarily due to many Confident (not Tentative/Unresolved) assumptions, `/fab-clarify` offers a bulk confirm flow. This displays all Confident assumptions in a numbered list and lets the user confirm, change, or request explanation in a single conversational turn — typically 10x faster than individual question/answer cycles.

Detection: triggered when `confident >= 3` and `confident > tentative + unresolved`. Counts are evaluated after tentative resolution in Step 1.5.

This flow runs as Step 2 in Suggest Mode, after the taxonomy scan and tentative resolution (Step 1.5). Items confirmed are upgraded to Certain (Rationale: `Clarified — user confirmed`, S dimension → 95); items changed are updated and upgraded; items not mentioned remain Confident. Auto Mode does not trigger bulk confirm.
