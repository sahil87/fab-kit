# Plan: `_cli-agents` Helper Extraction + Provider-Addressable Spawn

**Change**: 260805-nvad-cli-agents-helper-provider-spawn
**Intake**: `intake.md`

## Requirements

### Skills: The `_cli-agents` Helper

#### R1: New internal helper `src/kit/skills/_cli-agents.md`
The kit SHALL ship a new internal helper skill at `src/kit/skills/_cli-agents.md` carrying the generic agent-interaction procedures (spawn / pre-send validation / peek / await) and a three-provider operational dictionary (claude, codex, gemini). Its frontmatter MUST match the other `_` helpers: `name: _cli-agents`, a `description:` one-liner, `user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`. It MUST NOT be part of the always-load layer — it is opt-in via a consumer's frontmatter `helpers:` list.

- **GIVEN** a session (or skill) that needs to spawn, prompt, peek at, or await an agent CLI
- **WHEN** it loads `.claude/skills/_cli-agents/SKILL.md`
- **THEN** it finds the spawn-composition, pre-send-validation, delivery-probe, peek, and await procedures plus the per-provider grammar/discovery dictionary, with no operator-orchestration content (repo targeting, worktrees, enrollment, dependency resolution, autopilot)

#### R2: Half A — agent-interaction procedures, moved not rewritten
`_cli-agents.md` SHALL carry four procedural sections, whose mechanics are relocated from `fab-operator.md`:

1. **Spawn composition** — compose an interactive agent session command via `fab agent --print [--repo <path>]` (tier-addressed) or `fab agent --provider <name> --print [--model <id>] [--effort <level>]` (provider-addressed, R7), then open it with `tmux new-window -n <name> -c <dir> "<cmd> '<prompt>'"`.
2. **Pre-send validation + delivery-probe discipline** — the pane-exists → agent-idle (three-state `@rk_agent_state`) gate, plus the printed-prompt recovery probe (literal test send → `C-u` → retype → Enter → confirm via working spinner).
3. **Peek** — `fab pane capture` for raw output and the `@rk_agent_state` read for agent state, with the explicit caveat that the state option is written by run-kit's `rk agent-setup` global agent-harness hooks (covering Claude Code, Codex, Copilot, Gemini, OpenCode — per `_cli-fab.md` § fab pane → agent state), so an **uninstrumented** pane (no `rk agent-setup` run, or a harness its hooks don't cover) reads `—` (unknown) and capture is the universal fallback. <!-- clarified: scope corrected from "claude-hook-based / non-claude panes read unknown" — review cycle 1 found the canonical source contradicts that premise -->.
4. **Await** — a capture+state poll loop until a completion signal, with the honest note that no adapter recovers Agent-tool-style completion notifications; polling or an `rk notify` instructed in the worker's prompt are the available signals.

- **GIVEN** `fab-operator.md` §3 Pre-Send Validation and §6 spawn steps 5–6
- **WHEN** the extraction is applied
- **THEN** the generic mechanics live in `_cli-agents.md` and `fab-operator.md` references them, while operator-specific policy (confirmation tiers, bounded retries, enrollment, dependency resolution, autopilot) stays verbatim in `fab-operator.md`

#### R3: Half B — provider dictionary carries stable grammar + discovery recipes, never catalogs
The dictionary SHALL carry, per provider (claude / codex / gemini): interactive vs headless entry points, stdin/prompt-delivery behavior, structured-output flags, resume/session semantics where they exist, and a **model discovery recipe** (a command/probe against the *installed* CLI). It MUST NOT bake model-ID catalogs. It SHALL also carry the codex MCP-bridge recipe as a short subsection (recipe text only, no fab machinery) and interactive quirks only where already confirmed by real encounters.

- **GIVEN** a session that needs to know which models the installed `codex` accepts
- **WHEN** it reads the codex dictionary entry
- **THEN** it finds a discovery recipe to run against the installed binary, not a hardcoded model list that can rot

#### R4: `_preamble.md` registers `_cli-agents` as an allowed `helpers:` value
`src/kit/skills/_preamble.md` § Skill Helper Declaration **Allowed values** SHALL list `_cli-agents` alongside the existing seven (`_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake`), making the allowed set eight.

- **GIVEN** a skill declaring `helpers: [_cli-agents, …]`
- **WHEN** an agent reads `_preamble.md` § Skill Helper Declaration
- **THEN** `_cli-agents` is a listed allowed value and the count word reads "eight"

#### R5: `fab-operator.md` declares the helper and dedupes the moved mechanics
`fab-operator.md` frontmatter SHALL become `helpers: [_cli-agents, _cli-fab, _cli-external]`. The body sections whose mechanics moved SHALL be reduced to references into `_cli-agents` sections; every operator-specific policy statement SHALL survive verbatim.

- **GIVEN** the operator's §3 Pre-Send Validation and §6 spawn sequence
- **WHEN** the dedupe is applied
- **THEN** the pane-exists/idle-gate mechanics and the spawn-command-composition + window-open steps point at `_cli-agents`, while the confirmation tiers, bounded retries, change-active/branch-alignment checks, repo targeting, pointer activation, dependency resolution, and enrollment remain stated in `fab-operator.md`

### CLI: Provider-Addressable `fab agent`

#### R6: `fab agent --provider <name> [--model <id>] [--effort <level>]` bypasses tier resolution
`fab agent` SHALL accept a `--provider <name>` flag that looks up `providers.<name>` directly via `agent.ResolveProvider` (project config per-field-merged over built-ins) and composes its `session_command` through `spawn.WithProfile` with the `--model`/`--effort` values. When `--model`/`--effort` are omitted the value is empty and the existing `WithProfile` empty-value rule applies (template mode: placeholder token + preceding `-`-flag dropped; append mode: flag omitted).

- **GIVEN** a config with `providers.codex.session_command: 'codex -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --provider codex --print` runs
- **THEN** stdout is `codex` (both placeholder tokens and their preceding flags dropped), so the CLI's own default model applies
- **AND** `fab agent --provider codex --model gpt-5.3-codex --effort high --print` prints `codex -m gpt-5.3-codex -c model_reasoning_effort=high`

#### R7: `--provider` is mutually exclusive with the `[tier]` positional
Supplying both a `[tier]` positional and `--provider` SHALL be a usage error with a non-zero exit; `--print` and `--repo` SHALL compose with `--provider` unchanged.

- **GIVEN** `fab agent doing --provider codex`
- **WHEN** the command runs
- **THEN** it exits non-zero with an error naming the mutual exclusion, and no command is exec'd or printed

#### R8: Unknown provider name is a lookup failure that names the available providers
A `--provider <name>` naming a provider present in neither the project config nor the built-in table SHALL exit non-zero with an error listing the available provider names. This is a lookup failure, not validation of the command's content — the document-don't-validate contract is preserved (resolved command strings still pass through verbatim).

- **GIVEN** a config whose `providers:` block defines only `claude`
- **WHEN** `fab agent --provider bogus --print` runs
- **THEN** it exits non-zero and stderr/the error names `bogus` and lists the available provider names
- **AND GIVEN** a provider that resolves but carries no `session_command`
- **WHEN** the same command runs
- **THEN** the existing `configure providers.<name>.session_command` hint error is returned

#### R9: `--model`/`--effort` are scoped to the provider path
`--model` and `--effort` SHALL apply only in combination with `--provider`. Supplying either without `--provider` SHALL be a usage error with a non-zero exit (the tier path's profile comes from the resolved tier, so a bare `--model` has no coherent semantics).

- **GIVEN** `fab agent --model gpt-5 --print` (no `--provider`)
- **WHEN** the command runs
- **THEN** it exits non-zero with an error stating `--model`/`--effort` require `--provider`

#### R10: Behavior is otherwise identical to the tier path
On the `--provider` path the resolved command SHALL be exec'd via `sh -c` with no TTY guard by default and printed instead when `--print` is given, exactly as the tier path does. No existing path (tier resolution, defaults, `fab resolve-agent`, the operator launcher, `fab batch`) changes behavior.

- **GIVEN** any invocation without `--provider`
- **WHEN** it runs
- **THEN** its output and exec behavior are byte-identical to before this change

### Docs: `_cli-fab.md` + SPEC Mirrors

#### R11: `_cli-fab.md` § fab agent documents the new signature
`src/kit/skills/_cli-fab.md` § fab agent SHALL state the new signature `fab agent [tier] [--provider <name> [--model <id>] [--effort <level>]] [--print] [--repo <path>]` and document the provider-addressed form: direct `providers.<name>` lookup, tier bypass, the empty-model composition rule, mutual exclusion with `[tier]`, the `--model`/`--effort`-require-`--provider` rule, and the unknown-provider error.

- **GIVEN** the constitution's "CLI change ⇒ `_cli-fab.md` + tests" constraint
- **WHEN** the change ships
- **THEN** `_cli-fab.md` § fab agent carries the new flags and Go tests cover each documented behavior

#### R12: SPEC mirror class swept up front
The change SHALL update every member of the mirror class: a new `docs/specs/skills/SPEC-_cli-agents.md`; `docs/specs/skills/SPEC-_preamble.md` (allowed-values list + count + the helper-tree diagram); `docs/specs/skills/SPEC-fab-operator.md` (helpers row, the extraction boundary, the spawn/pre-send references); `docs/specs/skills.md` (allowed-values count, the helper mapping table, the SPEC-exclusion policy wording, the New Skill Checklist helper enumeration); and `docs/specs/stage-models.md` + `docs/specs/glossary.md` **only where they restate an affected fact** (grep-verified).

- **GIVEN** code-quality.md § Sibling & Mirror Sweeps
- **WHEN** apply finishes
- **THEN** a repo-wide grep for the old claims (`Allowed values` count, the `helpers:` mapping rows, `fab agent [tier]`) returns no un-updated occurrence outside frozen historical records

### Non-Goals

- No new built-in Go provider — `defaultProviders` stays claude-only; codex/gemini remain uncomment-to-opt-in config template text and markdown dictionary entries.
- No `docs/memory/` edits — that is hydrate's job.
- No config-schema change, no migration (no user-data restructuring).
- No Phase 2 interactive stage-dispatch adapter (drafted separately).
- No provider validation — fab never infers a provider from a model string, and command content is never validated.

### Design Decisions

#### `--provider` Is a Sibling Addressing Mode, Not a Tier Override
**Decision**: `--provider` bypasses tier resolution entirely (direct `providers.<name>` lookup + `--model`/`--effort` as the profile) rather than synthesizing an ad-hoc tier or overriding a named tier's fields.
**Why**: A tier is a `{provider, model, effort}` role profile owned by fab-kit's fixed mapping; provider-addressed spawning is a different question ("give me a codex session"), and mixing the two has no coherent semantics (a tier already names a provider). Bypass keeps `ResolveTier` untouched, so no existing path changes behavior.
**Rejected**: A `--tier-provider` style override (mutates role/budget policy to express a mechanics question); auto-creating a synthetic tier (invents state the config never declared).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

#### `--model`/`--effort` Require `--provider`
**Decision**: `--model` and `--effort` are usage errors without `--provider`.
**Why**: On the tier path the profile is *the tier's*, resolved through inheritance; a bare `--model` would either silently override the tier (an undocumented tier-override surface) or be silently ignored. An explicit usage error is the only honest option and is trivially relaxable later.
**Rejected**: Letting `--model` override a resolved tier's model (adds a second, undocumented tier-override surface); silently ignoring the flags (violates surprise-free CLI behavior).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

#### `_cli-agents` Gets a SPEC Despite the `_cli-*` Naming
**Decision**: Ship `docs/specs/skills/SPEC-_cli-agents.md` and narrow `skills.md`'s SPEC-exclusion policy to name the two pure-reference partials (`_cli-fab.md`, `_cli-external.md`) explicitly rather than the `_cli-*` prefix.
**Why**: The exclusion policy's basis is *behavioral*: a partial that only mirrors an external command surface would make its SPEC a third copy of the same tables. `_cli-agents.md` defines *procedures* (spawn/validate/peek/await discipline) — behavior — so it falls on the "every other behavioral partial gets a SPEC" side. The intake asks for the SPEC on the same reasoning.
**Rejected**: Treating the `_cli-` prefix as the exclusion trigger (would exempt a behavioral helper on a naming coincidence).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

## Tasks

### Phase 1: Setup

- [x] T001 Grep-verify the mirror-sweep class up front: enumerate every live restatement of (a) the `helpers:` allowed-values list/count, (b) the `fab-operator` `helpers:` mapping row, (c) the `fab agent [tier] [--print] [--repo <path>]` signature, across `src/kit/skills/`, `docs/specs/`, and `src/go/fab/`; record the hit list before editing <!-- R12 -->

### Phase 2: Core Implementation

- [x] T002 Create `src/kit/skills/_cli-agents.md` — frontmatter (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`), `## Contents`, Half A procedures (Spawn composition / Pre-Send Validation + delivery probe / Peek / Await) with the scope-boundary note <!-- R1 R2 --> <!-- rework: § Peek state-writer caveat must match corrected R2 — `rk agent-setup` hooks cover Claude Code/Codex/Copilot/Gemini/OpenCode (`_cli-fab.md:423`); the caveat scope is UNINSTRUMENTED panes, not non-claude. Fix heading + body at _cli-agents.md:103,139 -->
- [x] T003 Add Half B to `src/kit/skills/_cli-agents.md` — the provider dictionary (claude / codex / gemini: entry points, stdin & structured output, resume semantics, model discovery recipe), the codex MCP-bridge recipe subsection, and the confirmed-quirks-only rule <!-- R3 --> <!-- rework: the MCP-bridge recipe names the wrong subcommand — `codex mcp` manages external MCP servers; the stdio server is `codex mcp-server` (verified against installed binary). Fix ALL occurrences: _cli-agents.md:149 (table row), :172 (capability probe), :173 -->
- [x] T004 Register `_cli-agents` in `src/kit/skills/_preamble.md` § Skill Helper Declaration Allowed values <!-- R4 -->
- [x] T005 Add `--provider`/`--model`/`--effort` to `src/go/fab/cmd/fab/agent.go`: flag registration, mutual-exclusion + flag-scoping usage errors, direct `agent.ResolveProvider` lookup with an available-providers error, composition via `spawn.WithProfile`, `--print`/exec unchanged <!-- R6 R7 R8 R9 R10 --> <!-- rework: guards at agent.go:74-84 test string emptiness, not cmd.Flags().Changed() — `fab agent doing --provider= --print` and `fab agent --model= --print` bypass both usage errors; switch to Changed()-based detection -->
- [x] T006 Expose the available-provider-names list from `internal/agent` (a `ProviderNames(cfg)`-style helper merging built-in + project provider keys, sorted) for the unknown-provider error <!-- R8 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Dedupe `src/kit/skills/fab-operator.md`: frontmatter `helpers: [_cli-agents, _cli-fab, _cli-external]`, §2 Context Loading helper sentence, §3 Pre-Send Validation items 1–2 reduced to `_cli-agents` references (operator policy retained), §6 spawn steps 5–6 reduced to `_cli-agents` references (repo-targeting retained), §5 capture/re-capture references <!-- R5 --> <!-- rework: fab-operator.md:339 calls it the "claude-only state-writer caveat" — rename to the uninstrumented-pane scope per corrected R2 -->
- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab agent with the new signature and the provider-addressed form's semantics and errors <!-- R11 -->
- [x] T009 [P] Add Go tests to `src/go/fab/cmd/fab/agent_test.go`: usage error on tier+provider, usage error on `--model`/`--effort` without `--provider`, unknown provider names the available set, empty-model composition (`--provider codex --print` → bare `codex`), explicit `--model`/`--effort` substitution, provider path composes with `--repo`, no-`session_command` hint error, and the tier path unchanged <!-- R6 R7 R8 R9 R10 --> <!-- rework: add explicitly-empty-flag cases pinning the Changed() guards: tier positional + `--provider=` errors; `--model=` without --provider errors -->
- [x] T010 [P] Create `docs/specs/skills/SPEC-_cli-agents.md` (Summary + Section Structure + Primitives + Key Properties + Resolved Design Decisions) <!-- R12 --> <!-- rework: mirror both corrections — `codex mcp-server` at SPEC-_cli-agents.md:52,54,101 and the uninstrumented-pane state scope at :38 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-_preamble.md` — allowed-values list, the count word, and the helper-tree diagram <!-- R12 R4 -->
- [x] T012 [P] Update `docs/specs/skills/SPEC-fab-operator.md` — Helpers line, §2/§3/§6 section-structure entries reflecting the extraction, and the Primitives table row for `_cli-agents` <!-- R12 R5 --> <!-- rework: SPEC-fab-operator.md:131 restates the claude-only caveat — correct to the uninstrumented-pane scope (note :100's pre-existing multi-harness wording as the consistent baseline) -->
- [x] T013 [P] Update `docs/specs/skills.md` — § Skill Helpers allowed-values count + list, the mapping table's `fab-operator` row, the New Skill Checklist item-3 helper enumeration, the SPEC-exclusion-policy wording, and a `_cli-agents` entry where partial SPECs are enumerated <!-- R12 -->
- [x] T014 [P] Update `docs/specs/stage-models.md` and `docs/specs/glossary.md` only where they restate the `fab agent` signature or provider-addressing facts (grep-verified from T001) <!-- R12 R11 -->
- [x] T017 Fix the `session_command` reference-fence text in `src/go/fab/internal/configref/configref.go:455`: `{model}`/`{effort}` substitution now has two sources — the resolved tier profile (tier path) or the `--model`/`--effort` flags (`fab agent --provider` path); update the literal and add/extend a configref test pinning the corrected phrase <!-- R11 -->
- [x] T018 Dedupe `src/kit/skills/_cli-external.md` § tmux Usage Notes: reduce the agent-spawn `new-window` command form (line ~230) and the `fab agent --print --repo` restatement (line ~163) to references into `_cli-agents.md` § Spawn Composition; sweep `SPEC-_cli-external.md` if its section structure changes <!-- R2 R5 -->

### Phase 4: Polish

- [x] T015 Run the affected Go package tests (`./cmd/fab/...`, `./internal/agent/...`, `./internal/spawn/...`), then the full `go test ./...` for the `fab` module; fix failures <!-- R6 R7 R8 R9 R10 -->
- [x] T016 Re-run the T001 greps as a self-verification sweep; confirm no un-updated live restatement of any changed claim remains <!-- R12 -->

## Execution Order

- T001 precedes every doc edit (it defines the sweep class)
- T006 precedes T005's error path (the helper it calls)
- T005 precedes T009 (tests target the implemented flags)
- T002/T003 precede T007 (the operator references sections that must exist) and T010
- T004 precedes T011/T013
- T015 after T005/T006/T009; T016 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_cli-agents.md` exists with internal-helper frontmatter (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`) and is not referenced by the always-load layer
- [x] A-002 R2: All four Half-A procedures (spawn composition, pre-send validation + delivery probe, peek, await) are present, and the scope boundary excluding operator orchestration is stated <!-- resolved (cycle 2 re-review): § Peek's caveat at _cli-agents.md:103 now reads "uninstrumented panes read unknown" and names the full Claude Code/Codex/Copilot/Gemini/OpenCode hook coverage, matching the canonical _cli-fab.md:423; the same scope is mirrored in SPEC-_cli-agents.md:38, SPEC-fab-operator.md:90, and fab-operator.md:339 -->
- [x] A-003 R3: Each of claude/codex/gemini has an entry with entry points, prompt-delivery behavior, and a model **discovery recipe**; no baked model-ID catalog appears; the codex MCP-bridge recipe is present <!-- resolved (cycle 2 re-review): every occurrence now names `codex mcp-server` and explicitly distinguishes it from `codex mcp` (_cli-agents.md:149,172,173; SPEC-_cli-agents.md:52,54,101). Re-verified against the installed binary: `codex --help` lists `mcp` = "Manage external MCP servers for Codex" and `mcp-server` = "Start Codex as an MCP server (stdio)". -->
- [x] A-004 R4: `_preamble.md` § Skill Helper Declaration lists `_cli-agents` among the allowed values
- [x] A-005 R5: `fab-operator.md` declares `helpers: [_cli-agents, _cli-fab, _cli-external]` and its moved-mechanics sections reference `_cli-agents`
- [x] A-006 R6: `fab agent --provider <name>` resolves `providers.<name>` directly and composes its `session_command` via `spawn.WithProfile`
- [x] A-007 R11: `_cli-fab.md` § fab agent states the new signature and the provider-addressed semantics
- [x] A-008 R12: `SPEC-_cli-agents.md` exists and `SPEC-_preamble.md`, `SPEC-fab-operator.md`, `skills.md` are updated

### Behavioral Correctness

- [x] A-009 R6: With an omitted `--model`, a templated `session_command` drops the placeholder token and its preceding `-`-flag (`fab agent --provider codex --print` → `codex`), so the CLI's own default model applies
- [x] A-010 R7: `fab agent <tier> --provider <name>` exits non-zero as a usage error and neither prints nor execs a command <!-- review: holds for non-empty values; `--provider=` (explicit empty) falls through to the tier path and prints — see should-fix finding on Flag.Changed --> <!-- resolved: guards switched to cmd.Flags().Changed() (T005) with the explicitly-empty cases pinned by TestAgentEmptyProviderStillMutuallyExclusive / TestAgentEmptyProviderAloneIsLookupFailure / TestAgentEmptyModelEffortStillRequireProvider (T009) -->
- [x] A-011 R9: `--model`/`--effort` without `--provider` exits non-zero as a usage error
- [x] A-012 R8: An unknown `--provider` name exits non-zero and the error lists the available provider names
- [x] A-013 R10: Every invocation without `--provider` behaves byte-identically to before the change (existing `agent_test.go` cases still pass unmodified)

### Scenario Coverage

- [x] A-014 R6: A Go test pins `fab agent --provider codex --model … --effort … --print` output against a templated `session_command`
- [x] A-015 R7: A Go test asserts the tier+provider usage error
- [x] A-016 R8: A Go test asserts the unknown-provider error names the available providers
- [x] A-017 R10: A Go test asserts `--provider` composes with `--repo` (reads the target repo's `providers:` table)

### Edge Cases & Error Handling

- [x] A-018 R8: A `--provider` naming a provider that resolves but carries no `session_command` yields the existing `configure providers.<name>.session_command` hint error
- [x] A-019 R6: Both `--model` and `--effort` omitted against a non-templated `session_command` appends neither flag (append-mode empty-value rule)

### Code Quality

- [x] A-020 Pattern consistency: New Go code follows `cmd/fab` conventions (cobra `RunE`, `fmt.Errorf` wrapping, doc comments explaining the contract) and reuses `agent.ResolveProvider` + `spawn.WithProfile` rather than reimplementing lookup or substitution
- [x] A-021 No unnecessary duplication: The extraction MOVES operator mechanics into `_cli-agents.md` rather than copying them — no procedure is stated in both files <!-- resolved (cycle 2 re-review): _cli-external.md:163 and :234 are now pointers into `_cli-agents.md` § Spawn Composition (T018), and fab-operator.md carries no residual probe/capture/spawn mechanics (grep-verified for XYZTEST/C-u/spinner/printed-output/new-window). The one remaining literal — fab-operator.md:471's `tmux new-window -n "»<wt>" …` — is framed as the operator's window-marker policy over the referenced mechanics; noted in ## Deletion Candidates, not a duplication defect. -->
- [x] A-027 Documentation accuracy (user-facing string literals): the `configref.go` `session_command` fence text names BOTH substitution sources <!-- resolved (cycle 2 re-review): configref.go:452-459 now reads "substituted from the resolved tier profile, or from the --model/--effort flags on `fab agent --provider <name>` (which bypasses tier resolution)", pinned by TestConfigReferenceDocumentsBothSubstitutionSources (which also asserts the superseded tier-only phrasing is gone). -->
- [x] A-022 Canonical source only: every skill edit lands in `src/kit/skills/`; nothing under `.claude/skills/` is modified
- [x] A-023 SPEC-mirror sync: every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in the same change (except the policy-excluded pure-reference partials `_cli-fab.md`/`_cli-external.md`)
- [x] A-024 CLI ⇒ docs + tests: the `fab agent` signature change ships `_cli-fab.md` updates AND Go test updates in the same change
- [x] A-025 No migration needed: the change restructures no user data (no config-schema change, no `.status.yaml` shape change), so no `src/kit/migrations/` file is required
- [x] A-026 Tests green: the affected Go packages' tests pass (`./cmd/fab/...`, `./internal/agent/...`, `./internal/spawn/...`) and no test was bent to fit the implementation

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/kit/skills/fab-operator.md:471` (§6 spawn step 6) — still spells out the full `tmux new-window -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"` form immediately after pointing at `_cli-agents.md` § Spawn Composition for that same form; only the `»<wt>` marker name is operator policy, so the surrounding command skeleton is a candidate for reduction to `… with the window name "»<wt>"`. Deliberately left as-is this change (the literal is a useful single-glance reference at the one site that composes it), but it is the last remaining two-homed spawn-command string
- No Go symbol became redundant — `ResolveTier`, `ResolveProvider`, and `spawn.WithProfile` are all still called on both paths, and `config.ProviderNames()` is the single new accessor with exactly one (live) call site in `agent.ProviderNames`
- The two `_cli-external.md` candidates raised in review cycle 1 (§ tmux Usage Notes `new-window` bullet, § wt repo-targeted-spawning note) were **acted on** in that cycle's T018 — both are now pointers into `_cli-agents.md`, so they are no longer candidates

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `--model`/`--effort` without `--provider` is a usage error (not a silent tier override, not silently ignored) | Not specified in the intake; a bare `--model` on the tier path would either invent an undocumented tier-override surface or be silently ignored. An explicit usage error is the honest, trivially-relaxable default | S:55 R:85 A:85 D:75 |
| 2 | Confident | Ship `SPEC-_cli-agents.md` even though `skills.md`'s exclusion policy exempts the pure-reference `_cli-*` partials; narrow the policy wording to name `_cli-fab.md`/`_cli-external.md` explicitly | The exclusion basis is behavioral (a mirror of an external command surface), and `_cli-agents.md` defines procedures. The intake explicitly lists the SPEC as in-scope | S:80 R:85 A:80 D:75 |
| 3 | Confident | The unknown-provider error lists the available provider names from the merged built-in + project provider key set, sorted | Intake says "available provider names on stderr" without naming the source; merging built-ins with the project table is the only set that matches what `ResolveProvider` will accept | S:70 R:90 A:90 D:85 |
| 4 | Confident | Extraction boundary: only the four generic procedures move; the operator's confirmation tiers, change-active/branch-alignment pre-send items, repo targeting, pointer activation, dependency resolution, enrollment, and autopilot stay verbatim in `fab-operator.md` | Intake assumption 4 (Confident) — followed as stated; the boundary is "agent primitives vs operator orchestration" | S:75 R:70 A:80 D:75 |
| 5 | Confident | The moved mechanics are replaced in `fab-operator.md` by references, not duplicated — the operator's §3 items 1–2 and §6 steps 5–6 keep their operator-specific wrapper text and point at `_cli-agents` sections for the mechanics | Follows the established `_cli-external.md` slim precedent (fab-owned choreography stays, tool-owned contract delegated); avoids the duplicate-truth failure the sweep rules exist to prevent | S:70 R:80 A:85 D:80 |
| 6 | Confident | Provider-dictionary quirks are seeded only from facts already documented in this repo (the printed-prompt/send-keys trap, the `rk agent-setup` state writer's instrumented-harness scope per `_cli-fab.md`, the shipped codex/gemini starter-template strings, the gemini no-`-p`/no-`{effort}` facts) — nothing is invented about an uninstalled CLI | Intake R3/assumption 2 forbids volatile catalogs and says quirks accrete from real encounters; asserting unverified CLI behavior would violate document-don't-guess | S:80 R:75 A:70 D:80 |
| 7 | Tentative | `--provider` and the `[tier]` positional are checked for mutual exclusion in `RunE` (a hand-written check) rather than via cobra's `MarkFlagsMutuallyExclusive` | cobra's helper only relates *flags*; the tier is a positional, so no built-in mechanism covers this pairing <!-- assumed: hand-written mutual-exclusion check in RunE, since cobra's MarkFlagsMutuallyExclusive cannot relate a positional to a flag --> | S:60 R:90 A:85 D:60 |

7 assumptions (0 certain, 6 confident, 1 tentative).
