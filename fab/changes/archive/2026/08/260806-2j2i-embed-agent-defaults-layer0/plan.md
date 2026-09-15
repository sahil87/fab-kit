# Plan: Embed Agent Defaults as defaults.yaml (Layer 0)

**Change**: 260806-2j2i-embed-agent-defaults-layer0
**Intake**: `intake.md`

## Requirements

### internal/agent: Built-in defaults as an embedded data file

#### R1: The built-in agent defaults live in an embedded YAML data file
`src/go/fab/internal/agent/defaults.yaml` SHALL exist and carry fab-kit's built-in agent defaults — the top-level `providers:` table (three grammar-only providers) and the `agent.tiers` table (six tier profiles) — shaped exactly as a **user config-file fragment** in today's schema, with today's exact values. The file SHALL be compiled into the binary via `//go:embed` in package `agent`; it SHALL NOT be read from the kit cache or any other on-disk location at runtime.

- **GIVEN** the file `src/go/fab/internal/agent/defaults.yaml`
- **WHEN** the `fab` binary is built and run from a directory with no fab-kit cache present
- **THEN** `fab resolve-agent <stage>` resolves the built-in profiles normally — the defaults travel inside the binary, and no filesystem read of the kit cache occurs

#### R2: `defaultTiers` and `defaultProviders` are parsed from the embedded file, not Go literals
The `defaultTiers` and `defaultProviders` package variables in `src/go/fab/internal/agent/agent.go` SHALL be populated by parsing the embedded file once at package initialization, and SHALL NOT restate any provider command string, model ID, or effort level as a Go literal. The parse SHALL reuse the existing `internal/config` schema types (`config.Config` / `config.ProviderConfig` / `config.TierProfile`) rather than introducing a parallel struct definition, since the file is a config fragment by design.

- **GIVEN** `defaults.yaml` naming `doing: { provider: claude, model: claude-opus-5, effort: xhigh }`
- **WHEN** `DefaultTier("doing")` is called
- **THEN** it returns `Profile{Provider: "claude", Model: "claude-opus-5", Effort: "xhigh"}` and `ok == true`
- **AND** `grep` for that model ID in `src/go/fab/internal/agent/agent.go` finds no occurrence

#### R3: The public API of `internal/agent` is behaviorally unchanged
`DefaultTier()`, `TierForStage()`, `IsTierName()`, `TierNames()`, `StageNames()`, `ResolveTier()`, `Resolve()`, `ResolveProvider()`, and `ProviderNames()` SHALL keep their signatures and SHALL return byte-identical results for every input they accept today. The exported built-in command identifiers (`DefaultSessionCommand`, `DefaultCodexSessionCommand`, `DefaultCodexDispatchCommand`, `DefaultGeminiSessionCommand`, `DefaultGeminiDispatchCommand`) SHALL keep their names and string values, sourced from the parsed defaults; consumers that require a compile-time constant (today only `spawn.DefaultSpawnCommand`) SHALL be adjusted to the variable form without changing their resolved value.

- **GIVEN** the pre-change test suites in `internal/agent`, `internal/spawn`, `internal/configref`, `internal/configupgrade`, and `cmd/fab`
- **WHEN** they are run against the post-change tree **without modification**
- **THEN** every one of them passes — the unchanged suites are the behavior-neutrality proof
- **AND** `fab config reference` renders byte-identically to its pre-change output

#### R4: `stageTiers` stays in Go
The fixed stage→tier mapping (`stageTiers`) SHALL remain a Go map literal in `agent.go` and SHALL NOT move into `defaults.yaml`. The YAML/Go split SHALL be documented in both files as the overridable/fixed boundary: what lives in `defaults.yaml` is user-overridable by writing the same keys in `config.yaml`; what remains in Go is fab-owned policy.

- **GIVEN** a project `config.yaml` containing an `agent.tiers.doing` override
- **WHEN** `fab resolve-agent apply` runs
- **THEN** the override is honored (the tier is data)
- **AND** no configuration key exists that can reassign `apply` to a different tier (the mapping is policy)

#### R5: A validation test guards the embedded file
`internal/agent` SHALL carry a test that parses the embedded `defaults.yaml` and asserts: the parse itself succeeds; all six tiers (`default`, `operator`, `doing`, `review`, `hydrate`, `fast`) are present with non-empty `provider`, `model`, and `effort`; all three providers (`claude`, `codex`, `gemini`) are present; `claude` carries a `session_command` and **no** `dispatch_command`; `codex` and `gemini` carry both command fields; and no provider carries a `model`/`effort` fill. It SHALL also assert the file contains no keys outside the `providers:`/`agent.tiers` surface it is meant to define.

- **GIVEN** a typo is introduced into `defaults.yaml` (a malformed line, a dropped tier, or an emptied field)
- **WHEN** `go test ./internal/agent/...` runs
- **THEN** the validation test fails and names the defect — the safety net that was previously a compile error

#### R6: Documentation points at the new bump site
`docs/specs/stage-models.md` SHALL name `src/go/fab/internal/agent/defaults.yaml` as the source of the default tier profiles and as the file a model bump edits, while continuing to name `agent.go` for `stageTiers`. The "TO BUMP A MODEL" guidance block SHALL live in `defaults.yaml` itself, with `agent.go` pointing at it rather than restating it.

- **GIVEN** a reader following `docs/specs/stage-models.md` to bump a model
- **WHEN** they follow the § Default tier profiles and § Drift guard prose
- **THEN** they are directed to `defaults.yaml`, and the drift-guard test names remain accurate
- **AND** `TestDocTablesMatchAgentMaps`, `TestMirrorDocsMatchDefaultTiers`, and `TestCLIFabReferenceListsDefaultTiers` still pass unchanged

### Non-Goals

- **Schema reshape** (`agent.tiers` → `agent.profiles`, per-role provider fills, `agent.session`/`agent.workers`) — that is j9nh's change; this file keeps today's schema verbatim.
- **Built-in codex/gemini model fills** — that is ywkx's change; `defaults.yaml` ships grammar only, exactly as `defaultProviders` does today.
- **Config-merge/precedence rewrite** — `defaults.yaml` is *shaped* as a config fragment so it can later become layer 0 of the cascade, but the merge code paths are untouched here.
- **Runtime reading from `$(fab kit-path)`** — explicitly rejected; embedding is the point.
- **Memory updates** (`docs/memory/runtime/providers-and-tiers.md`, `docs/memory/_shared/configuration.md`) — those are hydrate's stage, not apply's.

### Design Decisions

#### Built-in Command Strings Become Package Vars Sourced from the Embedded File
**Decision**: `DefaultSessionCommand` and the four `DefaultCodex*`/`DefaultGemini*` identifiers change from `const` to `var`, initialized from the parsed `defaults.yaml` entries. `spawn.DefaultSpawnCommand`, the only compile-time-constant consumer, becomes a `var` alias of the same value.
**Why**: The file now owns these strings. Keeping the Go constants as literals would duplicate every command string in two places and reintroduce, by hand, exactly the drift this change removes by construction. Keeping the identifiers (rather than deleting them in favor of `ResolveProvider(nil, name)` calls) leaves `internal/configref`, `internal/configupgrade`, and every test call site untouched, which is what makes the unchanged suites a usable behavior-neutrality proof.
**Rejected**: Keeping the constants as the canonical strings and having `defaults.yaml` restate them (drift-by-test, the thing being removed); a test asserting const↔YAML equality (same drift, one indirection later); deleting the identifiers and rewriting every consumer (inflates the diff of a behavior-neutral change and destroys the unchanged-suite proof).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0

#### The Embedded File Parses Through the Existing Config Schema Types
**Decision**: The embedded bytes unmarshal into `config.Config` — the same struct `config.LoadPath` fills from a user's `config.yaml` — and the `map[string]config.TierProfile` result is converted to the package's `map[string]Profile`.
**Why**: The file's whole design premise is that it is a config-file fragment; parsing it through any other struct would let the two shapes diverge silently. It also makes j9nh's layer-0 unification a merge-order change rather than a parser change.
**Rejected**: A bespoke `defaultsFile` struct in `internal/agent` (a second schema definition to keep in sync); parsing into `map[string]any` and hand-walking it (loses the schema's yaml tags and its zero-value semantics).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0

#### A Parse Failure Panics at Package Initialization
**Decision**: The embedded parse runs in a package-level initializer and panics on a YAML error.
**Why**: The bytes are compiled into the binary, so a parse failure is a defective build artifact, not a runtime condition a user can produce or recover from. Returning an error would force every existing caller of `DefaultTier`/`ResolveTier` to grow an error path for a state that cannot occur in a released binary — precisely the "`fab resolve-agent` cannot break from a missing/corrupt file" property the intake requires be preserved.
**Rejected**: Falling back to hardcoded Go values on a parse error (restores the duplicate literals); returning errors from the accessors (widens the API of a behavior-neutral change); `sync.Once`-on-first-use (defers a build defect to an arbitrary call site with no benefit — there is no I/O to make lazy).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/agent/defaults.yaml` with the config-fragment header comment, the three-provider `providers:` block, and the six-tier `agent.tiers:` block carrying today's exact values (verbatim from `agent.go`), plus the relocated "TO BUMP A MODEL" guidance as YAML comments <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 In `src/go/fab/internal/agent/agent.go`: add the `embed` + `yaml.v3` imports, the `//go:embed defaults.yaml` directive, and a `mustParseDefaults` initializer that unmarshals the bytes into `config.Config` and panics on error <!-- R2 -->
- [x] T003 Replace the `defaultProviders` and `defaultTiers` map literals in `agent.go` with values derived from the parsed defaults (converting `config.TierProfile` → `Profile`), keeping both identifiers and their doc comments (rewritten to name `defaults.yaml` as the source) <!-- R2 -->
- [x] T004 Convert `DefaultSessionCommand`, `DefaultCodexSessionCommand`, `DefaultCodexDispatchCommand`, `DefaultGeminiSessionCommand`, and `DefaultGeminiDispatchCommand` from `const` to `var`, sourced from the parsed provider entries, and update their doc comments <!-- R3 -->
- [x] T005 Update `src/go/fab/internal/spawn/spawn.go` to declare `DefaultSpawnCommand` as a `var` (the const form no longer compiles against a non-constant source), preserving its doc comment's meaning <!-- R3 -->
- [x] T006 Keep `stageTiers` in `agent.go` and update the package doc comment plus the `stageTiers` comment to state the YAML/Go boundary (data vs. fab-owned policy) <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Add `src/go/fab/internal/agent/defaults_test.go` asserting the embedded file parses, covers exactly the six tiers with non-empty fields, covers exactly the three providers with the expected command-field presence/absence, carries no provider `model`/`effort` fill, and defines no out-of-surface top-level keys <!-- R5 -->
- [x] T008 Run `go test ./internal/agent/... ./internal/spawn/... ./internal/configref/... ./internal/configupgrade/... ./cmd/fab/...` and confirm every pre-existing test passes **unmodified**; widen to `go test ./...` and `go vet ./...` for the module <!-- R3 -->

### Phase 4: Polish

- [x] T009 Update `docs/specs/stage-models.md` prose (header note, § Default tier profiles, § The fixed stage → tier mapping, § Fable upgrade path, § Drift guard) so the tier profiles are attributed to `defaults.yaml` while `stageTiers` stays attributed to `agent.go`; leave both mirrored tables byte-unchanged <!-- R6 -->
- [x] T010 [P] Verify no other doc or skill names `agent.go` as the model-bump site, and no doc, skill, or comment still calls the converted identifiers Go constants — swept across `src/go/`, `src/kit/`, and `docs/specs/` (`docs/memory/` is hydrate's) <!-- R6 -->

## Execution Order

- T001 blocks T002–T004 (the file must exist for `go:embed` to compile)
- T002 blocks T003; T003 blocks T004; T004 blocks T005
- T007 requires T001–T004; T008 requires everything in Phases 1–3
- T009/T010 are documentation-only and independent of the Go work, but T010's grep is meaningful only after T009

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/go/fab/internal/agent/defaults.yaml` exists, is valid YAML in today's config schema, and is embedded via a `//go:embed defaults.yaml` directive in package `agent` — directive at `agent.go:50`; the file parses through `config.Config`
- [x] A-002 R2: `defaultTiers` and `defaultProviders` are populated from the embedded file; `agent.go` contains no model ID, effort level, or provider command literal — verified by grep (the only `claude-*` literals left are the pre-existing `modelAliases` family prefixes at `agent.go:285-288`, which are alias-mapping inputs, not default values)
- [x] A-003 R4: `stageTiers` remains a Go map literal in `agent.go`, and no config key can reassign a stage's tier — `agent.go:207+`; no `stage_tiers` key exists anywhere in `src/go/` or `src/kit/`
- [x] A-004 R5: `internal/agent` carries a validation test over the embedded file covering tier coverage, field non-emptiness, provider coverage, and command-field presence/absence — `defaults_test.go`
- [x] A-005 R6: `docs/specs/stage-models.md` names `defaults.yaml` as the tier-profile source and bump site, and `agent.go` for `stageTiers`

### Behavioral Correctness

- [x] A-006 R3: Every pre-existing test in `internal/agent`, `internal/spawn`, `internal/configref`, `internal/configupgrade`, and `cmd/fab` passes **without any assertion change**; `go test ./...` and `go vet ./...` are green module-wide (note: `internal/configref` ships no test files, so it is vacuous there — its behavior is covered through `configupgrade`). Review rework 1 added two **doc-comment-only** edits to pre-existing test files — `internal/agent/agent_test.go:44-52` (bump site now `defaults.yaml`) and `cmd/fab/config_test.go:324` ("agent command vars", not "constants") — both stale-prose fixes the review raised as should-fix; no test body, table, or expectation is touched, so the unchanged-suite behavior-neutrality proof holds on its substance
- [x] A-007 R3: The exported command identifiers keep their names and resolved string values; `spawn.DefaultSpawnCommand` resolves to the same command text as before — verified empirically: `fab agent --provider claude|codex|gemini --print` is byte-identical between a binary built at `origin/main` and this tree
- [x] A-008 R3: `fab config reference` and the `fab config upgrade` managed fence render byte-identically to their pre-change output (asserted by the unchanged `configref`/`configupgrade` suites) — additionally verified by direct `diff` of `fab config reference` output between the base-commit binary and this tree's: identical
- [x] A-009 R6: `TestDocTablesMatchAgentMaps`, `TestMirrorDocsMatchDefaultTiers`, and `TestCLIFabReferenceListsDefaultTiers` pass unchanged after the doc edits

### Scenario Coverage

- [x] A-010 R2: `DefaultTier(tier)` returns the profile written in `defaults.yaml` for each of the six tiers, with `ok == true` — `TestPackageTablesMatchDefaultsFile` + `TestDefaultTierProfilesArePinned`; `fab resolve-agent` output identical to the base binary for all six tiers and all six stages
- [x] A-011 R1: Resolution performs no filesystem read for the defaults — the only `defaults.yaml` reference in non-test code is the `go:embed` directive — confirmed; `fab resolve-agent apply` resolves the built-in profiles with an empty `$HOME` and no kit cache

### Edge Cases & Error Handling

- [x] A-012 R5: A malformed or incomplete `defaults.yaml` is caught — a parse error panics at package initialization, and a structurally-valid-but-wrong file (missing tier, emptied field, missing provider) fails the validation test with a message naming the defect — exercised on a scratch copy of the module: malformed YAML panicked with `internal/agent: embedded defaults.yaml is malformed: …`, and a dropped `fast` tier, an emptied `review.model`, a missing `gemini` provider, a stray `providers.codex.model` fill, and out-of-surface `source_paths`/`agent.profiles` keys each produced a distinct named failure

### Code Quality

- [x] A-013 Pattern consistency: The embed + parse follows the codebase's existing conventions (the `go:embed` pattern in `cmd/fab/skill.go`, `yaml.v3` unmarshalling as in `internal/config`), and the new test follows the existing `internal/agent` test style
- [x] A-014 No unnecessary duplication: The parse reuses `internal/config`'s schema types rather than defining a parallel struct, and no command string, model ID, or effort level appears in more than one place in the tree — scoped to *unguarded* duplication: the remaining restatements are the pre-existing drift-guarded ones (`TestDefaultTierProfilesArePinned`'s deliberate pin, the two `stage-models.md` mirror tables, `_cli-fab.md`'s enumeration) plus the pre-existing prose copy of the claude session command in `docs/memory/distribution/kit-architecture.md:311`, none introduced here
- [x] A-015 Go changes ship tests: The `.go` changes are accompanied by the new validation test (project code-review rule: a `.go` change without test updates is a must-fix gap)
- [x] A-016 Sibling & mirror sweep: The `agent.go` ↔ `docs/specs/stage-models.md` mirror class was swept up front, and no other doc or skill still names `agent.go` as the bump site. Both skill↔SPEC mirror pairs this change touches now move together — `src/kit/skills/_cli-agents.md:131` ↔ `docs/specs/skills/SPEC-_cli-agents.md:56,:99` (the built-in `defaultProviders` table, rows in the embedded `defaults.yaml`) and `src/kit/skills/_cli-fab.md:386` ↔ `docs/specs/skills/SPEC-_cli-fab.md:25` (canonical Go *symbol*, not constant). Re-verified by a wider grep over `src/go/`, `src/kit/`, and `docs/specs/` for "canonical Go constant", "Go built-in provider table", "agent constants", "constant-backed/-sourced", and "update defaultTiers in agent.go": zero residuals (`docs/memory/` deliberately excluded — hydrate's stage)
- [x] A-017 No magic strings: Tier and provider names in the new test use the existing `Tier*` constants and `DefaultProviderName` rather than bare literals where those exist — plus the two new `providerCodex`/`providerGemini` constants; the only bare literals are YAML key names (`providers`, `agent`, `tiers`), which have no constants

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [ ] A-NNN **N/A**: {reason}`

## Deletion Candidates

*(Replaced in review cycle 2. The rework-1 candidate — the gemini `{effort}`/`-p` assertions formerly at `defaults_test.go:98-103` — was **actioned**: deleted as verbatim duplicates of `TestResolveProvider_BuiltInCodexAndGemini` (`agent_test.go:636-641`), with a pointer comment left in its place.)*

- `internal/configref.providerDefaults()` (`configref.go:220-231`) — a hand-written three-row mirror of the built-in provider table (with bare `"codex"`/`"gemini"` literals, the only ones left in the tree now that `agent` has `providerCodex`/`providerGemini`). This change makes the rows *data*, so the whole function body is derivable from `agent.ProviderNames(nil)` + `agent.ResolveProvider(nil, name)` projected onto `providerDefault` — deleting the literal row list and both magic strings. **Follow-up**, not this change: it renders identically but widens a behavior-neutral diff into `configref`
- `src/go/fab/internal/agent/defaults_test.go:138-149` (the `DefaultTier` and `ResolveProvider` loops in `TestPackageTablesMatchDefaultsFile`) — cannot fail independently: both sides derive from the same embedded bytes through the same conversion. Re-confirmed in review cycle 2 by a scratch mutation (typo'd nested keys) in which these loops stayed green. Only the command-var table at `:151-166` detects a real mis-wiring
- `src/go/fab/internal/agent/defaults_test.go:45-59` (the per-tier non-emptiness loop) — strictly weaker than `TestDefaultTierProfilesArePinned`'s exact-value pin at `agent_test.go:53`, so it can only fire on defects the pin already names. Keep only if the file-independent read is wanted for its own sake; the exhaustive key-set assertion above it is the part with unique coverage
- `agent.DefaultCodexSessionCommand` / `DefaultCodexDispatchCommand` / `DefaultGeminiSessionCommand` / `DefaultGeminiDispatchCommand` (`internal/agent/agent.go:135-148`) — now thin lookups into `defaultProviders`, which `ResolveProvider(nil, name)` already exposes; their only non-test consumer is `internal/configref:222-229,558-567`. Deliberately retained here to keep the unchanged-suite behaviour-neutrality proof (plan § Design Decisions), so this is a **follow-up** candidate for j9nh, not for this change

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The exported built-in command identifiers become package `var`s sourced from the embedded file (and `spawn.DefaultSpawnCommand` follows), rather than staying Go `const` literals | The file now owns the strings; keeping the consts would duplicate every command string and reintroduce the drift this change removes. `const`→`var` is a compile-visible but signature-preserving change with exactly one affected declaration site | S:70 R:80 A:85 D:70 |
| 2 | Certain | The embedded bytes are unmarshalled into `config.Config` — the same struct the user-config loader fills — and converted to `map[string]Profile` | The intake specifies the file is shaped exactly as a config-file fragment; parsing it through the config schema is the only reading that keeps the two shapes from diverging, and it is what makes j9nh's layer-0 unification a merge-order change | S:85 R:85 A:90 D:80 |
| 3 | Confident | A parse failure panics at package initialization rather than returning an error or falling back | The bytes are compiled in, so a failure is a defective build artifact, not a runtime state; returning errors would widen the API of a behavior-neutral change and break the "resolution cannot fail" property the intake requires be preserved | S:60 R:85 A:85 D:70 |
| 4 | Certain | The "TO BUMP A MODEL" comment block moves into `defaults.yaml`, with `agent.go` pointing at it | The intake states the comment "moves to/points at the YAML file"; putting the guidance next to the lines it describes is the reading that makes the file self-explanatory to a non-Go reader, which is the change's stated motivation | S:80 R:90 A:85 D:85 |
| 5 | Certain | Apply touches `docs/specs/stage-models.md` only; no `src/kit/skills/*.md` or memory file needs a bump-site edit | Verified by grep — `agent.go` is named in `docs/specs/stage-models.md` and nowhere else under `docs/` or `src/kit/`; `_cli-fab.md` enumerates the tier values (drift-guarded) but names no bump site, and memory edits belong to hydrate | S:70 R:85 A:90 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative).
