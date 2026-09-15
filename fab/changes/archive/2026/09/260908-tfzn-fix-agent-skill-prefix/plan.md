# Plan: Fix agent skill invocation prefixes

**Change**: 260908-tfzn-fix-agent-skill-prefix
**Intake**: [intake.md](intake.md)

## Requirements

### Runtime: Shared skill rendering

**R1**: A single pure renderer SHALL produce `$<skill>` for provider `codex` and `/<skill>` for all other provider strings, including unknown and empty. An optional argument string MUST retain its bytes and receive one separating space when nonempty. It SHALL NOT rewrite arguments, infer providers, or require config.

- GIVEN Codex and skill `fab-fff` with arguments `ab12`, WHEN rendered, THEN the prompt is `$fab-fff ab12`.
- GIVEN a custom provider and arguments containing `$HOME`, `/fab-new`, and newlines, WHEN rendered, THEN the prefix is `/` and arguments remain literal.

**R2**: `fab skill-prompt <skill> [arguments] [--provider <name>|--repo <path>] [--shell-quote|--json]` SHALL expose R1 without launching or sending. The explicit `--repo` mode SHALL resolve the target repository's default-role provider without stage-dispatch capability checks and SHALL be mutually exclusive with `--provider`; otherwise the query stays config-free. Bare names MUST be validated as ASCII letters, digits, hyphens, and underscores, starting with a letter or underscore. Invalid arity/names or conflicting sinks SHALL exit 2. Default output SHALL be the prompt plus a newline; shell output SHALL quote it as one POSIX shell token; JSON SHALL expose provider, skill, and prompt.

- GIVEN an unregistered provider, WHEN the query runs outside a Fab project, THEN it succeeds using `/`.
- GIVEN Codex and shell metacharacters in the arguments, WHEN shell-quoted output is evaluated as one argument, THEN the receiver gets the exact literal prompt with no substitutions.

### Launchers: Preserve receiving provider

**R3**: Batch new/switch and the fallback operator launcher SHALL retain the provider with their resolved executable and use R1 for initial prompts. Existing fallback to the Claude executable SHALL also use Claude syntax. Worker env prefixes, role fills, positional delivery, and shell fallback SHALL remain unchanged. Delegated `rk operator` SHALL remain untouched.

- GIVEN `agent.session=codex`, WHEN each launcher opens a window, THEN its single prompt starts with `$` and the shell cannot expand it.
- GIVEN another/custom provider, WHEN launched, THEN its prompt starts with `/`.
- GIVEN an unavailable interactive command, WHEN the launcher falls back to Claude, THEN it uses `/` regardless of the configured provider.

### Operator: Shared routing instruction

**R4**: `_cli-agents.md` SHALL own usage of the CLI renderer. Fresh spawns SHALL resolve the target repo's default-role provider through the renderer's `--repo` mode; existing-pane routing SHALL use the receiving live harness from process evidence (unknown uses default), never the sender or mutable repo defaults. `fab-operator.md` SHALL route all explicit skill sends/spawns through this procedure. Args MUST survive both the invoking shell and the new-window shell.

- GIVEN a Claude operator and a Codex target, WHEN sending `fab-switch`, THEN the rendered prompt is `$fab-switch <change>`.
- GIVEN a custom target or unknown identity, WHEN sending a skill, THEN it receives `/` syntax.

**R5**: CLI help/reference and current behavior documentation SHALL describe the renderer and its consumers. Stage pointer prompts, arbitrary text, key delivery, run-kit, and heartbeat/native TUI commands SHALL retain their existing behavior.

- GIVEN a dispatch prompt-file pointer or ordinary answer containing slash text, WHEN delivered, THEN it remains literal and bypasses rendering.

### Non-Goals

Run-kit changes, live pane mutation during development, heartbeat portability, prompt delivery redesign, migrations, and new provider config keys.

### Design Decisions

**Decision**: Use one renderer and a config-free CLI with explicit receiver input.
**Why**: Go callers and markdown orchestration need the same open-ended prefix rule; explicit receiver input prevents sender-config mistakes.
**Rejected**: A provider capability config field, blanket string replacement, or separate Claude/Codex branches in every caller.
*Introduced by*: 260908-tfzn-fix-agent-skill-prefix

## Tasks

### Phase 1: Implementation

- [x] T001 Implement and test shared renderer in `src/go/fab/internal/agent/skill_prompt.go` and read-only CLI in `src/go/fab/cmd/fab/skill_prompt.go`, registered in `main.go`. <!-- R1 R2 -->
- [x] T002 Wire receiving provider and shared rendering into `src/go/fab/cmd/fab/batch.go`, `batch_new.go`, `batch_switch.go`, and `operator.go`; add launcher regressions and quote round-trip tests. <!-- R3 -->
- [x] T003 Update `src/kit/skills/_cli-agents.md`, `fab-operator.md`, `_cli-fab.md` and affected reference prose to consume renderer; verify routing coverage and run affected Go suites. <!-- R4 R5; rework: preserve trailing newlines in existing-pane sends and add glossary CLI cross-reference -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Codex uses `$`; Claude, other built-ins, custom, and unknown providers use `/`; argument bytes are preserved.
- [x] A-002 R2: CLI works without project/config, supports raw/quoted/JSON outputs, validates bare names and arity, and rejects conflicting outputs with exit 2.
- [x] A-003 R3: All three launchers use the shared renderer with the receiving provider, including Claude fallback; role fills/env/shell fallback remain intact.
- [x] A-004 R4: Operator skill routing and every spawn entry use the shared procedure with correct receiver identity and shell quoting.
- [x] A-005 R5: CLI/reference docs match implementation; generic text, pointers, heartbeat, and delegated run-kit behavior remain intact.

### Scenario Coverage

- [x] A-006 R2: Shell round-trip tests preserve dollar prefixes, quotes, backticks, substitutions, whitespace and newlines without execution.
- [x] A-007 R3: Regression tests exercise batch new/switch and fallback operator under Codex and non-Codex configurations.

### Code Quality

- [x] A-008: Code uses existing Cobra/config/quoting patterns with focused functions, readable names, and no unnecessary duplication.
- [x] A-009: Shared rules have one owner; skill consumers point to it and changed behavior has appropriate tests.
- [x] A-010: Relevant tests pass; edits target canonical skill sources; no migration is required because existing user data is unchanged.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fab skill-prompt` as a pure query with one optional argument string | Allows byte preservation, default fallback for arbitrary providers, and direct shell composition | S:85 R:95 A:95 D:85 |
| 2 | Certain | Existing-pane provider selection stays in the established process-inspection workflow | Operator already uses live process evidence; no new detection schema or state is required | S:85 R:95 A:90 D:85 |
| 3 | Certain | Add an explicit `--repo` mode to the renderer for fresh default-role spawns | `fab agent -o yaml` also selects a dispatch adapter and can reject valid interactive-only providers; reuse role resolution without that check | S:85 R:95 A:95 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).

## Deletion Candidates

- `internal/spawn.Command` (`src/go/fab/internal/spawn/spawn.go:32`) — the launcher refactor removes its final production call site; only its dedicated tests remain, while `roleSessionCommand` now owns active interactive-command resolution.
