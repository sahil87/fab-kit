# Plan: Codex/Gemini Provider Bypass Flags

**Change**: 260808-clxw-codex-gemini-bypass-flags
**Intake**: `intake.md`

## Requirements

### Agent Providers: Autonomous Built-in Command Grammars

#### R1: Codex Commands Bypass Approvals and Sandboxing

The built-in codex `session_command` and `dispatch_command` MUST include Codex's installed-CLI-supported `--dangerously-bypass-approvals-and-sandbox` flag while preserving their `{model}` and `{effort}` placeholders and stdin-based `codex exec` dispatch behavior.

- **GIVEN** a role resolving to the built-in codex provider
- **WHEN** fab composes either its interactive session command or its headless dispatch command
- **THEN** the command includes `--dangerously-bypass-approvals-and-sandbox`
- **AND** the session form remains a `codex` invocation while the dispatch form remains a `codex exec` invocation
- **AND** model and reasoning-effort profile substitution remains intact

#### R2: Gemini Commands Auto-Approve All Actions

The built-in gemini `session_command` and `dispatch_command` MUST include the current non-deprecated Gemini CLI approval grammar `--approval-mode=yolo`, MUST preserve `{model}` substitution, and MUST continue omitting `{effort}` and `-p`.

- **GIVEN** a role resolving to the built-in gemini provider
- **WHEN** fab composes either its interactive session command or its stdin-driven dispatch command
- **THEN** the command includes `--approval-mode=yolo`
- **AND** both forms retain `-m {model}` without an effort placeholder or prompt-text flag

### Configuration Reference: Explain the Autonomous Policy

#### R3: Rendered Provider Reference Is Accurate and Regression-Guarded

The rendered config reference MUST derive the updated command strings from `internal/agent`, MUST explain that codex and gemini ship approval-bypass grammar because unattended workers cannot answer prompts, and tests MUST pin the bypass policy on both command forms for both providers.

- **GIVEN** the embedded defaults and config reference renderer
- **WHEN** the relevant Go tests parse, resolve, and render the built-in providers
- **THEN** all four non-claude command forms contain their provider's full-auto flag
- **AND** the rendered prose accurately explains the security and automation rationale
- **AND** gemini's existing no-`{effort}` and no-`-p` invariants still pass

### Documentation: Canonical and Mirrored Claims Stay Synchronized

#### R4: Specs and CLI Reference Use the Shipped Grammars

Every current command-string restatement in `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `src/kit/skills/_cli-fab.md`, and `src/kit/skills/_cli-agents.md` MUST show the shipped bypass flags, and both corresponding files under `docs/specs/skills/` MUST mirror their canonical skill's updated provider-policy contract. Historical change artifacts, migrations, arbitrary user-config test fixtures, deployed `.claude/skills/`, and `docs/memory/` content MUST remain unchanged during apply.

- **GIVEN** the repository's canonical docs, skill source, and SPEC mirror
- **WHEN** the documented codex/gemini provider examples are read
- **THEN** active documentation matches the command grammars embedded in `defaults.yaml`
- **AND** a repository-wide old-grammar sweep leaves only intentionally historical or arbitrary-user-config occurrences
- **AND** no deployed skill copy or affected memory content file is edited during apply

### Security: Deliberate Full-Auto Posture

#### R5: Non-Claude Built-ins Match Fab's Unattended Worker Contract

Codex and gemini built-in commands MUST deliberately run with approvals bypassed by default, matching claude's existing full-auto posture, while project and system provider overrides MUST remain the supported escape hatch for users who require approval-gated commands.

- **GIVEN** a pipeline stage dispatched through a built-in codex or gemini provider
- **WHEN** the stage needs shell, network, or repository operations
- **THEN** it can proceed without an interactive approval response
- **AND** no new config schema, migration, or fab CLI command signature is introduced

### Non-Goals

- Changing claude's existing command grammar.
- Installing a Gemini CLI or changing user-level provider overrides.
- Editing deployed `.claude/skills/` copies or hydrating `docs/memory/` during apply.
- Rewriting historical change artifacts, migrations, or tests whose command strings intentionally model arbitrary user configuration.

### Design Decisions

#### Full-Auto Flags Live in Both Command Forms

**Decision**: Ship the codex and gemini bypass grammar in both each provider's session and dispatch commands.
**Why**: Both interactive-pane workers and headless workers owe unattended stage completion, and fab's pipeline has no approval-answering channel.
**Rejected**: Updating dispatch commands only, which would leave pane-mode workers approval-gated; per-project stopgaps, which would preserve broken defaults for every other project.
*Introduced by*: 260808-clxw-codex-gemini-bypass-flags

## Tasks

### Phase 1: Setup

- [x] T001 Verify codex and gemini bypass grammar using installed CLI help where available, with any authoritative fallback recorded in this plan's `## Assumptions`. <!-- R1, R2 -->

### Phase 2: Core Implementation

- [x] T002 Update both codex and both gemini commands plus adjacent rationale comments in `src/go/fab/internal/agent/defaults.yaml`, and add pinned bypass-policy assertions in `src/go/fab/internal/agent/agent_test.go`. <!-- R1, R2, R3, R5 -->
- [x] T003 Update the per-provider explanatory prose in `src/go/fab/internal/configref/configref.go` and the relevant rendered-reference expectations in `src/go/fab/cmd/fab/config_test.go`. <!-- R3, R5 -->

### Phase 3: Integration & Edge Cases

- [x] T004 [P] Synchronize active provider examples and rationale in `docs/specs/stage-models.md` and `docs/specs/architecture.md`, and verify `docs/specs/config.md` has no stale command literal requiring an edit. <!-- R4, R5 -->
- [x] T005 [P] Synchronize `src/kit/skills/_cli-fab.md` and `src/kit/skills/_cli-agents.md` plus both full SPEC mirrors under `docs/specs/skills/` with the bypassed codex/gemini command policy. <!-- R4, R5 -->
- [x] T006 Run `go test ./internal/agent/... ./internal/configref/... ./internal/config/... ./cmd/fab/...` from `src/go/fab`, fix failures, and complete the required repo-wide old-grammar grep classification without editing historical/user-fixture occurrences. <!-- R1, R2, R3, R4, R5 -->

## Execution Order

- T001 establishes the exact grammar used by T002-T005.
- T002 establishes the canonical embedded defaults and blocks T003 and T006.
- T004 and T005 are independent documentation mirrors after T001/T002.
- T006 runs after all implementation and documentation tasks.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Both shipped codex commands include `--dangerously-bypass-approvals-and-sandbox` and preserve profile substitution.
- [x] A-002 R2: Both shipped gemini commands include `--approval-mode=yolo`, preserve model substitution, and omit `{effort}` and `-p`.
- [x] A-003 R3: Config reference output derives and explains all updated commands, with tests pinning the four-command bypass policy.
- [x] A-004 R4: Active specs, canonical `_cli-fab.md`, and `SPEC-_cli-fab.md` match the embedded defaults.
- [x] A-005 R5: Codex/gemini unattended stages use full-auto defaults without a schema, migration, or command-signature change.

### Behavioral Correctness

- [x] A-006 R1: Installed `codex --help` and `codex exec --help` confirm the selected bypass flag on both invocation forms.
- [x] A-007 R2: Gemini grammar is supported by the available authoritative CLI source, and any lack of installed-binary verification is explicitly recorded rather than concealed.

### Scenario Coverage

- [x] A-008 R3: Agent/default/config reference tests cover parsing, resolution, rendering, bypass presence, and gemini's no-effort/no-prompt-text constraints.
- [x] A-009 R4: The required repo-wide grep identifies every old command-string occurrence and confirms each remaining match is intentionally outside the current-truth mirror set.

### Edge Cases & Error Handling

- [x] A-010 R1: Codex session and `exec` flag placement remains accepted and cannot be removed by empty placeholder token-drop behavior.
- [x] A-011 R2: Gemini uses the non-deprecated approval-mode spelling and does not accidentally introduce `-p` or reasoning-effort grammar.

### Code Quality

- [x] A-012 Pattern consistency: Changes follow existing embedded-default, config-reference, and table-test patterns.
- [x] A-013 No unnecessary duplication: Rendered reference command strings continue to derive from exported `internal/agent` defaults.
- [x] A-014 Readability and maintainability: Comments explain the unattended-worker rationale without obscuring the invocation grammar.
- [x] A-015 Existing patterns: No new helper or abstraction is introduced for four declarative command-string checks.
- [x] A-016 Mirror discipline: The canonical skill and complete SPEC mirror are updated together, and deployed `.claude/skills/` copies remain untouched.

### Security

- [x] A-017 R5: Documentation explicitly identifies the bypass as deliberate full-auto behavior and retains provider overrides as the approval-gated escape hatch.

## Notes

- Check acceptance items during review, not apply.
- Historical change artifacts, migrations, and arbitrary user-command fixtures may legitimately retain old grammar and are classified rather than rewritten.

## Deletion Candidates

- None — this declarative default-policy update does not make existing code, files, branches, or configuration redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `--approval-mode=yolo` for both gemini commands instead of the intake's likely `--yolo` spelling | `gemini --help` could not run because Gemini CLI is not installed on this worker; the current official Gemini CLI reference explicitly marks `--yolo` deprecated and directs callers to `--approval-mode=yolo`. The change is a command-template edit and remains easily reversible if a pinned project requires an older CLI. | S:85 R:90 A:90 D:90 |
| 2 | Certain | Pin bypass presence in the existing `TestResolveProvider_BuiltInCodexAndGemini` table test rather than creating a new abstraction | The neighboring test already owns both built-in command pairs and gemini grammar guards, making direct provider-specific assertions the smallest regression guard consistent with surrounding patterns. | S:90 R:95 A:95 D:95 |

2 assumptions (2 certain, 0 confident, 0 tentative).
