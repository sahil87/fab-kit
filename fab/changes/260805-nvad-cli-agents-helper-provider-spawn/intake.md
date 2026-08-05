# Intake: `_cli-agents` Helper Extraction + Provider-Addressable Spawn

**Change**: 260805-nvad-cli-agents-helper-provider-spawn
**Created**: 2026-08-05

## Origin

> Extract the operator's spawn/peek/send pattern into a reusable `_cli-agents` helper skill with a provider dictionary (claude/codex/gemini stable grammar + model discovery recipes), plus `fab agent --provider` flag for provider-addressable session spawning

Conversational origin (`/fab-discuss` session, 2026-08-05). The discussion started from "should agent orchestration be broken out into a standalone tool (sagent)?", narrowed to the concrete mid-session pain — switching to codex requires `codex -h` archaeology, temp-file I/O, and loses all peek/converse richness — and landed on a two-phase plan:

- **Phase 1 (this change)**: extract the generic agent-interaction procedures locked inside `fab-operator.md` into a new `_cli-agents` helper, seed it with a three-provider operational dictionary, and add provider-addressable spawning to `fab agent`.
- **Phase 2 (separate drafted change)**: an interactive stage-dispatch adapter (dispatch a pipeline stage into a watchable/steerable tmux pane), built ON these primitives.

Key decisions from the discussion are encoded as assumptions below. The standalone-tool question (sagent / rk divestment) was explicitly deferred — this change keeps everything in fab-kit but shaped for a cheap later move (one helper file + markdown dictionary).

## Why

1. **The pain point**: cross-provider agent work mid-session is unassisted today. The knowledge of how to spawn an agent CLI, deliver a prompt reliably into its TUI, peek at its output, and wait for it lives only inside `/fab-operator` — a 700-line orchestrator skill that no other skill (and no ad-hoc session) can load. It is also claude-only by convention. A session asked to "get codex's opinion on this diff" reinvents invocation flags, pipes temp files, and works blind.

2. **The consequence of not fixing it**: every cross-provider moment stays bespoke and error-prone (the printed-prompt/send-keys trap is a documented recurring failure), and Phase 2 (interactive stage dispatch) has no primitive layer to build on — it would have to re-extract or duplicate the operator's procedures.

3. **Why this approach**: the infrastructure already exists (`fab pane {map,capture,send}`, `fab agent --print`, the `providers:` table with the ho9y three-provider starter template). What's missing is (a) the *procedural* knowledge in a loadable place, (b) a *provider-addressable* spawn seam (today `fab agent` is tier-addressable only — there is no way to say "spawn a codex session" without first defining a tier whose provider is codex), and (c) a maintained *operational dictionary* for the three common agent CLIs. A helper skill + one small CLI flag delivers all three with no new tool, no new binary, and no behavior change to existing paths.

## What Changes

### 1. New helper skill: `src/kit/skills/_cli-agents.md`

A new internal helper (frontmatter like the other `_` helpers: `user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`), with two halves:

**Half A — Agent-interaction procedures**, extracted (moved, not rewritten) from `fab-operator.md`:

- **Spawn composition**: compose an interactive agent session command from a provider's `session_command` — via `fab agent --print [--repo <path>]` for tier-addressed spawns, or the new `--provider` form (§3) for provider-addressed spawns; open it in a tmux window/pane (`tmux new-window -n <name> -c <dir> "<cmd>"` per the operator's step-6 pattern).
- **Pre-send validation + delivery-probe discipline**: the operator's §3 Pre-Send Validation sequence (pane exists → agent idle per three-state `@rk_agent_state` → confirm on `active`/`waiting`/unknown), plus the delivery-probe recovery for the printed-prompt trap: a `/command` visible at an agent's `❯` prompt may be printed output, not a live buffer — probe with a literal test send, then `C-u` + retype + Enter, confirm via working spinner.
- **Peek**: `fab pane capture` for raw output; `@rk_agent_state` read for agent state — with the explicit caveat that the state option is written by run-kit's `rk agent-setup` global agent-harness hooks (Claude Code, Codex, Copilot, Gemini, OpenCode per `_cli-fab.md` § fab pane → agent state), so an **uninstrumented** pane reads `—` (unknown) and capture-pane is the universal fallback. <!-- clarified: premise corrected in review cycle 1 — writer is multi-harness, not claude-hook-based -->.
- **Await**: a poll loop (capture + state read on an interval) until a completion signal — with the honest note that no adapter recovers Agent-tool-style completion notifications; polling or an `rk notify` instructed in the worker's prompt are the available signals.

Scope boundary: only the *generic* mechanics move. Repo-targeting, worktree creation, change-pointer activation, monitored-set enrollment, dependency resolution, autopilot — all stay in `fab-operator.md` (they are operator orchestration, not agent primitives).

**Half B — Provider dictionary**: one section per provider (claude, codex, gemini), each carrying only **stable grammar + discovery recipes**, never volatile catalogs:

- Interactive vs headless entry points (`claude` / `claude -p`; `codex` / `codex exec`; `gemini` — headless reads stdin in non-TTY mode, no `-p`), consistent with the shipped ho9y starter-template strings.
- Stdin/prompt-delivery behavior and structured-output flags (`claude -p --output-format stream-json`; `codex exec --json`), resume/session semantics where they exist.
- **Model discovery recipes** — the command/probe that enumerates or verifies models on the *installed* CLI (e.g., probe `codex --version` then consult its help/model flag), instead of baked model IDs. Rationale: model catalogs rot in weeks; discovery recipes don't. Matches fab's document-don't-validate posture.
- The **codex MCP-bridge recipe** as a short subsection: registering `codex mcp` (capability-probe the installed version) as an MCP server in a claude session for tool-mediated multi-turn cross-provider conversation. Recipe text only — no fab machinery.
- Interactive quirks (first-run trust prompts, submit-key behavior) are seeded only where already confirmed by real encounters; the dictionary accretes quirks over time rather than speculating.

### 2. `_preamble.md` — register the helper

Add `_cli-agents` to the **Allowed values** list in § Skill Helper Declaration (currently `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake`).

### 3. Go: `fab agent --provider <name> [--model <id>] [--effort <level>]`

Extend `fab agent` (current signature: `fab agent [tier] [--print] [--repo <path>]`) with a provider-addressable form that bypasses tier resolution:

- `--provider <name>` looks up `providers.<name>` directly (project config per-field-merged over built-ins, exactly as `agent.ResolveProvider` does today) and composes its `session_command` via the existing `spawn.WithProfile`.
- `--model` / `--effort` supply the profile values. When omitted, the value is empty and follows the existing WithProfile empty-value rule (placeholder token + preceding flag dropped) — so `fab agent --provider codex --print` composes a bare `codex` invocation and the CLI's own default model applies.
- `--provider` is mutually exclusive with the `[tier]` positional (usage error if both given). `--print` and `--repo` compose with it unchanged.
- Unknown provider name → non-zero exit with the available provider names on stderr (this is a lookup failure, not validation of the command's content — document-don't-validate is preserved).
- Errors/behavior otherwise identical to the tier path: exec via `sh -c`, no TTY guard.

### 4. `fab-operator.md` — declare + dedupe

- Frontmatter becomes `helpers: [_cli-agents, _cli-fab, _cli-external]`.
- Body sections whose mechanics moved (Pre-Send Validation mechanics, the spawn-command composition + window-open steps, the probe/peek references) are reduced to references into `_cli-agents` sections; operator-specific policy (confirmation tiers, bounded retries, enrollment, dependency resolution, autopilot) stays verbatim.

### 5. Sweep class (declared up front per code-quality.md § Sibling & Mirror Sweeps)

- `src/kit/skills/_cli-fab.md` § fab agent — the new `--provider`/`--model`/`--effort` flags (constitution: CLI change ⇒ `_cli-fab.md` + tests).
- Go tests for the new flag paths (usage-error on tier+provider, unknown provider, empty-model composition, `--print` output).
- `docs/specs/skills/SPEC-_cli-agents.md` (new), `SPEC-fab-operator.md`, `SPEC-_preamble.md` (helper list), `SPEC-fab-draft.md`-style aggregate touch-points: `docs/specs/skills.md` and `docs/specs/glossary.md` if they enumerate helpers.
- `docs/specs/stage-models.md` — only if its `fab agent` description restates the signature (grep-verify during apply).

## Affected Memory

- `runtime/operator.md`: (modify) spawn/peek/send mechanics now referenced from `_cli-agents`; operator keeps orchestration-only content
- `runtime/providers-and-tiers.md`: (modify) `fab agent` gains the provider-addressable form; document the tier-bypass semantics
- `_shared/context-loading.md`: (modify) helper roster gains `_cli-agents` (the allowed-values list is mirrored there)
- `runtime/agent-primitives.md`: (new) the `_cli-agents` helper's own memory — procedures, provider dictionary model (stable-grammar + discovery-recipe split), uninstrumented-pane state caveat (hydrate may instead fold this into `operator.md`/`providers-and-tiers.md` if it judges a new file too thin — the split above is the default)

## Impact

- `src/kit/skills/`: `_cli-agents.md` (new), `_preamble.md`, `fab-operator.md`, `_cli-fab.md`
- `src/go/fab/`: `cmd/fab/agent*.go` (flag plumbing), reuse of `internal/agent.ResolveProvider` + `internal/spawn.WithProfile`; corresponding `_test.go` files
- `docs/specs/skills/`: `SPEC-_cli-agents.md` (new), `SPEC-fab-operator.md`, `SPEC-_preamble.md`; aggregate specs (`skills.md`, `glossary.md`) as swept
- No config schema change, no migration (no user-data restructuring), no behavior change to any existing dispatch or spawn path — the tier path and all defaults are untouched
- Enables Phase 2 (interactive stage dispatch, drafted separately) without being coupled to it

## Open Questions

*(none — the design decisions were resolved in the originating discussion; see Assumptions)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `_cli-agents` is an opt-in helper (frontmatter `helpers:`), NOT part of the always-load layer | Discussed — user asked "must load like _cli-fab?"; corrected: `_cli-*` helpers are opt-in, always-load stays lean; user agreed | S:90 R:85 A:95 D:90 |
| 2 | Certain | Provider dictionary ships stable grammar + model **discovery recipes**; no baked model catalogs | Discussed and agreed — catalogs rot monthly (shll-conformance precedent); recipes probe the installed CLI | S:85 R:80 A:90 D:85 |
| 3 | Certain | Go `defaultProviders` stays claude-only; codex/gemini remain uncomment-to-opt-in template text | Discussed — preserves ho9y decision; dictionary lives in markdown, updates per kit release without behavior risk | S:85 R:75 A:95 D:90 |
| 4 | Confident | Extraction boundary: generic spawn/peek/send/await mechanics move; repo-targeting, worktree, enrollment, deps, autopilot stay in `fab-operator.md` | Proposed in discussion, user agreed to the phase plan; boundary follows "agent primitives vs operator orchestration" | S:75 R:70 A:80 D:75 |
| 5 | Confident | `--provider` with omitted `--model` composes an empty model (WithProfile token-drop) so the CLI's own default model applies | Not explicitly discussed; follows the existing documented empty-value rule — consistent, reversible, one obvious default | S:55 R:80 A:85 D:75 |
| 6 | Confident | `--provider` is mutually exclusive with the `[tier]` positional (usage error) | Not explicitly discussed; mixing them has no coherent semantics (tier already names a provider); cheap to relax later | S:50 R:85 A:80 D:70 |
| 7 | Confident | MCP-bridge recipe rides in the dictionary as recipe text; no fab machinery, no separate phase | Discussed — "it's session config, not fab machinery; doesn't deserve its own phase" | S:75 R:90 A:85 D:80 |
| 8 | Confident | Helper name is `_cli-agents` (matches the `_cli-*` reference-helper convention) | Name used throughout the discussion without objection; pure naming, trivially reversible pre-ship | S:70 R:90 A:85 D:80 |
| 9 | Tentative | New memory file `runtime/agent-primitives.md` for the helper (vs folding into existing runtime files) | Reasonable default; hydrate stage may legitimately choose the fold-in — flagged so it decides consciously <!-- assumed: new runtime memory file for the helper; hydrate may fold into operator.md/providers-and-tiers.md instead --> | S:45 R:85 A:60 D:45 |

9 assumptions (3 certain, 5 confident, 1 tentative, 0 unresolved).
