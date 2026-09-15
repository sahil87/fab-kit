# Plan: Agent Profiles Reshape + Session/Workers Knobs

**Change**: 260806-j9nh-agent-profiles-session-workers
**Intake**: `intake.md`

## Requirements

### Config: The two advertised depth knobs

#### R1: `agent.session` / `agent.workers` select the provider by agent depth
`fab/project/config.yaml` SHALL support two scalar keys, `agent.session` and `agent.workers`, each naming a provider. `agent.session` governs the **Tier 1** roles (agents the user talks to); `agent.workers` governs the **Tier 2** roles (agents pipeline stages dispatch to). Both SHALL default to `claude`. The role→depth partition SHALL be fixed and fab-owned (not user-overridable): `default`, `operator` → session; `doing`, `review`, `hydrate`, `fast` → workers. Both keys SHALL be scope `both`.

- **GIVEN** a config with `agent: { workers: gemini }` and no other agent keys
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `provider=gemini` (apply → `doing` → a workers role)
- **AND** `fab resolve-agent operator` still emits `provider=claude` (a session role)

- **GIVEN** a config with no `agent:` block at all
- **WHEN** any role or stage is resolved
- **THEN** the provider is `claude` for every role (byte-identical to today)

### Config: `agent.profiles` — the sparse per-role override

#### R2: `agent.tiers` is renamed to `agent.profiles` with sparse per-role semantics
The `agent.tiers` map SHALL be renamed `agent.profiles`, keyed by the six fixed **role** names, each value a `{provider, model, effort}` object with every field optional. Each set field SHALL beat the depth knob (for `provider`) and the provider fill (for `model`/`effort`). The agent-side `default`-role inheritance SHALL be **removed**: `agent.profiles.default` is the `default` role's own override only, never a fallback source for another role.

- **GIVEN** `agent.profiles: { default: { model: X }, doing: { effort: high } }`
- **WHEN** `fab resolve-agent apply` (role `doing`) runs
- **THEN** `effort=high` and the model comes from the resolved provider's `doing` fill — **not** from `agent.profiles.default.model`

#### R3: A legacy `agent.tiers:` block keeps resolving (read-time carry-forward)
A config still carrying `agent.tiers:` SHALL keep resolving: the loader SHALL read `tiers` as a deprecated alias consulted **per role** when `agent.profiles` has no entry for that role. The registry row for `agent.profiles` SHALL carry `renamed_from: agent.tiers`, and the shipped migration SHALL perform the on-disk rewrite (R8).

- **GIVEN** a config carrying only `agent: { tiers: { doing: { effort: medium } } }`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `effort=medium` — the legacy key still resolves
- **AND** a config carrying **both** `profiles.doing` and `tiers.doing` resolves from `profiles.doing`

### Config: per-role provider fills

#### R4: `providers.<name>.profiles.<role>` supplies per-role model/effort
Each provider entry SHALL support a nested `profiles` map keyed by role name, each value `{model, effort}` — "when this provider plays this role, use this model/effort". The map SHALL per-field merge over the built-in provider table exactly as the command fields do. The flat `providers.<name>.model`/`.effort` fill SHALL be removed from the documented surface and folded into `providers.<name>.profiles.default`; the loader SHALL keep reading the flat fields as a deprecated alias for `profiles.default` so a pre-migration config keeps resolving.

- **GIVEN** `providers.codex.profiles: { review: { model: gpt-5.3-codex, effort: high } }` and `agent.workers: codex`
- **WHEN** `fab resolve-agent review` runs
- **THEN** `provider=codex`, `model=gpt-5.3-codex`, `effort=high`
- **AND** with only `providers.codex.profiles.default.model` set, every workers role resolves that model

#### R5: The provider-name cutoff rule is removed
The cross-provider field cutoff (`cutForeignFields`, per-field ownership tracking, the net-configured-provider gate) SHALL be deleted. Cross-role fallback SHALL exist only on the provider side, so a role whose provider is swapped naturally refills from that provider's own profile map.

- **GIVEN** `agent.workers: codex` and no `providers.codex.profiles`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=` is empty and no `effort=` line is emitted (never claude's values), with `dispatch=codex exec …` and both placeholder tokens dropped

### Resolution: the precedence chain

#### R6: `fab resolve-agent` implements the new precedence, CLI surface unchanged
Resolution SHALL follow:

- **provider**: invocation `--provider` → `agent.profiles.<role>.provider` → the depth knob for the role's partition → built-in `claude`
- **model / effort (per field)**: invocation flag → `agent.profiles.<role>.<field>` → `providers.<prov>.profiles.<role>.<field>` → `providers.<prov>.profiles.default.<field>` → empty

where `<prov>` is the **resolved** provider (post-override). The command surface SHALL be unchanged: `fab resolve-agent <stage|role> [--alias] [--provider|--model|--effort]`, the same `model=`/`effort=`/`provider=`/`dispatch=` output lines, the same `--alias` semantics, the same `dispatch=` emission rules (`dispatch_command` presence; the `dispatch.watchable` + tmux opt-in), the same verbatim no-validation pass-through, and the same unknown-`--provider` lookup error.

- **GIVEN** the default config
- **WHEN** `fab resolve-agent apply --alias` runs
- **THEN** the output is byte-identical to today's (`model=opus`, `effort=high`, `provider=claude`)
- **WHEN** `fab resolve-agent apply --provider codex` runs
- **THEN** the provider swaps and model/effort refill from codex's own profiles (then empty) — no cutoff rule involved

#### R7: Launch-time spawners resolve through the same chain
`fab agent` (role-addressed), `fab batch new`/`switch`, and the `fab operator` launcher SHALL compose their session command from the role profile resolved by the same chain, so `agent.session` governs which provider a Tier-1 session launches on. Stage-dispatch sites SHALL be untouched — they already call `fab resolve-agent <stage>`.

- **GIVEN** `agent.session: codex` and `providers.codex.profiles.operator: { model: gpt-5.3-codex }`
- **WHEN** `fab operator` composes its launch command
- **THEN** it composes `providers.codex.session_command` with that model
- **AND** `fab agent --print` (role `default`) likewise composes the codex session command

### Defaults & registry

#### R8: `defaults.yaml` is reshaped; resolved defaults are byte-identical
The embedded `src/go/fab/internal/agent/defaults.yaml` SHALL carry `agent.session: claude`, `agent.workers: claude`, and the six claude role models under `providers.claude.profiles.<role>`. It SHALL define no `agent.tiers`/`agent.profiles` block. Every stage's resolved `{provider, model, effort}` SHALL be byte-identical to today's.

- **GIVEN** an empty project config
- **WHEN** each of the six stages is resolved
- **THEN** the resolved profiles equal today's shipped values (`default` fable/high, `operator` sonnet/medium, `doing` opus/high, `review` opus/high, `hydrate` opus/high, `fast` sonnet/medium — `doing`/`review` are `high`, not `xhigh`, since #537 landed mid-apply; see the Apply note)

#### R9: The registry advertises the two knobs and demotes the machinery
`internal/configref` SHALL register `agent.session` and `agent.workers` (scope `both`, `advertise: true`) and SHALL demote `agent.profiles` and `providers` to `advertise: false` while keeping them in `fab config reference` (YAML + `--json`). `agent.profiles` SHALL carry `renamed_from: agent.tiers`. The `agent:` YAML block SHALL be rendered by exactly one segment (the `agent.session` row) so the reference keeps round-tripping. The rendered fence's provider/agent commentary SHALL shrink from ~90 lines to at most ~20.

- **GIVEN** a project with no `agent:`/`providers:` overrides
- **WHEN** `fab config upgrade` regenerates the managed fence
- **THEN** the fence carries the two-knob `agent:` block and no `providers:`/`agent.profiles` scaffold
- **AND** `fab config reference` still documents both, and `fab config reference --json` still lists them

#### R10: A migration rewrites existing user configs
`src/kit/migrations/2.16.19-to-2.17.0.md` SHALL rewrite `agent.tiers:` → `agent.profiles:` and `providers.<p>.model`/`.effort` → `providers.<p>.profiles.default.{model,effort}` in `fab/project/config.yaml` **and** in the system config `~/.fab-kit/config.yaml`. It SHALL be sentinel-guarded and idempotent.

- **GIVEN** a config with `agent: { tiers: { doing: { effort: high } } }`
- **WHEN** the migration runs
- **THEN** the key is `agent: { profiles: { doing: { effort: high } } }` with values and comments preserved
- **AND** re-running the migration is a complete no-op

### Nomenclature & docs

#### R11: Vocabulary is role / profile / provider / Tier 1–2, sweeping the whole mirror class
The vocabulary SHALL be: **role** = one of the six fixed slot names; **profile** = a `{provider, model, effort}` value; **provider** = grammar plus per-role fills; **Tier 1 / Tier 2** = agent *depth*. Go identifiers SHALL follow (`TierDefault` → `RoleDefault`, `defaultTiers` → parsed per-role profiles, `TierForStage` → `RoleForStage`, `ResolveTier` → `ResolveRole`, `IsTierName` → `IsRoleName`, `TierNames` → `RoleNames`, `DefaultTier` → `DefaultProfile`, `stageTiers` → `stageRoles`, `config.TierProfile` → `config.RoleProfile`); the six role **strings** SHALL be unchanged, so `fab resolve-agent review` keeps working. The doc sweep SHALL cover `docs/specs/stage-models.md`, `config.md`, `glossary.md`, `architecture.md`, `src/kit/skills/_preamble.md`, `_cli-fab.md`, `_cli-agents.md`, and every affected `docs/specs/skills/SPEC-*.md` mirror.

- **GIVEN** the shipped binary
- **WHEN** `fab resolve-agent review` / `fab agent operator` run
- **THEN** they resolve exactly as before (the role strings are the compat surface)
- **AND** no `src/kit/skills/*.md` change ships without its `docs/specs/skills/SPEC-*.md` mirror update

#### R12: The effort-asymmetry note rides along
`docs/specs/stage-models.md` SHALL record that effort injected via a subagent-prompt instruction is **not reliably honored** on the native Agent-tool arm (session-level effort dominates — a known Claude Code limitation, GitHub issues #64033/#39220), so effort differentiation is trustworthy only where `--effort` rides a composed CLI command (`fab dispatch` headless/pane, the operator launcher). Model differentiation works on both arms.

- **GIVEN** a reader of `stage-models.md` § Skill wiring
- **WHEN** they read the effort seam description
- **THEN** the reliability caveat and its scope (native arm only) are stated

### Non-Goals

- Codex/gemini per-role fills — deferred to change `ywkx`. Until then `workers: codex|gemini` resolves empty models (the provider CLI's own default applies), an accepted intermediate state.
- Renaming the memory file `runtime/providers-and-tiers.md` — a hydrate-stage decision, not apply's.
- Cascade-aware (per-scope) ownership — the deferred follow-up; the cutoff rule this change deletes was its only consumer, so the limitation and its pinning test go away with it.
- A user-overridable role→depth partition or stage→role mapping — both stay fab-owned.

### Design Decisions

#### The depth knob is the provider rung, not a whole-profile switch
**Decision**: `agent.session`/`agent.workers` supply only the **provider** rung of the chain; model/effort always come from the resolved provider's own per-role fills.
**Why**: It is what makes "claude for tier 1, gemini for tier 2" a two-word config while still letting each provider keep sensible per-role models. A knob that carried a whole profile would have to name models fab cannot ship (they rot at CLI cadence).
**Rejected**: Making the knob a `{provider, model, effort}` object (re-imports the model-rot problem into the advertised surface); keying the partition off stage names (the partition is about *depth*, and `default`/`operator` are not stages).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

#### One fallback chain, on the provider side only
**Decision**: The agent-side `default`-role inheritance is deleted; cross-role fallback exists only as `providers.<p>.profiles.default`.
**Why**: Two competing fallback chains are what forced the provider-name cutoff rule into existence — a rule that only ever existed to stop one chain leaking another provider's values into the other. With a single provider-anchored chain there is nothing foreign to cut, so both the rule and its per-field ownership machinery disappear.
**Rejected**: Keeping agent-side inheritance and retaining the cutoff (preserves the footgun and its documentation debt); erroring when a role resolves an empty model (an empty model is the established inherit-the-session-model signal).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

#### Read-time aliases back the rename, not just the migration
**Decision**: `agent.tiers` and the flat `providers.<p>.model`/`.effort` stay readable as deprecated aliases in `internal/config`, below their new counterparts in the chain; the migration performs the on-disk rewrite.
**Why**: The registry's `renamed_from` carry-forward is a **top-level-key** operation in `configupgrade` — it deliberately skips same-top-level renames like `agent.tiers` → `agent.profiles` — so `renamed_from` alone is metadata, not a working carry. Without a read-time alias, `fab config upgrade` (auto-run by `fab upgrade-repo`) would leave a live `agent:` block above the fence that silently stopped being read until the user ran `/fab-setup migrations`. The alias closes that window; the migration still removes the legacy spelling.
**Rejected**: Teaching `configupgrade` nested rename carry (a splice-engine change well beyond this scope, and the migration already does it correctly); relying on the migration alone (a silent behavior-change window for every user between upgrade and migration run).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

#### The machinery is demoted from the fence, not from the reference
**Decision**: `agent.profiles` and `providers` become `advertise: false` (so `fab config upgrade` stops scaffolding ~90 lines into every project) but keep their registry rows, their `--json` defaults, and — for `providers` — a short rendered segment in `fab config reference`.
**Why**: The fence is per-project noise; the reference is the canonical schema surface and must stay complete. Demotion targets the noise without hollowing out `fab config reference`.
**Rejected**: Dropping the segments entirely (the reference header promises every key is documented, and `fab config init --system` renders from the same segments); keeping `advertise: true` (leaves the fence bloated, which is the change's explicit size requirement).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### Deprecated Requirements

#### Cross-provider field cutoff (`cutForeignFields`, per-field ownership)
**Reason**: It existed only to stop the agent-side `default`-role chain leaking one provider's model into another provider's role. Removing that chain removes the leak.
**Migration**: None needed — an all-claude config resolves identically. A config that relied on the cutoff to *blank* a value now resolves that value from the named provider's own fills (or empty), which is the same outcome the cutoff produced.

#### Cross-scope ownership limitation and `TestResolveCrossScopeCascadeLimitation`
**Reason**: The limitation was a property of the cutoff's per-field ownership computation, which no longer exists.
**Migration**: N/A — the pinning test is deleted along with the behavior it pinned.

#### Flat provider fill `providers.<name>.model` / `.effort` (documented surface)
**Reason**: One fill per provider resolves the same model for every role, defeating role differentiation on provider swap.
**Migration**: Folded into `providers.<name>.profiles.default` by `2.16.19-to-2.17.0.md`; read-time alias retained for the pre-migration window.

## Tasks

### Phase 1: Schema & defaults

- [x] T001 Reshape the config schema in `src/go/fab/internal/config/config.go`: rename `TierProfile` → `RoleProfile`; add `ProviderProfile{Model, Effort}`; add `ProviderConfig.Profiles map[string]ProviderProfile` and mark `Model`/`Effort` deprecated aliases; reshape `AgentConfig` to `{Session, Workers string; Profiles map[string]RoleProfile; Tiers map[string]RoleProfile /*legacy*/}`; add accessors `GetAgentProfile(role)`, `GetAgentSession()`, `GetAgentWorkers()` (nil-safe, legacy-aware) replacing `GetAgentTier`. <!-- R2 R3 R4 -->
- [x] T002 [P] Extend `src/go/fab/internal/config/config_test.go` for the new schema: `profiles` parsing, the per-role `tiers` legacy fallback, the provider `profiles` map, the flat-fill legacy alias, and the two knob accessors. <!-- R2 R3 R4 -->
- [x] T003 Reshape `src/go/fab/internal/agent/defaults.yaml` to `agent.session/workers: claude` plus the six claude role models under `providers.claude.profiles`, and rewrite its bump-procedure comment to name the new drift guards. <!-- R8 -->

### Phase 2: Resolution core

- [x] T004 Rewrite `src/go/fab/internal/agent/agent.go`: role constants (`RoleDefault`…`RoleFast`), the fixed `roleDepth` partition and `stageRoles` map, `RoleNames`/`IsRoleName`/`RoleForStage`/`StageNames`/`DefaultProfile`, `ResolveRole`/`ResolveRoleWith`/`Resolve`, per-role provider fill lookup in `ResolveProvider` (deep-merging `profiles`), and deletion of `cutForeignFields` + the ownership tracking + the old `ApplyOverrides`. <!-- R1 R2 R5 R6 -->
- [x] T005 Update `src/go/fab/internal/agent/agent_test.go` and `defaults_test.go` to the role/profile vocabulary and the new chain: knob-driven provider selection, per-role provider fills, no default-role inheritance, override behavior, and byte-identical shipped defaults; delete the cutoff tests. <!-- R1 R2 R5 R6 R8 -->

### Phase 3: CLI & registry wiring

- [x] T006 Update `src/go/fab/cmd/fab/resolve_agent.go` to resolve a stage-or-role name to a role and call `agent.ResolveRoleWith`, keeping the output contract and the `dispatchLineFor` rules unchanged. <!-- R6 -->
- [x] T007 [P] Update the remaining Go call sites for the rename — `src/go/fab/cmd/fab/agent.go`, `batch.go`, `operator.go`, `dispatch_start.go`, and `src/go/fab/internal/spawn/spawn.go`. <!-- R7 R11 -->
- [x] T008 <!-- rework cycle 2: (1) defaultSubtree first-match bug — cmd/fab/config.go:283 returns on FIRST agent.* row so `config show --origin` drops agent.workers entirely; same bug in configupgrade.go:779-795 twin defaultSubtreeFor (breaks presence=intent B-hygiene for a both-knobs agent: block); MERGE all matching rows' subtrees in both copies + add config_show_init_test.go assertion for `agent.workers = claude # default`; (2) roleRow.Referents is write-only (configref.go:287,311) with a fail-loud gate that can brick five config commands for a table nothing renders — delete field+gate (keep DefaultProfile invariant) or restore referent lines; fix comments :281,:296,:336; (3) role→depth partition re-encoded at configref.go:704-709 — export a depth query from internal/agent or add a drift assertion; (4) stale renamed_from-is-empty claims at configref.go:158-160 + configupgrade.go:460-461 --> Update `src/go/fab/internal/configref/configref.go`: add the `agent.session`/`agent.workers` rows with the single short `agent:` segment, demote `agent.profiles` (with `renamed_from: agent.tiers`) and `providers` to `advertise: false`, project the per-role provider fills into the providers JSON default, and refresh the reference header. <!-- R9 -->
- [x] T009 <!-- rework: stale refs to deleted symbols in test comments (agent.DefaultTier/defaultTiers/TestDefaultTierProfilesArePinned at resolve_agent_test.go:30-33, dispatch_start_test.go:181 — A-014 grep fails); stale Test*Tier* function names + half-updated comments (agent_test.go:42,59,128,162,254,282,284; resolve_agent_test.go:110; batch_new_test.go:337; operator_test.go:153); document+pin the cross-scope legacy-alias precedence sibling (config.go:551-560) --> Update the affected Go tests in `src/go/fab/cmd/fab` (`resolve_agent_test.go`, `agent_test.go`, `operator_test.go`, `batch_new_test.go`, `config_test.go`, `config_show_init_test.go`, `config_upgrade_test.go`) and `src/go/fab/internal/spawn/spawn_test.go`; delete `TestResolveCrossScopeCascadeLimitation`. <!-- R5 R6 R7 R9 -->

### Phase 4: Migration, docs & SPEC mirrors

- [x] T010 <!-- rework cycle 2: both-keys-present bullet (migration :92-99) is self-contradictory and its profiles_from_tiers arm is a silent data-loss path (writes a key no fab code reads, dropping the legacy per-role fallback) — delete that clause, keep ONLY warn-and-leave-as-is --> Write `src/kit/migrations/2.16.19-to-2.17.0.md` — sentinel-guarded, idempotent rewrite of `agent.tiers:` → `agent.profiles:` and the flat provider fill → `providers.<p>.profiles.default`, sweeping both the project config and `~/.fab-kit/config.yaml`. <!-- R10 -->
- [x] T011 Rewrite `docs/specs/stage-models.md` to the roles/profiles/knobs model (including the § Default role profiles and § stage→role tables the drift guards parse) and add the § Effort asymmetry note. <!-- R11 R12 -->
- [x] T012 Update the drift guards in `src/go/fab/internal/agent/stagemodels_doc_test.go` and `defaulttiers_mirrors_test.go` to the new headings, vocabulary, and `_cli-fab.md` enumeration shape. <!-- R11 -->
- [x] T013 [P] Update `src/kit/skills/_cli-fab.md` (§ fab resolve-agent, § fab agent, § fab config, § fab dispatch) and `src/kit/skills/_cli-agents.md` for the new schema and vocabulary. <!-- R11 -->
- [x] T014 [P] Update `src/kit/skills/_preamble.md` § Per-Stage Model Resolution + § CLI-Adapter Dispatch for the role/profile vocabulary and the knobs, keeping the Tier 1/Tier 2 depth usage. <!-- R11 -->
- [x] T015 [P] Update `docs/specs/config.md` (registry table, scope/advertise sections, `--json` shape), `docs/specs/architecture.md`, and `docs/specs/glossary.md`. <!-- R9 R11 -->
- [x] T016 <!-- rework cycle 2: vocabulary sweep incomplete — SPEC-_preamble.md:11 (mis-resolved tier / resolved tier's provider / config-tier override), SPEC-_cli-fab.md:28 (resolved tier's provider / tier-to-provider re-resolution / config-tier override), harness-adapters.md:37,449,455,476 (stage-to-tier chain, resolved tier, config/tier override) all retain retired 'tier' where sources now say role/provider/config-override; grep 'resolved tier', 'config/tier', 'tier→provider', 'stage → tier' across docs/specs --> Update the SPEC mirrors for every touched skill — `docs/specs/skills/SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`, `SPEC-_preamble.md`, and any other mirror carrying tier prose. <!-- R11 -->

### Phase 5: Verification

- [x] T017 Run the affected Go packages' tests (`internal/config`, `internal/agent`, `internal/configref`, `internal/configupgrade`, `internal/spawn`, `cmd/fab`), then the full `go test ./...`, and sanity-check `fab config reference` / `fab config reference --json` / `fab resolve-agent` output. <!-- R6 R8 R9 -->
- [x] T018 <!-- plan revision (rework cycle 3): T016's enumerative sweep definition twice left stragglers one hop out; replaced with a mechanical closure --> Closure-based retired-vocabulary sweep: (1) apply the cycle-3 review's enumerated fixes in `src/kit/skills/fab-proceed.md` (:145,:152,:161,:169), `src/kit/skills/fab-continue.md` (:34,:84), `docs/specs/skills/SPEC-fab-proceed.md` (:10), `docs/specs/skills/SPEC-fab-continue.md` (:12,:16) — including the stale `fab agent <tier>` CLI signature (binary now says `agent [role]`); (2) CLOSURE: for EVERY file touched by this change (`git diff --name-only` + `git status --porcelain`) AND every `docs/specs/skills/SPEC-*.md` mirror of a touched skill (touched or not), grep case-insensitive `tier` and rewrite each hit to role/profile/depth vocabulary UNLESS it is a legitimate use: Tier 1/Tier 2 depth, two-tier tmux hierarchy, `agent.tiers` legacy-alias/migration/renamed_from documentation, `_review.md` three-tier severity, memory-index/exit/loss tiers, or an explicitly historical note; (3) verify with the reviewer's grep — `grep -rniE 'tier' src/kit/skills/fab-proceed.md src/kit/skills/fab-continue.md docs/specs/skills/SPEC-fab-proceed.md docs/specs/skills/SPEC-fab-continue.md | grep -viE 'tier 1|tier 2|two-tier|failed tier|non-empty'` returns zero rows — then run the same closure grep over the full touched-file set and confirm every remaining `tier` is on the allowlist. <!-- R11 -->

## Execution Order

- T001 blocks T004 (the resolver consumes the new schema types).
- T003 blocks T004/T005 (the embedded defaults define the built-in fills).
- T004 blocks T006–T009 (every call site depends on the renamed API).
- T008 blocks T009's `config_test.go` / `config_upgrade_test.go` updates.
- T011 blocks T012 (the guards parse the rewritten headings).
- T013 blocks T012's `_cli-fab.md` enumeration assertion.
- T017 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `agent.session` and `agent.workers` are parsed, default to `claude`, are scope `both`, and select the provider for their partition's roles (`default`/`operator` vs `doing`/`review`/`hydrate`/`fast`).
- [x] A-002 R2: `agent.profiles.<role>.{provider,model,effort}` overrides resolve per field, and `agent.profiles.default` no longer re-bases any other role.
- [x] A-003 R3: A config carrying only `agent.tiers:` still resolves, and `agent.profiles` carries `renamed_from: agent.tiers` in `fab config reference --json`.
- [x] A-004 R4: `providers.<name>.profiles.<role>.{model,effort}` resolves, per-field merges over the built-in table, and `providers.<name>.profiles.default` acts as the provider's cross-role fill.
- [x] A-005 R6: `fab resolve-agent <stage|role> [--alias] [--provider|--model|--effort]` keeps its argument grammar, output lines, `--alias` semantics, `dispatch=` emission rules, and unknown-provider lookup error.
- [x] A-006 R7: `fab agent`, `fab batch new`/`switch`, and the `fab operator` launcher compose their session command from the role profile resolved through the new chain, so `agent.session` governs Tier-1 launches.
- [x] A-007 R9: `agent.session`/`agent.workers` are registered `advertise: true`; `agent.profiles` and `providers` are `advertise: false` yet still present in `fab config reference` and `--json`; the rendered fence's agent/provider section is at most ~20 lines. *(Verified: rendered fence agent section = 20 lines, no `providers:` scaffold.)*
- [x] A-008 R10: `src/kit/migrations/2.16.19-to-2.17.0.md` exists with Summary / Pre-check / Changes / Verification sections, covers both config scopes, and is sentinel-guarded. *(Slot re-numbered from `2.16.18-to-2.17.0` per Assumption 10.)*
- [x] A-009 R12: `docs/specs/stage-models.md` states the native-arm effort-asymmetry caveat with its scope and issue references.

### Behavioral Correctness

- [x] A-010 R8: Every stage's resolved `{provider, model, effort}` is byte-identical to the pre-change values under an empty project config. *(Verified against the POST-rebase baseline — `doing`/`review` are `high`, not `xhigh`, since v2.16.19/#537 merged mid-apply; see the Apply note.)*
- [x] A-011 R5: The provider-name cutoff is gone — no `cutForeignFields`, no ownership tracking, no cross-scope limitation test — and a role whose provider is swapped refills from that provider's own fills, then empty. *(Verified empirically: `agent.workers: codex` → `model=` empty; `--provider claude` → `model=claude-opus-5`.)*
- [x] A-012 R2: With `agent.profiles.default.model` set and `agent.profiles.doing` omitting `model`, the `doing` role resolves the provider's `doing` fill, not the `default` role's model.
- [x] A-013 R11: The six role strings are unchanged on the CLI — `fab resolve-agent review`, `fab resolve-agent fast`, and `fab agent operator` all still resolve. *(All six roles + all six stages verified against the built binary.)*

### Removal Verification

- [x] A-014 R5: `grep -rn "cutForeignFields\|TierDefault\|ResolveTier\|defaultTiers\|stageTiers\|GetAgentTier" src/go` returns no hits outside deliberate legacy-alias documentation. *(Rework cycle 1: the last stale references — `agent.DefaultTier`/`defaultTiers`/`TestDefaultTierProfilesArePinned` in `resolve_agent_test.go` and `dispatch_start_test.go` comments — are corrected; the grep, extended with `DefaultTier` and `TestDefaultTierProfilesArePinned`, now returns zero hits across `src/go`, `src/kit`, and `docs/specs`.)*
- [x] A-015 R4: No code path reads the flat `providers.<name>.model`/`.effort` as the documented fill — only as the documented deprecated alias for `profiles.default`. *(Third rung of `providerFill`, below `profiles.default`; omitted from every registry `default`.)*

### Scenario Coverage

- [x] A-016 R1: A test asserts `agent.workers: <name>` moves only the four workers roles and leaves `default`/`operator` on the session knob. *(`TestDepthKnobSelectsProvider`, `TestRoleDepthPartition`.)*
- [x] A-017 R3: A test asserts the per-role legacy `tiers` fallback and that `profiles` wins when both are present. *(`TestLegacyAgentTiersAlias`, `TestGetAgentProfile_LegacyTiersAlias`.)*
- [x] A-018 R6: A test asserts a `--provider` swap refills model/effort from the new provider's per-role fills and re-derives the `dispatch=` line. *(`TestResolveAgentOverrideProviderTakesFill`, `…NoFill`, `…DispatchDisappearsOnNativeSwap`.)*
- [x] A-019 R9: A test asserts the fence contains the two knobs and contains no `providers:`/`profiles:` scaffold, and that `fab config reference` round-trips (one `agent:` key). *(`TestRender_FenceDemotesAgentMachinery` + the `config_test.go` reference round-trip.)*

### Edge Cases & Error Handling

- [x] A-020 R6: An unknown role/stage name still exits non-zero naming the valid set; an unknown `--provider` still raises the shared lookup error.
- [x] A-021 R4: A provider with no `profiles` and no legacy fill resolves an empty model/effort (placeholder tokens dropped), never another provider's values. *(Verified: `dispatch=codex exec` with both tokens dropped and no `effort=` line.)*
- [x] A-022 R8: A malformed `defaults.yaml` still panics at package init, and `defaults_test.go` asserts the reshaped file covers every role and provider.

### Code Quality

- [x] A-023 Pattern consistency: New code follows the package's naming, doc-comment, and nil-safe-accessor patterns.
- [x] A-024 No unnecessary duplication: Resolution stays implemented once in `internal/agent`; `cmd/fab` reimplements no rung of the precedence chain. *(`configref` derives from `DefaultProfile`/`ResolveProvider`; no second built-in table.)*
- [x] A-025 Canonical source only: No edit lands under `.claude/skills/` — every kit change is in `src/kit/skills/`.
- [x] A-026 SPEC-mirror sync: Every touched `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in the same change. *(Verified at review cycle 3 re-review. The closure sweep (T018) is complete: the reviewer's grep over `fab-proceed.md` / `fab-continue.md` / `SPEC-fab-proceed.md` / `SPEC-fab-continue.md` returns zero rows, and the stale `fab agent <tier>` signature now reads `fab agent <role>`, matching the binary's `Use:` string (`agent [role]`, confirmed against the built binary). The closure grep over the full touched-file set ∪ every SPEC mirror of a touched skill leaves only allowlisted hits: Tier-1/Tier-2 depth, the two-tier tmux hierarchy, `agent.tiers` legacy-alias/migration prose, operator Confirmation Tiers, memory-index/DisplayStage-failed drift tiers, `docs/findings/per-stage-model-tier-application.md` as a filename, and `config_test.go:290`'s deliberate negative assertion. The three touched skills whose mirrors are unmodified — `_pipeline.md`, `fab-ff.md`, `fab-fff.md` — owe no mirror edit: each source change is a single `mis-resolved tier`→`role` swap, and all three mirrors contain zero `tier` occurrences and phrase the same seam as "a skip is detectable". `go test ./...` green.)*
- [x] A-027 CLI ⇒ docs + tests: The `fab resolve-agent` / `fab config` surface changes are reflected in `src/kit/skills/_cli-fab.md` and carry test updates.
- [x] A-028 Migration discipline: The user-data restructure ships as `src/kit/migrations/2.16.19-to-2.17.0.md`, not an ad-hoc script or subcommand. *(Slot re-numbered from `2.16.18-to-2.17.0` per Assumption 10.)*
- [x] A-029 Test-alongside: Every changed `.go` file ships its test updates in this change, and the affected packages pass. *(`go build ./...` clean; `go test ./...` all packages ok.)*

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [ ] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `config.ProviderConfig.Model` / `.Effort` (`src/go/fab/internal/config/config.go:103-104`) plus the third rung of `providerFill` (`internal/agent/agent.go:457-458`) — the deprecated flat provider fill. Deletable one release after `2.16.19-to-2.17.0` has shipped and rewritten user configs; keep for the pre-migration window.
- `config.AgentConfig.Tiers` (`src/go/fab/internal/config/config.go:156`) and its fallback branch in `GetAgentProfile` — the deprecated `agent.tiers` spelling. Same release-lag deletion as the flat fill; `TestLegacyAgentTiersAlias` / `TestGetAgentProfile_LegacyTiersAlias` retire with it, as does `TestResolveCrossScopeLegacyAliasPrecedence` (which pins the alias's one cross-scope precedence inversion — the alias resolves after the cascade, so a system-layer `agent.profiles.<role>` beats a project-layer `agent.tiers.<role>`; documented at `GetAgentProfile`'s LIMITATION note and cured by the migration, which sweeps both scopes).
- `configref.Field.RenamedFrom` carry machinery in `configupgrade.carryRenames` (`internal/configupgrade/configupgrade.go:313-320`) — now provably unreachable: the only row carrying `renamed_from` is a same-top-level rename the carry deliberately skips, so no row can exercise it. Either keep as documented capacity for a future top-level rename (current choice, and it is now documented as such) or delete and drop the field to `--json` metadata only.
- `agent.IsRoleName` (`internal/agent/agent.go:278`) has no caller outside the package any more — `RoleForName` absorbed the positional dispatch that `resolve_agent.go` used to do with `agent.IsTierName`. Could be unexported; kept exported for symmetry with `RoleForStage`/`RoleNames`/`StageNames` and because R11 names the rename explicitly.
- `docs/findings/per-stage-model-tier-application.md` — its Gap-2 "✅ Done (effort-via-prompt)" verdict is superseded by `stage-models.md` § Effort asymmetry. Not a code deletion; a candidate for a superseded-marker or a rewrite during hydrate.
- The mirrored generic-value helper cluster across `src/go/fab/cmd/fab/config.go` and `src/go/fab/internal/configupgrade/configupgrade.go` — `normalizeToGeneric`, `asGenericMap`, `genericEqual`, `defaultSubtree`/`defaultSubtreeFor`, and now `mergeGeneric` (`cmd/fab/config.go:307-330` / `configupgrade.go:806-826`, both added by this change). Five near-verbatim twins, kept apart only because `internal/configupgrade` cannot import the `main` package. Extracting them into a small `internal/genericvalue` package would delete one copy of each; deliberately out of scope here, and the cost of the split just doubled — a merge bug now has to be fixed twice, which is exactly what rework cycle 2 did to `defaultSubtree`/`defaultSubtreeFor`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Read-time deprecated aliases for both `agent.tiers` and the flat provider fill, alongside the migration | The intake's "registry `renamed_from` carry-forward, so old configs keep resolving" does not hold mechanically — `configupgrade` skips same-top-level renames by design — so an alias is what makes the stated intent true; extending it to the flat fill is the same hazard and the same fix | S:70 R:75 A:85 D:65 |
| 2 | Confident | Advertise-demoted rows keep their registry Segments, so `fab config reference` (and `init --system`) still document them; only the per-project fence loses them | The intake says "visible in `fab config reference --json` … not rendered per-project"; the reference header promises full key coverage and `init --system` renders from Segments, so hollowing them out would regress two surfaces the intake never named | S:60 R:85 A:75 D:60 |
| 3 | Confident | The whole `agent:` YAML block is rendered by one segment (owned by the `agent.session` row), with `agent.profiles` as a pointer line inside it | Two segments emitting a live `agent:` parent would produce a duplicate YAML key and break the reference's documented round-trip; this is the existing `project.name` / `dispatch.watchable` block-ownership pattern | S:65 R:80 A:90 D:75 |
| 4 | Certain | Migration slot `2.16.18-to-2.17.0` (minor bump — schema change), with `src/kit/VERSION` left to the release commit | FROM = the real current released VERSION per the chaining precedent; the catalog shows VERSION is bumped by the `release:` commit, not by the change | S:85 R:85 A:95 D:85 |
| 5 | Confident | `ApplyOverrides` is replaced by `ResolveRoleWith(cfg, role, overrides)` rather than kept as a post-hoc mutator | With the cutoff gone, an override is just the top rung of one chain; re-running the chain with the flags on top is strictly simpler than resolving then patching, and a `--provider` swap needs the role anyway to find the new provider's per-role fill | S:65 R:80 A:85 D:70 |
| 6 | Confident | An explicit `agent.profiles.<role>.model` survives a `--provider` swap | It follows directly from the intake's stated chain (the role override sits above the provider fills) and is the honest reading of "no special inheritance-loss rule remains" — an explicit pin is the user's own escape hatch | S:70 R:70 A:75 D:60 |
| 7 | Certain | The role→depth partition and the stage→role mapping stay Go policy (not in `defaults.yaml`) | `defaults.yaml`'s documented contract is "everything here is user-overridable"; both mappings are fab-owned taxonomy, exactly like the existing `stageTiers` | S:85 R:80 A:95 D:85 |
| 8 | Confident | `DefaultProfile(role)` is defined as resolution against a nil config rather than a separate built-in table | There is no longer a single built-in role→profile map to read (the values live under `providers.claude.profiles`), and nil-config resolution is exactly "the built-in answer" — it also keeps `fab config reference` sourcing its defaults from the resolver | S:70 R:80 A:85 D:70 |
| 9 | Confident | `docs/memory/` is left untouched by apply | Constitution II / the six-stage pipeline put memory writes in hydrate; the intake's Affected Memory list is hydrate's input, not apply's | S:80 R:90 A:90 D:80 |
| 10 | Certain | Migration slot re-numbered `2.16.18-to-2.17.0` → **`2.16.19-to-2.17.0`** mid-apply | An upstream release (v2.16.19) merged into the branch while apply was running; the chaining precedent (`2.11.0-to-2.12.0` slot note) fixes FROM at the real current released VERSION | S:90 R:90 A:95 D:90 |
| 11 | Confident | `TestResolveCrossScopeCascadeLimitation` is **reframed and renamed** (`TestResolveCrossScopeRoleProfileMerge`) rather than deleted | The plan's Deprecated Requirements said delete, because the *limitation* it pinned dies with the cutoff. But the bytes it asserts remain correct under the new chain — an explicit `agent.profiles.<role>` pin outranking the provider's fills — so the scenario is now intended behavior worth a regression guard, not a limitation to retire. Deleting would have dropped real cross-scope coverage | S:70 R:85 A:80 D:65 |

11 assumptions (3 certain, 8 confident, 0 tentative).

<!-- Apply note: an upstream commit (6e48c0a4, "Lower doing/review built-in tier
     effort from xhigh to high", #537) plus release v2.16.19 landed on this branch
     mid-apply and git auto-merged them into the in-flight reshape. Their
     xhigh→high change is preserved throughout (defaults.yaml, the pinned table,
     stage-models.md, _cli-fab.md), so the "byte-identical resolved defaults"
     requirement (R8/A-010) holds against the POST-merge baseline, not the
     pre-merge one. -->

