# Intake: Fix agent skill invocation prefixes

**Change**: 260908-tfzn-fix-agent-skill-prefix
**Created**: 2026-09-08

## Origin

The user asked for an audit of commands injected into agent TUIs, then specified: “anything non claude, non codex, default to / command.” The agreed rule is Codex → `$skill-name`, every other or unknown provider → `/skill-name`. The user authorized change 1 (fab-kit) only: “Go ahead with 1.”

## Why

Fab resolves providers when composing executables but hardcodes slash-prefixed skills in batch new, batch switch, the fallback operator launcher, and operator routing instructions. Codex requires dollar-prefixed explicit skill mentions. Selecting Codex therefore launches the right executable with the wrong skill prompt. One shared renderer prevents Go launchers and markdown-driven routing from drifting.

## What Changes

### Shared rendering and CLI

Add a pure shared renderer receiving provider, bare skill name, and an optional argument string. Only the exact provider name `codex` selects `$`; all other names, including unknown/empty/custom providers, select `/`. Preserve argument content verbatim. Do not infer provider from model names or rewrite arbitrary prompt text. No provider configuration field is needed.

Expose it as a read-only `fab skill-prompt <skill> [arguments] --provider <name>` CLI for operator instructions. Offer shell-quoted output using the existing POSIX quoting utility for embedding as one launch argument, plus JSON output. Validate bare skill names and command arity; accept unregistered provider names without loading configuration.

### Launchers

Use the renderer for `fab batch new`, `fab batch switch`, and the built-in operator launcher. Keep the provider identity with the resolved command, including the actual Claude provider when an unavailable interactive command triggers the existing Claude fallback. Keep positional delivery, worker environment scoping, and interactive shell fallback behavior.

### Operator routing

Update canonical `src/kit/skills/_cli-agents.md` to own skill rendering usage. Fresh spawns use the target repo's resolved provider. Existing-pane sends use the receiving live harness (inspect process evidence as needed), never the operator's provider or current repo defaults. Unknown receivers use the slash default. Update `fab-operator.md` to consume this rule across switch/branch/pipeline commands and existing-change, raw-request, backlog, autopilot, and watch launches. Keep ordinary text, answers, keys, and prompt-file pointers literal.

### Scope

Do not change run-kit, delegated `rk operator` kickoff, `/loop`, heartbeat scheduling, general launch-delivery portability, or provider configuration. Update CLI reference and affected behavior documentation; ship regression tests for all launchers and shell quoting.

## Affected Memory

- `runtime/agent-primitives`: (modify) Shared skill-prompt command, receiver selection, quoting, and spawning/routing usage.
- `runtime/operator`: (modify) Receiver-aware skill routing and fallback launcher prompts.
- `runtime/providers-and-profiles`: (modify) Resolved provider retained for skill rendering in launchers.

## Impact

Go changes touch `internal/agent`, `cmd/fab` launchers, CLI registration and tests. Canonical skill changes touch `_cli-agents.md`, `fab-operator.md`, and `_cli-fab.md`. No new dependencies, configuration keys, migrations, or cross-repo writes.

## Open Questions

None. Prefix policy and fab-kit-only scope were explicitly settled in discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Codex uses `$`; every other provider uses `/`, with no config knob | Explicit user decision | S:100 R:95 A:100 D:100 |
| 2 | Certain | Limit implementation to fab-kit; retain delivery and heartbeat behavior | User authorized change 1 after the two-repo scope was explained | S:100 R:95 A:95 D:100 |
| 3 | Certain | Provide a pure CLI renderer with optional shell quoting and JSON | Shared implementation is needed by both Go launchers and markdown routing; shellquote already exists | S:85 R:95 A:95 D:85 |
| 4 | Certain | Select the receiving live harness for existing panes and resolved provider for fresh launches | Discussed mixed-provider routing; process inspection already exists | S:90 R:90 A:90 D:90 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
