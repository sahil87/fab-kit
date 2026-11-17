# Plan: fab agent models — ask each tool for its models

**Change**: 261117-rhpz-agent-models-discovery
**Intake**: `intake.md`

## Requirements

### CLI: `fab agent models`

#### R1: Subcommand surface
`fab agent models [provider] [--json] [--repo <path>]` SHALL be registered as a cobra child of `agentCmd()` and list the models each provider (agent CLI tool) offers. With no positional it MUST cover every name from `agent.ProviderNames(cfg)` in that (sorted) order; with one positional it MUST cover only that provider; more than one positional is a usage error. An unknown provider name MUST exit non-zero with an error naming the available providers (same wording family as `fab agent --provider`'s lookup failure). `--repo` MUST reuse `loadRepoConfig`, so the verb works outside a fab project via the project-free cascade. Adding the child MUST NOT change any existing `fab agent` form.

- **GIVEN** a directory with no `fab/` anywhere up the tree
- **WHEN** `fab agent models --json` runs
- **THEN** it exits 0 and prints rows for `agy`, `claude`, `codex`, `kimi`
- **AND** `fab agent models nope` exits non-zero naming `agy, claude, codex, kimi`
- **AND** `fab agent review -o yaml`, `fab agent --print apply`, and `fab agent --print -- --resume` behave exactly as before

#### R2: Output formats and streams
`--json` SHALL print exactly one JSON array of row objects to stdout (no envelope, `efforts` always an array, never `null`). Without `--json`, stdout SHALL be a column-aligned table with header `CLI  MODEL  LABEL  EFFORTS  SOURCE  DEFAULT` (efforts comma-joined, `*` in DEFAULT for the tool default). Discovery warnings MUST go to stderr, one line each, as `warning: <provider>: <reason>`, in both modes. Discovery failures MUST NOT change the exit code (0); only usage errors, unknown provider, and config-load errors exit non-zero. The command is read-only and caches nothing.

- **GIVEN** `codex` is not on `$PATH`
- **WHEN** `fab agent models codex --json` runs
- **THEN** stdout is a valid JSON array holding codex's kit fills with `source: "fill"`
- **AND** stderr carries one `warning: codex: …` line, and the exit code is 0

### Discovery: row model and merge

#### R3: Row schema
Each row SHALL be `{cli, id, label, efforts, default, source}`: `cli` = provider name, `id` = the string fab would substitute into `{model}`, `label` = tool display name falling back to `id`, `efforts` = per-model list when the tool reports one else the tool's known set else `[]`, `default` = true only for the tool's own default model, `source` ∈ {`tool`, `alias`, `fill`}.

- **GIVEN** a codex discovery row with `display_name` `GPT-6.1 Sol`
- **WHEN** rows are emitted
- **THEN** the row reads `{"cli":"codex","id":"gpt-6.1-sol","label":"GPT-6.1 Sol",…,"source":"tool"}`

#### R4: Merge order, fills, dedupe, default
Per provider, rows SHALL be assembled as (1) adapter/`models_command` rows in tool order, (2) kit fills — every distinct non-empty `model` across `agent.ResolveProvider(cfg, name).Profiles` (user fill overrides included, `agent.profiles` pins excluded), sorted, `source: fill`, (3) the tool-reported default when no row has that id, appended as `{source: tool, default: true}`. Rows MUST be de-duplicated by `id`: the first occurrence keeps its position, the higher source wins (`tool` > `alias` > `fill`), `default` is OR-ed, and an empty label/efforts on the winner is filled from the duplicate. Fills MUST always be present, so the list is never smaller than the fill set.

- **GIVEN** claude settings with no `modelPicker` and top-level `model: claude-fable-5-1[1m]`
- **WHEN** claude rows are built
- **THEN** rows are `opus`, `sonnet`, `haiku`, `fable` (alias; the `opus`/`sonnet` fills merge into them) followed by `claude-fable-5-1[1m]` (tool, default)

### Discovery: built-in adapters

#### R5: codex adapter
The codex adapter SHALL run `codex debug models` (binary from `$PATH`) under a timeout and parse ONLY: top-level `models[]`, and per model `slug`, `display_name`, `visibility`, `supported_reasoning_levels[].effort`. It MUST keep only `visibility == "list"`. The default SHALL be the top-level `model` key of `$CODEX_HOME/config.toml` (`CODEX_HOME` default `~/.codex`); a missing config file yields no default and no warning; a config read error yields a warning. Binary absent, non-zero exit, timeout, or malformed/`models`-less JSON MUST yield a warning plus fills.

- **GIVEN** the recorded fixture with `gpt-reserve` (`hide`) and `gpt-6-astra` carrying an `ultra` effort
- **WHEN** the adapter parses it
- **THEN** `gpt-reserve` is absent and `gpt-6-astra`'s efforts end with `ultra`

#### R6: claude adapter
The claude adapter SHALL read `settings.json` and `settings.local.json` from `$CLAUDE_CONFIG_DIR` (default `~/.claude`), with the local file's top-level `model` and `modelPicker` replacing the base file's when present. Rows: the aliases `opus`, `sonnet`, `haiku`, `fable` (`source: alias`) unless `modelPicker.replaceBuiltInOptions` is `true`, then each `modelPicker.options[]` (`id` = `.model`, `label` = `.label`, `source: tool`). Default = top-level `model`. Every claude row's efforts = `low, medium, high, xhigh, max`. Missing files MUST NOT warn; a malformed file MUST warn and still yield the aliases.

- **GIVEN** `replaceBuiltInOptions: true` and two options
- **WHEN** claude rows are built
- **THEN** no alias rows appear except where a fill re-adds `opus`/`sonnet` with `source: fill`

#### R7: kimi adapter
The kimi adapter SHALL read `~/.kimi-code/config.toml`: each `[models."<alias>"]` (or bare `[models.<alias>]`) table becomes a row (`id` = alias, `label` = the table's `display_name`, `efforts` = the table's `support_efforts`, `source: tool`); default = top-level `default_model`. Other tables MUST be ignored, and no value outside model tables (e.g. `[providers.*]` credentials) may appear in rows or warnings. A missing file MUST NOT warn.

- **GIVEN** a config with `default_model`, two model tables, and a `[providers.openrouter]` table with `api_key`
- **WHEN** kimi rows are built
- **THEN** exactly the two models appear (one marked default if it matches) and the api key appears nowhere in output

#### R8: agy and unknown providers
agy SHALL have no adapter: fills only and no warning. A user-defined provider without `models_command` SHALL likewise yield its fills only, with no warning.

- **GIVEN** the built-in agy
- **WHEN** `fab agent models agy --json` runs
- **THEN** stdout holds agy's fills as `source: fill` and stderr is empty

### Config: `providers.<name>.models_command`

#### R9: models_command
`config.ProviderConfig` SHALL gain `ModelsCommand` (`yaml:"models_command"`), merged per field by `agent.ResolveProvider`. When set on ANY provider it MUST replace the built-in adapter: run via `sh -c` under the same timeout; stdout is a JSON array of `{id, label?, efforts?, default?}`; fab MUST overwrite `cli` (provider name) and `source` (`tool`) and drop rows with empty `id`. Non-zero exit, timeout, or malformed JSON MUST yield a warning plus fills. `fab config set providers.<name>.models_command <cmd>` MUST be accepted (configref dynamic key match), and `fab config explain providers` MUST document the field. No migration, no managed-fence advert.

- **GIVEN** `providers: {mytool: {interactive_command: "mytool", models_command: "printf '[{\"id\":\"m1\",\"cli\":\"x\",\"source\":\"fill\"}]'"}}`
- **WHEN** `fab agent models mytool --json` runs
- **THEN** the row reads `{"cli":"mytool","id":"m1","label":"m1","efforts":[],"default":false,"source":"tool"}`

### Docs

#### R10: Documentation
`src/kit/skills/_cli-fab.md` § fab agent SHALL document `fab agent models` (signature, row schema, merge/dedupe, adapters and config locations, fail-open + stderr warnings, exit codes, `models_command`) without citing fab-kit-only paths. `_cli-agents.md`'s per-provider Model discovery rows SHALL point at `fab agent models <provider>` as the first recipe. `docs/specs/stage-models.md` SHALL gain a short model-discovery section, and `docs/site/skill.md` SHALL list `fab agent models` among config-free commands.

- **GIVEN** an agent reading the deployed `_cli-fab.md`
- **WHEN** it needs the models a tool offers
- **THEN** it finds `fab agent models [provider] [--json]` with its row schema and fail-open contract

### Non-Goals

- F3 native-rung descent, F2 tool-default token, N1 provider→cli rename — later plan steps.
- A stale-pin warning in `fab agent <stage> -o yaml` — plan open decision, follow-up.
- Caching discovery; probing agy (`agy models`) — Brief says fills only for agy.
- Validating model strings anywhere in resolution.

### Design Decisions

#### fab owns model discovery and fails open
**Decision**: One verb, `fab agent models`, asks each installed tool for its models through per-tool adapters (or a user `models_command`); every failure degrades to the kit fills plus a stderr warning, exit 0.
**Why**: One implementation serves hexokit's settings page, the operator, and later `fab config explain`; discovery reads unstable surfaces (`codex debug models` is a debug command) and must never block a caller.
**Rejected**: hexokit reading tool configs itself (re-implementation drift, cf. the agy inherit-label bug); caching in v1 (codex discovery measured ~0.16 s).
*Introduced by*: 261117-rhpz-agent-models-discovery

#### Minimal TOML line scanner instead of a TOML dependency
**Decision**: Read codex's `model` and kimi's model tables with a small in-package scanner that tracks `[table]` headers and reads single-line string / string-array values.
**Why**: Only four keys are needed; `go.mod` stays at cobra/pflag/yaml.v3.
**Rejected**: Adding a TOML module — more robust to exotic syntax but a new dependency for four keys; easy to swap in later behind the same function.
*Introduced by*: 261117-rhpz-agent-models-discovery

## Tasks

### Phase 1: Setup

- [x] T001 Add `ModelsCommand string \`yaml:"models_command"\`` to `config.ProviderConfig` in `src/go/fab/internal/config/config.go` (doc comment alongside the command fields) and merge it per field in `agent.ResolveProvider` (`src/go/fab/internal/agent/agent.go`); add tests in `internal/config/config_test.go` and `internal/agent/agent_test.go` (user override sets it; built-ins carry none) <!-- R9 -->
- [x] T002 Register `models_command` in `src/go/fab/internal/configref/configref.go`: add it to the `providers.<name>.{interactive_command,headless_command}` dynamic key case, add `ModelsCommand` (omitempty) to `providerDefault`, and extend the `providers` row Description + long `Segment` prose to document it (optional, replaces the built-in adapter for `fab agent models`, prints JSON rows); update configref tests/goldens that snapshot provider fields <!-- R9 -->

### Phase 2: Core Implementation

- [x] T003 Create `src/go/fab/internal/agentmodels/` with `models.go`: `Row` type (JSON tags per R3, `Efforts` never nil), source constants, `discoveryTimeout` named constant (3s), a `Discover(cfg *config.Config, name string) (rows []Row, warnings []string)` entry that selects `models_command` > built-in adapter by name > none, then merges fills and the default per R4; dedupe helper; an injectable environment struct (home dir, env lookup, command runner) so tests can isolate `HOME`/`CLAUDE_CONFIG_DIR`/`CODEX_HOME`/`PATH` <!-- R3, R4, R8 -->
- [x] T004 [P] `internal/agentmodels/toml.go`: minimal line scanner — top-level `key = "string"` before the first header, `[a."b"]` / `[a.b]` / `[[x]]` header tracking, and inside a table `key = "string"` and single-line `key = [ "a", "b" ]`; comments and blank lines skipped; never returns content from tables the caller did not ask for <!-- R5, R7 -->
- [x] T005 [P] `internal/agentmodels/codex.go`: run `codex debug models` with `exec.CommandContext` + timeout; pinned parse struct (slug, display_name, visibility, supported_reasoning_levels[].effort); keep `list`; default from `$CODEX_HOME/config.toml` via the scanner; errors → warnings <!-- R5 -->
- [x] T006 [P] `internal/agentmodels/claude.go`: settings.json + settings.local.json merge under `$CLAUDE_CONFIG_DIR`; aliases unless `replaceBuiltInOptions`; options → rows; default; fixed efforts `low, medium, high, xhigh, max` (named constant) <!-- R6 -->
- [x] T007 [P] `internal/agentmodels/kimi.go`: `~/.kimi-code/config.toml` model tables → rows with `display_name`/`support_efforts`; `default_model`; missing file silent <!-- R7 -->
- [x] T008 `internal/agentmodels/command.go`: `models_command` runner via `sh -c` with timeout; JSON array parse; overwrite `cli`/`source`; drop empty ids; errors → warnings <!-- R9 -->
- [x] T009 `src/go/fab/cmd/fab/agent_models.go`: cobra `models` child (`Use: "models [provider]"`, `cobra.MaximumNArgs(1)`, `--json`, `--repo`), unknown-provider error listing `agent.ProviderNames`, JSON array vs aligned table (`text/tabwriter`) on stdout, warnings to stderr; register via `cmd.AddCommand(agentModelsCmd())` in `agentCmd()` (`src/go/fab/cmd/fab/agent.go`) <!-- R1, R2 -->

### Phase 3: Integration & Edge Cases

- [x] T010 Adapter tests with fixtures under `src/go/fab/internal/agentmodels/testdata/`: recorded codex `debug models` JSON (incl. a `hide` row and an `ultra` effort), malformed JSON, missing `models`, non-zero exit, timeout (stub script that sleeps, on a temp `PATH`, with a short test timeout), binary absent; codex default via `CODEX_HOME`; claude aliases-only / options+labels / `replaceBuiltInOptions` / local-over-base / malformed / `CLAUDE_CONFIG_DIR`; kimi quoted + bare headers, display_name/support_efforts, default_model, `[providers.*]` secrets never surfaced, missing file silent; `models_command` valid / cli+source overwritten / non-zero / malformed / timeout; TOML scanner unit cases <!-- R5, R6, R7, R9 -->
- [x] T011 Merge tests in `internal/agentmodels`: dedupe precedence `tool > alias > fill` with first-position kept, default OR-ing, label/efforts back-fill, fills always present (incl. a user fill override), unlisted default appended, agy fills-only with no warning, user provider without `models_command` fills-only, `efforts` marshals as `[]` <!-- R4, R8 -->
- [x] T012 Command tests in `src/go/fab/cmd/fab/agent_models_test.go`: `--json` array shape, table header/default marker, positional filter, unknown provider error, too many args, `--repo`, works with no fab project, warnings on stderr with exit 0; regression that `fab agent review -o yaml`, `fab agent --print apply`, and `fab agent --print -- --resume` are unchanged with the child registered; check `help_dump`/`fab_help` tests still pass with the new child <!-- R1, R2 -->

### Phase 4: Polish

- [x] T013 Docs: `src/kit/skills/_cli-fab.md` § fab agent — new `### fab agent models` subsection (signature, rows, merge, adapters + config locations, fail-open/stderr, exit codes, `models_command`) and add the subcommand to the § Contents list if subsections are listed; `src/kit/skills/_cli-agents.md` — each provider's Model discovery row leads with `fab agent models <provider>`; `docs/specs/stage-models.md` — short "Model discovery — `fab agent models`" section (verify `internal/agent/stagemodels_doc_test.go` passes); `docs/site/skill.md` — add `fab agent models` to the config-free commands; `docs/specs/config.md` — mention `models_command` where provider fields are enumerated <!-- R10 -->
- [x] T014 Run `go test ./...` in `src/go/fab` (scoped packages first: `internal/agentmodels`, `internal/agent`, `internal/config`, `internal/configref`, `cmd/fab`) and `go vet`; also build and smoke-run `fab agent models` against the real machine configs (read-only) <!-- R1, R2 -->

## Execution Order

- T001 blocks T002, T003, T008
- T004 blocks T005 and T007
- T003 blocks T005–T009
- T009 blocks T012

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `fab agent models [provider] [--json] [--repo]` exists as a child of `fab agent`, covers all `agent.ProviderNames` by default, filters by one positional, and errors naming the available providers on an unknown name
- [ ] A-002 R2: `--json` prints one JSON array (efforts never null); the default table has the specified header and `*` default marker; warnings go to stderr as `warning: <provider>: <reason>`
- [ ] A-003 R3: Rows carry exactly `cli, id, label, efforts, default, source` with `source` ∈ {tool, alias, fill}
- [ ] A-004 R4: Fills from the resolved provider profiles are always present; dedupe/default/back-fill rules hold
- [ ] A-005 R5: codex adapter keeps only `list` models, takes efforts from `supported_reasoning_levels`, default from `$CODEX_HOME/config.toml`
- [ ] A-006 R6: claude adapter honours `CLAUDE_CONFIG_DIR`, local-over-base merge, `replaceBuiltInOptions`, option labels, top-level default, fixed efforts
- [ ] A-007 R7: kimi adapter lists `[models."…"]` tables with display_name/support_efforts and `default_model`
- [ ] A-008 R8: agy and user providers without `models_command` yield fills only, no warning
- [ ] A-009 R9: `models_command` loads, merges, replaces the adapter, is settable via `fab config set`, and is documented in `fab config explain providers`
- [ ] A-010 R10: `_cli-fab.md`, `_cli-agents.md`, `stage-models.md`, `docs/site/skill.md` document the verb

### Behavioral Correctness

- [ ] A-011 R1: Every pre-existing `fab agent` form (selector, `--provider`, `-o yaml`, `--print`, `-t`, `--` passthrough) behaves identically with the child registered

### Scenario Coverage

- [ ] A-012 R5: A recorded `codex debug models` fixture test exists and pins the parsed fields
- [ ] A-013 R1: A test runs the verb with no fab project and asserts success

### Edge Cases & Error Handling

- [ ] A-014 R2: Binary absent, non-zero exit, timeout, and malformed output each produce fills + one warning + exit 0 (codex and `models_command`)
- [ ] A-015 R6: Malformed claude settings warns and still yields the aliases; missing files are silent
- [ ] A-016 R7: Missing kimi config is silent; values from non-model tables never appear in rows or warnings

### Code Quality

- [ ] A-017 Pattern consistency: New code follows the naming, doc-comment density, and cobra/test-helper patterns of `cmd/fab/agent*.go` and `internal/agent`
- [ ] A-018 No unnecessary duplication: `loadRepoConfig`, `agent.ProviderNames`, `agent.ResolveProvider` are reused rather than re-implemented
- [ ] A-019 Readability over cleverness: each adapter is a small focused function; no god functions (>50 lines without reason)
- [ ] A-020 No magic strings/numbers: timeout, effort set, alias list, and source values are named constants
- [ ] A-021 CLI ⇒ docs + tests: the new command ships tests and its `_cli-fab.md` reference in this change
- [ ] A-022 Deployed content cites no fab-kit-only paths: `_cli-fab.md` / `_cli-agents.md` edits name no `docs/` or `src/go/` paths

### Security

- [ ] A-023 R7: The kimi parser never copies credential-bearing values (e.g. `[providers.*] api_key`) into output; `models_command` output is parsed as data only

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Table header `CLI MODEL LABEL EFFORTS SOURCE DEFAULT`, `*` marks default | Intake illustrated this shape; human output is not a machine contract | S:65 R:90 A:70 D:65 |
| 2 | Confident | `discoveryTimeout` = 3s, overridable in tests via the injectable env | Intake proposal; codex measured ~0.16 s | S:70 R:90 A:70 D:70 |
| 3 | Confident | Claude settings merge is shallow on the two keys `model` and `modelPicker` | Intake specifies exactly that | S:75 R:85 A:75 D:70 |
| 4 | Confident | `_cli-agents.md` model-discovery rows and `docs/site/skill.md` are in the sibling-sweep class | They already describe model discovery / config-free `fab agent`; leaving them stale contradicts the new verb | S:60 R:85 A:75 D:70 |
| 5 | Confident | No new go.mod dependency; TOML via a minimal scanner | Intake assumption 7, carried | S:60 R:70 A:65 D:55 |

5 assumptions (0 certain, 5 confident, 0 tentative).
