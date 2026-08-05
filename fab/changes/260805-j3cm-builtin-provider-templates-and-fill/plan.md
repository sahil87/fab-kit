# Plan: Built-in Provider Templates + Per-Provider Fill + Invocation Overrides

**Change**: 260805-j3cm-builtin-provider-templates-and-fill
**Intake**: `intake.md`

## Requirements

### Providers: Built-in codex and gemini (grammar only)

#### R1: `defaultProviders` carries three built-in providers
`internal/agent`'s `defaultProviders` table SHALL contain `claude`, `codex`, and `gemini`. The `codex` and `gemini` entries SHALL carry **only** command grammar (`session_command` + `dispatch_command`) with `{model}`/`{effort}` placeholders — and SHALL carry **no** model or effort fill values (model IDs rot at CLI cadence; the binary MUST NOT bake them). The strings SHALL be exactly the previously-shipped starter-template text, exposed as named Go constants alongside `DefaultSessionCommand` so no literal is duplicated between `internal/agent` and `internal/configref`.

- **GIVEN** a project with no `providers:` block at all
- **WHEN** `agent.ResolveProvider(cfg, "codex")` is called
- **THEN** it returns `{SessionCommand: "codex -m {model} -c model_reasoning_effort={effort}", DispatchCommand: "codex exec -m {model} -c model_reasoning_effort={effort}"}` with `ok == true`
- **AND** `agent.ResolveProvider(cfg, "gemini")` returns `{SessionCommand: "gemini -m {model}", DispatchCommand: "gemini -m {model}"}` with `ok == true`
- **AND** `agent.ProviderNames(cfg)` returns `["claude", "codex", "gemini"]`

#### R2: Naming a built-in provider is sufficient — no `providers:` block required
A tier override or an invocation flag that NAMES `codex`/`gemini` SHALL resolve without any `providers:` config. Because the codex/gemini built-ins DO carry a `dispatch_command` (unlike claude, which carries none), naming one in a tier SHALL flip that tier's stages from native Agent-tool dispatch to CLI dispatch — the intended meaning of selecting a non-claude provider.

- **GIVEN** a config with `agent.tiers.review: { provider: codex, model: gpt-5.3-codex, effort: high }` and no `providers:` block
- **WHEN** `fab resolve-agent review` runs
- **THEN** stdout carries `provider=codex` and a `dispatch=codex exec -m gpt-5.3-codex -c model_reasoning_effort=high` line
- **GIVEN** the same config with no `agent.tiers` override
- **WHEN** `fab resolve-agent review` runs
- **THEN** no `dispatch=` line is emitted (the built-in claude provider carries no `dispatch_command`) — a built-in provider is inert until named

#### R3: Reference and scaffold present codex/gemini as built-in defaults
`fab config reference` (and therefore the `fab config upgrade` managed fence) SHALL present the `providers:` block with codex and gemini as **commented reference-style built-in defaults** — the same presentation every other non-overridden default uses — rather than as "uncomment to opt in" starter blocks. The prose SHALL no longer claim "No new built-in providers are added in Go" nor "template text only until you uncomment them". The per-provider notes (codex reads stdin; gemini has no `{effort}` flag and no `-p`) SHALL be preserved verbatim in meaning. The commented blocks SHALL still parse as **absent from `Config`** (presence=intent for *behavior*: a commented block registers no project override), and whole-block uncommenting SHALL still yield valid YAML.

- **GIVEN** `fab config reference` output
- **WHEN** it is parsed back into `Config`
- **THEN** `GetProvider("codex")` and `GetProvider("gemini")` report `!ok` (commented, not live) while `agent.ResolveProvider` still resolves them from the Go built-in table
- **AND** the rendered text no longer contains the phrases "No new built-in providers are added in Go" or "uncomment and adapt a block to add that provider"

### Config: Per-provider default fill

#### R4: `providers.<name>.model` / `providers.<name>.effort`
`config.ProviderConfig` SHALL gain two optional string fields, `Model` (`yaml:"model"`) and `Effort` (`yaml:"effort"`), carrying a provider's **default fill** for the `{model}`/`{effort}` placeholders. They SHALL be per-field merged over the built-in table by `agent.ResolveProvider` exactly as the command fields are. The Go built-in table SHALL supply **no** fill values for codex/gemini (R1); claude's fill continues to live on the built-in tiers, not on the provider. The change SHALL be purely additive — no migration.

- **GIVEN** `providers: { codex: { model: gpt-5.3-codex, effort: high } }` in either the project or the system config
- **WHEN** `agent.ResolveProvider(cfg, "codex")` is called
- **THEN** the resolved `ProviderConfig` carries `Model: "gpt-5.3-codex"`, `Effort: "high"`, and the **built-in** command strings (per-field merge — the config supplied no commands)

#### R5: Registry rows for the fill fields
`internal/configref` SHALL document the two fill fields: the `providers` row's `Description` and rendered `Segment` SHALL explain them, and the row's structured `Default` SHALL keep exposing only the built-in values that actually exist (claude's `session_command`; codex/gemini's command pair — never a fabricated model). The `providers` key's existing scope (`both`) SHALL be unchanged, so a fill value is settable once per machine in `~/.fab-kit/config.yaml`.

- **GIVEN** `fab config reference --json`
- **WHEN** the `providers` row is inspected
- **THEN** its `default` object contains `claude`, `codex`, and `gemini` entries, each carrying only the command fields that exist (no `model`/`effort` keys for any provider), and its `description` names the `model`/`effort` fill fields
- **AND** the JSON/YAML key-parity guard still passes (no new top-level registry key is introduced — `providers` remains one override unit)

#### R6: Fill precedence
The resolved `{model, effort}` for any invocation SHALL be determined by, in order: **invocation flag > explicit tier field > named provider's default fill > empty**. An empty value keeps today's meaning (`spawn.WithProfile` token-drop → the CLI's own default; an empty `model=` line = "inherit the session model").

- **GIVEN** `providers.codex.model: gpt-5.3-codex` and `agent.tiers.doing: { provider: codex, model: gpt-5.2-codex }`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=gpt-5.2-codex` (the explicit tier field beats the provider fill)
- **WHEN** `fab resolve-agent apply --model gpt-5.4-codex` runs
- **THEN** `model=gpt-5.4-codex` (the invocation flag beats both)
- **GIVEN** the same `providers.codex.model` but `agent.tiers.doing: { provider: codex }` (no model)
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=gpt-5.3-codex` (the provider fill supplies it)

#### R7: Cross-provider inheritance fix
When a tier's resolved provider comes from an **explicit config `provider:`** setting (on the requested tier's own override or on the project's `default` tier) and that provider differs from the provider the built-in tier profile named, every resolved `model`/`effort` value **owned by a different provider** SHALL fill from **the resolved provider's default fill, then empty** — never from the built-in tier's (or another provider's) model/effort. A value's **owner** is the provider in effect at the layer that supplied it: the layer's own `provider:` when it names one, otherwise the provider the layers below it resolved. So a `model` written on the project `default` tier with no `provider:` beside it is owned by the built-in's provider and does NOT survive a cross-provider switch made on the requested tier, while a `model` written at (or above) that switch is owned by the new provider and does. A tier that does not set `provider:` SHALL inherit exactly as today, and a chain whose **net** provider equals the built-in's is not a switch at all (plan Assumption 2). The all-claude default world SHALL be byte-unchanged (every built-in tier pins an explicit model).

**Scope: ownership is computed over the MERGED config** (revised, rework cycle 3). The system↔project cascade (`config.LoadPath`) deep-merges `agent.tiers` per-key BEFORE resolution, so per-SCOPE ownership is not tracked: when the system scope and the project scope both contribute to one tier and name **different** providers, the merged layer's values are attributed to the merging layer's named provider, and the cutoff does not fire across that scope boundary (e.g. a system-scope codex model surviving a project-scope `provider: gemini` switch). This is a **documented limitation**, NOT a supported behavior: it SHALL be (a) stated in `stage-models.md` and in `internal/agent`'s ownership doc comment — which SHALL NOT overclaim (no "never from an intermediate layer's" phrasing covering scopes it cannot see), and (b) pinned by a test that reproduces the cross-scope scenario and asserts the current behavior with a comment marking it the documented limitation. Cascade-aware ownership (folding per-scope layers in `ResolveTier`) is deferred to a follow-up change.

- **GIVEN** `agent.tiers.doing: { provider: codex }` with no `providers.codex.model`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=` is empty (NOT `claude-opus-5`) and no `effort=` line is emitted — the claude-shaped built-in values are not inherited across the provider switch
- **GIVEN** `agent.tiers.default: { model: claude-fable-5, effort: medium }` + `agent.tiers.doing: { provider: codex }` + `providers.codex: { model: gpt-5.3-codex, effort: high }`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=gpt-5.3-codex` and `effort=high` — the `default`-tier values are claude-owned and do not survive the switch
- **GIVEN** `agent.tiers.doing: { effort: high }` (no `provider:`)
- **WHEN** `fab resolve-agent apply` runs
- **THEN** provider and model inherit exactly as today (`provider=claude`, `model=claude-opus-5`) with `effort=high` winning
- **GIVEN** a config with no overrides at all
- **WHEN** every stage is resolved
- **THEN** the output is byte-identical to the pre-change output for all six stages

### CLI: Invocation-time overrides on `fab resolve-agent`

#### R8: `--provider` / `--model` / `--effort` flags
`fab resolve-agent <stage|tier>` SHALL accept `--provider <name>`, `--model <id>`, and `--effort <level>`. `--provider` SHALL swap the resolved profile's provider and re-derive the `dispatch=` line from the **named** provider's `dispatch_command`; a provider swap SHALL NOT retain the tier's explicit model/effort (it fills from the new provider's default fill, then empty — the same rule as R7). `--model`/`--effort` SHALL override the corresponding field within the otherwise-resolved profile and are valid **without** `--provider` (unlike `fab agent`, where `--model`/`--effort` require `--provider`; the asymmetry — pure query vs session launcher — SHALL be documented). An unknown `--provider` name SHALL be a non-zero-exit **lookup** failure listing the resolvable names, mirroring `fab agent`'s error. Overrides SHALL apply with no validation (provider neutrality).

- **GIVEN** a default config (apply → doing → claude/claude-opus-5/xhigh)
- **WHEN** `fab resolve-agent apply --provider codex` runs
- **THEN** `provider=codex`, `model=` is empty (no codex fill configured), no `effort=` line, and `dispatch=codex exec` appears with both placeholder tokens dropped
- **WHEN** `fab resolve-agent apply --provider codex --model gpt-5.3-codex --effort high` runs
- **THEN** `model=gpt-5.3-codex`, `effort=high`, `provider=codex`, `dispatch=codex exec -m gpt-5.3-codex -c model_reasoning_effort=high`
- **WHEN** `fab resolve-agent apply --effort high` runs (no `--provider`)
- **THEN** `provider=claude`, `model=claude-opus-5` (the tier's), `effort=high` — a within-tier override, not a usage error
- **WHEN** `fab resolve-agent apply --provider bogus` runs
- **THEN** it exits non-zero naming `bogus` and listing `claude, codex, gemini`, and prints no profile

#### R9: `--alias` interplay is unchanged
`--alias` SHALL keep its best-effort semantics under overrides: a non-Claude overridden model passes through verbatim on the `model=` line, and the `dispatch=` line SHALL always embed the FULL model ID.

- **GIVEN** `fab resolve-agent apply --provider codex --model gpt-5.3-codex --alias`
- **WHEN** it runs
- **THEN** `model=gpt-5.3-codex` (verbatim — no Claude prefix matched) and `dispatch=` embeds the same full ID

### Docs: Skills, specs, and the sweep class

#### R10: `_cli-fab.md` documents the new surface
`src/kit/skills/_cli-fab.md` § fab resolve-agent SHALL document the three override flags, the fill precedence chain, the cross-provider fill rule, the unknown-provider lookup error, and the documented asymmetry with `fab agent`. Its § fab config / providers prose SHALL name the three built-in providers and the per-provider fill fields.

- **GIVEN** the edited `_cli-fab.md`
- **WHEN** § fab resolve-agent is read
- **THEN** the usage line reads `fab resolve-agent <stage|tier> [--alias] [--provider <name>] [--model <id>] [--effort <level>]` and the precedence chain is stated

#### R11: `_preamble.md` gains one passthrough paragraph
`src/kit/skills/_preamble.md` § Per-Stage Model Resolution SHALL gain exactly one paragraph: when the user directs a provider/model for specific stages, the dispatch site passes the override flags on its **existing single** `resolve-agent` call. The seam rules, the branch-on-`dispatch=` rule, and compliance visibility SHALL be unchanged, and no new dispatch machinery SHALL be introduced. Because `fab dispatch start` carries **no override surface** (it re-resolves the stage from config), the paragraph SHALL scope invocation-time overrides to the **native Agent-tool arm** and SHALL state that relocating a stage between the native and CLI adapters requires a config/tier override, not an invocation flag. Every other doc restating the override surface (`stage-models.md`, `harness-adapters.md`, `_cli-fab.md`, and their SPEC mirrors) SHALL carry the same scope.

- **GIVEN** the edited `_preamble.md`
- **WHEN** § Per-Stage Model Resolution is read
- **THEN** it states that override flags ride the existing single `fab resolve-agent <stage> --alias` call, that they bind the native Agent-tool arm only, and that an override-only `dispatch=` line is not actionable because `fab dispatch start` re-resolves from config
- **AND** the paragraph is still exactly one paragraph

#### R12: `_cli-agents.md` drops uncomment-first phrasing
`src/kit/skills/_cli-agents.md` SHALL state that codex/gemini are **built-in providers** resolvable by name, with volatile model IDs filled via `providers.<name>.model` (or `--model`), replacing the "commented until a user opts in / Go built-in provider table stays claude-only" phrasing.

- **GIVEN** the edited `_cli-agents.md`
- **WHEN** its provider-dictionary preamble is read
- **THEN** it names the three built-in providers and points at the fill field, and no sentence claims the Go table is claude-only

#### R13: SPEC mirrors and design specs stay in sync
Every edited `src/kit/skills/*.md` file SHALL carry its `docs/specs/skills/SPEC-*.md` mirror update in this change (constitution Additional Constraints). `docs/specs/stage-models.md` SHALL record the three built-ins, the fill fields, the fill precedence, the cross-provider rule, and the resolve-agent override flags; `docs/specs/config.md` SHALL record the fill fields on the `providers` registry row. Aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) and `docs/specs/harness-adapters.md` SHALL be swept for any restatement of the one-built-in-provider or uncomment-to-opt-in model.

- **GIVEN** a repo-wide grep for the phrases "claude-only", "starter template", "uncomment", and "built-in provider" under `docs/specs/` and `src/kit/`
- **WHEN** the change is complete
- **THEN** every hit either reflects the new three-built-in model or is unrelated to the provider table

#### R14: Go changes ship tests
Every Go change SHALL ship accompanying tests in the same change (constitution VII, code-review.md § Go changes ship tests): the built-in provider table, the fill-precedence table, the cross-provider fill rule, the override flags (including the unknown-provider error and the `--alias` interplay), and the reference/fence text guards.

- **GIVEN** `go test ./...` in `src/go/fab`
- **WHEN** it runs after the change
- **THEN** it passes, and the new behaviors each have at least one dedicated test case

### Non-Goals

- No baked model IDs in Go for codex/gemini — built-ins are grammar-only.
- No semantic provider branching in Go (`if codex then …`) — provider names stay opaque.
- No named tier-profile sets (`agent.profiles.*`).
- No validation that a named provider's binary exists (document-don't-validate stands).
- No change to the fixed stage→tier mapping, the dispatch five-state machine, or the pane adapter.
- No relaxation of `fab agent`'s requires-`--provider` rule for `--model`/`--effort`.
- No migration — the fill fields are purely additive config keys.

### Design Decisions

#### Grammar in Go, Fill Values in Config
**Decision**: Promote the codex/gemini invocation *templates* to Go built-in providers (`defaultProviders`) while keeping their `{model}`/`{effort}` fill values in user config (`providers.<name>.model`/`.effort`, `agent.tiers`, invocation flags).
**Why**: Invocation grammar changes at binary-release cadence and is safe to ship; model IDs rot in weeks and would make every fab release carry stale strings. Splitting them lets a fresh project name `codex` with zero config while the volatile half is settable once per machine via the system config layer.
**Rejected**: Baking model IDs into Go (rots); leaving codex/gemini as commented template text (the ho9y state — keeps cross-provider work config-gated at exactly the frictionless moment); inferring a provider from a model string (breaks provider neutrality).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

#### Explicit `provider:` Cuts Off Cross-Provider Field Inheritance
**Decision**: When a tier's provider comes from an explicit config `provider:` that differs from the built-in tier profile's provider, every `model`/`effort` **owned by another provider** fills from the resolved provider's default fill and then empty — not from default-tier/built-in-tier inheritance. A value's *owner* is the provider in effect at the layer that supplied it, so a `default`-tier model written with no provider beside it is claude-owned and does not survive a codex switch on the requested tier, while a model written at or above that switch does. The same rule governs a `--provider` swap on `fab resolve-agent` — both paths call one `cutForeignFields` helper, differing only in how they record owners (config-layer fold vs. flag layer).
**Why**: Per-field inheritance across a provider switch supplies another provider's model, which is the documented footgun fab previously chose to document rather than fix — there was no correct value to fill with. The per-provider fill slot now supplies one, so the footgun can be closed instead of documented. The all-claude default world is unaffected because every built-in tier pins an explicit model.
**Rejected**: Keeping the footgun documented-only (a correct fill value now exists); erroring on a cross-provider tier with no model (a resolvable empty model is the existing "inherit / CLI default" signal and is strictly more useful); validating provider/model compatibility (breaks provider neutrality).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

#### Overrides Land on `resolve-agent`, Not New Dispatch Machinery
**Decision**: Invocation-time provider/model/effort overrides are flags on `fab resolve-agent` — the single resolution call every dispatch site already makes — and the skill wiring is one passthrough paragraph in `_preamble.md`.
**Why**: Every dispatch site already makes exactly one `resolve-agent --alias` call and already branches on `dispatch=`; overriding at that call reuses the whole seam (including the native-vs-CLI branch) with zero new machinery. A separate override channel would need its own precedence rules and its own compliance-visibility contract.
**Rejected**: Per-stage override config keys (persistent state for a per-run intent); a new `fab dispatch --provider` flag (duplicates resolution and skips the native/CLI branch); named tier-profile sets (deferred — per-stage flags cover "use codex for the next N stages").
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

#### `resolve-agent` Allows Bare `--model`/`--effort`; `fab agent` Still Requires `--provider`
**Decision**: On `fab resolve-agent`, `--model`/`--effort` are valid without `--provider` (a within-tier override). On `fab agent` the shipped requires-`--provider` rule stands. The asymmetry is documented in `_cli-fab.md`.
**Why**: `resolve-agent` is a pure query whose whole output is a profile — overriding one field of the profile it would otherwise print is meaningful and unambiguous. `fab agent` is a session launcher with two mutually exclusive addressing modes, where a bare `--model` would either invent an undocumented tier-override surface or be silently ignored (the nvad reasoning, unchanged).
**Rejected**: Relaxing `fab agent` to match (re-litigates a deliberate shipped decision); forbidding bare `--model` on `resolve-agent` to force symmetry (a usage error for an unambiguous, useful query).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

#### The ho9y "No New Built-in Providers" Decision Is Explicitly Reversed
**Decision**: This change reverses ho9y's "No new built-in providers are added in Go — codex/gemini are template text only." The reversal is narrow (grammar strings only) and recorded as a Design Decision in both the plan and memory rather than applied silently.
**Why**: ho9y's reasoning was that presence implies intent, so anything whose presence changes behavior must ship commented. That reasoning still holds — and it is *preserved*, because a built-in provider is inert until a tier or a flag names it, so adding the row changes no default behavior. What ho9y additionally assumed (that a built-in row would need model fill values) is what the per-provider fill slot removes.
**Rejected**: Keeping the ho9y state (leaves cross-provider work config-gated); reversing it silently (a reader of the config fence would see the old "template text only" claim contradicted with no record).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

## Tasks

### Phase 1: Go — built-in providers and config fields

- [x] T001 Add `DefaultCodexSessionCommand`, `DefaultCodexDispatchCommand`, `DefaultGeminiSessionCommand`, `DefaultGeminiDispatchCommand` constants and the `codex`/`gemini` rows to `defaultProviders` in `src/go/fab/internal/agent/agent.go` (grammar only — no fill values), updating the package/table doc comments. <!-- R1 -->
- [x] T002 [P] Add `Model` / `Effort` fields to `config.ProviderConfig` in `src/go/fab/internal/config/config.go` with doc comments naming them the per-provider default fill and the fill-precedence position. <!-- R4 -->

### Phase 2: Go — resolution precedence

- [x] T003 Extend `agent.ResolveProvider` in `src/go/fab/internal/agent/agent.go` to per-field merge the new `Model`/`Effort` fields alongside the command fields. <!-- R4 -->
- [x] T004 Implement the cross-provider fill rule in `agent.ResolveTier` (`src/go/fab/internal/agent/agent.go`): track whether the resolved provider came from an explicit config `provider:` differing from the built-in tier's provider; when so, drop the inherited model/effort and fill from `ResolveProvider(cfg, provider).Model/.Effort`, then empty. <!-- R7 --> <!-- rework cycle 2: review must-fix — ResolveTier flattens the project `default` tier and the requested tier into one `configured Profile` (agent.go:303-313), so a `default`-tier model/effort reads as "explicitly set" at the cutoff (agent.go:324) and SURVIVES a provider switch: `default: {model: claude-fable-5, effort: medium}` + `doing: {provider: codex}` + `providers.codex: {model: gpt-5.3-codex, effort: high}` resolves model=claude-fable-5 (a claude ID handed to the codex CLI) instead of the codex fill — violates R7's "never from the built-in tier's (or another provider's) model/effort". FIX: anchor the cutoff to the provider that supplied each inherited value rather than to builtin.Provider over a flattened profile; note ApplyOverrides (agent.go:382) already compares `o.Provider != p.Provider` correctly — collapse the two implementations of the one rule into one where feasible. -->
- [x] T005 Add an exported `agent.ApplyOverrides` (or equivalently-named) helper in `src/go/fab/internal/agent/agent.go` implementing the R6 precedence for an invocation-time `{provider, model, effort}` override set, including the provider-swap fill rule, so `cmd/fab` does not reimplement precedence. <!-- R6, R8 -->
- [x] T006 Tests for T001–T005 in `src/go/fab/internal/agent/agent_test.go`: built-in table contents + `ProviderNames`, fill-field merge, the full fill-precedence table, the cross-provider fill rule (both directions), and the all-claude byte-unchanged case. <!-- R14 --> <!-- rework cycle 2: review should-fix — TestResolveCrossProviderCutoff (agent_test.go:226-241) covers only the requested-tier-sets-provider variant; add the `default`-tier-supplies-model/effort variants (the exact broken combination from the T004 must-fix): default-tier model + requested-tier cross-provider switch (cutoff MUST fire, provider fill wins), and default-tier provider switch with built-in tier model (cutoff MUST fire per plan Assumption 2). -->
- [x] T007 [P] Tests for the new config fields in `src/go/fab/internal/config/config_test.go`: parse `providers.<name>.model/effort` from the project layer and from the system layer (cascade), and the nil-safe/absent cases. <!-- R14 -->

### Phase 3: Go — CLI override flags

- [x] T008 Add `--provider` / `--model` / `--effort` to `resolveAgentCmd` in `src/go/fab/cmd/fab/resolve_agent.go`: apply overrides via the T005 helper, re-derive `dispatch=` from the overridden provider, keep `--alias` semantics, and emit the unknown-provider lookup error listing `agent.ProviderNames(cfg)`. <!-- R8, R9 --> <!-- rework: review must-fix — the unknown-provider lookup error block (resolve_agent.go:118-129) duplicates agent.go:134-144 verbatim; extract a shared `unknownProviderError(cfg, name)` helper in package main and call it from both sites (fixes A-025). While editing this file: move the dispatch-substitution comment (lines 114-116) directly above the `dispatchLine` assignment it explains, and hoist the twice-evaluated `cmd.Flags().Changed("provider")` into a local `providerSet` (pattern agent.go:83 uses). -->
- [x] T009 Tests in `src/go/fab/cmd/fab/resolve_agent_test.go`: each override flag alone and combined, bare `--model`/`--effort` (no `--provider`) succeeding, the provider-swap fill/empty behavior, the `dispatch=` re-derivation, `--alias` passthrough of a non-Claude model, and the unknown-provider error. <!-- R14 -->

### Phase 4: Go — reference/fence presentation

- [x] T010 Rewrite `providersSegment()` (and the `providers` row's `Default`/`Description`) in `src/go/fab/internal/configref/configref.go` to present codex/gemini as commented built-in defaults sourced from the new agent constants, document the `model`/`effort` fill fields, and drop the "no new built-in providers in Go" / "uncomment to add that provider" claims. <!-- R3, R5 --> <!-- rework: review should-fix — the cross-provider cutoff prose (configref.go:506-508) frames the rule as cross-VENDOR; state it as provider-NAME-based ("a provider name differing from the built-in tier's — even the same vendor under a different name loses tier model/effort inheritance"). Behavior is correct per R7 (name inequality is the plan's rule); only the framing is wrong. -->
- [x] T011 Update the reference/fence guards in `src/go/fab/cmd/fab/config_test.go`: commented-block round-trip (`GetProvider("codex")` still `!ok`), the three-provider text guard, gemini's no-`{effort}`/no-`-p` guard, the retired-phrase negative guards, and the fill-field documentation guard. <!-- R14, R3 -->
- [x] T012 Run the affected Go packages' tests (`internal/agent`, `internal/config`, `internal/configref`, `cmd/fab`), then the full `go test ./...` in `src/go/fab`. <!-- R14 -->

### Phase 5: Skills + specs sweep

- [x] T013 Update `src/kit/skills/_cli-fab.md` § fab resolve-agent (override flags, precedence chain, cross-provider fill, unknown-provider error, the `fab agent` asymmetry) and its providers/fill prose. <!-- R10 --> <!-- rework: review should-fix — (a) state the cross-provider cutoff as provider-NAME-based, not vendor-based (~line 282); (b) add one sentence at the swap-rule prose (~line 322) noting a swap back to claude always lands on an EMPTY model (claude's fill lives on the built-in tiers, not the provider), so "run this stage on claude" needs an explicit --model (or a providers.claude.model fill). --> <!-- rework cycle 2: review should-fix — line ~1147 claims the no-session_command error is "reachable only for a user-defined dispatch-only provider, since all three built-ins ship a session_command"; FALSE for a provider NAMED codex/gemini (per-field merge inherits the built-in session command, so `providers.codex: {dispatch_command: ...}` still launches a session). Behavior is correct; fix only the sentence. -->
- [x] T014 [P] Add the one override paragraph to `src/kit/skills/_preamble.md` § Per-Stage Model Resolution. <!-- R11 --> <!-- rework: review should-fix — extend the EXISTING override paragraph (~line 318) with the same swap-back-to-claude empty-model sentence; keep it one paragraph total (R11's "exactly one paragraph" must still hold). --> <!-- rework cycle 2: review must-fix — the override paragraph (~lines 318,348) directs sites to override, branch on dispatch=, then run `fab dispatch start`, but `fab dispatch start` has NO override plumbing (it re-resolves via agent.Resolve internally): the documented override→CLI-dispatch workflow errors on the headless arm and silently runs the WRONG provider on the pane arm. Per plan Non-Goals ("no new dispatch machinery"), the fix is the CAVEAT, not plumbing: state that invocation-time overrides bind the NATIVE Agent-tool arm only — a `--provider` swap whose resolved provider carries a dispatch_command cannot ride `fab dispatch` (which re-resolves from config); moving a stage to CLI dispatch requires a config/tier override, not an invocation flag. ALSO should-fix: the always-load config.yaml enumeration (~line 54) still reads "per-provider session_command/dispatch_command" — add the model/effort default fill (its two aggregate-spec mirrors skills.md:65 + glossary.md:77 already say "and default fill"; the canonical source is the odd one out). Keep the override paragraph ONE paragraph. -->
- [x] T015 [P] Replace the uncomment-first / claude-only phrasing in `src/kit/skills/_cli-agents.md` with the built-in-providers + fill-field model. <!-- R12 --> <!-- rework: review should-fix — § Spawn Composition "Empty model/effort is a feature" (~line 62) must carry the caveat that `fab agent --provider <name>` bypasses tier resolution AND the provider default fill (providers.<name>.model is deliberately NOT consulted on this path — mirror the caveat _cli-fab.md:1144 already carries); do NOT change fab agent behavior (nvad's requires-provider/bypass rule stands, per plan Non-Goals). -->
- [x] T016 Update the SPEC mirrors for every skill edited above: `docs/specs/skills/SPEC-_cli-fab.md`, `SPEC-_preamble.md`, `SPEC-_cli-agents.md`. <!-- R13 --> <!-- rework: mirrors follow the T013–T015 rework edits (constitution SPEC-mirror sync). --> <!-- rework cycle 2: mirrors follow the cycle-2 T013/T014 edits, PLUS two mirror-specific gaps: SPEC-_cli-fab.md:24 lacks the supplied-vs-emptiness flag guard (`--model=` explicitly clears rather than being ignored — asserted in the skill, absent from the mirror); SPEC-_cli-agents.md:35 still says --model/--effort are "only valid alongside --provider" without the resolve-agent exception its source now notes (mirror stricter than source). -->
- [x] T017 [P] Update `docs/specs/stage-models.md` (three built-ins, fill fields, fill precedence, cross-provider rule, resolve-agent override flags) and `docs/specs/config.md` (the `providers` registry row's fill fields). <!-- R13 --> <!-- rework: review should-fix — restate the cross-provider cutoff as provider-NAME-based in stage-models.md (~lines 203-210), consistent with the T010/T013 rewording. --> <!-- rework cycle 2: review must-fix (doc half) — stage-models.md claims an invocation-time --provider override "can move a stage between the native and CLI adapters"; not executable (fab dispatch start re-resolves without overrides). Narrow to the T014 caveat: overrides bind the native arm; adapter moves require config/tier overrides. -->
- [x] T018 Grep-sweep `docs/specs/` (incl. `skills.md`, `glossary.md`, `architecture.md`, `harness-adapters.md`) and `src/kit/` for restatements of the one-built-in-provider or uncomment-to-opt-in model and update every hit. <!-- R13 --> <!-- rework cycle 2: review must-fix (doc half) — harness-adapters.md:285-288 carries the same not-executable claim (invocation-time override moving a stage between adapters); narrow it per the T014 caveat, and re-sweep for any other restatement of override→CLI-dispatch as an end-to-end workflow. -->
- [x] T019 Re-run the Go test suite after the doc sweep (`internal/agent`'s doc drift-guards and `cmd/fab`'s `_cli-fab.md` reference guard read the markdown). <!-- R14 --> <!-- rework: re-run after the rework edits (the extracted helper touches cmd/fab; the doc edits touch guard-read markdown). --> <!-- rework cycle 2: re-run after the T004 code fix + doc edits. -->

### Phase 6: Rework cycle 3 — cascade limitation (documented + pinned) and residual retired claims

<!-- rework cycle 3 (Revise requirements — mandated escalation after 2 consecutive fix-code cycles): R7 revised with the "ownership is computed over the MERGED config" scope paragraph. The cross-scope cascade gap (system scope and project scope naming different providers for one tier — a codex model ID handed to the gemini CLI) is resolved as a DOCUMENTED LIMITATION pinned by test, per the reviewer's option (b); cascade-aware ownership (folding per-scope layers in ResolveTier) is deferred to a follow-up change. -->

- [x] T020 Narrow the ownership doc comment in `src/go/fab/internal/agent/agent.go` (~line 283): drop the "never from the built-in tier's (or an intermediate layer's) foreign values" overclaim — ownership sees only the merged-config layers (`builtin ← default tier ← requested tier`), NOT the system↔project cascade, which `config.LoadPath` deep-merges per-key before resolution. State the cross-scope limitation explicitly. Add a pinning test reproducing the reviewer's cross-scope scenario (system `providers.codex` fill + `agent.tiers.doing: {provider: codex, model, effort}`; project `agent.tiers.doing: {provider: gemini}`) asserting the CURRENT merged-layer attribution with a comment marking it the R7 documented limitation — use the existing system-config test isolation pattern from `cmd/fab`'s config cascade tests (or add HOME isolation if `internal/agent` lacks one; the test may live at whichever layer can compose the two scopes). <!-- R7 (revised) -->
- [x] T021 Fix the two residual retired-claim sites: `src/go/fab/cmd/fab/resolve_agent.go` ~lines 58-60 (doc comment: "--provider ... can flip a stage between native Agent-tool dispatch and CLI dispatch" → restate as a QUERY RESULT per the narrowed markdown wording — the `dispatch=` line reports the named provider's `dispatch_command`; `fab dispatch start` re-resolves from config and takes no overrides) and `src/go/fab/cmd/fab/resolve_agent_test.go` ~line 487 (same phrasing in the test comment). <!-- R11 (narrowed claim), review cycle-3 must-fix 2 -->
- [x] T022 Record the cross-scope cascade limitation in `docs/specs/stage-models.md` (beside the cutoff/ownership rule), and tighten the narrowed override paragraph's FIRST remedy in `src/kit/skills/_preamble.md` (+ `stage-models.md` § Skill wiring and `harness-adapters.md` where the same remedy sentence lands): "dispatch the stage natively with the overridden model/effort" is only executable for a within-claude `--model`/`--effort` override (the Agent tool's `model` param is a Claude-alias enum), so for a cross-provider `--provider` override the config/tier override is the SOLE path. Keep `_preamble.md`'s override paragraph exactly one paragraph (R11). Update the SPEC mirrors for every skill file touched (`SPEC-_preamble.md`, and `SPEC-_cli-fab.md` if `_cli-fab.md` carries the remedy sentence). <!-- R7 (revised), R11, R13; review cycle-3 should-fix -->
- [x] T023 Re-run the affected package tests then the full `go test ./...` in `src/go/fab` (drift-guards read the edited markdown; T020's pinning test and T021's comment edits touch tested packages). <!-- R14 -->

## Execution Order

- T001 blocks T003, T004, T005, T010 (the constants and table are the source everything else reads).
- T002 blocks T003 (the merge needs the fields).
- T004 blocks T005 (the override helper reuses the cross-provider fill rule).
- T005 blocks T008.
- T010 blocks T011.
- T013–T015 block T016 (mirrors follow their skills); T018 runs after T013–T017.
- T019 is last (it validates the markdown the Go drift-guards read).
- Rework cycle 3: T020 and T021 are independent; T022 follows the R7 revision (its spec text cites the T020 comment wording); T023 is last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaultProviders` contains claude, codex, and gemini; the codex/gemini rows carry only command grammar (no model/effort), sourced from named Go constants with no duplicated literal in `configref`.
- [x] A-002 R2: A tier or flag naming `codex`/`gemini` resolves with no `providers:` block, and `dispatch=` appears for those providers while the built-in claude provider still emits none.
- [x] A-003 R3: `fab config reference` presents codex/gemini as commented built-in defaults; the retired "no new built-in providers in Go" and "uncomment … to add that provider" claims are gone; the commented blocks still parse as absent from `Config`.
- [x] A-004 R4: `config.ProviderConfig` carries `Model`/`Effort` (`yaml:"model"`/`"effort"`), per-field merged by `ResolveProvider`, with no migration shipped.
- [x] A-005 R5: The `providers` registry row documents the fill fields; its `Default` exposes only real built-in values (no fabricated model); scope stays `both`; key-parity holds.
- [x] A-006 R6: The fill precedence flag > tier field > provider fill > empty holds at every layer, verified by a test table.
- [x] A-007 R7: An explicit cross-provider `provider:` cuts off model/effort inheritance and fills from the provider's default fill, then empty; a tier without `provider:` inherits as before. **FIXED in rework cycle 2 (T004)**: the flattened `configured Profile` is gone. `ResolveTier` now FOLDS the config layers over the built-in in precedence order, recording each value's OWNING provider (the provider in effect at the layer that set it — `provider:` is applied before `model`/`effort` within a layer, so a value written beside a switch is owned by the new provider). The cutoff then refills only the fields whose owner differs from the resolved provider, via the new shared `cutForeignFields` helper that `ApplyOverrides` also calls (the two implementations of the one rule are now one). The Assumption-2 gate is unchanged — the *net* configured provider vs the built-in's — so a chain ending back on the built-in's provider is still not a switch. Verified: `default: {model: claude-fable-5, effort: medium}` + `doing: {provider: codex}` + `providers.codex: {model: gpt-5.3-codex, effort: high}` → `model=gpt-5.3-codex effort=high` (was `claude-fable-5`/`medium`).
- [x] A-008 R8: `fab resolve-agent` accepts `--provider`/`--model`/`--effort`, re-derives `dispatch=` from the named provider, allows bare `--model`/`--effort`, and errors on an unknown provider listing the resolvable names.
- [x] A-009 R9: `--alias` passes a non-Claude overridden model through verbatim while `dispatch=` keeps the full ID.
- [x] A-010 R10: `_cli-fab.md` § fab resolve-agent documents the flags, precedence, cross-provider fill, lookup error, and the `fab agent` asymmetry.
- [x] A-011 R11: `_preamble.md` § Per-Stage Model Resolution carries exactly one new override paragraph with the seam/branch/visibility rules unchanged. **FIXED in rework cycle 2 (T014, docs-narrowed per plan Non-Goals "no new dispatch machinery")**: the paragraph — still **exactly one paragraph** — now states that an invocation-time override binds the **native Agent-tool arm only**, that `fab dispatch start` takes no override flags (it re-resolves from config), that an override-only `dispatch=` line is therefore **not actionable**, and that moving a stage onto CLI dispatch requires a config/tier override. The same narrowing landed in `stage-models.md` (both the § Resolution step 2a flag doc and § Skill wiring), `harness-adapters.md` § Relationship to `stage-models.md`, `_cli-fab.md` (§ fab resolve-agent + a config-only note on § fab dispatch → start), and the `SPEC-_preamble.md` / `SPEC-_cli-fab.md` mirrors. Verified against `cmd/fab/dispatch_start.go` (flags: `--timeout`/`--pane`/`--server`; resolution via `agent.Resolve`).
- [x] A-012 R12: `_cli-agents.md` names the three built-in providers and the fill field, with no claude-only/uncomment-first claim left.
- [x] A-013 R13: Every edited skill has its SPEC mirror updated; `stage-models.md` and `config.md` record the new model; the aggregate-spec sweep is complete.
- [x] A-014 R14: Every new Go behavior has at least one dedicated test and `go test ./...` passes in `src/go/fab`. **FIXED in rework cycle 2 (T006)**: added `TestResolveCrossProviderCutoffAcrossLayers` — a 6-row table covering the previously-uncovered `default`-tier-supplies-`model`/`effort` variants: the must-fix combination (with fill, and with no fill → empty), the `default`-tier-switch-cuts-the-built-in's-values mirror, a value written *beside* a switch surviving, requested-tier fields written *above* a `default`-tier switch surviving, and the net-no-op provider chain cutting nothing (Assumption 2). Written as a failing test BEFORE the T004 fix (2 of 6 rows failed, exactly the reviewed defect) and green after. `go test ./...` re-run with a cleared test cache: all pass; `go vet` + `gofmt -l` clean.

### Behavioral Correctness

- [x] A-015 R7: With no config overrides, all six stages resolve byte-identically to the pre-change output (the all-claude default world is unchanged).
- [x] A-016 R2: Naming a built-in codex/gemini provider in a tier flips that tier's stages to CLI dispatch (a `dispatch=` line appears) — while an unnamed built-in stays inert.
- [x] A-017 R3: `fab config upgrade` remains idempotent (running it twice is byte-identical) with the regenerated providers segment.

### Scenario Coverage

- [x] A-018 R6: The precedence scenarios in R6 (tier field beats provider fill; flag beats both; provider fill supplies an unset tier field) are each exercised by a test.
- [x] A-019 R8: The R8 scenarios (`--provider` alone with no fill; full override triple; bare `--effort`; unknown provider) are each exercised by a test.
- [x] A-020 R1: `agent.ProviderNames` on an empty config returns exactly `[claude, codex, gemini]`.

### Edge Cases & Error Handling

- [x] A-021 R8: `--provider=` (explicitly empty) and an unknown name both produce the non-zero lookup error rather than silently resolving the tier's provider.
- [x] A-022 R7: A cross-provider tier with no provider fill resolves to an empty model and omits the `effort=` line rather than emitting another provider's values. **FIXED in rework cycle 2 (T004/T006)**: the `default`-tier-supplied case now resolves empty. Verified via the CLI — `default: {model: claude-fable-5, effort: medium}` + `doing: {provider: codex}` with no `providers.codex` fill → `model=` empty, no `effort=` line, `dispatch=codex exec` with both tokens dropped (was `model=claude-fable-5`). Covered by the new `TestResolveCrossProviderCutoffAcrossLayers` "no provider fill resolve empty" row.
- [x] A-023 R4: An absent `providers:` block, a nil config, and a provider entry with only fill fields (no commands) each resolve without panic, inheriting the built-in commands where they exist.
- [x] A-030 R7 (revised): The cross-scope cascade limitation is documented in `internal/agent`'s ownership doc comment (no overclaim covering scopes it cannot see) and in `stage-models.md`, and a test pins the cross-scope scenario's current behavior with a comment marking it the documented limitation. **DONE in rework cycle 3 (T020)**: `ResolveTier`'s comment dropped the "(or an intermediate layer's)" overclaim and gained a `SCOPE OF OWNERSHIP — a DOCUMENTED LIMITATION` block naming the merged-config boundary (`config.LoadPath` deep-merges system+project per key before resolution), the concrete failing case, the pinning test, and the deferral. `stage-models.md` § Fill precedence gained the matching **Scope of ownership** paragraph. `TestResolveCrossScopeCascadeLimitation` (`cmd/fab/resolve_agent_test.go` — the layer that can compose both scopes; `TestMain` already isolates HOME and the test points it at its own tree to write a system config) reproduces the reviewer's scenario and asserts the CURRENT bytes `model=gpt-5.3-codex / effort=high / provider=gemini / dispatch=gemini -m gpt-5.3-codex`, with a comment stating it pins the limitation and what to change when ownership becomes cascade-aware. Behavior first reproduced against a built binary before the test was written (same bytes).
- [x] A-031 R11: No site — markdown, Go doc comments, or test comments — still asserts that an invocation-time override can move a stage between the native and CLI adapters (grep-verified across `src/` and `docs/`). **DONE in rework cycle 3 (T021)**: the two residual Go sites were the last ones — `cmd/fab/resolve_agent.go` (doc comment: "so an override can flip a stage between native Agent-tool dispatch and CLI dispatch" → "That is a QUERY RESULT, not an adapter move: … relocating a stage … takes a config/tier override, never an invocation flag") and `cmd/fab/resolve_agent_test.go`'s `TestResolveAgentOverrideDispatchDisappearsOnNativeSwap` comment (same reframing; the assertion itself is unchanged and still correct — the query reports the named provider's `dispatch_command` or its absence). **Grep-verified** across `src/` and `docs/` (all `.md` + `.go`) for `adapter[ -]move`, for `override … (flip|move|relocat) … (adapter|native Agent|CLI dispatch)`, and for `(--provider|invocation-time) … (flip|move|relocat)`: 7 hits, **every one in the negated/narrowed form** (`_preamble.md`:318 "CANNOT move a stage onto CLI dispatch"; `_cli-fab.md`:322 + `stage-models.md`:343 + `resolve_agent.go`:61 + `resolve_agent_test.go`:489 "query result, NOT an adapter move"; `SPEC-_cli-fab.md`:24 + :28 "an adapter move requires a config/tier override"). Every `flips` hit repo-wide is about *naming a provider in config* (which genuinely does select the adapter) or unrelated (`backlog.MarkDone`, score-gate prose). `docs/memory/` carries zero occurrences — hydrate has not run yet, which is correct for this stage.
- [x] A-032 R11: The narrowed override paragraph's native-dispatch remedy is scoped to within-claude `--model`/`--effort` overrides everywhere the remedy sentence appears (a cross-provider `--provider` override's sole executable path is the config/tier override), with SPEC mirrors in sync. **DONE in rework cycle 3 (T022)**: all **six** sites carrying the remedy sentence now state the two remedies are **not interchangeable** — native dispatch is executable only for a within-claude `--model`/`--effort` override (the Agent tool's `model` param is a Claude-alias enum `opus`/`sonnet`/`haiku`/`fable`, so a non-Claude model has no native seam), leaving the config/tier override as the **sole executable path** for a cross-provider `--provider` override. Sites: `src/kit/skills/_preamble.md`:318 (still **exactly one paragraph** — verified: single line bounded by blank lines, R11 holds), `src/kit/skills/_cli-fab.md`:322, `docs/specs/stage-models.md` § Skill wiring, `docs/specs/harness-adapters.md` § Relationship to `stage-models.md`, and the mirrors `docs/specs/skills/SPEC-_preamble.md` (prose line 11 + the ASCII wiring diagram) and `SPEC-_cli-fab.md`:24. **SPEC-mirror sync**: the only skill files touched this cycle are `_preamble.md` and `_cli-fab.md`, and both mirrors were updated in the same edit pass. `_cli-fab.md` § fab dispatch → `start`'s config-only note was re-read and needed no change (it never carried the native-remedy claim).

### Code Quality

- [x] A-024 Pattern consistency: New code follows the surrounding naming, doc-comment, and error-message conventions (`mergeTierField`-style helpers, lookup-error phrasing mirroring `fab agent`).
- [x] A-025 No unnecessary duplication: The codex/gemini command strings exist as single Go constants consumed by both `defaultProviders` and `configref`; the override precedence lives in one `internal/agent` helper reused by `cmd/fab`.
- [x] A-026 Canonical source only: All kit edits are under `src/kit/`; nothing under `.claude/skills/` was edited.
- [x] A-027 SPEC-mirror sync: Every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` update in this change.
- [x] A-028 CLI ⇒ docs + tests: The `fab resolve-agent` signature change updates `_cli-fab.md` and ships tests.
- [x] A-029 No migration needed: The change adds only optional config keys and a Go built-in table row, so no `src/kit/migrations/` file is required.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [ ] A-NNN **N/A**: {reason}`

## Deletion Candidates

- ~~`src/go/fab/cmd/fab/agent.go:134-144` + `src/go/fab/cmd/fab/resolve_agent.go:118-129` — the duplicated unknown-provider lookup-error block.~~ **RESOLVED in rework cycle 1**: extracted to a single `unknownProviderError(cfg, name)` helper at `cmd/fab/agent.go:172-179`, called from `agent.go:135` and `resolve_agent.go:120`. Verified by grep — one definition, two call sites, no remaining copy.
- `src/kit/migrations/2.13.1-to-2.13.2.md:117` — carries the retired "commented starter TEMPLATE — uncomment and adapt to add that provider" framing. **Not** a deletion candidate: migration files are frozen historical records of past state and are deliberately never rewritten. Listed only to record that the sweep saw it and ruled it out.
- ~~`src/go/fab/internal/agent/agent.go:341-349` (`fillProfileFromProvider`) — flagged as the consolidation seam for the A-007 fix.~~ **RESOLVED in rework cycle 2**: `fillProfileFromProvider` (which took two "was this set?" booleans) is replaced by `cutForeignFields(cfg, p, modelOwner, effortOwner)`, which takes the two values' OWNING providers and compares each against `p.Provider`. `ResolveTier`'s cutoff and `ApplyOverrides`' swap are now the *same* rule expressed once — `ResolveTier` supplies owners recorded while folding the config layers, `ApplyOverrides` supplies owners recorded across the flag layer. Verified by grep: one definition, two call sites, no residual copy and no orphaned symbol.
- No Go symbol, branch, or config key became unreachable: the change is additive (three built-in rows where there was one, two new `ProviderConfig` fields, one new exported helper + override struct, three new flags). `providersSegment()`'s superseded prose and `TestConfigReferenceDocumentsThreeProviderTemplate`'s ho9y assertions were rewritten in place rather than left alongside replacements, so no dead duplicate remains there.
- Re-verified at the cycle-2 re-review: `mergeTierField` survives the `ResolveTier` rewrite legitimately (4 live call sites in `ResolveProvider`), `fillProfileFromProvider` has zero residual references repo-wide, and `unknownProviderError` has one definition (`cmd/fab/agent.go:172`) with two call sites. Every remaining literal codex/gemini command string in `src/go` is a *test fixture* pinning its own input, not a copy of a production constant — A-025's single-constant rule holds in non-test code.
- Re-verified at the cycle-3 re-review: **nothing new became redundant.** Cycle 3 added only doc text (`ResolveTier`'s `SCOPE OF OWNERSHIP` block, the narrowed `resolve_agent.go`/`resolve_agent_test.go` comments, the six remedy-sentence sites) plus one test (`TestResolveCrossScopeCascadeLimitation`) — no executable code path was added, replaced, or orphaned. Grep-confirmed zero call sites lost: every new exported symbol (`ApplyOverrides`, `Overrides`, `cutForeignFields`, the four `DefaultCodex*`/`DefaultGemini*` constants, `providerDefaults`, `unknownProviderError`) has ≥3 non-test references, and `fillProfileFromProvider` remains absent repo-wide. The `2.13.1-to-2.13.2.md` migration entry above still stands as ruled-out (frozen historical record).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The codex/gemini command strings become named Go constants in `internal/agent` (like `DefaultSessionCommand`) that `internal/configref` interpolates, rather than literals duplicated in both packages | The package doc already mandates "GENERATED, NOT HAND-WRITTEN — every default value that has a canonical Go constant is sourced from that constant, never copied"; promoting the strings to built-ins makes them canonical constants by definition | S:85 R:85 A:95 D:90 |
| 2 | Confident | "Explicitly sets `provider:`" (R7) means an explicit config `provider:` on the requested tier's own override OR on the project's `default` tier, resolved against the built-in tier profile's provider — cutting inheritance only when the two differ | The footgun is cross-provider field inheritance; a `default`-tier provider switch is the same footgun one layer up, and an explicit provider equal to the built-in's is not a switch at all (so the all-claude world stays byte-identical) | S:70 R:70 A:85 D:75 |
| 3 | Confident | A cross-provider tier with no provider fill resolves to an EMPTY model/effort rather than erroring | Empty already has a defined meaning at both seams (inherit-session on `model=`; `WithProfile` token-drop → the CLI's own default), so empty is the strictly-more-useful terminal state and matches the intake's stated "→ empty" precedence tail | S:80 R:75 A:85 D:80 |
| 4 | Confident | The `providers` registry row's structured `Default` gains codex/gemini entries carrying only their command fields — no `model`/`effort` keys, since no built-in fill exists | The registry's documented empty-default convention says a non-nil Default "always denotes a real built-in value"; emitting an empty-string model would assert a built-in fill that deliberately does not exist | S:70 R:80 A:85 D:80 |
| 5 | Confident | The override precedence is implemented as one exported `internal/agent` helper consumed by `cmd/fab`, not inline in the cobra `RunE` | The package already owns every resolution rule (`ResolveTier`/`ResolveProvider`/`Resolve`) and the cross-provider fill rule must be shared between tier resolution and the flag path; inlining it in `cmd/fab` would split precedence across two packages | S:65 R:80 A:85 D:80 |
| 6 | Confident | `providers` keeps its single registry row (no new `providers.<name>.model` rows) so the JSON/YAML key-parity guard and the map-valued override-unit model are unchanged | `configref`'s `Field` doc states map-valued fields are single rows "matching the downstream per-field deep-merge"; the fill fields are leaves inside that map, exactly like `session_command` is today | S:75 R:85 A:90 D:85 |
| 7 | Confident | The `fab/project/config.yaml` fence in this repo is regenerated by running `fab config upgrade` (with the freshly built binary) as part of the sweep rather than hand-edited, and only if the installed-binary/version skew allows it — otherwise the fence is left to regenerate at release time as the intake states | The intake says the fence "regenerates via `fab config upgrade` at release time (not hand-edited here beyond what tests require)"; hand-editing a managed fence is exactly what the fence forbids | S:70 R:80 A:80 D:70 |
| 8 | Tentative | `--provider` on `fab resolve-agent` does NOT retain the tier's explicit model — a swap fills from the new provider's fill, then empty <!-- assumed: consistency with R7 chosen over "keep the tier's model unless overridden"; the intake states this rule explicitly, but a user overriding only the provider might expect their tier's model to persist --> | The intake states it ("tier explicit field is NOT retained for a swapped provider's model … same rule as §2"), and retaining a claude model across a swap to codex is precisely the footgun R7 closes; reversible via a follow-up flag if it proves surprising | S:60 R:75 A:75 D:55 |

| 9 | Certain | Rework cycle 1: the shared `unknownProviderError(cfg, name) error` helper lives in `cmd/fab/agent.go` (beside `runAgent`, the original call site) rather than in a new file, and `resolve_agent.go` calls it | The two call sites are the same `package main`; `agent.go` already owns the provider-lookup path and the error's phrasing contract, and adding a file for one 8-line formatter would be less discoverable than the existing home | S:90 R:90 A:95 D:90 |
| 10 | Confident | Rework cycle 1: `fab/project/config.yaml`'s reference fence is left UNregenerated (Assumption 7's second branch) even though the providers-segment prose changed | The installed binary is 2.16.12 (pre-change), and the worktree binary stamps the fence `kit dev` instead of a real version — committing that stamp is worse than the documented release-time regeneration. Verified by running the worktree binary's `config upgrade` (correct new prose, byte-idempotent on a second pass) against a scratch copy, then reverting the repo file. No test reads this repo's own fence | S:75 R:90 A:85 D:80 |
| 11 | Confident | Rework cycle 2: a value's OWNER is the provider in effect at the layer that supplied it, computed **bottom-up** (the layer's own `provider:` if it names one, else whatever the layers below resolved) — with `provider:` applied before `model`/`effort` *within* a layer, so a value written beside a switch is owned by the new provider | This is the reading that makes "never from another provider's values" (R7) hold across all three layers while keeping every previously-passing case intact. A `default`-tier model with no provider beside it was authored under the built-in's claude, so claude owns it; a model written next to `provider: codex` was authored for codex. Top-down ownership (attributing every unqualified value to the *final* provider) would defeat the cutoff entirely | S:75 R:80 A:85 D:80 |
| 12 | Confident | Rework cycle 2: the Assumption-2 **gate** stays anchored to the NET configured provider vs `builtin.Provider` — the per-field owner check runs only *inside* that gate, never as a standalone per-layer fold | A pure layer-by-layer fold would cut on an intermediate switch that the top layer reverses (`default: {provider: codex, effort: medium}` + `doing: {provider: claude, model: X}` would lose `medium`), changing behavior `TestResolveOverrideBeatsDefaultTier` already pins and that Assumption 2 explicitly rules out ("an explicit provider equal to the built-in's is not a switch at all"). Constitution VII: the spec governs, so the gate is kept and only the per-field anchor changed | S:80 R:80 A:85 D:85 |
| 13 | Confident | Rework cycle 2: the T014/T017/T018 must-fix is closed by **narrowing the docs** (invocation-time overrides bind the native Agent-tool arm only) rather than plumbing `--provider`/`--model`/`--effort` through `fab dispatch start` | Plan Non-Goals state "no new dispatch machinery", and the orchestrator's rework directive named the narrowing explicitly. Plumbing would add a second resolution surface with its own precedence and compliance-visibility contract — the exact thing the "Overrides Land on `resolve-agent`, Not New Dispatch Machinery" Design Decision rejected. A later change can still add the flags; the docs now describe what the code does | S:85 R:80 A:90 D:85 |

| 14 | Confident | Rework cycle 3: the cross-scope pinning test lives in `cmd/fab/resolve_agent_test.go` (not `internal/agent`), asserting the CLI's resolved stdout bytes rather than a `Profile` struct | `internal/agent` takes an already-merged `*config.Config`, so it structurally CANNOT compose two scopes — reproducing the limitation there would mean hand-forging the merged tree, which pins the merge's *output* rather than the cascade that produces it. `cmd/fab` runs the real `config.Load` cascade end-to-end, and its `TestMain` already isolates HOME (with a documented per-test `t.Setenv` escape for writing a system config), so the test exercises the actual reader path a user hits. The plan's T020 explicitly allowed "whichever layer can compose the two scopes" | S:85 R:85 A:90 D:85 |
| 15 | Confident | Rework cycle 3: the narrowed remedy is expressed as "the two remedies are NOT interchangeable" with the Claude-alias-enum reason stated inline at each of the six sites, rather than stated once and cross-referenced | The reason (the Agent tool's `model` param is a Claude-alias enum) is short and is the whole load-bearing content of the narrowing — a bare cross-reference would leave a reader at, say, `harness-adapters.md` believing native dispatch is a general escape hatch. This matches the repo's existing pattern for this paragraph, where every site already restates the "`fab dispatch start` takes no override flags" reason rather than pointing elsewhere for it | S:75 R:85 A:85 D:80 |
| 16 | Confident | Rework cycle 3: the limitation is documented WITHOUT adding a runtime warning (no `fab: warning:` when the two scopes name different providers for one tier) | Detecting the condition would need the per-scope layers at resolution time — which is precisely the cascade-aware ownership the revised R7 defers — so a warning is not a cheaper subset of the fix, it *is* the fix's hard half. R7 (revised) prescribes exactly two artifacts (the doc statement and the pinning test) and the plan's Non-Goals bar new machinery; a warning would also fire on the legitimate case of a user deliberately re-pointing a tier machine-wide | S:70 R:85 A:85 D:80 |

16 assumptions (2 certain, 13 confident, 1 tentative).
