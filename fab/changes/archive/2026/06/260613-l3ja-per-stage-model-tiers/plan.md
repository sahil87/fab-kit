# Plan: Per-Stage Model Selection via Named Tiers

**Change**: 260613-l3ja-per-stage-model-tiers
**Intake**: `intake.md`

## Requirements

<!-- Derived from the settled intake design. RFC-2119 statements with stable R# IDs,
     each with at least one GIVEN/WHEN/THEN scenario. -->

### Config: `agent.tiers` override surface

#### R1: Optional `agent.tiers` config field
The Go `Config` struct SHALL model an optional `agent.tiers` map under the existing `agent:` block. Each entry maps a tier name (`thinking`/`doing`/`ship`) to a `{model, effort}` object (yaml keys `model`, `effort`). The struct widening SHALL NOT break existing configs (yaml.v3 ignores unknown keys; a config without `agent.tiers` parses cleanly).

- **GIVEN** a `config.yaml` with `agent.tiers.doing: {model: claude-sonnet-4-6, effort: medium}`
- **WHEN** the config is loaded
- **THEN** `AgentConfig.Tiers["doing"]` SHALL hold `{Model: "claude-sonnet-4-6", Effort: "medium"}`
- **AND** a config with no `agent.tiers` block SHALL load without error and yield an empty/absent tier map

#### R2: Nil-safe tier accessor
The config package SHALL expose a nil-safe accessor returning the configured `{model, effort}` override for a tier name (and whether one was set), returning the zero value for a nil `*Config`, an absent `tiers` block, or an unconfigured tier.

- **GIVEN** a nil `*Config` or a `Config` with no `agent.tiers`
- **WHEN** the tier accessor is called for any tier name
- **THEN** it SHALL report "no override" without panicking

### Tier/Stage Tables: fab-owned defaults and the fixed mapping

#### R3: Default tier → `{model, effort}` table
fab-kit SHALL own a default tier→profile table in Go: `thinking` = `{claude-opus-4-8, xhigh}`, `doing` = `{claude-opus-4-8, high}`, `ship` = `{claude-sonnet-4-6, low}`. This is the single place bumped when a new top model lands (Fable upgrade path).

- **GIVEN** no project override for a tier
- **WHEN** that tier is resolved
- **THEN** the resolved profile SHALL equal fab-kit's default for that tier

#### R4: Fixed, non-overridable stage → tier mapping
fab-kit SHALL own a fixed stage→tier mapping covering exactly the six pipeline stages: `thinking` = {`intake`, `review`}, `doing` = {`apply`, `review-pr`, `hydrate`}, `ship` = {`ship`}. The mapping SHALL NOT be user-overridable (there is no `stage_tiers` config and no per-stage escape hatch). `review` (generative → `thinking`) and `review-pr` (responsive → `doing`) SHALL NOT be grouped together.

- **GIVEN** the stage `review`
- **WHEN** its tier is looked up
- **THEN** the tier SHALL be `thinking`
- **AND GIVEN** the stage `review-pr`, its tier SHALL be `doing`

### Resolver command: `fab resolve-agent <stage>`

#### R5: Stage→tier→profile resolution
`fab resolve-agent <stage>` SHALL map the stage through the fixed stage→tier mapping (R4), then resolve the tier to a `{model, effort}` profile: the project's `agent.tiers.<tier>` override per-field-merged over fab-kit's default (R3) when present, else fab-kit's default. An omitted override field SHALL fall back to the default for that field.

- **GIVEN** a project config with `agent.tiers.doing: {effort: medium}` (model omitted)
- **WHEN** `fab resolve-agent apply` runs (apply ∈ doing)
- **THEN** the resolved model SHALL be the default `claude-opus-4-8` and the resolved effort SHALL be `medium`

#### R6: Verbatim pass-through — NO validation
`fab resolve-agent` SHALL NOT validate the resolved model or effort against any provider's accepted set. It SHALL echo both strings verbatim (provider-neutral). It SHALL NOT enforce any effort enum, SHALL NOT drop an effort the model would reject, and SHALL NOT correct a misconfigured pair.

- **GIVEN** a project override `agent.tiers.ship: {model: claude-sonnet-4-6, effort: xhigh}` (Sonnet rejects `xhigh` at dispatch)
- **WHEN** `fab resolve-agent ship` runs
- **THEN** it SHALL emit `effort=xhigh` verbatim without error (the harness, not fab, is the safety net)

#### R7: Output shape — `model=` / `effort=` lines, byte-stable
`fab resolve-agent` SHALL write two stdout lines: `model=<id>` and `effort=<level>`. The `effort=` line SHALL be omitted when the resolved tier has no effort (empty/absent). An empty resolved model SHALL emit `model=` empty (signals "inherit the session/orchestrator model"). Output SHALL be byte-stable for the same config.

- **GIVEN** the default `thinking` tier resolves for `intake`
- **WHEN** `fab resolve-agent intake` runs
- **THEN** stdout SHALL be exactly `model=claude-opus-4-8\neffort=xhigh\n`
- **AND GIVEN** a tier resolving to an empty effort, the `effort=` line SHALL be omitted

#### R8: Error semantics
`fab resolve-agent` SHALL exit non-zero only on a real error: an unreadable/malformed config, or an unknown stage name. A stage resolving to a default SHALL be success (exit 0), not an error.

- **GIVEN** an unknown stage name `frobnicate`
- **WHEN** `fab resolve-agent frobnicate` runs
- **THEN** it SHALL exit non-zero with an error naming the unknown stage
- **AND GIVEN** a valid stage with no overrides, it SHALL exit 0

### Drift Guard: docs ↔ Go table parity

#### R9: Drift-guard test for both tables
A Go test SHALL guard against drift between the Go default-tier table (R3) and the fixed stage→tier mapping (R4) and their mirror tables in `docs/specs/stage-models.md`, mirroring the `TestDocTablesMatchScoringMaps` pattern (`internal/score` ↔ `docs/specs/change-types.md`). The test SHALL fail if either doc table disagrees with the Go maps, or covers a different set of tiers/stages.

- **GIVEN** the doc table in `stage-models.md` and the Go maps
- **WHEN** the drift-guard test runs
- **THEN** it SHALL pass only when every tier's default profile and every stage's tier assignment match between doc and code, for the exact same tier/stage sets

### Skill wiring: orchestrators consume `fab resolve-agent`

#### R10: Dispatch resolves and passes model + effort
The orchestrator dispatch contract SHALL instruct callers to run `fab resolve-agent <stage>` immediately before dispatching each stage's sub-agent and to pass the resolved model AND effort to the Agent dispatch. An empty model SHALL omit the `model` param (inherit); an empty effort SHALL omit the effort flag. The Claude Code adapter — the Agent tool `model` parameter — SHALL be named explicitly as harness-specific; the resolution itself SHALL be described as provider-neutral.

- **GIVEN** an orchestrator about to dispatch the apply sub-agent
- **WHEN** it follows the dispatch contract
- **THEN** it SHALL first run `fab resolve-agent apply` and pass the resolved model/effort to the Agent tool, omitting an empty model/effort

#### R11: Review resolves once for both reviewers + merge
The `review` stage dispatch SHALL resolve `fab resolve-agent review` ONCE and apply the same `{model, effort}` profile to both reviewer sub-agents (inward + outward) and the merge.

- **GIVEN** the review stage dispatch
- **WHEN** it resolves the agent profile
- **THEN** it SHALL call `fab resolve-agent review` exactly once and reuse the result for inward, outward, and merge

#### R12: Foreground advisory-only
The skills SHALL document that per-stage selection applies to orchestrated/sub-agent runs only; a foreground `/fab-continue` run cannot switch the session model mid-run and the configured tier is advisory there (MAY note the mismatch, MUST NOT attempt to switch).

- **GIVEN** a user running a stage directly in the foreground
- **WHEN** the stage's configured tier differs from the session model
- **THEN** the skill MAY note the mismatch but MUST NOT attempt a switch

### Documentation & Migration

#### R13: `docs/specs/stage-models.md` reconciled to final design
`docs/specs/stage-models.md` SHALL be reconciled to the settled design: tiers as `{model, effort}`; the three named tiers `thinking`/`doing`/`ship` with the final stage groupings; `agent.tiers` as the sole override; the fixed non-overridable stage→tier mapping; default profiles; `fab resolve-agent`; NO validation / verbatim pass-through (removing all "degrade gracefully / enforce effort enum / drop incompatible effort" language); provider-neutral-by-construction with the harness-adapter boundary; Haiku-excluded-from-defaults (not forbidden); apply↔review coupling; Fable upgrade path; foreground-advisory-only. It SHALL carry the drift-guard mirror tables (R9).

- **GIVEN** the reconciled spec
- **WHEN** it is read
- **THEN** it SHALL contain no `stage_models` map, no `high`/`mid`/`low` tier names, and no effort-validation/degrade-gracefully rule

#### R14: `docs/specs/architecture.md` documents `agent.tiers`
`docs/specs/architecture.md` SHALL document the `agent.tiers` config block alongside the existing `stage_hooks` example in the config-schema section.

- **GIVEN** the architecture config-schema example
- **WHEN** it is read
- **THEN** it SHALL show an `agent.tiers` block with the fixed-mapping reference and override shape

#### R15: `_cli-fab.md` documents `fab resolve-agent`
`src/kit/skills/_cli-fab.md` SHALL document the new `fab resolve-agent <stage>` command signature, output shape, and error semantics (constitution CLI constraint).

- **GIVEN** the CLI reference
- **WHEN** `fab resolve-agent` is looked up
- **THEN** it SHALL document the stage arg, the `model=`/`effort=` output, verbatim pass-through, and error-only-on-real-error semantics

#### R16: Migration adds a fully-commented `agent.tiers` block
A new migration file in `src/kit/migrations/` SHALL add a FULLY COMMENTED `agent.tiers` block to existing `fab/project/config.yaml` files, documenting the fixed stage→tier mapping (reference), fab-kit's default profiles, and the override shape — all commented out (fab-kit uses built-in defaults by default). It SHALL be idempotent (re-run does not duplicate the block).

- **GIVEN** a project config without an `agent.tiers` block
- **WHEN** the migration is applied
- **THEN** a fully-commented `agent.tiers` reference block SHALL be inserted under `agent:`
- **AND** re-running the migration SHALL be a no-op (block already present)

### Design Decisions

1. **Package placement: new `internal/agent` package** (owns the default tier table, the fixed stage→tier mapping, and the `Resolve` function) — *Why*: `internal/config` documents itself as the single owner of *config.yaml parsing* (the `AgentConfig.Tiers` field + nil-safe accessor belong there). The tier/stage *tables* and the *resolution cascade* are a distinct domain — exactly parallel to how `internal/score` owns the `expectedMin`/`gateThresholds` tables + resolution logic (with its drift-guard test in-package) rather than living in `internal/config`. A small new package keeps config-parsing and tier-resolution concerns separated and gives the drift-guard test a natural home. — *Rejected*: cramming the tables + resolver into `internal/config` (mixes parsing with policy, and the score precedent argues for a domain package).
2. **Tiers are `{model, effort}` profiles; three tiers `thinking`/`doing`/`ship`** — *Why*: settled in the intake; effort is a first-class spend lever. — *Rejected*: bare-model tiers (effort is the primary Opus-stage lever).
3. **Stage→tier mapping is fab-owned and fixed** — *Why*: the taxonomy is fab's considered judgment (dimensional analysis); users override budget, not taxonomy. — *Rejected*: `stage_tiers` reassignment / per-stage escape hatch.
4. **`agent.tiers` is the sole override surface; per-field merge over defaults** — *Why*: an omitted field should inherit fab's default, not blank it. — *Rejected*: whole-profile replacement (loses the merge ergonomics).
5. **No validation — verbatim pass-through** — *Why*: provider neutrality (Constitution Principle I); validating against Claude's effort enum would hard-code Claude. The safety net moves to the runtime/harness. — *Rejected*: effort-enum enforcement / degrade-gracefully drop (the earlier `stage-models.md` iteration — explicitly removed).
6. **Provider-neutral by construction; Claude Code adapter named explicitly** — *Why*: config + resolution are provider-agnostic; only the dispatch injection (Agent tool `model` param) is harness-specific, and that coupling pre-exists this feature. v1 is architecture-neutral + documented, NOT shipped/tested against a non-Claude harness. — *Rejected*: per-provider default tables / provider-detection (far larger change, out of scope).
7. **Haiku excluded from defaults (not forbidden)** — *Why*: no effort param (400s); ship needs faithful PR-description comprehension. A user MAY still override a tier to Haiku via pass-through. — *Rejected*: a special-case no-effort path in the tier system.
8. **apply↔review coupling keeps apply on `doing` (Opus/high)** — *Why*: apply produces the diff review critiques; a cheaper apply drives more (capped-at-3) rework rounds; savings come from effort (high not xhigh), not a model downgrade. — *Rejected*: dropping apply to `ship`/Sonnet.
9. **Fable upgrade path** — *Why*: a non-overriding project upgrades for free when fab bumps the default table (thinking→Fable/xhigh, doing→Opus/xhigh); an overriding project opts out for that tier (correct, documented). — *Rejected*: pinned model IDs in config.
10. **Output shape `model=`/`effort=` lines** (Confident, recorded below) and **migration version `2.2.0-to-2.3.0`** (FROM = current release, TO = next minor — j6cs/c5tr precedent).

### Non-Goals

- Shipped/tested multi-provider support (non-Claude harness integration, per-provider default tables, provider-detection) — explicitly out of scope; v1 proves architecture-neutrality, not a running non-Claude harness.
- A user (`~/.fab-kit`) config layer — dropped in design.
- Role-granular keys (`review.inward`, `review.merge`) and per-invocation `--model-<stage>` flags — deferred.
- Any `.status.yaml` schema change — this is config-only.
- Cost/latency telemetry per tier.

## Tasks

### Phase 1: Go core — config, tables, resolver

- [x] T001 Add `TierProfile` struct (`Model`, `Effort` with yaml `model`/`effort`) and `Tiers map[string]TierProfile` (yaml `tiers`) to `AgentConfig` in `src/go/fab/internal/config/config.go`; add a nil-safe accessor `GetAgentTier(tier string) (TierProfile, bool)`. Extend `src/go/fab/internal/config/config_test.go` (load with tiers, no tiers, nil-safe accessor, malformed coupled-failure parity). <!-- R1 --> <!-- R2 -->
- [x] T002 Create `src/go/fab/internal/agent/` package: `agent.go` with the default tier→profile table (`thinking`=opus-4-8/xhigh, `doing`=opus-4-8/high, `ship`=sonnet-4-6/low), the fixed stage→tier mapping (thinking={intake,review}, doing={apply,review-pr,hydrate}, ship={ship}), and a `Resolve(cfg *config.Config, stage string) (Profile, error)` that maps stage→tier→per-field-merged profile (override-over-default), NO validation, unknown stage → error. Add `agent_test.go` (resolution, per-field merge, verbatim pass-through, unknown-stage error, default-on-no-override, review→thinking / review-pr→doing). <!-- R3 --> <!-- R4 --> <!-- R5 --> <!-- R6 -->
- [x] T003 Add the `fab resolve-agent <stage>` cobra command in `src/go/fab/cmd/fab/resolve_agent.go` (cobra patterns mirroring `resolve.go`/`score.go`): `ExactArgs(1)`, calls `agent.Resolve`, prints `model=<id>` and (when non-empty) `effort=<level>`, byte-stable, returns error (non-zero exit) only on malformed config / unknown stage. Register in `src/go/fab/cmd/fab/main.go`. Add `src/go/fab/cmd/fab/resolve_agent_test.go` (default output exact bytes, override merge, effort-omitted line, empty-model inherit, unknown-stage error). <!-- R7 --> <!-- R8 -->

### Phase 2: Drift guard + docs reconciliation

- [x] T004 Add the drift-guard test `src/go/fab/internal/agent/stagemodels_doc_test.go` mirroring `TestDocTablesMatchScoringMaps`: parse the default-tier table and the stage→tier table from `docs/specs/stage-models.md` (walk-up `findDocFile`, line-based pipe parsing — no markdown lib), assert both directions (tier/stage set coverage + per-entry value match against the Go maps). <!-- R9 -->
- [x] T005 Reconcile `docs/specs/stage-models.md` to the final design (R13): tiers as `{model,effort}`; `thinking`/`doing`/`ship` with final groupings; `agent.tiers` sole override; fixed non-overridable stage→tier mapping; default profiles; `fab resolve-agent` (verbatim pass-through, error semantics); remove all validation/degrade-gracefully/effort-enum-enforcement language; provider-neutral-by-construction + harness-adapter boundary; Haiku-excluded-from-defaults rationale; apply↔review coupling; Fable upgrade path; foreground-advisory-only. Include the two drift-guard mirror tables in the exact shape the T004 parser expects. <!-- R13 -->
- [x] T006 [P] Document the `agent.tiers` config block in `docs/specs/architecture.md` alongside the `stage_hooks` example (~line 217-232). <!-- R14 -->

### Phase 3: CLI doc + skill dispatch wiring

- [x] T007 [P] Document `fab resolve-agent` in `src/kit/skills/_cli-fab.md` (new `## fab resolve-agent` section: signature, `model=`/`effort=` output, verbatim pass-through, error-only-on-real-error). Surface in `_preamble.md` § Common fab Commands only if warranted (it is orchestrator-internal — note decision in Assumptions). <!-- R15 -->
- [x] T008 Wire dispatch in `src/kit/skills/_pipeline.md` (Steps 1–3 + the Dispatch note): before each `/fab-continue`-behavior subagent dispatch, run `fab resolve-agent <stage>` and pass resolved model+effort to the Agent dispatch (empty model ⇒ omit/inherit; empty effort ⇒ omit). Name the Claude Code adapter (Agent tool `model` param) explicitly; resolution is provider-neutral. <!-- R10 -->
- [x] T009 Wire the per-stage resolution + the "resolve once for both reviewers + merge" rule and the foreground-advisory note into `src/kit/skills/fab-continue.md` (Apply/Review/Hydrate dispatch + a foreground note) and the orchestrator dispatch in `src/kit/skills/fab-fff.md`, `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-proceed.md`, and `src/kit/skills/_preamble.md` § Subagent Dispatch (the dispatch contract). <!-- R10 --> <!-- R11 --> <!-- R12 -->
- [x] T010 Update the corresponding `docs/specs/skills/SPEC-*.md` for every changed skill (constitution: skill change ⇒ SPEC update): `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-fab-fff.md`, `SPEC-fab-ff.md`, `SPEC-fab-proceed.md`, `SPEC-_preamble.md`. <!-- R10 --> <!-- R11 --> <!-- R12 -->

### Phase 4: Migration + verification

- [x] T011 Add migration `src/kit/migrations/2.2.0-to-2.3.0.md` (per `docs/memory/distribution/migrations.md` format): Summary / Pre-check (config.yaml present; `agent.tiers` absent) / Changes (insert a FULLY COMMENTED `agent.tiers` reference block under `agent:` — fixed mapping reference, default profiles, override shape, all commented out) / Verification (idempotent; other keys untouched). <!-- R16 -->
- [x] T012 Run `cd src/go/fab && go build ./... && go test ./...`; fix failures at the source; ensure the drift-guard test passes against the reconciled `stage-models.md`. <!-- R9 --> <!-- R13 -->

## Execution Order

- T001 → T002 → T003 (config field → tables/resolver → command depend in sequence).
- T004 depends on T002 (Go maps) and is validated by T005 (doc tables) — author T005 before running T012 so the drift-guard passes.
- T006, T007 are independent ([P]).
- T008 → T009 → T010 (dispatch contract in `_pipeline`/`_preamble` first, then driver skills, then SPECs).
- T011 independent of the Go work.
- T012 last (full build + test sweep).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `AgentConfig` has an optional `Tiers` map of tier→`{model,effort}`; a config with and without `agent.tiers` both load without error.
- [x] A-002 R2: a nil-safe tier accessor returns "no override" for nil `*Config`, absent block, and unconfigured tier without panicking.
- [x] A-003 R3: the Go default-tier table is `thinking`=opus-4-8/xhigh, `doing`=opus-4-8/high, `ship`=sonnet-4-6/low.
- [x] A-004 R4: the fixed stage→tier mapping is thinking={intake,review}, doing={apply,review-pr,hydrate}, ship={ship}; `review`→thinking and `review-pr`→doing are not grouped.
- [x] A-005 R5: `fab resolve-agent <stage>` resolves stage→tier→per-field-merged profile (override over default, omitted field inherits default).
- [x] A-006 R6: the resolver performs NO validation and echoes model+effort verbatim (no effort-enum enforcement, no incompatible-pair correction).
- [x] A-007 R7: stdout is `model=<id>` + `effort=<level>`; the `effort=` line is omitted when the tier has no effort; empty model emits empty `model=`; output is byte-stable.
- [x] A-008 R8: the command exits non-zero only on malformed config or unknown stage; a default resolution exits 0.
- [x] A-009 R9: a drift-guard test fails if the Go default-tier table or stage→tier mapping diverges from the `stage-models.md` mirror tables (both directions: set coverage + per-entry values).
- [x] A-010 R10: the dispatch contract instructs callers to run `fab resolve-agent <stage>` before each stage's sub-agent dispatch and pass model+effort (empty model ⇒ inherit; empty effort ⇒ omit), naming the Claude Code Agent-tool `model` adapter explicitly.
- [x] A-011 R11: the review dispatch resolves `fab resolve-agent review` once and applies it to both reviewers + merge.
- [x] A-012 R12: foreground runs document the advisory-only behavior (MAY note mismatch, MUST NOT switch).
- [x] A-013 R13: `stage-models.md` reflects the final design and contains no `stage_models` map, no `high`/`mid`/`low` tiers, and no validation/degrade-gracefully language.
- [x] A-014 R14: `architecture.md` documents the `agent.tiers` block alongside `stage_hooks`.
- [x] A-015 R15: `_cli-fab.md` documents `fab resolve-agent` (signature, output, pass-through, error semantics).
- [x] A-016 R16: the migration adds a fully-commented `agent.tiers` block idempotently and leaves other config keys untouched.

### Behavioral Correctness

- [x] A-017 R5: a partial override (`agent.tiers.doing: {effort: medium}`) yields default model + overridden effort.
- [x] A-018 R6: an intentionally-incompatible override (Sonnet + `effort=xhigh`) is emitted verbatim with exit 0.

### Scenario Coverage

- [x] A-019 R7: `fab resolve-agent intake` on a default config emits exactly `model=claude-opus-4-8\neffort=xhigh\n`.
- [x] A-020 R8: `fab resolve-agent frobnicate` exits non-zero naming the unknown stage.
- [x] A-021 R16: re-running the migration on a config that already has `agent.tiers` is a no-op.

### Edge Cases & Error Handling

- [x] A-022 R7: an empty resolved model emits an empty `model=` line (inherit) and omits no other contract.
- [x] A-023 R8: a malformed config surfaces a non-zero exit with stderr error.

### Code Quality

- [x] A-024 Pattern consistency: new Go follows the cobra patterns of `resolve.go`/`score.go`, the package-with-tables+drift-test pattern of `internal/score`, and the nil-safe accessor style of `internal/config`.
- [x] A-025 No unnecessary duplication: resolution reuses `internal/config` loading + accessors and the established `findDocFile`/pipe-parse drift-test helpers rather than reimplementing.
- [x] A-026 Readability over cleverness: no god functions (>50 lines); tier/stage tables are named maps, not magic strings scattered across call sites.
- [x] A-027 Composition: the resolver composes config loading + the agent-package tables; it does not duplicate config parsing.

### Documentation Accuracy (checklist.extra_categories)

- [x] A-028 Documentation accuracy: `stage-models.md`, `architecture.md`, and `_cli-fab.md` describe the as-built behavior (verbatim pass-through, fixed mapping, `model=`/`effort=` output) with no stale `stage_models`/`high`/`mid`/`low`/validation residue.

### Cross References (checklist.extra_categories)

- [x] A-029 Cross references: every changed `src/kit/skills/*.md` has its matching `docs/specs/skills/SPEC-*.md` updated; the migration is recorded per `migrations.md` format; doc tables and Go maps cross-reference each other (drift-guard cite both ways).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | New `internal/agent` package holds the tier/stage tables + resolver (not `internal/config`) | `internal/config` documents itself as the single config-parsing owner; tables+resolution are a distinct domain mirroring `internal/score` (tables + in-package drift test). Reversible (internal package, no external API). Intake left this an apply-time structure decision. | S:70 R:75 A:85 D:75 |
| 2 | Confident | `fab resolve-agent` output = `model=<id>` / `effort=<level>` stdout lines, effort line omitted when empty | Intake assumption #11 (Confident); consistent with byte-stable `fab resolve` query family; consuming skills only need a documented stable contract. | S:75 R:75 A:85 D:80 |
| 3 | Confident | Drift-guard test mirrors `TestDocTablesMatchScoringMaps` (walk-up findDocFile + line-based pipe parse, both-direction assertions) | Intake assumption #12 (Confident); established project pattern for doc↔Go parity. | S:85 R:80 A:90 D:85 |
| 4 | Confident | Migration file named `2.2.0-to-2.3.0.md` (FROM = current release 2.2.0, TO = next minor 2.3.0) | j6cs/c5tr naming precedent (FROM = current release, TO = next minor); range non-overlap with the existing `2.1.6-to-2.2.0.md` (its TO = this FROM, no overlap). | S:80 R:75 A:90 D:85 |
| 5 | Confident | `fab resolve-agent` documented in `_cli-fab.md` only (NOT surfaced in `_preamble.md` Common fab Commands) | It is orchestrator-internal dispatch plumbing, not a top-6 most-used family; the Common table is reserved for high-frequency user/skill commands. Reversible doc placement. | S:70 R:85 A:80 D:75 |
| 6 | Confident | Tier accessor signature `GetAgentTier(tier) (TierProfile, bool)` returns the override + presence flag (per-field merge done in the resolver, not the accessor) | Mirrors the nil-safe accessor style (`GetStageHook` returns zero-value; the bool lets the resolver distinguish "no override" from "override with empty fields" for per-field merge). | S:70 R:80 A:85 D:75 |
| 7 | Confident | `src/kit/VERSION` left at 2.2.0 (NOT bumped to 2.3.0 in this change); the bump is release-owned | The recent history shows VERSION/`release: vX` as dedicated release commits, not feature commits; `release.sh` only warns when no migration targets the new version — the migration now exists, so the next release can flip VERSION to 2.3.0 cleanly. Bumping VERSION mid-feature would mark the engine as 2.3.0 before a release ships. Reversible (a one-line release-time edit). | S:70 R:90 A:75 D:75 |
| 8 | Confident | Migration appends a fully-COMMENTED block (no live `tiers:` key) using a reference-comment sentinel (`# agent.tiers — per-stage model`) for idempotency | Intake assumption #7 (fully commented; fab-kit uses built-in defaults); the comment sentinel mirrors the commented-field idempotency check of `0.34.0-to-0.37.0.md` (linear_workspace) since a commented block leaves no parseable yaml key to detect on re-run. | S:90 R:80 A:90 D:85 |

8 assumptions (0 certain, 8 confident, 0 tentative).
