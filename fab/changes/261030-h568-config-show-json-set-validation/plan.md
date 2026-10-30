# Plan: fab config show --json, set validation, and three config bug fixes

**Change**: 261030-h568-config-show-json-set-validation
**Intake**: `intake.md`

## Requirements

### Config CLI: `fab config show --json`

#### R1: `--json` is a format switch over the plain show forms
`fab config show` SHALL accept a boolean `--json` flag. Without `--origin`: bare `show --json` MUST print the fully composed effective config (the same tree the YAML form prints) as one JSON object; keyed `show <key> --json` MUST print the key's effective value as a JSON value (string/number/bool/array/object/`null`). Values MUST be typed (an int prints `45`, never `45.0` or `"45"`; a bool prints `true`). Output is one JSON document plus a trailing newline, indented like `fab config explain --json`. Unknown keys error non-zero exactly as today with nothing on stdout.

- **GIVEN** a project with no overrides
- **WHEN** `fab config show dispatch.column_width --json` runs
- **THEN** stdout is `45` followed by a newline
- **AND** `fab config show --json` parses as a JSON object whose `dispatch.reap_done` is the JSON bool `true`

#### R2: `--origin --json` emits winner-only typed leaf rows
`fab config show --origin --json` SHALL print a JSON array of rows, one per leaf, in the same order and over the same leaves as the text `--origin` form (map-valued fields `agent.profiles`, `providers` drill down per leaf). Each row MUST carry `key` (dotted string), `value` (typed), `origin` (exactly one of `default`, `env`, `system`, `project`), `origin_detail` (env variable name without `$`, absolute file path for system/project, `""` for default — always present), and `status` (`"effective"` for every row in this form). A row MAY carry `derived_from` (R8); the field is omitted otherwise.

- **GIVEN** cwd outside any fab project, `~/.fab-kit/config.yaml` with `agent:\n  workers: agy`, and `FAB_AGENT_SESSION=codex`
- **WHEN** `fab config show --origin --json` runs
- **THEN** the `agent.workers` row is `{"key":"agent.workers","value":"agy","origin":"system","origin_detail":"<home>/.fab-kit/config.yaml","status":"effective"}`
- **AND** the `agent.session` row has `origin` `env` and `origin_detail` `FAB_AGENT_SESSION`
- **AND** the `dispatch.min_cols` row has `value` `50` (a JSON number), `origin` `default`, `origin_detail` `""`

#### R3: keyed `--origin --json` emits the full tier stack
`fab config show <key> --origin --json` SHALL print a JSON array of rows (R2 schema) forming the key's full stack: for a scalar/list key, one row per defining tier, highest precedence first, the first `status: "effective"` and the rest `"shadowed"`; for a map-valued key, the per-leaf drill-down with each leaf contributing its own stack rows in the text form's order. A key no tier defines MUST print `[]`. A descendant whose ancestor is replaced by a higher tier's non-map MUST print exactly one row: the requested key, `value: null`, the ancestor tier's `origin`/`origin_detail`, `status: "effective"`.

- **GIVEN** system `agent.workers: agy` and `FAB_AGENT_WORKERS=codex`
- **WHEN** `fab config show agent.workers --origin --json` runs
- **THEN** it prints three rows in order: env/`FAB_AGENT_WORKERS`/codex/effective, system/`<path>`/agy/shadowed, default/`""`/claude/shadowed

#### R10: text output is unchanged
Every existing `show` form without `--json` MUST produce byte-identical output to the pre-change binary (the existing show tests pass unmodified).

- **GIVEN** the existing `config_show_init_test.go` / `config_test.go` show/origin tests
- **WHEN** the suite runs after the refactor
- **THEN** they pass without edits to their expected strings

### Config CLI: `fab config set` validation

#### R4: value-domain validation for enum and ranged keys
After the existing kind check, a `set` (project or `--system`, and the `fab setup` wizard's writes) MUST refuse — non-zero exit, nothing written — a value outside the key's domain, with an error naming the key, the value, and the accepted set or range:
- `dispatch.mode`: one of `pane`, `native`, `headless`
- `dispatch.column_width`: integer 1..99
- `dispatch.min_cols`, `dispatch.min_rows`: integer ≥ 1
- `autopilot.merge_mode`: one of `config.ValidAutopilotMergeModes`
- `dispatch.reap_done`: bool (already enforced by the kind check; regression-tested)

The accepted sets/ranges MUST come from one Go source each, shared with the read-path accessors (`GetDispatchMode`, `GetDispatchColumnWidth`, `GetDispatchMinCols`/`MinRows`) — no second literal.

- **GIVEN** any cwd and an isolated HOME
- **WHEN** `fab config set --system dispatch.mode weird` runs
- **THEN** it exits non-zero with an error naming `dispatch.mode`, `weird`, and `pane, native, headless`
- **AND** `~/.fab-kit/config.yaml` is not created/modified
- **AND** `fab config set --system dispatch.column_width 500` and `… min_rows 0` are likewise refused, while `column_width 99` and `min_cols 1` succeed

#### R5: provider-name validation against the effective providers table
`set` of `agent.session`, `agent.workers`, or `agent.profiles.<role>.provider` MUST refuse a value that is not in the effective providers table — `agent.ProviderNames(cfg)` (built-ins ∪ `providers:` keys) computed over the same cascade the read path uses from the cwd (project ⊕ system ⊕ env inside a project; system ⊕ env outside). The error MUST list the available names. When the read model cannot be loaded (e.g. a malformed project file), validation MUST fall back to the project-less cascade (system ⊕ env ∪ built-ins) rather than failing the write for that reason alone. The read path stays provider-neutral (no change to resolution or `fab agent`).

- **GIVEN** cwd outside a project, isolated HOME
- **WHEN** `fab config set --system agent.workers bogus` runs
- **THEN** it exits non-zero with an error naming `agent.workers`, `bogus`, and the available providers (e.g. `agy, claude, codex, kimi`), and writes nothing
- **AND** `fab config set --system agent.workers codex` succeeds
- **AND** inside a project whose config defines `providers.mytool`, `fab config set agent.profiles.review.provider mytool` succeeds

### Config runtime: bug fixes

#### R6: invalid `dispatch.mode` never panics
`(*Config).GetDispatchMode` MUST emit its invalid-mode warning through the package's `warnf` (nil `warnw` ⇒ `os.Stderr`) and return the default. No call site in `internal/config` may write to `warnw` directly.

- **GIVEN** `~/.fab-kit/config.yaml` containing `dispatch:\n  mode: weird` and `warnw` nil (production state)
- **WHEN** `fab agent review -o yaml` runs (or `GetDispatchMode` is called)
- **THEN** it does not panic, prints `fab: warning: invalid dispatch.mode "weird"; using "native"` on stderr, and resolves with the default mode

#### R7: shadow/live-tier notices work outside a fab project
`set`'s shadow warning and `unset`'s live-tier notice MUST resolve the read model with the same project-or-projectless layer loading `show` uses (`ErrNoFabRoot` ⇒ `config.LoadLayersNoProject()`), via one shared helper. They stay fail-open on genuine load errors.

- **GIVEN** cwd outside any fab project and `FAB_AGENT_WORKERS=codex`
- **WHEN** `fab config set --system agent.workers agy` runs
- **THEN** it exits 0 and stderr carries `fab: warning: agent.workers is shadowed by env $FAB_AGENT_WORKERS — the written value is not in effect`
- **AND** from the same cwd, with `FAB_AGENT_SESSION=codex` and no `agent.session` in the system file, `fab config unset --system agent.session` exits 0 with a notice naming `env $FAB_AGENT_SESSION`

### Config CLI: derived role provenance

#### R8: `derived_from` on derived default-tier role rows (JSON only)
In both `--origin --json` forms, a row for `agent.profiles.<role>.provider|model|effort` whose tier is `default` SHALL carry `derived_from: {key, origin, origin_detail}` naming the config leaf that actually determined the derived value and that leaf's own winning tier — provider: `agent.workers` / `agent.session` (from `agent.ResolveRoleWithSource` against the knobs-only config); model/effort: the provider-fill leaf path, e.g. `providers.agy.profiles.doing.model` (against the fills-only config). It MUST be omitted when the source is `built-in`, when the source key is defined by no tier, or when the source key's own winning tier is `default` (no user config involved — `origin: default` is then the whole truth). Rows won by env/system/project never carry it. The text `--origin` output is unchanged (follow-up).

- **GIVEN** cwd outside a project and `agent.workers: agy` in the system file
- **WHEN** `fab config show --origin --json` runs
- **THEN** the `agent.profiles.doing.provider` row has `value` `agy`, `origin` `default`, and `derived_from` `{"key":"agent.workers","origin":"system","origin_detail":"<home>/.fab-kit/config.yaml"}`
- **AND** with no config anywhere, no `agent.profiles.*` row carries `derived_from`

### Documentation

#### R9: contracts documented at their owners
`src/kit/skills/_cli-fab.md` § fab config MUST document `show --json` (four forms, row schema incl. optional `derived_from`, `[]` and ancestor-null edge cases) and the `set` value-domain rules; its stale "Requires a fab repo" claim for `show` MUST be corrected to the project-less degrade. `docs/specs/config.md` gets a minimal design-intent addition. The cobra help for `show` (`--json` example) and `set` (validation sentence) MUST be updated.

- **GIVEN** the change is applied
- **WHEN** a reader opens `_cli-fab.md` § fab config show
- **THEN** the JSON row schema and the set validation rules are stated there, with no fab-kit-only path citations

### Non-Goals

- Relabelling derived rows in the TEXT `--origin` output — follow-up (intake §5)
- Read-path validation of provider names or ranges — read path stays fail-open and provider-neutral
- Validating `unset` values — `unset` is the repair path and never inspects values
- Collapsing `effectiveTierFor`'s second loader pass (duplicate warnings on a malformed system file) — pre-existing, out of scope

### Design Decisions

#### JSON is a format switch over one provenance walk
**Decision**: the provenance walk produces structured rows (key, raw value, tier, status, optional derived_from); a text renderer and a JSON renderer consume the same rows.
**Why**: text and JSON cannot diverge on leaves, order, or edge cases; text stays byte-identical.
**Rejected**: a second, JSON-only walk — two copies of the empty-skip/ancestor-replacement logic would drift.
*Introduced by*: 261030-h568-config-show-json-set-validation

#### Validation is write-time only, via one cmd-level helper
**Decision**: `fab config set` and the `fab setup` wizard call one shared value-domain validator in `cmd/fab` before `configupgrade.Set`/`SetSystem`; the read path keeps warn-and-default.
**Why**: the provider table depends on cwd cascade discovery, which `configupgrade` must not own; a hand-edited bad value must still not brick every repo.
**Rejected**: validating in the loader (would brick repos on a bad personal file); validating inside `configupgrade` (would need cwd/project discovery there).
*Introduced by*: 261030-h568-config-show-json-set-validation

#### `derived_from` instead of re-labelling `origin`
**Decision**: derived default-tier role rows keep `origin: "default"` (the tier that holds the row) and add `derived_from` naming the determining key and its tier.
**Why**: `origin` keeps one meaning (which tier supplies this leaf) across all rows; consumers that want the "why" read `derived_from`.
**Rejected**: rewriting `origin` to the knob's tier — would make the default tier's shadowed rows in keyed stacks inconsistent and break the effective/shadowed semantics.
*Introduced by*: 261030-h568-config-show-json-set-validation

## Tasks

### Phase 1: Setup

- [x] T001 [P] `src/go/fab/internal/config/config.go`: export `ValidDispatchModes = []string{"pane","native","headless"}` (beside `ValidAutopilotMergeModes`) and named bounds for the dispatch ints (e.g. `MinDispatchColumnWidth = 1`, `MaxDispatchColumnWidth = 99`, `MinDispatchFloor = 1`); rewrite `GetDispatchMode`'s switch to use `ValidDispatchModes`, route its warning through `warnf`, and make `GetDispatchColumnWidth`/`MinCols`/`MinRows` read the named bounds. Grep `internal/config` for any other direct `warnw` write. Add a regression test in `internal/config/config_test.go` that leaves `warnw` nil, swaps `os.Stderr` to capture, and asserts no panic + the warning text + default return <!-- R6 -->
- [x] T002 [P] `src/go/fab/cmd/fab/config.go`: extract the show command's layer loading (FabRoot → `config.LoadLayers`; `resolve.ErrNoFabRoot` → `config.LoadLayersNoProject()`; other error → error) into one helper (e.g. `loadReadLayers() (*config.Layers, error)`) used by `configShowCmd` and `effectiveTierFor` (which stays fail-open on error) <!-- R7 -->

### Phase 2: Core Implementation

- [x] T003 `src/go/fab/cmd/fab/config.go` (or a new sibling `config_validate.go`): add `validateSetValue(key, rawValue string) error` implementing R4 (enum/range against the T001 symbols and `config.ValidAutopilotMergeModes`; ints parsed from the already-kind-checked value) and R5 (provider keys — `agent.session`, `agent.workers`, `agent.profiles.<role>.provider` — against `agent.ProviderNames(cfg)` where `cfg` is `config.FromMap` of the `loadReadLayers` effective map, falling back to `config.LoadLayersNoProject()` when loading fails). Errors: `invalid value "weird" for dispatch.mode — must be one of: pane, native, headless` style, naming key + value + accepted set/range <!-- R4 -->
- [x] T004 Wire `validateSetValue` into `configSetCmd` (after the empty-value guard, before any write) and into `setupWizard.writeOne` in `src/go/fab/cmd/fab/setup_wizard.go` before `configupgrade.Set`/`SetSystem`. Keep the kind-check error ordering: a kind mismatch (e.g. `column_width abc`) must still report the existing kind error — run `configupgrade`'s kind check first or make `validateSetValue` skip values that don't parse as the key's kind <!-- R5 -->
- [x] T005 `src/go/fab/cmd/fab/config.go`: refactor the provenance walk (`originLine`, `cascadeTier.origin/stackOrigin`, `flattenOrigin`, `stackLines`, `keyedOriginLines`, `renderOriginLines`) so each row keeps the raw value, tier kind, origin detail (env var name w/o `$`, path, or ""), and status; the text renderer reproduces today's strings byte-for-byte (`renderScalar` for winner-only values, `renderCompactValue` for keyed stack values, `$VAR`/path/`default` labels, `  (effective)`/`  (shadowed)` markers, the undefined and ancestor lines) <!-- R10 -->
- [x] T006 `src/go/fab/cmd/fab/config.go`: add `--json` to `configShowCmd` and JSON renderers for all four forms (R1 plain/keyed value; R2 winner-only rows; R3 keyed stack rows, `[]` for undefined, single null row for ancestor-replaced). Ensure typed values (float64 integral defaults encode as ints — `encoding/json` does this for float64 45 → `45`; verify) and that `map[any]any` values are normalized via `asGenericMap` before marshalling. Match `configref.RenderJSON`'s indentation <!-- R1 -->
- [x] T007 `src/go/fab/cmd/fab/config.go`: implement `derived_from` (R8) — for `default`-tier rows of `agent.profiles.<role>.{provider,model,effort}`, resolve the source via `agent.ResolveRoleWithSource` on the same knobs-only (provider) / fills-only (model, effort) projections `configref.DefaultsMapFor` uses (export small helpers from `configref` if needed rather than duplicating them), map the source to a leaf key (`agent.workers`, `agent.session`, `providers.<p>.profiles.<r>.model|effort`), and resolve that key's winning tier through `descendPath` over the ALREADY-loaded layers. Omit when source is `built-in`/`flag`, the key is undefined, or its winner is `default` <!-- R8 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Tests in `src/go/fab/cmd/fab/config_show_init_test.go` (HOME isolated via `setupConfigRepo`/a project-less variant using `chdirTestEnv` to a bare temp dir — never the real `~/.fab-kit`): R1 plain/keyed JSON typing; R2 row schema incl. env/system/default details from a project-less cwd; R3 stack order + `[]` + ancestor-null; R4 each refused value writes nothing + boundary values accepted + reap_done bool regression; R5 bogus refused with available list, built-in accepted, project-defined provider accepted, malformed-project fallback still accepts `codex --system` (keep `TestConfigSet_ShadowWarning`'s fail-open subtest green); R7 project-less `set --system` shadow warning and `unset --system` env live-tier notice; R8 derived_from present for system knob / absent with no config; wizard path refuses an invalid value <!-- R3 -->
- [x] T009 Run `gofmt -l`, `go vet`, and `go test ./cmd/fab/... ./internal/config/... ./internal/configref/... ./internal/configupgrade/...` from the module root `src/go/fab`; then the full `go test ./...` <!-- R10 -->

### Phase 4: Polish

- [x] T010 [P] `src/go/fab/cmd/fab/config.go` cobra help: `show` Long/Example gain `--json` (incl. `fab config show --origin --json`); `set` Long gains the value-domain validation sentence (keep the shadow-warning sentence). Check `shll standards` for help/`--json` conventions if `shll` is available <!-- R9 -->
- [x] T011 [P] `src/kit/skills/_cli-fab.md` § fab config: signature block (`show [key] [--origin] [--json]`), the show mode table gains the JSON forms, a row-schema subsection (fields, `origin` vocabulary, `origin_detail`, `status`, optional `derived_from`, `[]`/ancestor-null edge cases, a JSON example), fix "Requires a fab repo" → project-less degrade (env > system > defaults); § fab config set / unset gains the value-domain rules (accepted sets/ranges, provider table definition, write-time only) and notes notices work outside a project. Also sweep `src/go/fab/cmd/fab/skill.md` if it restates the show forms <!-- R9 -->
- [x] T012 [P] `docs/specs/config.md`: minimal additions next to the `fab config show` bullet (JSON is the machine contract; one schema; `derived_from`) and the `set` description (write-time value-domain validation; read path stays fail-open) <!-- R9 -->

## Execution Order

- T001 and T002 first (T003 depends on both); T005 before T006 before T007; T008 after T003–T007; T009 last among code tasks

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `show --json` prints a JSON object of the composed config and `show <key> --json` prints a typed JSON value
- [ ] A-002 R2: `show --origin --json` rows carry exactly `key`, typed `value`, `origin` ∈ {default, env, system, project}, `origin_detail`, `status: effective` (plus optional `derived_from`)
- [ ] A-003 R3: keyed `--origin --json` lists the full stack winner-first with effective/shadowed status
- [ ] A-004 R4: `dispatch.mode`, `column_width`, `min_cols`, `min_rows`, `autopilot.merge_mode` out-of-domain values are refused non-zero with nothing written
- [ ] A-005 R5: provider keys are validated against built-ins ∪ cascaded `providers:` keys, error lists available names
- [ ] A-006 R6: `GetDispatchMode` with an invalid mode and nil `warnw` warns on stderr and returns the default — no panic
- [ ] A-007 R7: `set --system` shadow warning and `unset --system` live-tier notice fire from a project-less cwd
- [ ] A-008 R8: derived default-tier role rows carry `derived_from` with the knob/fill key and its tier
- [ ] A-009 R9: `_cli-fab.md`, `docs/specs/config.md`, and cobra help document `--json` and set validation

### Behavioral Correctness

- [ ] A-010 R10: all pre-existing show/origin text tests pass unmodified
- [ ] A-011 R4: the read-path accessors and the set validator share one source for each accepted set/range (no duplicate literal)

### Scenario Coverage

- [ ] A-012 R2: a test exercises project-less cwd with env + system + default rows and checks `origin_detail` for each
- [ ] A-013 R5: a test covers the `fab setup` wizard write path refusing an invalid value
- [ ] A-014 R7: a test covers project-less `set --system` shadowed by an env var

### Edge Cases & Error Handling

- [ ] A-015 R3: undefined key prints `[]`; ancestor-replaced descendant prints one `value: null` effective row
- [ ] A-016 R5: a malformed project file does not make `set --system agent.workers codex` fail (fallback provider table)
- [ ] A-017 R4: a kind mismatch (`column_width abc`, `reap_done bogus`) still reports the existing kind error
- [ ] A-018 R8: no config anywhere ⇒ no `derived_from`; a role with its own `agent.profiles.<role>.provider` override ⇒ provider row won by that tier carries no `derived_from`

### Code Quality

- [ ] A-019 Pattern consistency: new code follows naming and structural patterns of `cmd/fab/config.go` and `internal/config`
- [ ] A-020 No unnecessary duplication: one provenance walk, one layer loader, one validator, existing `configref` knob/fill projections reused
- [ ] A-021 Readability: no new god function (>50 lines without clear reason); no magic strings/numbers for modes or bounds
- [ ] A-022 CLI ⇒ docs + tests: `_cli-fab.md` updated and every Go change has tests; tests isolate HOME
- [ ] A-023 Deployed content cites no fab-kit-only paths (`_cli-fab.md` restates rules, no `docs/specs`/`src/go` pointers)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Validation also covers the `fab setup` wizard's writes via the same helper | Wizard documents itself as writing "through the same surgical … write `fab config set` performs"; sibling writer | S:70 R:80 A:80 D:75 |
| 2 | Confident | On read-model load failure, provider validation falls back to the project-less cascade | Preserves the existing fail-open write test (malformed project + `--system` write) and never refuses a built-in | S:60 R:80 A:75 D:65 |
| 3 | Confident | `derived_from` omitted when the source key's own winner is `default` (not just `built-in`) | "Default derived from a default" carries no user signal; keeps the field meaning "user config drove this" | S:55 R:80 A:70 D:60 |
| 4 | Certain | Text output stays byte-identical; JSON renders from the same structured rows | R10; existing tests pin the text | S:90 R:85 A:90 D:90 |

4 assumptions (1 certain, 3 confident, 0 tentative).
