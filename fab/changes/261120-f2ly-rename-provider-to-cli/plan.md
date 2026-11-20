# Plan: Rename provider → cli

**Change**: 261120-f2ly-rename-provider-to-cli
**Intake**: `intake.md`

## Requirements

### Config: canonical `clis` spelling with deprecated aliases

#### R1: `clis:` is the canonical CLI-tool table
The built-in table in `src/go/fab/defaults.yaml` SHALL live under top-level `clis:` (entries claude, codex, agy, kimi with unchanged content). The typed config (`internal/config`) SHALL expose it as the canonical field (`CLIs map[string]CLIConfig`, yaml `clis`); `ProviderConfig` SHALL be renamed `CLIConfig` and `ProviderProfile` → `CLIProfile`.

- **GIVEN** a config with no user override
- **WHEN** `fab config show --json` runs
- **THEN** the built-in entries appear under `clis.` keys and no `providers.` keys are emitted

#### R2: Deprecated `providers:` and `.provider` are read, normalized, new wins per field
Every raw config layer (project file, system file, environment overlay) SHALL be normalized before merge by one shared function in `internal/config`: a live top-level `providers:` map is folded into `clis:` per entry and per leaf (recursing into `profiles.<role>`), a non-empty `clis` leaf winning over the `providers` leaf; `agent.profiles.<role>.provider` (and the legacy `agent.tiers.<role>.provider`) is folded into `.cli` the same way. After normalization no `providers` / `.provider` key remains in the layer, so every consumer (resolver, `show`, `--origin`, `explain`, setup checks, `fab agent models`) sees only the new shape. An explicit `native: false` under either spelling SHALL still disable a built-in `true`. Ordinary reads SHALL stay silent (no stderr warning).

- **GIVEN** a project config with only `providers: {codex: {profiles: {review: {model: x}}}}` and `agent.profiles.review.provider: codex`
- **WHEN** `fab agent review -o yaml` runs
- **THEN** it resolves exactly what the same config spelled `clis:`/`.cli` resolves (`cli: codex`, `model: x`) and stderr is empty
- **GIVEN** one layer with `clis.codex.profiles.review.model: new` and `providers.codex.profiles.review.model: old` plus `providers.codex.profiles.review.effort: high`
- **WHEN** the layer is normalized
- **THEN** the result is `clis.codex.profiles.review: {model: new, effort: high}`
- **GIVEN** system config on `providers:` and project config on `clis:`
- **WHEN** resolution runs
- **THEN** the cascade merges them exactly as today's same-spelling layers do

#### R3: `FAB_CLIS` env var with `FAB_PROVIDERS` alias
The environment override for the top-level table SHALL be `FAB_CLIS`; `FAB_PROVIDERS` SHALL still be read as a deprecated alias, folded per R2 with `FAB_CLIS` winning. `Layers.EnvOrigins` SHALL name the variable actually used.

- **GIVEN** `FAB_PROVIDERS='{custom: {headless_command: "t"}}'` is set and `FAB_CLIS` is not
- **WHEN** config loads
- **THEN** `clis.custom.headless_command` is `t` and its env origin is `FAB_PROVIDERS`

### Config: registry, fence and `explain`

#### R4: Registry rows use the new keys and carry the rename
`internal/configref` SHALL key the table row `clis` with `RenamedFrom: "providers"` (so `fab config upgrade`'s top-level carry rewrites a live `providers:` to `clis:`), keep the older `session_command`/`dispatch_command` rename history in the row's prose, accept `cli` as the canonical `agent.profiles.<role>` leaf and `provider` as a deprecated spelling in key matching, and rename the JSON defaults (`roleProfileDefault.Provider` → `CLI` with json `cli`; `providerDefault`/`providerProfileDefault` → `cliDefault`/`cliProfileDefault`). `defaultsmap.go` and `configscope` SHALL follow. The advertised fence prose SHALL use "CLI"/`<cli>` and the examples `{ cli: codex }`.

- **GIVEN** a project config with a live top-level `providers:` block and no `clis:`
- **WHEN** `fab config upgrade` runs
- **THEN** the key is carried to `clis:` with its value block verbatim and the fence re-renders with the new wording

#### R5: Fence digests stay upgradable
Because the advert text changes, the digests of the current (pre-rename) generated fence segments SHALL be added to `internal/configupgrade`'s legacy-digest list (the mechanism F1/#705 used), so an existing untouched fence is recognised as generated and re-rendered rather than treated as user-edited.

- **GIVEN** a config whose fence was rendered by 2.30.1
- **WHEN** `fab config upgrade` runs on the new binary
- **THEN** the fence is replaced with the new text and no user-edit warning is printed

#### R6: `fab config explain` and keyed `fab config show` answer both spellings
Keyed `fab config show <key>` (with or without `--origin`/`--json`) SHALL map a deprecated key (`providers…`, `agent.profiles.<role>.provider`) through `config.CanonicalKey` and answer for the new key, printing `fab: note: <old> is the deprecated spelling of <new>` on stderr only (stdout unchanged in shape). <!-- rework: review cycle 1 must-fix — keyed show of old keys errored -->

`fab config explain clis` SHALL explain the table; `fab config explain providers` (and `explain agent.profiles.<role>.provider` if keyed lookups reach leaves) SHALL resolve to the same entry with a one-line note that the key is the deprecated spelling of the new one. The explain prose SHALL say "CLI tool".

- **GIVEN** the new binary
- **WHEN** `fab config explain providers` runs
- **THEN** it exits 0 and prints the `clis` explanation plus `providers is the deprecated spelling of clis`

### Config: `set` / `unset` accept old keys

#### R7: `set` with an old key writes the new key and warns
`fab config set` (both project and `--system`) SHALL map a key starting `providers.` → `clis.` and `agent.profiles.<role>.provider` → `.cli` before validation and writing, print `<old> is deprecated, wrote <new>` on stderr, and exit 0. The value gate (known-CLI check, enum/range gates) SHALL run on the mapped key. When the target file holds the old spelling at the same path, the write SHALL remove it, so one spelling remains. A set of the new key also removes a same-path old-spelling twin in that file. The `.model` empty-value hint and shadow notices SHALL operate on the mapped key.

- **GIVEN** a system config with `agent.profiles.review.provider: codex`
- **WHEN** `fab config set --system agent.profiles.review.provider claude` runs
- **THEN** the file holds `agent.profiles.review.cli: claude`, no `.provider` line, stdout reports `Set agent.profiles.review.cli …`, and stderr carries `agent.profiles.review.provider is deprecated, wrote agent.profiles.review.cli`

#### R8: `unset` removes both spellings
`fab config unset` given either spelling SHALL remove both spellings at that path in the target file (warning as in R7 when the old key was named). It reports a change when either was removed; the "not set" notice fires only when neither was live.

- **GIVEN** a project config with `agent.profiles.review.provider: codex` only
- **WHEN** `fab config unset agent.profiles.review.cli` runs
- **THEN** the `.provider` line is gone and the command reports it unset

#### R9: `fab config validate` flags deprecated spellings
`fab config validate` (and the setup check that surfaces config issues, if it reports per-key issues) SHALL report each live deprecated spelling as a warning naming the new key and the migration, without failing validation.

- **GIVEN** a project config with `providers:`
- **WHEN** `fab config validate` runs
- **THEN** it emits a warning `providers is deprecated; rename to clis (fab-setup migrations)` and exits 0 when nothing else is wrong

### CLI: `fab agent`

#### R10: `--cli` flag with hidden deprecated `--provider`
`fab agent` SHALL take `--cli <name>` with today's `--provider` semantics (with a selector: refill from that CLI; alone: address `clis.<name>` directly). `--provider` SHALL remain as a hidden flag that behaves identically and prints `--provider is deprecated, use --cli` on stderr. Both supplied with different values SHALL be a usage error; equal values are accepted. Explicit-empty `--cli=` keeps today's explicit-empty semantics. Help text for `--workers`, `--model`, `--effort`, `-t`, `--headless`, and `fab agent models [<cli>]` SHALL say CLI.

- **GIVEN** `fab agent --provider codex --print`
- **WHEN** it runs
- **THEN** stdout equals `fab agent --cli codex --print`'s and stderr carries the deprecation line
- **GIVEN** `fab agent review --cli codex --provider claude -o yaml`
- **WHEN** it runs
- **THEN** it exits non-zero with a usage error naming both flags

#### R11: `-o yaml` emits `cli:` plus a deprecated duplicate `provider:`
`agent.Resolution` and `agent.Source` SHALL carry `cli` in the slot `provider` occupies today, immediately followed by `provider` holding the same value. Source values naming config keys SHALL use the `clis.` / `.cli` spellings. The bare-flag form's `kind:` value SHALL be `cli`. `fab resolve-agent`'s byte-stable `provider=` line SHALL stay unchanged.

- **GIVEN** `agent.profiles.review.cli: codex`
- **WHEN** `fab agent review -o yaml` runs
- **THEN** output contains `cli: codex` followed by `provider: codex`, and `source:` contains `cli: …` followed by `provider: …` with equal values, and `source.model` reads `clis.codex.profiles.review`
- **GIVEN** `fab agent --cli codex -o yaml`
- **THEN** `kind: cli`

### Code: identifiers and messages

#### R12: Go identifiers and user-facing strings say CLI
Go identifiers, comments, help text and error strings that denote a fab table entry SHALL be renamed (e.g. `ProviderConfig`→`CLIConfig`, `ResolveProvider`→`ResolveCLI`, `ProviderNames`→`CLINames`, `GetProvider`→`GetCLI`, `unknownProviderError`→`unknownCLIError`, `providerFills`→`cliFills`, `Overrides.Provider`/`ProviderSet`→`CLI`/`CLISet`, error `CLI %q has no %s; configure clis.%s.%s`). Where "provider" means a model vendor (kimi `[providers.<vendor>]`, provider 5xx/auth wording about the vendor's API), it SHALL stay. Every existing test SHALL pass with only spelling updates (no behaviour change).

- **GIVEN** the repo after the change
- **WHEN** `grep -rni provider src/go --include='*.go'` runs
- **THEN** every remaining hit is a deprecated-alias path, a test of one, `fab resolve-agent`'s `provider=` line, or a model-vendor meaning

#### R13: Every in-repo consumer of the output moves to `cli:`
Go callers (`dispatch_start.go`, `dispatch_open.go`, `pane_open.go`, `batch*.go`, `resolve_agent.go`, `setupcheck`, `agentmodels`) and kit skills that parse or quote `fab agent -o yaml` SHALL read/quote `cli` (not `provider`).

- **GIVEN** `src/kit/skills/`
- **WHEN** searching for "surface … `provider`" style instructions
- **THEN** every dispatch site says `cli`

### Migration

#### R14: Migration rewrites user config files
`src/kit/migrations/2.30.1-to-2.31.0.md` SHALL (modeled on `2.18.1-to-2.19.0.md`, with the placeholder-range note from `2.29.0-to-2.30.0.md`) sweep `fab/project/config.yaml` and `~/.fab-kit/config.yaml`: rename live top-level `providers:` → `clis:` (merging per leaf, new wins, when both are live), rename live `agent.profiles.<role>.provider:` (and `agent.tiers.<role>.provider:`) → `cli:`, skip when no live old key exists (idempotent), never touch commented fence lines, and report (not edit) a `FAB_PROVIDERS` environment variable. `src/kit/VERSION` SHALL NOT change.

- **GIVEN** both config files already on the new spelling
- **WHEN** the migration runs
- **THEN** it prints its skip line and edits nothing

### Docs: skills, specs, site

#### R15: Kit skills, help text and specs use the new vocabulary
`src/kit/skills/` (`_cli-fab.md`, `_cli-fab-pane.md`, `_cli-agents.md`, `_preamble.md`, `_pipeline.md`, `_srad.md`, `fab-setup.md`, `fab-ff.md`/`fab-fff.md` and every other skill quoting the YAML keys), `src/go/fab/cmd/fab/skill.md`, `docs/specs/` (`stage-models.md`, `config.md`, `harness-adapters.md`, `architecture.md`, `templates.md`, `skills.md`, `glossary.md`, `overview.md`/`user-flow.md` if they mention it) and `docs/site/` SHALL use `clis:`, `.cli`, `--cli`, `cli:`, "CLI"/"CLI tool". `_cli-fab.md` SHALL document `--cli`, the hidden `--provider` alias, the dual `cli:`/`provider:` output keys and their window, `kind: cli`, the unchanged `resolve-agent` `provider=` line, and `set`/`unset` old-key behaviour. `glossary.md` SHALL add a "CLI tool" entry and record the rename (the old `providers` entry becomes `clis`, noting `providers` is the pre-2.31 spelling still read). Historical text (existing migrations, memory logs, archived changes, plans) SHALL NOT be rewritten. Deployed skills SHALL NOT cite fab-kit-only paths.

- **GIVEN** `src/kit/skills/_preamble.md`
- **WHEN** read
- **THEN** its always-load bullet names "the `clis:` table" and the override table row reads "Cross-CLI `--cli`"

### Non-Goals

- Behaviour changes of any kind — precedence, rungs, fills and dispatch are untouched.
- Removing the old spellings — a later change after the one-minor-release window.
- Memory (`docs/memory/`) edits — hydrate owns them (including renaming `runtime/providers-and-profiles.md` → `runtime/clis-and-profiles.md`).
- rk/hexokit naming; releasing (gated on hexokit H5).

### Design Decisions

#### Normalize deprecated spellings at the raw-layer seam
**Decision**: One `internal/config` function folds `providers`/`.provider` into `clis`/`.cli` on every raw layer map (project, system, env) before `MergeLayers`, new winning per leaf.
**Why**: Config is merged as generic maps and decoded once (`FromMap`); normalizing the maps means the resolver, `show --origin`, `explain`, setup checks and `fab agent models` need no dual-spelling logic, and the typed `Config` needs no deprecated `Providers` field.
**Rejected**: Keeping a deprecated `Providers` struct field and merging in each getter (the 2.19 `SessionCommand` style) — it would leak into every consumer and into `show`, which renders raw maps.
*Introduced by*: 261120-f2ly-rename-provider-to-cli

#### Deprecation warnings only at write/validate seams
**Decision**: Reads are silent; `set`/`unset` with an old key, hidden `--provider`, and `fab config validate` warn.
**Why**: Skills parse `fab agent` output; per-call stderr noise risks breaking captures. User-confirmed at intake.
**Rejected**: Warning on every config load.
*Introduced by*: 261120-f2ly-rename-provider-to-cli

## Tasks

### Phase 1: Config core

- [x] T001 Rename the built-in table to `clis:` in `src/go/fab/defaults.yaml` (and `defaults.go` comment) <!-- R1 -->
- [x] T002 In `src/go/fab/internal/config/config.go`: rename `ProviderConfig`→`CLIConfig`, `ProviderProfile`→`CLIProfile`, `Config.Providers`→`CLIs` (yaml `clis`), `RoleProfile.Provider`→`CLI` (yaml `cli`), getters `GetProvider`/`ProviderNames`→`GetCLI`/`CLINames`; add the raw-layer normalizer (fold `providers`→`clis` per entry/leaf incl. `profiles.<role>`, `agent.profiles|tiers.<role>.provider`→`cli`, new wins, `native` presence preserved) and apply it to the project, system and env layer maps in `layersFrom`/`fromProjectMap`/`LoadPath`/`LoadNoProject` paths <!-- R2 -->
- [x] T003 Env tier: `FAB_CLIS` canonical via the registry key; read `FAB_PROVIDERS` as alias in `loadEnvLayer` with origin recorded (`internal/config/config.go`, `internal/configscope`) <!-- R3 -->
- [x] T004 Tests in `internal/config/config_test.go`: old-only, new-only, mixed per-leaf, mixed across tiers, `native: false` under old spelling, `agent.profiles.<role>.provider` alias, `FAB_PROVIDERS`/`FAB_CLIS` precedence and origin <!-- R2 -->

### Phase 2: Resolver, flag and output

- [x] T005 Rename identifiers across `src/go/fab/internal/agent/` (`agent.go`, `resolution.go`, `skill_prompt.go`), `internal/agentmodels/`, `internal/spawn/`, `internal/dispatch/`, `internal/pane/`, `internal/shellparse/`, `internal/setupcheck/`; source values use `clis.`/`.cli` spellings; update their tests <!-- R12 -->
- [x] T006 `Resolution`/`Source` in `internal/agent/resolution.go`: add `CLI` (yaml `cli`) in `provider`'s slot followed by deprecated `Provider` duplicate; keep `Lines()`'s `provider=` line; bare-flag `kind: cli` in `cmd/fab/agent.go`; tests for key order and duplicates <!-- R11 -->
- [x] T007 `cmd/fab/agent.go` + `agent_models.go`: `--cli` flag, hidden `--provider` alias with stderr warning and conflict usage error, `CLISet` explicit-empty semantics, help/Use text; tests <!-- R10 -->
- [x] T008 Remaining `cmd/fab/` callers (`resolve_agent.go`, `resolution.go`, `dispatch_start.go`, `dispatch_open.go`, `pane_open.go`, `pane_process.go`, `pane_ready.go`, `batch*.go`, `setup.go`, `setup_wizard.go`, `config_show_json.go`) and `cmd/fab/skill.md` move to the new names/vocabulary; tests updated <!-- R13 -->

### Phase 3: Registry, upgrade, set/unset, validate

- [x] T009 `internal/configref/configref.go` + `defaultsmap.go`: `clis` row with `RenamedFrom: "providers"` (older rename history kept in prose), `cli` leaf canonical with `provider` accepted as deprecated in key matching, JSON default structs renamed, fence/explain prose "CLI"/`{ cli: codex }`; `explain providers` resolves to `clis` with deprecation note; tests <!-- R4 --> <!-- R6 -->
- [x] T010 `internal/configupgrade/configupgrade.go`: add the pre-rename fence digests to the legacy list; golden/freeze fixtures updated; test that a 2.30.1-rendered fence re-renders and a live `providers:` carries to `clis:` <!-- R5 --> <!-- R4 -->
- [x] T011 `fab config set`/`unset` old-key mapping (`cmd/fab/config.go` and/or `internal/configupgrade/mutation.go`): map, warn on stderr, remove same-path old twin on set, remove both spellings on unset, mapped-key value gate; tests for project and `--system` <!-- R7 --> <!-- R8 -->
- [x] T012 `cmd/fab/config_validate.go` (and setupcheck config issues if per-key): warn on live deprecated spellings; tests <!-- R9 -->

### Phase 4: Migration and docs

- [x] T013 Write `src/kit/migrations/2.30.1-to-2.31.0.md` per R14 <!-- R14 -->
- [x] T014 Update kit skills in `src/kit/skills/` per R15 (`_cli-fab.md` full `fab agent`/config surface; `_preamble.md`; `_pipeline.md`; `_cli-fab-pane.md`; `_cli-agents.md`; `_srad.md`; `fab-setup.md`; `fab-fff.md`/`fab-ff.md`/`fab-continue.md`/`fab-adopt.md` wherever they say "surface … `provider`") <!-- R15 --> <!-- R13 -->
- [x] T015 Update `docs/specs/` (`stage-models.md`, `config.md`, `harness-adapters.md`, `architecture.md`, `templates.md`, `skills.md`, `glossary.md` incl. new "CLI tool" entry) and `docs/site/` per R15 <!-- R15 -->
- [x] T016 Run `go build ./...`, `go vet ./...` and `go test ./...` under `src/go/` (all modules), plus the residual `grep -rni provider` audit of `src/go`, `src/kit/skills`, `docs/specs`, `docs/site`; fix stragglers <!-- R12 -->

### Phase 5: Rework (review cycle 1)

- [x] T017 Keyed `fab config show <old key>` (incl. `--origin`, `--json`): map via `config.CanonicalKey` in `configShowCmd`/`renderShowKey` (`src/go/fab/cmd/fab/config.go`), stderr deprecation note; tests in `src/go/fab/cmd/fab/rename_compat_test.go` for `show providers`, `show providers.codex`, `show agent.profiles.review.provider` (+ `--origin`) <!-- R6 --> <!-- rework: review must-fix 1 -->
- [x] T018 Tests for `fab config unset --system` with the old key and with the new key against a system file holding the other spelling, and a direct `configupgrade.UnsetSystem` twin-key test (`src/go/fab/cmd/fab/rename_compat_test.go`, `src/go/fab/internal/configupgrade/rename_test.go`) <!-- R8 --> <!-- rework: review must-fix 2 (A-019) -->
- [x] T019 Should-fix cleanups: `src/go/fab/internal/config/deprecated.go` comments (:16, :163, :200) say `fab setup check`, not the nonexistent `fab config validate`, and stop listing validate as a `CanonicalKey` consumer; `src/kit/migrations/2.30.1-to-2.31.0.md` § Changes 1 rewords the both-live case as a normal half-migrated state (including the one `fab config set` leaves when only the same-path twin is removed), dropping the "YAML forbids two keys"/"same name" claims while keeping the merge instruction <!-- R14 --> <!-- rework: review should-fix -->
- [x] T020 Nice-to-have cleanups: say `fab setup check` (not bare `fab setup`) in `src/kit/skills/_cli-fab.md:638`, `docs/specs/stage-models.md:602` and the migration's Verification step 6; capitalize the user-facing label `CLI` in `src/go/fab/internal/setupcheck/probes.go:261` and the `src/go/fab/cmd/fab/config_validate.go:74` comment; restore sentence-initial `Providers advert` in the historical digest comments at `src/go/fab/internal/configupgrade/configupgrade.go:172-187`; make `TwinKey`/set/unset of `agent.profiles.<role>.cli` also remove a live legacy `agent.tiers.<role>.provider` in the same file (`src/go/fab/internal/config/deprecated.go:183-197`) with a test <!-- R7 --> <!-- rework: review nice-to-have -->

### Phase 6: Rework (review cycle 2)

- [x] T021 Docs sweep for T017/T020: in `src/kit/skills/_cli-fab.md` (~:638 and the `### fab config show` table ~:533-544) add keyed `fab config show <old key>` (+ `--origin`/`--json`) to the old-key seams — same stdout as the canonical key, stderr-only `fab: note:` — and narrow "show stays silent" to plain/unkeyed `show`; in `docs/specs/stage-models.md` (~:590-594) say `config.CanonicalKey` serves set, unset, `explain` and keyed `show`; in both, note that set/unset of `agent.profiles.<role>.cli` also removes a live legacy `agent.tiers.<role>.provider` <!-- R15 --> <!-- rework: review cycle 2 must-fix -->

## Execution Order

- T002 blocks T003–T012 (identifier renames compile against it)
- T005 blocks T006–T008
- T009 blocks T010–T012
- T013–T015 are independent of the Go phases but T014/T015 should describe the final behaviour from T006/T007/T011

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` uses `clis:` and `fab config show --json` emits built-ins under `clis.` only
- [x] A-002 R2: A config spelled entirely with `providers:`/`.provider` resolves byte-identically (apart from key names) to the same config spelled `clis:`/`.cli`, with empty stderr
- [x] A-003 R3: `FAB_CLIS` is the env override; `FAB_PROVIDERS` still works and `FAB_CLIS` wins when both are set
- [x] A-004 R4: The registry row is `clis` with `renamed_from: providers`; `fab config upgrade` carries a live `providers:` to `clis:`
- [x] A-005 R5: A 2.30.1-rendered fence re-renders on upgrade without a user-edit warning
- [x] A-006 R6: `fab config explain providers` exits 0 with the `clis` explanation and a deprecation note
- [x] A-007 R7: `fab config set` with an old key writes the new key, removes the old twin, warns `<old> is deprecated, wrote <new>`, exit 0
- [x] A-008 R8: `fab config unset` with either spelling removes both
- [x] A-009 R9: `fab config validate` warns on deprecated spellings without failing (seam is `fab setup check` per plan Assumption 6 — no `fab config validate` command exists; verified it warns for project + system files and `FAB_PROVIDERS`, Warn never Fail)
- [x] A-010 R10: `--cli` works; hidden `--provider` works and warns; conflicting values error
- [x] A-011 R11: `-o yaml` carries `cli:` then `provider:` (and in `source`), `kind: cli` for the bare form, `resolve-agent` `provider=` unchanged
- [x] A-012 R12: Go identifiers/messages renamed; residual "provider" hits are only aliases, vendor meanings, or `resolve-agent`'s line
- [x] A-013 R13: Go callers and kit skills consume `cli`
- [x] A-014 R14: Migration file exists with sweep of both scopes, idempotent sentinel, placeholder-range note; `src/kit/VERSION` unchanged
- [x] A-015 R15: Kit skills, `skill.md`, specs and site use the new vocabulary; glossary has "CLI tool" and records the rename

### Behavioral Correctness

- [x] A-016 R2: Precedence and resolution results are unchanged for every existing test fixture (only key spellings in tests changed)
- [x] A-017 R11: Key order of `-o yaml` keeps every pre-existing key in its relative order

### Scenario Coverage

- [x] A-018 R2: Unit tests cover old-only, new-only, mixed-in-layer, and mixed-across-tier configs, and `native: false` via the old spelling
- [x] A-019 R7: Tests cover set/unset with old and new keys in project and `--system` scopes (cycle 2: `TestConfigUnsetSystemRemovesBothSpellings` in rename_compat_test.go and `TestUnsetSystem_RemovesBothSpellings` in configupgrade/rename_test.go now cover `unset --system` with either spelling against a file holding the other)
- [x] A-020 R10: Tests cover `--provider` alias, warning, and conflict

### Edge Cases & Error Handling

- [x] A-021 R2: A layer with both spellings where the new leaf is empty falls back to the old leaf (empty-skip consistent with the cascade)
- [x] A-022 R7: Setting an old key whose mapped value fails the gate (unknown CLI) is refused with the mapped key named, writing nothing
- [x] A-023 R14: Migration handles a file with both `providers:` and `clis:` live without producing a duplicate top-level key

### Code Quality

- [x] A-024 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] A-025 No unnecessary duplication: Existing utilities reused where applicable (one normalizer, one key-mapping helper shared by set/unset/validate/explain)
- [x] A-026 Readability over cleverness: alias handling is localized and commented with the deprecation window
- [x] A-027 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-028 CLI ⇒ docs + tests: `_cli-fab.md` documents every changed signature and tests accompany every Go change
- [x] A-029 Migration for user-data restructuring: the config rename ships as a `src/kit/migrations/` file
- [x] A-030 Deployed content cites no fab-kit-only paths (the Go guard passes)
- [x] A-031 R6: Keyed `fab config show` of `providers`, `providers.<n>` and `agent.profiles.<role>.provider` (with and without `--origin`) answers for the new key with a stderr-only deprecation note, and tests cover it

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/internal/config/deprecated.go:187-189` (`TwinKeys`' deprecated-input branch) — unreachable in production: every caller (`configupgrade.Set`/`Unset`/`SetSystem`/`UnsetSystem`, mutation.go:57/103/136/252) runs `config.CanonicalKey` before `dropTwinPath`/`unsetBothSpellings`, so `TwinKeys` only ever sees canonical keys; only `TestCanonicalKeyAndTwins` exercises it.
- Otherwise none for existing code. (The compatibility shims it adds — `internal/config/deprecated.go`, the `FAB_PROVIDERS` source in `envSourcesForKey`, `cmd/fab/cli_flag.go`'s hidden `--provider`, the `Provider` duplicates in `agent.Resolution`/`agent.Source` and `paneOpenJSON`, and `setupcheck.ProbeDeprecatedSpellings` — are *planned* removals once the one-minor-release window closes, not redundancies today.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Normalize deprecated spellings on raw layer maps before merge, dropping the typed deprecated `Providers` field | Config merges generic maps then decodes once; one seam covers resolver, show and explain | S:60 R:75 A:75 D:65 |
| 2 | Confident | `set` of the new key also removes a same-path old-spelling twin in the target file | Keeps one spelling per file; symmetric with the old-key set path | S:55 R:85 A:70 D:65 |
| 3 | Confident | `fab config validate` is the read-side warning seam (not every load) | Intake #7, user-confirmed | S:85 R:80 A:80 D:80 |
| 4 | Confident | Memory edits deferred to hydrate | Pipeline convention: hydrate writes docs/memory | S:80 R:90 A:90 D:85 |
| 5 | Confident | `ProviderProfile` renamed `CLIProfile` alongside `ProviderConfig` → `CLIConfig` | Same table, same rename rule | S:65 R:85 A:80 D:75 |
| 6 | Confident | No `fab config validate` command exists (`config_validate.go` is the `set` value gate), so R9's read-side warning seam is `fab setup` (the `setupcheck` doctor): a Warn finding per live deprecated spelling in either config file plus `FAB_PROVIDERS`, never a Fail | The brief's "validate/setup checks" names two seams; only the setup check exists. Adding a new `validate` verb would be scope creep for a pure rename | S:70 R:85 A:75 D:70 |
| 7 | Confident | `fab resolve-agent` and `fab pane open` get `--cli` plus the same hidden `--provider` alias (one shared helper, `cmd/fab/cli_flag.go`); `fab pane open --json` dual-emits `"cli"` and `"provider"` | The brief names only `fab agent`, but both commands expose the same flag today and external callers may use it; dropping it would break them with no window. `resolve-agent`'s `provider=` output line stays byte-stable | S:65 R:80 A:70 D:65 |
| 8 | Confident | Compatibility is a raw-layer normalizer (`config.NormalizeDeprecated`) and one key-mapping helper (`config.CanonicalKey`/`TwinKey`/`DeprecatedUses`); the typed `Config` carries no deprecated `Providers` field and `RoleProfile` no `Provider` | Design Decision above; every consumer then sees one shape | S:70 R:75 A:80 D:70 |
| 9 | Confident | `FAB_PROVIDERS` is folded as a second env source for the `clis` registry key (`EnvOrigins["clis"]` names `FAB_PROVIDERS` when only it is set, `FAB_CLIS` otherwise); a malformed alias warns naming `$FAB_PROVIDERS` and is skipped | Same fail-open rule as every env override | S:70 R:80 A:75 D:70 |
| 10 | Confident | The `fab config upgrade` fence digests: the 2.30.1 agent/providers advert digests were already catalogued (the catalog is append-only and self-enforcing), so only the two renamed advert digests are appended; a 2.30.1-rendered fence fixture (`testdata/config-2.30.1-providers.yaml`, generated by the pre-change binary) proves it re-renders without a user-edit notice | The fence body is always regenerated; the digest list governs legacy unfenced paragraphs | S:70 R:85 A:80 D:75 |
| 11 | Confident | Historical byte-exact test fixtures (`legacyV2198ToV2220ProvidersAdvert`, the digest catalog) keep "provider"; the two "mixed paragraph" tests count one historical (`explain providers`) plus one regenerated (`explain clis`) advert | Rewriting a historical fixture would falsify the digest it pins | S:75 R:90 A:85 D:80 |
| 12 | Confident | `fab config unset` warning text is `<old> is deprecated, unset <new> (both spellings are removed)`; `set`'s is `<old> is deprecated, wrote <new>`, both `fab: warning:`-prefixed on stderr | Brief fixes the set text; unset's wording is unspecified | S:60 R:90 A:75 D:65 |
| 13 | Confident | `fab config explain <deprecated key>` prints the canonical explanation with a leading `# <old> is the deprecated spelling of <new>` comment (YAML) or the same line on stderr (`--json`, keeping stdout valid JSON) | Keeps keyed lookups of old names working per R6 without breaking JSON consumers | S:65 R:85 A:75 D:70 |

13 assumptions (0 certain, 13 confident, 0 tentative).
