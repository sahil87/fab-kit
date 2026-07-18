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

## Contents

- Path Convention
- Context Loading
- Skill Helper Declaration
- Naming Conventions
- Run-Kit (rk) Reference
- Common fab Commands
- Next Steps Convention
- Skill Invocation Protocol (pointer)
- Subagent Dispatch (Orchestrator Skills)
- SRAD Autonomy Framework (pointer)
- Confidence Scoring

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

This layer applies to every skill **unless the skill's own Context Loading section says otherwise** — the skill file wins. The exception set is **derived, never enumerated here**: consult the skill file itself (its `## Context Loading` section, or an explicit context note near its header) for any override — e.g., `/fab-setup` and `/docs-hydrate-memory` skip the layer entirely, `/fab-operator` loads a reduced 3-file set. A skill that declares no override loads the full layer.

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/project/config.yaml`** — project configuration: identity (name/description), `source_paths`/`test_paths`, true-impact excludes, plan-acceptance extra categories, the `providers:` table (per-provider `session_command`/`dispatch_command`), `agent.tiers` (the five role tiers), optional `stage_hooks`
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
4. **Log command**: Call `fab log command "<skill-name>" "<id>"` where `<skill-name>` is the invoking skill (e.g., `fab-continue`) and `<id>` is the `id` field from the preflight YAML output. This is best-effort — the command always exits 0 given valid usage (internal failures surface only as a stderr warning; cobra arg-count errors are usage errors that exit 2 before RunE), so no shell guard is needed.
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

**Allowed values**: `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake`.

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

### Detection (universal rule)

Before using any rk capability, check availability:

```sh
command -v rk >/dev/null 2>&1 || return  # in functions
command -v rk >/dev/null 2>&1            # in conditionals
```

If `rk` is not available, skip all rk operations silently. Never error, never warn. This fail-silent discipline applies to every rk command.

### Command Reference (delegated to `rk skill`; fab-owned usage in `_cli-external.md`)

The `rk` command surface — `rk context` (server-URL discovery, iframe windows, the proxy pattern, the Visual Display Recipe) and the `rk notify` contract — is **tool-owned**: read it at use-time via `command -v rk >/dev/null 2>&1 && rk skill` (gated, fail-silent). An installed `rk` may predate its `skill` subcommand, so **capability-probe** it — `rk skill` failing (non-zero exit or no output) is the probe — and fall back **silently** to the shll.ai bundle-page pointer `https://shll.ai/rk/skill` (present-but-old → the pointer; absent → the `command -v` gate skips entirely). The **fab-owned** rk usage — the operator's escalation `rk notify` send (message/title template) — lives in **`_cli-external.md` § rk (run-kit)**, loaded by operator skills only (not the always-load layer). The detection/fail-silent rule plus this `rk skill` delegation (with its version-skew fallback) is the only rk content every skill carries inline — the command surface itself is not.

---

## Common fab Commands

These command families cover ~90% of skill usage. See `_cli-fab` for the full reference (argument formats, every subcommand, flag details).

| Command | Purpose | Canonical form |
|---------|---------|----------------|
| `fab preflight [<change>]` | Validate init + resolve active change; outputs YAML with `id`/`name`/`change_dir`/`stage`/`display_stage`/`display_state`/`progress`/`plan`/`confidence`. Non-zero exit on error. | `fab preflight` |
| `fab score [--check-gate] [--stage intake] <change>` | Compute SRAD confidence from `intake.md` (the sole scoring source). `--check-gate` returns non-zero below the single intake gate (flat 3.0 for all types). `--stage` defaults to `intake`. | `fab score --check-gate --stage intake <id>` |
| `fab log command "<skill>" [<change>]` | Best-effort command telemetry — always exits 0 given valid usage (internal failures become a stderr warning, never an error; cobra arg-count errors are usage errors that exit 2 before RunE). No shell guard needed. | `fab log command "fab-continue" "<id>"` |
| `fab change <sub>` | Change lifecycle: `new --slug <slug>`, `switch <name>\|--none`, `resolve [<override>]`, `rename`, `list [--archive] [--show-stats]`, `archive <change>`, `restore <change> [--switch]`. | `fab resolve --folder` *(note: the query flags live on top-level `fab resolve` only — `fab change resolve` takes a bare `[<override>]`, no flags)* |
| `fab resolve [--id\|--folder\|--dir\|--status\|--pane] [<change>]` | Pure query — converts change reference to canonical output (4-char ID by default). No side effects. | `fab resolve --folder 2>/dev/null` |
| `fab status <sub> <change>` | State machine + metadata. Key subcommands: `finish <stage>` (auto-activates next), `advance <stage>`, `start <stage>`, `reset <stage>`, `skip <stage>`, `fail <stage>` (review/review-pr only), `set-change-type <type>`, `set-acceptance <field> <value>` (updates `plan:` block), `add-issue <id>`, `add-pr <url>`. | `fab status finish <id> <stage>` |

**Key behaviors** to remember without loading `_cli-fab`:

- `fab status finish <change> <stage>` auto-activates the next pending stage — never call `start` after `finish`.
- `fab status finish <change> review` auto-logs review `"passed"`; `fab status fail <change> review` auto-logs `"failed"`.
- `fab log command` is best-effort and always exits 0 (given valid usage — cobra arg-count errors are usage errors that exit 2 before RunE) — no shell guard needed (internal failures print a stderr warning only).
- `<change>` argument everywhere accepts 4-char ID, folder substring, or full folder name.
- **Failure rule**: any fab command that exits non-zero → STOP and surface stderr; resumability handles the re-run. (`fab log command` can never trip this rule through internal failure — it owns its best-effort contract and exits 0 given valid usage; a cobra arg-count error is a usage error that still exits non-zero — exit 2 — before RunE.) This rule defers to explicit per-skill handling where a skill intentionally branches on a non-zero exit.

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

> **Adoption note**: `/fab-adopt` (bring an off-pipeline change in) needs no new State Table row. It enters the pipeline late with `apply: skipped` and an `active` review, then drives review → hydrate → ship → review-pr — all states the table already covers. `apply: skipped` is not itself a `Next:` lookup target: the lookup keys on the stage that is `active`/`ready`, which after adoption is review (then the normal tail). The skipped apply stage is simply passed over by the lookup, exactly as a `done` stage is.

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

Orchestrator skills (`/fab-ff`, `/fab-fff`, and the prefix-step orchestrator `/fab-proceed`) invoke other skills as sub-operations — `/fab-ff`/`/fab-fff` run multi-stage pipelines; `/fab-proceed` runs prefix steps (`/fab-new`, `/fab-switch`, `/git-branch`) before delegating. To preserve the orchestrator's pipeline context, sub-skills are dispatched as **subagents** using the Agent tool (`subagent_type: "general-purpose"`) — never the Skill tool.

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

**Nested dispatch**: When a subagent dispatches its own sub-subagent, the inner prompt MUST also include the standard subagent context instruction. The same 5 files are loaded at every nesting level.

`general-purpose` subagents have full tool access (Read, Edit, Write, Bash, Agent) and can execute any skill behavior including file modifications and nested subagent dispatch.

### Per-Stage Model Resolution

Per-stage model selection is wired into the dispatch seam. **Immediately before dispatching each pipeline stage's sub-agent**, the dispatching skill (the orchestrators `/fab-ff`, `/fab-fff`, `/fab-proceed`, and `/fab-continue`'s own sub-agent dispatch) runs:

```sh
fab resolve-agent <stage>
```

and passes the resolved profile into the dispatch (the Agent tool on the native path, else the `fab dispatch` CLI adapter — branch on `dispatch=` per the bullets below):

- Output is a byte-stable `model=<id>` stdout line, then optional `effort=<level>` and `provider=<name>` lines (each omitted when its field is empty), **plus an optional `dispatch=<command>` line** emitted only when the resolved tier's provider carries a `dispatch_command` (the CLI-dispatch opt-in — its absence means native Agent-tool dispatch, with NO fallback to a session command). **Dispatch-seam skills branch on `dispatch=` presence** (see § CLI-Adapter Dispatch below): absent ⇒ the native Agent-tool path (`model=`/`effort=` consumed through the two seams below, byte-preserving); present ⇒ the CLI adapter (`fab dispatch`), where the profile rides the `dispatch=` command itself. See `_cli-fab.md` § fab resolve-agent.
- **Empty model** ⇒ omit the dispatch `model` param entirely (inherit the orchestrator/session model — today's behavior). **Empty effort** ⇒ omit the effort instruction (below).
- The resolver maps `<stage>` → its fixed fab-owned tier → a `{provider, model, effort}` profile, then looks up the provider's `{session_command, dispatch_command}` in the top-level `providers:` table (the `dispatch_command` drives the optional `dispatch=` line — its absence means native Agent-tool dispatch; `agent.tiers` / `providers` project overrides per-field-merged over fab-kit's defaults, with `default`-tier inheritance). The stage→tier mapping is NOT user-overridable; `agent.tiers` (tier redefinition) + `providers` (command grammar) are the override surfaces. Full design: `docs/specs/stage-models.md`.
- **No validation**: the resolved strings are passed through verbatim — fab enforces no effort enum and corrects no incompatible pair. Compatibility is the harness's concern.

**The two halves dispatch through different seams (model param + prompt instruction).** The resolved profile has a model half and an effort half, and Claude Code consumes them through *different* seams:

- **Model → the Agent tool's `model` parameter.** The Agent `model` param is a hard enum of short aliases (`opus`/`sonnet`/`haiku`/`fable`), so resolve the model half with `fab resolve-agent <stage> --alias` (emits the alias directly — see the Harness-adapter boundary note below). Empty model ⇒ omit it (inherit the session/orchestrator model).
- **Effort → an explicit instruction in the subagent prompt.** The Agent tool has **no `effort` parameter**, so the only available seam for the effort half is the dispatched prompt itself: append an imperative line such as ``Operate at `xhigh` reasoning effort for this task.`` so the sub-agent self-selects its reasoning effort. Empty effort ⇒ omit the instruction entirely (mirroring the empty-model omit rule). This is imperfect — it relies on the sub-agent honoring the instruction — but it is the only per-sub-agent effort seam available today; a per-sub-agent `effort` parameter on the Agent tool would close it cleanly and is tracked as the residual harness ask (`docs/findings/per-stage-model-tier-application.md` § Suggested directions item 4).

**Compliance visibility.** Each dispatch site MUST **surface the resolved `model=/effort=` lines** — carry them into the dispatch prompt (the effort line is already there per the seam above; co-locate the model line) and/or emit them in the orchestrator's own step output — so a *skipped* `fab resolve-agent` call (the sub-agent silently inherits the session profile) or a mis-resolved tier is **visible in output rather than silent**. There is no code-level guard fab can install (dispatch is harness-internal), so visibility is the available seam. An all-empty resolution (both `model=` and `effort=` empty) is itself worth surfacing — treat a non-empty resolved line as the expected case and a fully-empty result as a signal to flag rather than dispatch blind.

**Harness-adapter boundary (Claude Code).** The resolution (stage→tier→`{model, effort}`) is **provider-neutral**. Injecting the resolved model into the actual dispatch is **harness-specific**: for Claude Code that adapter is the **Agent tool's `model` parameter**, and the effort half is injected via the subagent-prompt instruction described above. One concrete harness detail: the Agent tool's `model` param takes a **short alias** (`opus` / `sonnet` / `haiku` / `fable`), **not** the full versioned id (`claude-opus-4-8`) the plain command emits. So for Agent-tool dispatch, **resolve the model half with `fab resolve-agent <stage> --alias`** — the `--alias` flag emits the Agent-tool-valid short alias directly on the `model=` line (empty ⇒ omit/inherit; a non-Claude override passes through verbatim), so no agent ever hand-maps the id. This is named explicitly as the Claude-Code adapter, not as universal truth — and the coupling is not new (fab's entire subagent-dispatch design is already Claude-Code-shaped). *(The operator launcher path is the deliberate exception: it resolves the **operator**-tier profile WITHOUT `--alias` — `spawn.WithProfile` composes a `claude` **CLI** invocation, which accepts full IDs. `WithProfile` is grammar-forgiving: when the resolved provider's `session_command` is a **template** containing `{model}`/`{effort}` — including the built-in claude default, which is templated, as well as a codex `session_command` — it **substitutes** the resolved values into the template (all-or-nothing; an empty value drops the placeholder's token and a preceding `-`-flag); when the command carries **no placeholder** (a plain command, e.g. a user's plain-form config carried forward from before the templated default) it instead **appends** `--model <full-id> --effort <level>` as before. Placing the built-in claude default's placeholders last makes substitution byte-identical to that former append, so a non-Claude worker CLI is configurable without the launcher emitting Claude-only flags. See `stage-models.md` § Skill wiring.)*

**Review is unexceptional.** The `review` stage dispatches a **single** review sub-agent, so it resolves `fab resolve-agent review --alias` **once** and applies the resolved profile to that one agent — exactly like every other post-intake stage (the `--alias` flag emits the Agent-tool-valid short alias on the `model=` line, per the Harness-adapter boundary note above). There is no nested reviewer dispatch and no separate merge, so review carries no special resolution rule.

**Per-stage selection applies on every post-intake stage.** Per-stage selection is a property of dispatched sub-agent runs — and **every post-intake stage now dispatches a sub-agent**, including plain `/fab-continue` (which is a one-stage sequencer that resolves `fab resolve-agent <stage>` and dispatches its stage's block — see `fab-continue.md` Normal Flow Step 1). So there is no post-intake foreground execution path left to be the exception; `fab resolve-agent` applies uniformly across apply/review/hydrate regardless of caller. Intake is pre-boundary (it runs in the main session and is not tiered by `/fab-continue`). The only residual "advisory" case is a stage skill genuinely run with no dispatch at all — fab cannot switch the session model mid-run, so such a skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch. *(The effort half of the tier is now injected via the subagent-prompt instruction described above — see "The two halves dispatch through different seams"; the lone remaining residual is a first-class per-sub-agent `effort` parameter on the Agent tool, which is a harness ask outside fab's control, not a fab gap.)*

### CLI-Adapter Dispatch (the `dispatch=` path)

This is the **canonical** cross-harness dispatch procedure. Dispatch sites (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) **reference this subsection** and do NOT restate the five-state machine. The full cross-adapter contract is `docs/specs/harness-adapters.md`; the runtime is `_cli-fab.md` § fab dispatch.

**Branch at the single `fab resolve-agent <stage> --alias` call.** Every dispatch site already makes exactly one such call and surfaces the resolved profile (§ Compliance visibility). Branch on the resolved `dispatch=` line:

- **`dispatch=` absent** ⇒ **native Agent-tool dispatch** — the two seams above (model on the Agent `model` param, effort via the prompt instruction). Unchanged; byte-preserving in behavior. This is the default for every fab-kit built-in tier (whose provider carries no `dispatch_command`).
- **`dispatch=` present** ⇒ the **CLI adapter** (`fab dispatch`). There is **NO fallback** to a session command (a provider's `session_command` is a separate, independent field; the absence of a resolved provider `dispatch_command` is itself the native-dispatch signal). The choice is per-stage/per-tier: one pipeline run can mix native and CLI dispatches across stages.

**Model/effort seams do NOT apply on the CLI path.** The `dispatch=` command ALWAYS embeds the FULL model ID and the substituted effort (via `internal/spawn` — even under `--alias`), so the Agent-tool seams (the `model` alias param + the imperative effort-prompt line) are **not** applied for a CLI dispatch — the profile rides the `dispatch=` command itself. The site keeps its single `--alias` call and branches; it makes **no second resolve call**.

**Compliance visibility extends to `dispatch=`.** Each site MUST surface the resolved `dispatch=` line alongside the `model=`/`effort=`/`provider=` surfacing, so a CLI dispatch (or a `dispatch=` line resolved but not honored) is visible in orchestrator output rather than silent.

**CLI-adapter procedure** (when `dispatch=` is present):

1. **Start.** `fab dispatch start <change> <stage>` with the full stage prompt on **stdin** — the same block prompt the Agent tool would receive, composed per § Dispatch-Prompt Obligations below. `start` resolves the tier → provider → `dispatch_command` internally and launches it detached. **No `--timeout` in v1** (orphan detection + `fab dispatch kill` cover the failure modes).
2. **Poll.** `fab dispatch status <change> <stage>` with `sleep 30` between polls (fixed cadence, no backoff in v1) until a terminal state.
3. **Five-state handling** (observed via `fab dispatch status`):
   - `running` → keep polling.
   - `done` → read `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml` as the block's returned result and proceed with the normal sequencer transition (finish/fail per the stage's contract). A review `verdict: fail` **inside** a `done` result is a **review outcome**, not a dispatch failure — the orchestrator takes the normal review-fail path.
   - `failed` → infrastructure/worker failure (a non-zero exit, incl. `124` timeout) — NOT a review-verdict fail: surface `fab dispatch logs <change> <stage> --tail N` and stop per the stage's failure path.
   - `failed (no-result)` → **contract violation** (clean exit, no result file); **NEVER treat as done** — surface logs and stop.
   - `orphaned` → the worker died with no recorded exit (reboot / `kill -9` / crash): surface and stop with re-run guidance (`fab dispatch start` over a completed/orphaned attempt overwrites it).
4. **No cleanup after `done`.** `.fab-dispatch/` is transient comms with **no automatic GC** — cleanup is archive-time deletion + explicit `fab dispatch clean` only. The wiring adds no cleanup call after a `done` dispatch.

### Dispatch-Prompt Obligations (bind BOTH adapters)

Per `docs/specs/harness-adapters.md` § Dispatch-prompt obligations, **whatever adapter dispatches a stage**, the prompt handed to the worker MUST:

1. **Instruct the worker to produce `{stage}-result.yaml`** — for the **CLI adapter** a real file at `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml`; for the **native adapter** the structural equivalent (the returned result). The result is the contract's success token — its **presence** is required for `done` (a clean exit without it is `failed (no-result)`). Minimal schema (3d):

   ```yaml
   # apply (mirrors "returns completion status or failure with task ID and reason")
   stage: apply
   status: success            # success | failure  — the WORKER/infra outcome
   summary: "12/12 tasks complete, tests green"
   # on failure only:
   failed_task: T007
   reason: "tests failing in internal/x after 3 attempts"
   ```

   ```yaml
   # review (mirrors "merged prioritized findings + pass/fail verdict")
   stage: review
   status: success            # the review RAN to completion (infra outcome)
   verdict: pass              # pass | fail  — the REVIEW outcome (distinct from status)
   findings:
     must_fix: []             # each finding a self-contained string (file/line refs inline)
     should_fix:
       - "src/x.md:41 — stale claim Y"
     nice_to_have: []
   summary: "2 should-fix, verdict pass"
   ```

   ```yaml
   # hydrate (mirrors "returns completion status")
   stage: hydrate
   status: success
   summary: "updated docs/memory/runtime/dispatch.md, regenerated indexes"
   ```

   The **`status` vs `verdict` split is load-bearing**: a completed review with `verdict: fail` is dispatch-state `done` (result present) — the orchestrator then takes the normal review-fail path. Dispatch-state `failed` is reserved for worker/infrastructure failure.
2. **Carry the standard subagent context files** — `fab/project/config.yaml`, `fab/project/constitution.md`, and (optional) `context.md` / `code-quality.md` / `code-review.md` (§ Standard Subagent Context). Already true for native prompts; the CLI prompt content MUST carry the same instruction — a worker on a fresh harness has no other awareness of project principles.
3. **End with a terminal `fab status refresh` epilogue** so the worker recomputes state from artifacts after finishing (the 3a pull-based recompute). This is the sole `fab status` command a dispatched block runs — see the block-contract carve-out below.

**Block-contract carve-out.** The universal block-contract line the dispatch sites carry — "do NOT run `fab status` commands; return results only" — is refined to prohibit `fab status` **transition** commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) while **REQUIRING** the terminal `fab status refresh`: refresh is a pull-based recompute, not a transition, so it does not violate the invariant that **the orchestrator (sequencer) owns all transitions**. Both adapters' block prompts carry this carve-out.

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
- `/fab-new` and `/fab-draft` (after intake generation, `--stage intake`) — persist the intake score
- `/fab-clarify` (**both modes** — Suggest Step 7 and Auto Mode step 4) — re-persists the intake score after resolving assumptions

`/fab-continue` does NOT score at apply entry — intake is authoritative, and there is no scoring at any post-intake stage.

Both `/fab-ff` and `/fab-fff` gate at a single point: the intake gate via `fab score --check-gate --stage intake` before starting the automated bracket. The `--force` flag on either skill bypasses it.

### Bulk Confirm (pointer)

`/fab-clarify` offers a bulk-confirm flow for Confident assumptions — defined in `fab-clarify.md` (Step 2, Suggest Mode), the sole authority for its trigger and semantics.
