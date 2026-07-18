# Plan: Agent Config v3 — Providers & Role Tiers

**Change**: 260702-tykw-agent-providers-role-tiers
**Intake**: `intake.md`

## Requirements

### Providers: config schema

#### R1: Top-level `providers:` section with two command fields
The config schema SHALL gain a top-level `providers:` map, keyed by opaque
user-chosen provider names. Each provider MAY carry `session_command` (opens an
interactive session — the relocated `agent.spawn_command` semantics) and/or
`dispatch_command` (runs one headless stage task — the relocated per-tier
`spawn_command` semantics). The two fields SHALL NOT be merged. fab-kit's
built-in provider table SHALL contain `claude` with the default session command
and no `dispatch_command` (native dispatch). A project's `providers:` block
per-field merges over the built-in.

- **GIVEN** a config with `providers.codex.dispatch_command`
- **WHEN** a stage whose tier points at `codex` is dispatched
- **THEN** `fab dispatch` runs that command; absent `dispatch_command` ⇒ native Agent-tool dispatch (no fallback to `session_command`)

#### R2: `session_command` / `dispatch_command` placeholder substitution
Both command fields SHALL reuse `internal/spawn`'s `{model}`/`{effort}`
substitution (append-mode for a plain Claude command, template-mode when a
placeholder is present).

- **GIVEN** `providers.codex.dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** resolved for a tier with model/effort set
- **THEN** placeholders are substituted; an empty value drops the token + preceding `-`-flag

### Tiers: five role vocabulary

#### R3: Five role tiers replacing thinking/doing/fast
`agent.tiers` keys SHALL become `default`, `operator`, `doing`, `review`, `fast`.
Tier values SHALL be `{provider, model, effort}` (no command field). The built-in
default profiles SHALL be: `default` = claude/claude-fable-5/xhigh; `operator` =
claude/claude-sonnet-5/medium; `doing` = claude/claude-opus-4-8/xhigh; `review` =
claude/claude-fable-5/xhigh; `fast` = claude/claude-sonnet-5/low. `thinking` is
removed entirely.

- **GIVEN** the built-in tier table
- **WHEN** `fab resolve-agent apply` runs (apply ∈ doing)
- **THEN** it resolves to claude-opus-4-8 / xhigh

#### R4: Fixed stage→tier mapping updated
The fab-owned, non-overridable stage→tier mapping SHALL be: intake→default
(advisory), apply→doing, review→review, hydrate→doing, ship→fast,
review-pr→doing.

- **GIVEN** the fixed mapping
- **WHEN** `fab resolve-agent review` runs
- **THEN** it resolves the `review` tier (claude-fable-5/xhigh), distinct from `doing`

#### R5: Per-field inheritance from `default`
Any tier field left unset (provider, model, effort) SHALL inherit from the
project's `default` tier, then from fab-kit's built-in for that tier. Provider is
written explicitly on every tier line as documented style; inheritance is the
safety net.

- **GIVEN** a project `agent.tiers.doing: { effort: high }` (no provider/model)
- **WHEN** resolved
- **THEN** provider+model inherit from the project's `default` tier (or its built-in), effort=high wins

### Review tools: config → prose

#### R6: Retire `review_tools`; move toggles to `code-review.md § Review Tools`
The `review_tools` config key SHALL be removed from the schema, `configref.go`,
`_cli-fab.md`, and all docs. Its two live semantics move to a
`fab/project/code-review.md § Review Tools` section (prose): the outward-reviewer
Codex→Claude cascade toggles and the Copilot request toggle. Absent
section/file/key = all enabled. `_review.md` and `git-pr-review.md` re-point their
config reads at the new prose section.

- **GIVEN** `code-review.md § Review Tools` with `codex: false`
- **WHEN** the outward reviewer runs
- **THEN** it skips Codex; absent section ⇒ all enabled (unchanged default)

### fab agent command

#### R7: New `fab agent [tier] [--print] [--repo <path>]`, retire `fab spawn-command`
A new `fab agent` command SHALL resolve a tier profile (`default` when omitted),
compose `providers.<provider>.session_command` with `{model}`/`{effort}`
substituted (or Claude-style flags appended), and exec it in the current shell.
`--print` prints the fully-resolved (profile-substituted) command instead of
executing. `--repo <path>` reads the target repo's config. `fab spawn-command`
SHALL be removed in the same release (no deprecation alias).

- **GIVEN** the default tier resolving to a claude session command
- **WHEN** `fab agent --print` runs
- **THEN** it prints the command with model/effort resolved (not placeholder-stripped)

### Resolver and dispatch plumbing

#### R8: `fab resolve-agent` renames `spawn=` → `dispatch=`; gains tier-name acceptance
`fab resolve-agent` SHALL keep its stage-name contract for the dispatch seam and
additionally accept the five tier names (for `fab agent` / operator launcher
tier-level resolution). The optional third output line SHALL be renamed from
`spawn=` to `dispatch=` (matching the field rename). Output MAY gain a `provider=`
line. The `dispatch=` line ALWAYS embeds the FULL model ID even under `--alias`.

- **GIVEN** a tier resolving to a provider with a `dispatch_command`
- **WHEN** `fab resolve-agent <stage>` runs
- **THEN** it emits `dispatch=<command>` (not `spawn=`), full-ID model

#### R9: `fab dispatch start` resolves provider `dispatch_command`
`fab dispatch start` SHALL resolve the stage's tier → provider →
`dispatch_command` (error when absent, no fallback — message updated for the new
key path).

- **GIVEN** a stage whose tier's provider has no `dispatch_command`
- **WHEN** `fab dispatch start` runs
- **THEN** it errors pointing at `providers.<name>.dispatch_command`

#### R10: Operator launcher resolves the `operator` tier
The operator launcher SHALL resolve the `operator` tier (not borrow `doing` via
`fab resolve-agent apply`) via the internal resolution, keeping its distinct
tmux/prompt responsibilities. `fab batch new`/`switch` and worker spawns SHALL
compose from `providers.<default.provider>.session_command` + the default tier
profile (workers spawn WITH a profile; the placeholder-stripping print path
disappears with `fab spawn-command`).

- **GIVEN** the operator launcher
- **WHEN** it composes its session command
- **THEN** it uses the operator tier profile (claude-sonnet-5/medium)

#### R11: configref scaffold rewrite
`configref.go` SHALL render a `providers:` block (claude live, codex commented),
the five-tier `agent.tiers` with explicit providers, and SHALL remove
`review_tools` and `agent.spawn_command`.

- **GIVEN** `fab config reference`
- **WHEN** rendered
- **THEN** it shows `providers:` + five tiers, no `review_tools`, no `agent.spawn_command`

### Migrations

#### R12: Migration restructuring user configs + fab-kit's own config
A migration file SHALL: move `agent.spawn_command` → `providers.claude.session_command`
(verbatim value; provider `claude` assumed); map `agent.tiers.{thinking,doing,fast}`
→ the five-tier shape (`doing`/`fast` field-by-field; `thinking` → `review`;
per-tier `spawn_command` → `providers.<name>.dispatch_command` with the tier
pointing at the provider); remove `review_tools` (seed `code-review.md § Review
Tools` only when a key was explicitly `false`). fab-kit's own `config.yaml` SHALL
be updated to the target shape.

- **GIVEN** an existing config with `agent.spawn_command` + `review_tools: {all true}`
- **WHEN** the migration runs
- **THEN** the spawn command relocates to `providers.claude.session_command`, `review_tools` is deleted (no `code-review.md` seed for all-true), idempotent on re-run

### Design Decisions

1. **`fab resolve-agent` tier acceptance is positional, disambiguated by name-set** (resolves Assumption #14 / intake Open Question 1): stage names and tier names are disjoint sets, so `resolve-agent <name>` accepts either — a stage resolves via the fixed mapping, a tier name resolves directly. — *Why*: no new flag surface; the disjoint sets make positional unambiguous. — *Rejected*: a `--tier` flag (adds surface for no disambiguation benefit).
2. **Output gains a `provider=` line** — *Why*: `fab agent`/operator need the provider to look up the session command, and surfacing it aids compliance visibility. — *Rejected*: inferring provider downstream (re-does resolution).
3. **`--alias` on a native-dispatch (no-`dispatch_command`) tier** aliases the `model=` line only; there is no `dispatch=` line to worry about (resolves Open Question 1's `--alias` sub-question) — the misconfig footgun (non-claude provider + native dispatch) is documented, not validated.
4. **Migration halts-and-asks on a non-claude `agent.spawn_command` template** (resolves Open Question 2): a templated (non-claude) existing spawn command cannot be auto-named, so the migration relocates it under a placeholder provider name and instructs the user to rename. — *Why*: auto-creating a `codex` provider guesses the grammar; safer to surface. — *Rejected*: silently naming it `codex`.
5. **`fab agent` exec does not TTY-guard** (resolves Open Question 3): exec-and-let-the-CLI-fail is acceptable and simpler; the underlying agent CLI already handles no-TTY. — *Why*: matches the no-validation/document-don't-guard contract.

### Non-Goals

- `checklist.extra_categories` → `code-quality.md` prose (separate change per intake).
- Any change to the six-stage pipeline or SRAD/scoring.
- Shipped/tested non-claude harness integration (architecture-neutrality only, per stage-models.md).

## Tasks

### Phase 1: Go core — schema, resolver, provider lookup

- [x] T001 Add `Providers map[string]ProviderConfig` (with `SessionCommand`/`DispatchCommand` yaml fields) to `internal/config/config.go`; widen `TierProfile` with `Provider` and remove its `SpawnCommand` field; add `GetProvider(name)` and keep `GetAgentTier`. Removed `GetSpawnCommand`/`AgentConfig.SpawnCommand`. <!-- R1 R3 -->
- [x] T002 Rewrite `internal/agent/agent.go`: five-tier `defaultTiers` with `{Provider, Model, Effort}`; new `stageTiers` (intake→default, apply→doing, review→review, hydrate→doing, ship→fast, review-pr→doing); `Profile` gains `Provider`, drops `SpawnCommand`; `ResolveTier`+`Resolve` do per-field merge with `default`-tier inheritance then built-in; added `IsTierName`; built-in `defaultProviders` table (`claude` session command, no dispatch). <!-- R3 R4 R5 -->
- [x] T003 Added `ResolveProvider` in `internal/agent`: resolve a provider name → its `{session_command, dispatch_command}` (project `providers:` per-field-merged over built-in). <!-- R1 R2 -->
- [x] T004 [P] Updated `internal/spawn/spawn.go`: `Command` reads the default tier's provider `session_command` (re-exported `DefaultSpawnCommand = agent.DefaultSessionCommand`); `WithProfile`/`resolveTemplate` unchanged; `StripPlaceholders` retirement decided in Phase 2. <!-- R2 R7 R10 -->

### Phase 2: Go commands

- [x] T005 Updated `cmd/fab/resolve_agent.go`: accepts stage OR tier positionally (`resolveStageOrTier`/`IsTierName`); renamed `spawn=` → `dispatch=`; added `provider=` line; resolves provider `dispatch_command`; full-ID under `--alias`. <!-- R8 -->
- [x] T006 Added `cmd/fab/agent.go`: `fab agent [tier] [--print] [--repo <path>]` — resolve tier, compose provider `session_command` via `WithProfile`, exec via `/bin/sh -c` (or print); registered in main.go. <!-- R7 -->
- [x] T007 Removed `cmd/fab/spawn_command.go` + `_test.go`; swapped registration to `agentCmd()`. <!-- R7 -->
- [x] T008 Updated `cmd/fab/dispatch_start.go`: resolves stage→tier→provider→`dispatch_command`; error names `providers.<name>.dispatch_command` (no fallback). <!-- R9 -->
- [x] T009 Updated `cmd/fab/operator.go`: resolves `operator` tier in-process (`operatorSpawnCommand`/`operatorProfile`); composes from the operator provider. <!-- R10 -->
- [x] T010 [P] Updated batch: shared `defaultTierSpawnCommand` in `batch.go` composes default-tier provider session_command + profile (substituted); removed `StripPlaceholders` (now dead). <!-- R10 -->
- [x] T011 Rewrote `internal/configref/configref.go`: `providers:` (claude live, codex commented), five-tier live `agent.tiers` with providers, removed `review_tools` + `agent.spawn_command`; verified valid YAML via `fab config reference` + yq. <!-- R11 -->

### Phase 3: Go tests + drift guard

- [x] T012 Updated `internal/agent/stagemodels_doc_test.go`: 4-column tier-profile parser (provider column); drift guard compares provider+model+effort. `TestDocTablesMatchAgentMaps` green against the rewritten maps + `stage-models.md` tables. <!-- R3 R4 -->
- [x] T013 [P] Updated Go tests: `config_test.go` (providers, five tiers, GetProvider), `agent_test.go` (five tiers, ResolveTier/ResolveProvider/IsTierName, default-tier inheritance), `resolve_agent_test.go` (dispatch=/provider=/tier-name/full-ID-under-alias), `dispatch_start_test.go` (provider dispatch_command), `config_test.go` cmd (providers reference + retired-keys guard), `batch_new_test.go`/`batch_switch_test.go` (profile injection), `spawn_test.go` (session_command); added `agent_test.go`. Full `go test ./...` + `go vet` green (0 failures). <!-- R7 R8 R9 R10 R11 -->

### Phase 4: Scaffold + migration

- [x] T014 [P] Updated `src/kit/scaffold/fab/project/config.yaml`: `providers.claude.session_command` + live five-tier `agent.tiers`, removed `agent.spawn_command`; added `src/kit/scaffold/fab/project/code-review.md § Review Tools` comment block. <!-- R6 R11 -->
- [x] T015 Updated fab-kit's own `fab/project/config.yaml` to the target shape (providers + five tiers, dropped `review_tools` + `agent.spawn_command`); resolves correctly via the new binary. <!-- R12 -->
- [x] T016 Added `src/kit/migrations/2.12.1-to-2.13.0.md`: relocate `agent.spawn_command`→`providers.claude.session_command`; map tiers thinking→review, doing/fast field-by-field, per-tier spawn_command→provider dispatch_command; remove `review_tools` (seed `code-review.md` only on explicit false); halt-and-ask on non-claude spawn template; sentinel-guarded on top-level `providers:`. <!-- R12 -->

### Phase 5: Skill + spec doc sweep (mirror class)

- [x] T017 Updated `src/kit/skills/_cli-fab.md`: rewrote `## fab resolve-agent` (dispatch=/provider=/tier-name), added `## fab agent`, removed `## fab spawn-command`, updated `fab dispatch` + `fab batch` + config-reference schema list + command list + operator section. <!-- R7 R8 R9 R11 -->
- [x] T018 Updated `src/kit/skills/_preamble.md` (§ Per-Stage Model Resolution / § CLI-Adapter Dispatch: spawn=→dispatch=, provider= line, {provider,model,effort}, providers table, agent.spawn_command→session command, operator tier). <!-- R1 R8 -->
- [x] T019 [P] Updated dispatch-seam skills `{fab-continue,_pipeline,fab-adopt}.md` (spawn=→dispatch=, no-fallback wording); fab-ff/fab-fff needed no change (surface only model=/effort=). <!-- R8 -->
- [x] T020 [P] Updated `_review.md` (cascade reads `code-review.md § Review Tools`) + `git-pr-review.md` (copilot toggle reads the same section) + their SPEC mirrors. <!-- R6 -->
- [x] T021 [P] Updated `{fab-operator,_cli-external}.md` (fab spawn-command → fab agent --print; operator tier resolution). <!-- R7 R10 -->
- [x] T022 Rewrote `docs/specs/stage-models.md` (five tiers + providers section + 4-col tier table + stage→tier table — drift-guarded and green; resolution/skill-wiring/boundary/Fable sections updated). <!-- R3 R4 R5 -->
- [x] T023 [P] Updated `docs/specs/{harness-adapters,architecture,glossary,skills,index}.md` (dispatch=, providers, five tiers, review_tools→code-review.md, fab agent). <!-- R1 R6 R7 R8 -->
- [x] T024 [P] Updated SPEC mirrors `SPEC-{_cli-fab,_preamble,_review,git-pr-review,fab-operator,_cli-external,fab-continue,_pipeline,fab-adopt,fab-ff}.md` for every touched skill; verified no stale spawn=/spawn-command/review_tools remain outside deliberate history notes. <!-- R6 R7 R8 -->

### Phase 6: Build + full test

- [x] T025 Built `src/go/fab` (clean), `go test ./...` (0 failures), `go vet ./...` (clean), drift guard green; `fab config reference` round-trips as valid YAML; end-to-end verified `fab agent --print`, `fab resolve-agent <stage|tier> [--alias]`, and a codex-provider `dispatch=` scenario. <!-- R11 -->

## Execution Order

- Phase 1 (T001–T004) blocks Phase 2 (commands consume the schema/resolver).
- T011 (configref) depends on T002/T003 (tier + provider tables).
- T012 (drift-guard test) depends on T022 (stage-models.md tables) — run/verify after T022.
- Phase 5 doc sweep can proceed in parallel with Phase 3 tests once Phase 2 lands.
- T025 is last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `providers:` section parses; built-in `claude` provider present; project block per-field merges; two command fields never merged — verified config.go `ProviderConfig`/`GetProvider`, agent.go `defaultProviders`/`ResolveProvider` per-field merge; config_test.go providers coverage
- [x] A-002 R2: `{model}`/`{effort}` substitution works for both command fields (append + template modes) — spawn.go `WithProfile` reused for both; verified live (codex session + dispatch templates substituted, empty-value token-drop covered by spawn_test.go)
- [x] A-003 R3: five tiers resolve with correct built-in profiles; `thinking` gone — agent.go `defaultTiers` = default/operator/doing/review/fast; verified `resolve-agent apply`=opus/xhigh; no `thinking` in Go
- [x] A-004 R4: stage→tier mapping is intake→default, apply→doing, review→review, hydrate→doing, ship→fast, review-pr→doing — agent.go `stageTiers`; drift-guard green
- [x] A-005 R5: per-field inheritance from `default` tier works (unset field inherits) — `ResolveTier` middle-layer default merge; resolve_agent_test.go `TestResolveAgentEmptyOverrideEffortInheritsDefault` + live codex `doing:{provider}` inheriting model from default
- [x] A-006 R6: `review_tools` removed from schema/configref/docs; `code-review.md § Review Tools` is the new home; absent = enabled — no `ReviewTools` in config.go; config_test.go retired-keys guard; `_review.md`/`git-pr-review.md` re-point to `code-review.md § Review Tools`; scaffold code-review.md seeds the section
- [x] A-007 R7: `fab agent [tier] [--print] [--repo]` resolves + execs/prints; `fab spawn-command` removed — agent.go new cmd registered in main.go; spawn_command.go/_test.go deleted; verified `agent --print`/`--repo`
- [x] A-008 R8: `fab resolve-agent` accepts tier names; emits `dispatch=` (not `spawn=`) + `provider=`; full-ID under `--alias` — resolve_agent.go `resolveStageOrTier`/`formatAgentProfile`; verified live + resolve_agent_test.go
- [x] A-009 R9: `fab dispatch start` resolves provider `dispatch_command`; errors on absent with new key path — dispatch_start.go; verified error names `providers.claude.dispatch_command`, no fallback; dispatch_start_test.go
- [x] A-010 R10: operator launcher resolves `operator` tier; batch/worker spawns carry the default profile — operator.go `operatorProfile`/`operatorSpawnCommand`; batch.go `defaultTierSpawnCommand`; verified `resolve-agent operator`=sonnet/medium
- [x] A-011 R11: `fab config reference` renders providers + five tiers, no `review_tools`, no `agent.spawn_command` — configref.go rewrite; config_test.go `TestConfigReferenceRetiresLegacyKeys` + providers-block guard
- [x] A-012 R12: migration relocates spawn command, maps tiers, removes review_tools; idempotent; fab-kit's own config updated — 2.12.1-to-2.13.0.md (sentinel-guarded on `providers:`); fab/project/config.yaml on v3 shape and resolves

### Behavioral Correctness

- [x] A-013 R8: `--alias` aliases `model=` while `dispatch=` keeps full ID (where a dispatch_command exists) — resolve_agent.go substitutes dispatch from full model BEFORE alias; resolve_agent_test.go `TestResolveAgentAliasDispatchUsesFullModelID`; verified live
- [x] A-014 R12: migration halts-and-asks on a non-claude spawn template; seeds `code-review.md` only on explicit `review_tools` false — migration §1.2 (UNNAMED_PROVIDER halt) + §4 (all-true silent delete / false-seed). Instruction-file correctness only (markdown migration; not executable-testable here)
- [x] A-015 R4: drift-guard `TestDocTablesMatchAgentMaps` passes against the rewritten Go maps + stage-models.md tables — ran explicitly: PASS (5 tier + 6 stage subtests)
- [x] A-016 R7: `go test ./...` green in `src/go/fab` — build + `go test ./...` + `go vet ./...` all clean (0 failures)
- [x] A-017 R1: absent `dispatch_command` ⇒ native dispatch (no fallback to `session_command`) — verified: claude review/apply emit no `dispatch=` line even with a session_command present; resolve_agent_test.go `TestResolveAgentNoDispatchThreeLines`
- [x] A-018 R9: absent provider `dispatch_command` on a CLI-dispatch stage ⇒ clear error, no fallback — verified live error; dispatch_start_test.go `TestDispatchStart_NoDispatchCommandErrors`
- [x] A-019 Pattern consistency: new Go code follows existing naming/error-handling/nil-safe-accessor patterns — `GetProvider` mirrors `GetAgentTier` nil-safe shape; `ResolveProvider` mirrors `ResolveTier`; error messages name config keys
- [x] A-020 No unnecessary duplication: reuse `internal/spawn` substitution; no re-implemented resolution — all four command-composition sites (agent/resolve-agent/dispatch-start/operator/batch) call `spawn.WithProfile`; deadcode shows no new zero-call-site symbols
- [x] A-021 Canonical source only: all skill edits in `src/kit/skills/`, none under `.claude/skills/` — verified `git status .claude/` empty
- [x] A-022 SPEC-mirror sync: every touched `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` updated — 8 of 9 mirrors touched; `SPEC-_cli-external.md` not touched but carries no stale content (see should_fix)
- [x] A-023 CLI ⇒ docs + tests: every CLI signature change updates `_cli-fab.md` + ships tests — `_cli-fab.md` `## fab agent`/`## fab resolve-agent`/`## fab dispatch`/`## fab batch` updated; agent_test.go/resolve_agent_test.go/dispatch_start_test.go/batch tests ship
- [x] A-024 Migration discipline: user-data restructuring ships as a `src/kit/migrations/` file, not an ad-hoc script — 2.12.1-to-2.13.0.md; no ad-hoc script added

### documentation_accuracy

- [x] A-025: stage-models.md, architecture.md, glossary.md, harness-adapters.md reflect the five-tier/providers/dispatch= reality with no stale thinking/spawn=/review_tools claims — all four rewritten; grep confirms remaining thinking/spawn=/review_tools mentions are deliberate rename/history notes only

### cross_references

- [x] A-026: cross-file references (skill↔SPEC, spec↔spec, memory-affected paths) stay consistent; no dangling `fab spawn-command`/`spawn=`/`review_tools` mentions in swept files — verified in src/kit + docs/specs; docs/memory stale refs are hydrate-stage scope (plan Notes), warning-only

## Notes

- Memory hydration (docs/memory) is a hydrate-stage concern, not apply — the Affected Memory files in intake.md are updated during hydrate, not here.
- This intake noted (Assumption #16) it may split into a series; apply produces one plan and executes it — no fab-change split is performed at apply.

## Deletion Candidates

- `src/go/fab/cmd/fab/spawn_command.go` + `spawn_command_test.go` — already deleted by this change (T007); `fab spawn-command` retired in favor of `fab agent`. No remaining Go callers; the only skill consumer (fab-operator) was updated in the same kit. No further action.
- `spawn.StripPlaceholders` — already deleted by this change (T010); the empty-profile leak-prevention print path disappeared with `fab spawn-command` (workers now spawn WITH a profile). No remaining call sites. No further action.
- `docs/memory/runtime/operator.md:93`, `docs/memory/_shared/configuration.md:343,345`, `docs/memory/distribution/kit-architecture.md:140,141,327,332` — stale `spawn.StripPlaceholders` / `fab spawn-command` / `agent.spawn_command` prose describing now-deleted code. These are **hydrate-stage** deletions/rewrites (Affected Memory in intake.md; plan Notes defer docs/memory to hydrate), not apply-stage — listed for the hydrate agent, NOT to delete now.
- `dispatch.Dispatch.SpawnCmd` (`yaml:"spawn_cmd"`, `src/go/fab/internal/dispatch/dispatch.go:85`) — NOT a deletion candidate: this is the persisted dispatch-record field, a separate concept from the config `spawn_command`→`dispatch_command` rename; renaming it would require its own state migration. Deliberately retained.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `fab resolve-agent` accepts stage OR tier name positionally (disjoint name sets), no `--tier` flag | Intake Open Q1 + Assumption #14; disjoint sets make positional unambiguous; matches the pure-query family | S:45 R:65 A:70 D:60 |
| 2 | Confident | `fab resolve-agent` output gains a `provider=` line | `fab agent`/operator need provider for session-command lookup; surfacing aids compliance visibility | S:50 R:70 A:70 D:65 |
| 3 | Confident | `spawn=` output line renamed `dispatch=` to match the field rename | Stated in intake §5 + Assumption #14; mechanical rename tracking the semantic | S:70 R:70 A:80 D:75 |
| 4 | Confident | Migration halts-and-asks on a non-claude `agent.spawn_command` template rather than auto-naming a provider | Intake Open Q2; auto-naming guesses grammar; surfacing is the safer, reversible choice | S:45 R:65 A:70 D:60 |
| 5 | Confident | `fab agent` exec does not TTY-guard (exec-and-let-CLI-fail) | Intake Open Q3; matches document-don't-validate contract; agent CLI handles no-TTY | S:45 R:75 A:70 D:65 |
| 6 | Confident | Tiers stay nested under `agent.tiers` (no flatten to top-level `tiers:`) | Intake Assumption #13; limits config churn; user phrasing implied keeping `agent:` | S:55 R:80 A:70 D:65 |
| 7 | Certain | Migration file named `2.12.1-to-2.13.0.md` (minor bump: new command + schema change) | Current VERSION is 2.12.1; SemVer minor for additive command + schema | S:80 R:85 A:90 D:85 |
| 8 | Confident | Memory (docs/memory) updates are deferred to hydrate, not done in apply | Constitution II + fab-continue Hydrate Behavior owns docs/memory; apply owns source + specs (SPEC mirror is constitution-required in-change) | S:70 R:75 A:85 D:75 |
| 9 | Confident | No fab-change series split at apply; one plan executes the full intake scope | Block contract prohibits status transitions; splitting is an orchestration decision above apply | S:60 R:70 A:80 D:70 |
| 10 | Tentative | `provider=` line placement and `--alias` interaction on native tiers are the plan's call; exact byte-order of resolve-agent output settled during T005 | Intake Open Q1 left the precise output contract to plan generation; low blast radius, test-pinned | S:35 R:60 A:55 D:45 |

10 assumptions (1 certain, 8 confident, 1 tentative).
