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

- **`fab/project/config.yaml`** — project configuration: identity (name/description), `source_paths`/`test_paths`, true-impact excludes, plan-acceptance extra categories, the `providers:` table (independent per-provider `interactive_command`/`headless_command`/`native` capabilities plus the per-role `profiles.<role>.{model,effort}` fills), `agent.session`/`agent.workers` (the two depth knobs) and the sparse `agent.profiles` per-role overrides, `dispatch.mode` (the `pane → native → headless` preference ceiling; default `native`), `dispatch.column_width` (the pane-worker column's width) and `dispatch.reap_done` (reclaim a done pane worker's pane; default `true`), optional `stage_hooks`
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
4. **Log command**: Call `fab log command "<skill-name>" "<id>"` where `<skill-name>` is the invoking skill (e.g., `fab-continue`) and `<id>` is the `id` field from the preflight YAML output.
5. Load all completed artifacts in the change folder (`intake.md`, `plan.md`) — read each file that exists so you have full context of what has been decided so far.

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

**Allowed values**: `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake`.

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

### Command Reference (delegated to `rk skill`; fab-owned usage in `_cli-external.md`)

The `rk` command surface — `rk context` (server-URL discovery, iframe windows, the proxy pattern, the Visual Display Recipe) and the `rk notify` contract — is **tool-owned**: read it at use-time via `command -v rk >/dev/null 2>&1 && rk skill` (gated, fail-silent). An installed `rk` may predate its `skill` subcommand, so **capability-probe** it — `rk skill` failing (non-zero exit or no output) is the probe — and fall back **silently** to the shll.ai bundle-page pointer `https://shll.ai/rk/skill` (present-but-old → the pointer; absent → the `command -v` gate skips entirely). The **fab-owned** rk usage is indexed in **`_cli-external.md` § rk (run-kit)**, loaded by operator skills only (not the always-load layer): the operator's escalation `rk notify` send (message/title template) owned there in place, plus a pointer to the operator's startup role self-mark (owned by `fab-operator.md` §2 Startup). The detection/fail-silent rule plus this `rk skill` delegation (with its version-skew fallback) is the only rk content every skill carries inline — the command surface itself is not.

---

## Common fab Commands

These command families cover ~90% of skill usage. See `_cli-fab` for the full reference (argument formats, every subcommand, flag details).

| Command | Purpose | Canonical form |
|---------|---------|----------------|
| `fab preflight [<change>]` | Validate init + resolve active change; outputs YAML with `id`/`name`/`change_dir`/`stage`/`display_stage`/`display_state`/`progress`/`plan`/`confidence`. Non-zero exit on error. | `fab preflight` |
| `fab score [--check-gate] [--stage intake] <change>` | Compute SRAD confidence from `intake.md` (the sole scoring source). `--check-gate` returns non-zero below the single intake gate (flat 3.0 for all types). `--stage` defaults to `intake`. | `fab score --check-gate --stage intake <id>` |
| `fab log command "<skill>" [<change>]` | Best-effort command telemetry — always exits 0 given valid usage (internal failures become a stderr warning, never an error; cobra arg-count errors are usage errors that exit 2 before RunE). No shell guard needed. | `fab log command "fab-continue" "<id>"` |
| `fab change <sub>` | Change lifecycle: `new --slug <slug>`, `switch <name>\|--none`, `resolve [<override>]`, `rename`, `list [--archive] [--show-stats]`, `archive <change>`, `restore <change> [--switch]`. | `fab resolve --folder` *(note: the query flags live on top-level `fab resolve` only — `fab change resolve` takes a bare `[<override>]`, no flags)* |
| `fab resolve [--id\|--folder\|--dir\|--status\|--pane] [--or-none] [<change>]` | Pure query — converts change reference to canonical output (4-char ID by default). No side effects. `--or-none` makes absence a first-class result: prints exactly `(none)`, exit 0 (not-found always; ambiguous only bare — real errors stay non-zero). The probe form for "is a change active?"; `fab preflight` stays the strict validation gate. | `fab resolve --folder --or-none` |
| `fab status <sub> <change>` | State machine + metadata. Key subcommands: `finish <stage>` (auto-activates next), `advance <stage>`, `start <stage>`, `reset <stage>`, `skip <stage>`, `fail <stage>` (review/review-pr only), `set-change-type <type>`, `set-acceptance <field> <value>` (updates `plan:` block), `add-issue <id>`, `add-pr <url>`. | `fab status finish <id> <stage>` |

**Key behaviors** to remember without loading `_cli-fab`:

- `fab status finish <change> <stage>` auto-activates the next pending stage — never call `start` after `finish`.
- `fab status finish <change> review` auto-logs review `"passed"`; `fab status fail <change> review` auto-logs `"failed"`.
- `<change>` argument everywhere accepts 4-char ID, folder substring, or full folder name.
- **Failure rule**: any fab command that exits non-zero → STOP and surface stderr; resumability handles the re-run. This rule defers to explicit per-command or per-skill handling.

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

The `[AUTO-MODE]` prefix contract and `/fab-clarify`'s machine-readable Auto Mode are defined in `fab-clarify.md` § Skill Invocation Protocol and § Auto Mode.

---

## Subagent Dispatch (Orchestrator Skills)

Orchestrator skills (`/fab-ff`, `/fab-fff`, and the prefix-step orchestrator `/fab-proceed`) invoke other skills as sub-operations — `/fab-ff`/`/fab-fff` run multi-stage pipelines; `/fab-proceed` runs prefix steps (`/fab-new`, `/fab-switch`, `/git-branch`) before delegating. To preserve the orchestrator's pipeline context, sub-skills are dispatched as **subagents** using the Agent tool (`subagent_type: "general-purpose"`) — never the Skill tool.

**Why not the Skill tool?** The Skill tool expands the sub-skill's prompt into the orchestrator's execution context. After the sub-skill completes, the pipeline context is lost and execution halts. The Agent tool runs the sub-skill in a **separate context** and returns a structured result, keeping the pipeline intact.

**Dispatch pattern** — each subagent prompt includes:

1. The skill file to read (deployed to `.claude/skills/{skill}/SKILL.md`)
2. The specific behavior section to follow (e.g., "Apply Behavior", "Auto Mode")
3. The change ID for resolution
4. The expected return format
5. The standard subagent context files (see below)

### Standard Subagent Context

Every subagent **dispatch** prompt MUST instruct the subagent to read the following project files **before** executing its task. This ensures subagents operate with full awareness of project principles, constraints, and conventions — regardless of nesting depth. A *continuation* message to an already-running named worker is not a dispatch and deliberately does **not** re-carry them (§ Worker Continuation).

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

Immediately before dispatching a pipeline-stage sub-agent, the dispatching skill runs:

```sh
fab resolve-agent <stage> --alias
```

Output is byte-stable and ordered: `model=<alias-or-id>` first, then optional `effort=<level>`, `provider=<name>`, and `dispatch=<command>` lines; each optional line is omitted when empty. `--alias` adapts recognized Claude model IDs for the native Agent-tool seam; non-Claude IDs pass through, while `dispatch=` always embeds the full ID. The resolver's role/depth/fill precedence and the conditions that emit `dispatch=` live in `_cli-fab.md` § fab resolve-agent; `docs/specs/stage-models.md` owns the design. Resolved strings pass through verbatim — fab validates no effort enum and corrects no incompatible pair.

| Profile half | Native Agent-tool seam | Empty value | Constraint |
|--------------|------------------------|-------------|------------|
| Model | Agent tool `model` parameter, using `fab resolve-agent <stage> --alias` | Omit the parameter; inherit the orchestrator/session model | The parameter accepts the Claude aliases `opus`/`sonnet`/`haiku`/`fable` |
| Effort | Imperative prompt instruction, e.g. ``Operate at `high` reasoning effort for this task.`` | Omit the instruction | Advisory on the native arm because session effort dominates (GitHub #64033/#39220); binding only when `--effort` rides a composed CLI command |

User-directed overrides are per invocation and ride the same single `fab resolve-agent <stage> --alias` call:

| Override kind | Binds where | Executable path |
|---------------|-------------|-----------------|
| Within-Claude `--model` / `--effort` | Native Agent-tool arm | Use the two native seams above |
| Cross-provider `--provider` | Native arm only; it **CANNOT move a stage onto CLI dispatch** because `fab dispatch start` takes no override flags and re-resolves from config | A `dispatch=` line caused only by the override is not actionable; a non-Claude model has no native Agent-tool seam |
| Config override (`agent.workers` / `agent.session` / `agent.profiles.<role>.provider`) | Resolver and `fab dispatch start` | Sole executable cross-provider path; built-in providers refill from their own `profiles`, while a provider with no fills resolves empty/inherit unless paired with `--model` or configured fills |

Every dispatch site MUST surface the resolved `model=`/`effort=`/`provider=`/`dispatch=` lines; an all-empty resolution is a signal to flag, not a reason to dispatch blind. `--alias` emits the Agent-tool-valid short alias directly. The operator launcher is the deliberate exception: it resolves without `--alias`; `WithProfile` substitutes `{model}`/`{effort}` in templated `interactive_command` values (dropping an empty placeholder and its preceding flag) or appends `--model <full-id> --effort <level>` to plain commands. See `docs/specs/stage-models.md` § Skill wiring.

Every post-intake stage uses this resolution before its dispatched sub-agent, including plain `/fab-continue` (the one by-design no-dispatch exception: the `/fab-ff`/`/fab-fff` light lane runs non-review stages inline — `_pipeline.md` § Light Lane); intake remains in the main session. A stage skill genuinely run without dispatch MAY report the configured profile but MUST NOT attempt to switch the session model.

### Worker Continuation (native and pane arms)

An auto-rework-capable orchestrator MAY keep its **apply** worker alive across rework cycles instead of paying a cold start per cycle. A continued worker still holds the always-load layer, `intake.md`, `plan.md`, the affected memory, and the source files it just wrote — and, unlike a fresh worker, it also remembers what the reviewer already rejected. `_pipeline.md` § Auto-Rework Loop wires this; the mechanics live here.

The two resumable arms differ only in **how the worker is reached** — a named in-context handle on the native arm, a still-live tmux pane on the pane arm — and share every rule that matters: apply-only scope, a mandatory fresh-dispatch fallback, profile fixity, and reviewer independence. The rules below are the **native** arm's; § Pane-arm continuation states the pane arm's one different mechanism.

- **Naming.** When the native branch dispatches the **apply** stage from an auto-rework-capable orchestrator (`/fab-ff`, `/fab-fff`, and `/fab-adopt` as a partial consumer of that loop), the Agent call passes `name: "apply-{id}"`, where `{id}` is the 4-char change ID — e.g. `apply-tv3g`.
- **Continuation.** A later rework cycle resumes that worker by sending the new instructions to `apply-{id}` (SendMessage) instead of spawning a fresh agent. The continuation prompt carries the **triaged findings** and the **rework action** the orchestrator chose, and instructs the worker to RE-READ from disk every artifact the orchestrator edited at that item — always plan.md — because its in-context copy predates those edits. It re-states the block contract: return results only, run no `fab status` **transition** command (`start`/`advance`/`finish`/`reset`/`fail`/`skip`), and end with the terminal `fab status refresh <change>`. It deliberately does **NOT** re-carry the standard subagent context files (§ Standard Subagent Context) — the worker already holds them, which is the entire point of continuing it.
- **Fallback rule (load-bearing).** Continuation is an **optimization, never a correctness dependency**. Reachability is established **by attempting the send** — there is no separate probe, and a send that fails or errors *is* the unreachable signal. Dispatch fresh — per the ordinary Stage Dispatch Procedure, including the full § Dispatch-Prompt Obligations — whenever the named worker is unreachable:

  | Unreachable because | Consequence |
  |---------------------|-------------|
  | The orchestrator session was resumed or restarted (handles do not survive) | Fresh dispatch |
  | The harness has no named-agent / SendMessage capability | Fresh dispatch |
  | The send errors | Fresh dispatch |
  | The worker was never named (e.g. the stage went through the CLI adapter) | Fresh dispatch |

  A fresh fallback dispatch **re-establishes the name wherever the native branch and the naming capability exist**, so subsequent cycles can continue it. The fallback path is today's behavior verbatim — same prompt, same obligations, same transitions — so a broken resume path degrades to the status quo, never to a pipeline failure (Constitution III).
- **Profile fixity.** A resumed worker keeps the model and effort it was first dispatched with; `fab resolve-agent apply --alias` is **NOT** re-run on the resume path. Resolution runs — and is surfaced — only on **fresh** dispatches, initial or fallback. Re-resolving on a resume would surface a value the cycle cannot honor, which is the opposite of what the surfacing rule is for.
- **Scope guard.** Continuation exists **only** for the apply worker inside the auto-rework loop. **Review workers are never named and never continued** (`_pipeline.md` Auto-Rework Loop item 4's fresh-worker rule is reviewer-independence design and is untouched). Hydrate and every other stage are unaffected and always dispatch fresh. Within the CLI-adapter branch, **headless is non-resumable by decision**; the pane arm resumes through its own mechanism, below.

#### Pane-arm continuation (the `dispatch=` branch)

A pane worker never exits on completion — it writes its result and sits at its prompt — so the worker the native arm keeps in memory is, here, simply still on screen. Continuation is therefore the **same verified delivery step pointed at a different prompt**, and it is available exactly when the apply pane was not reaped (§ CLI-Adapter Dispatch step 3's stage-aware timing exists for this).

1. Write the continuation prompt to `.fab-dispatch/{4-char-change-id}/apply-continuation.md`. Its CONTENT rules are the native arm's verbatim: the triaged findings, the chosen rework action, and the instruction to RE-READ from disk every artifact you edited at that item — always plan.md. It carries § Dispatch-Prompt Obligations 1 and 3 (result file, terminal `fab status refresh <change>`) and the block-contract carve-out, and — being a continuation, not a dispatch — deliberately does **not** re-carry the standard context files.
2. Run `fab dispatch deliver <change> apply --prompt-file .fab-dispatch/{id}/apply-continuation.md`. A verified delivery clears the previous cycle's result file, so the dispatch reads `running` again and step 2's `wait` observes the new cycle rather than returning on the old result; a delivery that never verifies leaves that file in place, which is what keeps step 3's fallback executable without a `kill`.
3. **Fallback rule (load-bearing, same as the native arm).** Reachability is established **by attempting the delivery** — there is no separate probe. Any failure (the pane was reaped or killed, the record is not pane-mode, the choreography exhausted its retry, `deliver` refused because the worker is still mid-stage) means dispatch **fresh** through the ordinary open → gate → deliver path with the full obligations. Resume is an optimization, never a correctness dependency.
4. **Profile fixity** holds identically: a resumed pane keeps the model and effort it was opened with, and `fab resolve-agent apply --alias` is NOT re-run on the resume path.

### CLI-Adapter Dispatch (the `dispatch=` path)

This is the canonical cross-harness dispatch procedure. Dispatch sites (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) reference it instead of restating the machine; `docs/specs/harness-adapters.md` owns the three-adapter contract and `_cli-fab.md` § fab dispatch owns runtime details.

Branch once, at the single surfaced `fab resolve-agent <stage> --alias` result:

| `dispatch=` | Branch | Profile handling |
|-------------|--------|------------------|
| Absent | Native Agent-tool dispatch; the resolver selected the native rung | Use the model/effort seams above |
| Present | CLI adapter (`fab dispatch`); stages may mix adapters | The line carries the selected pane/headless command with full model ID and substituted effort, so do not apply the native seams, execute the value, or resolve again |

`dispatch.mode` is a preference ceiling over `pane → native → headless`. The resolver starts at that rung and descends only, using provider capabilities (`interactive_command`, `native`, `headless_command`) plus `$TMUX`; command presence describes how a rung runs and never selects policy by itself. The result still has only the two branches above: `dispatch=` is absent iff native resolved and present with the already-substituted pane/headless command otherwise. Dispatch sites branch only on presence and never execute the value; `_cli-fab.md` § fab resolve-agent owns the ladder and no-rung errors.

When `dispatch=` is present:

1. **Launch, by mode — and `start` is how you learn the mode.** The two modes have different entries, because a pane worker is spawned and delivered to in separate steps, but the `dispatch=` line is **unlabelled**: it carries a substituted command and never says which rung produced it. So do not try to infer the rung — **attempt `start` first and let its answer be the discriminator.** That probe is free: a pane landing is refused *before* stdin is read, before the refuse-if-running check, and before any state write, so nothing is consumed and the identical invocation can be re-run as `open`. (One cheap shortcut, since the ladder descends only and never ascends: a `dispatch.mode` other than `pane` can never land on pane, so under the default `native` the `dispatch=` branch is always headless.)
   - **Headless — `start` launched.** Send the full stage prompt on stdin to `fab dispatch start <change> <stage>`. Pass no `--timeout` by default: `start`/`restart --timeout` bounds and KILLS the headless worker; `wait --timeout` in step 2 bounds only the OBSERVER. Use a start timeout only on explicit direction.
   - **Pane — `start` refused, naming `fab dispatch open`.** `start` launches only the headless arm and never opens a pane itself. Run § The pane readiness gate below — `open` → gate → `deliver` — then continue at step 2. **Remember this landing**: it is the pane arm, and step 3's deferred apply reap fires only here.
2. **Wait (push, not poll).** Run blocking `fab dispatch wait <change> <stage> --timeout 300`, preferably as a background command through the harness notify-on-exit seam; use the same call in the foreground when that seam is unavailable. `wait` re-derives state on its internal ~2s tick, returns immediately on a terminal state, and prints `running` when its five-minute peek bound expires. Timeout expiry exits **0**; only real errors are non-zero. Never replace it with a `sleep`/`status` poll loop, and change the peek cadence only on explicit direction. A pane-mode no-progress timeout-return is the **stall guard**'s case — § The pane readiness gate owns it.
3. **Handle the printed state.** Use the table below. On `done`, read the result and reap. **WHEN you reap is stage-aware**, because the apply pane is the pane arm's resume target:

   | Stage | Reap `fab dispatch reap <change> <stage>` |
   |-------|-------------------------------------------|
   | apply | **NOT** at done-read — the pane must survive for a rework cycle to be delivered into. Reap it once **review passes**, or when the run stops/escalates past apply for good — and **only if step 1 landed on the pane arm** |
   | every other stage | immediately after reading its `done` result, as before |

   At **done-read** the call needs no mode or config check: you are inside the `dispatch=` branch that just wrote this stage's record, so the record is there and the Go-side guard settles the rest (pane + `done` + `dispatch.reap_done`), reporting every no-op and exiting 0.

   The **deferred apply reap is different, and is gated on the arm**: it fires at a moment the pipeline reaches on *every* arm, including the ones that wrote no `.fab-dispatch/` record at all. Run it **only when step 1's apply launch landed on pane** (equivalently: `.fab-dispatch/{4-char-change-id}/apply.yaml` exists and carries a `pane:` — the check for an orchestrator that lost that context). On the native arm — the shipped `dispatch.mode: native` default — apply never produced a record, and **a missing record is one of the two cases `fab dispatch reap` treats as a real error rather than a no-op** (`_cli-fab.md` § reap owns the exit-code surface), so an ungated call would exit non-zero and the standing Failure rule would halt the pipeline right after every passing review. Within the pane arm the call stays unconditional and dumb as ever — no mode or config check, only the *when*.
4. **Keep state after `done`.** `.fab-dispatch/` has no automatic GC: archive-time deletion and explicit `fab dispatch clean` are its cleanup paths. Reap is pane hygiene and removes no files; the record, result, prompt, and log survive, so a reaped dispatch still reads `done`.

| State | Meaning | Action |
|-------|---------|--------|
| `running` | Worker is live, or the wait bound expired | On timeout, take the read-only peek below, classify, and re-arm `wait` |
| `done` | Result exists; review `verdict: fail` remains a review outcome | Read `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml`, reap per step 3's stage-aware timing, then take the normal sequencer transition |
| `failed` | Worker/infrastructure exited non-zero, including `124`; not a review verdict | Surface `fab dispatch logs <change> <stage> --tail N`; restart only for a clearly transient signature and within budget |
| `failed (no-result)` | Clean exit without the required result; contract violation | Never treat as done and never restart; surface logs and escalate |
| `orphaned` | Worker died without a recorded exit | Spend the one automatic restart if available, then re-arm `wait`; otherwise escalate |

**Recovery policy — bounded, and the orchestrator's alone.** A restarted stage resumes from artifact checkpoints and the persisted `{stage}-prompt.md`; `fab dispatch restart` re-derives mode from the current environment.

| State/event | Automatic restart? | Budget |
|-------------|--------------------|--------|
| `orphaned` | Yes | Spend the single restart, then re-arm `wait` |
| `failed` | No | MAY spend that same restart only when the log tail is clearly transient (provider 5xx, overload, or rate-limit exhaustion) |
| `failed (no-result)` | Never | Escalate; a contract violation needs eyes |
| Parked/dead-ended `running` worker | Kill, then restart | Spend the same single restart |

The budget is exactly ONE restart per stage dispatch, tracked only in THIS orchestrator's context: no disk counter or attempt history. After context loss, at worst one extra restart occurs.

On every timeout-return of `fab dispatch wait`, take a read-only peek (`fab dispatch logs <change> <stage> --tail 40` for headless; the pane command below for pane mode) and classify:

| Classification | Action |
|----------------|--------|
| (a) Visibly progressing | Re-arm `wait`; peek again on its next timeout-return |
| (b) Parked at an error banner / dead-ended | `fab dispatch kill <change> <stage>`, then restart within the same budget; escalate if spent |
| (c) Waiting on genuine human input | Escalate without killing; a human may answer |
| (d) Delivered but silent — no result file, screen unchanged against the previous round | The **stall guard** (§ The pane readiness gate, the owner) decides: re-enter the gate's judgment rounds or re-arm |

Escalation surfaces per-mode evidence, sends `rk notify` only behind the fail-silent `command -v rk` gate, and stops on the existing failure path; it creates no state or transition. **The pipeline NEVER sends keys to a WORKER.** Its verb set against one is exactly **peek** (read-only), **kill**, **restart**, **notify**, **stop**, **reap**. Nudging and answering a delivered worker remain operator/user actions. The single carve-out is the **pre-delivery pane** — see § The pane readiness gate, where a pane that has not yet been delivered to is not yet a worker. **`reap` is NOT `kill`**: kill is ungated recovery; reap is `done`-only, knob-gated hygiene that cannot terminate `running`, `orphaned`, or `failed` work.

**Pane mode is an option inside the `dispatch=`-present branch, never a third branch:**

- Automatic mode passes no flag: `fab dispatch` re-resolves the configured `dispatch.mode` against current capabilities and environment. Force `--pane` or `--headless` on `restart` only for a one-shot override; the flags are mutually exclusive and forced prerequisites hard-error. `fab dispatch open` needs neither — it IS the pane entry, so pane is explicit there and a missing prerequisite hard-errors rather than descending. Automatic selection descends softly, never ascends, and surfaces `mode: <rung> (preferred)` or `mode: <rung> (descended: <reason>[; <reason>])`. Reasons are exactly `pane unavailable: no tmux`, `pane unavailable: tmux unreachable`, `pane unavailable: no interactive_command`, and `native unavailable`; combinations preserve ladder order. If re-resolution lands on native, `fab dispatch` errors before writing state and tells the caller to re-run `fab resolve-agent` for native dispatch; if it lands on pane, `start` errors the same way and names `open`.
- Pane mode reaches only `running` / `done` / `orphaned`; `failed` and `failed (no-result)` are unreachable without an exit-code channel. It inherits the same prompt, wait, recovery, and one-restart budget, and the same reap handling at the stage-aware moment step 3 fixes.
- Pane output is tmux scrollback, not `{stage}.log`. Use `fab pane capture [-L <server>] <pane>` for peek/escalation — the socket rides `fab dispatch status <change> <stage> --json` (the `server` key) and the exact socket-included command is printed by `fab dispatch logs <change> <stage>`. A non-`ready` `fab dispatch ready` report carries the pane and its socket inline, so the gate needs no separate lookup.
- Steering is contract-neutral: the worker still owes `{stage}-result.yaml` and terminal `fab status refresh <change>`, runs no transition command, and the orchestrator owns all transitions. Pane placement, column/sibling mechanics, identities, fallbacks, and the `opened …` / `delivered …` output forms live only in `_cli-fab.md` § fab dispatch.

#### The pane readiness gate

Step 1's pane branch is **open → gate → deliver**, and only then step 2's `wait`. It exists because a freshly spawned agent TUI is not necessarily ready to be typed at: it may still be booting, or parked behind a trust dialog, a survey, or a login wall. Classifying that screen is mechanical (the runtime's job); deciding what it wants is judgment (yours).

1. **`fab dispatch open <change> <stage>`** with the full stage prompt on stdin. The pane opens; nothing is delivered.
2. **Loop `fab dispatch ready <change> <stage>`** and branch on the reported word. Every non-`ready` report carries the pane, its socket, and a capture snippet, so you never need a second capture call.

   | Report | Do |
   |--------|-----|
   | `ready` | proceed to step 3 |
   | `booting` | wait briefly and re-probe. Boot re-probes do **NOT** consume judgment rounds; after **5 consecutive** `booting` reports treat the pane as `parked` and enter the rounds, so a TUI that never finishes starting cannot spin the loop forever |
   | `parked` | spend a **judgment round** (below) |

3. **`fab dispatch deliver <change> <stage>`**, then continue at step 2 of the numbered procedure (`wait`).

**Judgment rounds — the carve-out, and its bounds.** A pane between `open` and a successful `deliver` holds no stage context, so there is nothing a keystroke could corrupt: it is not yet a worker, and the no-input-injection rule has no subject. In that window you MAY answer the wall yourself with raw `tmux [-L <server>] send-keys -t <pane> …`, reading the snippet to decide what it wants.

- **Budget: at most 2 rounds per gate.** Re-probe after each. Still not `ready` after the second ⇒ escalate.
- **Login and credential walls escalate IMMEDIATELY and are never answered** — no round is spent, whatever the budget's state.
- **Escalate, do not descend.** Gate exhaustion surfaces the capture evidence, sends `rk notify` behind the fail-silent `command -v rk` gate, and stops on the existing failure path, leaving the pane **alive** for a human. It does NOT fall back to headless: descent is a pre-launch capability decision, and re-making it here would silently change which adapter ran the stage.
- **From successful delivery onward the ordinary rule applies again** — a wall that appears mid-stage escalates, exactly as before. One qualification: a FALSE-VERIFIED delivery is not a successful one. A worker that never received its prompt holds **no stage context**, so the no-input-injection rule still has no subject and the judgment rounds stay legal against it — which is what the stall guard below exists to detect.

**Stall guard — the delivered-but-silent worker.** When a `wait` round (step 2) returns `running` on its expired bound AND no `{stage}-result.yaml` has appeared AND the peeked screen is UNCHANGED against the previous round, judge the captured screen BEFORE re-arming another round: a first-run wall or a **bare shell prompt** means the delivery never happened (the false-verify the gate's takeover precondition exists to prevent — a bare shell prompt on screen is the conclusive signal), so the judgment rounds above re-enter; any other screen means the worker is doing something the peek cannot see, so re-arm. The guard is **capture-based and read-only** — it never re-runs a readiness verb: `fab dispatch ready` refuses exactly this state (`delivered: true` with no result is the mid-stage worker its guard rejects), and `fab pane ready` would type a sentinel into a possibly-live worker. It is a bounded diagnostic on a no-progress timeout-return — never a poll-loop step.

The gate **amortizes**: first-run walls are mostly workspace-scoped (trust is per-worktree-path), so the first gate pass in a checkout clears them for every later pane worker there and subsequent gates read `ready` on the first probe. It MAY also be run ahead of apply as a warm-up — same three steps, no special casing.

### Dispatch-Prompt Obligations (bind ALL THREE adapters)

Per `docs/specs/harness-adapters.md` § Dispatch-prompt obligations, **whatever adapter dispatches a stage** — native Agent-tool, headless CLI, or interactive pane — the prompt handed to the worker MUST:

1. **Instruct the worker to produce `{stage}-result.yaml`** — for **both `fab dispatch` modes** a real file at `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml`; for the **native adapter** the structural equivalent (the returned result). The result is the contract's success token — its **presence** is required for `done` (a clean exit without it is `failed (no-result)`). On the **pane** mode the result file is the *sole* completion signal (an interactive worker never exits on completion), so the obligation is load-bearing there rather than merely contractual. Minimal schema (3d):

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
2. **Carry the standard subagent context files** — `fab/project/config.yaml`, `fab/project/constitution.md`, and (optional) `context.md` / `code-quality.md` / `code-review.md` (§ Standard Subagent Context). Already true for native prompts; the CLI prompt content MUST carry the same instruction — a worker on a fresh harness has no other awareness of project principles. **This obligation binds every *dispatch***; a **continuation** message to an already-running named worker carries obligations 1 and 3 only, because the worker already holds the context files (§ Worker Continuation).
3. **End with a terminal `fab status refresh <change>` epilogue** (the worker substitutes the 4-char change ID it was dispatched with) so the worker recomputes state from artifacts after finishing (the 3a pull-based recompute). This is the sole `fab status` command a dispatched block runs — see the block-contract carve-out below.

**Delivery mechanism varies; the obligations do not.** *How* the prompt reaches the worker is adapter-specific — the dispatched prompt itself (native), the command's **stdin** (headless CLI), or a **prompt file** plus a one-line **pointer** to it that `fab dispatch deliver` types into the interactive worker after `open` (pane). Compose the **dispatch** prompt content **identically in every case**: the pane worker that follows its pointer is reading the same block prompt, with the same three obligations above. Nothing about the prompt is written differently for the pane arm; only `fab dispatch` chooses how it is handed over. (A continuation message is not a dispatch — see obligation 2's carve-out.)

**Block-contract carve-out.** The universal block-contract line the dispatch sites carry — "do NOT run `fab status` commands; return results only" — is refined to prohibit `fab status` **transition** commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) while **REQUIRING** the terminal `fab status refresh <change>`: refresh is a pull-based recompute, not a transition, so it does not violate the invariant that **the orchestrator (sequencer) owns all transitions**. Every adapter's block prompt carries this carve-out — including a pane dispatch, where a user may converse with the worker mid-stage: **steering is contract-neutral**, so a steered worker still owes its result file and its terminal refresh, and still never runs a transition command.

---

## SRAD Autonomy Framework (pointer)

SRAD is the decision framework planning skills use to score decision points (Signal, Reversibility, Agent Competence, Disambiguation → Certain/Confident/Tentative/Unresolved) and decide when to ask vs. assume. The full framework — scoring dimensions, grade thresholds, Critical Rule, artifact markers, and the Assumptions Summary block — lives in the `_srad` helper, declared via `helpers:` by the planning skills (`fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`). Non-planning skills do not need it.

---

## Confidence Scoring

`fab score` computes confidence from `intake.md`; agents never compute it. `_cli-fab.md` § fab score owns the `.status.yaml` schema, formula, and template details.

### Gate Threshold

| Gates | Stage/source | Threshold | Checked by | Bypass / future change |
|-------|--------------|-----------|------------|------------------------|
| One | Intake / `intake.md` | Flat **3.0** for all seven change types | `/fab-ff` and `/fab-fff` via `fab score --check-gate --stage intake` | `--force` bypasses; `getGateThreshold(changeType)` makes future per-type divergence data-only |

A failed gate prevents the automated bracket from entering apply. Apply has no Unresolved-to-intake bounce; the SRAD Critical Rule is enforced by intake-time skills (`/fab-new`, `/fab-clarify`).

See `docs/specs/change-types.md` for the full taxonomy.

### Invocation

- `/fab-new` and `/fab-draft` persist the intake score through `_intake.md` Step 7.
- `/fab-clarify` re-persists it after resolving assumptions in both modes.
- `/fab-ff` and `/fab-fff` check the one intake gate before the automated bracket.
- `/fab-continue` never scores at apply entry or later; intake is authoritative.

### Bulk Confirm (pointer)

`/fab-clarify` offers a bulk-confirm flow for Confident assumptions — defined in `fab-clarify.md` (Step 2, Suggest Mode), the sole authority for its trigger and semantics.
